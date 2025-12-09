# XcodeConsoleWatcher (xcw)

A Go CLI for tailing and watching Xcode iOS Simulator console logs, optimized for AI agents like Claude and Codex.

## Features

- **NDJSON Output**: Structured log output perfect for AI agent consumption
- **Real-time Streaming**: Tail logs with `xcw tail`
- **Historical Queries**: Query past logs with `xcw query`
- **Smart Filtering**: Filter by app bundle ID, log level, regex patterns
- **AI-Friendly Summaries**: Error counts, pattern detection, and analysis markers
- **Tmux Integration**: Persistent sessions for background log watching
- **Auto-Reconnection**: Automatically reconnects if simulator restarts
- **Auto-Boot**: Optionally boots simulator if not running
- **Session-Based File Output**: Per-session timestamped log files
- **Record/Replay**: Capture logs and replay them for analysis
- **Pattern Persistence**: Track known vs new error patterns across sessions
- **System Diagnostics**: Built-in doctor command to verify setup

## Installation

### Homebrew (Recommended)

```bash
brew tap vburojevic/tap
brew install xcw
```

### Go Install

```bash
go install github.com/vburojevic/xcw/cmd/xcw@latest
```

### From Source

```bash
git clone https://github.com/vburojevic/xcw.git
cd xcw
make install
```

## Quick Start

### List Simulators

```bash
# List all simulators
xcw list

# List only booted simulators
xcw list --booted-only

# Output as NDJSON
xcw list -f ndjson
```

### Tail Logs (Real-time)

```bash
# Tail logs from booted simulator (auto-detects single booted simulator)
xcw tail -a com.example.myapp

# Explicitly use booted simulator
xcw tail -a com.example.myapp --booted

# Specific simulator by name
xcw tail -s "iPhone 15 Pro" -a com.example.myapp

# Specific simulator by UDID
xcw tail -s "ABC123-DEF456-..." -a com.example.myapp

# With regex pattern filtering
xcw tail -a com.example.myapp -p "error|warning"

# Exclude noisy logs
xcw tail -a com.example.myapp -x "heartbeat|keepalive"

# Basic streaming (recommended for AI agents)
xcw tail -s "iPhone 17 Pro" -a com.example.myapp

# With tmux for persistent background monitoring
xcw tail -s "iPhone 17 Pro" -a com.example.myapp --tmux

# With heartbeat messages for connection health
xcw tail -a com.example.myapp --heartbeat 10s

# Output to tmux session
xcw tail -a com.example.myapp --tmux

# Output to session file (auto-timestamped)
xcw tail -a com.example.myapp --session-dir ~/.xcw/sessions

# Output to explicit file
xcw tail -a com.example.myapp --output logs.ndjson

# Filter by log level range
xcw tail -a com.example.myapp --min-level info --max-level error
```

### Query Historical Logs

```bash
# Query last 5 minutes
xcw query -a com.example.myapp --since 5m

# Query with analysis
xcw query -a com.example.myapp --since 10m --analyze

# Filter errors only
xcw query -a com.example.myapp --since 1h -l error

# Track patterns across sessions
xcw query -a com.example.myapp --since 1h --analyze --persist-patterns
```

### Record and Analyze Logs

```bash
# Record logs to file
xcw tail -a com.example.myapp --output session.ndjson

# Analyze recorded logs
xcw analyze session.ndjson

# Replay logs with original timing
xcw replay session.ndjson --realtime

# Replay at 2x speed
xcw replay session.ndjson --realtime --speed 2.0
```

### System Diagnostics

```bash
# Check system requirements
xcw doctor

# List installed apps on simulator
xcw apps -s "iPhone 15 Pro"
```

### Tmux Integration

```bash
# Start tailing in tmux session
xcw tail -a com.example.myapp --tmux

# Attach to the session from another terminal
tmux attach -t xcw-iphone-15-pro

# Clear session between test runs
xcw clear --session xcw-iphone-15-pro
```

