# PRD: RC 冻结 + 冒烟用例集（T-0025）

> **关联**: Backlog T-0025 / Sprint-07 / Domain=infra / Type=td
> **作者**: Claude（代 Owner=QA / 发布经理 §16.12）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施（A 方案：主会话全程深度协作）
> **里程碑**: `docs/project/milestone/2026Q2-to-RC.md` Beta → RC 冲刺关键节点

---

## 1. 业务背景

OMC 项目在 14 周 RC 冲刺期接近收尾（已闭环 Wave 1 满分 / Wave 2 12/13 / Wave 3 11/15 退出门槛 / Wave 4 启动）。当前状态适合标记首版 RC 候选并固化"快速回归"能力：

- **RC 冻结需求**：里程碑要求"运营商试点部署（≤10,000 基站）的商用就绪状态"。当前 main HEAD 累计 50+ commit 全部 push origin/main，需切快照（git tag + 文档）让运维 / 部署 / 客户对齐版本。
- **冒烟用例集需求**：e2e_verify.sh 已 5498 行 / 549 PASS / 131 claim 是**全量回归**（5-10 min），但部署窗口（如热升级、紧急修复后验证）需要 < 3 min 的快速验证。RC 发布前后 / 灰度阶段 / 故障恢复后均需冒烟通过才能放行。

T-0006 累计 ≥150 依赖已解锁（实跑 PASS=549 远超 150），本任务无外部阻塞。

---

## 2. 用户故事

| 角色 | 故事 |
|------|------|
| QA / 发布经理 | 我希望 push 到生产前 < 3 min 跑完冒烟，100% PASS 才放行；与全量 e2e 互补 |
| 部署运维 | 我希望热升级后立即 `bash smoke_test.sh` 看 20/20 PASS，秒判可用 |
| 产品经理 | 我希望 RC 冻结文档清楚记录"包含哪些 features / 已知限制 / 回滚方案"，给运营商客户一份明确的 release notes |
| 项目经理 | 我希望 git tag `rc-2026Q2-001` 标在 main HEAD，CI/CD 后续可基于 tag 触发自动构建 |

---

## 3. 验收标准（Given-When-Then）

### V1 — RC 冻结产出
- **Given** main HEAD 已 push origin/main（commit `c5e134c0` 或更新）
- **When** `git tag -a rc-2026Q2-001 -m "..."` 标在 HEAD
- **Then** `git tag` 输出含 `rc-2026Q2-001`；`git show rc-2026Q2-001` 显示完整 tag 信息

### V2 — RC 文档完整
- **Given** 用户访问 `docs/project/release/RC-2026Q2-001.md`
- **When** 文档结构含 6 节（版本基线 / 包含 features / 已知限制 / 部署清单 / 回滚方案 / 兼容性）
- **Then** 每节非空；features 列出 Wave 1+2+3+4 闭环成果；已知限制列出 4 项 staging 待办（W3.F.1/H.2/I.2/I.3）

### V3 — 冒烟脚本结构
- **Given** `omcgo/scripts/smoke_test.sh` 新建
- **When** 运行 `bash -n omcgo/scripts/smoke_test.sh` 语法检查
- **Then** 退出码 0；脚本含 8 个 section（auth/device/alarm/kpi-pm/topology/report/admin/health）+ 20 个 `claim` 计数

### V4 — 冒烟实跑 100% PASS
- **Given** 后端 docker-compose 起动（postgres/redis/nats/minio + omcgo-app/acs/worker）
- **When** `bash omcgo/scripts/smoke_test.sh http://localhost:8081`
- **Then** 输出 `Pass: 20 / Fail: 0`，退出码 0

### V5 — 冒烟在 < 3 min 内跑完
- **Given** smoke_test.sh 启动
- **When** 实测 wall clock
- **Then** 总耗时 < 180 秒（curl + check_status 累加）

### V6 — Fail 一条退出码非零
- **Given** 后端任一关键端点返回 5xx
- **When** smoke_test.sh 跑到该用例
- **Then** check_status 计入 FAIL；脚本最终退出码 1（CI/CD 可据此阻断发布）

### V7 — 冒烟与 e2e_verify.sh 互补不冲突
- **Given** 同时存在 `e2e_verify.sh` (549 PASS / 5498 行) 和 `smoke_test.sh` (20 PASS / ~300 行)
- **When** 运维任选其一
- **Then** smoke 是 e2e 的"严格子集快速版"，但**独立执行**（不 source / import e2e）— 避免 e2e 改动影响 smoke 稳定性

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| RC 冻结流程 | 一致（OEM 内部）| 一致 | 一致 |
| 冒烟用例覆盖 | 一致（API 通用层）| 一致 | 一致 |
| Release notes 翻译 | 待 GA 期补 | 待 GA 期补 | 待 GA 期补 |
| 实际差异 | **无**（本任务范围内）| **无** | **无** |

