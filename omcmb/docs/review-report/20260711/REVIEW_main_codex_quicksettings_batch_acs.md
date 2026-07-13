# Review: quicksettings batch ACS response

Date: 2026-07-11
Branch: fix/quicksettings-batch-acs-response
Base: main
Author: codex
Scope: quicksettings, acs
Result: PASS_WITH_WARNINGS

## Summary

本次审查覆盖快速设置多实例参数批量处理、提交后回读状态收敛、LTE QOffset 前端校验，以及 ACS RPC Response 与任务关联逻辑。

## Findings

### WARNING

- 真实设备的 AddObject/SPV/DeleteObject 回读仍依赖基站按序返回和后端 GPV 写库时机。当前实现已在提交后保留待确认行、失败时删除本地新增快照、成功后按回读列表收敛，但若基站长时间延迟回读，Tag 会继续展示等待状态。

### INFO

- ACS 现在校验 `*Response` 方法必须匹配任务方法，避免 `DeleteObjectResponse` 被错误挂到 `AddObject` 任务；无法匹配时保留未关联日志，便于后续定位异常 CPE 会话。
- 快速设置列表新增本地暂存和批量提交状态，避免新增、编辑、删除单次操作立即下发。
- LTE QOffset 在前端限制为 `-24..24` 的偶数，减少已知设备侧 9007 参数错误。

## Files Reviewed

- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/MultiInstanceTable.tsx`
- `omcmb/frontend-core/src/store/quickSettingsFeedbackStore.ts`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/validators.ts`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/validators.test.ts`
- `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- `omcmb/frontend-core/src/i18n/en-US/index.ts`
- `omcgo/internal/acs/handler.go`

## Verification

- `cd omcmb && npm run typecheck` passed.
- `cd omcgo && go build ./...` passed.
- `cd omcgo && go test ./...` passed.
- `git diff --check` passed.
- Earlier focused checks also passed: `cd omcmb/webcode && npm run test -- QuickSettingsTab/__tests__/validators.test.ts`, `cd omcmb/webcode && npm run build`.
- Earlier deployment check passed: frontend hot redeploy returned `HTTP/1.1 200 OK`; ACS/app containers were rebuilt and restarted.

## Conclusion

未发现 CRITICAL 问题，可以提交。残余风险主要是现场基站回读延迟或异常 CWMP ID 行为，需要依赖任务日志和页面 Tag 状态继续观察。
