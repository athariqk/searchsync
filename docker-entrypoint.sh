#!/bin/sh
set -e

# Entrypoint for searchsync: check for config files, warn if missing, do not copy example files

APP=/app/searchsync
ENVF=/app/.env
REPLICA=/app/replica.yaml

echo "[entrypoint] checking configuration files"

if [ ! -f "$ENVF" ]; then
  echo "[entrypoint] WARNING: $ENVF not found in container."
fi
if [ ! -f "$REPLICA" ]; then
  echo "[entrypoint] WARNING: $REPLICA not found in container."
fi

# Check readability
for f in "$ENVF" "$REPLICA"; do
  if [ -f "$f" ] && [ ! -r "$f" ]; then
    echo "[entrypoint] WARNING: $f exists but is not readable."
  fi
done

echo "[entrypoint] starting searchsync"
exec "$APP" "$@"
