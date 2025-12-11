# XcodeConsoleWatcher (xcw)

> **Tail & Analyze iOS Simulator logs with ease — built for AI agents.**

`xcw` is a small Go-based CLI that streams and inspects Xcode iOS Simulator console logs.  It was originally built to help AI agents such as Claude and Codex monitor app logs in real-time, but it's just as useful for human developers.  Every line of output is emitted as newline-delimited JSON (NDJSON), so agents can process events incrementally without waiting for the stream to finish.  Schema versioning and clearly defined event types make it easy to adapt your parser over time.

---

## ⚡ Start Here

**This is the command you want:**

```sh
xcw tail -s "iPhone 17 Pro" -a com.example.myapp
```

That's it. This streams logs from your app in real-time.

- `-s` — Simulator name (run `xcw list` to see available simulators)
- `-a` — Your app's bundle ID (run `xcw apps` to list installed apps)

> **For AI agents:** Always start with `xcw tail -s <simulator> -a <bundle_id>`. This is the primary command. Only use `query`, `watch`, or `analyze` for specific use cases after you've tried `tail`.

---

## Highlights

* **Structured NDJSON output** – each log event, summary or error is emitted as a JSON object, perfect for incremental consumption.
* **Real-time streaming** – tail logs from a booted simulator or a specific device with `xcw tail`.
* **Automatic session tracking** – detects app relaunches and emits `session_start`/`session_end` events so AI agents know when the app restarted.
* **Tail-scoped IDs** – every event carries a `tail_id`, so agents can correlate only the events from the current tail invocation.
* **Per-run file rotation** – when recording to disk, each app relaunch (or idle rollover) writes to a new file for clean ingestion.
* **Historical queries** – query past logs with `xcw query` using relative durations such as `--since 5m`.
* **Smart filtering** – filter by app bundle ID, log level, regex patterns, field values (`--where`), or exclude noise.
* **Log discovery** – use `xcw discover` to understand what subsystems, categories, and processes exist before filtering.
* **Deduplication** – collapse repeated identical messages with `--dedupe` to reduce noise.
* **AI-friendly summaries & pattern detection** – periodic summary markers and analysis mode group similar errors and track new vs known patterns.
* **Session-based recording & replay** – write logs to timestamped files for later analysis and replay them with original timing.
* **Persistent monitoring** – run `xcw tail` in a tmux session to keep logs streaming in the background across terminals.
* **Self-documenting** – ask `xcw help --json` or `xcw examples` to get machine-readable help and curated usage examples.

## Installation

### Homebrew (recommended)

```sh
brew tap vburojevic/tap
brew install xcw
```

### Go install

You can also install directly from source using the Go toolchain:

```sh
go install github.com/vburojevic/xcw/cmd/xcw@latest
```

### Building from source

Clone the repository and run the provided make targets:

```sh
git clone https://github.com/vburojevic/xcw.git
cd xcw
make install
```

## Quick start

Run `xcw` with no arguments to see a quick start guide:

```sh
xcw
```

These commands give you a feel for `xcw` without any configuration.  They work on macOS with at least one iOS Simulator installed.

### List available simulators

```sh
# all simulators
xcw list

# only booted simulators
xcw list --booted-only

# machine-readable output (NDJSON)
xcw list -f ndjson
```

### List installed apps

```sh
# apps on the booted simulator
xcw apps

# apps on a specific simulator
xcw apps -s "iPhone 17 Pro"

# NDJSON format
xcw apps -f ndjson
```

### Get machine-readable help & examples

```sh
# complete documentation as JSON (useful for agents)
xcw help --json

# usage examples for all commands
xcw examples

# examples for a specific command
xcw examples tail

# machine-readable examples
xcw examples --json
```

## Streaming logs in real-time

The `tail` subcommand streams logs from the iOS Simulator.  It automatically picks the single booted simulator when no simulator is specified.  You must always provide an app bundle identifier via `-a`/`--app`.

```sh
# tail logs from the booted simulator
xcw tail -a com.example.myapp

# tail logs from a named simulator
xcw tail -s "iPhone 17 Pro" -a com.example.myapp

# force a new session if no logs arrive for 60s
xcw tail -s "iPhone 17 Pro" -a com.example.myapp --session-idle 60s

# filter logs by regex (--filter or --pattern or -p)
xcw tail -a com.example.myapp --filter "error|warning"

# exclude noisy messages (can be repeated)
xcw tail -a com.example.myapp -x heartbeat -x keepalive -x routine

# limit log level range
xcw tail -a com.example.myapp --min-level info --max-level error

# stream in a tmux session so it keeps running in the background
xcw tail -a com.example.myapp --tmux

# write logs to a timestamped file in ~/.xcw/sessions
xcw tail -a com.example.myapp --session-dir ~/.xcw/sessions
```

