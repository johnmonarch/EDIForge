# PRD: Local-First EDI-to-JSON Translator

## 1. Product Name

Working name: **EDIForge**

One-line description: A local-first, open-source EDI translator that converts X12 and UN/EDIFACT files into clean, inspectable JSON through a CLI, built-in REST API, and local web interface.

## 2. Background

EDI remains deeply embedded in logistics, freight, warehousing, retail, manufacturing, health care, and trade. Many existing tools are commercial, cloud-first, complex to deploy, or difficult for developers to use in local workflows. Modern teams increasingly want EDI data in JSON so it can be tested, searched, validated, routed, mapped, and fed into APIs, TMS/WMS/ERP systems, analytics pipelines, or AI-assisted operations.

This project will provide a FOSS, local-first translator that focuses on developer usability, privacy, repeatability, and practical logistics workflows.

## 3. Problem Statement

Logistics teams often receive EDI files but lack a simple, local, developer-friendly way to:

- Convert X12 or EDIFACT into JSON without sending files to a cloud service.
- Inspect envelope structure, transaction/message contents, delimiters, loops, and parsing errors.
- Validate syntax and trading-partner-specific rules.
- Expose translation through a CLI, REST endpoint, or local browser UI.
- Version and share mappings as code.

The result is slow onboarding, fragile one-off scripts, dependence on expensive middleware, and poor visibility into what an EDI file actually contains.

## 4. Goals

### Primary goals

1. Provide a single local-first binary or package that can translate X12 and EDIFACT to JSON.
2. Offer three interfaces from the same core engine:
   - CLI
   - Built-in REST API
   - Local web interface
3. Support both raw structural JSON and schema-driven semantic JSON.
4. Make onboarding easy enough for developers who know JSON but are not EDI experts.
5. Keep EDI data private by default with no cloud dependency.
6. Make schemas, mappings, validation rules, and test files easy to version in Git.
7. Build the project in a way that respects X12 licensing constraints and avoids bundling proprietary implementation guide content.

### Secondary goals

1. Provide a growing set of logistics-focused starter mappings.
2. Support batch processing and streaming large files.
3. Provide useful errors that explain where parsing failed.
4. Enable downstream integrations with TMS, WMS, ERP, customs, warehouse automation, carrier, broker, and 3PL systems.
5. Provide a stable JSON output contract that can be consumed by agents, APIs, and automated tests.

## 5. Non-Goals

For the first release, the product will not:

1. Replace a full VAN, AS2 gateway, managed EDI platform, or enterprise integration broker.
2. Provide cloud hosting as part of the core product.
3. Bundle copyrighted X12 implementation guides or proprietary trading partner maps.
4. Guarantee compliance with every industry-specific implementation guide out of the box.
5. Attempt full bidirectional JSON-to-EDI generation in MVP.
6. Manage trading partner communications, mailbox polling, SFTP, AS2, OAuth, or certificates in MVP.
7. Store customer EDI data in any external service.

## 6. Target Users

### Primary users

- Logistics software developers
- 3PL integration teams
- TMS/WMS/ERP implementation teams
- EDI analysts who need developer-friendly tools
- Freight brokers and shippers building internal integrations
- Open-source developers working on supply chain tooling

### Secondary users

- Data engineers building pipelines from EDI feeds
- QA teams testing trading partner files
- Support teams debugging failed EDI documents
- Consultants implementing customer-specific EDI maps
- AI/agent developers who need structured logistics messages

## 7. Key Product Principles

1. **Local-first:** All translation runs locally by default.
2. **One engine, multiple interfaces:** CLI, REST, and web UI must use the same translation core.
3. **Syntax first, semantics second:** Parse every valid EDI document into structural JSON even when no semantic map exists.
4. **Schema as code:** Mappings and validation rules should be versionable, diffable, and portable.
5. **Transparent errors:** Parsing failures should include segment position, line/byte offset, envelope context, and likely cause.
6. **No proprietary lock-in:** Output should be plain JSON. Schemas should be plain YAML or JSON.
7. **Privacy by design:** No telemetry, no uploads, no external dependency unless the user explicitly enables one.
8. **FOSS-friendly:** The core should be useful without paid services.

