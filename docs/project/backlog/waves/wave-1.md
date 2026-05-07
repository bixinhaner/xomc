# Wave 1 · 止血（已完结 2026-05-11，8/8 ✅）

> 从 `docs/project/backlog.md` §3 Wave 整改执行队列拆出（2026-05-07，纯搬运）。
> 历史快照，不再更新；当前 Wave 见主 backlog 顶部 ⛔ 横幅。

#### 🟢 Wave 1 · 止血（W1-W2，2026-04-27 ~ 2026-05-11）

| 序 | Task | 标题 | 状态 | Owner | DoD verify |
|----|------|------|------|-------|-----------|
| W1.1 | T-0038 | GitHub Actions CI workflow | ✅ 2026-04-27 | — | commit `aaffef29`；GH Actions 触发 |
| W1.2 | T-0039 | PR 模板 Wave 1 约束 | ✅ 2026-04-27 | — | commit `aaffef29` |
| W1.3 | T-0040 | acs/worker 加 `/healthz` + `/readyz` | ✅ 2026-04-27 | Claude | commit `d4019f9a`；6/6 health 单元测试 PASS / 100% 覆盖率；httptest 模拟 GET /healthz→200 + GET /readyz（依赖故障）→503 |
| W1.4 | T-0041 | `internal/core/middleware/ratelimit` 落地 | ✅ 2026-04-27 | Claude | commit `758aace9`；per-IP token bucket（`golang.org/x/time/rate`，sync.Map+atomic 无锁读路径，后台清扫）+ `cmd/app/provider/router.go` 接入 100 r/s burst 200，跳过 /healthz/readyz/metrics；10 单元测试 -race PASS；charter 两条 grep 全过；新指标 `http_ratelimit_rejections_total{path}` |
| W1.5 | T-0007 + T-0011（W1.5 子集） | F04 告警 webhook 端到端冒烟 | ✅ 2026-04-27 | Claude | commit `43903b81`；migration 000038 加 `alarm_filters.webhook_url` + CHECK 约束；新加 `notify_webhook` filter action + `WebhookDispatcher` 接口（HTTP/JSON，5s 超时，无重试）；8 单元测试 -race 全过（5 dispatcher + 3 filter engine 含 `TestProcessAlarm_NotifyWebhook_EndToEnd` 用 httptest.Server 验证 charter 步骤 4）；新指标 `alarm_webhook_dispatches_total{result}`；retry/dead-letter/HMAC/header/template/email/sms 留 Wave 2 Block A 全量做 |
| W1.6 | T-0006 | E2E 累计用例 @≥20 | ✅ 2026-04-27 | Claude | commit `328c1f48`；`scripts/e2e_verify.sh` 加 `claim()`/`CLAIM_COUNT` 计数器 + W1.6 段补 26 条 claim；实跑 W1.6 段 **27 PASS / 0 FAIL**；五域齐备（auth 5 / device 6 / alarm 5 / kpi+pm 5 / template 5）；断言只判 200/401/404/400 不依赖 seed 列表非空；template 用例自创建-回查-DELETE 闭环 |
| W1.7 | T-0008 | Prom/Grafana/AlertManager 容器编排 | ✅ 2026-04-28 (backfill) | Claude | commit `e878d5e0` (主体) + verify-T-0008.md §4.3 backfill (2026-04-28)；docker compose 起三容器 → Prom `Healthy.` / Grafana `database=ok v10.4.0` / AlertManager `OK` / `/api/v1/targets` 4 target 全 `up`（omc-app/acs/worker/prometheus）；DoD 满 |
| W1.8 | T-0042 | 数据库定时备份 + 恢复演练 | ✅ 2026-04-27 | Claude | commit `27fa743a`；DoD 三 grep 全过；真跑 backup + restore drill：6 项校验全 OK，**RTO 实测 1.034s**（44MB / 30,030 设备 / 138 表，本机 PG16） |

**Wave 1 退出（2026-05-11 对峙）**：≥ 6/8 ✅ + 用户尽责 → 进 Wave 2；< 4/8 + 用户尽责 → AI 嘴炮认账（见 `docs/methodology/AI承诺对峙清单.md` 第二章）

**📊 当前阶段计分（2026-04-28 第四次盘点 — 满分）**：**8.0 / 8 ✅** = W1.1 + W1.2 + W1.3 + W1.4 + W1.5 + W1.6 + W1.7 + W1.8 = 1×8 = 8.0；**Wave 1 满分提前 13 天达成**（对峙日 2026-05-11）。**第二章 W1 末通过率门槛全部突破**：8/8 = 100% > 75% 兑现门槛。**历史盘点**：一盘 4.5（W1.3/W1.7×0.5/W1.8 落地后）；二盘 6.5（W1.4/W1.6 闭环后）；三盘 7.5（W1.5 闭环后）；**四盘 8.0（W1.7 docker backfill 收尾，docker compose up 三容器 + 三 curl + targets 全 up，verify-T-0008.md §4.3 实跑日志已贴）**。

**📌 W1.6 spec 决策（2026-04-27）**：
- 现状：`scripts/e2e_verify.sh` 已含 226 处 `check_status`（多为框架骨架 / 占位），R-002 描述实际可跑用例数为 0（grep `claim` = 0）
- 字面规则 `grep -c check_status ≥ 20` 已天然满足，无效
- **采纳口径（与 `AI承诺对峙清单.md §W1.6` 一致）**：
  1. 实跑 `bash omcgo/scripts/e2e_verify.sh http://localhost:8081 2>&1 | tail -5` 的 `Pass:` 计数 ≥ 20
  2. 覆盖五域：登录/设备/告警/KPI/模板，每域 ≥ 1 条
  3. 新增用例必须用 `claim "<测试名>"` 起头，便于后续 `grep -c claim` 自动核销
- 仪表盘 `E2E 累计用例 0/200` 改读"实跑 Pass 数"（待 T-0006 推进时累加更新）

---

← 返回 [`docs/project/backlog.md`](../../backlog.md)
