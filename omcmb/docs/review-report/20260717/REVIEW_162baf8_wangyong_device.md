# Review Report

- Date: 2026-07-17
- Scope: device
- Base: 162baf8
- Conclusion: PASS

## Summary

本次变更集中在设备详情和设备分组前端：

- 新增设备详情页“密码管理”页签，通过参数下发 LMT 登录密码并跟踪设备任务状态。
- 增加中英文国际化文案，覆盖密码管理和默认设备组展示。
- 修正设备分组弹窗、编辑、删除确认中的内置默认组和多语言名称展示。
- 创建设备分组时按当前语言写入 `name_i18n`，编辑时保留已有多语言名称并更新当前语言。

## Findings

No CRITICAL findings.

## Checks

- React/TypeScript: hook 使用顺序正常，查询启用条件受 `deviceId` / `taskId` 控制。
- Security: 密码仅通过 `Input.Password` 和参数下发 mutation 传递，未写入本地持久化存储。
- i18n: 新增用户可见文案均进入 `frontend-core` 中英文资源。
- Data compatibility: 分组名称展示通过 `getRecordI18n` 回退 legacy 字段；编辑时保留已有 `nameI18n`。

## Validation

- `cd omcmb && npm run typecheck` — PASS

## Residual Risk

- 密码路径探测依赖设备已上报参数或 schema；未探测到时会回退到标准 LMT 密码路径，需后端/CPE 对该路径支持。
