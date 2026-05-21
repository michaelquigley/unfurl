# unfurl — Work Order

## Context

The design agent has produced a spec for `unfurl`, a surgical markdown unwrap utility (`docs/future/unfurl-spec.md`). The spec leaves two things to the planning agent: the implementation strategy (hand-rolled tokenizer vs goldmark-based) and the slicing of work into deliverable chunks. This work order grounds the spec in the actual project state — currently a bare Go module with no source code — and translates it into a concrete implementation plan that an implementation agent can execute end-to-end.

The work order is meant to enter mercurius review alongside the spec. After convergence, an implementation agent picks it up.

## Settled Decisions

Three load-bearing calls ratified with Michael before drafting:

1. **Implementation strategy: hand-rolled line tokenizer for the transform, goldmark for the property-test invariant only.** Goldmark stays in the test build, not the runtime binary. Rationale: line-oriented transform matches the line-oriented problem; byte-for-byte preservation of non-paragraph blocks is trivial under hand-rolled (just write the bytes back); lore parses frontmatter by hand for exactly this fidelity reason; the AST-equivalence property test using goldmark catches mistakes the hand-rolled classifier might make on CommonMark edge cases. Implicit reconsideration trigger: if more than ~3 nontrivial classification bugs emerge post-implementation, pull the chain and switch to goldmark before the hand-rolled classifier accretes a long tail of fixes.
2. **Byte fidelity is total.** Detect line-ending discipline (LF vs CRLF) from input and emit in kind. Preserve UTF-8 BOM. Preserve absence of trailing newline. Idempotence is structural, not coincidental.
3. **No `--version` flag.** Keeps `main.go` to a single command surface. `--help` is the only meta-command.

## Stack Conventions to Mimic

Drawn from `../lore/` and `../archive/`:

- **Layout.** `cmd/<binary>/main.go` for the CLI entry, `internal/` for library code that isn't the public package, root package files (`unfurl.go`) for the public API.
- **CLI.** spf13/cobra directly (no fang). Flags via `PersistentFlags().BoolVarP(...)`. Use `RunE` for error-returning runners.
- **Logging.** `dl.Init(dl.DefaultOptions().SetTrimPrefix("github.com/michaelquigley/"))` in `main.go`'s `init()`. (lore and archive use `git.hq.quigley.com/products/` because that's their namespace; unfurl's module is `github.com/michaelquigley/unfurl`, so the trim prefix matches.) Verbose flag re-inits with `slog.LevelDebug` in `PersistentPreRun`. The rest of the code logs via standard `log/slog` once `dl` is initialized.
- **Testing.** Stdlib `testing.T`, `*_test.go` next to production code, table-driven where it fits. `make test` runs `go test ./... -count=1 && go vet ./...`.
- **Makefile.** Minimal: `build` (go install), `test`, `clean`.
- **AGENTS.md.** Source of truth; `CLAUDE.md` is a symlink to `AGENTS.md`. Tone: direct, technical, includes glossary, architecture overview, conventions. For a tight utility, keep it short.
- **No `dd`.** unfurl has no configuration to bind.

## Repo Skeleton

All paths relative to `/home/michael/Repos/q/products/unfurl/`.

**Production:**

- `cmd/unfurl/main.go` — cobra root, `dl.Init` in `init()`, `-i/--in-place` and `-v/--verbose` flags, `RunE` dispatching stdin / file / in-place, atomic in-place write helper.
- `unfurl.go` (package `unfurl`) — public API: `Unfurl(r io.Reader, w io.Writer) error` and `UnfurlBytes(src []byte) ([]byte, error)`. Thin wrappers over the internal pipeline.
- `internal/tokenize/types.go` — `LineKind`, `Line`, `BlockKind`, `Block`, predicate helpers.
- `internal/tokenize/tokenize.go` — pass 1: line classifier.
- `internal/tokenize/group.go` — pass 2: block grouping with container-context stack, lazy continuation, setext/table retroactive reclassification.
- `internal/reflow/reflow.go` — paragraph reflow with hard-break preservation and container-prefix-aware emission.
- `internal/emit/emit.go` — pass 3: block-tree to bytes writer.

