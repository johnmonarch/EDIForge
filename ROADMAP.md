# Roadmap

This roadmap is intentionally short and practical. Priorities may change as parser correctness, security defaults, and release quality needs become clearer.

## Near Term

- Keep X12 and UN/EDIFACT structural translation reliable across CLI, REST API, and web UI.
- Improve validation messages without exposing raw EDI or sensitive identifiers in logs.
- Expand public-safe example schemas and mappings for common logistics and commerce workflows.
- Keep build, release, and container artifacts repeatable.

## Later

- Broaden annotated and semantic output coverage through declarative schemas and mappings.
- Improve local web workflows for inspection, validation, and JSON export.
- Add more integration examples for CI, local automation, and private infrastructure.

## Non-Goals

- Bundling paid standards text, proprietary trading-partner maps, or restricted code lists.
- Adding telemetry or normal-workflow outbound network calls.
- Executing code from schemas or mappings.
