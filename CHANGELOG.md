# Changelog

All notable changes to XcodeConsoleWatcher (xcw) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Test: `xcw tail` integration-style coverage using a stubbed `xcrun` to validate `--max-logs` cutoffs.

### Fixed
- `--max-level` is now truly optional: when unset, `tail`/`query`/`watch` no longer apply an implicit maximum that could drop `error`/`fault` logs.

## [0.19.2] - 2025-12-14

### Added
- CI benchmark regression guard on pull requests (compares base vs head benchmarks for hot packages).
- Unit tests for resume state helpers.

### Changed
- README now includes concrete recipes for `xcw watch` triggers and `xcw tail --resume`, and clarifies `xcw query` data source.

### Fixed
- `xcw tail --resume` backfill is best-effort on restart/reconnect; failures no longer abort tailing and warnings include gap context.

## [0.19.1] - 2025-12-14

### Changed
- Contributor docs now require using Beads CLI (`bd`) for task management (`AGENTS.md`, `CLAUDE.md`).

## [0.19.0] - 2025-12-14

### Added
- CI now runs on macOS as well as Linux, including docs/schema drift checks.
- Shell completion scripts are generated from the Kong CLI model to avoid drift.
- `xcw watch` now supports `--where`, `--dedupe`, `--dedupe-window`, repeatable `--exclude`, and `--process`.
- NDJSON `trigger_result` plus correlation fields (`tail_id`, `session`, `trigger_id`) for watch triggers.
- `xcw tail --resume` to optionally backfill gaps on restart/reconnect, emitting NDJSON `gap_detected`/`gap_filled`.

### Changed
- Watch triggers now emit start (`trigger`) + completion (`trigger_result`) events, with `trigger_error` reserved for failures.

### Fixed
- Removed unused `embedfiles` package and cleaned up dead `internal/app` directory.

## [0.18.0] - 2025-12-14

### Added
- NDJSON `stats` event emitted alongside heartbeats with stream diagnostics (reconnects, parse drops, backpressure).
- `xcw config show` now includes config provenance (config file + per-key sources: flag/env/config/default).
- Parser fixture coverage and fuzz tests for NDJSON parsing and `--where` expressions.

### Changed
- Unified simulator selection and `--booted` semantics across commands; improved multiple-booted error guidance (`xcw pick simulator`).
- Stream/query robustness: timeouts for `simctl` operations and queries; verbose mode surfaces `xcrun` stderr.
- Heartbeats now always include `contract_version` for agent consumers.
- Expanded JSON schema coverage and refreshed machine-readable help.

### Fixed
- Removed duplicate application of pattern/exclude filters across tail/query/watch pipelines.
- Cleaned up generated sample config YAML and config path guidance.

## [0.17.1] - 2025-12-14

### Added
- Goleak-based goroutine leak checks in the test suite.

### Changed
- Faster simulator NDJSON parsing via `gjson` field extraction (reduced allocations vs full unmarshal).
- Commands now use `signal.NotifyContext` for shutdown handling; `watch` trigger processes are cancelled and awaited on exit.

## [0.17.0] - 2025-12-14

### Added
- Explicit `--all` guardrail for running `tail`/`query`/`watch`/`ui` without `--app` or `--predicate` (intentionally broad output).
- Boolean `--where` expressions: `AND`/`OR`/`NOT`, parentheses, and regex literals like `message~/timeout|crash/i`.

### Changed
- `--app` is no longer required when `--predicate` is provided (or when `--all` is set) for `tail`/`query`/`watch`/`ui`; `tail` disables session tracking when `--app` is omitted.
- Improved structured error hints (NDJSON `error.hint`) for common simulator selection and Xcode tooling failures; `ui` now emits consistent structured errors.

## [0.16.0] - 2025-12-11

### Added
- Rotation NDJSON event (`rotation`) emitted on per-session file switches, with schema update.
- Safer watch triggers via `--trigger-no-shell` to bypass `sh -c` when desired.
- Flexible filtering: numeric `--where` comparisons for `pid`/`tid`, quoted values, and glob/regex process matching.
- Hot-path benchmarks for parser, filters, and analyzer.
- Contributor docs: `CONTRIBUTING.md`, `ROADMAP.md`, and `AGENTS.md`.

### Changed
- Simulator device listing now uses a short TTL cache to reduce `simctl` overhead.
- Streamer reconnect jitter uses per-stream RNG; shutdown waits for goroutines and closes channels cleanly.
- Timestamp parsing supports arbitrary offsets/fractional precision; verbose mode surfaces parse drops.
- Analyzer no longer depends on `samber/lo`, reducing dependency surface.
- TUI performance improved with incremental filtering/viewport updates; added follow/details toggles and search highlighting.
- Config load validates enums/durations/limits early and fails fast with structured errors.

## [0.15.1] - 2025-12-11

### Fixed
- Restored default log level to `debug` when no CLI level is provided, preventing tail from dropping debug entries after the Viper config migration.

