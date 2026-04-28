# Technical Engineering Guide: Local-First EDI-to-JSON Translator

Companion document to the PRD for **EDIForge**.

## 1. Engineering Summary

EDIForge should be built as a local-first developer tool with one parsing and translation core exposed through three surfaces:

1. CLI
2. Local REST API
3. Local web interface

The recommended v1 implementation is **Go for the base application**, with an embedded **React + TypeScript + Vite** local web UI.

The strongest alternative is **Rust for the base application**, using **Clap** for the CLI and **Axum + Tokio** for the REST API.

## 2. Recommended Stack

### Recommended v1 stack

```text
Core language: Go
CLI: Cobra or Kong
REST API: Go net/http ServeMux, or Chi if you want middleware ergonomics
Web UI: React + TypeScript + Vite
Static asset delivery: Go embed
Config: YAML
Schema/mapping files: YAML and JSON
Local optional storage: SQLite
Container: Docker
Build/release: GoReleaser
Tests: Go test, golden fixtures, fuzz tests
```

### Why Go is the recommended v1 choice

Go is the best first implementation choice for this project because it gives you:

1. Single static binary distribution.
2. Simple cross-compilation for Linux, macOS, and Windows.
3. Fast startup.
4. Low memory use.
5. Excellent standard library support for HTTP, JSON, files, streams, and embedding.
6. Easier onboarding for logistics industry developers than Rust.
7. Enough performance for large EDI files.
8. Simple Docker images.
9. Good support for streaming parsers.
10. A straightforward path to embedding the web UI into the same binary.

For this specific tool, most performance risk will come from poor parser design, repeated allocations, and loading huge files into memory, not from Go being too slow.

### When Rust would be better

Rust is a strong choice if the project prioritizes:

1. Maximum parser correctness and safety.
2. A zero-copy parser architecture.
3. WASM support as a major product goal.
4. Library use by other Rust projects.
5. Very strict memory behavior.
6. Long-term ability to expose the parser core in multiple runtimes.

Rust is a more demanding first version because it increases complexity around lifetimes, async runtime choices, contributor onboarding, and binary integration. It may be worth using later for a parser core if the Go version proves useful.

### Final recommendation

Build v1 in Go.

Use Rust only if you already know you want the parser core to become a formal, low-level parsing library with WASM as a first-class target.

## 3. High-Level Architecture

```text
                   +----------------------+
                   |      CLI Surface     |
                   +----------+-----------+
                              |
                              v
+------------------+    +-----+------+    +------------------+
| Local Web UI     | -> | REST API   | -> | Translation Core |
| React/Vite       |    | net/http   |    | Go packages      |
+------------------+    +------------+    +--------+---------+
                                                |
                                                v
                                      +---------+----------+
                                      | Parser Engines     |
                                      | X12 and EDIFACT    |
                                      +---------+----------+
                                                |
                                                v
                                      +---------+----------+
                                      | Output Engines     |
                                      | Structural JSON    |
                                      | Annotated JSON     |
                                      | Semantic JSON      |
                                      +--------------------+
```

The CLI should be able to call the translation core directly. The REST API should call the same core. The web UI should call the REST API.

Do not build separate parser logic for the CLI, API, and web UI.

## 4. Repository Structure

```text
ediforge/
  README.md
  LICENSE
  NOTICE
  CITATION.cff
  CONTRIBUTING.md
  SECURITY.md
  go.mod
  go.sum

  cmd/
    edi-json/
      main.go

  internal/
    app/
      app.go
      version.go

    api/
      server.go
      routes.go
      handlers_translate.go
      handlers_validate.go
      handlers_detect.go
      handlers_schema.go
      middleware.go
      errors.go

    cli/
      root.go
      translate.go
      validate.go
      detect.go
      serve.go
      schemas.go
      explain.go

    config/
      config.go
      loader.go
      defaults.go

    detect/
      detect.go
      x12.go
      edifact.go

    model/
      document.go
      envelope.go
      segment.go
      element.go
      errors.go
      metadata.go

    parse/
      x12/
        tokenizer.go
        parser.go
        delimiters.go
        validator.go
        errors.go
      edifact/
        tokenizer.go
        parser.go
        delimiters.go
        release.go
        validator.go
        errors.go

    translate/
      translator.go
      options.go
      result.go

    jsonout/
      structural.go
      annotated.go
      semantic.go

    schema/
      loader.go
      schema.go
      validate.go
      registry.go

    mapping/
      mapper.go
      expression.go
      path.go
      transforms.go
      overlays.go

    validate/
      syntax.go
      schema.go
      partner.go

    redact/
      redact.go
      patterns.go

    storage/
      sqlite.go
      history.go

    web/
      embed.go
      dist/

  pkg/
    translator/
      translator.go

  schemas/
    examples/
      x12-850-basic.yml
      edifact-orders-basic.yml

  testdata/
    x12/
    edifact/
    malformed/

  web/
    package.json
    vite.config.ts
    tsconfig.json
    src/
      main.tsx
      App.tsx
      api/
      components/
      pages/
      styles/

  docker/
    Dockerfile

  scripts/
    build.sh
    release.sh
```

## 5. Core Design Principle