## Output Format

All NDJSON events include a `schemaVersion` field for forward compatibility. AI agents should check this field and adapt to future schema changes.

### NDJSON Log Entry

```json
{"type":"log","schemaVersion":1,"timestamp":"2024-01-15T10:30:45.123Z","level":"Error","process":"MyApp","pid":1234,"subsystem":"com.example.myapp","category":"network","message":"Connection failed"}
```

### Summary Marker

```json
{"type":"summary","schemaVersion":1,"windowStart":"2024-01-15T10:25:00Z","windowEnd":"2024-01-15T10:30:00Z","totalCount":150,"errorCount":4,"faultCount":1,"hasErrors":true,"hasFaults":true,"errorRate":0.8}
```

### Error Output

```json
{"type":"error","schemaVersion":1,"code":"NO_BOOTED_SIMULATOR","message":"No booted simulator found"}
```

## Commands

### `xcw list`

List available iOS Simulators.

```
FLAGS:
  -b, --booted-only        Show only booted simulators
      --runtime=STRING     Filter by iOS runtime version
```

### `xcw tail` (default)

Stream real-time logs from a simulator.

```
FLAGS:
  -s, --simulator=STRING   Simulator name or UDID
  -b, --booted             Use booted simulator (error if multiple)
  -a, --app=STRING         App bundle identifier (required)
  -p, --pattern=STRING     Regex pattern to filter messages
  -x, --exclude=STRING     Regex pattern to exclude from messages
      --exclude-subsystem  Exclude logs from subsystem (repeatable)
      --subsystem=STRING   Filter by subsystem (repeatable)
      --category=STRING    Filter by category (repeatable)
      --min-level=STRING   Minimum log level (overrides global --level)
      --max-level=STRING   Maximum log level
      --buffer-size=INT    Recent logs buffer (default: 100)
      --summary-interval=  Emit periodic summaries (e.g., '30s')
      --heartbeat=STRING   Emit periodic heartbeat messages (e.g., '10s')
      --tmux               Output to tmux session
      --session=STRING     Custom tmux session name
      --output=FILE        Write to explicit file path
      --session-dir=PATH   Directory for session files (default: ~/.xcw/sessions)
      --session-prefix=STR Prefix for session filename (default: app bundle ID)
```

**Simulator Selection Behavior:**

| `--simulator` | `--booted` | Behavior |
|---------------|------------|----------|
| not set | not set | Auto-detect: use single booted simulator, error if 0 or multiple |
| set | not set | Use specified simulator by name or UDID |
| not set | set | Use booted simulator, error if multiple |
| set | set | Error: flags are mutually exclusive |

### `xcw query`

Query historical logs from a simulator.

```
FLAGS:
  -s, --simulator=STRING   Simulator name or UDID
  -b, --booted             Use booted simulator (error if multiple)
  -a, --app=STRING         App bundle identifier (required)
      --since=STRING       How far back to query (default: 5m)
      --until=STRING       End time for query (RFC3339 or relative)
  -p, --pattern=STRING     Regex pattern to filter messages
  -x, --exclude=STRING     Regex pattern to exclude from messages
      --exclude-subsystem  Exclude logs from subsystem (repeatable)
      --min-level=STRING   Minimum log level (overrides global --level)
      --max-level=STRING   Maximum log level
      --limit=INT          Maximum logs to return (default: 1000)
      --predicate=STRING   Raw NSPredicate filter (advanced)
      --analyze            Include AI-friendly analysis summary
      --persist-patterns   Save detected patterns for future reference
      --pattern-file=PATH  Custom pattern file path
```

### `xcw clear`

Clear a tmux session's content.

```
FLAGS:
      --session=STRING     Tmux session name (required)
```

### `xcw watch`

Watch logs and trigger commands on specific patterns.

