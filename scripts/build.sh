#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

: "${GOCACHE:=${TMPDIR:-/tmp}/ediforge-go-build}"
: "${GOMODCACHE:=${TMPDIR:-/tmp}/ediforge-go-mod}"
: "${BIN_DIR:=bin}"
export GOCACHE GOMODCACHE

echo "==> Checking embedded web assets"
test -f internal/web/dist/index.html
test -f internal/web/dist/styles.css
test -f internal/web/dist/app.js

if [ -f web/package.json ] && [ -d web/node_modules ]; then
  echo "==> Building React web assets"
  (cd web && npm run build)
else
  echo "==> Skipping React build; run npm install in web/ to enable it"
fi

if [ -f go.mod ] && [ -d cmd/edi-json ]; then
  echo "==> Building edi-json"
  mkdir -p "$BIN_DIR"
  go build -o "$BIN_DIR/edi-json" ./cmd/edi-json
else
  echo "==> Skipping Go build; go.mod or cmd/edi-json is not present yet"
fi

echo "==> Build scaffold complete"
