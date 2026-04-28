# EDIForge

EDIForge is a local-first, open-source EDI-to-JSON translator for logistics developers. It is designed to convert X12 and UN/EDIFACT files into inspectable JSON through a CLI, built-in REST API, and local web interface without sending shipment, order, invoice, or warehouse data to a cloud service.

This repository includes the project requirements, Go implementation work, public-safe schema examples, static web UI assets for embedding, future React/Vite web scaffolding, Docker packaging scaffolding, and contributor/security documentation.

## Goals

- Translate X12 and EDIFACT into structural JSON first.
- Add annotated and semantic JSON through user-provided schemas and mappings.
- Expose one translation engine through CLI, REST API, and the local web UI.
- Keep data private by default: no telemetry, no uploads, no external network calls.
- Avoid bundling copyrighted X12 implementation guides or proprietary partner maps.

## Quickstart

Build the CLI and run the local workflow:

```bash
./bin/edi-json translate ./examples/order.edi --mode structural --pretty
./bin/edi-json detect ./examples/order.edi --json
./bin/edi-json validate ./examples/order.edi --level syntax --json
./bin/edi-json serve --host 127.0.0.1 --port 8765
```

Build helpers are provided, but they are defensive while the Go application is still being created:

```bash
./scripts/build.sh
./scripts/test.sh
```

The embedded static UI lives in `internal/web/dist` and expects the local server to expose the API under the same origin.

## CLI Examples

Translate a file to structural JSON:

```bash
edi-json translate input.edi --standard auto --mode structural --pretty
```

Translate a folder of EDI files:

```bash
edi-json translate ./incoming --pretty
```

Inspect parsed EDI with schema-derived annotations:

```bash
edi-json translate input.edi --mode annotated --schema-id x12-850-basic --pretty
```

Write JSON to a file:

```bash
edi-json translate input.edi --output output.json
```

Use a schema for semantic output:

```bash
edi-json translate input.edi \
  --mode semantic \
  --schema ./schemas/examples/x12-850-basic.json
```

Validate in CI:

```bash
edi-json validate input.edi --json
```

Recommended exit codes:

- `0`: success
- `1`: parse or validation error
- `2`: invalid CLI usage
- `3`: schema or mapping error
- `4`: file or permission error
- `5`: internal error
- `6`: unsafe server configuration

## REST API Examples

Start the local server:

```bash
edi-json serve --host 127.0.0.1 --port 8765
```

Translate:

```bash
curl -s http://127.0.0.1:8765/api/v1/translate \
  -H 'Content-Type: application/json' \
  -d '{
    "input": "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260427*1200*U*00401*000000001*0*T*>~GS*PO*SENDER*RECEIVER*20260427*1200*1*X*004010~ST*850*0001~BEG*00*SA*PO-10001**20260427~SE*2*0001~GE*1*1~IEA*1*000000001~",
    "standard": "auto",
    "mode": "structural",
    "options": {
      "pretty": true,
      "includeEnvelope": true
    }
  }'
```

Validate:

```bash
curl -s http://127.0.0.1:8765/api/v1/validate \
  -H 'Content-Type: application/json' \
  -d '{"input":"UNB+UNOC:3+SENDER+RECEIVER+260427:1200+1'\''UNH+1+ORDERS:D:96A:UN'\''UNT+2+1'\''UNZ+1+1'\''","standard":"auto","level":"syntax"}'
```

Detect:

```bash
curl -s http://127.0.0.1:8765/api/v1/detect \
  -H 'Content-Type: application/json' \
  -d '{"input":"ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260427*1200*U*00401*000000001*0*T*>~"}'
```

Expected MVP endpoints:

- `GET /health`
- `GET /api/v1/version`
- `POST /api/v1/detect`
- `POST /api/v1/translate`
- `POST /api/v1/validate`
- `GET /api/v1/schemas`
- `POST /api/v1/schemas/validate`
- `POST /api/v1/explain`

## Web UI

When the server is running, open:

```text
http://127.0.0.1:8765
```

The static UI in `internal/web/dist` supports:

- Paste or file-loaded EDI input.
- Structural, annotated, and semantic mode selection.
- Optional schema ID.
- Calls to `POST /api/v1/detect`, `POST /api/v1/translate`, and `POST /api/v1/validate`.
- JSON rendering with warnings and errors.
- Client-side copy and download of the API response JSON.

The future React/Vite source scaffold lives in `web/`. The Go server must not require `npm install` to serve the embedded UI.

## Configuration

EDIForge loads optional user config from `~/.edi-json/config.yml` and project config from `./edi-json.yml`. Project config can set translation defaults and schema search paths while preserving the built-in public-safe examples as a fallback.

```yaml
translation:
  defaultMode: annotated
schemas:
  paths:
    - ./schemas
```

## Schema Examples

Public-safe schema examples are available in:

- `schemas/examples/x12-850-basic.json`
- `schemas/examples/x12-810-basic.json`
- `schemas/examples/x12-856-basic.json`
- `schemas/examples/x12-214-basic.json`
- `schemas/examples/x12-990-basic.json`
- `schemas/examples/x12-997-basic.json`
- `schemas/examples/x12-999-basic.json`
- `schemas/examples/edifact-orders-basic.json`
- `schemas/examples/edifact-ordrsp-basic.json`
- `schemas/examples/edifact-desadv-basic.json`
- `schemas/examples/edifact-invoic-basic.json`

These examples are starter templates. They intentionally avoid proprietary implementation-guide text, partner-specific rules, and restricted code lists. Real trading-partner maps should be supplied by users who have the right to use them.

## Local-First Privacy

EDIForge is designed to run locally by default:

- No outbound network calls in normal CLI, API, or web workflows.
- No telemetry unless explicitly added and opted into later.
- Server binds to `127.0.0.1` by default.
- Binding outside localhost should require an explicit flag and API token.
- Raw EDI should not be logged by default.
- Local history/storage should remain disabled by default.
- Browser input should remain in memory unless a user explicitly saves or downloads output.

## X12 IP Policy

X12 standards and many implementation guides are copyrighted. EDIForge may support generic X12 syntax, user-provided schemas, public-safe examples, and community-authored metadata that is legally redistributable. EDIForge must not bundle paid X12 standards text, proprietary trading-partner guides, or derivative content from restricted materials without explicit rights.

Users are responsible for ensuring they have the right to import, use, and share any X12 guide, mapping, code list, or partner overlay they provide.

## Docker

The Dockerfile scaffold is in `docker/Dockerfile`.

Intended runtime:

```bash
docker build -f docker/Dockerfile -t ediforge/edi-json .
docker run --rm -p 8765:8765 -v "$PWD:/work" ediforge/edi-json \
  serve --host 0.0.0.0 --port 8765
```

## Documentation

See `docs/` for focused guides:

- `docs/quickstart.md`
- `docs/cli.md`
- `docs/api.md`
- `docs/web-ui.md`
- `docs/schemas-and-mapping.md`
- `docs/validation.md`
- `docs/standards-ip-policy.md`
- `docs/docker.md`

## License

EDIForge is licensed under Apache-2.0. See `LICENSE` and `NOTICE`.
