# Chunk 7 — Property Test and Goldmark Integration

Implemented the AST-equivalence safety net:

- Added test-only `github.com/yuin/goldmark` and `go.abhg.dev/goldmark/frontmatter` dependencies.
- Added `unfurl_property_test.go`, which parses input and output with goldmark configured for GFM plus front matter.
- The comparator canonicalizes the AST recursively, compares node kinds and kind-specific attributes, and normalizes only authorized soft-break seams to a single ASCII space.
- Byte-significant nodes such as fenced code, indented code, HTML blocks, raw HTML, code spans, and reference definitions include source text or structural attributes in the comparison.
- Front matter is compared through parsed metadata because the frontmatter extension removes the node after storing data in parser context.
- Property fixtures live under `testdata/property/`; the future spec document is also included by path.
- Idempotence now runs over the same fixture set.
- `TestScenarioClaudeOutput` uses the Claude-output fixture and asserts both line-count reduction and AST equivalence.

The property fixture corpus includes spec scenarios, comparator-tightness cases for intra-line multi-space, intra-line tab, and single trailing space at a soft wrap, plus six grimoire-style mixed notes.

Deviation from the work order: the AST-property corpus does not include the mismatched table-delimiter and bare/setext-shaped continuation fixtures as property cases, because goldmark parses those source documents differently from the stricter work-order policy. Those behaviors remain covered by focused grouper/API tests. BOM, CRLF, and trailing-newline-presence fixtures remain for Chunk 8.
