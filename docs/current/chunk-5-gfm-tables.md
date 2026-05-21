# Chunk 5 — GFM Table Detection

Implemented retroactive GFM table grouping:

- Added `BlockTable`; emit already preserves it byte-for-byte as a non-paragraph block.
- A paragraph line followed by a candidate delimiter row becomes a table only when the delimiter row has valid GFM delimiter-cell shape and the header/delimiter cell counts match.
- Header and delimiter cell counts split on unescaped pipes while ignoring a single leading and trailing pipe.
- Once a valid table starts, body rows accumulate permissively until a blank line or paragraph-interrupting block start.
- Body rows are preserved regardless of pipe count, including pipeless rows and rows with fewer or more cells than the header.
- Invalid header/delimiter candidates stay paragraph content and reflow normally.

Tests cover matching tables, pipeless body rows, varying body cell counts, body termination at blanks and block starts, arity mismatch, invalid delimiter shape, escaped pipes in headers, public table preservation, and invalid-candidate paragraph reflow.

No list or blockquote container behavior was added in this chunk; that remains assigned to Chunk 6.