**Tests:**

- `unfurl_test.go` — public-API smoke tests, spec-scenario tests, construct-preservation tests.
- `unfurl_property_test.go` — goldmark AST-equivalence property test + idempotence test, both walking `testdata/property/`.
- `internal/tokenize/tokenize_test.go`, `internal/tokenize/group_test.go`, `internal/reflow/reflow_test.go` — unit tests next to production code.
- `cmd/unfurl/main_test.go` — CLI integration tests (stdin, file arg, in-place, error paths).
- `testdata/property/` — seed fixtures (every scenario input from the spec, the spec document itself, a handful of real grimoire-style notes).

**Build & docs:**

- `Makefile`, `README.md`, `AGENTS.md`, `CLAUDE.md` (symlink → `AGENTS.md`).

## Public API

```go
package unfurl

import "io"

// Unfurl reads markdown from r and writes the unfurled output to w.
// Soft line breaks inside paragraph content are collapsed; every other
// CommonMark/GFM construct is preserved byte-for-byte.
func Unfurl(r io.Reader, w io.Writer) error

// UnfurlBytes is a convenience wrapper around Unfurl for in-memory use.
// The returned slice is a fresh allocation; the input is not modified.
func UnfurlBytes(src []byte) ([]byte, error)
```

No exported types, no exported errors, no configuration surface. Errors are stdlib `fmt.Errorf` wrappings of the underlying I/O error.

## Internal Architecture

**Three passes.** Pass 1 classifies each physical line in isolation. Pass 2 groups lines into typed blocks, doing the small amount of contextual reasoning that requires lookback/lookahead (lazy continuation, setext lookback, fence open/close pairing, HTML block end detection, retroactive table reclassification). Pass 3 emits — non-paragraph blocks pass through original bytes verbatim; paragraph blocks call reflow then emit.

**Pass 1 — line classifier.** A pre-pass strips a leading UTF-8 BOM (`\xEF\xBB\xBF`) if present and stashes a `hadBOM` flag for the emit pass; all subsequent classification operates on BOM-free input so the BOM does not contaminate first-line construct detection (frontmatter delimiter, ATX heading, etc.). Then read input as `bufio.Reader.ReadBytes('\n')` so trailing newlines are preserved in `Raw`. For each line, apply pattern predicates in priority order to assign a `LineKind` (frontmatter delimiter, ATX heading, hrule, setext underline tentative, fence open, HTML block start, blockquote marker, list marker, table delimiter tentative, indented code, reference definition, blank, default paragraph text). Hard-break detection — line ending with `  \n` (two+ trailing spaces) or `\\\n` — sets `HardBreak` on paragraph-text lines and stashes which marker was used so reflow can re-emit the original.

**Pass 2 — block grouping with container-context stack.** Maintain a stack of container contexts: document, blockquote(depth), list-item(contentCol), fenced-code(closer), indented-code, html-block(endRule), frontmatter(closer). The grouper walks the line stream emitting `Block` records. Critical state:

