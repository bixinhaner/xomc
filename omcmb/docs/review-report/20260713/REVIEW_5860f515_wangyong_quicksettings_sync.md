# 快速设置同步与首屏加载审查报告

- 结论: PASS
- 范围: BSC Path B 实例展开、快速设置同步终态判断、快速设置首屏加载优化、seed 迁移编号调整
- 审查时间: 2026-07-13

## 审查文件

- `omcgo/internal/device/device_param_pg_repository.go`
- `omcgo/internal/provision/sync_pathb_expand.go`
- `omcgo/internal/provision/sync_pathb_expand_test.go`
- `omcgo/migrations/seed/000012_update_mml_script_menu_label_order.sql`
- `omcmb/frontend-core/src/components/QuickSettingsSyncWatcherCore.tsx`
- `omcmb/frontend-core/src/utils/quickSettingsSyncStatus.ts`
- `omcmb/frontend-core/src/utils/quickSettingsSyncStatus.test.ts`
- `omcmb/webcode/src/components/Layout/QuickSettingsSyncWatcher.tsx`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/index.tsx`
- `omcmb/webcode/src/pages/device/DeviceDetail/index.tsx`

## 重点审查

### BSC Path B 实例展开

- `DeviceGSM.Bts.` cold-start 仍会按 1..256 展开，避免整对象 GPV 超过 NATS payload。
- `DeviceGSM.Bts.0.*` 被明确排除在实例展开外，符合站级参数语义。
- DB rows 迭代错误已补充 `rows.Err()` 检查。
- 单测覆盖无历史、DB 错误、异常大历史实例号和分组预算。

### 快速设置同步终态

- 成功/失败判定抽到共享 util，避免 `frontend-core` 与 `webcode` 两份 watcher 逻辑漂移。
- sourceId 不匹配的成功终态不会误结束当前 monitor。
- device-wide failure 在 `syncing` 中不会被归因到当前 quick settings monitor，降低并发同步误报。
- 新增 Vitest 覆盖 source mismatch、syncing failure 和 idle failure。

### 快速设置首屏加载

- `DeviceDetail` 概览小区实例解析仅在 `basic` tab 启用，避免进入 quick settings 时重复 schema fan-out。
- `QuickSettingsGroupGate` 通过 IntersectionObserver 延迟挂载非首屏分组，首屏保留上方外层分组和第一个实例分组立即加载。
- fallback 在不支持 IntersectionObserver 时直接渲染，兼容旧环境。

## 风险

- `QuickSettingsGroupGate` 延迟挂载会让未滚动到的分组稍后才发起 schema/search 请求；这是预期的首屏减载行为。
- 本次 seed 迁移为编号重排，不改变 SQL 内容；需确保目标环境尚未执行旧编号文件。

## 验证

- `cd omcgo && go test ./internal/provision ./internal/device` 通过
- `cd omcgo && go build ./...` 通过
- `cd omcgo && go test ./...` 通过
- `cd omcmb && npm run typecheck` 通过
- `cd omcmb && npm run test --workspace webcode -- ../frontend-core/src/utils/quickSettingsSyncStatus.test.ts` 通过
- `cd omcmb/webcode && npm run build` 通过
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web` 通过
- `curl -I --max-time 10 http://localhost:8081/` 返回 200 OK

## Findings

无 CRITICAL / WARNING。