## Advanced filtering

`xcw` provides powerful filtering options for finding exactly the logs you need:

```sh
# filter by field with --where (supports =, !=, ~, !~, >=, <=, ^, $)
xcw tail -a com.example.myapp --where level=error
xcw tail -a com.example.myapp --where "message~timeout"
xcw tail -a com.example.myapp --where "subsystem^com.example"

# combine multiple where clauses (AND logic)
xcw tail -a com.example.myapp --where level>=error --where "message~network"

# filter by process name
xcw tail -a com.example.myapp --process MyApp --process MyAppExtension

# collapse repeated identical messages
xcw tail -a com.example.myapp --dedupe
xcw tail -a com.example.myapp --dedupe --dedupe-window 5s
```

**Where operators:**

| Operator | Meaning | Example |
|----------|---------|---------|
| `=` | Equals | `level=error` |
| `!=` | Not equals | `level!=debug` |
| `~` | Contains (regex) | `message~timeout` |
| `!~` | Not contains | `message!~heartbeat` |
| `>=` | Greater or equal (for levels) | `level>=error` |
| `<=` | Less or equal (for levels) | `level<=info` |
| `^` | Starts with | `subsystem^com.example` |
| `$` | Ends with | `message$failed` |

**Supported fields:** `level`, `subsystem`, `category`, `process`, `message`, `pid`

## Discovering log sources

Use `xcw discover` to understand what subsystems, categories, and processes are generating logs:

```sh
# discover all logs from the last 5 minutes
xcw discover -s "iPhone 17 Pro" --since 5m

# discover logs for a specific app
xcw discover -s "iPhone 17 Pro" -a com.example.myapp --since 10m

# show more results
xcw discover -b --since 1h --top-n 30
```

This is especially useful for AI agents to understand the logging landscape before applying filters.

## Pre-launch log capture

To capture logs from the very first moment an app launches (including startup logs), use `--wait-for-launch`:

```sh
xcw tail -s "iPhone 17 Pro" -a com.example.myapp --wait-for-launch
```

This emits a `ready` event immediately when log capture is active:

```json
{"type":"ready","schemaVersion":1,"timestamp":"...","simulator":"iPhone 17 Pro","udid":"...","app":"com.example.myapp"}
```

**AI agent workflow:** Start `xcw tail --wait-for-launch`, wait for the `ready` event, then trigger your build/run process:

```sh
# Terminal 1: Start log capture (emits ready event immediately)
xcw tail -s "iPhone 17 Pro" -a com.example.myapp --wait-for-launch

# Terminal 2: After seeing ready event, build and run
xcodebuild -scheme MyApp build
xcrun simctl install booted MyApp.app
xcrun simctl launch booted com.example.myapp
```

## Querying historical logs

`xcw query` reads previously recorded logs (from `--output` files or sessions) and applies filters.  Use relative durations (`--since 5m`, `--since 1h`) and optionally request an analysis summary.

```sh
# query the last 5 minutes of logs for your app
xcw query -a com.example.myapp --since 5m

# query with analysis to group and count error patterns
xcw query -a com.example.myapp --since 10m --analyze

# restrict results to errors only
xcw query -a com.example.myapp --since 1h -l error

# persist detected patterns across sessions
xcw query -a com.example.myapp --since 1h --analyze --persist-patterns
```

## Capturing print() statements

`xcw tail` uses macOS unified logging, which captures `Logger`, `os_log`, and `NSLog` calls.  Swift `print()` statements go to stdout and are **not captured by unified logging**.

To capture print() output, use `xcw launch`:

```sh
# launch app and capture stdout/stderr
xcw launch -s "iPhone 17 Pro" -a com.example.myapp

# terminate any existing instance first
xcw launch -s "iPhone 17 Pro" -a com.example.myapp --terminate-existing
```

**Output format:**

```json
{"type":"console","schemaVersion":1,"timestamp":"2024-01-15T10:30:45Z","stream":"stdout","message":"Hello from print()","process":"com.example.myapp"}
```