The core should expose a stable internal API that knows nothing about CLI flags, HTTP requests, or React.

Recommended public internal interface:

```go
type Translator interface {
    Translate(ctx context.Context, input Input, opts TranslateOptions) (*TranslateResult, error)
    Validate(ctx context.Context, input Input, opts ValidateOptions) (*ValidateResult, error)
    Detect(ctx context.Context, input Input) (*DetectResult, error)
}
```

The CLI and API should only adapt their inputs into this interface.

## 6. Package Boundary Rules

### Allowed dependencies

```text
cmd/edi-json -> internal/cli
internal/cli -> internal/translate, internal/config
internal/api -> internal/translate, internal/config
internal/translate -> parse, detect, validate, schema, mapping, jsonout
internal/parse -> model
internal/jsonout -> model, schema
internal/mapping -> model, schema
pkg/translator -> stable wrapper around internal/translate
```

### Forbidden dependencies

```text
parse -> api
parse -> cli
parse -> web
schema -> api
mapping -> cli
jsonout -> api
```

Parser code should not know whether it is being called from a CLI, REST API, test, or web UI.

## 7. Core Data Model

### Document

```go
type Document struct {
    Standard      Standard       `json:"standard"`
    Version       string         `json:"version,omitempty"`
    Interchanges  []Interchange  `json:"interchanges"`
    Errors        []EDIError     `json:"errors,omitempty"`
    Warnings      []EDIWarning   `json:"warnings,omitempty"`
    Metadata      Metadata       `json:"metadata"`
}
```

### Interchange

```go
type Interchange struct {
    Standard       Standard      `json:"standard"`
    SenderID       string        `json:"senderId,omitempty"`
    ReceiverID     string        `json:"receiverId,omitempty"`
    ControlNumber  string        `json:"controlNumber,omitempty"`
    Groups         []Group       `json:"groups,omitempty"`
    Messages       []Message     `json:"messages,omitempty"`
    RawEnvelope    []Segment     `json:"rawEnvelope,omitempty"`
}
```

### Group

```go
type Group struct {
    FunctionalID   string        `json:"functionalId,omitempty"`
    Version        string        `json:"version,omitempty"`
    ControlNumber  string        `json:"controlNumber,omitempty"`
    Transactions   []Transaction `json:"transactions"`
}
```

### Transaction

```go
type Transaction struct {
    Type           string        `json:"type"`
    Version        string        `json:"version,omitempty"`
    ControlNumber  string        `json:"controlNumber,omitempty"`
    Segments       []Segment     `json:"segments"`
    SegmentCount   int           `json:"segmentCount"`
}
```

### Message

Used for EDIFACT.

```go
type Message struct {
    Type            string        `json:"type"`
    Version         string        `json:"version,omitempty"`
    Release         string        `json:"release,omitempty"`
    ControllingOrg  string        `json:"controllingOrg,omitempty"`
    AssociationCode string        `json:"associationCode,omitempty"`
    Reference       string        `json:"reference,omitempty"`
    Segments        []Segment     `json:"segments"`
    SegmentCount    int           `json:"segmentCount"`
}
```

### Segment

```go
type Segment struct {
    Tag       string     `json:"tag"`
    Position  int        `json:"position"`
    Elements  []Element  `json:"elements"`
    Raw       string     `json:"raw,omitempty"`
    Offset    int64      `json:"offset,omitempty"`
}
```

### Element

```go
type Element struct {
    Index       int         `json:"index"`
    Value       string      `json:"value,omitempty"`
    Components  []string    `json:"components,omitempty"`
}
```

### Error model

```go
type EDIError struct {
    Severity        string `json:"severity"`
    Code            string `json:"code"`
    Message         string `json:"message"`
    Standard        string `json:"standard,omitempty"`
    Segment         string `json:"segment,omitempty"`
    SegmentPosition int    `json:"segmentPosition,omitempty"`
    Element         string `json:"element,omitempty"`
    ByteOffset      int64  `json:"byteOffset,omitempty"`
    Hint            string `json:"hint,omitempty"`
}
```

## 8. Translation Pipeline

Every input should pass through the same pipeline.

```text
1. Read input stream
2. Detect standard and delimiters
3. Tokenize segments
4. Parse envelope/group/message structure
5. Validate syntax
6. Load schema if requested
7. Apply schema validation if requested
8. Apply mapping if semantic mode is requested
9. Render JSON output
10. Return result with metadata, warnings, and errors
```

Recommended top-level function:

```go
func (t *Service) Translate(ctx context.Context, input Input, opts TranslateOptions) (*TranslateResult, error) {
    detected, err := t.detector.Detect(ctx, input.PeekableReader())
    if err != nil {
        return nil, err
    }

    parser := t.parsers.For(detected.Standard)
    doc, parseErr := parser.Parse(ctx, input.Reader(), detected)

    result := &TranslateResult{
        Standard: detected.Standard,
        Metadata: Metadata{},
        Warnings: doc.Warnings,
        Errors: doc.Errors,
    }

    if parseErr != nil && !opts.AllowPartial {
        return result, parseErr
    }

    switch opts.Mode {
    case ModeStructural:
        result.Result = t.json.Structural(doc, opts)
    case ModeAnnotated:
        result.Result = t.json.Annotated(doc, opts)
    case ModeSemantic:
        schema, err := t.schemas.Resolve(opts.SchemaID, opts.SchemaPath)
        if err != nil {
            return nil, err
        }
        result.Result, result.Errors = t.mapper.Map(doc, schema, opts)
    }

    return result, nil
}
```

