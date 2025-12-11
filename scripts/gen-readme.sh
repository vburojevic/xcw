#!/usr/bin/env bash
set -euo pipefail

# Regenerate machine-readable help to keep README/examples in sync.
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/docs/help.json"

echo "Generating $OUT from xcw help --json"
cd "$ROOT"
go run ./cmd/xcw help --json > "$OUT"
echo "Done. Review $OUT and sync snippets into README.md as needed."

