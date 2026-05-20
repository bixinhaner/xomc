# 代码审查报告 — T-0161 TR069 报文跟踪删除功能

| 字段 | 值 |
|------|------|
| 任务 | T-0161（TR069 报文抓取页面 — 单条删除 + 批量删除） |
| 范围 | trace 模块（backend handler/service/repo + frontend api/hook/page + i18n） |
| 改动量 | 9 文件（4 Go / 4 TS / 1 新增 Go 测试） |
| 审查结论 | **PASS** |

---

## 1. 变更概览

为 TR069 报文跟踪（"TR069 Message Trace"）页面补齐删除能力：行内单删（含二次确认弹窗）+ 表头批量删除（基于已勾选行）。

后端在 `internal/trace/` 已有 `StopTask(purge=true)` 的"停止并清理"路径，但只翻转状态、不删 `trace_tasks` 行；本次新增的是任务行的物理删除：DELETE `trace_tasks` 同时清掉它的 messages，并发 `trace.task.purged` 事件让 sweeper 异步清 MinIO 对象。

前端复用现有的 DataTable `selectable` + `batchActions` 机制，行内删除按钮在 `running` 状态下 disabled（避免发请求被后端 400 拒）。

---

## 2. 变更清单

### 后端（4 文件）

| 文件 | 改动 |
|------|------|
| `omcgo/internal/trace/repository.go` | Repository 接口新增 `DeleteTask` / `BatchDeleteTasks` |
| `omcgo/internal/trace/pg_repository.go` | 两个新方法的 Squirrel/pgx 实现 |
| `omcgo/internal/trace/service.go` | `Service.DeleteTask` 单删 + `Service.BatchDeleteTasks` 批删（沿用 device 模块 `BatchOperationResult` 风格的 `DeleteResult`） |
| `omcgo/internal/trace/handler.go` | 路由 `DELETE /trace/tasks/:id` + `POST /trace/tasks/batch-delete`；批量删除审计 `auditBatchDelete` |

### 后端测试（1 文件，3 mock 同步）

| 文件 | 改动 |
|------|------|
| `omcgo/internal/trace/delete_test.go` | 新增 5 个测试用例（成功 / running 拒绝 / 未找到 / 批量混合 / 空切片） |
| `omcgo/internal/trace/whitelist_test.go` | mockRepo 补齐两个新方法 |
| `omcgo/internal/trace/service_test.go` | captureMockRepo 补齐两个新方法 |

### 前端（4 文件）

| 文件 | 改动 |
|------|------|
| `omcmb/frontend-core/src/services/api/traceApi.ts` | 新增 `deleteTask` / `batchDeleteTasks` + 后端响应类型 `BackendTraceBatchDeleteResult` |
| `omcmb/frontend-core/src/hooks/api/useTrace.ts` | 新增 `useDeleteTraceTask` / `useBatchDeleteTraceTasks` Hook |
| `omcmb/webcode/src/pages/ops/MessageTrace/index.tsx` | 行内删除按钮（Popconfirm 二确） + 表头批量删除（Modal.confirm 二确）；`selectable` 勾选行 |
| `omcmb/frontend-core/src/i18n/{zh-CN,en-US}/index.ts` | 8 条新 key：`trace.action.delete` / `trace.action.batchDelete` / `trace.confirm.delete` / `trace.confirm.batchDelete` / `trace.delete.runningHint` / `trace.delete.result` |

---

## 3. 关键设计决策

### D1：running 任务禁止直接删除

`Service.DeleteTask` 在 `t.Status == TaskStatusRunning` 时返回 `ErrInvalidInput`，前端把按钮 disable + Tooltip 提示"请先停止"。理由：

- 删除 running 任务会让 ACS 进程的 SN 白名单残留，下一条 capture 投递时找不到 task 行（孤儿 message）
- 用户操作流应当先 "停止" 再 "删除"，避免误操作丢数据
- 现有的"停止并清理"路径（`StopTask purge=true`）仍保留，与本任务的"删除"互补：前者保留任务行（status=purged）做审计，后者完全清账

### D2：批量删除走"逐个调 DeleteTask"而非"一条 SQL"

`Service.BatchDeleteTasks` 内部 for 循环调 `DeleteTask`，没有用 repo 的 `BatchDeleteTasks` 一把梭。理由：