- *Open paragraph tracking — paragraph-interruption matrix.* A paragraph opens on the first `LineParagraphText` and closes only on the following terminators. Any candidate block-start not in the "interrupts" column is demoted to paragraph text and extends the paragraph instead. Container close also closes the paragraph.

  | Candidate block-start | Interrupts open paragraph? |
  | --- | --- |
  | Blank line | yes |
  | ATX heading | yes |
  | HRule | yes |
  | Fence open | yes |
  | Frontmatter delimiter | only at line 0 |
  | HTML block types 1–6 | yes |
  | HTML block type 7 | **no** (demote to paragraph text) |
  | Setext underline | yes — but *only* when the underline appears in the **same container context** as the open paragraph above. A `---`-or-`===` line that arrives as lazy continuation from a different container shape (e.g. inside a deeper or different blockquote/list-item context) does not retroactively reclassify the outer paragraph as a setext heading. Among ambiguous setext-vs-HRule-vs-list-marker `-` lines, setext wins only when it satisfies both this same-context rule and the "paragraph open immediately above" rule. |
  | Indented code (4+ spaces) | **no** (a 4+-space-indented line during an open paragraph is paragraph continuation text, not the start of an indented code block) |
  | Ordered list marker | **only when** start number is `1` **and** the item is non-empty. `2. foo` cannot interrupt; `1.` alone (empty item) cannot interrupt; `1. foo` can. |
  | Bullet list marker | only when the item is non-empty. `- foo` can interrupt; a bare `-` (empty item) cannot. |
  | Blockquote marker | yes |
  | Table delimiter | only when it forms a valid GFM header/delimiter pair (see retroactive table reclassification below) |

  Demoted candidates extend the paragraph; the open paragraph carries until a row in the "yes" column or container close arrives.
- *Lazy continuation.* Inside a blockquote, a paragraph-text line *without* a `>` prefix extends the open paragraph. Inside a list item, a paragraph-text line whose indent is `< contentCol` but paragraph-shaped extends the open paragraph. Container prefixes are captured **per segment**, not per paragraph — see the reflow algorithm below. The prefix attached to a segment is the prefix of the *first source line in that segment*, so a paragraph with hard breaks that span different container shapes emits each segment with its own correct prefix.
- *Retroactive setext reclassification.* When a setext underline appears **in the same container context as the open paragraph immediately above**, the open paragraph above becomes a setext heading block (which is non-reflowable). Setext underline of `-` characters collides with HRule and list-marker patterns; setext wins when (a) a paragraph is currently open immediately above and (b) the underline is in the same container context as that paragraph. A `---` line that arrives as lazy continuation from a different container shape (e.g. inside a deeper blockquote level or list item than the paragraph above) does NOT reclassify — it is just continuation text or a hrule/list-marker per its own context's rules.
- *Retroactive table reclassification.* A paragraph-text line followed by a candidate table delimiter row becomes a table block **only when the two lines form a valid GFM header/delimiter pair**: the delimiter must satisfy GFM's delimiter shape (each cell `:?-+:?` with optional leading/trailing pipes and surrounding spaces) *and* its cell count must match the header line's cell count. Cell counting splits each line on unescaped `|`, ignoring a single leading and a single trailing pipe if present (so `| a | b |` and `a | b` both count as 2). If the pair is invalid — wrong shape, mismatched arity, or the candidate "header" is not paragraph-shaped text — both lines stay paragraph and reflow normally. When the pair is valid, the prior line becomes the table header row, and **body rows accumulate per GFM rules**: every following nonblank line is a body row until a blank line or a block-start terminator (any row in the paragraph-interruption "yes" column above). Body rows are accepted regardless of pipe count and regardless of cell-count match with the header — a pipeless single-cell row like `bar` is still a valid body row in GFM, missing cells are padded empty, and excess cells are ignored. Tables are non-reflowable; emit byte-for-byte including any pipeless body rows. **Conservatism matters in the wrong direction here:** an over-eager *header*-reclassification turns a normal wrapped paragraph that happens to contain a dashes-and-pipes line into a "table" that then passes through unchanged — silent corruption. Better to false-negative on header detection (treat a real table as paragraphs, which the property test catches via AST divergence) than false-positive. But once the header/delimiter pair is validated, body accumulation is permissive per the GFM rule above — false-negative on body rows truncates real tables.
- *HTML block start, interrupt, and end conditions.* Each of the seven CommonMark HTML block types has its own start rule, its own end rule, **and its own paragraph-interrupt rule**. Lift the full table verbatim into a comment at the top of the HTML-block-handling code. The interrupt column matters specifically: types 1–6 (script/pre/style/textarea, `<!--` comments, `<?` processing instructions, `<!` declarations, `<![CDATA[`, and the block-level tag whitelist) **may** interrupt an open paragraph; type 7 (any other complete open or close tag on a line by itself) **must not**. When a candidate type-7 start appears while a paragraph is open, it does not open an HTML block — it is treated as paragraph text and the paragraph continues. Types 1–6 close the paragraph and open the HTML block per their respective end rules. One unit test per type for both the start/interrupt behavior and the end rule. One property-test fixture: a wrapped paragraph whose middle line is a standalone-looking type-7 HTML tag must reflow to a single line, not split.
- *Reference definitions (v1).* Single-line form only. Multi-line refdef (title on next line) is a documented limitation; property test surfaces it if it bites.

