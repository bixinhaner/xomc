# Review: BM Quick Settings Sync Source

## Scope

- BM quick settings XML for NTP, sync source, GNSS constellation, and PTP parameters.
- BM parameter mappings for NTP, PpsTimeMode, GNSS SyncSource, and PTP1588 fields.
- Quick settings form behavior for bitmask select values, conditional GNSS/PTP visibility, NTP server filtering, and post-SPV readback refresh.
- Sync GPV stale task handling and recovery task expiry.

## Result

PASS

No CRITICAL issues found.

## Findings

- INFO: BM bitmask fields currently have browser-level smoke coverage and manual review, but no focused unit test for option generation or conditional visibility. Consider extracting the bitmask option and visibility helpers if this logic grows further.

## Checks

- Verified that `PpsTimeMode` and `GNSS.SyncSource` combination values are accepted by backend validation because MappingValidator only enforces access and numeric/string range, not enum membership.
- Verified that hidden quick settings fields are removed from `visibleParams`, so hidden GNSS constellation does not participate in save payloads.
- Verified that sync GPV normal batches and recovery batches set finite expiry before repository queries started ignoring nil `expires_at` tasks.
- Verified that NTP server filtering hides empty and invalid `0.0.0.0` values without hiding valid server names.

## Validation

- `cd omcmb && npm run typecheck` — passed
- `cd omcgo && go test -count=1 ./internal/quicksettings ./internal/config/parammodel ./internal/task` — passed
- `cd omcgo && go build ./...` — passed
- Local Docker web rebuild/restart and `http://localhost:8081/` smoke check were completed during implementation.

## Risk

- Low. Main risk is UI behavior around BM-specific bitmask fields, mitigated by browser smoke testing and narrow path checks.
