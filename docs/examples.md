# Examples

The files in `schemas/examples/` are intentionally small public-safe starter schemas.

## X12 Starter Packs

- `x12-850-basic.json`: purchase order.
- `x12-810-basic.json`: invoice.
- `x12-856-basic.json`: advance ship notice / ship notice manifest.
- `x12-214-basic.json`: transportation carrier shipment status.
- `x12-990-basic.json`: response to load tender.
- `x12-997-basic.json`: functional acknowledgment.
- `x12-999-basic.json`: implementation acknowledgment.

## EDIFACT Starter Packs

- `edifact-orders-basic.json`: purchase order.
- `edifact-ordrsp-basic.json`: purchase order response.
- `edifact-desadv-basic.json`: despatch advice.
- `edifact-invoic-basic.json`: invoice.

Each starter pack includes an `exampleInput` field and a matching fixture under `testdata/`.

## Contributor Rule

Do not add real partner samples unless they are approved for public redistribution. Prefer synthetic examples with obvious placeholder values.
