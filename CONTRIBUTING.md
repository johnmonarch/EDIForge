# Contributing to EDIForge

EDIForge is being built as a local-first, standards-aware EDI translator. Contributions should preserve privacy defaults, keep parser logic separate from CLI/API/web adapters, and avoid restricted standards content.

## Development Principles

- Keep one translation core behind the CLI, REST API, and web UI.
- Prefer structural JSON support before semantic conveniences.
- Keep the default experience local-only and private.
- Do not log raw EDI, partner IDs, names, addresses, or control numbers by default.
- Treat schemas and mappings as data, not executable code.
- Keep examples synthetic or otherwise clearly redistributable.

## Repository Boundaries

The intended architecture is:

```text
cmd/edi-json        CLI entrypoint
internal/api        HTTP adapters
internal/cli        CLI adapters
internal/detect     standard and delimiter detection
internal/parse      X12 and EDIFACT parsers
internal/model      shared document model
internal/translate  translation service
internal/jsonout    structural and annotated JSON output
internal/schema     schema loading and validation
internal/mapping    semantic mapping
internal/web/dist   embedded static UI
pkg/translator      supported Go API wrapper
```

Parser, schema, mapping, and JSON output packages must not import CLI, API, or web packages.

## Local Checks

Use the helper scripts:

```bash
./scripts/build.sh
./scripts/test.sh
```

When the Go codebase exists, expected checks include:

```bash
go test ./...
go vet ./...
```

When the React source is being actively changed:

```bash
cd web
npm install
npm run build
npm run lint
```

The embedded UI in `internal/web/dist` must remain usable without requiring npm.

## Schema and Mapping Contributions

Every bundled schema or mapping must include:

- Stable `id`
- `standard`
- transaction or message identifier
- version or release when known
- `source`
- `license`
- short IP/safety note when useful

Do not submit:

- Paid X12 guide text.
- Proprietary trading-partner maps.
- Restricted code lists.
- Generated content derived from materials you do not have permission to redistribute.

If you contribute partner overlays, use synthetic partner names unless the partner has explicitly approved publication.

## Pull Requests

For each PR, include:

- What changed.
- Which CLI/API/web behavior is affected.
- Whether sample EDI, schemas, or maps are synthetic/public-safe.
- Which tests or checks were run.

Security fixes should follow `SECURITY.md` instead of a public issue when they expose sensitive behavior.

