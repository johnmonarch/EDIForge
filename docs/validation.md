# Validation

EDIForge validation should be layered so syntax checks work without a schema and deeper checks activate when schemas or partner overlays are supplied.

## Level 1: Syntax

Always available.

Checks include:

- Separator detection.
- Envelope open/close pairs.
- Control number consistency.
- Transaction or message segment count.
- Empty required envelope values.
- EDIFACT release character handling.
- Unexpected segments outside transactions or messages.

## Level 2: Schema

Available when a schema is supplied.

Checks include:

- Required segments.
- Segment order.
- Loop structure.
- Element required/optional status.
- Maximum repeats.
- Date, time, number, and decimal formats.
- Basic code lists when legally bundled or user-provided.

## Level 3: Partner

Available when partner overlays are supplied.

Checks include:

- Partner-required fields.
- Partner-specific allowed codes.
- Conditional requirements.
- Business rules.
- Custom warnings and severity levels.

## Result Shape

```json
{
  "ok": false,
  "errors": [
    {
      "severity": "error",
      "code": "EDIFACT_CONTROL_REFERENCE_MISMATCH",
      "message": "UNB control reference does not match UNZ control reference.",
      "standard": "edifact",
      "segment": "UNZ",
      "segmentPosition": 8,
      "element": "UNZ02",
      "hint": "Check whether the interchange is truncated or concatenated incorrectly."
    }
  ],
  "warnings": [],
  "metadata": {
    "segments": 8
  }
}
```

