# unfurl — A Markdown Unwrap Utility

## The Problem

Many models — and some humans — produce markdown documents that hard-wrap their physical lines at eighty columns. The wrapping is invisible to renderers that collapse soft line breaks to spaces, but quite visible to those that don't. Obsidian with certain settings, GitHub Flavored Markdown, and several other widely-used renderers preserve soft line breaks as visible line breaks in the rendered output, producing a stair-stepped appearance where the source assumed a uniform paragraph.

The hand-authoring convention in this practice is the opposite: don't wrap, let the renderer decide where lines break. Documents that arrive wrapped need to be unwrapped before they fit cleanly into that convention. Doing this by hand is tedious and error-prone, especially in long documents with mixed content. A small, focused tool for the job earns its place quickly.

The niche is specifically the *surgical* version of this transformation. Existing markdown formatters — markdownfmt, mdformat, prettier, pandoc — all do far more: they normalize bullet styles, emphasis markers, heading types, table column widths, and other formatting choices. Running any of them on a grimoire note rewrites deliberate decisions. `unfurl` does only the one thing.

## The Name

`unfurl` is the inverse of reefing a sail. A sail that has been reefed is folded and bound; an unfurled sail is extended to its full length. The same shape applies to markdown that has been wrapped into eighty-column lines: it's been bound up and needs releasing back into its natural full-width form. The name pairs deliberately with `reef`, which serves a different role in the stack but reads as part of the same nautical family.

## What unfurl Does

`unfurl` reads a markdown document and produces an equivalent document with soft line breaks collapsed inside paragraph content. Every load-bearing construct in the original — headings, lists, code blocks, tables, blockquotes, HTML blocks, front matter, reference definitions, hard breaks — is preserved exactly. Only the soft wraps inside paragraph text are removed.

Concretely:

- A wrapped paragraph at the top level becomes a single line.
- Paragraph content inside a list item is unwrapped; the list marker and its indentation stay.
- Paragraph content inside a blockquote is unwrapped; the `>` prefix on each line stays.
- Hard breaks — a line ending in two trailing spaces or a backslash — are preserved as hard breaks even when the surrounding paragraph reflows around them.
- Fenced code blocks, indented code blocks, and mermaid diagrams (which are just fenced code with a `mermaid` info string) pass through byte-for-byte.
- Tables pass through byte-for-byte. Pipe-table rows are line-significant.
- HTML blocks pass through byte-for-byte.
- Front matter (YAML or TOML, delimited by `---` or `+++`) passes through byte-for-byte.
- Reference definitions, headings, horizontal rules pass through byte-for-byte.

## The Property

The correctness property `unfurl` should satisfy: given an input document, parse it with a CommonMark parser. Parse the output document with the same parser. Walk the resulting ASTs and verify they are equivalent *modulo soft-break normalization* — block structure identical, inline content identical once any soft-break tokens are treated as space characters, raw text inside code blocks and other byte-significant regions matching exactly.

This captures the actual goal: change the visible rendering only in the way that comes from collapsing soft breaks. Nothing else moves.

The parser used to verify the property should be configured for the extensions the documents in the practice use — front matter, GFM tables, autolinks — since these are part of the construct vocabulary the tool must preserve. Pure CommonMark mode would misread front matter as a setext heading and fail the comparison for reasons unrelated to anything the tool is doing.

Two operational properties fall out of the AST invariant:

**Idempotence.** Running `unfurl` on an already-unwrapped document produces an identical document. A second pass over any input is a no-op.

**Construct preservation.** The byte-for-byte preservation of code, tables, HTML, front matter, and reference definitions means a user can run `unfurl` on a grimoire note or a project README without fearing that a code block's indentation, a mermaid diagram's structure, or a YAML front-matter block's formatting will be subtly altered.

## Scenarios

**The Claude output.** A Claude session produces a 400-line markdown document wrapped at eighty columns. The document has front matter, a few headings, several paragraphs, two fenced code blocks (one Go, one mermaid), and a couple of bulleted lists. Running `unfurl document.md` produces the same document with every paragraph reflowed to a single physical line, every list item with a single-line body, and every code block plus the front matter untouched.

**The hand-edited file with intentional breaks.** A markdown file contains a paragraph where one sentence is followed by two trailing spaces and a line break, used as a visual pause. `unfurl` preserves the hard break. The surrounding paragraph, if wrapped, reflows around it.

**The already-unwrapped file.** A grimoire note authored by hand, with no wrapping, is run through `unfurl`. Output is byte-identical to input.

**The table.** A markdown table with pipe-separated rows passes through unchanged. Even if rows happen to look like they could be wrapped paragraph content, the table structure is detected and preserved.

**The wrapped list item.**

