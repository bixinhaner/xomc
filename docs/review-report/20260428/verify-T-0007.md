# T-0007 EmailDispatcher 接口 + SMTP 实现验证报告

> Wave 2 Block A.1（部分实现）
> 章程依据：`docs/methodology/AI承诺对峙清单.md#W2.A.1`
> 任务编号：T-0007（backlog）
> 工作树：`.claude/worktrees/agent-a5148380`，分支 `worktree-agent-a5148380`
> 验证日期：2026-04-28

## §1 任务摘要

### 范围（本任务实际交付）

实现 F04 告警通道的邮件派发能力（**仅 EmailDispatcher 内部 + SMTP 实现 + 单测**）：

1. `EmailDispatcher` 接口定义（`Dispatch(ctx, to, subject, body) error`）
2. `EmailConfig` 配置结构（Host/Port/Username/Password/From/UseTLS/UseSTARTTLS/Timeout）
3. `SMTPEmailDispatcher` 基于标准库 `net/smtp` + `crypto/tls` 的实现
4. `EmailMetrics` Prometheus counter `alarm_email_dispatches_total{result=success|failure|timeout}`
5. `noopEmailDispatcher` 占位实现（未注入时使用，避免散落 nil-check）
6. 单元测试（5 条 / -race 全过）

### 部分实现说明 — 与 T-0011 路径互斥

按主会话切片要求，**本任务严禁触碰** `filter_engine.go` 与 `filter_model.go`（并行任务 T-0011 在改动这两个文件，必须路径互斥）。所以以下接入工作**留给主会话整合**：

- `notify_email` filter action enum
- `filter_engine.go` 对 `notify_email` action 的 dispatchEmail 分支与 metrics 累加
- `EmailConfig` 从 `appconfig` 注入到 router DI
- 数据库 migration（如需把 SMTP 配置进表）
- HTML body / 多语言模板引擎（属 T-0043）

本任务交付的是「可即插即用的 EmailDispatcher 组件」，主会话整合时只需 `NewSMTPEmailDispatcher(cfg, logger, metrics)` 注入到 FilterEngine 即可。

## §2 改动文件清单

| # | 路径 | 类型 | 行数 | 说明 |
|---|------|------|------|------|
| 1 | `omcgo/internal/alarm/email_dispatcher.go` | 新建 | 256 | EmailDispatcher 接口 + SMTP 实现 + Metrics + noop 占位 |
| 2 | `omcgo/internal/alarm/email_dispatcher_test.go` | 新建 | 280 | 5 条单测 + in-memory mock SMTP server |
| 3 | `docs/review-report/20260428/verify-T-0007.md` | 新建 | 本文件 | 验证报告 |

**未触碰的关键文件（路径互斥保证）**：

```
$ git status --short
?? .wave-progress.log
?? .wave-status.txt
?? docs/review-report/20260428/verify-T-0007.md
?? omcgo/internal/alarm/email_dispatcher.go
?? omcgo/internal/alarm/email_dispatcher_test.go
```

无任何 `M`（modified）状态的既有文件 → 与 T-0011 在 `filter_engine.go` / `filter_model.go` 上零冲突。

## §3 自跑命令完整输出

### 3.1 全仓编译

```bash
$ cd omcgo && go build ./...
（无输出 → 编译通过）
```

### 3.2 包级 vet

```bash
$ cd omcgo && go vet ./internal/alarm/...
（无输出 → 通过）
```

### 3.3 新文件存在性

```bash
$ ls omcgo/internal/alarm/email_dispatcher*.go
omcgo/internal/alarm/email_dispatcher.go
omcgo/internal/alarm/email_dispatcher_test.go
```

### 3.4 新单测 -race -v 全输出

