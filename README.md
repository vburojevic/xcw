# XcodeConsoleWatcher (xcw)

> **Tail & Analyze iOS Simulator logs with ease — built for AI agents.**

`xcw` is a small Go-based CLI that streams and inspects Xcode iOS Simulator console logs.  It was originally built to help AI agents such as Claude and Codex monitor app logs in real-time, but it's just as useful for human developers.  Every line of output is emitted as newline-delimited JSON (NDJSON), so agents can process events incrementally without waiting for the stream to finish.  Schema versioning and clearly defined event types make it easy to adapt your parser over time.

## Highlights

* **Structured NDJSON output** – each log event, summary or error is emitted as a JSON object, perfect for incremental consumption.
* **Real-time streaming** – tail logs from a booted simulator or a specific device with `xcw tail`.
* **Historical queries** – query past logs with `xcw query` using relative durations such as `--since 5m`.
* **Smart filtering** – filter by app bundle ID, log level, regex patterns, or exclude noise.
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
xcw tail -s "iPhone 15 Pro" -a com.example.myapp

# filter logs by regex
xcw tail -a com.example.myapp -p "error|warning"

# exclude noisy messages
xcw tail -a com.example.myapp -x "heartbeat|keepalive"

# limit log level range
xcw tail -a com.example.myapp --min-level info --max-level error

# stream in a tmux session so it keeps running in the background
xcw tail -a com.example.myapp --tmux

# write logs to a timestamped file in ~/.xcw/sessions
xcw tail -a com.example.myapp --session-dir ~/.xcw/sessions
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

### Example AI agent workflow

Here's a typical workflow for a script-based agent (e.g. Codex):

1. **Stream logs directly to stdout** so the agent can process each NDJSON line as soon as it arrives:

   ```sh
   xcw tail -s "iPhone 17 Pro" -a com.example.myapp
   ```

2. **Optionally write to file** using `--output` if you want to replay or share the session later:

   ```sh
   xcw tail -s "iPhone 17 Pro" -a com.example.myapp --output session.ndjson
   ```

3. **Analyze or replay** recorded sessions on demand:

   ```sh
   xcw analyze session.ndjson
   xcw replay session.ndjson --realtime
   ```

4. **Use tmux for persistent monitoring**: start a background session and query logs periodically while continuing other work.

   ```sh
   xcw tail -s "iPhone 17 Pro" -a com.example.myapp --tmux
   # later, query errors from the last 5 minutes only
   xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m -l error
   ```

## Output format & JSON schema

By default `xcw` writes NDJSON to stdout.  Each event includes a `type` and `schemaVersion` field.  Types include `log`, `summary`, `analysis`, `heartbeat`, `error`, `info`, `warning`, `tmux`, `trigger`, `app`, `doctor`, `pick` and `session`.  The current schema version is `1`.

Example log entry:

```json
{"type":"log","schemaVersion":1,"timestamp":"2024-01-15T10:30:45.123Z","level":"Error","process":"MyApp","pid":1234,"subsystem":"com.example.myapp","category":"network","message":"Connection failed"}
```

Example summary marker:

```json
{"type":"summary","schemaVersion":1,"windowStart":"2024-01-15T10:25:00Z","windowEnd":"2024-01-15T10:30:00Z","totalCount":150,"errorCount":4,"faultCount":1,"hasErrors":true,"hasFaults":true,"errorRate":0.8}
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

`xcw` was built with AI consumption in mind.  Key properties:

1. **Structured output** – NDJSON is easy to parse incrementally.
2. **Schema versioning** – every record contains a `schemaVersion` so agents can handle future changes.
3. **Summary markers** – periodic summaries include error counts and rates.
4. **Pattern detection & persistence** – analysis mode groups similar errors and optionally persists known patterns.
5. **File recording & replay support** – capture logs and replay them with original timing.
6. **Tmux persistence** – background sessions persist across terminal tabs.
7. **Non-interactive** – all commands accept flags; no interactive prompts are required for normal operation.
8. **Self-diagnostics** – `xcw doctor` checks your environment and prints a diagnostics report.

For a complete list of output types and fields, see the **NDJSON output types reference** in the official README or inspect the JSON schema.

## Requirements

* macOS 14 (Sonoma) or later
* Xcode with the iOS Simulator installed
* `tmux` (optional, required only if you use `--tmux` sessions)

## License

`xcw` is licensed under the MIT License.  See the [LICENSE](https://github.com/vburojevic/xcw/blob/main/LICENSE) file for details.
