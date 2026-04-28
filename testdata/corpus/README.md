# Public-safe EDI corpus

`catalog.json` lists reusable validation fixtures and the behavior expected from
the translate service. Fixture paths are relative to the catalog file and may
point at existing public-safe samples elsewhere under `testdata`.

Add cases by appending a fixture object with:

- `name`: stable test name
- `path`: fixture path relative to `testdata/corpus/catalog.json`
- `category`: `valid`, `malformed`, or `partial`
- `standard`: `x12` or `edifact`
- `documentType`: expected transaction/message type when recoverable
- `ok`: expected translate result status
- `errors` / `warnings`: expected diagnostic codes

