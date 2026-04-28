# CLI Reference

The CLI executable is expected to be named `edi-json`.

## Commands

```bash
edi-json translate input.edi
edi-json translate ./incoming-folder --pretty
edi-json translate input.edi --standard x12 --pretty
edi-json translate input.edi --output output.json
edi-json translate input.edi --mode structural
edi-json translate input.edi --mode annotated
edi-json translate input.edi --mode semantic --schema ./schemas/examples/x12-850-basic.json
edi-json validate input.edi
edi-json detect input.edi
edi-json serve
edi-json schemas list
edi-json schemas validate ./schemas/examples/x12-850-basic.json
edi-json explain input.edi --segment BEG
```

## Global Flags

```text
--config string
--log-level string
--json-errors
--no-color
--quiet
--verbose
```

## Translate Flags

```text
--standard auto|x12|edifact
--mode structural|annotated|semantic
--schema string
--schema-id string
--pretty
--compact
--output string
--include-raw
--include-offsets
--allow-partial
--no-store
```

When `translate` receives a directory path, it recursively processes files ending in
`.edi`, `.x12`, `.edifact`, or `.txt` and emits a deterministic batch JSON response.
The command exits non-zero if any file fails, while still writing per-file results.

## Configuration

The CLI loads configuration from `~/.edi-json/config.yml` and then `./edi-json.yml`
when those files exist. Project config overrides user config for scalar settings.
Schema paths from project config are searched before user schema paths, followed by
the built-in public-safe examples.

```yaml
translation:
  defaultMode: annotated
  includeRawSegments: false
schemas:
  paths:
    - ./schemas
server:
  host: 127.0.0.1
  port: 8765
```

## Validate Flags

```text
--standard auto|x12|edifact
--schema string
--schema-id string
--level syntax|schema|partner
--json
--strict
```

## Serve Flags

```text
--host string
--port int
--token string
--require-token
--open
--no-web
--max-body-mb int
--cors-origin string
```

## Exit Codes

```text
0 success
1 parse or validation error
2 CLI usage error
3 schema or mapping error
4 file or permission error
5 internal error
6 security or unsafe config error
```
