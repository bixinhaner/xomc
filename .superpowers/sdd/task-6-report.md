# Task 6 implementation report

## Scope

- Added the server-owned `MMLTemplate.txt` with `go:embed` and the download endpoint.
- Added authenticated multipart TXT validation, imported-script creation, replacement validation, and replacement save endpoints.
- Added request-body limits, `.txt` validation, strict save JSON decoding, stable import error codes, per-line validation payloads, and metadata-only ordinary script updates.
- Wired the Redis import-session store, PG authority validator, and transactional import service in the app module.

## Tests

Focused handler tests cover:

- template content and attachment headers;
- unauthenticated requests and non-TXT rejection;
- 422 validation responses carrying line issues;
- 201 imported-script creation;
- 403 ownership failure;
- 409 consumed-token replay;
- 413 oversized multipart uploads.

Commands:

```text
cd omcgo && /usr/local/go/bin/go test ./internal/mml -run 'TestHandler_.*ScriptImport' -count=1 -v
cd omcgo && /usr/local/go/bin/go test ./internal/mml -count=1
cd omcgo && /usr/local/go/bin/go build ./cmd/app
```

All three commands passed. No unrelated modules or user reference documents were changed.
