# Chunk 1 — Skeleton and CLI Shell

Implemented the initial project skeleton for `unfurl`:

- `unfurl.go` exposes `Unfurl` and `UnfurlBytes` as pass-through stubs.
- `cmd/unfurl/main.go` provides the cobra root command, `-i` / `--in-place`, `-v` / `--verbose`, `df/dl` initialization, stdin/file/stdout dispatch, and same-directory atomic in-place writes.
- `Makefile` defines `build`, `test`, and `clean`.
- `README.md` documents the current CLI and development commands.
- Tests cover the pass-through public API and CLI paths for stdin, file input, in-place rewrite, missing file, and `--in-place` without a file argument.

No deviations from the work order. The transform is still a no-op until Chunk 2 and later pipeline work land.