## 9. Input Handling

Input should support:

1. File path
2. Stdin
3. Raw string from API
4. Uploaded file from API or web UI
5. Folder batch input
6. Future stream input

Input abstraction:

```go
type Input struct {
    Name        string
    Reader      io.Reader
    Size        int64
    ContentType string
}
```

For detection, wrap readers in a buffered reader so the detector can peek without consuming the stream permanently.

```go
br := bufio.NewReaderSize(input.Reader, 64*1024)
sample, _ := br.Peek(minBytes)
```

## 10. X12 Parser Engineering

### X12 detection

An X12 interchange usually starts with `ISA`.

Detection rules:

1. Trim UTF-8 BOM if present.
2. Ignore leading whitespace only if configured to be forgiving.
3. Detect `ISA` at the start.
4. Read the fixed-width ISA segment.
5. Extract delimiters from ISA positions.
6. Confirm `GS`, `ST`, or valid segment terminator nearby.

### X12 delimiters

X12 delimiters are defined by the ISA segment.

Common defaults:

```text
Element separator: *
Segment terminator: ~
Component separator: >
Repetition separator: ^ in later versions
```

Implementation notes:

1. Do not hard-code `*` and `~`.
2. ISA has fixed element widths.
3. The element separator is the character after `ISA`.
4. The component separator is the ISA16 value.
5. The segment terminator is the character immediately after the ISA segment.
6. Repetition separator is version-dependent and often available from ISA11 in newer versions.

### X12 tokenization

Tokenizer output:

```go
type Token struct {
    Type       TokenType
    Tag        string
    Elements   []string
    Raw        string
    Offset     int64
    Position   int
}
```

Tokenizer steps:

```text
1. Read until segment terminator.
2. Track byte offset and segment position.
3. Split segment by element separator.
4. First token is segment tag.
5. Split composite elements by component separator.
6. Preserve raw segment if requested.
7. Emit segment token.
```

### X12 parser state machine

```text
Start
  -> ISA
  -> GS
  -> ST
  -> Segments
  -> SE
  -> GE
  -> IEA
  -> Done
```

Parser should tolerate multiple functional groups and multiple transactions per group.

### X12 validation checks

Always check:

1. ISA exists.
2. IEA exists.
3. ISA13 matches IEA02.
4. GS exists where expected.
5. GE exists where expected.
6. GS06 matches GE02.
7. ST exists where expected.
8. SE exists where expected.
9. ST02 matches SE02.
10. SE01 segment count matches actual count.
11. Segment tags are syntactically valid.
12. Empty transaction bodies are flagged.
13. Unexpected segment outside transaction is flagged.

## 11. EDIFACT Parser Engineering

### EDIFACT detection

EDIFACT may start with:

```text
UNA
UNB
```

Detection rules:

1. Trim UTF-8 BOM if present.
2. If `UNA` exists, parse service string advice.
3. If no `UNA`, use default EDIFACT separators.
4. Confirm `UNB` appears after `UNA` or at start.
5. Detect `UNH` message header.

### EDIFACT default separators

If `UNA` is absent:

```text
Component separator: :
Data element separator: +
Decimal mark: .
Release character: ?
Reserved: space
Segment terminator: '
```

### EDIFACT release character

The release character escapes special characters.

Example:

```text
FTX+AAI+++Text with ?+ plus sign'
```

The tokenizer must not split on escaped separators.

Tokenizer logic:

```text
1. Scan character by character.
2. If release character is found, append next character literally.
3. If segment terminator is found and not released, end segment.
4. If data element separator is found and not released, end element.
5. If component separator is found and not released, end component.
```

### EDIFACT parser state machine

```text
Start
  -> UNA optional
  -> UNB
  -> UNG optional
  -> UNH
  -> Segments
  -> UNT
  -> UNE optional
  -> UNZ
  -> Done
```

### EDIFACT validation checks

Always check:

1. UNB exists.
2. UNZ exists.
3. UNB control reference matches UNZ control reference.
4. UNH exists.
5. UNT exists.
6. UNH message reference matches UNT message reference.
7. UNT segment count matches actual count.
8. Release characters are valid.
9. Unterminated composites are flagged.
10. Unexpected segment outside message is flagged.

## 12. Streaming vs Tree Mode

### Tree mode

Tree mode builds the full `Document` in memory.

Use for:

1. Web UI.
2. Annotated JSON.
3. Semantic mapping.
4. Small and medium files.
5. Developer inspection.

### Streaming mode

Streaming mode emits parse events without holding the entire document.

Use for:

1. Large files.
2. Batch validation.
3. Future NDJSON output.
4. Future pipeline integrations.

Event interface:

```go
type Event struct {
    Type      EventType
    Segment   *Segment
    Error     *EDIError
    Metadata  map[string]any
}
```

Event types:

