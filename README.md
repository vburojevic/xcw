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
* **Agent-ready output** – `agent_hints`, `metadata`, `reconnect_notice`, `clear_buffer`, `cutoff_reached`, `heartbeat` with `last_seen_timestamp`, and a machine-friendly preset.
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

The `tail` subcommand streams logs from the iOS Simulator. It automatically picks the single booted simulator when no simulator is specified.

By default, provide an app bundle identifier via `-a`/`--app`. For advanced use cases you can omit `--app` when supplying a raw `--predicate`, or use `--all` to explicitly stream unfiltered simulator logs (can be very noisy).

```sh
# tail logs from the booted simulator
xcw tail -a com.example.myapp

# tail logs from a named simulator
xcw tail -s "iPhone 17 Pro" -a com.example.myapp

# force a new session if no logs arrive for 60s
xcw tail -s "iPhone 17 Pro" -a com.example.myapp --session-idle 60s

# machine-friendly preset for agents
xcw --machine-friendly tail -s "iPhone 17 Pro" -a com.example.myapp

# dry-run to see the resolved options as JSON (no streaming)
xcw tail -s "iPhone 17 Pro" -a com.example.myapp --dry-run-json

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

# boolean where expressions (OR/AND/NOT + parentheses)
xcw tail -a com.example.myapp --where '(level=error OR level=fault) AND message~timeout'

# regex literal with flags (case-insensitive)
xcw tail -a com.example.myapp --where 'message~/timeout|crash/i'

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

**Supported fields:** `level`, `subsystem`, `category`, `process`, `message`, `pid`, `tid`

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

## Backfilling gaps with --resume (NDJSON)

If you see `reconnect_notice`, there may be log gaps. For NDJSON tails, you can enable best-effort backfill for small gaps:

```sh
xcw tail -s "iPhone 17 Pro" -a com.example.myapp --resume --resume-max-gap 2m --resume-limit 2000
```

- Requires `-f ndjson` (or `--machine-friendly`) and `--app`.
- Persists resume state to `~/.xcw/resume/<bundle_id>.json` (override with `--resume-state`).
- Emits `gap_detected` and, when backfilled, `gap_filled` (window is `(from_timestamp, to_timestamp]`).

## Querying historical logs

`xcw query` queries historical logs from the iOS Simulator via macOS unified logging (best when you forgot to start `tail`). It does **not** read your recorded `--output`/session files — use `xcw analyze` / `xcw replay` for those.

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

## Watching logs and running triggers

`xcw watch` streams logs like `tail`, and can run commands when it sees errors/faults or message patterns.

```sh
# run a command when an error-level log appears (capture command output into trigger_result)
xcw watch -s "iPhone 17 Pro" -a com.example.myapp --where level>=error --on-error "./notify.sh" --trigger-output capture

# run a command when a regex matches the message (pattern:command; can be repeated)
xcw watch -s "iPhone 17 Pro" -a com.example.myapp --on-pattern 'crash|fatal:./notify.sh' --cooldown 10s
```

In NDJSON mode, trigger executions are correlated by `trigger_id` (and scoped by `tail_id`/`session`):

- `trigger`: emitted when a trigger starts
- `trigger_result`: emitted when it completes (`exit_code`, `duration_ms`, `timed_out`, optional `output`/`error`)
- `trigger_error`: emitted only on failures (same `trigger_id`)

Trigger output modes:

- `discard` (default): do not capture stdout/stderr
- `capture`: capture combined stdout/stderr into `trigger_result.output` (truncated)
- `inherit`: stream trigger output to stdout/stderr (can break NDJSON if stdout is used)

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

## Configuration & precedence

`xcw` reads settings in this order (highest wins): **CLI flags → environment variables → config file → built-in defaults**. This keeps AI agents predictable when they reuse the same tail session across relaunches.

- **Environment**: prefix every key with `XCW_`. Common shortcuts: `XCW_FORMAT`, `XCW_LEVEL`, `XCW_QUIET`, `XCW_VERBOSE`, `XCW_APP`, `XCW_SIMULATOR`. Nested keys work too: `XCW_TAIL_HEARTBEAT=2s`, `XCW_QUERY_LIMIT=200`, `XCW_WATCH_COOLDOWN=1s`.
- **Config file locations** (first found is used): `./.xcw.yaml`/`./.xcw.yml`/`./xcw.yaml`/`./xcw.yml`, then `~/.xcw.yaml`/`~/.xcw.yml`, then `~/.config/xcw/config.yaml` (or `$XDG_CONFIG_HOME/xcw/config.yaml`), then `/etc/xcw/config.yaml`.
- **Per-command defaults**: set sticky values without repeating flags:

```yaml
format: ndjson
level: debug
quiet: false
verbose: false

