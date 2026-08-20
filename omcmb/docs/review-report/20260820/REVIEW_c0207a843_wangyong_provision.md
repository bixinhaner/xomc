# Review Report: provision original version and parameter mode

- Result: PASS_WITH_WARNINGS
- Scope: provision / PlugAndPlay
- Author: wangyong
- Base: c0207a843
- Date: 2026-08-20

## Summary

本次改动为即插即用策略增加显式参数配置模式，避免指定设备参数被公共参数错误合并；同时允许软件升级原始版本使用 tags 输入，并在提交前归一化、去重。

## Files Reviewed

- `omcgo/internal/provision/handler.go`
- `omcgo/internal/provision/policy_common_parameters.go`
- `omcgo/internal/provision/policy_common_parameters_test.go`
- `omcmb/webcode/src/pages/device/PlugAndPlay/AddPolicyPage.tsx`
- `omcmb/webcode/src/pages/device/PlugAndPlay/softwareUpgradeOriginalVersion.ts`
- `omcmb/webcode/src/pages/device/PlugAndPlay/softwareUpgradeOriginalVersion.test.ts`
- `omcmb/webcode/src/pages/device/PlugAndPlay/specifiedParamConfigEditor.test.ts`

## Findings

### CRITICAL

None.

### WARNING

1. `go test ./...` currently fails outside this change set:
   - `github.com/omcgo/omcgo/internal/agentruntime`: handbook package is stale and needs regeneration from `../../data/agent-skill/omc-operations/references`.
   - `github.com/omcgo/omcgo/internal/alarm/definition`: builtin alarm definition count expects 8 items but finds 9, including `GSM.xml`.

These failures are not in the provision/PlugAndPlay files touched here, but they still keep the repository-wide Go test gate from being fully green.

### INFO

1. Backend compatibility was checked: explicit `specified` mode skips common parameter materialization, while legacy policies without `paramConfigMode` but with `commonParamConfig` still follow historical common-mode behavior.
2. Frontend submit behavior now clears the inactive parameter payload so common and specified-device modes do not leak values into each other.
3. Original software versions are normalized through a small helper with tests for editable tags, delimiter splitting, deduplication, and the all-version sentinel.

## Validation

- `cd omcgo && go test ./internal/provision` — PASS
- `cd omcgo && go build ./...` — PASS
- `cd omcmb && npm run typecheck` — PASS
- `cd omcmb && npm run test --workspace webcode -- src/pages/device/PlugAndPlay/specifiedParamConfigEditor.test.ts src/pages/device/PlugAndPlay/softwareUpgradeOriginalVersion.test.ts` — PASS, 2 files / 8 tests
- `cd omcgo && go test ./...` — FAIL, unrelated failures listed above

## Conclusion

No blocking issue was found in the staged diff. The change is suitable to commit with the repository-wide Go test warning documented for follow-up.
