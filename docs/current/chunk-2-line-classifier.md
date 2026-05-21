# Chunk 2 — Line Classifier

Implemented pass 1 under `internal/tokenize`:

- `Tokenize` / `TokenizeBytes` read physical lines while preserving raw bytes and line endings.
- A leading UTF-8 BOM is stripped before classification and recorded on the returned document.
- `LineKind`, `Line`, and supporting metadata types capture front matter delimiters, ATX headings, thematic breaks, setext candidates, fences, HTML block starts, blockquotes, list markers, table delimiter rows, indented code, reference definitions, blanks, and paragraph text.
- Hard-break metadata preserves the exact marker bytes for two-or-more trailing spaces and backslash hard breaks.
- Unit tests cover the work-order classifier matrix, including every ATX level, every thematic-break marker, both fence families with and without info strings, all seven HTML block start types, list marker forms, table delimiter alignments, indented-code forms, single-line reference definitions, BOM stripping, raw-line preservation, and hard-break detection.

No public transform wiring changed in this chunk; `unfurl` remains a pass-through until the grouping and emit phases land.
