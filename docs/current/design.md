# unfurl — design

Reference for `unfurl`'s internals. The user-facing description lives in `README.md`; this document is for someone reading or modifying the transform.

## Overview

`unfurl` is a three-pass line-oriented transform:

1. **Classify** — read the input line-by-line and assign each physical line a `LineKind` from a small fixed alphabet.
2. **Group** — walk the classified lines and emit a flat sequence of typed `Block` records, doing the small amount of contextual reasoning that requires lookback or lookahead.
3. **Emit** — walk the block sequence. Non-paragraph blocks write their original source bytes verbatim. Paragraph blocks pass through `reflow` first, then write.

The transform is hand-rolled rather than driven by a third-party markdown parser. The reasoning: the problem is line-oriented (collapse soft breaks inside paragraphs, preserve everything else), and the byte-for-byte preservation contract for non-paragraph blocks is trivial under a line-classifier model — the original bytes are right there in `Line.Raw`. Goldmark is present only as a test-time safety net (see *The correctness property* below); it is not compiled into the runtime binary.

Top-level pipeline lives in `unfurl.go`:

```go
doc, err := tokenize.Tokenize(r)
blocks := tokenize.Group(doc)
err = emit.Emit(w, doc, blocks)
```

Each pass is a separate package under `internal/`.

## Pass 1 — line classifier

Implemented in `internal/tokenize/tokenize.go`. Operates on the input as a stream of `bufio.Reader.ReadBytes('\n')` slices, so trailing line endings are preserved on `Line.Raw`. A pre-pass strips a leading UTF-8 BOM (`\xEF\xBB\xBF`) if present and records `Document.HadBOM`; all subsequent classification operates on BOM-free input so first-line constructs (frontmatter delimiter, ATX heading) are recognized correctly. A second pre-pass sniffs `Document.LineEnding` — CRLF if the first non-empty line ends `\r\n`, LF otherwise — and tracks `Document.EndedWithNewline` for trailing-newline-presence preservation.

For each line, pattern predicates fire in priority order to assign a `LineKind`:

| Kind | Pattern |
| --- | --- |
| `LineFrontMatterDelimiter` | bare `---` or `+++` |
| `LineATXHeading` | `^#{1,6}( |$)` after up-to-3-space indent |
| `LineHRule` | three or more `-`, `_`, or `*` with optional spaces, after up-to-3-space indent |
| `LineSetextUnderline` | `^=+ *$` or `^-+ *$` (tentative — resolved in pass 2) |
| `LineFence` | three or more `` ` `` or `~`, after up-to-3-space indent |
| `LineHTMLBlockStart` | one of the seven CommonMark HTML block start patterns; `Line.HTMLType` records which |
| `LineBlockquote` | leading `>` after up-to-3-space indent |
| `LineListMarker` | bullet (`-`, `+`, `*`) or ordered (`\d+[.)]`) marker |
| `LineTableDelimiter` | GFM pipe-table delimiter shape (`:?-+:?` cells); `Line.TableDelimiter.CellCount` recorded |
| `LineIndentedCode` | four or more leading spaces or one leading tab |
| `LineReferenceDefinition` | `[label]: dest` on a single line (v1 form only) |
| `LineBlank` | empty or whitespace-only |
| `LineParagraphText` | default fallback |

Several fields ride on the `Line` record for use by pass 2:

- `HardBreak` and `HardBreakMarker` — set when a paragraph-text line ends in two or more trailing spaces or a single trailing backslash. `HardBreakMarker` captures the exact bytes consumed (e.g. `"   "` for a three-space marker) so reflow can re-emit them verbatim. Three trailing spaces in → three trailing spaces out; this is part of the byte-fidelity contract.
- `Indent` — number of leading spaces (with tab expansion), used by the grouper's lazy-continuation logic.
- `Fence`, `HTMLType`, `ListMarker`, `TableDelimiter`, `BlockquotePrefixLen` — populated when the corresponding kind is assigned.

The classifier is intentionally context-free. Same-line decisions (is this a fence open? a list marker? a setext underline shape?) are made here. Cross-line decisions (does this setext underline actually close an open paragraph above? does this delimiter row form a valid table with the line above?) are deferred to pass 2.

## Pass 2 — block grouping

Implemented in `internal/tokenize/group.go`. Walks the classified line stream maintaining a stack of **container contexts**: document, blockquote (depth), list-item (content column), fenced-code (closer fence), indented-code, html-block (end rule), frontmatter (closer delimiter). Output is a flat `[]Block` with `BlockKind` matching each construct.

### The paragraph-interruption matrix

A paragraph opens on the first `LineParagraphText` and closes only when one of the following arrives:

| Candidate | Interrupts open paragraph? |
| --- | --- |
| Blank line | yes |
| ATX heading | yes |
| HRule | yes |
| Fence open | yes |
| Frontmatter delimiter | only at line 0 |
| HTML block types 1–6 | yes |
| HTML block type 7 | **no** (demote to paragraph text) |
| Setext underline | yes — only when in the **same container context** as the open paragraph |
| Indented code (4+ spaces) | **no** (continuation of the open paragraph) |
| Ordered list marker | only when start number is `1` **and** item is non-empty |
| Bullet list marker | only when item is non-empty (bare `-` cannot interrupt) |
| Blockquote marker | yes |
| Table delimiter | only when it forms a valid GFM header/delimiter pair (see below) |
| Container close | yes |

Candidates not in the "yes" column are demoted to paragraph text and extend the open paragraph instead. The rows marked "no" are silent-failure hot spots — getting them wrong turns a wrapped paragraph into a misclassified block that then disappears from reflow.

### Lazy continuation

Inside a blockquote, a paragraph-text line *without* a `>` prefix extends the open paragraph at the blockquote level above. Inside a list item, a paragraph-text line whose indent is less than the item's content column but paragraph-shaped extends the open paragraph. Container prefixes are captured **per segment**, not per paragraph (see *Reflow* below) so that hard breaks crossing container shape emit each segment with its source-correct prefix.

### Retroactive setext reclassification

When a setext underline appears in the **same container context** as the open paragraph immediately above, the paragraph is reclassified as `BlockSetextHeading` (non-reflowable). A `---` or `===` line that arrives via lazy continuation from a different container shape — for example, inside a deeper blockquote level than the paragraph above — does **not** reclassify; it remains continuation text or an HRule per its own context's rules. Among the ambiguous setext-vs-HRule-vs-bullet `-` cases, setext wins only when both the same-context rule and the "paragraph open immediately above" rule are satisfied.

### Retroactive table reclassification

A paragraph-text line followed by a candidate table delimiter row becomes a `BlockTable` **only when the two lines form a valid GFM header/delimiter pair**: the delimiter must satisfy GFM's cell shape (`:?-+:?` with optional leading/trailing pipes and surrounding spaces), and its cell count must match the header line's cell count. Cell counting splits each line on unescaped `|`, ignoring a single leading and trailing pipe if present.

If the pair is invalid, both lines remain paragraph and reflow normally. If the pair is valid, body rows accumulate per GFM rules: every following nonblank line is a body row until a blank or a row in the interruption "yes" column. Body rows are accepted regardless of pipe count and regardless of cell-count match with the header — a pipeless single-cell row like `bar` is still a valid body row in GFM. Tables are non-reflowable and emit byte-for-byte including any pipeless rows.

Conservatism here goes one direction only: false-negative on *header* detection is recoverable (the property test catches it as AST divergence on the reflowed paragraph); false-positive turns a real wrapped paragraph that happens to contain a dashes-and-pipes line into a "table" that then passes through silently. Body-row accumulation, by contrast, is permissive — false-negative truncates real tables.

### HTML block start, interrupt, and end rules

Each of the seven CommonMark HTML block types carries its own start pattern, end rule, and paragraph-interrupt rule. Types 1–6 (script/pre/style/textarea, `<!--` comments, `<?` processing instructions, `<!` declarations, `<![CDATA[`, and the block-level tag whitelist) may interrupt an open paragraph; type 7 (any other complete open or close tag on a line by itself) must not. When a candidate type-7 start appears while a paragraph is open, the line is treated as paragraph text and the paragraph continues. The seven type-specific end conditions are encoded inline in the grouper with a unit test per type.

### Reference definitions

V1 supports single-line refdefs only. The multi-line form (title on the following line) is a documented limitation; the property test surfaces it if it bites a real document.

## Pass 3 — emit

Implemented in `internal/emit/emit.go`. Walks the flat `[]Block` in order:

- Non-paragraph blocks write `Line.Raw` for each line verbatim. No transformation. This is what makes byte-for-byte preservation trivial — the original bytes are still right there.
- `BlockParagraph` is handed to `reflow.NewParagraph` and then re-emitted as one physical line per segment.
- If `Document.HadBOM`, the BOM is re-prepended as the first bytes of output.
- If `Document.EndedWithNewline` is false and the final emitted byte is `\n`, the trailing newline is suppressed.

The line-ending discipline is preserved by carrying the original line endings on `Line.Raw` for non-paragraph blocks and by using `Document.LineEnding` as the joiner inside reflowed paragraphs.

## Reflow algorithm

Implemented in `internal/reflow/reflow.go`. A paragraph is modeled as an ordered list of **segments** separated by hard breaks; each segment is an ordered list of source lines; each segment carries its own captured prefix (the prefix of *its first source line*):

```go
type Segment struct {
    Prefix          []byte
    Lines           []tokenize.Line
    HardBreakMarker []byte
}

type Paragraph struct {
    Segments []Segment
}
```

Pass 2 builds the segment list while accumulating paragraph lines: each `HardBreak == true` line closes the current segment with its marker recorded and opens a new one whose `Prefix` is captured from the next source line. Reflow then emits one physical line per segment:

```
Prefix + (segment inner content joined with a single space) + (HardBreakMarker if not last) + line-ending
```

N hard breaks → N+1 emitted physical lines, each carrying its own segment prefix. In flat documents this collapses to the simple case (all segments share the same empty or container prefix). In nested containers with hard breaks that cross container shape, each segment emits with its source-correct prefix.

### Byte-fidelity rules in reflow

- **Hard-break markers are exact bytes.** A three-trailing-space marker stays three; a four-trailing-space marker stays four; a backslash hard break stays a backslash. Never normalize a multi-space run down to two.
- **Single trailing non-hard space at a wrap seam is dropped.** When a source line ends with exactly one trailing space immediately before a soft break (`"foo \nbar"`), the trailing space is dropped before joining: `"foo \nbar"` → `"foo bar"`, not `"foo  bar"`. A single trailing space is invisible in compliant renderers, so preserving it would create a visible double-space the author did not write. The property-test comparator encodes the same policy.

## Byte fidelity

The transform's byte-fidelity contract:

1. **Line endings.** Detected once on input and used as the joiner inside reflowed paragraphs and as the suffix on emitted segment lines. LF input → LF output; CRLF input → CRLF output, hard-break joiners included.
2. **UTF-8 BOM.** Stripped before classification (so first-line constructs are recognized correctly) and re-attached as the first bytes of output when `Document.HadBOM` is set.
3. **Trailing newline presence.** If the input did not end with `\n`, the output does not either. Tracked on `Document.EndedWithNewline`.
4. **Non-paragraph blocks.** Every byte from every `Line.Raw` in a non-paragraph block is emitted verbatim, including any trailing whitespace, tabs, mixed indentation in code blocks, and table cell padding.
5. **Hard-break markers.** Re-emitted verbatim from `HardBreakMarker`. No normalization.

Idempotence falls out of these rules structurally rather than coincidentally — there is no formatting step that could decide to "improve" the output of a previous run.

## The correctness property

The load-bearing test, in `unfurl_property_test.go`. For each fixture in `testdata/property/`:

1. Parse the input with goldmark configured for GFM + frontmatter:
   ```go
   goldmark.New(
       goldmark.WithExtensions(
           extension.GFM,
           &frontmatter.Extender{},
       ),
   )
   ```
   Pure CommonMark mode would misread frontmatter as a setext heading and fail comparisons for reasons unrelated to anything the transform is doing.
2. Run `Unfurl` on the input.
3. Parse the output with the same goldmark configuration.
4. Walk both ASTs with a custom comparator and assert equivalence.

The comparator compares `node.Kind()` and kind-specific structural attributes (heading level, list ordering, fence info string, blockquote nesting, table alignment). For text leaves, source bytes are compared **exactly**, with one and only one normalization: an `ast.SoftLineBreak` token is treated as a single ASCII space. No collapsing of multi-space runs, no tab normalization, no other whitespace fuzzing. The transform is authorized to replace soft breaks with a space and nothing else; the comparator matches that authorization precisely so anything broader is caught.

For byte-significant nodes (`ast.FencedCodeBlock`, `ast.CodeBlock`, `ast.HTMLBlock`, raw HTML inline, code spans, and frontmatter nodes from the `frontmatter` extension), bytes are compared exactly. No normalization.

Two operational properties fall out of the AST invariant:

- **Idempotence.** `UnfurlBytes(UnfurlBytes(src)) == UnfurlBytes(src)` byte-for-byte, asserted for every fixture.
- **Construct preservation.** The byte-for-byte preservation of code, tables, HTML, frontmatter, and reference definitions means running `unfurl` on a grimoire note or project README will not subtly alter a code block's indentation, a mermaid diagram's structure, or a YAML block's formatting.

## Test layers

- **`internal/tokenize/tokenize_test.go`** — one row per `LineKind`, every variant covered.
- **`internal/tokenize/group_test.go`, `container_test.go`, `table_test.go`** — block-grouping cases including lazy continuation, retroactive setext and table reclassification, the paragraph-interruption matrix, HTML block end conditions per type, frontmatter at line 0, container nesting.
- **`internal/reflow/reflow_test.go`** — single-line, multi-line, internal hard break with each marker variant, multi-segment per-prefix capture, hard-break marker byte-fidelity (three-, four-space, backslash).
- **`internal/emit/emit_test.go`** — emit pass against assembled block sequences.
- **`unfurl_test.go`** — public-API smoke tests, construct-preservation table (fenced code, indented code, GFM table, all seven HTML block types, YAML and TOML frontmatter, refdef collections), scenario tests, edge cases (empty, CRLF, BOM combinations, trailing-newline-absent, mixed tabs/spaces, unclosed fence).
- **`unfurl_property_test.go`** — AST-equivalence property test and idempotence test over `testdata/property/`. Seed corpus: every scenario from `README.md`, several grimoire-style notes, and explicit fixtures for the silent-failure cases (paragraph-interruption matrix "no" rows, the comparator-tightness fixtures, GFM table edge cases).
- **`cmd/unfurl/main_test.go`** — CLI integration: stdin path, file-arg path, in-place path (file rewritten, mode preserved), `-i` without file arg (error), missing file (wrapped error).

Run via `make test` — `go test ./... -count=1 && go vet ./...`.
