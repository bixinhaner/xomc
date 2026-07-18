# Review Report: device quicksettings bsc

- Date: 2026-07-18
- Author: wangyong
- Scope: device
- Base: 9f90f5f39
- Result: PASS_WITH_WARNINGS

## Summary

This review covers BSC/BTS QuickSettings changes for multi-checkbox serialization, CodecSupport handling, add-modal SPV payload normalization, table/selector diff comparison, and the related Vitest coverage.

## Files Reviewed

- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/BscBtsAddModal.tsx`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/CellParameterForm.tsx`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/InstanceSelectorForm.tsx`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/MultiInstanceTable.tsx`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/validators.ts`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/bscBtsAddModal.test.ts`

## Findings

### CRITICAL

None.

### WARNING

- The implementation and automated checks are limited to `omcmb/webcode` v1 QuickSettings code. No v2/v3 skin files were changed in this patch.

### INFO

- CodecSupport now keeps `fr` selected in the UI while omitting it from the SPV value, matching the device encoding expected by the BSC/BTS add flow.
- Generic multi-checkbox values now parse both comma-separated and dash-separated forms, then serialize to the dash-separated device representation before comparison and submission.
- Enum display labels in the BSC/BTS add modal are normalized back to schema enum values before SPV.

## Validation

- `cd omcmb && npm run typecheck` - passed.
- `cd omcmb && npm exec --workspace webcode vitest -- run src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/bscBtsAddModal.test.ts` - passed, 3 tests.

## Conclusion

No blocking issues found. The staged changes are acceptable to commit.
