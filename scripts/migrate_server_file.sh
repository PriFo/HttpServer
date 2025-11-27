#!/bin/bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: migrate_server_file.sh [--dry-run] <source_file> <target_dir>

Copies a legacy server handler to the new legacy tree, renames it, and patches package/import paths.
EOF
}

if [[ $# -lt 2 ]]; then
  usage
  exit 1
fi

DRY_RUN=0
if [[ $1 == "--dry-run" || $1 == "-n" ]]; then
  DRY_RUN=1
  shift
fi

if [[ $# -lt 2 ]]; then
  usage
  exit 1
fi

SOURCE_FILE=$1
TARGET_DIR=$2

if [[ ! -f $SOURCE_FILE ]]; then
  echo "Source file not found: $SOURCE_FILE" >&2
  exit 1
fi

mkdir -p "$TARGET_DIR"

BASENAME=$(basename "$SOURCE_FILE" .go)
LEGACY_NAME="${BASENAME#server_}_legacy.go"
TARGET_FILE="$TARGET_DIR/$LEGACY_NAME"

if [[ $DRY_RUN -eq 1 ]]; then
  echo "[DRY-RUN] Would migrate $SOURCE_FILE -> $TARGET_FILE"
  exit 0
fi

cp "$SOURCE_FILE" "$TARGET_FILE"

if grep -q '^package ' "$TARGET_FILE"; then
  sed -i '1s/^package .*/package legacy/' "$TARGET_FILE"
else
  sed -i '1ipackage legacy\n' "$TARGET_FILE"
fi

sed -i 's|"../internal/|"../../internal/|g' "$TARGET_FILE"
sed -i 's|"../server/|"../../server/|g' "$TARGET_FILE"

grep -q 'TODO:legacy-migration' "$TARGET_FILE" || sed -i '2i// TODO:legacy-migration revisit dependencies after handler extraction' "$TARGET_FILE"

echo "Миграция $SOURCE_FILE завершена: $TARGET_FILE"

