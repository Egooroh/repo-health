#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd -- "$repo_root"
runtime=auto
port="${REPO_HEALTH_PORT:-8080}"
prompt_token=1
while [ "$#" -gt 0 ]; do
  case "$1" in
    --runtime|--port)
      if [ "$#" -lt 2 ]; then printf '%s\n' 'Missing option value.' >&2; exit 1; fi
      if [ "$1" = --runtime ]; then runtime="$2"; else port="$2"; fi
      shift 2 ;;
    --no-token-prompt) prompt_token=0; shift ;;
    *) printf '%s\n' 'Usage: bash start.sh [--runtime auto|go|docker] [--port 8081] [--no-token-prompt]' >&2; exit 1 ;;
  esac
done
case "$port" in ''|*[!0-9]*) echo 'Port must be 1..65535.' >&2; exit 1;; esac
if [ "${#port}" -gt 5 ]; then echo 'Port must be 1..65535.' >&2; exit 1; fi
port=$((10#$port))
if [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then echo 'Port must be 1..65535.' >&2; exit 1; fi
if [ "$runtime" = auto ]; then
  if command -v go >/dev/null 2>&1; then runtime=go
  elif command -v docker >/dev/null 2>&1; then runtime=docker
  else echo 'Install Go (go.dev/dl) or Docker with Compose, then retry.' >&2; exit 1; fi
fi
case "$runtime" in
  go)
    command -v go >/dev/null 2>&1 || { echo 'Go is not installed.' >&2; exit 1; }
    command -v npm >/dev/null 2>&1 || { echo 'npm is required to build the web UI (web/dist is embedded into the binary). Install Node.js or use --runtime docker.' >&2; exit 1; } ;;
  docker)
    command -v docker >/dev/null 2>&1 || { echo 'Docker is not installed.' >&2; exit 1; }
    docker compose version >/dev/null 2>&1 || { echo 'Docker Compose v2 is required.' >&2; exit 1; }
    docker info >/dev/null 2>&1 || { echo 'Start Docker (Linux containers), then retry.' >&2; exit 1; } ;;
  *) echo 'Runtime must be auto, go or docker.' >&2; exit 1 ;;
esac
printf '%s\n' 'Connect your personal SourceCraft token on the website.'
printf 'Repo Health: %s, http://127.0.0.1:%s (Ctrl+C to stop)\n' "$runtime" "$port"
if [ "$runtime" = docker ]; then
  export REPO_HEALTH_PORT="$port"
  exec docker compose --project-directory "$repo_root" -f "$repo_root/compose.yaml" up --build --exit-code-from repo-health
fi
export GOCACHE="${GOCACHE:-$repo_root/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$repo_root/.cache/go-mod}"
export GOTMPDIR="${GOTMPDIR:-$repo_root/.cache/tmp}"
mkdir -p -- "$GOCACHE" "$GOMODCACHE" "$GOTMPDIR"
if [ ! -d "$repo_root/web/frontend/node_modules" ]; then
  npm --prefix "$repo_root/web/frontend" install
fi
npm --prefix "$repo_root/web/frontend" run build
export HTTP_ADDR="127.0.0.1:$port"
export DATABASE_PATH="${DATABASE_PATH:-$repo_root/data/repo-health.db}"
build_dir="$(mktemp -d "$GOTMPDIR/repo-health-launch.XXXXXX")"
cleanup() { rm -f -- "$build_dir/repo-health"; rmdir -- "$build_dir" 2>/dev/null || true; }
trap cleanup EXIT
go build -trimpath -o "$build_dir/repo-health" ./cmd/repo-health
"$build_dir/repo-health"
