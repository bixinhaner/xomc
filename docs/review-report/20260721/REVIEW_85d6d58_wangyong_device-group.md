# Review: Issue 134 Device Group Matching

## 结论

PASS

本次变更无 CRITICAL 问题，可提交。

## 范围

- `omcgo/internal/topology/*`
- `omcmb/frontend-core/src/services/api/*`
- `omcmb/frontend-core/src/i18n/*`
- `omcmb/webcode/src/pages/device/DeviceGrouping/*`

## 审查要点

### 后端

- 默认 L2 源组匹配现在同时覆盖无 membership 的旧未分组设备，以及显式归属默认组的设备。
- 默认源组自动移动使用 `ON CONFLICT ... DO UPDATE`，且 `WHERE device_group_members.group_id = $4` 保留源组边界，不会覆盖其他真实分组的设备。
- 单设备匹配路径同步接受当前组为显式默认组，避免注册/心跳路径与分组编辑路径语义不一致。
- SQL 使用参数绑定，无字符串拼接注入风险。

### 前端

- `parent_id` 省略时归一为 `null`，根分组不会被误判为可选二级源组。
- 源设备组选项使用 `parentId != null`，仅展示真实二级分组。
- 编辑匹配规则成功提示改为“规则已保存、后台匹配已触发”，不再承诺一定有设备数据更新。

## 风险与建议

- 分组匹配仍是异步执行，页面计数和列表需要刷新后查看；当前文案已显式弱化为“命中设备会自动归入”。
- 本次没有新增同步返回匹配数量的接口，若后续需要即时反馈命中数，建议新增后端任务状态或匹配结果查询接口。

## 验证

- `go test ./internal/topology` — 通过
- `go test ./...` — 通过
- `go build ./...` — 通过
- `npm run test --workspace webcode -- ../frontend-core/src/services/api/__tests__/deviceApi.test.ts` — 27 passed
- `npm run typecheck` — 通过
- `npm run build` — 通过，只有既有 Vite chunk/config 警告
- Docker Compose 热重启 — 通过
- `curl -I http://localhost:8081/` — HTTP 200
- 真实 SN 精确匹配自测：`matched=1 moved=1`
- 真实设备名称包含匹配自测：`matched=1 moved=1`
- 非命中规则 `startWith JB0` 自测：后端 `matched=0 moved=0`，页面显示 0，提示文案已调整
