# Docker

The Dockerfile scaffold is in `docker/Dockerfile`.

## Build

```bash
docker build -f docker/Dockerfile -t ediforge/edi-json .
```

Official release images publish to `ghcr.io/johnmonarch/ediforge` from version tags.

## Run

```bash
docker run --rm \
  -p 8765:8765 \
  -v "$PWD:/work" \
  ediforge/edi-json serve --host 0.0.0.0 --port 8765
```

For a published release image:

```bash
docker run --rm \
  -p 8765:8765 \
  -v "$PWD:/work" \
  ghcr.io/johnmonarch/ediforge:0.1.0-alpha.1 serve --host 0.0.0.0 --port 8765
```

When binding outside localhost, the application should require an API token unless the user explicitly chooses an unsafe override.

## Notes

The final image should contain the compiled Go binary and embedded web assets. It should not require Node.js at runtime.
