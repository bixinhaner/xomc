# Review: DeviceGrouping Auto Assign Toggle

- Date: 2026-07-24
- Author: Codex
- Scope: DeviceGrouping
- Result: PASS

## Summary

本次变更为设备分组 L2 新增/编辑抽屉增加“自动归属设备组”开关。开关关闭时隐藏源设备组与匹配方式配置，新增时不提交匹配规则，编辑时清空匹配规则与源设备组，从而让后端现有逻辑不执行自动归属。

## Files Reviewed

- `omcmb/webcode/src/pages/device/DeviceGrouping/AddChildGroupDrawer.tsx`
- `omcmb/webcode/src/pages/device/DeviceGrouping/EditLevel2GroupDrawer.tsx`
- `omcmb/webcode/src/pages/device/DeviceGrouping/GroupDialogs.tsx`
- `omcmb/webcode/src/pages/device/DeviceGrouping/index.tsx`
- `omcmb/webcode/src/pages/device/DeviceGrouping/useGroupActions.tsx`
- `omcmb/frontend-core/src/hooks/api/useDevices.ts`
- `omcmb/frontend-core/src/services/api/deviceApi.ts`
- `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- `omcmb/frontend-core/src/i18n/en-US/index.ts`

## Findings

### CRITICAL

None.

### WARNING

None.

### INFO

- 当前实现复用既有后端字段语义，没有新增数据库字段或迁移；开关状态通过是否存在匹配规则推断，符合本次轻量修复范围。
- 本次只改动 v1 `webcode` 页面及 `frontend-core` 类型/文案。`webcode-v2`、`webcode-v3` 未发现同名设备分组抽屉实现需要同步。

## Verification

- `cd omcmb && npm run typecheck` — PASS
- `cd omcmb/webcode && npm run build` — PASS
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web` — PASS
- `curl -I --max-time 10 http://localhost:8081/` — PASS, `HTTP/1.1 200 OK`

## Risk

风险较低。主要影响设备分组 L2 新增/编辑抽屉的匹配规则提交行为；关闭开关会清空已有匹配规则，符合“关闭时不走该逻辑”的需求。