## [0.15.0] - 2025-12-11

### Added
- Viper-backed configuration with env-prefix `XCW_`, nested key replacer, and per-command sections (`tail`, `query`, `watch`).
- Config show now prints tail/query/watch defaults; README documents precedence and sample config tuned for AI agents.
- Tests for env overrides, config precedence, and apply-default helpers for tail/query/watch, plus Globals config fallback coverage.

### Changed
- CLI format/level/quiet now honor config when CLI flags stay at their defaults.
- Tail/query/watch commands consume config defaults automatically to reduce repeated flags for agents.

## [0.14.1] - 2025-12-11

### Added
- Clock injection (benbjohnson/clock) for deterministic timers in tail/watch.
- Zap-based verbose logger carrying tail_id/session.
- Lipgloss-styled text output for human mode.
- Session_debug NDJSON event and schema entry; embedded docs/schema via go:embed for offline use.
- CI workflow with golangci-lint and doc/schema drift check.
- Auto-disable styling when stdout is not a TTY to keep piped logs clean.

### Changed
- Heartbeats pooled to reduce allocations; filter pipeline reused across commands.
- Dedupe respects entry timestamps for windowed mode.
- Structured errors now support optional hint field.

## [0.13.1] - 2025-12-11

### Added
- Generated machine-readable help (`docs/help.json`) and schema (`schemas/generated.schema.json`) to keep docs in sync.
- Added tests covering filter pipeline, tail flag parsing, session tracker binary UUID detection, help/schema drift, and dedupe windowing.
- Session debug NDJSON event (verbose) and structured errors now include optional hints.
- Filter pipeline reused, tail flags grouped, flag doctor helper, heartbeat pool allocation reduction.

## [0.13.0] - 2025-12-11

### Added
- Filter pipeline abstraction (`internal/filter/pipeline.go`) and adoption in tail for extensible predicate chaining (pattern/exclude/where).
- Grouped tail flags into logical structs while preserving flag names for agents.
- Scripts: `scripts/gen-readme.sh` (help JSON regeneration) and `scripts/gen-schema.sh` (schema refresh); Makefile targets `docs` and `schema`.

### Changed
- Session tracker refactored with clearer relaunch decision helper.
- NDJSON/test improvements from 0.11.x retained.

## [0.11.4] - 2025-12-11

### Changed
- Launch command now uses shared error emitter; minor struct tag cleanup.
- Examples JSON schema fields aligned for agents.

## [0.11.3] - 2025-12-11

### Added
- Common error emitter ensures consistent NDJSON/text error output across commands.

### Changed
- Cleaned unused imports and standardized flag validation paths using shared helper.

## [0.11.2] - 2025-12-11

### Added
- Context-aware debug logger prefixes tail/session ids for agent-friendly diagnostics.
- NDJSON writer now disables HTML escaping to reduce overhead and keep raw log text intact.
- Lifecycle NDJSON snippet test covering heartbeat, cutoff, and reconnect markers.

### Changed
- `tail`/`watch` flag validation now rejects quiet+text and enforces ndjson for dry-run-json.
- `watch` uses shared filter helpers and level resolver.

## [0.11.1] - 2025-12-11

### Changed
- `xcw watch` now uses shared filter/level helpers for consistent pattern/exclude handling and min/max level overrides across commands.

## [0.11.0] - 2025-12-11

### Added
- Shared filter helpers now power `xcw query` for consistent pattern, exclude, and where clause handling across commands.
- Unit tests cover filter compilation and level resolution.

### Changed
- `xcw query` now reports filter errors consistently and honors min/max level overrides using the common resolver.
- README updated to clarify AI-agent friendly filtering behavior.

## [0.10.0] - 2025-12-11

### Added
- Agent-facing NDJSON events: `metadata`, `cutoff_reached`, `reconnect_notice`, `clear_buffer` hints, and enriched `heartbeat` with `last_seen_timestamp`.
- `--machine-friendly` preset, `handoff` command for AI context handoff, `--dry-run-json` for tail, `--max-duration`/`--max-logs` cutoffs with structured markers.

### Changed
- Ready/heartbeat timestamps now UTC; tail emits metadata on start.

## [0.8.0] - 2025-12-10

### Added
- `tail_id` added to all tail outputs (log, session_start/end, heartbeat, summary) so agents can correlate a single tail invocation.
- New `--session-idle` flag to force session boundaries after periods of inactivity.
- Session metadata now includes app version/build and binary UUID to detect reinstalls/hot-redeploys.
- Per-session file rotation when using `--output` or `--session-dir` (one file per app run).

### Changed
- `xcw tail` now emits a final `session_end` when the stream stops, ensuring the last run is closed.

## [0.5.2] - 2024-12-09

### Changed
- Improved documentation to make `xcw tail` command more prominent
  - Added "Start Here" section to README with explicit `-s` and `-a` flags
  - Updated quick start guide to emphasize the primary command
  - Made agent guidance more prescriptive: "ALWAYS START WITH: xcw tail"
  - All examples now include simulator name (`-s`) parameter

