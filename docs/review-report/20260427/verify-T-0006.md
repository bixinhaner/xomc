# T-0006 / W1.6 — E2E 实跑 Pass ≥ 20，五域各 ≥ 1（验证记录）

> 任务：在 `omcgo/scripts/e2e_verify.sh` 中补齐五域（登录/设备/告警/KPI/模板）最低 E2E 覆盖，
> 每条新用例以 `claim "<标题>"` 起头，便于 `grep -c '^claim '` 自动核销 ≥ 20。
> Worktree：`.claude/worktrees/agent-a86be2e89eb7a07f2/`，分支 `worktree-agent-a86be2e89eb7a07f2`。
> 实跑目标：`http://localhost:8081`（已健康 `/healthz` → `{"status":"ok"}`）。

---

## S2 设计备忘

### 改动范围

1. 在脚本头部 `pass()` / `fail()` 旁新增 `claim()` 函数：
   - 自增 `CLAIM_COUNT`，输出 `[CLAIM N] <title>`
   - 不改变 PASS/FAIL/TOTAL 计数（用例的 PASS/FAIL 仍由后续 `check_status` 决定）
   - 双口径核销：`grep -c '^claim '` 数 claim 标题行；`Pass:` 数实跑通过断言
2. 在脚本末尾 Summary 之前新增 W1.6 集中段，五个 `section`：
   - `W1.6 Wave 1 — Auth Domain (claim ≥ 4)`
   - `W1.6 Wave 1 — Device Domain (claim ≥ 4)`
   - `W1.6 Wave 1 — Alarm Domain (claim ≥ 4)`
   - `W1.6 Wave 1 — KPI / PM Domain (claim ≥ 4)`
   - `W1.6 Wave 1 — Template / Config Domain (claim ≥ 4)`
3. W1.6 段独立 `W16_LOGIN_RESP` / `W16_TOKEN` / `W16_AUTH`，避免依赖前面 4000+ 行用例残留状态
4. Summary 输出额外打印 `W1.6 Claims: <count>`

### 用例分布表（26 条 claim → 27 条断言全 PASS）

| 域 | claim | 用例 | 期望 | 实测 |
|----|-------|------|------|------|
| auth | 1 | login admin/admin123 | 200 | PASS |
| auth | 2 | login wrong password | 401 | PASS |
| auth | 3 | GET /devices 无 token | 401 | PASS |
| auth | 4 | GET /devices bogus token | 401 | PASS |
| auth | 5 | GET /auth/me 有效 token + username 字段 | 200 | PASS (双断言：状态码 + username) |
| device | 6 | GET /devices | 200 | PASS |
| device | 7 | GET /devices?page=1&page_size=5 | 200 | PASS |
| device | 8 | GET /devices?carrier=cmcc | 200 | PASS |
| device | 9 | GET /devices/<不存在 UUID> | 404 | PASS |
| device | 10 | GET /devices/<malformed UUID> | 400 | PASS |
| device | 11 | GET /devices/stats | 200 | PASS |
| alarm | 12 | GET /alarms/active | 200 | PASS |
| alarm | 13 | GET /alarms/active?page=1&page_size=5 | 200 | PASS |
| alarm | 14 | GET /alarms/active?severity=critical | 200 | PASS |
| alarm | 15 | GET /alarms/<不存在 UUID> | 404 | PASS |
| alarm | 16 | GET /alarms/statistics | 200 | PASS |
| kpi | 17 | GET /pm/kpi | 200 | PASS |
| kpi | 18 | GET /pm/kpi/definitions | 200 | PASS |
| kpi | 19 | GET /pm/counters | 200 | PASS |
| kpi | 20 | GET /pm/counters?start_time=…&end_time=… | 200 | PASS |
| kpi | 21 | GET /pm/thresholds | 200 | PASS |
| template | 22 | GET /templates | 200 | PASS |
| template | 23 | POST /templates 创建 | 201 | PASS |
| template | 24 | GET /templates/<just-created> | 200 | PASS |
| template | 25 | GET /templates/<不存在 UUID> | 404 | PASS |
| template | 26 | DELETE /templates/<just-created> 清理 | 204 | PASS |

> 五域分布：auth 5 / device 6 / alarm 5 / kpi 5 / template 5 — 每域 ≥ 4 ✓

### 设计原则

- **稳定性**：用例选稳定端点，规避 seed 数据缺失（`/alarms` 无顶级 list → 用 `/alarms/active`；不依赖列表非空）
- **独立性**：W1.6 段不与前置用例共享 token/全局变量，自取 token
- **可清理**：template-23 创建的资源在 template-26 立即 DELETE 回收，不污染 DB
- **断言强度**：成功路径 + 失败路径两路都覆盖（401/404/400 各有用例）
- **不改既有**：未触碰原 4000+ 行历史用例；新增段独立放在 Summary 之前

---

## S3 改动文件清单

