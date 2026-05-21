# Chunk 6 — Container Contexts

Implemented paragraph handling inside blockquote and list-item containers:

- Lines now carry an emitted `Prefix` separate from their classification/content bytes.
- Raw emit writes `Prefix + Raw`, preserving container-prefixed non-paragraph lines byte-for-byte.
- Reflow captures the prefix from the first line in each hard-break segment, so later segments can emit with their own source-correct continuation prefix.
- Blockquote grouping strips one `>` marker layer and recurses, including lazy continuation lines when a paragraph is open.
- List grouping strips the item marker / continuation indent and recurses, including lazy continuation lines when a paragraph is open.
- Nested blockquotes, lists inside blockquotes, blockquotes inside lists, and nested lists are handled by recursive grouping.
- Setext-shaped lazy continuation lines from a different prefix context are demoted to paragraph continuation instead of retroactively reclassifying the outer paragraph as a setext heading.

Tests cover the spec list-item and blockquote scenarios, blockquote and list lazy continuation, nested blockquote/list combinations, list hard breaks with continuation-prefix segment emission, prefix metadata in the grouper, and prefix capture in reflow.

Remaining major work follows the work order: goldmark property/idempotence tests in Chunk 7, then BOM/CRLF/trailing-newline edge polish in Chunk 8.
