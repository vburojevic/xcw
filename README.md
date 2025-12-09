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
- **File Output with Rotation**: Log to file with automatic rotation
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

# With periodic AI summaries every 30 seconds
xcw tail -a com.example.myapp --summary-interval 30s

# With heartbeat messages for connection health
xcw tail -a com.example.myapp --heartbeat 10s

# Output to tmux session
xcw tail -a com.example.myapp --tmux

# Output to file with rotation
xcw tail -a com.example.myapp --output logs.ndjson --rotate-size 10MB --rotate-count 5

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

### NDJSON Log Entry

```json
{"timestamp":"2024-01-15T10:30:45.123Z","level":"Error","process":"MyApp","pid":1234,"subsystem":"com.example.myapp","category":"network","message":"Connection failed"}
```

### Summary Marker

```json
{"type":"summary","windowStart":"2024-01-15T10:25:00Z","windowEnd":"2024-01-15T10:30:00Z","totalCount":150,"errorCount":4,"faultCount":1,"hasErrors":true,"hasFaults":true,"errorRate":0.8}
```

### Error Output

```json
{"type":"error","code":"NO_BOOTED_SIMULATOR","message":"No booted simulator found"}
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
      --output=FILE        Write to file instead of stdout
      --rotate-size=SIZE   Max file size before rotation (e.g., '10MB')
      --rotate-count=INT   Number of rotated files to keep
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
2. **Summary Markers**: Periodic summaries with error counts help identify issues
3. **Pattern Detection**: Analysis mode groups similar errors
4. **Pattern Persistence**: Track known vs new errors across sessions
5. **File Recording**: Capture logs to file for later analysis
6. **Replay Support**: Replay recorded sessions with timing
7. **Tmux Persistence**: Sessions persist for background monitoring
8. **Non-Interactive**: All commands work without user input
9. **Self-Diagnostics**: `xcw doctor` verifies the environment is set up correctly

### Example AI Workflow

```bash
# 1. Verify setup
xcw doctor

# 2. Start log monitoring with file output
xcw tail -a com.example.myapp --output test-run.ndjson --summary-interval 30s

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
xcw tail -a com.example.myapp --tmux --summary-interval 30s

# 2. Run tests (logs stream to tmux)
xcodebuild test ...

# 3. Query for errors after tests
xcw query -a com.example.myapp --since 5m -l error --analyze

# 4. Clear for next test run
xcw clear --session xcw-iphone-15-pro
```

## Requirements

- macOS 14+ (Sonoma) or later
- Xcode with iOS Simulator
- tmux (optional, for persistent sessions)

## License

MIT
