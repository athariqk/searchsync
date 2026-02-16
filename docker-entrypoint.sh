#!/bin/sh
set -e

# Entrypoint for searchsync:
# - If /app/config.yaml exists, run the binary
# - Otherwise copy config.example.yaml -> config.yaml (non-secret defaults)
#   and warn the user that real config/secrets must be provided at runtime

APP=/app/searchsync
CFG=/app/config.yaml
EXAMPLE=/app/config.example.yaml

echo "[entrypoint] checking configuration"

if [ -f "$CFG" ]; then
  echo "[entrypoint] found $CFG"
else
  if [ -f "$EXAMPLE" ]; then
    echo "[entrypoint] $CFG not found — copying $EXAMPLE -> $CFG"
    cp "$EXAMPLE" "$CFG"
    echo "[entrypoint] copied example config; ensure you mount a real config.yaml or set secrets via envs"
  else
    echo "[entrypoint] no config found (neither $CFG nor $EXAMPLE)."
    echo "[entrypoint] Please mount /app/config.yaml or provide configuration via environment variables."
  fi
fi

echo "[entrypoint] starting searchsync"
exec "$APP" "$@"
