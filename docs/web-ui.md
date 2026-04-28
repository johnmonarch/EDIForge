# Web UI Guide

The embedded static UI is in `internal/web/dist` and is intended to be served by the Go binary at:

```text
http://127.0.0.1:8765
```

It is dependency-free at runtime and does not require npm for the Go server to work.

## MVP Capabilities

- Paste EDI into a text area.
- Load a local file in the browser.
- Choose `structural`, `annotated`, or `semantic` mode.
- Provide an optional schema ID.
- Call `POST /api/v1/detect`.
- Call `POST /api/v1/translate`.
- Call `POST /api/v1/validate`.
- Render JSON, errors, warnings, and metadata.
- Copy or download the current response JSON.

## Privacy Behavior

The browser UI should keep raw EDI input in memory only. It should not store raw EDI in `localStorage`, cookies, IndexedDB, or analytics tools by default.

## Future React Source

The React/Vite scaffold in `web/` is for future development. A future frontend build should output static assets that can be copied into `internal/web/dist`.

