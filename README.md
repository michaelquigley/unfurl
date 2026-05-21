# unfurl

`unfurl` is a surgical markdown unwrap utility. It collapses soft line breaks inside paragraph content while preserving every other CommonMark/GFM construct byte-for-byte.

It reflows paragraphs at top level and inside list or blockquote containers, preserves hard breaks, preserves line endings, BOMs, trailing-newline absence, code blocks, HTML blocks, front matter, reference definitions, and GFM tables. The runtime transform is hand-rolled; goldmark is used only in tests for AST-equivalence and idempotence coverage.

## Usage

```sh
unfurl [flags] [file]
```

With no file argument, `unfurl` reads stdin and writes stdout. With a file argument, it reads that file and writes stdout. With `-i` / `--in-place`, it rewrites the file argument atomically.

```sh
unfurl note.md
unfurl -i note.md
cat note.md | unfurl
```

## Development

```sh
make test
make build
```

The public API is intentionally small:

```go
func Unfurl(r io.Reader, w io.Writer) error
func UnfurlBytes(src []byte) ([]byte, error)
```
