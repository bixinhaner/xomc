# Code Review Report

| Item | Detail |
|------|--------|
| Date | 2026-03-22 |
| Author | watermelon |
| Scope | acs |
| Type | fix |
| Conclusion | PASS_WITH_WARNINGS |

## Changed Files

| File | Changes | Description |
|------|---------|-------------|
| `pkg/soap/envelope.go` | +54 -54 | DetectRPCMethod: string matching → proper XML Body child element parsing |
| `internal/acs/cmdqueue/queue.go` | +1 | Add CWMPID field to Command struct |
| `internal/acs/handler.go` | +35 -12 | Skip test tasks for TC/ATC, Connection:close on 204, remove CurrentTime, fix test task JSON templates |
| `internal/acs/rpc/dispatcher.go` | +25 -13 | Use cmd.CWMPID for SOAP header ID, keep cmd.CommandKey for RPC-level key |
| `internal/acs/rpc/dispatcher_test.go` | +17 -9 | Update assertions for cwmpID separation and snake_case JSON |
| `pkg/soap/templates.go` | +36 -18 | Add json struct tags to DownloadData/UploadData |
| `scripts/cpe_simulator.py` | +1951 | New CPE simulator for protocol testing |

## Review Findings

### CRITICAL: None

### WARNING

1. **[W1] Large test script addition** (`scripts/cpe_simulator.py`, 1951 lines)
   - New CPE simulator added as untracked file. Acceptable as a development/testing tool under `scripts/`.
   - Contains hardcoded localhost URLs — appropriate for local testing only.

### INFO

1. **[I1] DetectRPCMethod XML parsing** — Correctly implements CWMP spec: first child element of `<soap:Body>` is the RPC method. Previous `strings.Contains` approach caused false matches (e.g., parameter name `PeriodicInformEnable` matched `Inform`).

2. **[I2] CWMPID vs CommandKey separation** — SOAP header `cwmp:ID` is now properly separated from RPC-level `CommandKey`. All 11 RPC handlers updated consistently.

3. **[I3] JSON struct tags** — `DownloadData` and `UploadData` now have snake_case json tags, matching the JSON format used in command queue params.

4. **[I4] Connection: close on 204** — Added per TR-069 spec to signal session end. Applied in all 3 empty-response paths.

5. **[I5] TC/ATC session handling** — Test task injection correctly skipped for TransferComplete and AutonomousTransferComplete sessions.

6. **[I6] Test task template fixes** — Multiple JSON key corrections: `parameter_path` → `path`, `parameters` → `values`/`attributes`, added required type fields.

## Conclusion

All changes are well-scoped fixes for TR-069 protocol compliance issues. No security concerns. Tests updated to match new behavior. **PASS_WITH_WARNINGS**.