```bash
$ cd omcgo && go test -run "TestSMTPEmailDispatcher_|TestBuildEmailMessage" -race -v ./internal/alarm/...
=== RUN   TestSMTPEmailDispatcher_Dispatch_Success
=== PAUSE TestSMTPEmailDispatcher_Dispatch_Success
=== RUN   TestSMTPEmailDispatcher_Dispatch_Timeout
=== PAUSE TestSMTPEmailDispatcher_Dispatch_Timeout
=== RUN   TestSMTPEmailDispatcher_Dispatch_AuthFail
=== PAUSE TestSMTPEmailDispatcher_Dispatch_AuthFail
=== RUN   TestSMTPEmailDispatcher_Dispatch_ParamValidation
=== PAUSE TestSMTPEmailDispatcher_Dispatch_ParamValidation
=== RUN   TestBuildEmailMessage
=== PAUSE TestBuildEmailMessage
=== CONT  TestSMTPEmailDispatcher_Dispatch_Timeout
=== CONT  TestSMTPEmailDispatcher_Dispatch_ParamValidation
=== CONT  TestBuildEmailMessage
=== CONT  TestSMTPEmailDispatcher_Dispatch_AuthFail
=== CONT  TestSMTPEmailDispatcher_Dispatch_Success
=== RUN   TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_to
=== PAUSE TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_to
=== RUN   TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_addr_in_list
=== PAUSE TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_addr_in_list
=== RUN   TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_subject
--- PASS: TestBuildEmailMessage (0.00s)
=== PAUSE TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_subject
=== CONT  TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_subject
=== CONT  TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_addr_in_list
=== CONT  TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_to
--- PASS: TestSMTPEmailDispatcher_Dispatch_ParamValidation (0.00s)
    --- PASS: TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_subject (0.00s)
    --- PASS: TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_addr_in_list (0.00s)
    --- PASS: TestSMTPEmailDispatcher_Dispatch_ParamValidation/empty_to (0.00s)
--- PASS: TestSMTPEmailDispatcher_Dispatch_AuthFail (0.00s)
--- PASS: TestSMTPEmailDispatcher_Dispatch_Success (0.00s)
--- PASS: TestSMTPEmailDispatcher_Dispatch_Timeout (0.20s)
PASS
ok  	github.com/omcgo/omcgo/internal/alarm	2.251s
```

5 条新单测全部 -race PASS。

### 3.5 alarm 包整体回归（跳过 4 条预存 race 用例）

```bash
$ cd omcgo && go test -race -skip "^TestIntegration_FullPipeline_" ./internal/alarm/...
ok  	github.com/omcgo/omcgo/internal/alarm	(cached)
```

### 3.6 alarm 包整体（无 -race，含 integration）

```bash
$ cd omcgo && go test ./internal/alarm/...
ok  	github.com/omcgo/omcgo/internal/alarm	2.016s
```

### 3.7 staticcheck

```bash
$ /Users/cb/go/bin/staticcheck ./internal/alarm/... | grep -v fips140only
（无输出 → 包级 staticcheck clean）
```

### 3.8 关于 `TestIntegration_FullPipeline_*` 预存 race（与本任务无关）

执行 `go test -race ./internal/alarm/...` 时 4 个 `TestIntegration_FullPipeline_*` 用例会因 mockAlarmStore 的 map 访问触发 race detector。已通过 `git stash -u` 隔离我两个新文件后在 clean baseline 上复现确认，**这是 W1.5 既存问题，与 T-0007 无关**：

```bash
$ git stash -u
Saved working directory and index state WIP on worktree-agent-a5148380: 1fd41095 ...

$ cd omcgo && go test -race -run "TestIntegration_FullPipeline_RaceCondition_NewAlarmNotYetProcessed" ./internal/alarm/...
==================
WARNING: DATA RACE
Previous write at 0x00c0004a83f0 by goroutine 46:
  github.com/omcgo/omcgo/internal/alarm.(*mockAlarmStore).SaveActive()
      .../engine_test.go:27 +0x4c
  github.com/omcgo/omcgo/internal/alarm.(*AlarmEngine).Process()
      .../engine.go:123 +0xe70
  ...
--- FAIL: TestIntegration_FullPipeline_RaceCondition_NewAlarmNotYetProcessed (0.10s)
FAIL

$ git stash pop
（恢复我的两个新文件）
```

故障点位于 `engine_test.go:27` 的 `mockAlarmStore.SaveActive()`，与 EmailDispatcher 完全无交互。**T-0007 改动既不引入新 race 也不修复此 race**——属另立 issue。

## §4 章程 W2.A.1 grep 命中说明

### 4.1 章程要求 vs 实际交付

