# Repository Guidelines

## Project Structure & Module Organization

- `cmd/xcw/`: CLI entrypoint (`main.go`) and Kong command wiring.
- `internal/`: core implementation, grouped by domain:
  - `cli/` (commands, flag parsing, rotation, schema/help output)
  - `simulator/` (simctl discovery, streaming, parsing)
  - `filter/` (pattern/exclude/where/dedupe pipelines)
  - `output/` (NDJSON/text emitters, analyzer, styles)
  - `session/`, `tmux/`, `tui/`, `config/`, `domain/`
- `docs/help.json` and `schemas/generated.schema.json`: generated + embedded artifacts.
- `scripts/`: regen helpers (`gen-readme.sh`, `gen-schema.sh`).
- `.beads/`: Beads issue tracker data (use `bd --no-daemon` if daemon path mismatches).

## Task Management (Beads)

- Always use Beads CLI (`bd`) for task management (create issues/epics, update status, add comments, close work).
- Prefer `bd --no-daemon` for deterministic local runs (and when daemon paths mismatch).
- Do not hand-edit `.beads/*` JSONL/db files; use `bd` commands to avoid drift and hash mismatches.

## Architecture & Extensibility

- Each command is a Kong struct in `internal/cli/<command>.go` with `Run(*Globals)`, registered in `root.go`.
- Simulator access is via `internal/simulator/Manager` plus `Streamer`/`Query` calling `xcrun simctl`.
- Output defaults to NDJSON for agents (`internal/output/ndjson.go`); `-f text` uses lipgloss styles.
- Filters are composed in `internal/filter` (level/app/regex/exclude/where/dedupe). Keep `--simulator` and `--booted` exclusive.

## Build, Test, and Development Commands

- `make build` / `make build-release`: build local/release binary.
- `make test` / `go test ./...`: run tests.
- `make lint` / `make fmt`: lint + format.
- `make docs schema`: regenerate help/schema after CLI or NDJSON changes.
- `go run ./cmd/xcw <cmd>`: run without installing.

## Coding Style & Naming Conventions

- Use tabs; keep code `gofmt`/`goimports` clean.
- Prefer stdlib over new deps; keep binaries small.
- Keep packages within existing `internal/*` domains.
- NDJSON output is a contract; avoid breaking changes unless bumping `SchemaVersion`.

## Testing Guidelines

- Go `testing` + `stretchr/testify`; name tests `TestXxx`, benches `BenchmarkXxx`.
- After CLI/NDJSON changes, run `make docs schema` and commit regenerated artifacts (CI enforces drift).

## Commit & Pull Request Guidelines

- Commit messages follow Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`, `test:`, `ci:`).
- Update README.md and `CHANGELOG.md` (`[Unreleased]`) for user‑visible changes.
- PRs should link a Beads issue ID, explain intent, list verification commands, and include a small NDJSON/text sample if output changed.

### Releasing

1. Move `[Unreleased]` entries in `CHANGELOG.md` to a new version section and commit.
2. Tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z`.
3. GitHub Actions + GoReleaser build macOS `arm64`/`amd64` binaries, publish the GitHub release, and update the Homebrew tap (`vburojevic/homebrew-tap`). Verify the `Release` workflow succeeded; users upgrade via `brew tap vburojevic/tap && brew upgrade xcw`.

## Configuration Tips

- Precedence: flags → env (`XCW_*`) → config file (`.xcw.yaml`, `~/.config/xcw/config.yaml`) → defaults.
- Validate new config fields in `internal/config` and document them in help/schema.
