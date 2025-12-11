# xcw Roadmap

This is a lightweight view of where the project is headed. Dates are indicative.

## Near term

- Improve physical device support when Apple exposes stable CLI access.
- Expand filtering (`--where`, process matching) and presets for common workflows.
- Continue performance work on hot paths (parser, filters, TUI).

## Mid term

- Richer TUI: split panes, per‑session navigation, export helpers.
- Pluggable analysis/pattern engines for CI and agent workflows.
- Optional gRPC/stdio mode for agent integrations.

## Long term

- Cross‑platform log streaming where possible (macOS + VisionOS simulators).
- Library extraction so other tools can embed xcw’s streaming/analysis engine.

Have ideas? Create a Beads issue and tag it `feature`.

