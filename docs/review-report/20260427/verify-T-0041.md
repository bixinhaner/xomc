# Verify Report — T-0041 / W1.4 Rate Limit Middleware

worktree: `.claude/worktrees/agent-ae2732f9cdbd1d844`
branch: `worktree-agent-ae2732f9cdbd1d844`

## S2 设计备忘

- 选型：`golang.org/x/time/rate` 的 token bucket（stdlib 友好，已在 go.mod direct 依赖中，无需调 go mod tidy）。
- 粒度：per-IP（取 `c.ClientIP()`）。每个 IP 维护独立 limiter，互不干扰。
- 中间件签名：`func RateLimit(cfg RateLimitConfig) gin.HandlerFunc`。配置含 `RatePerSecond float64` / `Burst int` / `IdleTTL` / `SweepInterval` / `Skipper func(*gin.Context) bool` / `Registerer prometheus.Registerer`。
- 默认值：`RatePerSecond=100`、`Burst=200`、`IdleTTL=5min`、`SweepInterval=1min`，与 OMC 单实例 app 服务一般负载（管理面 REST，多由前端调用）匹配；`RatePerSecond <=0` 视为 no-op，避免错配置直接全拒。
- 响应：被拒返回 `429` + JSON `{"code":"RATE_LIMITED","message":"too many requests, please retry later"}`，并 Inc Prometheus counter `http_ratelimit_rejections_total{path}`，写一行 `zap.Warn("ratelimit rejected", client_ip, method, path, rate_per_second, burst)`。
- 内存策略：`sync.Map` + 后台 goroutine 周期清扫（每 `SweepInterval` 扫一次，剔除超过 `IdleTTL` 未访问的条目），`lastSeen` 用 `atomic.Int64` 存 unix-nano 实现读路径无锁。属于"够用即止"方案，未来如需 LRU / 分布式可平滑替换。
- 路由接入点：`omcgo/cmd/app/provider/router.go` 的 `setupMiddleware`，紧跟 `RequestLogger`、在 `PrometheusMetrics` / `SecurityHeaders` 之前；`Skipper` 放行 `/healthz`、`/readyz`、`/metrics` 探针。
- ACS 是否接入：本次仅接入 app（charter 验证命令明确针对 `cmd/app/.../router.go`），ACS 是 net/http stdlib + 自有 admission/per-device 限流（见 `internal/acs/`），结构差异较大，不在本任务范围；后续可单立 task 再做。
- 路径差异说明：W1.4 charter 写 `omcgo/cmd/app/router/router.go`，实际仓库里 router.go 位于 `omcgo/cmd/app/provider/router.go`；grep "RateLimit\|ratelimit" 在 `omcgo/cmd/app/` 下命中，charter 验证语义上达成。

## S3 改动文件清单

| 文件 | 行数 | 变更类型 |
|------|-----:|---------|
| `omcgo/internal/core/middleware/ratelimit.go` | 176 | 新增 |
| `omcgo/internal/core/middleware/ratelimit_test.go` | 218 | 新增 |
| `omcgo/cmd/app/provider/router.go` | +13 / -0 | 修改（在 `setupMiddleware` 中插入 `r.Use(middleware.RateLimit(...))`） |

> go.mod / go.sum 无变化（`golang.org/x/time v0.12.0` 已是 direct 依赖）。

## S4 验证命令实跑输出

### `go build ./...`

```
$ cd omcgo && go build ./...
（无输出 → 0 error）
```

### `go test ./internal/core/middleware/... -count=1 -race -v`

关键尾部输出（已剪去其它中间件用例，只保留 ratelimit 部分 + 总结）：