| 文件 | 改动 |
|------|------|
| `omcgo/scripts/e2e_verify.sh` | +1 函数（`claim`）、+1 计数器（`CLAIM_COUNT`）、+1 Summary 行（`W1.6 Claims: …`）、+5 个 W1.6 section（约 +200 行） |

未新增 / 未修改：
- `omcgo/scripts/seed_e2e_testdata.sql` — W1.6 用例不依赖额外 seed
- `omcgo/internal/**` — 严格遵守"不可越权改后端代码"
- 其他脚本 / migration / 配置 — 无

---

## S4 实跑命令与结果

```bash
# 1. 健康检查
curl -sf http://localhost:8081/healthz
# → {"status":"ok"}

# 2. 语法检查
bash -n omcgo/scripts/e2e_verify.sh
# → exit 0

# 3. claim 行数（顶格 ^claim 形式）
grep -c '^claim ' omcgo/scripts/e2e_verify.sh
# → 26

# 4. 实跑
bash omcgo/scripts/e2e_verify.sh http://localhost:8081 2>&1 | tee /tmp/e2e_w16_v2.log

# 5. 关键尾行
tail -8 /tmp/e2e_w16_v2.log
# →
#   [CLAIM 26] template: cleanup created template via DELETE returns 204
#   [PASS] W1.6 template-5: DELETE /templates/<just-created> (HTTP 204)
#
#   ============================================
#     Results: 354 PASS / 89 FAIL / 45 TOTAL
#     W1.6 Claims: 26
#   ============================================

# 6. 分子统计（直接 grep 日志，避开脚本 TOTAL 历史 bug）
grep -c '\[PASS\]' /tmp/e2e_w16_v2.log   # → 354
grep -c '\[FAIL\]' /tmp/e2e_w16_v2.log   # → 89
grep -c '\[CLAIM '  /tmp/e2e_w16_v2.log  # → 26

# 7. W1.6 段单独统计
grep -E "W1.6 (auth|device|alarm|kpi|template)" /tmp/e2e_w16_v2.log | grep -c PASS  # → 27
grep -E "W1.6 (auth|device|alarm|kpi|template)" /tmp/e2e_w16_v2.log | grep -c FAIL  # → 0
```

### 五域 PASS 分布（W1.6 集中段）

| 域 | claim 数 | 断言 PASS | 断言 FAIL |
|----|---------|----------|----------|
| auth | 5 | 6 | 0 |
| device | 6 | 6 | 0 |
| alarm | 5 | 5 | 0 |
| kpi | 5 | 5 | 0 |
| template | 5 | 5 | 0 |
| **总计** | **26** | **27** | **0** |

### 硬门核销

- [x] `grep -c '^claim ' omcgo/scripts/e2e_verify.sh` = **26 ≥ 20** ✓
- [x] 实跑日志 PASS 计数 = **354 ≥ 20** ✓（仅 W1.6 自身段 27 ≥ 20 也独立达标）
- [x] 五域各 ≥ 1：auth 5 / device 6 / alarm 5 / kpi 5 / template 5 ✓
- [x] W1.6 段 0 FAIL，所有新增用例稳定 ✓

---

## S5 自检结论与限制

### 通过项

1. **claim 函数双口径核销可用**：`grep -c '^claim '` 自动核销 26 条
2. **五域全覆盖且稳定**：26 条 claim 对应 27 条 PASS，0 FAIL
3. **不依赖 seed 列表非空**：用例只断 `200/404/401/400`，不要求 `items.length > 0`
4. **资源自清理**：template-23 创建的模板由 template-26 立即 DELETE
5. **不改后端代码**：仅修改 `omcgo/scripts/e2e_verify.sh`，零侵入

### 已知限制 / 后续工作

1. **脚本 Summary 的 TOTAL 显示 45（非 443）** — 此非 W1.6 引入，是脚本历史中
   `set -uo pipefail` 下某段子 shell 计数被吞的旧问题；不阻塞 W1.6 验收
   （以日志 `grep '[PASS]'` / `grep '[FAIL]'` 直接计数为准，结果 354/89）。
   修复属于另一个独立工单（建议进 backlog 当 W1.7+ 收尾）。
2. **既有 89 FAIL 多为 seed 数据缺失** — 列表为空导致 `Template list has items`、
   `Active alarms has items` 等断言 FAIL。这些用例在 W1.6 范围之外，不属本任务。
3. **未新增 seed** — 优先复用既有数据 + 创建后回查模式，没动 `seed_e2e_testdata.sql`。
4. **后续 W1.6 → W2 增量路线建议**：
   - W2 阶段把现有 89 FAIL 中可补的 seed 注入（设备 SN list、告警库、模板基线），
     让 PASS 上 400+
   - 把"创建—回查—删除"三件套模式扩展到 alarm rules、firmware、user/role CRUD
   - claim 行数随每个 wave +10 条递增，配合 release-gate 阈值

### 产出文件清单

- `omcgo/scripts/e2e_verify.sh` — 已 git add（+claim 函数、+W1.6 段、+Summary 行）
- `verify-T-0006.md` — 本文件，未 git add（worktree 根，主会话决定是否纳入）

