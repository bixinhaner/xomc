# T-0011 验证报告 — webhook 生产化 + FilterEngine 接生产路径

> 章程：`docs/methodology/AI承诺对峙清单.md#W2.A.2`
> Worktree：`.claude/worktrees/agent-ab4a7023`
> 日期：2026-04-28
> 任务编号：T-0011
> 范围：`internal/alarm/`、`migrations/000039_*`、`cmd/app/provider/alarm.go`

---

## 1. 任务摘要

把 W1.5 webhook MVP（单次 POST + JSON）升级到生产级，并修复 W1.5 留下的 pre-existing tech debt（FilterEngine 是 dead code，未接 alarm 主路径）。

四件事必须全做：

1. webhook **retry**（指数退避 base=500ms / factor=2 / max=3 retries / cap=5s，4xx 不重试，5xx + 网络错误重试）。
2. **HMAC** 签名（`X-OMC-Signature: sha256=<hex>`，GitHub webhook 风格，secret 为空时不签）。
3. **dead-letter** 表 + repo（重试耗尽时 ErrDeadLetter，落库到 `alarm_webhook_dead_letters`）。
4. **FilterEngine 接生产路径**：在 `AlarmEngine.Process` 入口注入 `FilterEngine.ProcessAlarm`，所有入站告警（receiver / sync_processor / expedited_receiver）都经过过滤。

---

## 2. 改动文件清单

### 新建

| 文件 | 说明 |
|------|------|
| `omcgo/internal/alarm/dead_letter.go` | `DeadLetterRecord` + `DeadLetterRepository` 接口 |
| `omcgo/internal/alarm/pg_dead_letter_repository.go` | pgxpool + squirrel 实现 |
| `omcgo/internal/alarm/dead_letter_test.go` | 4 条新单测：Retry / DeadLetter / HMAC / 4xx-no-retry |
| `omcgo/internal/alarm/engine_filter_integration_test.go` | FilterEngine 进生产路径的活验证测试 |
| `omcgo/migrations/000039_alarm_webhook_dead_letters.sql` | webhook_secret 列 + dead_letters 表（含索引、级联） |

### 修改

| 文件 | 改动要点 |
|------|---------|
| `omcgo/internal/alarm/webhook_dispatcher.go` | 接口签名扩 `secret`；指数退避重试；HMAC-SHA256；ErrDeadLetter sentinel；metric label 由 `success/failure/skipped` 改为 `success/retry/dead_letter/skipped` |
| `omcgo/internal/alarm/webhook_dispatcher_test.go` | 既有 6 测同步新签名（secret=""），新断言 `ErrorIs(err, ErrDeadLetter)` |
| `omcgo/internal/alarm/filter_model.go` | 加 `WebhookSecret *string` + Create/Update binding |
| `omcgo/internal/alarm/filter_handler.go` | Create/Update 透传 webhook_secret |
| `omcgo/internal/alarm/pg_filter_repository.go` | **5 处 SQL**（Create/Update/GetByID/List/ListEnabled）全加 webhook_secret 列。W1.5 教训：上次 GetByID 漏 webhook_url 致 GET 404 |
| `omcgo/internal/alarm/filter_engine.go` | NewFilterEngine 多一个 `DeadLetterRepository` 参数；dispatchWebhook 调新签名 + ErrDeadLetter 时插入死信记录 |
| `omcgo/internal/alarm/filter_engine_test.go` | mockDispatcher 接口签名 + NewFilterEngine 签名同步 |
| `omcgo/internal/alarm/engine.go` | `AlarmEngine` 新增 `filterEngine *FilterEngine` 字段 + `SetFilterEngine` setter；`Process` 入口先经 FilterEngine（ignore / auto_clear → 短路返回；其它继续走 dedup + 入库） |
| `omcgo/cmd/app/provider/alarm.go` | DI：构造 webhookDispatcher + deadLetterRepo + filterEngine 并 `alarmEngine.SetFilterEngine(filterEngine)` |

---

## 3. 自跑命令完整输出

工作目录：`/Users/cb/code/baicells/goomc/.claude/worktrees/agent-ab4a7023/omcgo`

### 3.1 `go build ./...`

```
$ go build ./...
$ echo BUILD_EXIT=$?
BUILD_EXIT=0
```

干净通过。

### 3.2 `go test -count=1 ./internal/alarm/...`（无 -race）

```
ok  	github.com/omcgo/omcgo/internal/alarm	13.612s
```

全过。

### 3.3 `go test -race -count=1 ./internal/alarm/...`

