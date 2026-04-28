# T-0043.B 后端验证报告 — 通知模板 + 历史记录

| 项 | 内容 |
|---|---|
| 任务 | T-0043 (W2.A.4 通知模板 + 历史记录) — 后端部分 |
| 子代理 | T-0043.B (worktree-agent-a40a27b8) |
| 日期 | 2026-04-28 |
| 工作树 | `.claude/worktrees/agent-a40a27b8` |
| 分支 | `worktree-agent-a40a27b8` |

## 1. 改动清单

### 1.1 数据库迁移 (1 个新文件)

- `omcgo/migrations/000041_notification_templates_history.sql`
  - 新建 `notification_templates` 表 (id/name/channel/language/subject/body/variables[]/enabled/timestamps)
  - 新建 `notification_history` 表 (id/template_id/channel/recipients[]/subject/body/status/error_message/alarm_id/retry_count/sent_at/created_at)
  - 索引：channel/enabled/template_id/alarm_id/status/(channel,created_at DESC)
  - 字段约束：channel ∈ {email,sms,webhook}; status ∈ {pending,sent,failed,dead_letter}; FK template_id ON DELETE SET NULL
  - Down 段完整对应 Up 段，遵循 §5.5.10 自查清单

### 1.2 通知模板子模块 (8 个新文件)

| 文件 | 行数概览 | 角色 |
|---|---|---|
| `template_model.go` | ~80 | 领域结构体 + Filter + Create/Update Request + 频道/语言常量 |
| `template_repository.go` | ~20 | TemplateRepository 接口 (List/GetByID/GetByName/Create/Update/Delete) |
| `pg_template_repository.go` | ~250 | PostgreSQL 实现，10 列 templateColumns 共享于全部 5 处 SQL，去重 W1.5 教训 |
| `template_service.go` | ~170 | 业务校验 (channel allow-list / 名字 trim / variables 去重) |
| `template_handler.go` | ~140 | Gin handler + RegisterRoutes (`/templates`) |
| `template_service_test.go` | ~270 | 9 个测试，含 in-memory mock repo |
| `template_handler_test.go` | ~170 | 8 个 HTTP-level 测试 |
| `pg_template_repository_test.go` | ~55 | 列名漂移防护、sort 白名单一致性 |

### 1.3 通知历史子模块 (8 个新文件)

| 文件 | 行数概览 | 角色 |
|---|---|---|
| `history_model.go` | ~50 | 领域结构体 + Filter + 状态常量 |
| `history_repository.go` | ~25 | HistoryRepository 接口 (List/GetByID/Insert/UpdateStatus) |
| `pg_history_repository.go` | ~210 | PostgreSQL 实现，12 列 historyColumns 共享于全部 4 处 SQL |
| `history_service.go` | ~120 | 业务校验 + MarkSent/MarkFailed/MarkDeadLetter 状态转换 API |
| `history_handler.go` | ~110 | Gin handler，仅暴露只读端点 (`/history`) |
| `history_service_test.go` | ~250 | 9 个测试，含 in-memory mock repo |
| `history_handler_test.go` | ~110 | 6 个 HTTP-level 测试 |
| `pg_history_repository_test.go` | ~50 | 列名漂移防护、sort 白名单一致性 |

### 1.4 严守边界

- ❌ 未改 `omcmb/` 任何文件（前端 sub-agent 工作）
- ❌ 未改 `cmd/app/provider/modules.go` / `cmd/app/router/*`（主会话整合）
- ❌ 未改 `scripts/e2e_verify.sh`（主会话整合）
- ❌ 未改 `internal/notification/{model,handler,pg_repository,repository,service}.go` 既有 5 文件
- ❌ 未引入新 go.mod 依赖
- ❌ 未 commit / push / pull / 改 backlog / charter

## 2. API 契约

### 2.1 模板 CRUD

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/notifications/templates` | List with `?channel=&language=&enabled=&page=&page_size=&sort_by=&sort_dir=` |
| GET | `/api/v1/notifications/templates/:id` | GetByID |
| POST | `/api/v1/notifications/templates` | Create |
| PUT | `/api/v1/notifications/templates/:id` | Update (partial) |
| DELETE | `/api/v1/notifications/templates/:id` | Delete |

### 2.2 历史只读

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/notifications/history` | List with `?channel=&status=&template_id=&alarm_id=&page=&page_size=` |
| GET | `/api/v1/notifications/history/:id` | GetByID |