```
FLAGS:
  -s, --simulator=STRING   Simulator name or UDID
  -b, --booted             Use booted simulator (error if multiple)
  -a, --app=STRING         App bundle identifier (required)
      --min-level=STRING   Minimum log level (overrides global --level)
      --max-level=STRING   Maximum log level
      --on-error=STRING    Command to run when error-level log detected
      --on-fault=STRING    Command to run when fault-level log detected
      --on-pattern=STRING  Pattern:command pairs (repeatable)
      --cooldown=DURATION  Minimum time between triggers (default: 5s)
      --output=FILE        Write logs to explicit file path
      --session-dir=PATH   Directory for session files
      --session-prefix=STR Prefix for session filename
```

### `xcw summary`

Output a summary of recent buffered logs.

```
FLAGS:
  -s, --simulator=STRING   Simulator name or UDID
  -b, --booted             Use booted simulator (error if multiple)
  -a, --app=STRING         App bundle identifier (required)
```

### `xcw schema`

Output JSON Schema for xcw output types.

```
FLAGS:
      --type=STRING        Specific type(s) to output (comma-separated)
```

### `xcw config`

Show or manage configuration.

```
COMMANDS:
  show      Show current configuration
  path      Show config file path
  generate  Output sample config YAML to stdout
```

### `xcw doctor`

Check system requirements and configuration.

```
Checks:
  - Xcode and command line tools
  - xcrun simctl availability
  - tmux installation (optional)
  - Config file validity
  - Available simulators
```

### `xcw apps`

List installed apps on a simulator.

```
FLAGS:
  -s, --simulator=STRING   Simulator name or UDID
  -b, --booted             Use booted simulator
```

### `xcw pick`

Interactively select a simulator or app using fuzzy search.

```
USAGE:
  xcw pick <simulator|app> [flags]

FLAGS:
  -s, --simulator=STRING   Simulator for app picking (uses booted if omitted)
      --user-only          Show only user-installed apps (for app picking)
```

**Examples:**
```bash
# Pick a simulator interactively
xcw pick simulator

# Pick an app from booted simulator
xcw pick app

# Pick an app from specific simulator
xcw pick app -s "iPhone 15"

# Use in scripts (outputs UDID/bundle_id for piping)
UDID=$(xcw pick simulator -f text | cut -f1)
xcw tail -s "$UDID" -a com.example.app
```

**Note:** Requires an interactive terminal. For scripted use, use `xcw list` and `xcw apps` instead.

### `xcw analyze`

Analyze a recorded NDJSON log file for patterns and statistics.

```
FLAGS:
      --persist-patterns   Save detected patterns for future reference
      --pattern-file=PATH  Custom pattern file path
```

### `xcw replay`

Replay a recorded NDJSON log file.

```
FLAGS:
      --realtime           Replay with original timing
      --speed=FLOAT        Playback speed multiplier (default: 1.0)
      --follow             Follow file for new entries (like tail -f)
```

### `xcw completion`

Generate shell completions.

```
FLAGS:
      --shell=STRING       Shell type: bash, zsh, fish
```

### `xcw ui`

Interactive TUI log viewer with real-time filtering.

```
FLAGS:
  -s, --simulator=STRING   Simulator name or UDID
  -b, --booted             Use booted simulator (error if multiple)
  -a, --app=STRING         App bundle identifier (required)
      --buffer-size=INT    Number of logs to buffer (default: 1000)
```

### `xcw version`

Show version information.

```bash
xcw version           # Text output: xcw version v1.0.0 (abc1234)
xcw version -f ndjson # NDJSON output with version and commit
```

### `xcw update`

Show how to upgrade xcw.

```bash
xcw update            # Text instructions for Homebrew and Go install
xcw update -f ndjson  # NDJSON output with upgrade commands
```

**Example output:**
```
xcw update instructions

Current version: v1.0.0 (abc1234)

To upgrade via Homebrew:
  brew update && brew upgrade xcw

To upgrade via Go:
  go install github.com/vburojevic/xcw/cmd/xcw@latest

For release notes, see:
  https://github.com/vburojevic/xcw/releases
```