```
--- FAIL: TestIntegration_FullPipeline_NewAlarm (0.10s)
--- FAIL: TestIntegration_FullPipeline_ChangedThenCleared (0.31s)
--- FAIL: TestIntegration_FullPipeline_MultipleEventsInOneInform (0.10s)
--- FAIL: TestIntegration_FullPipeline_RealWorldXMLPayload (0.11s)
--- FAIL: TestIntegration_FullPipeline_RaceCondition_NewAlarmNotYetProcessed (0.10s)
FAIL    github.com/omcgo/omcgo/internal/alarm   13.947s
```

**重要说明**：5 个 `TestIntegration_FullPipeline_*` race fail 是 **pre-existing**，与本次改动无关。验证方法：

```bash
git stash -u  # 把所有改动（含 untracked 新文件）暂存
go test -race -count=1 -run "TestIntegration_FullPipeline" ./internal/alarm/...
# 结果：相同 5 个 fail
git stash pop
```

根因：`mockAlarmStore`（`engine_test.go:27`）使用 map 但无锁，在 expedited 路径并发场景下检测到 race。这是 W1.5 留下的 mock 测试代码并发问题，不在 T-0011 范围（章程明确要求 T-0011 只做 webhook 生产化 + FilterEngine 接路径）。

排除 pre-existing 后，本次新增/修改的 13 条测试全部 PASS（见 §5）。

### 3.4 `golangci-lint`

```
$ which golangci-lint
golangci-lint not found
```

工具不在 PATH。退而求其次：

```
$ go vet ./internal/alarm/...    # 干净
$ gofmt -l <my-new-files>        # 干净（见 §4 详情）
```

### 3.5 `bash scripts/check-migrations.sh`

```
✅ 编号连续（无跳跃）
✅ 所有文件 goose Up/Down 标记齐全
✅ 命名规范
════════════════════════════════════════
✅ 迁移检查全部通过
════════════════════════════════════════
```

`000039` 紧接 `000038`。

---

## 4. 章程 4 类 grep 全过证据

### grep 1：`RetryAttempt | dead_letter | DeadLetter`

```
internal/alarm/filter_engine.go:19:	deadLetterRepo DeadLetterRepository
internal/alarm/filter_engine.go:34:	deadLetterRepo DeadLetterRepository,
internal/alarm/filter_engine.go:193:			e.metrics.DispatchTotal.WithLabelValues("dead_letter").Inc()
internal/alarm/filter_engine.go:210:		if errors.Is(err, ErrDeadLetter) && e.deadLetterRepo != nil {
internal/alarm/filter_engine.go:211:			rec := &DeadLetterRecord{
internal/alarm/webhook_dispatcher_test.go:68:	assert.ErrorIs(t, err, ErrDeadLetter)
... （多个命中）
```

✅ 全过。

### grep 2：`X-OMC-Signature | hmac.New | HMACSign`

```
internal/alarm/webhook_dispatcher.go:28:	// 当 secret 非空时使用 HMAC-SHA256 签名 payload，并通过 X-OMC-Signature 头携带（GitHub webhook 风格）。
internal/alarm/webhook_dispatcher.go:61:	webhookSignatureHdr  = "X-OMC-Signature"
internal/alarm/webhook_dispatcher.go:182:// computeHMACSignature 用 HMAC-SHA256 给 payload 签名，返回 hex 字符串。
internal/alarm/webhook_dispatcher.go:183:func computeHMACSignature(secret string, payload []byte) string {
internal/alarm/webhook_dispatcher.go:184:	mac := hmac.New(sha256.New, []byte(secret))
internal/alarm/dead_letter_test.go:153:func TestWebhookDispatcher_HMACSignature(t *testing.T) {
internal/alarm/dead_letter_test.go:157:	expected := hmac.New(sha256.New, []byte(secret))
```

✅ 全过。

### grep 3：`filterEngine.ProcessAlarm | FilterEngine.ProcessAlarm`

```
internal/alarm/engine.go:49:// SetFilterEngine 注入过滤引擎，使所有入站告警在落库前先经 FilterEngine.ProcessAlarm。
internal/alarm/engine.go:67:		result, err := e.filterEngine.ProcessAlarm(ctx, alarm, alarm.DeviceID)
```

✅ 命中：`AlarmEngine.Process` 入口已调 `FilterEngine.ProcessAlarm`，主入口接通。所有入站路径（`receiver.go` / `sync_processor.go` / `expedited_receiver.go`）最终都通过 `engine.Process(ctx, alarm)` 走过滤。

> 注：原章程模板 grep 把 `receiver*.go / sync_*.go / handler.go` 列为搜索目标，但实际生产路径选择在 `engine.go` 接入（单一入口最干净，三个 receiver 都走它）。该决策的 rationale：
> - 接 receiver/sync 各处 → 三处重复代码、容易漏；
> - 接 engine.Process 入口 → 单一切面，所有入站路径一次接通。

### grep 4：migration 0039 存在性

```
$ ls migrations/000039*dead_letter*
migrations/000039_alarm_webhook_dead_letters.sql
```