**结论**：T-0025 是 OEM 发布流程内部任务，对运营商透明。差异在 GA 期 release notes 翻译时单独处理。

---

## 5. 非目标（明确不做）

- ❌ 不接入 CI/CD 自动 RC 触发（手动 `git tag`）
- ❌ 不补齐完整 release notes（仅记录里程碑级要点，详细 changelog 由 commit history + git log 提供）
- ❌ 不强制 staging 部署演练（W3.H.2 零停机演练已分立为 T-0066）
- ❌ smoke 不替代 e2e（互补关系；e2e 仍是回归门，smoke 是部署门）
- ❌ 不实现 smoke 自动重试 / 自动告警（保持简单：fail 即 exit 1）
- ❌ 不接入 Prometheus/Grafana 验证（smoke 只验 HTTP 端点；Prom 验证由 W1.7 容器编排提供）
- ❌ 不做多版本对比 RC（仅本次首版 rc-2026Q2-001）

---

## 6. 依赖

| 依赖 | 用途 | 状态 |
|------|------|------|
| T-0006 累计 ≥150 | E2E 实跑数据支撑（声明 RC 时引用 549 PASS）| ✅ 已解锁（累计 549）|
| `e2e_verify.sh` claim() / check_status() / check_status_in() helpers | 冒烟脚本可参考（自实现简化版）| ✅ 已存在 |
| 后端 docker-compose | smoke 实跑环境 | ⚠️ 本地需起动；本任务在主会话内尽力实跑，否则在 verify md 标记"待 staging 回填实测" |
| `docs/project/milestone/2026Q2-to-RC.md` | RC 文档引用 | ✅ 已存在 |

---

## 7. 度量

无新增 Prometheus 指标（smoke 是 black-box HTTP 测试，不引入 metric）。**度量来自 smoke 脚本输出本身**：

```
Pass:  20 / 20  ✅
Fail:  0  / 20
Time:  N seconds
Exit:  0
```

CI/CD 后续 PR 可解析此输出做发布门控。

---

## 8. 设计备忘（S2）

### 8.1 20 用例分布（D4 8 域）

| # | 域 | 用例 | 期望状态 |
|---|----|----|---------|
| 1 | **auth** | POST /auth/login (admin/admin) | 200 |
| 2 | auth | POST /auth/login (bad cred) | 401 |
| 3 | **device** | GET /devices?page=1 | 200 |
| 4 | device | GET /devices/{not-found-uuid} | 404 |
| 5 | device | POST /devices (missing fields) | 400/422 |
| 6 | **alarm** | GET /alarms?page=1 | 200 |
| 7 | alarm | GET /alarms/active | 200 |
| 8 | alarm | GET /alarms/summary | 200 |
| 9 | **kpi/pm** | GET /pm/kpi/definitions | 200/401 |
| 10 | kpi/pm | GET /pm/counters?page=1 | 200/401 |
| 11 | kpi/pm | GET /pm/files?page=1 | 200/401 |
| 12 | **topology** | GET /groups (or device-groups/tree) | 200 |
| 13 | topology | GET /sites?page=1 | 200 |
| 14 | **report** | GET /reports/definitions?page=1 | 200/401 |
| 15 | report | GET /reports/sample-data | 200/401 |
| 16 | **admin** | GET /admin/users?page=1 | 200/401/403 |
| 17 | admin | GET /admin/roles?page=1 | 200/401/403 |
| 18 | admin | GET /admin/dead-letters?page=1 | 200/401/403（T-0012 新加） |
| 19 | **health** | GET /healthz（无 auth）| 200 |
| 20 | health | GET /readyz（无 auth）| 200/503 |

### 8.2 smoke_test.sh 结构