## [0.5.1] - 2024-12-09

### Changed
- Default log level changed from `default` to `debug` - now shows all logs by default
  - Use `-l error` to filter to errors only
  - Use `-l default` for previous behavior

## [0.5.0] - 2024-12-09

### Added
- `--wait-for-launch` flag for `xcw tail` - Start log capture before app launches
  - Emits `{"type":"ready"}` event when log capture is active
  - AI agents can wait for this event, then trigger build/run process
  - Ensures no startup logs are missed
- New `ready` output type documenting when log capture is active

### Changed
- Updated README with "Pre-launch log capture" section
- Updated help.go with ready output type documentation

## [0.4.0] - 2024-12-09

### Added
- `xcw launch` command - Launch app and capture stdout/stderr (print statements)
  - Captures print() output not available via unified logging
  - Outputs NDJSON with type "console" and stream "stdout" or "stderr"
  - Supports `--terminate-existing` to kill existing app instance first
  - Supports `-w/--wait` to wait for debugger to attach

### Changed
- Updated documentation to explain difference between `tail` (unified logging) and `launch` (stdout/stderr)
- Added recommendation to use Logger/os_log instead of print() for best xcw compatibility

## [0.3.0] - 2024-12-09

### Added
- Quick start guide shown when running `xcw` with no arguments
- AI agent hint in `--help` output pointing to `xcw help --json`

### Changed
- Streamlined README with cleaner structure and concise examples

## [0.2.0] - 2024-12-09

### Added
- `xcw examples` command - Show usage examples for all commands
  - `xcw examples [command]` - Examples for specific command
  - `xcw examples --json` - Machine-readable format for AI agents
- `xcw help --json` - Complete CLI documentation as JSON for AI agents
  - All commands with usage and examples
  - Output types with example values
  - Error codes with recovery steps
  - Workflow patterns (Codex streaming, Claude Code background, etc.)
- `xcw update` command showing upgrade instructions for Homebrew and Go install
- `xcw sessions` command for managing session log files
  - `xcw sessions list` - List session files sorted by date
  - `xcw sessions show` - Show path to a session file (for piping)
  - `xcw sessions clean` - Delete old session files, keeping most recent
- Session-based file output for `tail` and `watch` commands
  - `--session-dir` - Directory for session files (default: ~/.xcw/sessions)
  - `--session-prefix` - Prefix for session filename (default: app bundle ID)
  - Files are named with timestamps: `20251209-153045-com.example.app.ndjson`
- File output support for `watch` command (`--output`, `--session-dir`, `--session-prefix`)
- CHANGELOG.md for version history tracking

### Changed
- Simplified README AI agent examples with explicit `-s` simulator flag
- File output now creates fresh per-session files instead of rotating logs
- Removed lumberjack dependency for simpler, lighter codebase

### Removed
- **Breaking:** `--rotate-size` and `--rotate-count` flags from `tail` command
  - Use `--session-dir` for automatic timestamped session files instead
  - For explicit file output, use `--output` (no rotation)

## [0.1.0] - 2024-12-09

### Added
- **Core Commands**
  - `xcw tail` - Real-time log streaming with auto-reconnection
  - `xcw query` - Historical log queries with `--since` and `--until` flags
  - `xcw list` - List available simulators
  - `xcw apps` - List installed apps on a simulator
  - `xcw pick` - Interactive fuzzy picker for simulators and apps
  - `xcw watch` - Pattern-triggered command execution
  - `xcw summary` - Log statistics summary
  - `xcw analyze` - Analyze recorded NDJSON log files
  - `xcw replay` - Replay recorded sessions with timing
  - `xcw doctor` - System diagnostics and setup verification
  - `xcw schema` - JSON Schema output for all NDJSON types
  - `xcw config` - Configuration management
  - `xcw ui` - Interactive TUI log viewer
  - `xcw version` - Version information
  - `xcw completion` - Shell completions (bash, zsh, fish)

- **Output Formats**
  - NDJSON output optimized for AI agent consumption
  - Text output for human readability
  - Schema versioning for forward compatibility

- **Filtering**
  - App bundle ID filtering (`-a`)
  - Log level filtering (`--level`, `--min-level`, `--max-level`)
  - Regex pattern matching (`-p`, `-x`)
  - Subsystem and category filters

- **Advanced Features**
  - Pattern persistence for tracking known vs new errors
  - File output with session-based naming (`--output`, `--session-dir`)
  - Heartbeat messages for connection health
  - Periodic summary markers
  - Tmux integration for persistent sessions
  - Trigger commands on error patterns

- **Documentation**
  - Comprehensive README with AI agent examples
  - NDJSON types reference table
  - Error codes reference (29 codes)
  - End-to-end integration examples

### Changed
- Module path aligned to `github.com/vburojevic/xcw`

### Fixed
- Reliability issues with simulator reconnection
- Config loading with clear precedence order
