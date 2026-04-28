# Docker

The Dockerfile scaffold is in `docker/Dockerfile`.

## Build

```bash
docker build -f docker/Dockerfile -t ediforge/edi-json .
```

## Run

```bash
docker run --rm \
  -p 8765:8765 \
  -v "$PWD:/work" \
  ediforge/edi-json serve --host 0.0.0.0 --port 8765
```

When binding outside localhost, the application should require an API token unless the user explicitly chooses an unsafe override.

## Notes

The final image should contain the compiled Go binary and embedded web assets. It should not require Node.js at runtime.