### `xcw sessions`

Manage session log files.

```
COMMANDS:
  list     List session files (default)
  show     Show path to a session file
  clean    Delete old session files

LIST FLAGS:
      --dir=PATH           Session directory (default: ~/.xcw/sessions)
      --limit=INT          Max sessions to show (default: 20)

SHOW FLAGS:
      --index=INT          Session index from list (1-based)
      --latest             Show most recent session

CLEAN FLAGS:
      --keep=INT           Number of sessions to keep (default: 10)
      --dry-run            Show what would be deleted without deleting
```

**Session File Naming:**

Files are named with timestamp and prefix: `YYYYMMDD-HHMMSS-prefix.ndjson`

Example: `20251209-153045-com.example.myapp.ndjson`

**Example AI Agent Workflow:**

```bash
# Start session (auto-generates timestamped file)
xcw tail -a com.example.myapp --session-dir ~/.xcw/sessions

# List recent sessions
xcw sessions list

# Get path to latest session (for piping)
xcw analyze $(xcw sessions show --latest)

# Clean old sessions, keep 5 most recent
xcw sessions clean --keep 5
```

## Global Flags

```
  -f, --format=STRING      Output format: ndjson, text (default: ndjson)
  -l, --level=STRING       Min log level: debug, info, default, error, fault
  -q, --quiet              Suppress non-log output
  -v, --verbose            Show debug output (predicates, reconnections)
```

## For AI Agents

This tool is designed for AI agents to monitor iOS app logs. Key features:

1. **Structured Output**: NDJSON format is easily parseable
2. **Schema Versioning**: All events include `schemaVersion` for compatibility
3. **Summary Markers**: Periodic summaries with error counts help identify issues
4. **Pattern Detection**: Analysis mode groups similar errors
5. **Pattern Persistence**: Track known vs new errors across sessions
6. **File Recording**: Capture logs to file for later analysis
7. **Replay Support**: Replay recorded sessions with timing
8. **Tmux Persistence**: Sessions persist for background monitoring
9. **Non-Interactive**: All commands work without user input
10. **Self-Diagnostics**: `xcw doctor` verifies the environment is set up correctly

### JSON Schema

Machine-readable JSON Schema files are available for validation:

```bash
# Generate schema dynamically
xcw schema > xcw-schema.json

# Generate specific types only
xcw schema --type log,error,summary

# Static schema files are at:
# schemas/v1/xcw-schema.json
```

Schema includes all output types: `log`, `summary`, `heartbeat`, `error`, `tmux`, `info`, `warning`, `trigger`, `doctor`, `app`, `pick`, `session`, `analysis`, `trigger_error`.

### NDJSON Output Types Reference

All outputs include `type` and `schemaVersion` fields. Current schema version is `1`.

| Type | Description | Key Fields |
|------|-------------|------------|
| `log` | Individual log entry from iOS Simulator | `timestamp`, `level`, `process`, `pid`, `message`, `subsystem`, `category` |
| `summary` | Periodic log statistics | `totalCount`, `errorCount`, `faultCount`, `errorRate`, `hasErrors`, `hasFaults` |
| `analysis` | Summary with pattern detection | `summary`, `patterns[]`, `new_pattern_count`, `known_pattern_count` |
| `heartbeat` | Stream keepalive message | `timestamp`, `uptime_seconds`, `logs_since_last` |
| `error` | Structured error from xcw | `code`, `message` |
| `info` | Informational message | `message`, `device`, `udid` |
| `warning` | Warning message | `message` |
| `tmux` | Tmux session info | `session`, `attach` |
| `trigger` | Trigger event fired | `trigger_type`, `command`, `match` |
| `trigger_error` | Trigger execution failed | `command`, `error` |
| `app` | Installed app info | `bundle_id`, `name`, `app_type`, `version` |
| `doctor` | System diagnostic report | `checks[]`, `all_passed`, `error_count`, `warn_count` |
| `pick` | Interactive selection result | `picked`, `name`, `udid` or `bundle_id` |
| `session` | Session file info | `path`, `name`, `timestamp`, `size`, `prefix` |