**Recommendation:** For best results with `xcw`, use Swift's `Logger` API instead of `print()`:

```swift
import OSLog

let logger = Logger(subsystem: "com.example.myapp", category: "general")
logger.info("This message appears in xcw tail")
```

`Logger` provides log levels, subsystem filtering, and persistence — all accessible via `xcw tail` and `xcw query`.

## Recording, analyzing and replaying sessions

```sh
# record a session to an NDJSON file
xcw tail -a com.example.myapp --output session.ndjson

# analyze a recorded file
xcw analyze session.ndjson

# replay a recorded session with original timing
xcw replay session.ndjson --realtime

# replay at 2x speed
xcw replay session.ndjson --realtime --speed 2.0
```

## Background monitoring with tmux

Use the `--tmux` flag with `tail` to keep logs streaming while you do other work.  `xcw` will print a JSON object containing the session name.  Attach to the session at any time using the provided command.

```sh
# start streaming in a tmux session
xcw tail -a com.example.myapp --tmux

# the NDJSON output includes:
# {"type":"tmux","session":"xcw-iphone-17-pro","attach":"tmux attach -t xcw-iphone-17-pro"}

# attach later to view live logs
tmux attach -t xcw-iphone-17-pro

# clear the tmux pane between test runs
xcw clear --session xcw-iphone-17-pro
```

## Managing sessions

`xcw` names session files using the pattern `YYYYMMDD-HHMMSS-<prefix>.ndjson` (prefix defaults to the app bundle ID).  Sessions are stored in `~/.xcw/sessions` by default.  Use the `sessions` command to list, show or clean these files:

```sh
# list recent session files (sorted by date)
xcw sessions list

# show the path of the most recent session
xcw sessions show --latest

# delete old sessions, keeping only the latest 5
xcw sessions clean --keep 5
```

### For AI agents

**Primary command: `xcw tail`** – AI agents should use `tail` for real-time log streaming.  This is the main command for monitoring app behavior.

**Choosing an output mode:**

| Mode | Command | Best for |
|------|---------|----------|
| **Stdout** | `xcw tail -a APP` | Script agents (Codex) that process NDJSON line-by-line |
| **File** | `xcw tail -a APP --session-dir ~/.xcw/sessions` | Recording logs for later analysis with `xcw analyze` |
| **Tmux** | `xcw tail -a APP --tmux` | Humans watching logs visually in a terminal |

**Recommended AI agent workflow:**

```sh
# 1. Start recording logs to a session file
xcw tail -s "iPhone 17 Pro" -a com.example.myapp --session-dir ~/.xcw/sessions

# 2. After the test run, analyze the session
xcw analyze $(xcw sessions show --latest)
```

**When to use `xcw query`:**

`query` reads from macOS unified logging (system logs), not from your recorded sessions.  Use it only when you forgot to start `tail` and need to check what happened in the last few minutes:

```sh
# Check system logs from the last 5 minutes (not session files)
xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m -l error
```

**Note:** `--tmux` is designed for human visual monitoring.  AI agents should prefer `--session-dir` or `--output` for programmatic access to recorded logs.

## Automatic session tracking

`xcw` automatically detects when your iOS app is relaunched from Xcode.  When the app's PID changes, `xcw` emits session events so AI agents know they're looking at a fresh app instance without needing to restart tailing.

**How it works:**

1. On first log, `xcw` emits a `session_start` event with `session: 1`
2. All log entries include a `session` field matching the current session number
3. When the app relaunches (PID changes), `xcw` emits:
   - `session_end` with summary of the previous session (logs, errors, faults, duration)
   - `session_start` with `alert: "APP_RELAUNCHED"` and the new session number
4. When the stream stops, a final `session_end` is emitted so the last run is closed.
5. Optional: `--session-idle 60s` will force a `session_end`/`session_start` if no logs arrive for 60 seconds (useful to bracket manual test runs).

**Example session events:**

```json
{"type":"session_end","schemaVersion":1,"tail_id":"tail-abc","session":1,"pid":12345,"summary":{"total_logs":142,"errors":3,"faults":0,"duration_seconds":45}}
{"type":"session_start","schemaVersion":1,"tail_id":"tail-abc","alert":"APP_RELAUNCHED","session":2,"pid":67890,"previous_pid":12345,"app":"com.example.myapp","simulator":"iPhone 17 Pro","udid":"...","version":"1.4.0","build":"2201","binary_uuid":"C0FFEE...","timestamp":"2024-01-15T10:30:45Z"}
```