## 8. Supported Standards

### MVP required support

- X12 interchange parsing
- EDIFACT interchange parsing
- Automatic standard detection where possible
- Delimiter detection
- Segment parsing
- Composite element parsing
- Envelope extraction
- Transaction/message grouping
- Structural JSON output
- Syntax-level validation

### X12 MVP scope

Core support:

- ISA/IEA envelope
- GS/GE functional group
- ST/SE transaction set
- Segment and element parsing
- Repetition separators where applicable
- Control number consistency checks
- Basic transaction set identification

Initial logistics-focused transaction set targets:

- 204 Motor Carrier Load Tender
- 210 Motor Carrier Freight Details and Invoice
- 214 Transportation Carrier Shipment Status Message
- 810 Invoice
- 850 Purchase Order
- 855 Purchase Order Acknowledgment
- 856 Advance Ship Notice
- 940 Warehouse Shipping Order
- 945 Warehouse Shipping Advice

### EDIFACT MVP scope

Core support:

- UNA service string advice
- UNB/UNZ interchange
- UNG/UNE group, optional
- UNH/UNT message
- Segment, element, and composite parsing
- Release character handling
- Control reference consistency checks
- Basic message type identification

Initial logistics-focused message targets:

- ORDERS Purchase Order
- ORDRSP Purchase Order Response
- DESADV Despatch Advice
- INVOIC Invoice
- IFTMIN Instruction message
- IFTMCS Transport instruction
- IFTSTA International multimodal status report
- CUSDEC Customs declaration, stretch target
- CUSRES Customs response, stretch target

## 9. Translation Modes

The product should support three output modes.

### Mode 1: Structural JSON

This mode preserves the EDI structure without requiring a schema or mapping.

Use cases:

- Inspection
- Debugging
- Archiving
- Data discovery
- Test fixture creation
- Agent-readable parsing

Example shape:

```json
{
  "standard": "x12",
  "version": "004010",
  "interchange": {
    "senderId": "SENDER",
    "receiverId": "RECEIVER",
    "controlNumber": "000000905"
  },
  "groups": [
    {
      "functionalIdentifierCode": "PO",
      "transactions": [
        {
          "transactionSetCode": "850",
          "controlNumber": "0001",
          "segments": [
            {
              "tag": "BEG",
              "position": 4,
              "elements": ["00", "SA", "123456789"]
            }
          ]
        }
      ]
    }
  ],
  "errors": [],
  "warnings": []
}
```

### Mode 2: Annotated JSON

This mode adds friendly labels when the parser can legally and confidently identify segment and element names from open metadata, user-provided maps, or community-maintained schemas.

Use cases:

- Developer onboarding
- Debugging
- Human inspection
- Building maps

Example shape:

```json
{
  "transactionSetCode": "850",
  "transactionSetName": "Purchase Order",
  "segments": [
    {
      "tag": "BEG",
      "name": "Beginning Segment for Purchase Order",
      "elements": [
        { "id": "BEG01", "name": "Transaction Set Purpose Code", "value": "00" },
        { "id": "BEG02", "name": "Purchase Order Type Code", "value": "SA" }
      ]
    }
  ]
}
```

### Mode 3: Semantic JSON

This mode transforms EDI into a business-friendly JSON model using a schema/mapping pack.

Use cases:

- API ingestion
- TMS/WMS/ERP integration
- Analytics
- AI/agent workflows
- Business process automation

Example shape:

```json
{
  "documentType": "purchase_order",
  "standard": "x12",
  "sourceType": "850",
  "purchaseOrderNumber": "123456789",
  "orderDate": "2026-04-27",
  "buyer": {
    "name": "Example Retailer",
    "id": "BUYER01"
  },
  "shipTo": {
    "name": "Example DC",
    "address": {
      "line1": "100 Warehouse Road",
      "city": "Greenville",
      "state": "SC",
      "postalCode": "29601",
      "country": "US"
    }
  },
  "lineItems": []
}
```