```go
const (
    EventStartInterchange EventType = "start_interchange"
    EventStartGroup       EventType = "start_group"
    EventStartTransaction EventType = "start_transaction"
    EventSegment          EventType = "segment"
    EventEndTransaction   EventType = "end_transaction"
    EventEndGroup         EventType = "end_group"
    EventEndInterchange   EventType = "end_interchange"
    EventError            EventType = "error"
)
```

MVP can implement tree mode first, but tokenizers should be written so streaming mode is not blocked later.

## 13. JSON Output Design

### Structural JSON

Structural JSON should preserve the source document as faithfully as possible.

Rules:

1. Do not rename business fields.
2. Do not discard unknown segments.
3. Preserve order.
4. Preserve envelope metadata.
5. Include raw segment only if requested.
6. Include offsets only if requested.
7. Include parser warnings and errors.

### Annotated JSON

Annotated JSON should add labels from legal metadata sources.

Rules:

1. Never invent labels when unknown.
2. Keep original tag and element index.
3. Include source of label metadata when available.
4. Do not require paid guide content.
5. Allow user schemas to provide labels.

### Semantic JSON

Semantic JSON should be produced only with a schema/mapping file.

Rules:

1. Output shape is controlled by schema.
2. Unknown source fields should be collected if configured.
3. Required target fields should produce validation errors.
4. Transform functions should be explicit.
5. Mapping should be deterministic.

## 14. Schema and Mapping Engine

### Schema file goals

Schema files should define:

1. Document type.
2. Source standard.
3. Source transaction or message type.
4. Source version.
5. Segment expectations.
6. Loop definitions.
7. Field mappings.
8. Transform rules.
9. Validation rules.
10. Output JSON shape.

### Mapping path format

Use a simple path syntax instead of a full programming language.

Example:

```yaml
maps:
  purchaseOrderNumber: "BEG[0].BEG03"
  orderDate: "BEG[0].BEG05 | date('yyyyMMdd')"
  buyer.name: "N1[N101='BY'].N102"
  buyer.id: "N1[N101='BY'].N104"
```

### Transform functions

MVP transform functions:

```text
string()
trim()
upper()
lower()
date(inputFormat)
number()
integer()
decimal()
bool()
split(separator)
join(separator)
default(value)
required()
```

### Partner overlays

Base schema:

```yaml
id: x12-850-basic
standard: x12
transaction: "850"
version: "004010"
```

Partner overlay:

```yaml
id: partner-acme-x12-850
extends: x12-850-basic
partner:
  name: Acme Retail
rules:
  - field: buyer.id
    required: true
  - field: shipTo.address.postalCode
    pattern: "^[0-9]{5}(-[0-9]{4})?$"
```

### Schema registry

The local schema registry should resolve schemas from:

1. Explicit path passed by CLI or API.
2. Project `./schemas`.
3. User `~/.edi-json/schemas`.
4. Built-in public-safe examples.

Resolution order should prefer explicit local files over global or built-in schemas.

## 15. CLI Engineering

### Recommended CLI library

For Go:

```text
Cobra: best ecosystem and familiar subcommand structure
Kong: cleaner struct-based CLI with less boilerplate
urfave/cli: acceptable, but less ideal for complex nested commands
```

Recommendation: **Kong for cleaner engineering**, or **Cobra if you want maximum familiarity**.

### CLI command tree

```text
edi-json
  translate
  validate
  detect
  serve
  schemas
    list
    validate
    install
  explain
  version
```

### CLI global flags

```text
--config string
--log-level string
--json-errors
--no-color
--quiet
--verbose
```

### Translate command

```bash
edi-json translate input.edi \
  --standard auto \
  --mode structural \
  --pretty \
  --output output.json
```

Flags:

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

### Validate command

```bash
edi-json validate input.edi --schema ./schemas/x12-850.yml
```

Flags:

```text
--standard auto|x12|edifact
--schema string
--schema-id string
--level syntax|schema|partner
--json
--strict
```

### Detect command

```bash
edi-json detect input.edi --json
```

Output:

```json
{
  "standard": "x12",
  "confidence": 0.99,
  "delimiters": {
    "element": "*",
    "segment": "~",
    "component": ">",
    "repetition": "^"
  }
}
```

### Serve command

```bash
edi-json serve --host 127.0.0.1 --port 8765
```

Flags:

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

### Exit codes

```text
0 success
1 parse or validation error
2 CLI usage error
3 schema or mapping error
4 file or permission error
5 internal error
6 security or unsafe config error
```

## 16. REST API Engineering

### Recommended API stack in Go

Use standard `net/http` for MVP.

Recommended structure:

```go
mux := http.NewServeMux()

mux.HandleFunc("GET /health", s.handleHealth)
mux.HandleFunc("GET /api/v1/version", s.handleVersion)
mux.HandleFunc("POST /api/v1/detect", s.handleDetect)
mux.HandleFunc("POST /api/v1/translate", s.handleTranslate)
mux.HandleFunc("POST /api/v1/validate", s.handleValidate)
mux.HandleFunc("GET /api/v1/schemas", s.handleSchemasList)
mux.HandleFunc("POST /api/v1/schemas/validate", s.handleSchemasValidate)
```

