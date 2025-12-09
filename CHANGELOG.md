# Changelog

All notable changes to XcodeConsoleWatcher (xcw) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