## 10. Functional Requirements

### 10.1 CLI

The CLI should be the fastest path to value.

Required commands:

```bash
edi-json translate input.edi
edi-json translate input.edi --standard x12 --pretty
edi-json translate input.edi --output output.json
edi-json translate input.edi --mode structural
edi-json translate input.edi --mode annotated
edi-json translate input.edi --mode semantic --schema ./schemas/x12-850.yml
edi-json validate input.edi
edi-json detect input.edi
edi-json serve
edi-json schemas list
edi-json schemas validate ./schemas/x12-850.yml
edi-json explain input.edi --segment BEG
```

Required CLI behavior:

- Reads from file path or stdin.
- Writes to stdout by default.
- Supports pretty and compact JSON.
- Supports batch folder input.
- Supports exit codes suitable for CI.
- Provides parse errors in JSON with `--json-errors`.
- Supports `--no-store` to avoid writing anything to local history.
- Supports config at project level and user level.

Recommended exit codes:

- `0`: Success
- `1`: Parse or validation error
- `2`: Invalid CLI usage
- `3`: Schema/mapping error
- `4`: File or permission error
- `5`: Internal error

### 10.2 REST API

The local REST API should run from the same binary.

Command:

```bash
edi-json serve --host 127.0.0.1 --port 8765
```

Default behavior:

- Bind to `127.0.0.1` only.
- Require explicit flag to bind to LAN.
- No authentication on localhost by default.
- Require API token if bound outside localhost.
- No outbound network calls by default.

Required endpoints:

```http
GET /health
GET /api/v1/version
POST /api/v1/detect
POST /api/v1/translate
POST /api/v1/validate
GET /api/v1/schemas
POST /api/v1/schemas/validate
POST /api/v1/explain
```

Optional later endpoints:

```http
POST /api/v1/batch/translate
GET /api/v1/jobs/{jobId}
GET /api/v1/history
DELETE /api/v1/history
POST /api/v1/mappings/test
```

Example translate request:

```json
{
  "input": "ISA*00*...~GS*PO*...~ST*850*0001~...",
  "standard": "auto",
  "mode": "semantic",
  "schemaId": "x12-850-basic",
  "options": {
    "pretty": true,
    "includeRawSegments": false,
    "includeEnvelope": true
  }
}
```

Example translate response:

```json
{
  "ok": true,
  "standard": "x12",
  "documentType": "850",
  "mode": "semantic",
  "result": {},
  "warnings": [],
  "errors": [],
  "metadata": {
    "parseMs": 12,
    "segments": 143,
    "transactions": 1
  }
}
```

### 10.3 Local Web Interface

The web interface should be included in the same local server.

Default URL:

```text
http://127.0.0.1:8765
```

Required screens:

1. **Home / Upload**
   - Drag-and-drop EDI file
   - Paste EDI text
   - Select mode: structural, annotated, semantic
   - Select schema/mapping pack

2. **Parsed Document Viewer**
   - Envelope summary
   - Group summary
   - Transaction/message summary
   - Segment tree
   - JSON output panel
   - Errors/warnings panel
   - Copy/download JSON

3. **Validation View**
   - Segment count checks
   - Control number checks
   - Missing required segment warnings when schema is used
   - Unknown segment warnings
   - Invalid date/number/code warnings when schema supports it

4. **Schema/Mappings View**
   - List installed schemas
   - Import local schema file
   - Validate schema file
   - View mapping examples

5. **Examples / Playground**
   - Bundled public-safe examples
   - No proprietary or copyrighted guide content
   - Clear warning that real partner guides should be user-provided

Nice-to-have UI features:

- Side-by-side EDI and JSON view
- Click a segment and highlight corresponding JSON fields
- Search segments by tag
- Collapse loops
- Export parse report
- Dark mode
- Redaction mode for sharing examples

### 10.4 Schema and Mapping System

The schema system is the key to making the product useful without violating proprietary guide restrictions.

Required capabilities:

- User-provided schemas in YAML or JSON.
- Community schemas that only include legally safe metadata.
- Transaction/message specific mapping files.
- Partner-specific overlays.
- Rule inheritance.
- Validation rule definitions.
- Output field mapping.
- Code list hooks without bundling restricted code lists unless legally allowed.

Schema hierarchy:

```text
base standard parser
  -> public/community transaction schema
    -> industry profile
      -> trading partner overlay
        -> customer local override
```

Example schema file:

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
    maps:
      BEG03: purchaseOrderNumber
      BEG05: orderDate
  - tag: N1
    loop: parties
    maps:
      N101: qualifier
      N102: name
      N104: id
output:
  documentType: purchase_order
  fields:
    purchaseOrderNumber:
      type: string
      required: true
    orderDate:
      type: date
      format: yyyyMMdd
```

### 10.5 Validation

Validation should have levels.

#### Level 1: Syntax validation

Always available.

Checks:

- Valid separators
- Valid envelope structure
- Segment terminators
- Element counts where known
- Control numbers match
- Transaction/message segment counts match
- EDIFACT release character handling
- Empty required envelope values

#### Level 2: Schema validation

Available when a schema is supplied.

Checks:

- Required segments
- Segment order
- Loop structure
- Element required/optional status
- Max repeats
- Data type checks
- Date/time format checks
- Numeric checks
- Basic code list checks when provided

#### Level 3: Partner validation

Available when partner overlays are supplied.

Checks:

- Partner-specific required fields
- Partner-specific allowed codes
- Conditional requirements
- Business rules
- Custom warnings
- Custom severity levels

Error object shape:

```json
{
  "severity": "error",
  "code": "X12_CONTROL_NUMBER_MISMATCH",
  "message": "ST02 does not match SE02.",
  "standard": "x12",
  "segment": "SE",
  "segmentPosition": 42,
  "element": "SE02",
  "byteOffset": 1844,
  "hint": "Check whether the transaction was truncated or concatenated incorrectly."
}
```

## 11. Technical Architecture

### Recommended implementation approach

Use a modular architecture with a single core engine and multiple adapters.

Recommended language: **Go**

Rationale:

- Produces a single static binary for CLI, REST, and web UI.
- Fast startup and low memory use.
- Good fit for streaming parsers.
- Easy Docker distribution.
- Easy cross-platform releases for Linux, macOS, and Windows.
- Familiar to infrastructure and logistics integration teams.

Acceptable alternative: Java/Kotlin if the project chooses to build on existing EDI libraries like StAEDI or Smooks. This may reduce parser complexity but makes single-binary local distribution harder.

### Core modules

```text
/cmd
  /edi-json              CLI entrypoint
/internal
  /detect                Standard and delimiter detection
  /parse/x12             X12 tokenizer and parser
  /parse/edifact         EDIFACT tokenizer and parser
  /model                 Internal document model
  /jsonout               Structural and annotated JSON output
  /schema                Schema loading and validation
  /mapping               Semantic JSON mapping engine
  /validate              Syntax and schema validation
  /api                   REST API handlers
  /web                   Embedded local web UI
  /config                Config loading
  /storage               Optional local SQLite history/config
  /redact                Redaction utilities
/pkg
  /translator            Public library API
/schemas
  /examples              Public-safe examples only
/testdata
  /x12
  /edifact
```

### Data flow

```text
Input file/text
  -> detect standard and delimiters
  -> tokenize stream
  -> parse envelopes/groups/messages
  -> build internal event stream or document tree
  -> validate syntax
  -> apply schema, if present
  -> map to selected JSON mode
  -> return JSON plus warnings/errors/metadata
