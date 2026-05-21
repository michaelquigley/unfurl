# unfurl

A surgical markdown unwrap utility. `unfurl` collapses soft line breaks inside paragraph content and preserves every other CommonMark/GFM construct byte-for-byte. Single Go binary, one operation, no configuration. Library-callable as well as a CLI.

## The problem

Many models — and some humans — produce markdown documents that hard-wrap their physical lines at eighty columns. The wrapping is invisible to renderers that collapse soft line breaks to spaces, but quite visible to those that don't. Obsidian with certain settings, GitHub Flavored Markdown, and several other widely-used renderers preserve soft line breaks as visible line breaks, producing a stair-stepped appearance where the source assumed a uniform paragraph.

The hand-authoring convention behind this tool is the opposite: don't wrap, let the renderer decide where lines break. Documents that arrive wrapped need to be unwrapped before they fit cleanly into that convention. Doing it by hand is tedious and error-prone, especially in long documents with mixed content.

The niche is specifically the *surgical* version of this transformation. Existing markdown formatters — markdownfmt, mdformat, prettier, pandoc — all do far more: they normalize bullet styles, emphasis markers, heading types, table column widths, and other formatting choices. Running any of them on a hand-tuned document rewrites deliberate decisions. `unfurl` does only the one thing.

## What unfurl does

`unfurl` reads a markdown document and produces an equivalent document with soft line breaks collapsed inside paragraph content. Every load-bearing construct in the original is preserved exactly.

- Wrapped paragraphs at the top level become single lines.
- Paragraph content inside a list item is unwrapped; the list marker and its indentation stay.
- Paragraph content inside a blockquote is unwrapped; the `>` prefix on each line stays.
- Hard breaks — a line ending in two or more trailing spaces or a backslash — are preserved as hard breaks even when the surrounding paragraph reflows around them.
- Fenced code blocks, indented code blocks, and mermaid diagrams pass through byte-for-byte.
- Tables pass through byte-for-byte. Pipe-table rows stay line-significant.
- HTML blocks pass through byte-for-byte.
- Front matter (YAML or TOML, delimited by `---` or `+++`) passes through byte-for-byte.
- Reference definitions, headings, horizontal rules pass through byte-for-byte.

## Properties

**Idempotence.** Running `unfurl` on an already-unwrapped document produces an identical document. A second pass over any input is a no-op.

**Construct preservation.** The byte-for-byte preservation of code, tables, HTML, front matter, and reference definitions means you can run `unfurl` on a grimoire note or a project README without worrying that a code block's indentation, a mermaid diagram's structure, or a YAML front-matter block's formatting will be subtly altered.

**Byte fidelity.** Line-ending discipline (LF vs CRLF) is detected from input and preserved. A leading UTF-8 BOM is stripped before classification and re-attached on output. Absence of a trailing newline is preserved. Hard-break markers are re-emitted byte-for-byte — three trailing spaces in, three trailing spaces out.

## Quick start

```sh
go install github.com/michaelquigley/unfurl/cmd/unfurl@latest
```

```sh
# read from stdin, write to stdout
cat note.md | unfurl

# read a file, write to stdout
unfurl note.md

# rewrite a file in place (atomic; preserves mode)
unfurl -i note.md
```

## CLI

```
unfurl [flags] [file]
```

| Flag | Behavior |
| --- | --- |
| `-i`, `--in-place` | rewrite the file argument in place |
| `-v`, `--verbose` | emit progress and diagnostic information |
| `-h`, `--help` | usage |

Dispatch:

| Arguments | Behavior |
| --- | --- |
| no args | read stdin, write stdout |
| one file arg | read the file, write stdout |
| one file arg + `-i` | rewrite the file atomically (temp file in the same directory, `os.Rename` over the original, mode preserved) |
| `-i` without a file arg | error |

## Library

```go
import "github.com/michaelquigley/unfurl"

func Unfurl(r io.Reader, w io.Writer) error
func UnfurlBytes(src []byte) ([]byte, error)
```

The public API is intentionally small. There is no exported configuration, no exported types beyond the two functions, and no exported errors — failures are stdlib `fmt.Errorf` wrappings of the underlying I/O error. The tool does one thing.

## Scenarios

A wrapped list item:

```
- this is a list item whose body
  has been wrapped across multiple
  physical lines
- second item
```

becomes:

```
- this is a list item whose body has been wrapped across multiple physical lines
- second item
```

A wrapped blockquote:

```
> a quoted paragraph that has been
> wrapped at the source. each line
> begins with the marker.
```

becomes:

```
> a quoted paragraph that has been wrapped at the source. each line begins with the marker.
```

A paragraph with an intentional hard break:

```
this sentence is followed by a deliberate pause  
and continues here with the surrounding text wrapped
across two physical source lines.
```

becomes:

```
this sentence is followed by a deliberate pause  
and continues here with the surrounding text wrapped across two physical source lines.
```

The hard break — two trailing spaces — is preserved exactly. The paragraph reflows around it.

A table passes through unchanged regardless of how its rows happen to look:

```
| header a | header b |
| ---      | ---      |
| cell 1   | cell 2   |
| cell 3   | cell 4   |
```

stays byte-identical in the output.

An already-unwrapped grimoire note produces byte-identical output. Running `unfurl` twice is the same as running it once.

## Out of scope

The surgical scope is the whole virtue of the tool. The following are deliberately absent and won't be added:

- **Re-wrapping at a target column.** `unfurl` only unwraps. Re-wrapping would just recreate the problem the tool exists to solve. If a wrap-at-column tool is ever wanted, it's a separate utility.
- **Normalization of other formatting choices.** Bullet style (`-`/`*`/`+`), emphasis markers (`*`/`_`), heading style (ATX vs setext), table column padding, ordered-list numbering — none of these are touched. Adding any of them puts `unfurl` on a path toward becoming prettier or mdformat, and those already exist.
- **Linting and validation.** `unfurl` does not warn about malformed markdown, unclosed fences, broken reference links, or any other quality issue. Malformed input is treated the way CommonMark treats it — an unclosed fence, for instance, extends to EOF.
- **Configuration.** No config file, no rc file, no environment-variable overrides. Behavior is a fixed contract.
- **Recursive directory mode.** Compose with `find` and `xargs` or a shell loop.
- **File-watching mode.** Compose with `entr`, `fswatch`, or editor integrations.
- **Editor integrations.** The library API is the integration point. Specific Obsidian, VS Code, or Neovim plugins are separate projects.

## Development

```sh
make test    # go test ./... -count=1 && go vet ./...
make build   # go install ./cmd/unfurl
```

The runtime transform is hand-rolled (line classifier → block grouper → emit). Goldmark is a test-only dependency, used to verify an AST-equivalence property modulo soft-break normalization on every fixture in `testdata/property/`.

Architecture reference: [`docs/current/design.md`](docs/current/design.md).
