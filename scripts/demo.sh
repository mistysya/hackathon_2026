#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
for tool in go node npm curl; do
  command -v "$tool" >/dev/null || { echo "Required tool missing: $tool" >&2; exit 1; }
done
WORK="$(mktemp -d)"
API_PID=""
UI_PID=""
cleanup() {
  trap - EXIT INT TERM
  if [[ -n "$UI_PID" ]]; then kill "$UI_PID" 2>/dev/null || true; wait "$UI_PID" 2>/dev/null || true; fi
  if [[ -n "$API_PID" ]]; then kill "$API_PID" 2>/dev/null || true; wait "$API_PID" 2>/dev/null || true; fi
  rm -rf "$WORK"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

cd "$ROOT/frontend"
if [[ ! -d node_modules ]]; then npm ci; fi
cd "$ROOT/backend"
go build -o "$WORK/api" ./cmd/api
HTTP_ADDR=127.0.0.1:8080 "$WORK/api" &
API_PID=$!
ready=false
for ((i=0; i<50; i++)); do
  sleep 0.1
  if ! kill -0 "$API_PID" 2>/dev/null; then echo "Backend failed to start (is port 8080 occupied?)" >&2; exit 1; fi
  if curl --silent --fail http://127.0.0.1:8080/employees >/dev/null; then ready=true; break; fi
done
if [[ "$ready" != true ]]; then echo "Backend readiness timed out" >&2; exit 1; fi
cd "$ROOT/frontend"
VITE_USE_FIXTURES=false API_PROXY_TARGET=http://127.0.0.1:8080 node node_modules/vite/bin/vite.js --host 127.0.0.1 --port 5173 --strictPort &
UI_PID=$!
echo "Demo: http://127.0.0.1:5173 — real Go API + SQLite; deterministic agent fixtures, no credentials required."
echo "Upload docs/fixtures/employees.csv, select Demo User, and follow README.md. Ctrl-C stops both servers."
wait "$UI_PID"