```

### Parser design

The parser should support both:

1. **Streaming mode** for large files.
2. **Tree mode** for UI inspection and semantic mapping.

The internal event stream should expose events such as:

```text
StartInterchange
StartGroup
StartTransactionOrMessage
Segment
Element
Composite
EndTransactionOrMessage
EndGroup
EndInterchange
Error
```

## 12. Security and Privacy Requirements

1. No external network calls by default.
2. Server binds to localhost by default.
3. Explicit `--host 0.0.0.0` or config required for LAN access.
4. API token required for non-localhost binding.
5. CORS disabled by default.
6. Optional max file size limit.
7. Optional request body size limit.
8. Optional history disabled by default.
9. Redaction utility for sensitive data.
10. No telemetry unless a user explicitly opts in.
11. CLI should never upload sample files for bug reports automatically.
12. Temporary files should be cleaned up after processing.

## 13. Packaging and Distribution

Required:

- GitHub releases
- Homebrew tap
- Docker image
- Linux amd64/arm64 binaries
- macOS amd64/arm64 binaries
- Windows amd64 binary
- Install script
- Checksums and signatures

Example install:

```bash
curl -fsSL https://example.org/install.sh | sh
```

Docker:

```bash
docker run --rm -p 8765:8765 -v "$PWD:/work" ediforge/edi-json serve
```

## 14. Configuration

Global config:

```text
~/.edi-json/config.yml
```

Project config:

```text
./edi-json.yml
```

Example:

```yaml
server:
  host: 127.0.0.1
  port: 8765
  requireTokenOutsideLocalhost: true
translation:
  defaultMode: structural
  includeEnvelope: true
  includeRawSegments: false
schemas:
  paths:
    - ./schemas
privacy:
  storeHistory: false
  telemetry: false
limits:
  maxFileSizeMb: 50