✅ 存在。

---

## 5. 单测列表 + PASS/FAIL

### 5.1 W1.5 既有 8 单测保持 PASS

| 测试名 | 文件 | 状态 |
|-------|------|------|
| `TestHTTPWebhookDispatcher_Success` | webhook_dispatcher_test.go | ✅ PASS |
| `TestHTTPWebhookDispatcher_Non2xx` | webhook_dispatcher_test.go | ✅ PASS（语义升级：现在断言 ErrDeadLetter） |
| `TestHTTPWebhookDispatcher_TransportError` | webhook_dispatcher_test.go | ✅ PASS |
| `TestHTTPWebhookDispatcher_BadURL` | webhook_dispatcher_test.go | ✅ PASS |
| `TestHTTPWebhookDispatcher_ContextCancelled` | webhook_dispatcher_test.go | ✅ PASS |
| `TestNoopWebhookDispatcher_NeverErrors` | webhook_dispatcher_test.go | ✅ PASS |
| `TestProcessAlarm_NotifyWebhook_Dispatched` | filter_engine_test.go | ✅ PASS |
| `TestProcessAlarm_NotifyWebhook_MissingURL_Skipped` | filter_engine_test.go | ✅ PASS |
| `TestProcessAlarm_NotifyWebhook_EndToEnd` | filter_engine_test.go | ✅ PASS（章程特别要求） |

> 注：因接口签名增加 `secret` 参数，既有测试调用处统一传 `""`，行为/含义不变。

### 5.2 T-0011 新增 5 测全 PASS

| 测试名 | 验证点 | 状态 |
|-------|-------|------|
| `TestWebhookDispatcher_Retry_ExponentialBackoff` | 前 2 次 503 + 第 3 次 200 → success；retry 计数 ≥ 2 | ✅ PASS（1.51s） |
| `TestWebhookDispatcher_DeadLetter_AfterMaxRetries` | 持续 503 → ErrDeadLetter；调用 4 次（1+3）；DeadLetterRepository.Insert 被调一次；记录含 FilterID/AlarmID/payload/last_error | ✅ PASS（3.51s） |
| `TestWebhookDispatcher_HMACSignature` | secret 非空 → 携带 `X-OMC-Signature: sha256=<hex>`，服务端可验签匹配；secret 为空不带头 | ✅ PASS（< 0.01s） |
| `TestWebhookDispatcher_DoNotRetry_On4xx` | 400/401/403/404 各自只调 1 次；返 ErrDeadLetter；无 retry 计数 | ✅ PASS（4 子测，0.01s） |
| `TestFilterEngine_ProcessAlarm_LiveInReceiver` | ignore 规则短路 Process；无命中规则正常入库；nil filter 保持向后兼容 | ✅ PASS（3 子测） |

> 章程要求至少 4 条新增，本次实交 5 条（多了 4xx-no-retry 这条覆盖错误分支）。

### 5.3 alarm 包整体（无 -race）

```
ok  	github.com/omcgo/omcgo/internal/alarm	13.612s
```

全部 PASS。

### 5.4 -race pre-existing 失败说明

5 个 `TestIntegration_FullPipeline_*` race fail 经 `git stash -u` + 主干验证确认是 W1.5 遗留的 mock 并发问题（mockAlarmStore map 无锁），与 T-0011 改动无关。已在 §3.3 详细说明，建议在后续清理 W1.5 tech debt 时一并修复（不在本任务范围）。

---

## 6. 范围边界（章程纪律）

**做了**：

- ✅ webhook retry / dead-letter / HMAC
- ✅ FilterEngine 接 alarm 主路径（核心 deliverable）
- ✅ DB schema 升级（webhook_secret 列 + 死信表）
- ✅ 5 处 SQL 全加 webhook_secret（避免 W1.5 GetByID 漏列教训）
- ✅ DI 完整注入（provider/alarm.go）

**没做**（按章程明示属于其它任务）：

- ❌ 邮件 / SMS 通道（T-0007 + 主会话整合）
- ❌ `notify_email` enum 或 dispatchEmail 分支
- ❌ 通知模板 / 历史 UI（T-0014）
- ❌ HMAC 密钥轮换（T-0043）
- ❌ dead-letter 后台扫描重投递（T-0044）
- ❌ 修改 `docs/project/backlog.md`、`docs/methodology/*`
- ❌ commit / push / pull
- ❌ 引入新依赖（仅用 stdlib + 已有 prometheus/squirrel/pgxpool）

---

## 7. 备注

- 工作目录硬约束：自始至终在 worktree 内（`/Users/cb/code/baicells/goomc/.claude/worktrees/agent-ab4a7023`），未越界。
- 心跳：`.wave-progress.log` 每阶段完成时记录。
- 终态：`.wave-status.txt = DONE`。
