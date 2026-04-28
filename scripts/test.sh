#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

: "${GOCACHE:=${TMPDIR:-/tmp}/ediforge-go-build}"
: "${GOMODCACHE:=${TMPDIR:-/tmp}/ediforge-go-mod}"
export GOCACHE GOMODCACHE

echo "==> Validating JSON schema examples"
if command -v python3 >/dev/null 2>&1; then
  for file in schemas/examples/*.json; do
    python3 -m json.tool "$file" >/dev/null
  done
elif command -v node >/dev/null 2>&1; then
  for file in schemas/examples/*.json; do
    node -e "JSON.parse(require('fs').readFileSync(process.argv[1], 'utf8'))" "$file"
  done
else
  echo "No JSON parser found; install python3 or node to validate schema examples" >&2
  exit 1
fi

echo "==> Checking embedded web assets"
test -f internal/web/dist/index.html
test -f internal/web/dist/styles.css
test -f internal/web/dist/app.js

if [ -f go.mod ]; then
  echo "==> Running Go tests"
  go test ./...
else
  echo "==> Skipping Go tests; go.mod is not present yet"
fi

if [ -f web/package.json ] && [ -d web/node_modules ]; then
  echo "==> Running frontend type check"
  (cd web && npm run lint)
else
  echo "==> Skipping frontend type check; run npm install in web/ to enable it"
fi

echo "==> Test scaffold complete"
