# Issue 173 设备列表同步时间展示审查

- 日期: 2026-07-24
- 基线: ae6f162
- 分支: fix/172-device-list-param-sync-time
- 结论: PASS

## 范围

- 后端设备列表查询补充 `last_param_sync_at` 字段并映射到设备列表模型。
- 参数同步结果处理在列表/局部同步成功后同步更新设备的 `last_param_sync_at`。
- 设备列表新增“上次同步时间”列，参数/告警同步运行图标恢复到连接状态列，并在离线设备停止显示。
- 任务进度表删除“操作”和“类型”列，统一 SN 与设备名称的字体表现。

## Findings

未发现 CRITICAL / WARNING 级别问题。

## Notes

- `last_param_sync_at` 更新从全量同步分支移出后，手动列表同步完成也会刷新设备级同步时间，符合用户反馈的“刚点击同步仍显示几小时前”的根因。
- 连接状态列图标仅对在线设备展示，离线设备不会继续显示同步动画，避免离线状态下的误导。
- “上次同步时间”使用精确完成时间点展示，并随导出字段一起输出。
- 本次仅调整 seed 默认值；已存在环境的配置值仍以运行库配置为准。

## Validation

- `cd omcgo && go test ./internal/device ./internal/paramsync` — PASS
- `cd omcmb/webcode && npm run typecheck` — PASS
- `cd omcmb/webcode && npm run build` — PASS
- `git diff --check` — PASS
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web` — PASS
- `curl -I http://localhost:8081/` — PASS, HTTP 200
