#!/bin/sh
set -e

# Entrypoint for searchsync: check for config files, warn if missing, do not copy example files

APP=/app/searchsync
REPLICA=/app/replica.yaml

echo "[entrypoint] checking configuration files"

if [ ! -f "$REPLICA" ]; then
  echo "[entrypoint] WARNING: $REPLICA not found in container."
fi

# Check readability
if [ -f "$REPLICA" ] && [ ! -r "$REPLICA" ]; then
  echo "[entrypoint] WARNING: "$REPLICA" exists but is not readable."
fi

echo "[entrypoint] starting searchsync"
exec "$APP" "$@"
