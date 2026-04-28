# Quickstart

EDIForge is a local-first EDI-to-JSON translator for X12 and UN/EDIFACT. The Go application provides a CLI, local REST API, embedded web UI, structural output, annotated output, semantic mapping, and syntax/schema validation.

## Build

```bash
./scripts/build.sh
```

The script builds the React source if dependencies are already installed, checks that embedded static assets exist, and builds `bin/edi-json` once `cmd/edi-json` is available.

## Test

```bash
./scripts/test.sh
```

The script validates bundled JSON examples and runs Go/frontend checks when the corresponding project files are present.

## Translate

```bash
edi-json translate input.edi --standard auto --mode structural --pretty
```

Translate a folder of EDI files:

```bash
edi-json translate ./incoming --pretty
```

Use annotated mode with a schema:

```bash
edi-json translate input.edi --mode annotated --schema-id x12-850-basic --pretty
```

## Validate

```bash
edi-json validate input.edi --level syntax --json
```

## Detect

```bash
edi-json detect input.edi --json
```

## Serve the Local UI

```bash
edi-json serve --host 127.0.0.1 --port 8765
```

Then open:

```text
http://127.0.0.1:8765
```

The server should bind to localhost by default and should require a token before binding to non-localhost interfaces.
