# Chunk 4 — Hard-Break Preservation and Reflow Polish

Implemented hard-break-aware paragraph reflow:

- `internal/reflow` now owns paragraph segment modeling and emission.
- Paragraphs split into segments at hard breaks.
- Hard-break markers are re-emitted as exact source bytes, including three-or-more trailing-space markers and backslash markers.
- Soft wraps inside each segment still join with a single ASCII space.
- A single trailing space at a soft wrap is dropped before the join, while trailing space on a final single-line paragraph is preserved.
- Segment prefixes are represented in the reflow API so the later container-context phase can attach blockquote/list prefixes without changing the hard-break algorithm.
- `internal/emit` delegates paragraph output to `internal/reflow`.

Tests cover single-line and two-line paragraphs, both hard-break marker forms, three- and four-space hard breaks, hard-break idempotence, trailing-whitespace behavior, line-ending preservation inside reflow, segment-prefix emission, emit integration, and public API hard-break scenarios.

No table, list, or blockquote behavior was added in this chunk; those remain assigned to later work-order phases.
