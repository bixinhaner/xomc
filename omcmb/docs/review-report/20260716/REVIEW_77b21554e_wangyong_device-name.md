# 代码审查报告：网管名称与 LMT 名称拆分

- 日期：2026-07-16
- 基线：`77b21554e`
- 分支：`fix/76-separate-omc-lmt-name-edit`
- 范围：设备详情、快速设置、设备改名服务
- 结论：PASS

## 变更概述

- 设备详情新增网管名称独立编辑入口，通过设备改名接口只修改网管侧名称。
- 快速设置中的 `HNBName` / `gNBName` 统一按普通参数通过 SPV 下发，语义固定为基站侧/LMT 名称。
- 名称同步下发按制式选择 LTE `HNBName` 或 NR `gNBName` 标准路径。
- “LMT 覆盖网管”策略不再禁止人工修改网管名称；后续设备上报仍按同步策略处理差异。

## 审查结果

### CRITICAL

无。

### WARNING

无。

### INFO

- 审查过程中发现前端已放开编辑、后端仍会在 `auto_lmt_to_omc` 下拒绝改名；已在提交前修复，并补充服务层回归测试。
- 名称路由前端测试采用源码边界断言，用于防止快速设置再次误接设备改名 Hook；后续若拆出纯路由函数，可升级为行为级单元测试。

## 重点检查

- Go：无字符串拼接 SQL、裸 `panic`、资源泄漏或新增未鉴权端点。
- React/TypeScript：复用现有 API Hook，无新增 `any`、Token 处理或 XSS 注入点。
- 兼容性：无数据库迁移；LTE 原路径保留，NR 改名下发补齐 `gNBName` 路径。
- 失败语义：SPV 下发失败仍不回滚已更新的网管名称，与现有改名流程一致。

## 验证

- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb/webcode && npx vitest run src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/nameSubmissionRouting.test.ts`：2 项通过。
- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./...`：通过，包含 E2E 与 integration。
- 本地 Docker Compose 热部署：`web`、`app`、`acs` 正常运行，`http://localhost:8081/` 返回 HTTP 200。