**Pass 3 — emit.** Walk the block tree. Non-paragraph blocks write `Line.Raw` bytes verbatim, in order. Paragraph blocks call `reflow.Reflow` and write the result. Container blocks recurse with the container's captured prefix.

**Reflow algorithm.** A paragraph is modeled as an ordered list of **segments** separated by hard breaks; each segment is an ordered list of source lines; each segment carries its own captured prefix (the prefix of *its first source line*). Concrete types:

```go
type Segment struct {
    Prefix          []byte // captured from this segment's first source line
    Lines           []Line // one or more source lines that joined into this segment
    HardBreakMarker []byte // exact consumed marker bytes that ended this segment:
                           // either the full trailing-space run of length >= 2
                           // (e.g. "   " for three spaces), or the single final
                           // backslash "\\". nil for the last segment. byte
                           // fidelity requires re-emitting these bytes verbatim;
                           // never normalize a 3+-space run down to two spaces.
}

type Paragraph struct {
    Segments []Segment
}
```

Pass 2 builds the segment list while accumulating paragraph lines: each `HardBreak == true` line closes the current segment (with its marker recorded) and opens a new one, whose `Prefix` is captured from the next source line. Reflow then emits one physical line per segment: `Prefix` + segment-inner content joined with a single space + (if not the last segment) the `HardBreakMarker` + newline. Result: N hard breaks → N+1 emitted physical lines, each carrying its own segment prefix. In flat documents this collapses to the simple case (all segments share the same empty or container prefix); in nested containers with hard breaks that cross container shape, each segment emits with its source-correct prefix.

**Byte fidelity.** Detect line-ending discipline by sniffing the first non-empty line: CRLF if it ends `\r\n`, LF otherwise. Use the detected discipline as the joiner when emitting reflowed paragraphs (hard-break segments separated by `  \r\n` or `  \n`). BOM handling is **strip-and-reattach**: a leading UTF-8 BOM is stripped before line classification (so first-line constructs like frontmatter delimiters and ATX headings are recognized correctly) and re-emitted as the first bytes of output when `hadBOM` is set. Preserve absence of trailing newline by tracking whether the input's final byte was `\n`.

## CLI

```go
// cmd/unfurl/main.go (essential shape)

var (
    inPlace bool
    verbose bool
)

var rootCmd = &cobra.Command{
    Use:   "unfurl",
    Short: "surgical markdown unwrap",
    Long:  "unfurl reads markdown and collapses soft line breaks inside paragraph content, leaving every other construct intact.",
    Args:  cobra.MaximumNArgs(1),
    PersistentPreRun: func(cmd *cobra.Command, args []string) {
        if verbose {
            dl.Init(dl.DefaultOptions().
                SetTrimPrefix("github.com/michaelquigley/").
                SetLevel(slog.LevelDebug))
        }
    },
    RunE: runRoot,
}

func init() {
    dl.Init(dl.DefaultOptions().SetTrimPrefix("github.com/michaelquigley/"))
    rootCmd.PersistentFlags().BoolVarP(&inPlace, "in-place", "i", false, "rewrite the file argument in place")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "emit progress and diagnostics")
}
```

`runRoot` dispatches: no args → stdin/stdout; one arg, no `-i` → file/stdout; one arg with `-i` → atomic in-place rewrite. `-i` without a file argument is an error. **Atomic in-place write** creates a temp file in the same directory as the target (so the rename is atomic on POSIX), chmods to the original's mode, writes bytes, `os.Rename`s over the original. Avoids the half-written-file failure mode.

