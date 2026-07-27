# Review — 设备详情 License 刷新对齐快速设置

- Commit base: `c2345a221`
- Author: `wangyong`
- Scope: `device`
- Date: `2026-07-18`
- Result: `PASS_WITH_WARNINGS`

## 变更范围

- `DeviceDetail` 页头刷新按钮现在在 `license` tab 可见，并复用快速设置的 `sync-params` 参数同步流程。
- `LicenseParamsTab` 移除独立 `license-params/refresh` mutation，仅负责展示列表并向父组件暴露当前 license standardPath 集合。
- `QuickSettingsSyncWatcher` / core watcher 在同步完成后失效 `deviceLicenseParams` 查询缓存。
- `QuickSettingsSyncMonitor` 增加 `scope`，避免 License 刷新误清快速设置草稿或重挂载快速设置表单。

## 审查结论

### CRITICAL

无。

### WARNING

- License 参数列表为空时，页头刷新会以空 `parameterPaths` 调用 `sync-params`，后端语义为全量同步。该行为可保证首次拉取不被空表卡住，但对空 license 场景的同步范围较宽。当前与快速设置刷新逻辑一致，接受。

### INFO

- `license` scope 仍复用 quickSettingsSyncs 监控队列，以便共享 busy 状态、轮询和成功提示；watcher 中已按 scope 隔离 quick settings 草稿清理。
- `DeviceDetail/index.tsx` 中既有 lint warning 未在本次改动中展开处理，本次仅修复同文件已有的 unused catch error。

## 验证

- `npm run typecheck`（`/Users/wangyong/OBJECT/Codex/xomc/omcmb`）通过。
- `npm run build`（`/Users/wangyong/OBJECT/Codex/xomc/omcmb/webcode`）通过。
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web` 完成，`goomc-local-web-1`、`goomc-local-app-1`、`goomc-local-acs-1` 均为 `Up`。
- `curl -I --max-time 10 http://localhost:8081/` 返回 `HTTP/1.1 200 OK`。
