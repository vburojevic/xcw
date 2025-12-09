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

---

### End-to-End AI Agent Examples

#### Example 1: Crash Detection → Persist → GitHub Issue

Complete workflow that monitors logs, detects new crash patterns, and files GitHub issues:

```bash
#!/bin/bash
# ai-crash-monitor.sh - Detect crashes, persist patterns, file GitHub issue

set -e

APP_BUNDLE="com.example.myapp"
PATTERN_FILE="crash-patterns.json"
LOG_FILE="crashes.ndjson"

# 1. Monitor for crashes with pattern persistence (background)
xcw tail --booted \
  --app "$APP_BUNDLE" \
  --level Error \
  --persist-patterns \
  --pattern-file "$PATTERN_FILE" \
  --output "$LOG_FILE" &

TAIL_PID=$!
trap "kill $TAIL_PID 2>/dev/null" EXIT

echo "Monitoring started (PID: $TAIL_PID)"

# 2. Periodically check for new patterns
while true; do
  sleep 60

  # Analyze logs for new patterns
  ANALYSIS=$(xcw analyze "$LOG_FILE" \
    --persist-patterns \
    --pattern-file "$PATTERN_FILE" \
    -f ndjson 2>/dev/null | tail -1)

  NEW_COUNT=$(echo "$ANALYSIS" | jq -r '.new_pattern_count // 0')

  if [ "$NEW_COUNT" -gt 0 ]; then
    # Extract new patterns
    NEW_PATTERNS=$(echo "$ANALYSIS" | jq -r '
      .patterns[]
      | select(.is_new == true)
      | "- **\(.pattern)**: \(.count) occurrences\n  Sample: `\(.samples[0])`"
    ')

    # File GitHub issue
    gh issue create \
      --title "New crash pattern detected in $APP_BUNDLE" \
      --body "## New Crash Patterns Detected

$NEW_PATTERNS

**Detected at:** $(date -u +%Y-%m-%dT%H:%M:%SZ)
**Pattern file:** \`$PATTERN_FILE\`
**Log file:** \`$LOG_FILE\`

---
*Automated by xcw crash monitor*"

    echo "Filed GitHub issue for $NEW_COUNT new pattern(s)"
  fi
done
```

#### Example 2: Python Integration for Claude/Codex Agents

```python
#!/usr/bin/env python3
"""
xcw_agent.py - AI Agent integration with xcw

Parse NDJSON output and react to different event types.
Designed for use with Claude, Codex, or other AI agents.
"""

import json
import subprocess
import sys
from dataclasses import dataclass
from typing import Iterator, Optional

@dataclass
class LogEntry:
    timestamp: str
    level: str
    process: str
    pid: int
    message: str
    subsystem: Optional[str] = None
    category: Optional[str] = None

@dataclass
class Summary:
    total_count: int
    error_count: int
    fault_count: int
    error_rate: float
    has_errors: bool
    has_faults: bool

def stream_logs(
    app: str,
    level: str = "Error"
) -> Iterator[dict]:
    """Stream parsed NDJSON entries from xcw tail."""
    cmd = [
        "xcw", "tail", "--booted",
        "-a", app,
        "--level", level,
        "-f", "ndjson"
    ]

    proc = subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )

    for line in proc.stdout:
        line = line.strip()
        if not line:
            continue
        try:
            yield json.loads(line)
        except json.JSONDecodeError:
            continue

def handle_entry(entry: dict) -> None:
    """Handle different NDJSON types appropriately."""
    entry_type = entry.get("type")

    match entry_type:
        case "log":
            level = entry.get("level", "")
            if level in ("Error", "Fault"):
                print(f"[{level.upper()}] {entry['process']}: {entry['message']}")
                # AI agent action: analyze error, suggest fix, etc.

        case "summary":
            if entry.get("hasErrors") or entry.get("hasFaults"):
                print(f"[SUMMARY] Errors: {entry['errorCount']}, "
                      f"Faults: {entry['faultCount']}, "
                      f"Rate: {entry['errorRate']:.2f}/min")

        case "analysis":
            for pattern in entry.get("patterns", []):
                if pattern.get("is_new"):
                    print(f"[NEW PATTERN] {pattern['pattern']} "
                          f"({pattern['count']}x)")
                    # AI agent action: investigate new error pattern

        case "heartbeat":
            # Connection is alive, logs are flowing
            pass

        case "error":
            code = entry.get("code", "UNKNOWN")
            msg = entry.get("message", "")
            print(f"[XCW ERROR] {code}: {msg}", file=sys.stderr)
            # Handle tool errors (retry, fallback, etc.)

def query_recent_errors(app: str, since: str = "5m") -> list[dict]:
    """Query recent error logs and return as list."""
    result = subprocess.run(
        ["xcw", "query", "--booted", "-a", app,
         "--since", since, "--level", "Error",
         "--analyze", "-f", "ndjson"],
        capture_output=True, text=True
    )

    entries = []
    for line in result.stdout.strip().split("\n"):
        if line:
            try:
                entries.append(json.loads(line))
            except json.JSONDecodeError:
                continue
    return entries

def check_health(app: str, since: str = "5m") -> bool:
    """Check if app has errors in recent logs. Returns True if healthy."""
    entries = query_recent_errors(app, since)
    for entry in entries:
        if entry.get("type") == "summary":
            return not (entry.get("hasErrors") or entry.get("hasFaults"))
    return True  # No summary = no errors

if __name__ == "__main__":
    app = sys.argv[1] if len(sys.argv) > 1 else "com.example.myapp"
    print(f"Monitoring {app} for errors...")

    for entry in stream_logs(app, level="Error"):
        handle_entry(entry)
```

#### Example 3: CI/CD Health Check

Quick one-liner for CI pipelines to fail on errors:

```bash
# Fail CI if errors detected in last 5 minutes
xcw query --booted -a com.example.myapp --since 5m -f ndjson | \
  jq -e 'select(.type=="summary") | .hasErrors == false and .hasFaults == false' \
  > /dev/null || (echo "FAIL: Errors detected" && exit 1)
```

Extended CI script with pattern tracking:

```bash
#!/bin/bash
# ci-log-check.sh - Check logs after test run

APP="com.example.myapp"
SINCE="10m"

# Query with analysis
RESULT=$(xcw query --booted -a "$APP" --since "$SINCE" --analyze -f ndjson)

# Extract summary
SUMMARY=$(echo "$RESULT" | jq -s 'map(select(.type=="analysis")) | .[0]')

if [ -z "$SUMMARY" ] || [ "$SUMMARY" = "null" ]; then
  echo "No analysis data available"
  exit 0
fi

ERROR_COUNT=$(echo "$SUMMARY" | jq '.summary.errorCount // 0')
FAULT_COUNT=$(echo "$SUMMARY" | jq '.summary.faultCount // 0')
NEW_PATTERNS=$(echo "$SUMMARY" | jq '.new_pattern_count // 0')

echo "=== Log Analysis ==="
echo "Errors: $ERROR_COUNT"
echo "Faults: $FAULT_COUNT"
echo "New patterns: $NEW_PATTERNS"

# Fail on faults (critical)
if [ "$FAULT_COUNT" -gt 0 ]; then
  echo "FATAL: Fault-level logs detected!"
  echo "$RESULT" | jq -r 'select(.type=="log" and .level=="Fault") | .message'
  exit 1
fi

# Warn on new error patterns
if [ "$NEW_PATTERNS" -gt 0 ]; then
  echo "WARNING: New error patterns detected:"
  echo "$SUMMARY" | jq -r '.patterns[] | select(.is_new) | "  - \(.pattern)"'
fi

exit 0
```

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
