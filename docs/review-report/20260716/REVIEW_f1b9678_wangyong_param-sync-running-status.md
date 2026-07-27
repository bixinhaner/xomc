# 代码审查报告：参数同步运行态与定时同步

- 审查时间：2026-07-16
- 审查人：Codex
- 基线提交：f1b9678
- 关联 Issue：#80
- 范围：`omcgo` 参数同步/定时同步、设备列表 DTO、`omcmb` 设备列表与系统配置
- 结论：PASS_WITH_WARNINGS

## 审查摘要

本次改动将设备列表的参数同步动态标识切到 durable `parameter_sync_*` 数据面，新增 `param_sync_running` DTO 字段；同时把定时同步配置从小时改为分钟，补充自动同步弹窗和运行态短轮询。后端同步入口统一为先提交 durable parameter sync，旧 `sync-gpv` Path B 仅作为临时兜底，待 `param_sync_running` 稳定后删除。

## CRITICAL

无。

## WARNING

1. 过渡期 legacy fallback 任务不会反向参与 `param_sync_running` 计算。
   - 位置：`omcgo/internal/device/device_info_pg_repository.go`
   - 说明：设备列表动态标识只读取 `parameter_sync_requests` / `parameter_sync_runs` 的非终态状态，不再读取旧 `device_tasks sync-gpv-*`。这符合当前“所有同步先走 `parameter_sync_*`，旧 Path B 仅临时兜底”的策略；若 durable 不可用并实际 fallback 到旧链路，页面可能看不到旧任务运行态。
   - 建议：保持当前策略，后续 `param_sync_running` 稳定后删除旧 fallback；若过渡期必须展示旧兜底运行态，需要单独定义兼容口径，避免重新耦合旧任务表。

2. `DeviceList/index.tsx` 仍有既有 React hooks lint warning。
   - 位置：`omcmb/webcode/src/pages/device/DeviceList/index.tsx`
   - 说明：ESLint 报告 7 个 warning，主要是历史 effect setState 和依赖列表问题；本次改动未新增 error。
   - 建议：另开清理任务处理，避免和参数同步功能变更混在同一提交。

## INFO

- 定时同步启动后立即检查一次，并在 leader/DB 临时失败时不推进 `lastRunAt`，下一轮 poll 可尽快重试。
- `periodicSyncIntervalMinutes` 优先，保留 `periodicSyncIntervalHours` 兼容读取。
- 前端自动同步弹窗复用系统配置批量保存逻辑，保存启用后开启短时间列表轮询。
- 运行态图标为蓝色连接图标加旋转同步图标，无文字展示。

## 验证

- `cd omcgo && go build ./...`：通过
- `cd omcgo && go test ./...`：通过
- `cd omcmb && npm run typecheck`：通过
- `cd omcmb/webcode && npm run build`：通过
- `cd omcmb/webcode && npx eslint src/pages/device/DeviceList/index.tsx src/pages/system/SystemConfig/DeviceSettings.tsx src/pages/system/SystemConfig/index.tsx`：0 error，7 warnings
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web`：通过
- `curl -I --max-time 10 http://localhost:8081/`：`HTTP/1.1 200 OK`

## 结论

未发现阻断提交的问题。当前 warning 均为过渡期策略或既有 lint 债务，不影响本次提交。