写操作通过 `HistoryService.Insert / MarkSent / MarkFailed / MarkDeadLetter`，由
`internal/notification/service.go` 中的 webhook/email/sms dispatcher 调用，不暴露给 HTTP。

## 3. 自跑验证

### 3.1 编译

```
$ go build ./...
(成功，无 warning)

$ go build ./internal/notification/...
(成功)
```

### 3.2 单元测试

```
$ go test -race -count=1 ./internal/notification/...
ok    github.com/omcgo/omcgo/internal/notification    1.518s
PASS (32 个测试用例全过)
```

主要覆盖：
- 模板 channel allow-list / 名字 trim+blank 拒绝 / variables 去重 / 重复 name → ErrAlreadyExists
- 历史 channel + status allow-list / 空 recipients 拒绝 / nil 入参拒绝
- HTTP handler 成功/400/404 路径全部覆盖
- 列名漂移防护（W1.5 教训：所有 SQL 共享 templateColumns / historyColumns 列表）

### 3.3 全项目编译

```
$ go build ./...
(成功)
```

### 3.4 已知预先存在的失败（与本任务无关）

`go test -race -count=1 ./...` 报告以下包失败，**全部存在于改动前的基线**，与本任务无关：

- `internal/acs` — TestDeviceRateLimiter_ConcurrentAccess (race condition in 现有 limiter)
- `internal/acs/rpc` — TestDownloadHandler
- `internal/acs/stun` — Integration tests 需要网络
- `internal/admin` — 需要 DB 集成测试
- `internal/alarm` — TestIntegration_FullPipeline_* 需要 DB/Redis
- `internal/core/model` — Test_ListRequest_Limit / Test_ListRequest_Limit_MutatesPageSize（pageSize 上限校验）

通过 `git stash --include-untracked` 后再次运行同样测试确认全部为预先存在的失败。

### 3.5 迁移连续性

```
$ bash scripts/check-migrations.sh
共发现 39 个迁移文件
编号区间：000001 → 000041
⚠️  编号不连续，缺失：000039, 000040
```

**预期行为**：000039/000040 由并行 sub-agent (Wave 2 其他任务) 占据，主会话整合时三个文件会一起合入，届时编号连续。

## 4. 章程 W2.A.4 后端 Pass 标准对照

| 标准 | 实测 | 状态 |
|---|---|---|
| `ls omcgo/internal/notification/template*.go ≥ 4` | 6（含 2 _test.go） | ✅ |
| `ls omcgo/internal/notification/history*.go ≥ 4` | 6（含 2 _test.go） | ✅ |
| `ls omcgo/migrations/000041*notification_templates_history.sql` 存在 | 1 | ✅ |
| `go build ./...` 通过 | yes | ✅ |
| `go test ./internal/notification/...` 全过 | 32/32 PASS | ✅ |

## 5. 主会话整合 TODO（Sub-agent 不做）

1. `cmd/app/provider/modules.go` — 注册 `TemplateRepository / TemplateService / TemplateHandler` 与 `HistoryRepository / HistoryService / HistoryHandler` 到 DI 容器
2. `cmd/app/router/router.go` — 在 `/api/v1` 下挂载两个 Handler.RegisterRoutes
3. `scripts/e2e_verify.sh` — 增补模板 + 历史 happy path 用例
4. `migrations/` — 与并行 sub-agent 的 000039/000040 一起 migrate-up 演练
5. 将既有 `service.go` 的 webhook/email dispatcher 改为通过 `HistoryService.Insert + MarkSent/Failed` 写入 history（非本任务范围）

## 6. 风险与权衡

- 列表分页：使用 OFFSET/LIMIT，10 万行级别 history 在按 `created_at DESC` 排序时性能可接受；100 万级建议改 keyset 分页（保留为未来 issue）。
- channel allow-list 重复定义于 service 与 binding tag — 本意是双层防御（HTTP 层 fail-fast + service 层防直调）。如未来扩展 channel，需要同步更新两处。
- HistoryRepository.UpdateStatus 接受 nil errorMessage / sentAt，调用方需保证语义一致性（MarkSent 不带 message，MarkFailed/DeadLetter 带 message 不带 sentAt）。
