#!/usr/bin/env bash
set -euo pipefail

# Configuration
REMOTE_HOST="${REMOTE_HOST:-root@10.233.201.133}"
REMOTE_PATH="${REMOTE_PATH:-/usr/local/bin/launchpad}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$PROJECT_DIR/.." && pwd)"
EMBED_DIR="$PROJECT_DIR/internal/importfiles/files"
FILES_DIR="$REPO_ROOT/files"
OUT="$PROJECT_DIR/bin/launchpad-linux"

# Embed template files
echo "=> Embedding template files..."
mkdir -p "$EMBED_DIR"
cp -f "$FILES_DIR"/*.tar.gz "$FILES_DIR"/*.yaml "$EMBED_DIR/" 2>/dev/null || true

# Build
echo "=> Building linux/amd64..."
cd "$PROJECT_DIR"
GOOS=linux GOARCH=amd64 go build -o "$OUT" ./cmd/launchpad/

# Clean up embedded copies
rm -f "$EMBED_DIR"/*.tar.gz "$EMBED_DIR"/*.yaml

echo "=> Binary: $OUT ($(du -h "$OUT" | awk '{print $1}'))"

# Deploy
if [[ "${1:-}" == "--local" ]]; then
    echo "=> Local build only, skipping deploy."
else
    echo "=> Deploying to $REMOTE_HOST:$REMOTE_PATH..."
    scp -o ConnectTimeout=15 "$OUT" "$REMOTE_HOST:$REMOTE_PATH"
    echo "=> Done."
fi
