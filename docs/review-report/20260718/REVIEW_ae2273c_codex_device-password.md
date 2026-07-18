# Review: Issue #116 Device Password Management

## Scope

- Backend LMT password reset endpoint and task queueing path.
- ACS SOAP dispatcher support for vendor and common password reset RPC names.
- RPC failure fallback for password reset tasks.
- BaiBNQ LMT username/password parameter mappings.
- Device detail password management tab and device list sync indicator polish.

## Findings

### CRITICAL

- None.

### WARNING

- None.

### INFO

- `RPCResponseSubscriber.handlePasswordResetFallback` intentionally returns the fallback enqueue error after logging it, preserving visibility if fallback queueing fails.
- NR/gNB devices are now kept on update-only password flow even if model/product strings overlap with BNQ-style family names.

## Verification

- `cd omcmb && npm run typecheck` — PASS.
- `cd omcgo && go build ./...` — PASS.
- `cd omcgo && go test ./...` — PASS.
- Local Docker web redeploy completed before submission; `http://localhost:8081/` returned `HTTP/1.1 200 OK`.

## Conclusion

PASS.
