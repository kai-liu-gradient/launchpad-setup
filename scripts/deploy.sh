#!/usr/bin/env bash
set -euo pipefail

# Configuration
REMOTE_HOST="${REMOTE_HOST:-root@10.233.201.133}"
REMOTE_PATH="${REMOTE_PATH:-/usr/local/bin/launchpad}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
OUT="$PROJECT_DIR/bin/launchpad-linux"

# Build
echo "=> Building linux/amd64..."
cd "$PROJECT_DIR"
GOOS=linux GOARCH=amd64 go build -o "$OUT" ./cmd/launchpad/

echo "=> Binary: $OUT ($(du -h "$OUT" | awk '{print $1}'))"

# Deploy
if [[ "${1:-}" == "--local" ]]; then
    echo "=> Local build only, skipping deploy."
else
    echo "=> Deploying to $REMOTE_HOST:$REMOTE_PATH..."
    scp -o ConnectTimeout=15 "$OUT" "$REMOTE_HOST:$REMOTE_PATH"
    echo "=> Done."
fi