- 复用 running 拒绝 / messages 先清 / `trace.task.purged` 事件发布的完整链路
- batch 上限 100（handler 校验），N 次 GetTask + DELETE 的开销可接受
- 部分失败可定位：返回 `DeleteResult{succeeded, failed, errors[]}`，前端用 message.warning 把第一条 error 展示给用户

repo 层的 `BatchDeleteTasks` 方法仍然保留，留作未来"管理员强删（绕过 running 校验）"或定期清理脚本使用。

### D3：messages 清理走 PurgeTaskMessages，不依赖 FK CASCADE

`trace_messages` 是 TimescaleDB hypertable，PG 限制下不能让分区表被 FK 引用，所以没有 `ON DELETE CASCADE`。Service 层显式 `PurgeTaskMessages(taskID)` → `DeleteTask(taskID)` 两步。`trace_export_jobs` 有 CASCADE FK，免维护。MinIO 对象由 `trace.task.purged` 事件触发 sweeper 异步清理（与 `StopTask purge=true` 复用同一通道）。

### D4：批量删除二确用 Modal.confirm 而非 Popconfirm

DataTable 的 `batchActions` onClick 不带二确，而批量删除按钮渲染在 Toolbar 内由通用组件控制，没法就地包 Popconfirm。改用 `Modal.confirm` 在 handler 里弹，逻辑解耦也更稳。行内单删仍是 Popconfirm（在 column render 里直接包，符合 AntD 推荐用法）。

---

## 4. API 契约

### `DELETE /api/v1/trace/tasks/:id`

请求：无 body。

响应 200：
```json
{ "deleted": true, "task_id": "uuid-here" }
```

错误：
- 400 `invalid input`：任务为 `running` 状态（必须先 stop）
- 404 `resource not found`：任务不存在

### `POST /api/v1/trace/tasks/batch-delete`

请求：
```json
{ "ids": ["uuid1", "uuid2", ...] }
```

响应 200：
```json
{
  "total": 3,
  "succeeded": 2,
  "failed": 1,
  "errors": [
    { "id": "uuid", "message": "invalid input: task is running, stop it first" }
  ]
}
```

错误：
- 400 `ids` 为空或超过 100 条

---

## 5. 兼容性 / 数据迁移

- **Schema**：无新增表 / 列，无迁移文件
- **API**：纯新增端点，老前端不调用即可，不破坏现有调用方
- **事件**：复用现有 `trace.task.purged` subject，sweeper 已有订阅逻辑无需改动

---

## 6. 测试

### 后端单元测试（新增 5 个用例）

| 用例 | 验证点 |
|------|------|
| `TestService_DeleteTask_Success` | stopped 状态可删；PurgeTaskMessages 先于 DeleteTask 调用 |
| `TestService_DeleteTask_RunningRejected` | running 状态返 `ErrInvalidInput`；不触发 purge / delete |
| `TestService_DeleteTask_NotFound` | 不存在的 ID 返 `ErrNotFound` |
| `TestService_BatchDeleteTasks_MixedSuccessFailure` | 3 个 ID（ok / running / missing）→ succeeded=1, failed=2 |
| `TestService_BatchDeleteTasks_Empty` | 空切片 → total=0, succeeded=0, 不报错 |

```
$ go test ./internal/trace/...
ok  	github.com/omcgo/omcgo/internal/trace	1.181s
```

### 编译 / 类型检查

- `go build ./...`：PASS
- `go vet ./internal/trace/...`：PASS
- `npm run typecheck`：PASS

---

## 7. 风险

| 风险 | 评估 | 缓解 |
|------|------|------|
| 误删 stopped 但还有用的任务（例如刚导出但还想再查） | 中 | Popconfirm 二确 + AntD danger 红色按钮提示破坏性操作；删除会同步清 messages 与 export_jobs |
| 批量删除卡死（逐个 GetTask + DELETE） | 低 | handler 限制 batch ≤ 100；按 SN 索引扫描 PG 是 ms 级 |
| MinIO 残留对象（事件 publish 失败） | 低 | 与现有"停止并清理"路径共用 sweeper 订阅，sweeper 启动期会重放未消费事件；最坏情况留垃圾对象不影响功能 |

---

## 8. 后续工作（不在本任务范围）

- 可选"清空全部"按钮（默认未做，避免误操作；如运维有强需求可再加）
- 删除审计的 audit_logs 查询页面（当前已写入 audit，但 UI 上还没专门的过滤入口）

---

**结论**：通过审查。