defaults:
  simulator: "iPhone 17 Pro"
  app: com.example.myapp
  buffer_size: 200
  since: 5m
  limit: 2000

tail:
  heartbeat: 5s
  summary_interval: 20s
  session_idle: 60s

query:
  since: 15m
  limit: 1500

watch:
  cooldown: 2s
```

> Tip for agents: set `XCW_SIMULATOR="iPhone 17 Pro"` and `XCW_APP=<bundle>` once, then rely on config defaults so a relaunch is treated as the same tail session while still emitting `session_start`/`session_end` markers for each new app PID.

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
6. Watch for `reconnect_notice` (and `gap_detected`/`gap_filled` when `--resume` is enabled) to mark possible log gaps; watch `cutoff_reached` to know the stream ended intentionally.
7. Use `metadata` at startup for version/commit info; `heartbeat.last_seen_timestamp` to detect stalls.

## Output format & JSON schema

By default `xcw` writes NDJSON to stdout.  Each event includes a `type` and `schemaVersion` field.  Common types include `log`, `metadata`, `ready`, `heartbeat`, `stats`, `summary`, `analysis`, `session_start`, `session_end`, `clear_buffer`, `reconnect_notice`, `gap_detected`, `gap_filled`, `cutoff_reached`, `trigger`, `trigger_result`, `trigger_error`, `console`, `simulator`, `app`, `doctor`, `pick`, and `session`.  The current schema version is `1`.

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

# the canonical schema file lives in this repo at schemas/generated.schema.json
```

## Troubleshooting

### `--booted` errors / multiple booted simulators

If you see errors like “multiple booted simulators”, either:

- Specify a simulator explicitly: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp`
- Or pick one interactively: `xcw pick simulator`
- Or shut down the extra simulators: `xcrun simctl shutdown <udid>`

### `xcrun` / Xcode toolchain issues

`xcw` relies on `xcrun simctl` and `log stream`. If `xcrun` fails:

- Ensure Xcode and Command Line Tools are installed (`xcode-select -p`).
- Open Xcode once after updating to accept the license.
- Try `xcw doctor` for a quick environment check and actionable hints.

### No logs / wrong filters

- Verify the bundle ID: `xcw apps` (then use `-a <bundle_id>`).
- Use `xcw discover --since 5m` to learn valid subsystems/categories/processes before filtering.
- Remember: `print()` doesn’t show up in `xcw tail` (use `xcw launch` to capture stdout/stderr).

### Quoting predicates / regex

Shell quoting bites. Prefer quoting complex predicates/expressions:

- `xcw tail --where '(level=error OR level=fault) AND message~timeout'`
- `xcw tail --where 'message~/timeout|crash/i'`

### Stream reconnects and gaps

If you see `reconnect_notice`, there may be log gaps. Run with `-v/--verbose` to surface diagnostics (including `xcrun` stderr) and watch `heartbeat.last_seen_timestamp` to detect stalls.

For NDJSON tails with `--resume` (requires `--app`), `xcw` will emit `gap_detected` and may emit `gap_filled` after backfilling via `query` (bounded by `--resume-max-gap` and `--resume-limit`).

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
* Physical iOS devices are not supported yet; Apple doesn't provide a stable CLI for unified logs. Use Console.app or `idevicesyslog` as a workaround.
* `tmux` (optional, required only if you use `--tmux` sessions)

## License

`xcw` is licensed under the MIT License.  See the [LICENSE](https://github.com/vburojevic/xcw/blob/main/LICENSE) file for details.
