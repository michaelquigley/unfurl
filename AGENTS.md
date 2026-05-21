# unfurl

A surgical markdown unwrap utility. Collapses soft line breaks inside paragraph content; preserves every other CommonMark/GFM construct byte-for-byte. Single Go binary, one operation, no configuration. Library-callable as well as CLI.

## Status

Pre-1.0. Feature-complete; the runtime transform and property-test safety net are both in place. Architecture reference lives at `docs/current/design.md`; user-facing description and CLI / library shape are in `README.md`.

## Stack

- Go 1.26+, single module `github.com/michaelquigley/unfurl`.
- `github.com/spf13/cobra` for the CLI surface. No fang wrapper.
- `github.com/michaelquigley/df/dl` for logging, initialized in `cmd/unfurl/main.go`'s `init()` with `SetTrimPrefix("github.com/michaelquigley/")`. The prefix matches *this* module's namespace, not the sibling-project `git.hq.quigley.com/products/` convention used by lore and archive.
- `github.com/yuin/goldmark` plus `go.abhg.dev/goldmark/frontmatter` are test-only dependencies, used in `unfurl_property_test.go` for the AST-equivalence safety net. Not part of the runtime binary.
- No `df/dd`. unfurl has no configuration surface to bind.

## Layout

- `cmd/unfurl/main.go` — CLI entry, cobra root, `-i/--in-place` and `-v/--verbose` flags, atomic in-place write helper.
- `unfurl.go` (package `unfurl`) — public API: `Unfurl(r io.Reader, w io.Writer) error` and `UnfurlBytes(src []byte) ([]byte, error)`.
- `internal/tokenize/` — Pass 1 line classifier and Pass 2 block grouper (container-context stack, lazy continuation, paragraph-interruption matrix, retroactive setext / GFM-table reclassification, HTML-block start/end/interrupt rules per type).
- `internal/reflow/` — paragraph reflow with hard-break preservation and per-segment container-prefix emission.
- `internal/emit/` — Pass 3 block-tree to bytes writer.
- `testdata/property/` — seed corpus for the AST-equivalence property test.

## Key conventions

- **Bytes-first.** The transform's correctness is defined at the byte level — every non-paragraph block byte-for-byte; idempotence is structural, not coincidental. When in doubt about whether to normalize something, don't.
- **Byte fidelity is total.** Detect input's line-ending discipline (LF vs CRLF) and emit in kind. Preserve UTF-8 BOM via strip-and-reattach (strip before classification so first-line constructs are recognized; re-emit before output). Preserve absence of trailing newline.
- **Hand-rolled tokenizer for the transform, goldmark in tests only.** The split is deliberate: line-oriented problem, line-oriented transform, with the AST-equivalence property test as the safety net. If the property test starts failing on real grimoire content with more than ~3 nontrivial classification bugs, escalate to discuss switching to goldmark before accumulating fixes.
- **Hard-break markers are exact bytes.** Three trailing spaces in → three trailing spaces out. Never normalize a trailing-space run down to two.
- **README's "Out of scope" section is binding.** No re-wrapping at a target column, no formatting normalization (bullets, emphasis, headings, table padding), no linting, no recursive mode, no editor plugins, no configuration files. Out of scope means out of scope; resist the urge to add a flag.

## For implementation agents

Read order for someone modifying the transform: `README.md` → `docs/current/design.md` → this file → the code. The design doc covers the three-pass model, the paragraph-interruption matrix, retroactive setext/table reclassification, the reflow segment model, and the byte-fidelity rules. The original spec and work order have been retired into those synthesized docs.

When in doubt about whether to normalize something, don't. The byte-fidelity contract is total and idempotence is structural, not coincidental. Significant unresolved questions are a signal to surface rather than improvise.

## For agents arriving cold

This is `unfurl`, part of Michael Quigley's broader software practice. Related projects (`lore`, `archive`, `frame`, `mercurius`, etc.) live as siblings in `~/Repos/q/products/`. The grimoire at `~/Repos/q/writing/grimoire` is the canonical home for design and orientation docs that have demonstrated cross-project relevance; for unfurl specifically, the in-repo `docs/` directory is authoritative.

If you've been assigned a specific role (design / planning / implementation), read the grimoire's `practice/creative/agent-roles.md` for role-specific orientation.