**Stderr alert (for AI agents scanning stderr):**

```
[XCW] 🚀 NEW SESSION: App relaunched (PID: 67890) - Previous: 142 logs, 3 errors
```

**In tmux mode**, a visual separator banner is written when a new session starts:

```
══════════════════════════════════════════════════════════════
  🚀 SESSION 2: com.example.myapp (PID: 67890)
  Previous: 142 logs, 3 errors | 2024-01-15 10:30:45
══════════════════════════════════════════════════════════════
```

This allows AI agents to keep `xcw tail` running continuously while you rebuild and relaunch your app from Xcode—no need to restart tailing.

**Recording to files:** When you use `--output` or `--session-dir`, `xcw` now rotates to a fresh file on every app relaunch or idle rollover (one file per run). Filenames include the session number when you provide `--output`, or a new timestamped file is created when using `--session-dir`.

**Agent contract (do this!):**
1. Track the latest `session_start` and only process logs whose `session` equals that value.
2. Also require `tail_id` to match the current tail invocation; drop events from other tails.
3. On `session_start`, `session_end`, or `clear_buffer`, reset any caches (dedupe/pattern memory) before continuing.
4. When recording to disk, read only the newest rotated file unless explicitly comparing runs.
5. Use older sessions/files only when you are asked to compare behavior across runs.

## Output format & JSON schema

By default `xcw` writes NDJSON to stdout.  Each event includes a `type` and `schemaVersion` field.  Types include `log`, `console`, `ready`, `summary`, `analysis`, `heartbeat`, `error`, `info`, `warning`, `tmux`, `trigger`, `app`, `doctor`, `pick`, `session`, `session_start` and `session_end`.  The current schema version is `1`.

Example log entry:

```json
{"type":"log","schemaVersion":1,"tail_id":"tail-abc","timestamp":"2024-01-15T10:30:45.123Z","level":"Error","process":"MyApp","pid":1234,"subsystem":"com.example.myapp","category":"network","message":"Connection failed","session":1}
```

Example summary marker:

```json
{"type":"summary","schemaVersion":1,"tail_id":"tail-abc","windowStart":"2024-01-15T10:25:00Z","windowEnd":"2024-01-15T10:30:00Z","totalCount":150,"errorCount":4,"faultCount":1,"hasErrors":true,"hasFaults":true,"errorRate":0.8}
```

You can generate a machine-readable JSON schema for validation:

```sh
# all types
xcw schema > xcw-schema.json

# specific types only
xcw schema --type log,error,summary

# the canonical schema file lives in this repo at schemas/v1/xcw-schema.json
```

## Global flags

The following flags apply to all commands:

| Flag | Purpose |
|---|---|
| `-f, --format <ndjson\|text>` | Output format (defaults to NDJSON) |
| `-l, --level <debug\|info\|default\|error\|fault>` | Minimum log level to emit |
| `-q, --quiet` | Suppress non-log output |
| `-v, --verbose` | Show debug information (predicate evaluation, reconnection notices) |

## Designed for AI agents

`xcw` was built with AI consumption in mind.

**Start here:** Use `xcw tail` to stream logs.  Record to a file with `--session-dir` for later analysis.

Key properties:

1. **Primary command is `tail`** – stream logs in real-time; use `--session-dir` to record for analysis.
2. **Structured NDJSON output** – easy to parse incrementally, one JSON object per line.
3. **Schema versioning** – every record contains a `schemaVersion` so agents can handle future changes.
4. **Automatic session tracking** – detects app relaunches via PID changes and emits `session_start`/`session_end` events with summaries. No need to restart tailing when rebuilding from Xcode.
5. **Session recording** – capture logs to timestamped files, analyze later with `xcw analyze`.
6. **Pattern detection** – analysis mode groups similar errors and tracks new vs known patterns.
7. **Non-interactive** – all commands accept flags; no interactive prompts required.
8. **Self-documenting** – run `xcw help --json` for complete machine-readable documentation.
9. **Self-diagnostics** – `xcw doctor` checks your environment and prints a diagnostics report.

## Requirements

* macOS 14 (Sonoma) or later
* Xcode with the iOS Simulator installed
* `tmux` (optional, required only if you use `--tmux` sessions)

## License

`xcw` is licensed under the MIT License.  See the [LICENSE](https://github.com/vburojevic/xcw/blob/main/LICENSE) file for details.
