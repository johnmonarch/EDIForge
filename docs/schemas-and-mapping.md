# Schemas and Mapping

EDIForge supports three output modes:

- `structural`: preserve the parsed EDI structure without a schema.
- `annotated`: add labels from legal metadata sources or user schemas.
- `semantic`: map source EDI into business-friendly JSON using a schema.

## Schema Resolution

Recommended resolution order:

1. Explicit schema path.
2. Project `./schemas`.
3. User `~/.edi-json/schemas`.
4. Built-in public-safe examples.

## Public-Safe Examples

Bundled examples live in `schemas/examples/`.

Current starter packs cover:

- X12: `810`, `850`, `856`, `214`, `990`, `997`, `999`
- EDIFACT: `ORDERS`, `ORDRSP`, `DESADV`, `INVOIC`

Every bundled schema must include:

- `id`
- `standard`
- transaction or message type
- version or release when applicable
- `source`
- `license`

These files are starter templates, not official standards text.

## Mapping Paths

Recommended mapping path syntax:

```text
purchaseOrderNumber: BEG[0].BEG03
orderDate: BEG[0].BEG05 | date('yyyyMMdd')
buyer.name: N1[N101='BY'].N102
buyer.id: N1[N101='BY'].N104
lineItems[].productId: LIN[].LIN03
lineItems[].quantity: LIN[] > QTY[QTY01.1='21'].QTY01.2
```

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

## Partner Overlays

Partner overlays should extend a base schema and include only content the contributor has rights to publish.

```json
{
  "id": "partner-example-x12-850",
  "extends": "x12-850-basic",
  "partner": {
    "name": "Synthetic Partner"
  },
  "rules": [
    {
      "field": "buyer.id",
      "required": true
    }
  ]
}
```
