# Chunk 3 — Block Grouper and Emit

Implemented the first end-to-end transform pipeline:

- `internal/tokenize.Group` turns classified lines into flat block records for paragraphs, blanks, front matter, ATX headings, setext headings, thematic breaks, fenced code, indented code, HTML blocks, reference definitions, and temporary raw pass-through blocks for list and blockquote lines.
- Setext headings are retroactively reclassified from an open paragraph plus underline, while empty list-marker lines remain paragraph continuations.
- Fenced code, front matter, indented code, reference definitions, and HTML blocks are grouped so their raw bytes pass through unchanged.
- `internal/emit.Emit` writes non-paragraph blocks byte-for-byte and uses the Chunk 3 paragraph stub to join soft-wrapped paragraph lines with a single space.
- `Unfurl` now runs tokenize → group → emit instead of the Chunk 1 pass-through stub.

Tests cover flat paragraph grouping, setext reclassification, all seven HTML block end conditions, front matter, fenced and indented code, reference definitions, type-7 HTML non-interruption, paragraph non-interruption cases for indented code and empty/non-1 list markers, public API reflow, CLI reflow, and construct preservation for the non-container constructs in scope.

Known remaining work matches the work order: hard-break preservation is Chunk 4, GFM table reclassification is Chunk 5, list and blockquote container behavior is Chunk 6, property tests are Chunk 7, and full byte-fidelity edge cases are Chunk 8.
