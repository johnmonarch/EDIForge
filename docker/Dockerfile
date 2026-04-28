# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS build

WORKDIR /src
COPY . .

RUN test -f go.mod || (echo "go.mod is not present yet; add the Go core before building the runtime image." >&2; exit 1)
RUN test -d cmd/edi-json || (echo "cmd/edi-json is not present yet; add the CLI entrypoint before building the runtime image." >&2; exit 1)
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/edi-json ./cmd/edi-json

FROM scratch

COPY --from=build /out/edi-json /edi-json
WORKDIR /work
EXPOSE 8765

ENTRYPOINT ["/edi-json"]
