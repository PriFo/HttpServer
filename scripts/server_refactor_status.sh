#!/bin/bash
set -euo pipefail

TARGET=${1:-server/server.go}

if [[ ! -f $TARGET ]]; then
  echo "File not found: $TARGET" >&2
  exit 1
fi

LINES=$(wc -l < "$TARGET")
HANDLES=$(grep -c 'func (s \*Server) handle' "$TARGET")

printf "server.go size: %s lines\n" "$LINES"
printf "handle* functions: %s\n" "$HANDLES"

if (( LINES <= 4000 )); then
  echo "Stage 1 target reached (≤4000)."
else
  echo "Stage 1 pending (need ≤4000)."
fi

if (( LINES <= 2000 )); then
  echo "Stage 2 target reached (≤2000)."
else
  echo "Stage 2 pending (need ≤2000)."
fi

if (( LINES < 1000 )); then
  echo "Stage 3 target reached (<1000)."
else
  echo "Stage 3 pending (need <1000)."
fi