Use middleware wrappers for:

1. Request ID.
2. Logging.
3. Panic recovery.
4. Max body size.
5. Token auth when required.
6. CORS only when configured.

### API request and response envelope

All API responses should follow one envelope.

```go
type APIResponse[T any] struct {
    OK       bool        `json:"ok"`
    Result   T           `json:"result,omitempty"`
    Warnings []EDIWarning `json:"warnings,omitempty"`
    Errors   []EDIError   `json:"errors,omitempty"`
    Metadata Metadata     `json:"metadata,omitempty"`
}
```

### Translate request

```go
type TranslateRequest struct {
    Input      string            `json:"input"`
    Standard   string            `json:"standard"`
    Mode       string            `json:"mode"`
    SchemaID   string            `json:"schemaId,omitempty"`
    Schema     string            `json:"schema,omitempty"`
    Options    TranslateOptions  `json:"options"`
}
```

### File upload endpoint

MVP can skip multipart uploads and let the web UI read the file client-side, then send text to `/api/v1/translate`.

Add multipart later if needed.

### API security defaults

Default config:

```yaml
server:
  host: 127.0.0.1
  port: 8765
  requireTokenOutsideLocalhost: true
  maxBodyMb: 50
  cors:
    enabled: false
```

Rules:

1. If host is `127.0.0.1` or `localhost`, token is optional.
2. If host is `0.0.0.0`, token is required unless `--unsafe-no-token` is explicitly set.
3. If CORS is enabled, origins must be explicit.
4. Never default to `Access-Control-Allow-Origin: *`.

## 17. Web UI Engineering

### Recommended frontend

Use:

```text
React
TypeScript
Vite
Monaco Editor or CodeMirror
TanStack Query optional
No server-side rendering
```

### Why React/Vite

The web UI needs a good structured document viewer, side-by-side EDI/JSON inspection, collapsible segment trees, and editor panes. React is better suited for that than plain templates or HTMX.

### Build model

The frontend builds to static files:

```bash
cd web
npm install
npm run build
```

Generated output:

```text
web/dist/
```

Go embeds the files:

```go
//go:embed dist/*
var WebAssets embed.FS
```

The Go server serves them:

```go
fsys, _ := fs.Sub(WebAssets, "dist")
mux.Handle("/", http.FileServer(http.FS(fsys)))
```

### Web screens

1. Upload or paste
2. Parsed document viewer
3. Validation result viewer
4. Schema manager
5. Examples and playground
6. Settings

### Web UI state

The browser should hold unsaved EDI input in memory only.

Do not store EDI documents in localStorage by default.

Optional settings can be stored in localStorage:

```text
theme
last selected mode
editor font size
```

### Redaction mode

Add a UI toggle:

```text
Redact sensitive values before copy/export
```

Initial redaction patterns:

1. Email addresses
2. Phone numbers
3. ZIP/postal codes optional
4. Names optional
5. IDs optional
6. Long numeric identifiers optional

## 18. Configuration Engineering

Config loading order:

```text
1. Built-in defaults
2. Global config at ~/.edi-json/config.yml
3. Project config at ./edi-json.yml
4. Environment variables
5. CLI flags
```

Environment variables:

```text
EDI_JSON_CONFIG
EDI_JSON_HOST
EDI_JSON_PORT
EDI_JSON_TOKEN
EDI_JSON_SCHEMA_PATHS
EDI_JSON_STORE_HISTORY
EDI_JSON_MAX_BODY_MB
```

Config struct:

```go
type Config struct {
    Server      ServerConfig      `yaml:"server"`
    Translation TranslationConfig `yaml:"translation"`
    Schemas     SchemaConfig      `yaml:"schemas"`
    Privacy     PrivacyConfig     `yaml:"privacy"`
    Limits      LimitsConfig      `yaml:"limits"`
}
```

## 19. Optional Local Storage

MVP should not require storage.

Optional storage can use SQLite for:

1. Local parse history.
2. Schema registry index.
3. Recent files.
4. User settings.
5. Saved sample snippets.

Storage must be disabled by default.

Config:

```yaml
privacy:
  storeHistory: false
storage:
  path: ~/.edi-json/ediforge.db
```

Schema:

```sql
CREATE TABLE parse_history (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL,
  file_name TEXT,
  standard TEXT,
  document_type TEXT,
  mode TEXT,
  input_hash TEXT NOT NULL,
  result_json TEXT,
  warnings_json TEXT,
  errors_json TEXT
);
```

Do not store raw input unless explicitly enabled.

## 20. Logging and Observability

Use structured logs.

Recommended Go logging:

```text
log/slog
```

Log levels:

```text
debug
info
warn
error
```

Do not log raw EDI content by default.

Safe log fields:

```text
request_id
standard
document_type
mode
segment_count
transaction_count
parse_ms
error_code
```

Unsafe log fields:

```text
raw_edi
full_json_result
names
addresses
trading_partner_ids
control_numbers
```

Metrics can be added later behind a local-only endpoint.

## 21. Security Engineering

### Local server safety

Rules:

