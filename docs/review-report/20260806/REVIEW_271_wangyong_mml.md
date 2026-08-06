# Code Review: MML SignallingTrace Enable Compatibility

Date: 2026-08-06
Branch: fix/271-mml-signalling-trace-enable
Base: 594cb4c6d
Scope: mml
Result: PASS_WITH_WARNINGS

## Reviewed Changes

- Added the Baicells `Device.DeviceInfo.SignallingTrace.Enable` TR-069 compatibility override: encode boolean values as numeric `0`/`1` and send them as `xsd:string`.
- Kept the standard `xsd:boolean` wire type for other parameter paths.
- Normalized frontend MOD readback comparison so `true`/`false` and `1`/`0` are treated as equivalent.
- Invalidated the MML task query after successful command execution so the task view can refresh automatically.
- Added backend and frontend regression coverage for numeric boolean payloads, raw SignallingTrace values, and MOD/LST readback matching.

## Findings

### WARNING

- The complete `go test ./...` run has one unrelated failure in `omcgo/internal/task`: `TestService_PG_GetTask_TerminalTombstoneReturnsDurableDetails` fails while the test environment cannot connect to Redis on dynamically allocated local ports. The changed `internal/mml` package and all other reported packages pass.
- `npm run typecheck` is blocked by two pre-existing `Uint8Array<ArrayBufferLike>` versus `BlobPart` errors in `omcmb/webcode/src/pages/SystemLicense/History.tsx` and `omcmb/webcode/src/pages/SystemLicense/index.tsx`; neither file is part of this change.

### INFO

- The vendor-specific path override is deliberately narrow and preserves default TR-069 type mapping for all other paths.
- `git diff --cached --check` passed with no whitespace errors.

## Verification

- `go test ./internal/mml` passed.
- `go build ./...` passed in `omcgo`.
- `npm test -- --run src/pages/mml/Console/__tests__/buildMODReadbackRows.test.ts` passed: 6 tests.
- `go test ./...` completed with the unrelated `internal/task` environment-dependent failure described above.
- `npm run typecheck` reported only the unrelated SystemLicense errors described above.

## Conclusion

No CRITICAL findings. The MML change is acceptable to commit and open an MR, with the repository-wide validation warnings recorded above.
