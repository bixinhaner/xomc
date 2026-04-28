# T-0044 验证报告 — notification 模块覆盖率提升

**任务**：W2.A.5 / T-0044 — `internal/notification/` 测试覆盖率从 27.6% → ≥ 70%
**Owner**：sub-agent (worktree-agent-a97f4040)
**日期**：2026-04-28
**状态**：DONE  ✅ — **78.4% 覆盖率达成**（目标 ≥70%）

---

## §1 baseline

T-0044 任务卡的 baseline 数据（27.6%）来源于主分支历史快照。本 worktree 当前分支 `worktree-agent-a97f4040` 上 notification 模块只有 5 个生产源文件、零测试文件，因此实测 baseline 为 **0.0%**：

```
go test -coverprofile=cov.out ./internal/notification/...
ok  	github.com/omcgo/omcgo/internal/notification    coverage: 0.0% of statements
```

5 个生产源文件（共 647 行）：

| 文件 | 行数 | 职责 |
|------|------|------|
| `model.go` | 55 | 类型定义（NotificationType / Priority / Notification / NotificationFilter） |
| `repository.go` | 20 | Repository 接口（7 个方法） |
| `pg_repository.go` | 253 | PostgreSQL 实现（pgxpool + squirrel） |
| `service.go` | 146 | 业务层（List/Get/Create/Mark/Delete + SSE 推送 + Send/Broadcast 便利方法） |
| `handler.go` | 173 | HTTP 路由（5 个 GET/PUT/DELETE 端点） |

无任何低覆盖文件——零测试就是初始状态。

---

## §2 改动文件清单

仅新增 `_test.go` 文件，未改动任何生产代码，未新增 `go.mod` 依赖。

| 新文件 | 行数 | 内容 |
|------|------|------|
| `internal/notification/mock_repository_test.go` | 168 | `mockRepository` — 内存版 Repository 实现（含失败注入字段：listErr/getByIDErr/createErr 等），`errBoom` sentinel error |
| `internal/notification/service_test.go` | 268 | 25 个 service 层测试（mock repo + 真实 events.MessageHub + noopMessageStore） |
| `internal/notification/handler_test.go` | 287 | 23 个 handler 层测试（gin httptest + 中间件注入 username 模拟鉴权） |
| `internal/notification/pg_repository_test.go` | 175 | 14 个 pg_repository 测试（dead pool + fakeRow scan target + helper 单测） |

总计 **~898 行测试代码**，**62 个测试用例**（含子测试）。

测试模式与既有模块（`internal/alarm/engine_test.go` `mockAlarmStore` 模式、`internal/dashboard/handler_test.go` gin 路由模式）保持一致。

---

## §3 最终覆盖率

```
go test -race -count=1 -coverprofile=cov_notification.out ./internal/notification/...
ok  	github.com/omcgo/omcgo/internal/notification    1.777s    coverage: 78.4% of statements

go tool cover -func=cov_notification.out | tail -1
total:                                                  (statements)            78.4%
```

**78.4% ≥ 70.0%  ✅**（章程 W2.A.5 Pass 标准达成）。

按文件细分：

| 文件 / 函数 | 覆盖率 | 备注 |
|------|------|------|
| `handler.go` 全部 7 个函数 | 100.0% | List / GetUnreadCount / MarkRead / MarkAllRead / Delete / NewHandler / RegisterRoutes |
| `service.go` 9 个函数（除 pushSSEEvent） | 100.0% | NewService / List / GetByID / CreateNotification / MarkRead / MarkAllRead / GetUnreadCount / Delete / SendNotification / SendGlobalNotification / CreateAndBroadcast |
| `service.go` `pushSSEEvent` | 57.1% | SSE channel-full 分支需要真实 SSE 订阅者，单测覆盖较难（已通过 storeErr 路径覆盖大部分） |
| `pg_repository.go` `NewPgRepository` / `scanNotification` / `joinColumns` | 100.0% | 直接单测 |
| `pg_repository.go` 7 个 CRUD 方法 | 38–75% | 通过 dead pool 覆盖到 SQL 构建分支；SQL 执行 / RowsAffected 分支需真实 DB |

### §3.1 triage

无生产代码 bug 发现。pg_repository 各方法尾段（执行 SQL 之后、scan 错误处理之前）的分支需要真实 PostgreSQL 才能触达，按章程 W2.A.5 「只看覆盖率指标」标准，本任务范围不引入真 DB 集成测试（已超出 70% 阈值即合格）。

---

## §4 关键测试列表

### 4.1 service 层（25 个）

