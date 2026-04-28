# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS build

WORKDIR /src
COPY . .

ARG VERSION=0.1.0-dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN test -f go.mod || (echo "go.mod is not present yet; add the Go core before building the runtime image." >&2; exit 1)
RUN test -d cmd/edi-json || (echo "cmd/edi-json is not present yet; add the CLI entrypoint before building the runtime image." >&2; exit 1)
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/johnmonarch/ediforge/internal/app.Version=${VERSION} -X github.com/johnmonarch/ediforge/internal/app.Commit=${COMMIT} -X github.com/johnmonarch/ediforge/internal/app.Date=${DATE}" -o /out/edi-json ./cmd/edi-json

FROM scratch

ARG VERSION=0.1.0-dev
ARG COMMIT=unknown
ARG DATE=unknown

LABEL org.opencontainers.image.title="EDIForge"
LABEL org.opencontainers.image.description="Local-first EDI-to-JSON translator for X12 and UN/EDIFACT"
LABEL org.opencontainers.image.source="https://github.com/johnmonarch/EDIForge"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT}"
LABEL org.opencontainers.image.created="${DATE}"

COPY --from=build /out/edi-json /edi-json
WORKDIR /work
EXPOSE 8765

ENTRYPOINT ["/edi-json"]
