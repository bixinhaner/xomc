# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-28 16:49 |
| 提交 | a4380fd0 |
| 作者 | zhanglu |
| 范围 | fullstack-multi |
| 变更文件数 | 5 |
| 新增行数 | +180 |
| 删除行数 | -2 |

## 变更概要

本次变更修复了两类相关问题：一是设备详情页的活动告警分页条数选择未真正接入查询状态，导致失焦后回退到默认 20 条；二是告警规则创建设备分组可见范围未复用现有数据权限逻辑，导致普通用户可看到超权限的分组树。

后端为 topology 设备分组树接入了 PermissionService 过滤，并补充了对应测试；前端为通用 DataTable 页大小选择补齐 `onShowSizeChange`，同时将设备详情活动告警改为受控分页并在浏览器中完成了实测验证。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

- 已执行 `go test ./internal/topology/...`、`go test ./cmd/app/provider/...`、`npm run typecheck`，并在 Docker 8081 环境做了浏览器回归；当前这组修改的验证闭环完整。

## 详细分析

### `omcgo/internal/topology/handler.go`

- 新增 `VisibleGroupsResolver` 抽象和 `SetPermissionService` 注入点，复用了现有权限服务，没有引入新的运营商或权限硬编码。
- `GetTreeWithCounts` / `ListTree` 在 handler 层完成过滤，控制面定位正确；过滤后重建统计数据，避免返回全量树统计造成前后不一致。
- 错误处理保持显式返回，未发现 panic、裸错误吞掉或资源泄漏问题。

### `omcgo/cmd/app/provider/router.go`

- 路由组装处补齐了 `topologyHandler.SetPermissionService(c.PermService)`，依赖注入位置正确，未改变现有公开路由契约。

### `omcgo/internal/topology/handler_test.go`

- 增加了带鉴权上下文的测试路由和 `mockVisibleGroupsResolver`，覆盖了设备分组树按用户可见分组裁剪的关键路径。
- 测试同时断言了树结构和统计值，能够有效防止未来回归。

### `omcmb/webcode/src/components/DataTable/index.tsx`

- 为 antd `Pagination` 增加 `onShowSizeChange={onPageChange}`，修复了受控分页页大小切换时只触发 `onChange` 不足的问题。
- 修改保持在通用组件最小表面，没有改变现有 props 契约。

### `omcmb/webcode/src/pages/device/DeviceDetail/index.tsx`

- 设备详情活动告警此前使用硬编码 `{ page: 1, pageSize: 20 }` 查询参数，导致右下角条数选择无法持久；现已接入 `alarmPage` / `alarmPageSize` 受控状态，并回灌给 `DataTable`。
- 增加了 `sn` 变更时的分页重置，避免切换设备后沿用旧页码，行为合理。

## 业务完整性检查

- Handler-Service-Repository 链路：通过。后端修改复用了既有 `PermissionService` 和 group tree 查询链路，无空壳实现。
- 路由注册：通过。topology handler 已在 router 注入权限服务。
- 测试配套：通过。后端权限过滤新增了针对性测试；前端分页修复完成类型检查和浏览器回归。
- 前后端一致性：通过。前端活动告警分页参数继续使用既有 `page/page_size/device_sn` 契约，后端分组树接口路径未变。

## 审查结论

**PASS**

- 未发现阻塞提交的问题。
- 可以按 hotfix 流程提交。