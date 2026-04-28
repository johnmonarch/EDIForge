# EDIForge

[![CI](https://github.com/johnmonarch/ediforge/actions/workflows/ci.yml/badge.svg)](https://github.com/johnmonarch/ediforge/actions/workflows/ci.yml)
[![Release](https://github.com/johnmonarch/ediforge/actions/workflows/release.yml/badge.svg)](https://github.com/johnmonarch/ediforge/actions/workflows/release.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/johnmonarch/ediforge.svg)](https://pkg.go.dev/github.com/johnmonarch/ediforge)
[![Container](https://img.shields.io/badge/GHCR-ediforge-1f6feb)](https://github.com/johnmonarch/ediforge/pkgs/container/ediforge)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](LICENSE)
[![GitHub release](https://img.shields.io/github/v/release/johnmonarch/ediforge?include_prereleases&sort=semver)](https://github.com/johnmonarch/ediforge/releases)

EDIForge is a local-first, open-source EDI-to-JSON translator for logistics, commerce, and integration developers. It converts X12 and UN/EDIFACT files into inspectable JSON through one translation engine exposed by a CLI, a local REST API, and an embedded web interface.

The project is built for teams that need to inspect, validate, test, and automate EDI workflows without sending shipment, order, invoice, or warehouse data to a cloud service.

## Why EDIForge

- Translate X12 and UN/EDIFACT into structural JSON first.
- Add annotated and semantic JSON with user-provided schemas and mappings.
- Use the same engine from the CLI, REST API, and local web UI.
- Keep data local by default: no telemetry, uploads, or external network calls in normal workflows.
- Ship public-safe starter schemas without bundling copyrighted X12 guides or proprietary partner maps.

## Quick Install

Build from source:

```bash
git clone https://github.com/johnmonarch/ediforge.git
cd ediforge
./scripts/build.sh
```

Run the built CLI:

```bash
./bin/edi-json version
./bin/edi-json translate ./testdata/x12/850-basic.edi --mode structural --pretty
```

Install with Go:

```bash
go install github.com/johnmonarch/ediforge/cmd/edi-json@latest
```

Install with Homebrew:

```bash
brew tap johnmonarch/tap
brew install edi-json
```

Release binaries and container images, when published, are available from:

- [GitHub Releases](https://github.com/johnmonarch/ediforge/releases)
- [Packages](https://github.com/johnmonarch/ediforge/pkgs/container/ediforge)

## Quick Use

Detect the standard:

```bash
edi-json detect ./testdata/x12/850-basic.edi --json
```

Translate one file:

```bash
edi-json translate input.edi --standard auto --mode structural --pretty
```

Translate a folder:

```bash
edi-json translate ./incoming --pretty
```

Validate in CI:

```bash
edi-json validate input.edi --level syntax --json
```

Start the local API and web UI:

```bash
edi-json serve --host 127.0.0.1 --port 8765
```

Then open:

```text
http://127.0.0.1:8765
```

## Output Modes

- `structural`: parsed envelopes, groups, transactions, segments, elements, and components.
- `annotated`: structural output enriched with schema-derived labels and descriptions.
- `semantic`: business-shaped JSON produced from user-provided schemas and mappings.

Example semantic translation:

```bash
edi-json translate input.edi \
  --mode semantic \
  --schema ./schemas/examples/x12-850-basic.json \
  --pretty
```

## REST API

Start the server:

```bash
edi-json serve --host 127.0.0.1 --port 8765
```

Translate EDI:

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

Core endpoints:

- `GET /health`
- `GET /api/v1/version`
- `POST /api/v1/detect`
- `POST /api/v1/translate`
- `POST /api/v1/validate`
- `GET /api/v1/schemas`
- `POST /api/v1/schemas/validate`
- `POST /api/v1/explain`

## Local Web UI

The embedded UI is served by `edi-json serve` and expects the API under the same origin. It supports paste or file-loaded EDI input, structural/annotated/semantic mode selection, optional schema IDs, validation, warnings/errors, and client-side copy/download of JSON responses.

The checked-in static UI lives in `internal/web/dist`. The future React/Vite source scaffold lives in `web/`, but the Go server does not require `npm install` to serve the embedded UI.

## Local-First Privacy

EDIForge is designed to run on your workstation, build server, or private infrastructure.

- No outbound network calls in normal CLI, API, or web workflows.
- No telemetry.
- Server binds to `127.0.0.1` by default.
- Binding outside localhost should require an explicit flag and API token.
- Raw EDI should not be logged by default.
- Browser input remains in memory unless the user explicitly saves or downloads output.

## Schemas And Mapping

Starter schemas are provided in `schemas/examples/` for common X12 and EDIFACT messages, including X12 850, 810, 856, 214, 990, 997, 999 and EDIFACT ORDERS, ORDRSP, DESADV, and INVOIC.

These examples are intentionally public-safe. They do not include proprietary implementation-guide text, partner-specific rules, or restricted code lists. Real trading-partner maps should be supplied by users who have the right to use them.

Configuration can be loaded from `~/.edi-json/config.yml` and `./edi-json.yml`:

```yaml
translation:
  defaultMode: annotated
schemas:
  paths:
    - ./schemas
```

## Containers And Release Artifacts

Build the local container image:

```bash
docker build -f docker/Dockerfile -t ediforge/edi-json .
```

Run the local API and web UI in a container:

```bash
docker run --rm \
  -p 8765:8765 \
  -v "$PWD:/work" \
  ediforge/edi-json serve --host 0.0.0.0 --port 8765
```

The runtime image is intended to contain the compiled Go binary and embedded web assets, with no Node.js requirement at runtime. When official artifacts are published, prefer the signed or checksummed release asset for your platform, or the published container image for repeatable deployments.

Official container images publish to `ghcr.io/johnmonarch/ediforge` from version tags.

## Documentation

- [Install](docs/install.md)
- [Quickstart](docs/quickstart.md)
- [CLI](docs/cli.md)
- [REST API](docs/api.md)
- [Web UI](docs/web-ui.md)
- [Examples](docs/examples.md)
- [Schemas and mapping](docs/schemas-and-mapping.md)
- [Validation](docs/validation.md)
- [Docker](docs/docker.md)
- [Standards and IP policy](docs/standards-ip-policy.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)

## Contributing

Contributions are welcome. Start with [CONTRIBUTING.md](CONTRIBUTING.md) for local setup, development expectations, and pull request guidance.

Please keep examples, schemas, and fixtures redistributable. Do not contribute paid X12 standards text, restricted implementation-guide content, proprietary partner maps, or data that exposes real trading-party information unless you have explicit rights to share it publicly.

## Security

Report vulnerabilities using the process in [SECURITY.md](SECURITY.md). Avoid opening public issues for sensitive reports.

## License

EDIForge is licensed under the Apache License 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
