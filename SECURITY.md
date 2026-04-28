# Security Policy

EDIForge handles data that can include shipment details, customer names, addresses, account identifiers, order numbers, invoices, and warehouse activity. Treat raw EDI as sensitive.

## Supported Versions

Until the first stable release, security fixes apply to the default branch.

## Reporting a Vulnerability

Please do not open a public issue for vulnerabilities that could expose EDI data, bypass local server protections, execute untrusted mapping logic, or read/write unintended files.

Report privately to the project maintainers. If no private channel is published yet, open a minimal public issue requesting a security contact without including exploit details or sensitive sample data.

## Security Defaults

EDIForge should default to:

- No outbound network calls.
- No telemetry.
- REST server bound to `127.0.0.1`.
- Token required for non-localhost binding.
- CORS disabled unless explicitly configured.
- Request and file size limits.
- No raw EDI in logs.
- No local history unless explicitly enabled.
- Declarative schemas and mappings only.

## Sensitive Data in Reports

Do not include real customer EDI in public issues, tests, screenshots, or logs. Use synthetic samples or redact:

- Names
- Addresses
- Phone numbers
- Email addresses
- Trading partner IDs
- Control numbers
- Purchase order, shipment, invoice, and account identifiers

## Schema Safety

Schema and mapping files must not execute code. MVP mappings should not support shell commands, JavaScript, Go templates, dynamic plugins, network calls, or arbitrary file reads.

