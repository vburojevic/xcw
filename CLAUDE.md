# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
# Build
make build                    # Build binary to ./xcw
go build -o xcw ./cmd/xcw     # Alternative direct build

# Test
make test                     # Run all tests
go test ./...                 # Run all tests directly
go test -v ./internal/simulator/...  # Run specific package tests
go test -run TestMultiple ./internal/simulator/...  # Run specific test

# Lint
make lint                     # Run golangci-lint (requires installation)
go vet ./...                  # Basic static analysis

# Format
make fmt                      # Format with gofmt and goimports

# Install
make install                  # Install to /usr/local/bin
```

## Task Management (Beads)

This repo uses Beads for task tracking. Always manage work via the Beads CLI (`bd`) rather than ad-hoc TODO lists in chat.

```bash
# Prefer deterministic direct mode (and use it if daemon paths mismatch)
bd --no-daemon list --long

# Create a task/feature/bug
bd --no-daemon create --type task --priority P2 --title "Short title" --description "What/why + acceptance criteria"

# Start work / add progress notes
bd --no-daemon update <issue-id> --status in_progress
bd --no-daemon comment <issue-id> "Progress update + verification commands (e.g. go test ./...)"

# Close when done
bd --no-daemon close <issue-id>
```

Do not hand-edit `.beads/*` files; use `bd` to avoid database/JSONL drift.

## Architecture Overview

XcodeConsoleWatcher (xcw) is a CLI tool for streaming iOS Simulator console logs, optimized for AI agent consumption via NDJSON output.

### Core Flow

1. **CLI Layer** (`internal/cli/`) - Kong-based command parsing. Each command (tail, query, watch, list) is a separate file with a struct and `Run(globals *Globals)` method.

2. **Simulator Layer** (`internal/simulator/`) - Interfaces with `xcrun simctl`:
   - `manager.go` - Device discovery, boot, state management
   - `streamer.go` - Real-time log streaming via `log stream --style ndjson`
   - `query.go` - Historical logs via `log show --style ndjson`
   - `parser.go` - NDJSON parsing and timestamp conversion

3. **Output Layer** (`internal/output/`) - Formats output:
   - `ndjson.go` - Structured JSON output for AI consumption
   - `styles.go` - Lipgloss styling for text mode
   - `analyzer.go` - Log summarization and pattern detection

4. **Filter Chain** (`internal/filter/`) - Composable log filters (level, app, regex, exclude).

5. **Tmux Integration** (`internal/tmux/`) - Persistent session management for background log viewing.

### Key Patterns

- **Simulator Selection**: Commands use `--simulator` (name/UDID) and `--booted` flags, mutually exclusive. When neither is set, auto-detects single booted simulator or errors if multiple.

- **Output Format**: Default is NDJSON for AI agents. Use `-f text` for human-readable output.

- **Global Config**: `internal/config/` handles `.xcwrc` files and environment variables, merged with CLI flags.

- **Verbose/Debug**: Use `globals.Debug(format, args...)` for debug output controlled by `-v` flag.

### Adding a New Command

1. Create `internal/cli/<command>.go` with struct and `Run(globals *Globals) error` method
2. Add field to `CLI` struct in `internal/cli/root.go`
3. For simulator-based commands, use `mgr.FindBootedDevice()` or `mgr.FindDevice()` pattern

### Test Patterns

Tests use testify (`assert`, `require`). See `internal/simulator/manager_test.go` for examples of testing error types and utility functions.

## Development Workflow

**After every code change:**

1. **Write/update tests** - Add tests for new functionality in `*_test.go` files alongside the code
2. **Update README.md** - Document new flags, commands, or behavior changes
3. **Run tests** - `go test ./...` must pass before committing
4. **Commit and push often** - After completing any logical unit of work, commit and push to keep the repository up to date

## Versioning & Changelog

**Version management:**
- Version is injected at build time via Makefile ldflags from git tags
- To release a new version: `git tag v1.2.3 && git push --tags`
- Use semantic versioning: MAJOR.MINOR.PATCH

**Changelog maintenance:**
- Update `CHANGELOG.md` when adding features, fixing bugs, or making breaking changes
- Follow [Keep a Changelog](https://keepachangelog.com) format
- Add entries under `[Unreleased]` section during development
- Move unreleased entries to versioned section when tagging a release

**When to update CHANGELOG.md:**
- New commands or flags
- Bug fixes
- Breaking changes
- Significant improvements
- Dependency updates that affect functionality

## Releasing

**After completing work and pushing to main, create a new release:**

1. **Update CHANGELOG.md** - Move items from `[Unreleased]` to new version section
2. **Commit the changelog update**
3. **Create and push tag:**
   ```bash
   git tag v0.X.0
   git push origin v0.X.0
   ```

GoReleaser will automatically:
- Build macOS binaries (Intel + Apple Silicon)
- Create GitHub release with artifacts
- Update Homebrew formula in `vburojevic/homebrew-tap`

**Users can install/upgrade via Homebrew:**
```bash
brew tap vburojevic/tap
brew install xcw
# or upgrade existing
brew upgrade xcw
```