| 章程要求（T-0007 部分实现切片） | 交付证据 |
|----|----|
| 新建 `email_dispatcher.go` | `ls omcgo/internal/alarm/email_dispatcher.go` ✓ |
| 新建 `email_dispatcher_test.go` | `ls omcgo/internal/alarm/email_dispatcher_test.go` ✓ |
| `EmailDispatcher` 接口 | `email_dispatcher.go:21-25` ✓ |
| `EmailConfig` 结构 | `email_dispatcher.go:33-42` ✓ |
| `SMTPEmailDispatcher` 实现 | `email_dispatcher.go:71-78` + `Dispatch` 方法 ✓ |
| 用 `net/smtp` + `crypto/tls`，不引入 third-party | `import (... "crypto/tls" ... "net/smtp" ...)`，未改 `go.mod` ✓ |
| `fmt.Errorf("smtp dispatch: %w", err)` 包装 | `email_dispatcher.go:120/127/132` 等 ✓ |
| `ctx.Done()` 取消传播 | `email_dispatcher.go:106-118` select 块 ✓ |
| to 为空 / subject 为空 → 参数错 | `validateEmailParams` + `TestSMTPEmailDispatcher_Dispatch_ParamValidation` ✓ |
| RFC 822 格式（From/To/Subject/Date/MIME-Version/Content-Type） | `buildEmailMessage` + `TestBuildEmailMessage` 断言 7 个头字段 ✓ |
| Prometheus counter `alarm_email_dispatches_total{result=...}` | `NewEmailMetrics` line 51-65，三种 result label ✓ |
| 至少 3 条 -race 单测 | 实交 5 条（含 1 条参数验证 + 1 条消息构造）-race 全过 ✓ |
| Success：mock SMTP 收到正确 EHLO/MAIL FROM/RCPT TO/DATA 序列 | `TestSMTPEmailDispatcher_Dispatch_Success` 断言命令前缀 + DATA body ✓ |
| Timeout：200ms 超时，250ms 内返错 | `TestSMTPEmailDispatcher_Dispatch_Timeout`：实测耗时 0.20s，断言 `< 600ms` + `errors.Is(err, context.DeadlineExceeded)` ✓ |
| AuthFail：535 → dispatcher 返认证错 | `TestSMTPEmailDispatcher_Dispatch_AuthFail` 断言 `err.Error() contains "smtp auth"` ✓ |
| 不修改 `filter_engine.go` | `git status --short` 无 `M` 文件 ✓ |
| 不修改 `filter_model.go` | 同上 ✓ |
| 不动 migration / config DI / 模板引擎 | 同上 ✓ |
| W1.5 既有 8 alarm 单测保持 PASS | 跳过预存 race 后整包 PASS（见 §3.5）✓ |

### 4.2 关键 grep 验证

```bash
$ grep -n "^func.*Dispatcher" omcgo/internal/alarm/email_dispatcher.go
21:type EmailDispatcher interface {
71:type SMTPEmailDispatcher struct {
77:func NewSMTPEmailDispatcher(cfg EmailConfig, logger *zap.Logger, metrics *EmailMetrics) *SMTPEmailDispatcher {
98:func (d *SMTPEmailDispatcher) Dispatch(ctx context.Context, to []string, subject, body string) error {
（以及 noopEmailDispatcher 等）

$ grep -c "^func Test" omcgo/internal/alarm/email_dispatcher_test.go
5

$ grep "alarm_email_dispatches_total" omcgo/internal/alarm/email_dispatcher.go
			Name: "alarm_email_dispatches_total",
```

## §5 主会话整合 checklist（供 T-0007 终态收敛）

主会话拿到本 worktree 后整合工作：

- [ ] 在 `filter_model.go` 增加 `FilterActionNotifyEmail = "notify_email"` enum
- [ ] 在 `filter_engine.go` 注入 `EmailDispatcher` 字段 + dispatchEmail 分支
- [ ] `appconfig` 增加 `email:` 段（Host/Port/From/Username/Password/UseTLS/UseSTARTTLS/Timeout）
- [ ] router DI：`NewSMTPEmailDispatcher(cfg.Email, logger, NewEmailMetrics(prom.GlobalReg))`
- [ ] FilterEngine 单测覆盖 `notify_email` 分支（mock EmailDispatcher）
- [ ] backlog T-0007 状态由 in-progress → done
- [ ] charter §W2.A.1 三盘计分

## §6 风险与遗留

| 风险 | 影响 | 缓解 |
|------|------|------|
| 预存 mockAlarmStore race（W1.5 issue） | -race 模式 4 条 integration 用例 fail | 与 T-0007 无关；建议另立 backlog item 修 mockAlarmStore（map → sync.Map 或加 Mutex） |
| AUTH PLAIN 明文凭据 | 仅在 UseTLS / UseSTARTTLS 时安全 | 实现已支持二者；调用方应配置 TLS |
| net/smtp 标准库已 frozen | 长期需迁移 go-mail 等社区库 | 当前为冒烟版本；T-0043 全量化时再评估 |
| Prometheus registry 注入由调用方决定 | 单测用独立 NewRegistry()，生产用全局 | 与 webhook_dispatcher.go 同模式，主会话 DI 时统一 |

## §7 心跳

- `.wave-progress.log`：3 行（init / build / tests-pass）
- `.wave-status.txt`：DONE
