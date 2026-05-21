# unfurl

A surgical markdown unwrap utility. Collapses soft line breaks inside paragraph content; preserves every other CommonMark/GFM construct byte-for-byte. Single Go binary, one operation, no configuration. Library-callable as well as CLI.

## Status

Pre-1.0. The eight implementation chunks from `docs/future/unfurl-work-order.md` have landed. The design rationale lives at `docs/future/unfurl-spec.md`; the work order remains the record of intended implementation. `docs/current/` contains chunk-level notes describing what was actually built and any deviations.

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
- **Hand-rolled tokenizer for the transform, goldmark in tests only.** This split is deliberate — see the work order for the reasoning. If the property test starts failing on real grimoire content with more than ~3 nontrivial classification bugs, escalate to discuss switching to goldmark before accumulating fixes.
- **Hard-break markers are exact bytes.** Three trailing spaces in → three trailing spaces out. Never normalize a trailing-space run down to two.
- **The spec's "Deferred" section is binding.** No re-wrapping at a target column, no formatting normalization (bullets, emphasis, headings, table padding), no linting, no recursive mode, no editor plugins, no configuration files. Out of scope means out of scope; resist the urge to add a flag.

## For implementation agents

Read order: `docs/future/unfurl-spec.md` → `docs/future/unfurl-work-order.md` → this file. The work order's "Slicing" section names the eight chunks and their dependencies; each has explicit acceptance criteria.

Don't re-litigate decisions made during planning or mercurius review — if a question arises that the spec or work order doesn't answer, surface it rather than improvise. Significant unresolved questions are a signal that convergence wasn't real, which means escalation, not invention.

As chunks land, write brief notes to `docs/current/` describing what was built and any deviations from the work order. The spec and work order in `docs/future/` stay as record of intent.

## For agents arriving cold

This is `unfurl`, part of Michael Quigley's broader software practice. Related projects (`lore`, `archive`, `frame`, `mercurius`, etc.) live as siblings in `~/Repos/q/products/`. The grimoire at `~/Repos/q/writing/grimoire` is the canonical home for design and orientation docs that have demonstrated cross-project relevance; for unfurl specifically, the in-repo `docs/` directory is authoritative.

If you've been assigned a specific role (design / planning / implementation), read the grimoire's `practice/creative/agent-roles.md` for role-specific orientation.
