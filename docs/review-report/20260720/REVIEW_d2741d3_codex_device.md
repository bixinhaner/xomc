# Review: Issue #121 设备分组默认组与删除/回收站语义

结论：PASS

## 范围

- 后端设备分组、回收站列表、默认设备组归属逻辑。
- 拓扑设备组计数、删除分组后的成员迁移逻辑。
- 前端设备分组页移动目标、回收站按钮、删除按钮行为。

## 检查项

- 默认设备组现在按真实二级分组处理；新增设备、移动设备、添加设备、删除分组迁移均保持同一语义。
- 回收站列表继续保留软删除设备，查询列数与共享 scanner 对齐，避免 `scanDeviceWithInfoRow` 列数不匹配。
- 默认组过滤兼容历史无归属设备，同时保留新数据的真实 membership 查询。
- 设备分组页“移动”下拉会排除选中设备所在的当前分组，避免无意义移动。
- 设备分组页“回收站”继续软删；“删除”改为复用永久删除接口。
- 永久删除后刷新设备列表、回收站和分组缓存，避免界面残留。

## Findings

未发现 CRITICAL / WARNING 问题。

## 验证

- `cd omcgo && go test ./internal/device ./internal/topology` — 通过。
- `cd omcmb && npm run typecheck` — 通过。
- `cd omcmb && npm run test --workspace webcode -- frontend-core/src/utils/__tests__/deviceGroupTargets.test.ts` — 通过。
- 本地 Docker Compose web 已重建，`curl -I http://localhost:8081/` 返回 200。

## 风险

- “删除”现在是不可恢复的永久删除，前端确认文案已有不可恢复/删除全部数据提示；需要操作人员明确区分“回收站”和“删除”两个按钮。
- 默认组真实分组语义会影响历史“未分组 = 无 membership”数据的展示口径，当前代码已在列表、回收站和计数路径保留历史兜底。
