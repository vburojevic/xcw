# Contributing to xcw

Thanks for your interest in improving **XcodeConsoleWatcher (xcw)**.

## Getting started

1. Fork and clone the repo.
2. Install Go **1.25.5** (or rely on the `toolchain` directive in `go.mod`).
3. Run the tests:
   ```sh
   make test
   ```

## Development workflow

- Keep changes focused and small when possible.
- Prefer fixing root causes over adding surface patches.
- Run `make lint` and `make test` before opening a PR.
- If you change CLI flags or NDJSON output, also regenerate:
  ```sh
  make docs
  make schema
  ```

## Code style

- Follow existing patterns in `internal/*`.
- Avoid adding new dependencies unless clearly justified.
- Keep NDJSON output backward‑compatible; bump schema version only for breaking changes.

## Beads issues

This repo uses **Beads** for issue tracking.

- List issues: `bd list`
- Create issue: `bd create "Title"`
- Update status: `bd update <id> --status in_progress|closed`

## Submitting changes

1. Create a feature branch.
2. Commit with a clear message.
3. Open a PR against `main` describing:
   - What you changed and why.
   - How to reproduce/verify.
   - Any schema or docs updates.

## Reporting bugs / feature requests

Open a Beads issue (preferred) or a GitHub issue. Include:

- `xcw version`
- macOS + Xcode version
- Simulator/device name & runtime
- Repro steps and expected vs actual behavior