```
=== RUN   TestRateLimit_AllowsWithinLimit
--- PASS: TestRateLimit_AllowsWithinLimit (0.00s)
=== RUN   TestRateLimit_RejectsBeyondBurst
--- PASS: TestRateLimit_RejectsBeyondBurst (0.00s)
=== RUN   TestRateLimit_PerIPIsolation
--- PASS: TestRateLimit_PerIPIsolation (0.00s)
=== RUN   TestRateLimit_SkipperBypassesLimit
--- PASS: TestRateLimit_SkipperBypassesLimit (0.00s)
=== RUN   TestRateLimit_BurstAllowsImmediateConcurrency
--- PASS: TestRateLimit_BurstAllowsImmediateConcurrency (0.00s)
=== RUN   TestRateLimit_ZeroRateActsAsNoOp
--- PASS: TestRateLimit_ZeroRateActsAsNoOp (0.00s)
=== RUN   TestRateLimit_DefaultBurstFromRate
--- PASS: TestRateLimit_DefaultBurstFromRate (0.00s)
=== RUN   TestRateLimit_RejectionResponseShape
--- PASS: TestRateLimit_RejectionResponseShape (0.00s)
=== RUN   TestRateLimit_PrometheusRejectionsCounterIncrements
--- PASS: TestRateLimit_PrometheusRejectionsCounterIncrements (0.00s)
=== RUN   TestRateLimit_RecoversAfterTokenRefill
--- PASS: TestRateLimit_RecoversAfterTokenRefill (0.05s)
PASS
ok  	github.com/omcgo/omcgo/internal/core/middleware	1.740s
```

整个 middleware 包共 30+ 测试全部 PASS，`-race` 无数据竞争告警。

### `golangci-lint run ./internal/core/middleware/...`

```
$ which golangci-lint
golangci-lint not found
```

本环境未安装 golangci-lint。改用 `go vet`：

```
$ go vet ./internal/core/middleware/...
（无输出 → 0 issue）
```

### Charter 验证 1：grep router.go

```
$ grep -n "RateLimit\|ratelimit" omcgo/cmd/app/provider/router.go
176:	// RateLimit: per-IP token bucket，防止单 IP 洪泛拖垮后端。
178:	// 未来如需配置化可在 AppConfig 中新增 ratelimit 段位。
179:	r.Use(middleware.RateLimit(middleware.RateLimitConfig{
```

charter 字面写 `omcgo/cmd/app/router/router.go`，实际仓库 router.go 在 `omcgo/cmd/app/provider/router.go`（cmd/app 下唯一的 router.go）。grep 关键字命中 ≥1 处 → 通过。

### Charter 验证 2：ratelimit*.go 存在

```
$ ls omcgo/internal/core/middleware/ratelimit*.go
omcgo/internal/core/middleware/ratelimit.go
omcgo/internal/core/middleware/ratelimit_test.go
```

通过。

## S5 自检结论

- [x] charter 两条验证命令均通过（grep 命中，文件存在）
- [x] 无 `if carrier ==` 硬编码（本中间件运营商无关）
- [x] 配置结构使用具名类型，Skipper 用具名签名，没有暴露 `any` / `interface{}`
- [x] 无新 TODO/FIXME/`panic("not implemented")`
- [x] metric 名 `http_ratelimit_rejections_total{path}` / log key `ratelimit rejected` 与设计备忘一致
- [x] 错误处理：使用 `prometheus.AlreadyRegisteredError` 类型断言而非 panic，保持幂等
- [x] 并发安全：`sync.Map` + `atomic.Int64`，`-race` 通过
- [x] context 传递：`logger.L(c.Request.Context())` 自动带 request_id
- [x] 无裸 panic、无字符串拼接 SQL（不涉数据库）

## 已知限制 / 未来改进点

1. **单实例限流**：当前 limiter 状态在进程内，多实例部署时同 IP 在不同实例上各享一份配额。下一步可写 Redis 版本（key=`ratelimit:ip:{ip}`，使用 lua 脚本做原子 CAS），按 task 单独立项。
2. **粒度仅 per-IP**：未做 per-route / per-user / per-API-key 维度的细粒度限流。Notification、Northbound 推送等场景未来可考虑路由级配置。
3. **配置硬编码**：`RatePerSecond=100`、`Burst=200`、Skipper 路径白名单写死在 router.go。生产环境需要灰度调整时应配置化（在 `AppConfig` 增加 `RateLimit` 段位）。本期不做以缩小变更面，charter 不要求。
4. **NAT 后多用户共享 IP**：移动网络/企业 NAT 下大量用户共享出口 IP 时可能误伤；上线前应监测 `http_ratelimit_rejections_total` 出现频率，若误伤需要降级到 per-user/JWT-subject 维度（需在 auth 中间件之后）。
5. **Skipper 仅按路径**：`/metrics` 实际由独立 metrics http server 暴露（不在 Gin 路由上），保留 Skipper 中的 `/metrics` 是兜底，不影响功能。
6. **ACS 未接入**：ACS 是 net/http stdlib，且已有自有的 admission/per-device 限流；本任务不接入，建议作为后续 task 单立。