1. Bind to `127.0.0.1` by default.
2. Require token for non-localhost.
3. Warn loudly when binding to `0.0.0.0`.
4. Disable CORS by default.
5. Add request body limits.
6. Add file size limits.
7. Reject path traversal attempts.
8. Avoid writing temp files unless required.
9. Clean temp files.
10. Do not execute schema files as code.

### Schema safety

Mapping expressions must be declarative.

Do not allow:

1. Shell execution.
2. Embedded JavaScript.
3. Arbitrary Go templates.
4. Network calls from mappings.
5. File reads from mappings.
6. Dynamic plugin loading in MVP.

### Dependency safety

Recommended checks:

```bash
go test ./...
go vet ./...
govulncheck ./...
```

For frontend:

```bash
npm audit
npm run build
npm run lint
```

## 22. Performance Engineering

### Parser performance principles

1. Use streaming reads.
2. Avoid repeated string copies.
3. Avoid regex in hot parse loops.
4. Split segments manually where helpful.
5. Keep raw segment storage optional.
6. Keep offsets optional if they add overhead.
7. Use benchmark tests with realistic files.
8. Use golden tests to protect output stability.

### Performance targets

```text
1 MB EDI file: under 1 second
50 MB EDI file: no excessive memory growth
Memory target for normal files: under 250 MB
CLI cold start: under 150 ms where practical
REST translate overhead: under 25 ms beyond parse time
```

### Benchmarks

```go
func BenchmarkX12Parse1MB(b *testing.B) {}
func BenchmarkX12Parse50MB(b *testing.B) {}
func BenchmarkEDIFACTParse1MB(b *testing.B) {}
func BenchmarkSemanticMapping850(b *testing.B) {}
```

## 23. Testing Strategy

### Unit tests

Test:

1. X12 delimiter detection.
2. EDIFACT UNA parsing.
3. EDIFACT release character handling.
4. Segment tokenization.
5. Envelope parsing.
6. Control number validation.
7. Segment count validation.
8. Schema loading.
9. Mapping path resolution.
10. Transform functions.

### Golden tests

Golden test structure:

```text
testdata/
  x12/
    850-basic.edi
    850-basic.structural.json
    850-basic.semantic.json
  edifact/
    orders-basic.edi
    orders-basic.structural.json
    orders-basic.semantic.json
```

Golden test command:

```bash
go test ./internal/... -run Golden
```

Update golden files only with an explicit flag:

```bash
UPDATE_GOLDEN=1 go test ./internal/... -run Golden
```

### Fuzz tests

Add fuzz tests for:

1. X12 tokenizer.
2. EDIFACT tokenizer.
3. Delimiter detection.
4. Mapping path parser.

Example:

```go
func FuzzX12Tokenizer(f *testing.F) {
    f.Add("ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260427*1200*U*00401*000000001*0*T*>~")
    f.Fuzz(func(t *testing.T, input string) {
        _, _ = x12.Tokenize(strings.NewReader(input))
    })
}
```

### API tests

Test:

1. `/health`
2. `/api/v1/detect`
3. `/api/v1/translate`
4. `/api/v1/validate`
5. token required outside localhost
6. request body limit
7. invalid JSON request
8. malformed EDI response

### CLI tests

Use integration tests that call the built command.

Test:

1. translate file
2. translate stdin
3. output to file
4. validate success
5. validate failure
6. detect JSON output
7. serve starts and responds to health
8. invalid command returns usage error

## 24. Build and Release Engineering

### Local build

```bash
go build -o bin/edi-json ./cmd/edi-json
```

### Frontend build

```bash
cd web
npm install
npm run build
cd ..
go build -o bin/edi-json ./cmd/edi-json
```

### Dockerfile

```dockerfile
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/edi-json ./cmd/edi-json

FROM scratch
COPY --from=build /out/edi-json /edi-json
ENTRYPOINT ["/edi-json"]
```

If SQLite is enabled with CGO, use a small Alpine or distroless image instead of `scratch`, or choose a pure Go SQLite option.

### GoReleaser

Use GoReleaser for:

1. GitHub releases
2. Checksums
3. Docker images
4. Homebrew tap
5. Linux packages later

Example release targets:

```text
linux/amd64
linux/arm64
darwin/amd64
darwin/arm64
windows/amd64
```

## 25. Public Library API

Expose a small supported API for other Go projects.

Package:

```text
pkg/translator
```

Example use:

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/openedi/ediforge/pkg/translator"
)