## Test Plan

**Unit tests (next to production code).**

- `internal/tokenize/tokenize_test.go` — table-driven, one row per `LineKind`. Cover every ATX heading level, every HRule character, both fence types with and without info string, all seven HTML block start patterns, blockquote with and without trailing space, every list-marker form, table delimiter rows with alignment colons, indented code (4 spaces and 1 tab), single-line refdefs, hard-break detection (two spaces, three spaces, backslash, none).
- `internal/tokenize/group_test.go` — block-grouping cases: plain paragraph, paragraph + setext underline → setext heading, paragraph + table delimiter → table, blockquote with lazy continuation, list with lazy continuation, nested blockquote, nested list inside blockquote, fence containing other block-shaped lines, HTML block end conditions for each of the seven types, frontmatter at line 0 (YAML and TOML), frontmatter-shaped delimiters mid-document that must NOT be treated as frontmatter.
- `internal/reflow/reflow_test.go` — single-line paragraph, two-line paragraph, paragraph with internal hard break (both markers), paragraph with trailing-whitespace variations, list-item paragraph reflow, blockquote paragraph reflow, **multi-segment reflow with per-segment prefix capture** (paragraph with N hard breaks emits N+1 physical lines, each carrying its own segment prefix). Specific multi-segment cases: blockquote paragraph with a mid-paragraph hard break (segments share `> ` prefix); list-item paragraph with a mid-paragraph hard break (segments share the item's continuation indent); blockquote-inside-list paragraph whose hard break separates a `>`-prefixed segment from a lazy-continuation segment (each segment emits with its source-correct prefix). **Hard-break marker byte-fidelity**: a three-trailing-space hard break round-trips with three trailing spaces preserved (not normalized to two); a four-trailing-space hard break preserves four; a backslash hard break preserves the backslash; idempotence holds across all trailing-space counts.

**The AST-equivalence property test (`unfurl_property_test.go`).** Load-bearing. Parser:

```go
goldmark.New(
    goldmark.WithExtensions(
        extension.GFM,            // tables, strikethrough, autolinks, tasklists
        &frontmatter.Extender{},  // YAML and TOML — go.abhg.dev/goldmark/frontmatter
    ),
)
```

Spec quote belongs in the test file as a comment: *"Pure CommonMark mode would misread front matter as a setext heading and fail the comparison for reasons unrelated to anything the tool is doing."*

No off-the-shelf comparator exists for this exact purpose. Write a recursive walker over `goldmark/ast.Node` that:

1. Compares `node.Kind()` and kind-specific structural attributes (heading level, list ordering, fence info string, blockquote nesting, table alignment).
2. For text leaves, compares source bytes *exactly*, with one and only one normalization: an `ast.SoftLineBreak` token is treated as a single ASCII space. No collapsing of multi-space runs, no tab normalization, no other whitespace fuzzing. The transform is authorized to replace soft breaks with a space and nothing else; the comparator must match that authorization precisely so the property test catches anything broader.
3. For byte-significant nodes (`ast.FencedCodeBlock`, `ast.CodeBlock`, `ast.HTMLBlock`, raw HTML inline, code spans, and frontmatter nodes from the `frontmatter` extension), compares bytes exactly. No normalization.
4. Produces diffs as `(path, expected, actual)` tuples so failures are actionable.

**Boundary policy at a wrap seam.** When the source has a single trailing non-hard space immediately before a soft break (line ends `"foo \n"` with exactly one space, not two), the reflow **drops** the trailing space before joining: `"foo \nbar"` → `"foo bar"`, not `"foo  bar"`. A single trailing space is invisible in compliant renderers, so preserving it would create a visible double-space that the author did not write. Hard breaks (two or more trailing spaces, or a backslash) are unchanged and remain preserved exactly. The comparator encodes this same policy: when one side has a soft break preceded by a single space and the other side has just a space, they match.

**Comparator fixtures.** Beyond the general corpus, three explicit fixtures land in `testdata/property/` to defend the comparator's tightness:
- intra-line multi-space (`"foo   bar"` inside a paragraph — must survive byte-exact through a round trip);
- intra-line tab (`"foo\tbar"` inside a paragraph — must survive byte-exact);
- one trailing non-hard space at a wrap (`"foo \nbar\n"` — verifies the drop-the-single-space boundary policy and that the comparator does not false-pass on a bug that drops other trailing whitespace).

**Paragraph-interruption fixtures.** Beyond the general corpus, explicit fixtures land in `testdata/property/` for each row of the interruption matrix that does NOT interrupt a paragraph (these are the silent-failure cases):
- Wrapped paragraph whose continuation line is `    foo` (4 spaces) → reflows to single line; the 4-space-indented line does not become an indented code block.
- Wrapped paragraph whose continuation line starts `2. ` (numbered list marker, start≠1) → reflows normally; not an ordered-list start.
- Wrapped paragraph whose continuation line is `1.` (empty item) → reflows normally; empty list items don't interrupt.
- Wrapped paragraph whose continuation line is a bare `-` (empty bullet item) → reflows normally.
- Wrapped paragraph inside a blockquote whose lazy-continuation line is `---` (setext-shaped but in different container context) → reflows normally; the `---` does not retroactively reclassify the outer paragraph as a setext heading.

Property test iterates `testdata/property/` running `Unfurl` and asserting AST equivalence. Seed corpus: every scenario input from the spec, `docs/future/unfurl-spec.md` itself, a half-dozen real grimoire-style notes.

**Idempotence test.** For every fixture, `UnfurlBytes(UnfurlBytes(src)) == UnfurlBytes(src)` byte-for-byte.

**Construct-preservation tests (`unfurl_test.go`).** Table-driven, one case per construct, each asserting *entire output == entire input*: fenced code (Go), fenced code (mermaid), indented code, GFM table with alignment, each of the seven HTML block types, YAML frontmatter, TOML frontmatter, reference-definition collections (top and bottom).

**Scenario tests (`unfurl_test.go`).** One per spec scenario, using exact inputs from `docs/future/unfurl-spec.md`: Claude output (synthesized 400-line wrapped doc), hand-edited hard break, already-unwrapped, table, wrapped list item, blockquote.

**Edge cases (added to `unfurl_test.go`).** Empty input → empty output. Single newline → single newline. No trailing newline → preserve absence. CRLF input → CRLF output, including hard-break joiners. BOM at start of plain paragraph → preserve. **BOM + YAML frontmatter (`﻿---\n...`)** → frontmatter recognized, BOM re-emitted at start. **BOM + ATX heading (`﻿# title\n`)** → heading recognized, BOM re-emitted at start. Mixed tabs/spaces in indented code → byte-faithful. Trailing-whitespace-but-not-hard-break on a paragraph line → document the call and test it. Paragraph immediately after frontmatter. List item with leading nested list (no own paragraph). Blockquote-inside-list-inside-blockquote (deepest nesting expected). Multi-line refdef (v1 limitation; assert no panic, idempotent). Unclosed fence (extends to EOF per CommonMark).

**CLI integration tests (`cmd/unfurl/main_test.go`).** Stdin path, file path, in-place path (assert file rewritten + mode preserved), `-i` without file arg (assert error), missing file (assert wrapped error).

## Slicing

Each chunk is independently testable. Dependencies labeled.

**Chunk 1 — Skeleton and CLI shell.** Depends on: nothing.
- Add cobra and df/dl to `go.mod`.
- `cmd/unfurl/main.go` with cobra root, `-i`, `-v`, `dl.Init`, atomic in-place write helper.
- `unfurl.go` with `Unfurl` and `UnfurlBytes` as pass-through stubs (input → output unchanged).
- `Makefile`, `README.md`, `AGENTS.md`, `CLAUDE.md` symlink.
- CLI integration tests against the pass-through stub.
- *Acceptance:* `go install ./cmd/unfurl && unfurl --help` works; stdin/stdout, file arg, `-i file`, error paths all behave.

**Chunk 2 — Line classifier (pass 1).** Depends on: chunk 1.
- `internal/tokenize/types.go`, `internal/tokenize/tokenize.go`.
- Unit tests covering every `LineKind` including hard-break detection.
- *Acceptance:* `internal/tokenize/...` tests pass.

**Chunk 3 — Block grouper + emit for non-container blocks.** Depends on: chunk 2.
- `internal/tokenize/group.go`, `internal/emit/emit.go`.
- Handle paragraph, fenced code, indented code, ATX heading, setext heading (with retroactive reclassification), HRule, HTML block, frontmatter, refdef, blank.
- Paragraph reflow stub: join with space, no hard-break handling yet.
- Wire `Unfurl` to use the pipeline.
- Construct-preservation tests for all non-container constructs pass.
- Spec scenario `AlreadyUnwrapped` passes. (Spec scenario `Table` is owned by chunk 5 once GFM table retroactive detection lands; until then, table fixtures may pass-through as paragraphs and the scenario stays pending.)
- *Acceptance:* documents without lists/blockquotes/hard-breaks/paragraph-to-table round-trip and reflow correctly.

**Chunk 4 — Hard-break preservation and reflow polish.** Depends on: chunk 3.
- Hard-break-aware reflow in `internal/reflow/reflow.go`.
- Spec scenario `HardBreakPreserved` passes; reflow unit tests pass.

**Chunk 5 — GFM table retroactive detection.** Depends on: chunk 3.
- Table-delimiter retroactive reclassification in the grouper, gated by header/delimiter arity match (see Pass 2 above).
- Permissive body-row accumulation per GFM: every following nonblank line is a body row until blank or block-start terminator; pipeless and cell-count-mismatched body rows still count.
- Positive tests:
  - Real table (matching header + delimiter arity) with all-pipe body rows → unchanged passthrough.
  - **Real table with a pipeless body row** (e.g. header `| a | b |`, delimiter `| --- | --- |`, body rows mixing `| x | y |` and `bar`) → all rows preserved byte-exact; `bar` stays a single-cell body row, not a paragraph.
  - Real table with body rows of varying cell count (some fewer than header) → unchanged passthrough.
- Negative tests in `testdata/property/`:
  - Wrapped paragraph above a delimiter-shaped line whose cell count does not match the line above → must reflow normally, not become table.
  - Wrapped paragraph above a line that is all dashes and pipes but does not satisfy GFM delimiter shape (e.g. `---|---|---` without colons or with leading text) → must reflow normally.

**Chunk 6 — Container contexts: blockquote and list item.** Depends on: chunk 5.
- Container-stack model in the grouper.
- Lazy continuation in blockquotes and list items.
- Container-prefix-aware reflow.
- Spec scenarios `WrappedListItem` and `Blockquote` pass.
- Nested-container fixtures behave.

**Chunk 7 — Property test and goldmark integration.** Depends on: chunk 6.
- Add `github.com/yuin/goldmark` and `go.abhg.dev/goldmark/frontmatter` to `go.mod`.
- Implement AST walker and comparator in `unfurl_property_test.go`.
- Seed `testdata/property/` (scenario fixtures + spec doc + a half-dozen grimoire-style notes).
- Idempotence test.
- `TestScenario_ClaudeOutput` passes.
- *Acceptance:* property test passes on every fixture **except** the BOM, CRLF, and trailing-newline-presence fixtures, which are deferred to chunk 8 (these fixtures may be added to `testdata/property/` at this chunk but skipped via build tag, fixture filename convention, or `t.Skip`-with-reason until chunk 8 lands the byte-fidelity logic that makes them pass). Failure diffs on the non-deferred fixtures are actionable.

**Chunk 8 — Byte-fidelity edge cases and polish.** Depends on: chunk 7.
- Line-ending detection and discipline preservation.
- BOM strip-and-reattach (Pass 1 pre-pass and emit-pass per Byte fidelity section).
- Trailing-newline-presence preservation.
- Re-enable the BOM / CRLF / trailing-newline property fixtures skipped in chunk 7; assert they now pass.
- Unclosed fence test.
- README and AGENTS.md polish.

## Things to Watch (Risk Surface)

Where bugs are most likely:

- **Setext underline retroactive reclassification.** Test with: setext at top of doc, setext after blank, setext at end of doc, setext where `---` could also be parsed as HRule (CommonMark says setext wins when a paragraph precedes).
- **Table retroactive reclassification.** Same shape. Mis-ordering produces tables-classified-as-paragraphs, which then reflow across rows — catastrophic.
- **Lazy continuation.** Paragraph in blockquote with a lazy line missing `>`, then a line with `>` again — still one paragraph. Same for list items. Explicit fixtures.
- **Hard-break detection.** Two trailing spaces vs three vs zero. Backslash hard-break must not collide with escape sequences. Get the regex right, unit-test heavily.
- **HTML block end conditions.** Each of the seven types has a distinct end condition. Easy to apply the wrong predicate. Lift the CommonMark table into a comment, one test per type.
- **Container prefix capture.** Per-segment, not per-paragraph: a paragraph with N hard breaks emits N+1 physical lines, each carrying the prefix of its segment's first source line. Reflowing a paragraph inside a list item inside a blockquote — each emitted segment line needs `> ` + list-marker indent + paragraph text, captured from that segment's source. Multiple nesting depths plus hard breaks across segment boundaries in fixtures.
- **In-place write atomicity.** Panic mid-write must not leave a half-written file. Atomic temp + rename, not naive `os.WriteFile`.

Where mercurius is likely to push back:

- **Hand-rolled rationale.** Have the reasoning ready: minimal runtime surface, lore precedent (hand-parsed frontmatter), property-test safety net.
- **Byte-fidelity choices.** Articulated and tested, not implicit.
- **Coverage of lazy continuation and retroactive reclassification.** Named test fixtures.
- **Property test corpus breadth.** Seed beyond just the scenarios — spec doc + grimoire notes.

## Verification

End-to-end checks after implementation:

1. `make test` — all unit, scenario, construct-preservation, property, and idempotence tests pass.
2. `go install ./cmd/unfurl` — binary builds and installs.
3. Manual smoke (idempotence on a known-unwrapped fixture): create `/tmp/single-line.md` with one paragraph on a single physical line plus a fenced code block; run `unfurl /tmp/single-line.md | diff - /tmp/single-line.md` — diff should be empty (already unwrapped, no changes).
4. Manual smoke (reflow on a wrapped document): `unfurl docs/future/unfurl-spec.md | wc -l` should report fewer lines than `wc -l < docs/future/unfurl-spec.md` (the spec is physically wrapped, so unfurl should collapse paragraphs). Eyeball the output — paragraphs collapsed, frontmatter / code / tables / lists / blockquotes intact.
5. Manual smoke: `unfurl -i sample.md` on a copy, verify file is rewritten with original mode preserved.
6. Manual smoke: `echo "  - one  \n    two\n" | unfurl` — verify list-item lazy continuation reflow.
7. `unfurl -v` on a non-trivial file — diagnostics emitted via `dl`.
8. Property test on a half-dozen real grimoire notes — no AST divergences.

After verification, the work order and spec hand off through mercurius review. On convergence, the implementation agent takes the spec and work order from `docs/future/`.

## Out of Scope (Explicit)

Don't plumb in v1, even speculatively: re-wrapping at a target column; bullet/emphasis/heading/table normalization; linting or validation; configuration files, env vars, rc files; recursive directory mode; file-watching; editor plugins; `dd` config marshaling; subcommands beyond the root; parallelism flags; output format flags. The deferred section of the spec governs.

## Documents

- Spec: `docs/future/unfurl-spec.md`
- Work order (this document): `docs/future/unfurl-work-order.md`
- Mercurius config: `mercurius.yaml`
- Mercurius session state: `.mercurius/`
