#!/usr/bin/env bash
# Builds and runs the whole VN Stock Sim V1 demo in Docker (backend +
# frontend, see docker-compose.yml / DOCKER.md). Wraps `docker compose up
# --build` with a couple of sanity checks so failures are clearer than the
# raw Docker error.
#
# Usage:
#   ./run.sh          # build + run in the foreground (Ctrl+C stops both)
#   ./run.sh -d        # build + run detached, then exit
#   ./run.sh down       # stop and remove the containers
set -euo pipefail

# Always run relative to this script's own directory, so it works no
# matter where it's invoked from.
cd "$(dirname "${BASH_SOURCE[0]}")"

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker is not installed or not on PATH. Install Docker Desktop first." >&2
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "error: docker is installed but the daemon is not reachable. Is Docker Desktop running?" >&2
  exit 1
fi

if [ "${1:-}" = "down" ]; then
  exec docker compose down
fi

echo "Building and starting backend (:8080) and frontend (:3000) ..."
exec docker compose up --build "$@"