func main() {
    svc := translator.New()
    result, err := svc.Translate(context.Background(), strings.NewReader("ISA*00*...~"), translator.Options{
        Standard: "auto",
        Mode: "structural",
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(result.JSON)
}
```

Keep the public API narrow. Internal packages can change faster.

## 26. Rust Alternative Architecture

If implementing in Rust, use:

```text
CLI: Clap
REST API: Axum
Async runtime: Tokio
JSON: serde and serde_json
YAML: serde_yaml
Errors: thiserror and anyhow
Static assets: rust-embed or include_dir
Logging: tracing and tracing-subscriber
Tests: cargo test, insta snapshots, cargo fuzz
Build: cargo-dist or cross
```

Rust repository structure:

```text
ediforge/
  Cargo.toml
  crates/
    openedi-core/
      src/
        detect/
        parse/
        model/
        validate/
        mapping/
        jsonout/
    openedi-cli/
      src/main.rs
    openedi-api/
      src/
    openedi-web/
      dist/
  schemas/
  testdata/
  web/
```

Rust parser crate should be independent from CLI and API crates.

Recommended Rust crate split:

```text
openedi-core: parser, validation, mapping, JSON output
openedi-cli: command-line app
openedi-api: local server
ediforge: final binary that combines CLI and server
```

Rust final recommendation:

Use Rust if you are willing to pay more implementation complexity for stronger parser internals and future WASM. Use Go if you want to ship and attract contributors faster.

## 27. Implementation Milestones

### Milestone 0: Repo foundation

Deliverables:

1. Go module initialized.
2. CLI skeleton.
3. API skeleton.
4. Web skeleton.
5. CI running tests.
6. Basic README.

Acceptance:

```bash
edi-json version
edi-json serve
curl http://127.0.0.1:8765/health
```

### Milestone 1: X12 structural parser

Deliverables:

1. X12 detection.
2. X12 delimiter extraction.
3. X12 tokenizer.
4. ISA/GS/ST parser.
5. Structural JSON output.
6. Syntax validation for control numbers and counts.
7. Golden tests.

Acceptance:

```bash
edi-json translate testdata/x12/850-basic.edi --mode structural --pretty
```

### Milestone 2: EDIFACT structural parser

Deliverables:

1. EDIFACT detection.
2. UNA parsing.
3. Release character handling.
4. UNB/UNH parser.
5. Structural JSON output.
6. Syntax validation for references and counts.
7. Golden tests.

Acceptance:

```bash
edi-json translate testdata/edifact/orders-basic.edi --mode structural --pretty
```

### Milestone 3: REST API

Deliverables:

1. `/health`
2. `/api/v1/version`
3. `/api/v1/detect`
4. `/api/v1/translate`
5. `/api/v1/validate`
6. Request size limits.
7. Safe binding defaults.

Acceptance:

```bash
curl -X POST http://127.0.0.1:8765/api/v1/translate \
  -H "Content-Type: application/json" \
  -d '{"input":"ISA*00*...~","standard":"auto","mode":"structural"}'
```

### Milestone 4: Local web UI

Deliverables:

1. Paste EDI input.
2. Upload local file.
3. Select translation mode.
4. Show JSON output.
5. Show errors and warnings.
6. Download JSON.
7. Embedded static build in Go binary.

Acceptance:

```bash
edi-json serve --open
```

### Milestone 5: Schema and semantic mapping

Deliverables:

1. YAML schema loader.
2. Schema validation.
3. Mapping path parser.
4. Basic transforms.
5. Semantic JSON output.
6. Partner overlay support.
7. Example schemas.

Acceptance:

```bash
edi-json translate testdata/x12/850-basic.edi \
  --mode semantic \
  --schema schemas/examples/x12-850-basic.yml
```

### Milestone 6: Packaging and release

Deliverables:

1. Docker image.
2. GitHub release binaries.
3. Checksums.
4. Homebrew tap.
5. Documentation site or docs folder.
6. Security policy.

Acceptance:

```bash
docker run --rm -p 8765:8765 openedi/ediforge serve
```

## 28. Agent Build Instructions

If this guide is handed to an AI coding agent, the agent should follow this order:

1. Create the repository structure exactly as specified.
2. Build the core model types first.
3. Build X12 detection and tokenizer.
4. Add X12 parser and golden tests.
5. Build EDIFACT detection and tokenizer.
6. Add EDIFACT parser and golden tests.
7. Create CLI commands against the core service.
8. Create REST handlers against the same core service.
9. Create the React web UI.
10. Embed the web UI into the Go binary.
11. Add schema loading.
12. Add semantic mapping.
13. Add packaging.
14. Add docs.

Rules for the coding agent:

1. Do not put parsing logic in the REST handlers.
2. Do not put parsing logic in the CLI commands.
3. Do not add cloud calls.
4. Do not store raw EDI unless a config flag explicitly enables it.
5. Do not bundle proprietary X12 guide content.
6. Do not invent segment labels unless they are in a schema or public-safe metadata.
7. Add tests before expanding parser behavior.
8. Preserve structural JSON output compatibility once golden tests exist.
9. Keep the MVP small.
10. Prefer clear error messages over clever abstractions.

## 29. Engineering Decisions to Lock Early

### Decision 1: Go vs Rust

Recommended: Go for v1.

### Decision 2: CLI library

Recommended: Kong for cleaner struct-based CLI, or Cobra if you prefer wider community familiarity.

### Decision 3: REST framework

Recommended: Go `net/http` for MVP.

Move to Chi only if middleware/routing complexity grows.

### Decision 4: Web UI

Recommended: React + TypeScript + Vite.

### Decision 5: Embedded UI

Recommended: yes. Ship one binary.

### Decision 6: Storage

Recommended: no required storage in MVP.

Optional SQLite later, disabled by default.

### Decision 7: Schema format

Recommended: YAML first, JSON also accepted.

### Decision 8: Mapping language

Recommended: simple declarative path syntax, not embedded code.

### Decision 9: License

Recommended based on goals:

```text
Apache-2.0: maximum industry adoption
AGPL-3.0: stronger protection against closed SaaS wrappers
MPL-2.0: middle ground
```

## 30. Initial Implementation Checklist

```text
[ ] Create repo
[ ] Add LICENSE, NOTICE, README, SECURITY, CONTRIBUTING
[ ] Initialize Go module
[ ] Add CLI skeleton
[ ] Add config loader
[ ] Add model types
[ ] Add detect package
[ ] Add X12 delimiter detection
[ ] Add X12 tokenizer
[ ] Add X12 parser
[ ] Add X12 golden fixtures
[ ] Add EDIFACT delimiter detection
[ ] Add EDIFACT tokenizer with release character support
[ ] Add EDIFACT parser
[ ] Add EDIFACT golden fixtures
[ ] Add structural JSON renderer
[ ] Add validation result format
[ ] Add REST server
[ ] Add React web UI
[ ] Embed web UI
[ ] Add schema loader
[ ] Add semantic mapper
[ ] Add example schemas
[ ] Add Dockerfile
[ ] Add GoReleaser config
[ ] Add docs
```

## 31. Practical v1 Dependency List

### Go dependencies

Recommended minimal set:

```text
github.com/alecthomas/kong or github.com/spf13/cobra
gopkg.in/yaml.v3
github.com/google/uuid
```

Optional later:

```text
github.com/go-chi/chi/v5
github.com/mattn/go-sqlite3
modernc.org/sqlite
github.com/golang-migrate/migrate/v4
```

### Frontend dependencies

Recommended:

```text
react
react-dom
vite
typescript
@vitejs/plugin-react
codemirror or monaco-editor
```

Optional:

```text
@tanstack/react-query
zod
react-json-view-lite
```

Keep frontend dependencies modest. This should feel like a local developer tool, not a SaaS dashboard.

## 32. Example Go HTTP Server Skeleton

```go
package api

import (
    "encoding/json"
    "net/http"

    "github.com/openedi/ediforge/internal/translate"
)

type Server struct {
    translator *translate.Service
    mux        *http.ServeMux
}

func NewServer(translator *translate.Service) *Server {
    s := &Server{
        translator: translator,
        mux:        http.NewServeMux(),
    }

    s.routes()
    return s
}

func (s *Server) routes() {
    s.mux.HandleFunc("GET /health", s.handleHealth)
    s.mux.HandleFunc("GET /api/v1/version", s.handleVersion)
    s.mux.HandleFunc("POST /api/v1/detect", s.handleDetect)
    s.mux.HandleFunc("POST /api/v1/translate", s.handleTranslate)
    s.mux.HandleFunc("POST /api/v1/validate", s.handleValidate)
}

func (s *Server) Handler() http.Handler {
    return s.mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]any{
        "ok": true,
        "status": "healthy",
    })
}