**Log Levels:** `Debug` < `Info` < `Default` < `Error` < `Fault`

### Error Codes Reference

All errors are output as NDJSON with `type: "error"`, a machine-readable `code`, and human-readable `message`.

| Code | When It Occurs |
|------|----------------|
| `DEVICE_NOT_FOUND` | Simulator not found by name/UDID |
| `NO_BOOTED_SIMULATOR` | No booted simulator when `--booted` used |
| `INVALID_FLAGS` | `--simulator` and `--booted` used together |
| `INVALID_PATTERN` | Regex pattern compilation fails |
| `INVALID_EXCLUDE_PATTERN` | Exclude regex compilation fails |
| `INVALID_DURATION` | Duration parsing fails (`--since`, `--window`) |
| `INVALID_UNTIL` | Until time parsing fails |
| `INVALID_INTERVAL` | Summary interval invalid |
| `INVALID_HEARTBEAT` | Heartbeat interval invalid |
| `INVALID_COOLDOWN` | Watch cooldown invalid |
| `INVALID_TRIGGER` | Trigger format invalid (missing colon) |
| `INVALID_TRIGGER_PATTERN` | Trigger regex compilation fails |
| `INVALID_TRIGGER_TIMEOUT` | Trigger timeout invalid |
| `STREAM_FAILED` | Log streaming failed |
| `QUERY_FAILED` | Historical log query failed |
| `LIST_FAILED` | Device listing failed |
| `LIST_APPS_FAILED` | App listing failed |
| `FILE_NOT_FOUND` | Input file not found |
| `FILE_CREATE_ERROR` | Failed to create output file |
| `READ_ERROR` | File read error |
| `NO_ENTRIES` | No valid log entries in file |
| `DEVICE_NOT_BOOTED` | Device must be booted for operation |
| `TMUX_NOT_INSTALLED` | tmux not installed |
| `TMUX_ERROR` | tmux operation failed |
| `SESSION_NOT_FOUND` | tmux session not found |
| `SESSION_DIR_ERROR` | Failed to create/access session directory |
| `SESSION_ERROR` | Session file operation failed |
| `LIST_SESSIONS_ERROR` | Failed to list session files |
| `INVALID_INDEX` | Session index out of range |
| `NO_SESSIONS` | No session files found |
| `CLEAN_ERROR` | Failed to clean old sessions |
| `CLEAR_FAILED` | Clear tmux pane failed |
| `NOT_INTERACTIVE` | Terminal not interactive (required for `pick`) |
| `NO_SIMULATORS` | No simulators available |
| `NO_APPS` | No apps found on simulator |

### Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| `0` | Success |
| `1` | Any error occurred (check NDJSON `error` output for `code`) |

### Example AI Workflow

```bash
# 1. Verify setup
xcw doctor

# 2. Start log monitoring with file output
xcw tail -s "iPhone 17 Pro" -a com.example.myapp --output test-run.ndjson

# 3. Run tests (logs stream to file)
xcodebuild test ...

# 4. Analyze the recorded logs
xcw analyze test-run.ndjson --persist-patterns

# 5. Query for new errors (patterns seen before are marked as known)
xcw query -a com.example.myapp --since 5m -l error --analyze --persist-patterns
```

### Tmux Workflow

```bash
# 1. Start log monitoring in tmux
xcw tail -s "iPhone 17 Pro" -a com.example.myapp --tmux

# 2. Run tests (logs stream to tmux)
xcodebuild test ...

# 3. Query for errors after tests
xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m -l error --analyze

# 4. Clear for next test run
xcw clear --session xcw-iphone-15-pro
```

## Requirements

- macOS 14+ (Sonoma) or later
- Xcode with iOS Simulator
- tmux (optional, for persistent sessions)

## License

MIT
