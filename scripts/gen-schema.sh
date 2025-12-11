#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/schemas/generated.schema.json"

echo "Generating $OUT from xcw schema --json"
cd "$ROOT"
go run ./cmd/xcw schema --json > "$OUT"
echo "Generated schema; commit to keep schemas in sync."