| 测试 | 路径 |
|------|------|
| `TestNewService_Defaults` | 成功 |
| `TestService_List_OK` | 成功 |
| `TestService_List_Error` | 失败（repo 注入 errBoom） |
| `TestService_GetByID_OK` | 成功 |
| `TestService_GetByID_NotFound` | 失败（ErrNotFound） |
| `TestService_CreateNotification_NoHub` | 成功（无 SSE） |
| `TestService_CreateNotification_WithHub` | 成功（验证 SSE 持久化到 store） |
| `TestService_CreateNotification_RepoError` | 失败（create 抛错） |
| `TestService_MarkRead_OK` / `_NotFound` | 成功 + 失败 |
| `TestService_MarkAllRead_OK` / `_RepoError` | 成功 + 失败 |
| `TestService_GetUnreadCount` / `_RepoError` | 成功 + 失败 |
| `TestService_Delete_OK` / `_NotFound` / `_WrongOwner` | 成功 + 2 个失败路径 |
| `TestService_SendNotification_OK` / `_RepoError` | 成功 + 失败 |
| `TestService_SendGlobalNotification` / `_PartialFailure_NoAbort` | 成功 + 部分失败聚合 |
| `TestService_CreateAndBroadcast_NoHub` / `_WithHub` / `_RepoError` | 3 路径 |
| `TestService_pushSSEEvent_StoreError_LogsButNoPanic` | 失败但不 panic |
| `TestNotificationConstants` | 常量 wire 值 |

### 4.2 handler 层（23 个）

| 测试 | 验证点 |
|------|------|
| `TestNewHandler_Basic` | 构造器 |
| `TestHandler_List_OK` | 200 + 正确总数 |
| `TestHandler_List_Unauthorized` | 401（无 username） |
| `TestHandler_List_FilterByType` / `_FilterByIsRead_True` / `_FilterByIsRead_False` | query 过滤 |
| `TestHandler_List_BadQueryParams` | 400（page=0 触发 binding 校验） |
| `TestHandler_List_RepoError` | 500（HTTPStatusFromError 兜底） |
| `TestHandler_GetUnreadCount_OK` / `_Unauthorized` / `_RepoError` | 200/401/500 |
| `TestHandler_MarkRead_OK` / `_BadID` / `_Unauthorized` / `_NotFound` | 204 / 400 / 401 / 404 |
| `TestHandler_MarkAllRead_OK` / `_Unauthorized` / `_RepoError` | 204/401/500 |
| `TestHandler_Delete_OK` / `_BadID` / `_Unauthorized` / `_NotFound` | 204/400/401/404 |
| `TestHandler_RegisterRoutes_AllPathsReachable` | 5 路由全部注册 |

### 4.3 pg_repository 层（14 个）

| 测试 | 验证点 |
|------|------|
| `TestNewPgRepository` | 构造器赋值正确 |
| `TestJoinColumns` | 3 子用例：empty / single / multiple |
| `TestScanNotification_OK` | 11 列扫描成功 |
| `TestScanNotification_Error` | 扫描错误传播 |
| `TestPgRepository_List_DeadPool` | 默认参数路径 |
| `TestPgRepository_List_FullFilters_DeadPool` | 全部过滤器 + 自定义 sort_by/page |
| `TestPgRepository_List_DisallowedSortBy_DeadPool` | sort_by 非白名单回退分支 |
| `TestPgRepository_GetByID_DeadPool` | SQL 构建 + 执行错误 |
| `TestPgRepository_Create_DeadPool` | INSERT 构建 |
| `TestPgRepository_MarkRead_DeadPool` | UPDATE WHERE id+user |
| `TestPgRepository_MarkAllRead_DeadPool` | UPDATE WHERE user+is_read=false |
| `TestPgRepository_GetUnreadCount_DeadPool` | COUNT 构建 |
| `TestPgRepository_Delete_DeadPool` | DELETE WHERE id+user |

### 4.4 跑通命令

```bash
cd omcgo
go build ./...                                                        # ✅ 通过
go vet ./internal/notification/...                                    # ✅ 通过
go test -race -count=1 -coverprofile=cov_notification.out \
    ./internal/notification/...                                       # ✅ 1.777s
go tool cover -func=cov_notification.out | tail -1
# total:                                                  (statements)            78.4%
```

---

## §5 章程 W2.A.5 合规

- [x] 仅改 `internal/notification/*_test.go` + `mock_*.go` 同包文件
- [x] 未改 `cmd/`、其他模块、既有生产代码
- [x] 未新增 `go.mod` 依赖
- [x] 未删既有测试
- [x] 未 commit / push / pull
- [x] `go vet` 干净
- [x] `-race` 跑通
- [x] 覆盖率 78.4% ≥ 70%

任务 DONE。
