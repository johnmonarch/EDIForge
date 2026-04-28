# REST API Reference

The REST API runs from the same local binary as the CLI and web UI.

## Defaults

- Default bind address: `127.0.0.1`
- Default port: `8765`
- No authentication required on localhost
- API token required for non-localhost binding
- CORS disabled by default
- No outbound network calls by default

## Endpoints

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

## Response Envelope

```json
{
  "ok": true,
  "result": {},
  "warnings": [],
  "errors": [],
  "metadata": {}
}
```

## Translate Request

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

## Validate Request

```json
{
  "input": "UNB+UNOC:3+SENDER+RECEIVER+260427:1200+1'UNH+1+ORDERS:D:96A:UN'UNT+2+1'UNZ+1+1'",
  "standard": "auto",
  "level": "syntax",
  "schemaId": "edifact-orders-basic"
}
```

## Detect Request

```json
{
  "input": "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260427*1200*U*00401*000000001*0*T*>~"
}
```

## Error Shape

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