Input:
```
- this is a list item whose body
  has been wrapped across multiple
  physical lines
- second item
```

Output:
```
- this is a list item whose body has been wrapped across multiple physical lines
- second item
```

**The blockquote.**

Input:
```
> a quoted paragraph that has been
> wrapped at the source. each line
> begins with the marker.
```

Output:
```
> a quoted paragraph that has been wrapped at the source. each line begins with the marker.
```

## CLI Shape

```
unfurl [flags] [file]

Read markdown from stdin or from a file argument, write unfurled markdown to
stdout or in place.

Flags:
  -i, --in-place    rewrite the file argument in place
  -v, --verbose     emit progress and diagnostic information via dl
  -h, --help        usage
```

Default behavior: with no arguments, read stdin and write stdout. With a file argument, read the file and write stdout. With `-i` plus a file argument, rewrite the file in place. Without a file argument, `-i` is an error.

Built on `github.com/spf13/cobra` for command and flag handling, following the stack convention. Logging — including the verbose-mode diagnostics — runs through `github.com/michaelquigley/df/dl`.

## Library Shape

The core transformation is exposed as a Go package so that other tools in the stack (frame, grimoire tooling, possible editor integrations) can embed `unfurl` directly without shelling out to the CLI.

A minimal API surface:

```go
package unfurl

// Unfurl reads markdown from r and writes the unfurled output to w.
func Unfurl(r io.Reader, w io.Writer) error

// UnfurlBytes is a convenience for in-memory transformation.
func UnfurlBytes(src []byte) ([]byte, error)
```

Configuration is intentionally absent from the public API. The tool does one thing.

## Implementation Notes

Two viable implementation shapes, surfaced here so the planning agent doesn't re-derive them.

**Hand-rolled block tokenizer.** A line-oriented state machine that classifies each physical line into a block context — paragraph, fenced code, indented code, list item, blockquote, table, front matter, HTML block, heading, hrule, reference definition, blank. Paragraph blocks reflow; everything else passes through. This approach is small and dependency-light but requires careful handling of CommonMark's lazy continuation rules in lists and blockquotes plus a handful of other edge cases.

**goldmark-based block boundary detection.** Use `github.com/yuin/goldmark` purely as a block-level parser, walking the resulting block tree and emitting source segments verbatim for non-paragraph blocks while reflowing paragraph content. More correct on edge cases out of the box but introduces a heavyweight dependency and adds friction around extracting original source bytes through goldmark's segment APIs.

Either is workable. The "tight little utility" framing leans toward the hand-rolled approach. Final call is the planning agent's, grounded in the project repo.

Stack convention dependencies regardless of approach:

- `github.com/michaelquigley/df/dl` for logging.
- `github.com/michaelquigley/df/dd` available for any future configuration marshaling, though v1 has no configuration to bind.
- `github.com/spf13/cobra` for the CLI.

## Deferred (and Why)

**Re-wrapping at a target column.** `unfurl` only unwraps. Re-wrapping at any column would violate the practice's no-wrap convention and just recreate the problem the tool exists to solve. If a wrap-at-column tool is ever wanted, it's a separate utility, not a flag.

**Normalization of other formatting choices.** Bullet style (`-`/`*`/`+`), emphasis markers (`*`/`_`), heading style (atx vs setext), table column padding, ordered-list numbering — none of these are touched. The surgical scope is the whole virtue of the tool. Adding any of these features puts `unfurl` on a path toward becoming prettier or mdformat, and those already exist.

**Linting and validation.** `unfurl` does not warn about malformed markdown, unclosed fences, broken reference links, or any other quality issue. It is a transformation, not a checker. Malformed input is treated the way CommonMark treats it — an unclosed fence, for instance, extends to EOF.

**Configuration.** No config file, no rc file, no environment-variable overrides. Behavior is a fixed contract. If a future need surfaces a real configuration axis it can be added then; adding configuration speculatively bloats the surface area without earning its complexity.

**Recursive directory mode.** A user who wants to unfurl every markdown file in a tree can compose with `find` and `xargs`, or with shell loops. Building this into `unfurl` invites flags around glob patterns, ignore files, and parallelism — all of which are well-served by tools that already exist.

**File-watching mode.** Same reasoning. Compose with `entr`, `fswatch`, or editor integrations rather than building it in.

**Editor integrations.** Out of scope. The library API is the integration point; specific Obsidian, VS Code, or Neovim plugins are separate projects that can consume the library if and when they're wanted.

## Related

- `frame` — possible downstream consumer for normalizing inbound markdown ingested into site builds.
- `reef` — name-family sibling in the stack; no functional relationship.
- The grimoire authoring convention (no wrapping; let the renderer decide line breaks) that this tool exists to support.