```

## 15. Open Source and Licensing Strategy

Recommended license depends on project goals.

### Option A: Apache-2.0

Best for adoption by logistics companies, brokers, 3PLs, WMS/TMS vendors, and enterprise developers.

Pros:

- Business-friendly
- Compatible with many enterprise policies
- Encourages broad adoption
- Includes patent grant

Cons:

- Allows SaaS wrappers and proprietary forks
- Does not force contribution of modifications

### Option B: AGPL-3.0

Best if the goal is to prevent companies from creating hosted SaaS wrappers without sharing source code changes.

Pros:

- Strong copyleft for network use
- Better protection against closed SaaS clones
- Keeps improvements more likely to return to the commons

Cons:

- Some enterprises avoid AGPL
- May reduce commercial adoption

### Option C: MPL-2.0

Middle ground.

Pros:

- File-level copyleft
- More business-friendly than AGPL
- More protective than Apache-2.0

Cons:

- Does not prevent SaaS wrappers as strongly as AGPL

Recommendation:

- Use **Apache-2.0** if the primary goal is industry adoption.
- Use **AGPL-3.0** if the primary goal is preventing closed SaaS wrappers.
- Use **MPL-2.0** if the goal is a practical middle ground.

For attribution, include:

- NOTICE file
- CITATION.cff
- README attribution request
- CLI `about` command
- Optional generated metadata field: `translatedBy: EDIForge`

Do not rely on the license alone for marketing attribution unless using a license that legally supports the intended requirement.

## 16. Standards and IP Considerations

X12 standards and implementation guides are copyrighted. The project should not bundle proprietary X12 standards text, paid guide content, or trading-partner implementation guides unless the project has explicit rights.

Required policy:

1. The parser may support generic syntax and user-provided schemas.
2. The project may provide public-safe sample schemas and examples.
3. The project must document that users are responsible for ensuring they have rights to any X12 guides or partner maps they import.
4. Do not train, generate, or distribute derivative X12 guide content from restricted materials.
5. EDIFACT support may use publicly available UNECE directories where license terms permit.
6. Every bundled schema or mapping must include source and license metadata.

## 17. MVP Scope

### MVP must include

1. CLI translation from X12 to structural JSON.
2. CLI translation from EDIFACT to structural JSON.
3. Automatic delimiter detection for common X12 and EDIFACT files.
4. REST API with `/translate`, `/validate`, `/detect`, and `/health`.
5. Local web UI for paste/upload and JSON output.
6. Syntax validation for envelopes and control numbers.
7. Schema file loading for semantic mapping.
8. At least one public-safe X12 sample map.
9. At least one public-safe EDIFACT sample map.
10. Cross-platform binary builds.
11. Docker image.
12. Documentation with examples.

### MVP should not include

1. Managed file exchange.
2. AS2/SFTP connectors.
3. Cloud accounts.
4. Complex visual mapping designer.
5. Full JSON-to-EDI generation.
6. Full bundled commercial guide support.

## 18. Future Roadmap

### Phase 1: Parser and CLI

- X12 tokenizer
- EDIFACT tokenizer
- Structural JSON output
- Validation errors
- CLI commands
- Test fixtures

### Phase 2: REST API and Web UI

- Local server
- Translate endpoint
- Validate endpoint
- File upload UI
- Side-by-side EDI/JSON viewer
- Download JSON

### Phase 3: Schema and Semantic Mapping

- YAML schema loader
- Mapping engine
- Partner overlays
- Annotated JSON
- Semantic JSON
- Schema validator

### Phase 4: Logistics Starter Packs

- X12 204, 210, 214, 850, 856, 940, 945 starter maps
- EDIFACT ORDERS, DESADV, INVOIC, IFTMIN, IFTSTA starter maps
- Public-safe examples
- Community contribution guidelines

### Phase 5: Advanced Features

- Visual mapping helper
- Batch job queue
- Local SQLite history
- Redaction tools
- Golden test generator
- JSON Schema generation
- OpenAPI schema generation
- WASM library for browser-only parsing
- Optional JSON-to-EDI generation
- Optional SFTP/AS2 plugins

## 19. Success Metrics

### Developer adoption

- GitHub stars
- Forks
- Contributors
- Downloads
- Docker pulls
- Homebrew installs

### Product usage

- CLI translations completed locally
- REST API calls locally
- Schema files created
- Community mappings contributed
- Issue resolution speed

### Quality

- Parser test coverage
- Golden fixture coverage
- Number of standards/messages covered
- Parse speed on large files
- Memory usage on large files
- Validation accuracy
- Number of reported parsing edge cases resolved

### Community

- Number of accepted mapping packs
- Number of logistics companies using the tool
- Number of public integrations
- Number of docs/examples contributed

## 20. Acceptance Criteria

### CLI acceptance criteria

- Given a valid X12 file, `edi-json translate file.edi` outputs valid JSON.
- Given a valid EDIFACT file, `edi-json translate file.edi` outputs valid JSON.
- Given malformed EDI, CLI exits non-zero and returns useful errors.
- Given `--output`, JSON is written to the requested file.
- Given stdin input, CLI can translate without a file path.
- Given `--mode semantic --schema`, output follows the schema mapping.

### REST API acceptance criteria

- `GET /health` returns healthy status.
- `POST /api/v1/detect` identifies likely standard and delimiters.
- `POST /api/v1/translate` returns structural JSON for X12 and EDIFACT.
- `POST /api/v1/validate` returns warnings and errors without translation.
- Server binds to localhost by default.
- Non-localhost binding requires explicit config and token.

### Web UI acceptance criteria

- User can paste EDI and see parsed JSON.
- User can upload EDI and download JSON.
- User can switch translation modes.
- User can view errors and warnings.
- User can inspect envelope and segment tree.

### Mapping acceptance criteria

- User can load a schema file.
- Invalid schema returns understandable errors.
- Mapping can create business-friendly JSON fields.
- Partner overlay can override base schema behavior.

## 21. Example User Stories

1. As a logistics developer, I want to convert an X12 214 into JSON so I can ingest carrier status updates into my TMS.
2. As a 3PL integration analyst, I want to inspect an X12 940 locally so I can debug a warehouse shipping order without uploading customer data.
3. As a freight broker, I want to expose a local API so internal tools can translate EDI files into JSON during testing.
4. As a WMS vendor, I want to version customer-specific mappings in Git so onboarding changes are reviewable.
5. As an open-source contributor, I want to add a public-safe starter map for EDIFACT DESADV so other users can build from it.
6. As a QA engineer, I want non-zero exit codes and JSON errors so I can use the tool in CI.

## 22. Example README Positioning

EDIForge is a local-first, open-source EDI-to-JSON translator for logistics developers. It converts X12 and EDIFACT files into clean JSON through a CLI, REST API, and local web interface. It is built for teams that need to inspect, validate, test, and integrate EDI without sending sensitive shipment, order, invoice, or warehouse data to a cloud service.

## 23. Documentation Requirements

Required docs:

- Quickstart
- CLI reference
- REST API reference
- Web UI guide
- X12 parsing guide
- EDIFACT parsing guide
- Schema/mapping guide
- Partner overlay guide
- Validation guide
- Privacy and security guide
- Standards/IP policy
- Contribution guide
- Mapping pack contribution guide
- Test fixture guide
- Docker deployment guide

## 24. Testing Strategy

Required test categories:

1. Unit tests for delimiter detection.
2. Unit tests for tokenization.
3. Unit tests for X12 envelope parsing.
4. Unit tests for EDIFACT envelope parsing.
5. Golden file tests for structural JSON output.
6. Golden file tests for semantic mapping output.
7. Validation error tests.
8. Large file streaming tests.
9. CLI integration tests.
10. REST API integration tests.
11. Web UI smoke tests.
12. Fuzz tests for malformed EDI.
13. Security tests for file upload and local server settings.

## 25. Performance Requirements

MVP targets:

- Translate a 1 MB EDI file in under 1 second on a modern laptop.
- Translate a 50 MB EDI file without loading the entire file multiple times.
- Keep memory usage below 250 MB for typical batch files.
- Support streaming parse mode for files larger than configured memory limits.
- Return first useful error as early as possible.

## 26. Open Questions

1. Should the first implementation be pure Go, or should it wrap an existing Java parser for speed to MVP?
2. Should the project use AGPL-3.0 to discourage SaaS wrappers, or Apache-2.0 to maximize adoption?
3. Which transaction/message types should be the first official logistics starter packs?
4. Should the web UI be embedded as static assets in the binary or served from a separate dev server during development only?
5. Should schema packs be stored in a separate repository?
6. Should JSON Schema output be included in MVP or Phase 3?
7. Should the tool eventually support JSON-to-EDI generation?
8. Should the project include an optional plugin interface for AS2/SFTP later?

## 27. Recommended MVP Build Order

1. Define internal EDI document model.
2. Build X12 delimiter detection and tokenizer.
3. Build EDIFACT delimiter detection and tokenizer.
4. Emit structural JSON.
5. Add envelope validation.
6. Add CLI.
7. Add golden tests.
8. Add REST API.
9. Add local web UI.
10. Add schema loader.
11. Add semantic mapping engine.
12. Add starter maps.
13. Package releases and Docker image.
14. Publish documentation.

## 28. Suggested Repository Structure

```text
ediforge/
  README.md
  LICENSE
  NOTICE
  CITATION.cff
  CONTRIBUTING.md
  SECURITY.md
  docs/
  cmd/
    edi-json/
  internal/
    api/
    config/
    detect/
    jsonout/
    mapping/
    model/
    parse/
      x12/
      edifact/
    redact/
    schema/
    storage/
    validate/
    web/
  pkg/
    translator/
  schemas/
    examples/
    community/
  testdata/
    x12/
    edifact/
  web/
    src/
    dist/
  docker/
  scripts/
```

## 29. Suggested Roadmap Labels

- `parser:x12`
- `parser:edifact`
- `cli`
- `api`
- `web-ui`
- `schema`
- `mapping`
- `validation`
- `docs`
- `good-first-issue`
- `starter-pack`
- `security`
- `performance`
- `licensing`

## 30. Summary

EDIForge should be a practical, local-first bridge between legacy EDI and modern JSON-based logistics systems. The winning product angle is not just “free EDI translation.” It is developer-friendly, private, inspectable, schema-as-code EDI translation for the logistics industry.

The MVP should stay focused: parse X12 and EDIFACT, output structural JSON, expose CLI/API/web surfaces, and support user-provided schemas for semantic mapping. That creates immediate utility while leaving room for community-maintained logistics starter packs, advanced validation, and future bidirectional generation.
