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

## Installation

### Homebrew (Recommended)

```bash
brew tap vedranburojevic/tap
brew install xcw
```

### Go Install

```bash
go install github.com/vedranburojevic/xcw/cmd/xcw@latest
```

### From Source

```bash
git clone https://github.com/vedranburojevic/xcw.git
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
# Tail logs from booted simulator for your app
xcw tail -a com.example.myapp

# With regex pattern filtering
xcw tail -a com.example.myapp -p "error|warning"

# Specific simulator by name
xcw tail -s "iPhone 15 Pro" -a com.example.myapp

# With periodic AI summaries every 30 seconds
xcw tail -a com.example.myapp --summary-interval 30s

# Output to tmux session
xcw tail -a com.example.myapp --tmux
```

### Query Historical Logs

```bash
# Query last 5 minutes
xcw query -a com.example.myapp --since 5m

# Query with analysis
xcw query -a com.example.myapp --since 10m --analyze

# Filter errors only
xcw query -a com.example.myapp --since 1h -l error
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
  -s, --simulator=STRING   Simulator name, UDID, or 'booted' (default: booted)
  -a, --app=STRING         App bundle identifier (required)
  -p, --pattern=STRING     Regex pattern to filter messages
      --subsystem=STRING   Filter by subsystem (repeatable)
      --category=STRING    Filter by category (repeatable)
      --buffer-size=INT    Recent logs buffer (default: 100)
      --summary-interval=  Emit periodic summaries (e.g., '30s')
      --tmux               Output to tmux session
      --session=STRING     Custom tmux session name
```

### `xcw query`

Query historical logs from a simulator.

```
FLAGS:
  -s, --simulator=STRING   Simulator name, UDID, or 'booted' (default: booted)
  -a, --app=STRING         App bundle identifier (required)
      --since=STRING       How far back to query (default: 5m)
      --until=STRING       End time for query
  -p, --pattern=STRING     Regex pattern to filter messages
      --limit=INT          Maximum logs to return (default: 1000)
      --analyze            Include AI-friendly analysis summary
```

### `xcw clear`

Clear a tmux session's content.

```
FLAGS:
      --session=STRING     Tmux session name (required)
```

## Global Flags

```
  -f, --format=STRING      Output format: ndjson, text (default: ndjson)
  -l, --level=STRING       Min log level: debug, info, default, error, fault
  -q, --quiet              Suppress non-log output
```

## For AI Agents

This tool is designed for AI agents to monitor iOS app logs. Key features:

1. **Structured Output**: NDJSON format is easily parseable
2. **Summary Markers**: Periodic summaries with error counts help identify issues
3. **Pattern Detection**: Analysis mode groups similar errors
4. **Tmux Persistence**: Sessions persist for background monitoring
5. **Non-Interactive**: All commands work without user input

### Example AI Workflow

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

- macOS 26+ (Tahoe)
- Xcode with iOS Simulator
- tmux (optional, for persistent sessions)

## License

MIT