func writeJSON(w http.ResponseWriter, status int, value any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(value)
}
```

## 33. Example Go CLI Skeleton With Kong

```go
package cli

import (
    "context"

    "github.com/alecthomas/kong"
)

type CLI struct {
    Translate TranslateCmd `cmd:"" help:"Translate EDI to JSON."`
    Validate  ValidateCmd  `cmd:"" help:"Validate an EDI file."`
    Detect    DetectCmd    `cmd:"" help:"Detect EDI standard and delimiters."`
    Serve     ServeCmd     `cmd:"" help:"Run local REST API and web UI."`
    Version   VersionCmd   `cmd:"" help:"Print version."`
}

func Execute(ctx context.Context, args []string) error {
    cli := CLI{}
    parser, err := kong.New(&cli)
    if err != nil {
        return err
    }

    kctx, err := parser.Parse(args[1:])
    if err != nil {
        return err
    }

    return kctx.Run(ctx)
}
```

## 34. Example Mapping Schema

```yaml
id: x12-850-basic
standard: x12
transaction: "850"
version: "004010"
name: Purchase Order Basic Mapping
license: CC-BY-4.0
source: community

segments:
  - tag: BEG
    required: true
    max: 1

  - tag: N1
    loop: parties
    max: 200

output:
  documentType: purchase_order

maps:
  purchaseOrderNumber: "BEG[0].BEG03 | required"
  orderDate: "BEG[0].BEG05 | date('yyyyMMdd')"
  buyer.name: "N1[N101='BY'].N102"
  buyer.id: "N1[N101='BY'].N104"
  shipTo.name: "N1[N101='ST'].N102"
  shipTo.id: "N1[N101='ST'].N104"
```

## 35. Summary Recommendation

Build the first version in Go.

Use Go for:

1. Parser core
2. CLI
3. REST API
4. Embedded static web UI
5. Packaging

Use React + TypeScript + Vite for the local interface.

Keep the parser core separate from every interface. The long-term value of this project will come from a reliable local translation engine that can be used by humans, APIs, CI jobs, and eventually agents.

The v1 engineering goal should be simple:

```text
One binary.
No cloud dependency.
X12 and EDIFACT to structural JSON.
CLI, REST API, and local web UI.
Schema-based semantic mapping.
Safe defaults.
Fast enough for real logistics files.
```