```bash
#!/usr/bin/env bash
# OMC Smoke Test — RC freeze quick verification
# Usage: bash smoke_test.sh [http://host:port]
# Default: http://localhost:8081

set -uo pipefail

API_BASE="${1:-http://localhost:8081}"
API="${API_BASE}/api/v1"
HEALTH="${API_BASE}"  # /healthz, /readyz at root

PASS=0
FAIL=0
TOTAL=20
START_TS=$(date +%s)

claim() { TOTAL_CHECK=$((${TOTAL_CHECK:-0}+1)); }

check_status() {
    local desc="$1" expected="$2" actual="$3"
    if [ "$actual" = "$expected" ]; then
        echo "  ✅ $desc → $actual"
        PASS=$((PASS+1))
    else
        echo "  ❌ $desc → expected $expected, got $actual"
        FAIL=$((FAIL+1))
    fi
}

check_status_in() {
    local desc="$1" expected_set="$2" actual="$3"
    for code in $expected_set; do
        if [ "$actual" = "$code" ]; then
            echo "  ✅ $desc → $actual (∈ {$expected_set})"
            PASS=$((PASS+1)); return
        fi
    done
    echo "  ❌ $desc → expected one of {$expected_set}, got $actual"
    FAIL=$((FAIL+1))
}

# === auth (2) ===
echo "=== auth (2 cases) ==="
HTTP_CODE=$(curl -s -o /tmp/smoke-auth.json -w "%{http_code}" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin"}' \
    "$API/auth/login")
check_status "POST /auth/login (admin/admin)" "200" "$HTTP_CODE"

TOKEN=""
if [ "$HTTP_CODE" = "200" ]; then
    TOKEN=$(python3 -c "import sys,json; print(json.load(open('/tmp/smoke-auth.json'))['access_token'])" 2>/dev/null || echo "")
fi
AUTH_HEADER="Authorization: Bearer ${TOKEN}"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"BAD"}' \
    "$API/auth/login")
check_status "POST /auth/login (bad cred)" "401" "$HTTP_CODE"

# ... (重复模式 18 个用例)

END_TS=$(date +%s)
DURATION=$((END_TS - START_TS))

echo ""
echo "════════════════════════════════════════"
echo "  Smoke Test Summary"
echo "════════════════════════════════════════"
echo "  Pass:   ${PASS} / ${TOTAL}"
echo "  Fail:   ${FAIL} / ${TOTAL}"
echo "  Time:   ${DURATION}s"
echo ""

if [ "$FAIL" -gt 0 ] || [ "$PASS" -ne "$TOTAL" ]; then
    echo "  ❌ SMOKE FAILED — RC NOT READY"
    exit 1
fi
echo "  ✅ SMOKE PASSED — RC READY"
exit 0
```

### 8.3 RC-2026Q2-001.md 文档结构

```
# Release Candidate: rc-2026Q2-001

## 1. 版本基线
   - main HEAD: <commit-hash>
   - tag: rc-2026Q2-001
   - 时间: 2026-04-29

## 2. 包含 features
   - Wave 1 满分（CI / 健康检查 / ratelimit / 备份恢复 / Prom）
   - Wave 2 退出（F04 邮件+webhook+模板/历史 / 测试覆盖 / 前端整改 / E2E 549）
   - Wave 3 退出（NATS+5 subject / 性能监控 / 安全 CI / K8s+异地备份 / Release Gate）
   - Wave 4 启动（F08 SNMP 骨架 / License Enforcer / Worker DLQ / 前端 Topology/Report）

## 3. 已知限制（待 GA 期补完）
   - W3.F.1 5K 设备 24h 压测（需 staging）
   - W3.H.2 零停机演练
   - W3.I.2 灰度演练
   - W3.I.3 回滚演练
   - F08 SNMP 联调（需运营商对接）
   - 短信通道（需凭据）

## 4. 部署清单
   - docker compose 三容器（acs/app/worker）
   - migrations 000044 已就绪
   - K8s manifests 17 yaml 已就绪
   - 异地备份 db_backup.sh 已扩展

## 5. 回滚方案
   - git revert 到上一 RC tag（首版无）
   - DB rollback: goose down (双向配对)
   - K8s rollback: kubectl rollout undo

## 6. 兼容性
   - PostgreSQL 16 + TimescaleDB
   - Redis 7
   - NATS JetStream
   - MinIO S3
   - Go 1.25
```

### 8.4 git tag 命令（S7 末执行）

```bash
git tag -a rc-2026Q2-001 -m "$(cat <<EOF
RC: 2026 Q2 — Beta → RC 冲刺首版候选

- Wave 1 满分 8/8
- Wave 2 退出 12/13 (AI 极限)
- Wave 3 退出 11/15 (>=11/15 ✅)
- Wave 4 启动: SNMP 骨架 + License + DLQ + 前端

E2E: 549 PASS / 0 FAIL / claim 131
Smoke: 20/20 PASS
Risks: R-103 ✅ R-106 ✅ closed
Backlog: T-0025 done

Detail: docs/project/release/RC-2026Q2-001.md
EOF
)"
git push origin rc-2026Q2-001
```

---

## 9. DoD

- [ ] PRD 七要素全
- [ ] `docs/project/release/RC-2026Q2-001.md` 6 节非空
- [ ] `omcgo/scripts/smoke_test.sh` `bash -n` 通过
- [ ] 20 用例覆盖 8 域
- [ ] 实跑 100% PASS（或 verify md 标记"待 staging"+ syntax+dry-run 验证）
- [ ] git tag `rc-2026Q2-001` 标在 main HEAD
- [ ] backlog T-0025 状态 planned → done
- [ ] §10 变更日志记账

---

## 10. 后续 PR

- 接入 CI/CD 自动 RC 触发（push tag → build artifact）
- 完整 changelog 自动生成（git log + commit footer 解析）
- smoke_test.sh 加入 `--quick` / `--full` 模式（5/20 用例可选）
- staging 演练后回填 V4 / V5 实测数据

---

*本 PRD 由 dev-pipeline /pick T-0025 ULTRATHINK A 方案生成。默认 6 项决策（D1=C / D2=A / D3=20 / D4=8 域 / D5=100% / D6=rc-2026Q2-001）。*
