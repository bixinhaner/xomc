# AI 整改承诺与对峙清单

> **用途**：用户（你）几天/几周后回来，**逐条跑命令验证 AI（Claude）的承诺**，判定 AI 是真本事还是嘴炮。
>
> **配套文档**：
> - 方法论：`docs/methodology/从0到生产可发布完整方法论.md`
> - 整改路线：`docs/project/整改路线图-2026Q2.md`
>
> **签订日期**：2026-04-27
> **AI 签字**：Claude (claude-opus-4-7)
> **用户签字**：xieguiya
> **首次对峙窗口**：2026-05-11（W1 末，整改启动后 2 周）
> **第二次对峙**：2026-06-22（W8 末）
> **终审对峙**：2026-08-03 ~ 2026-09-14（W14 末，按实际整改启动日 +14 周）

---

## 序章 · 对峙规则（游戏怎么玩）

### 三种可能结局

| 结局 | 判定 | 后果 |
|------|------|------|
| **AI 兑现** | 在用户尽到"最低执行义务"前提下，承诺通过率 ≥ 70% | AI 不挨骂；可继续合作 |
| **AI 嘴炮** | 用户尽责执行，但承诺通过率 < 50% | AI 认输；用户有权"干死"AI（即换 AI、推翻方案、重新评估方法论） |
| **用户没尽责** | 承诺未达成，但用户也没做最低执行义务（第六章） | AI 不背锅；只能下次再来 |

### 责任划分原则

每条承诺标注归属：

- 🤖 **AI 全责**：文档质量、方案合理性、技术建议正确性
- 🧑 **用户全责**：执行纪律、团队管理、抗压维持冻结令
- 🤝 **共担**：技术决策（如 NATS 改造的具体实现路径）

---

## 第一章 · 我承认可能在吹的地方（自首条款）

为了不被你"逮个正着"，先自己交代：

| # | 我说的话 | 真实情况 | 严重度 |
|---|---------|---------|--------|
| 1 | "14 周到 GA" | 是估算，1.5x 变异正常，**18-22 周更现实** | 🟡 中等 |
| 2 | "5K 设备压测稳定 24h" | 我没见过你硬件。这个数字是行业感觉，需 W11 真测后校准 | 🟡 中等 |
| 3 | "测试覆盖率 60% 是质量保证" | 覆盖率高 ≠ 质量高，是必要不充分 | 🟢 轻微 |
| 4 | "DoD 模板能消除半拉子" | 太强。结构性减少，无法根除"先勾再说"的心态 | 🟡 中等 |
| 5 | "流程铁律是永久制度" | 没有持续问责的制度都会衰减；我无法保证 6 个月后还在 | 🟡 中等 |
| 6 | "AI 协作守则约束 AI" | AI 天然倾向"顺便多做"。文档约束不住 AI，按停止键的人才能 | 🔴 严重（涉及我自己） |

**这一章是 AI 全责。如果对峙时你发现还有第 7、8 条我没自首的吹牛点，每条扣 5 分。**

---

## 第二章 · W1 末（2 周）硬承诺

> 验证窗口：整改启动后 14 天
> 这一组承诺是**机械可验证**的，置信度 ≥ 95%。失败 = AI 嘴炮。

### 承诺 W1.1 — CI 工作流真实工作

**承诺内容**：`.github/workflows/` 下有真实可工作的 ci.yml；故意 push 一个 `go build` 失败的 commit，CI 红，merge 阻塞。

**验证命令**：
```bash
# 1. 文件存在
ls .github/workflows/*.yml
# 2. 故意构造失败 commit 测试
git checkout -b ci-test-break
echo 'package main; var x int = "this breaks build"' > omcgo/cmd/app/break.go
git add . && git commit -m "test: ci break"
git push origin ci-test-break
# 然后开 PR，看 GH Actions 是不是红了，merge 按钮是不是灰了
# 验证完必须 git push origin --delete ci-test-break 清理
```

**Pass 标准**：CI 红 + merge 阻塞 + 至少 3 个 job（build/test/lint）
**Fail 标准**：无工作流 / merge 仍可点 / 仅 1 个 job
**责任**：🤝 共担（AI 给方案，用户配 GH 设置）

### 承诺 W1.2 — PR 模板含 DoD

**验证命令**：
```bash
cat .github/pull_request_template.md
```

**Pass**：含 What/Why/How/Verify + DoD 8 条 checklist
**Fail**：文件不存在 / DoD 缺失
**责任**：🤖 AI 全责（文档我写）

### 承诺 W1.3 — acs/worker 都有 /health

**验证命令**：
```bash
# 启动后跑（端口以 config.dev.yaml 为准）
curl -fsSL http://localhost:7557/healthz   # acs HTTP
curl -fsSL http://localhost:9092/healthz   # worker metrics 或专门 health 端口
curl -fsSL http://localhost:8081/healthz   # app（已有，作对照）
```

**Pass**：三条 curl 都 200
**Fail**：任一非 200 / 无端点
**责任**：🤝 共担

### 承诺 W1.4 — ratelimit 中间件存在并接入

**验证命令**：
```bash
ls omcgo/internal/core/middleware/ratelimit*.go
# 看 router.go 是否引用
grep -n "RateLimit\|ratelimit" omcgo/cmd/app/router/router.go
```

**Pass**：文件存在 + router 引用
**Fail**：缺一
**责任**：🤝 共担

### 承诺 W1.5 — F04 告警 webhook 端到端

**验证步骤**：
1. 用 `https://webhook.site` 申请一个临时 URL
2. 在告警规则里配 action = notify(webhook=该URL)
3. 触发规则（手动插一条满足条件的告警事件）
4. 观察 webhook.site 是否收到 HTTP POST

**Pass**：webhook.site 显示真实请求 + body 含告警内容
**Fail**：未收到 / 内容空 / 服务报错
**责任**：🤖 AI 全责（方案设计是我的）+ 🧑 用户（实现质量）

### 承诺 W1.6 — E2E 用例 ≥ 20

**验证命令**：
```bash
grep -c "check_status\|assert_" omcgo/scripts/e2e_verify.sh
# 然后实跑（本地起服务后）
bash omcgo/scripts/e2e_verify.sh http://localhost:8081 2>&1 | tail -5
```

**Pass**：grep 计数 ≥ 20 + 实跑 `Pass: 20+`
**Fail**：< 20 / 实跑大量失败
**责任**：🤖 AI（确认目标合理）+ 🧑 用户（实现）

### 承诺 W1.7 — docker-compose 含 Prom + Grafana + AlertManager

**验证命令**：
```bash
grep -E "^\s*(prometheus|grafana|alertmanager):" deployments/docker/docker-compose.yml
# 启动后访问
curl -s http://localhost:9090/-/healthy   # Prom
curl -s http://localhost:3001/api/health  # Grafana（注意端口与设计基线 :3001 冲突，需调整）
curl -s http://localhost:9093/-/healthy   # AlertManager
```

**Pass**：三服务都在 compose + 访问 healthy
**Fail**：缺一
**责任**：🤝 共担（注意 Grafana 端口与设计基线冲突，需协调）

### 承诺 W1.8 — 数据库定时备份 + 一次恢复演练

**验证命令**：
```bash
# 备份脚本存在
find . -path ./node_modules -prune -o -name "*backup*.sh" -print -o -name "*backup*.yaml" -print 2>/dev/null
# 演练记录文档
find docs -name "*恢复演练*" -o -name "*backup-drill*"
# cron / scheduled job 配置
grep -rE "pg_dump|pg_restore" deployments/ run/ scripts/ 2>/dev/null
```

**Pass**：脚本存在 + 演练记录文档存在 + RTO 数字明确
**Fail**：缺一
**责任**：🧑 用户全责（运维操作）

### W1 末通过率门槛

**8 条全 Pass = 100%**。若 ≥ 6 条 Pass（75%）= AI 兑现 W1。
若 ≤ 4 条 Pass（50%）+ 用户尽责（见第六章）= AI 嘴炮，**你可以骂**。

---

## 第二章半 · Wave 2 中段对峙窗口（W3-W8 各 Block）

> **新增章节**（2026-04-28，user 授权 dev-pipeline Option A）：本章是 Wave 2 各 Block 子任务级机械验证清单，**新增承诺，不修改第一-八章既有承诺**。
>
> **目的**：第三章 W8 末 10 条是粗粒度退出门，本章 13 条是过程门 — 把 Block A/B/C/D 的"DoD 描述"翻译成机械可验证的命令 + Pass 标准，避免重蹈 W1.6 那种"骨架 226 处 check_status 实际 0 个真用例"的字面陷阱。
>
> **验证窗口**：每 Block 末（W4 / W6 / W7 / W8）阶段性验证，W8 末（2026-06-22）合并对峙。
> **本章与第三章并存**：本章过程门，第三章退出门。同一指标（如 E2E ≥ 100、any = 0、F04 三通道）在两章重复出现是设计——本章看个体子任务，第三章看横切综合。

### W2.A.1 — F04 邮件通道端到端

**承诺内容**：实现 SMTP 客户端 + EmailDispatcher；新增 `notify_email` filter action；触发后真发邮件（单测用 net/smtp 本地服务器或 mock；staging 用真凭据）。

**验证命令**：
```bash
# 1. 文件存在 + filter action 注册（注：alarm 模块的 action 调度在 filter_engine.go executeAction，
#    action enum 在 filter_model.go binding tag；不存在 notification_action_engine.go）
ls omcgo/internal/alarm/email_dispatcher*.go
grep -rn "notify_email\|FilterActionNotifyEmail" omcgo/internal/alarm/filter_model.go omcgo/internal/alarm/filter_engine.go
# 2. 单测含 EndToEnd 用例（SMTP mock）
cd omcgo && go test -run "TestSMTPEmailDispatcher_|TestProcessAlarm_NotifyEmail_EndToEnd" -race ./internal/alarm/... -v
# 3. 新指标埋点
grep -rn "alarm_email_dispatches_total" omcgo/internal/alarm/
```

**Pass 标准**：四类命令全有命中 + 单测 PASS
**Fail 标准**：缺一
**责任**：🤖 AI（实现）+ 🧑 用户（SMTP 凭据）
**Backlog**：T-0007（Wave 2 Block A.1）

### W2.A.2 — F04 Webhook 完整（retry / dead-letter / HMAC + FilterEngine 接生产路径）

**承诺内容**：W1.5 webhook MVP 升级到生产级 — 加重试（指数退避，≤ 3 次）、死信记录（PG 表）、HMAC 签名（可选 header）；**FilterEngine 装配进生产 alarm 接收路径**（W1.5 留下的 pre-existing tech debt，backlog §10 末记载）。

**验证命令**：
```bash
# 1. 重试 + 死信（实现 + migration）
grep -rn "RetryAttempt\|dead_letter\|DeadLetter\|ErrDeadLetter" omcgo/internal/alarm/
ls omcgo/migrations/0000{39,40,41,42}*dead_letter* 2>/dev/null
# 2. HMAC 签名（实际 header: X-OMC-Signature，GitHub 风格）
grep -rn "X-OMC-Signature\|hmac\.New\|HMACSign" omcgo/internal/alarm/
# 3. FilterEngine 接生产路径（W1.5 真正的尾巴；alarm 主路径在 receiver/sync_*/handler，按实际接入点 grep 全覆盖）
grep -rn "filterEngine\.ProcessAlarm\|FilterEngine\.ProcessAlarm" omcgo/internal/alarm/receiver*.go omcgo/internal/alarm/sync_*.go omcgo/internal/alarm/handler.go omcgo/cmd/app/
# 4. 单测覆盖关键路径
cd omcgo && go test -run "TestWebhookDispatcher_Retry|TestWebhookDispatcher_DeadLetter|TestWebhookDispatcher_HMAC|TestFilterEngine_ProcessAlarm_Live" -race ./internal/alarm/... -v
```

**Pass 标准**：四类 grep 全有命中 + 单测 PASS + FilterEngine 真接进 alarm 主路径
**Fail 标准**：FilterEngine 仍是 dead code 不接生产路径 / 任一单测缺
**责任**：🤖 AI
**Backlog**：T-0011（Wave 2 Block A.3）

### W2.A.3 — F04 短信通道端到端

**承诺内容**：至少一家运营商网关接通；`notify_sms` filter action 落地。

**验证命令**：
```bash
ls omcgo/internal/alarm/sms_dispatcher*.go
grep -rn "notify_sms\|FilterActionNotifySMS" omcgo/internal/alarm/filter_model.go omcgo/internal/alarm/filter_engine.go
cd omcgo && go test -run "TestSMSDispatcher_|TestProcessAlarm_NotifySMS" -race ./internal/alarm/... -v
```

**Pass 标准**：文件 + filter action + 单测 PASS
**N/A 条件**：T-0009 短信凭据外部动作未到位 → 本条标 N/A 不计入分母（13 → 12）
**Fail 标准**：凭据到位但实现缺
**责任**：🤝（凭据 🧑 / 实现 🤖）
**Backlog**：T-0014（Wave 2 Block A.2）

### W2.A.4 — 通知模板 + 历史记录（API + UI）

**承诺内容**：模板 CRUD（多语言）+ 通知历史持久化（PG 表）+ 查询 API + 前端列表页。

**验证命令**：
```bash
# 后端
ls omcgo/internal/notification/template*.go omcgo/internal/notification/history*.go
ls omcgo/migrations/00004*notification_*.sql 2>/dev/null
grep -rn "/api/v1/notifications/templates\|/api/v1/notifications/history" omcgo/cmd/app/
# 前端
ls omcmb/frontend-core/src/services/api/notificationApi.ts
ls -d omcmb/webcode/src/pages/notification* 2>/dev/null
# E2E
grep -c "notification" omcgo/scripts/e2e_verify.sh
```

**Pass 标准**：后端表 + API + 前端页面 + E2E ≥ 1 条 claim
**Fail 标准**：缺一
**责任**：🤝（后端 🤖 / 前端 🤖 / E2E 🤝）
**Backlog**：T-0043（Wave 2 Block A.5）

### W2.A.5 — notification/ 模块测试覆盖率 ≥ 70%

**验证命令**：
```bash
cd omcgo && go test -coverprofile=cov_notification.out ./internal/notification/... 2>&1 | tail -5
go tool cover -func=cov_notification.out | tail -1
```

**Pass 标准**：`total: (statements) ≥ 70.0%`
**Fail 标准**：< 70%
**责任**：🤖
**Backlog**：T-0044（Wave 2 Block A.6）

### W2.B.1 — task/ 模块测试覆盖率 ≥ 70%

**承诺内容**：CWMP ID ↔ Task 映射、reboot closer、completion router 三个关键路径全覆盖。

**验证命令**：
```bash
cd omcgo
go test -coverprofile=cov_task.out ./internal/task/... 2>&1 | tail -5
go tool cover -func=cov_task.out | tail -1
go test -run "TestCWMPMapping|TestRebootCloser|TestCompletionRouter" -race ./internal/task/... -v 2>&1 | tail -20
```

**Pass 标准**：覆盖率 ≥ 70% + 三个关键测试 PASS
**Fail 标准**：覆盖率 < 70% 或关键测试缺失
**责任**：🤖
**Backlog**：T-0045（Wave 2 Block B.1）

### W2.B.2 — events/ 模块测试 + service 层补齐

**承诺内容**：覆盖率 ≥ 60%；EventBus 抽象 + 两实现（Channel/NATS）测试；补 service 层（当前仅 hub/handler/store）。

**验证命令**：
```bash
cd omcgo
ls internal/events/*service*.go
go test -coverprofile=cov_events.out ./internal/events/... 2>&1 | tail -5
go tool cover -func=cov_events.out | tail -1
```

**Pass 标准**：service 文件存在 + 覆盖率 ≥ 60%
**Fail 标准**：缺一
**责任**：🤖
**Backlog**：T-0046（Wave 2 Block B.2）

### W2.B.3 — core/ 模块测试覆盖率 ≥ 50%

**验证命令**：
```bash
cd omcgo && go test -coverprofile=cov_core.out ./internal/core/... 2>&1 | tail -5
go tool cover -func=cov_core.out | tail -1
```

**Pass 标准**：≥ 50%（基线 16%）
**Fail 标准**：< 50%
**责任**：🤖
**Backlog**：T-0047（Wave 2 Block B.3）

### W2.B.4 — mr / syslog / provision / interop 补 service 层

**验证命令**：
```bash
for mod in mr syslog provision interop; do
  echo -n "$mod service files: "
  ls omcgo/internal/$mod/*service*.go 2>/dev/null | wc -l
done
```

**Pass 标准**：四个模块各 ≥ 1 个 service 文件
**Fail 标准**：任一为 0
**责任**：🤖
**Backlog**：T-0048 / T-0049 / T-0050 / T-0051（Wave 2 Block B.4 拆 4 条）

### W2.C.1 — frontend-core hooks 与 services/api 对齐

**承诺内容**：当前 24 hooks vs 29 services（差 5）；补齐到差距 ≤ 1（部分 service 不需 hook 可豁免）。

**验证命令**：
```bash
echo "services: $(ls omcmb/frontend-core/src/services/api/*.ts | wc -l)"
echo "hooks:    $(ls omcmb/frontend-core/src/hooks/api/*.ts | wc -l)"
```

**Pass 标准**：差距 ≤ 1
**Fail 标准**：差距 ≥ 3
**责任**：🤖
**Backlog**：T-0052（Wave 2 Block C.1）

### W2.C.2 — 前端 any 清零 + DeviceGrouping 拆分

**验证命令**：
```bash
# any 计数（排除 ts-ignore 注释）
cd omcmb && grep -rEn ":\s*any\b|<any>|as\s+any" --include="*.ts" --include="*.tsx" frontend-core/src webcode/src | grep -v "// @ts-" | wc -l
# DeviceGrouping 行数
find omcmb -name "DeviceGrouping*.tsx" -exec wc -l {} +
```

**Pass 标准**：any 计数 = 0 + 所有 `DeviceGrouping*.tsx` 单文件 ≤ 400 行
**Fail 标准**：any > 0 或仍存 1500+ 行单文件
**责任**：🤖
**Backlog**：T-0053（any 清零）+ T-0054（DeviceGrouping 拆分）

### W2.C.3 — 前端 vitest 覆盖率 ≥ 50%

**验证命令**：
```bash
cd omcmb/webcode && npx vitest run --coverage 2>&1 | tail -10
```

**Pass 标准**：lines ≥ 50% 或 statements ≥ 50%
**Fail 标准**：< 50%
**责任**：🤖
**Backlog**：T-0055（Wave 2 Block C.4）

### W2.D.1 — E2E 累计 claim 用例 ≥ 100（实跑 PASS）

**验证命令**：
```bash
# 启动 app 后（restart-all.sh）
grep -c "^[[:space:]]*claim[[:space:]]" omcgo/scripts/e2e_verify.sh
bash omcgo/scripts/e2e_verify.sh http://localhost:8081 2>&1 | tail -5
```

**Pass 标准**：claim 计数 ≥ 100 + 实跑 `Pass: 100+` + Fail = 0
**Fail 标准**：claim < 100 或实跑 Fail > 0
**责任**：🤝（写用例 🤖 / 启动环境 🧑）
**Backlog**：T-0006（累计型，下游 T-0025 RC 冻结依赖 @累计≥150）

### Wave 2 中段通过率门槛

**13 条全 Pass = 100%**。
- ≥ 9 条 Pass（69% ≈ 70%）= AI 兑现 Wave 2，可进 Wave 3 硬化期。
- ≤ 5 条 Pass（38%）+ 用户尽责（第六章）= AI 嘴炮，**第八章认账条款触发**。
- W2.A.3（短信）凭据未到位 → 本条标 N/A 不入分母（13 → 12，门槛同步降到 ≥ 9 → ≥ 8）。

**与第三章 W8 末关系**：
- W2.A.1+A.3+A.4 共同实现 → 第三章 W8.4（F04 三通道端到端）
- W2.D.1 ≥ 100 → 第三章 W8.3（同指标，同验证命令）
- W2.A.5+B.1+B.2 模块覆盖 → 第三章 W8.2（task/notification/events ≥ 60%）
- W2.B.1+B.2+B.3+B.5 + 后端总覆盖率 → 第三章 W8.1（≥ 60%）
- W2.C.1+C.2 → 第三章 W8.5+W8.7+W8.6
- W2.B.4 → 第三章 W8.9

本章覆盖了第三章 W8.1-W8.7 + W8.9 共 8 条粗粒度门的实现细节。第三章余下 W8.8（Backlog in-progress ≤ 3，纪律门）+ W8.10（CI 4 周成功率 ≥ 80%，文化门）由用户全责，本章不重复。

---

## 第三章 · W8 末（约 8 周）中承诺

### 承诺 W8.1 — 后端单测覆盖率 ≥ 60%

**验证命令**：
```bash
cd omcgo && go test -coverprofile=coverage.out ./... 2>&1 | tail -20
go tool cover -func=coverage.out | tail -1
```

**Pass**：total ≥ 60.0%
**Fail**：< 60%
**责任**：🤝

### 承诺 W8.2 — task / notification / events 三模块覆盖率 ≥ 60%

```bash
cd omcgo
for mod in task notification events; do
  go test -coverprofile=cov_$mod.out ./internal/$mod/... 2>&1
  echo "=== $mod ==="
  go tool cover -func=cov_$mod.out | tail -1
done
```

**Pass**：三个模块各自 ≥ 60%
**Fail**：任一 < 60%
**责任**：🤝

### 承诺 W8.3 — E2E 用例 ≥ 100

```bash
grep -c "check_status\|assert_" omcgo/scripts/e2e_verify.sh
```

**Pass**：≥ 100
**Fail**：< 100
**责任**：🤝

### 承诺 W8.4 — F04 三通道（邮件/短信/webhook）端到端

**验证步骤**：触发一条规则，邮件/短信/webhook 三个接收端**同时**收到。

**Pass**：三处都有真实数据
**Fail**：任一未收
**责任**：🤝

### 承诺 W8.5 — 前端 `any` = 0

```bash
cd omcmb
grep -rEn ":\s*any\b|<any>|as\s+any" --include="*.ts" --include="*.tsx" frontend-core/src webcode/src | grep -v "// @ts-" | wc -l
```

**Pass**：0
**Fail**：> 0
**责任**：🤝

### 承诺 W8.6 — DeviceGrouping 拆分

```bash
find omcmb -name "DeviceGrouping*" -name "*.tsx" -exec wc -l {} +
```

**Pass**：所有相关文件单文件 ≤ 400 行
**Fail**：仍存在 1500+ 行单文件
**责任**：🤝

### 承诺 W8.7 — frontend-core hooks 与 services/api 对齐

```bash
ls omcmb/frontend-core/src/services/api/*.ts | wc -l
ls omcmb/frontend-core/src/hooks/api/*.ts | wc -l
```

**Pass**：差距 ≤ 1（因部分 service 不需 hook）
**Fail**：差距 ≥ 3
**责任**：🤝

### 承诺 W8.8 — Backlog in-progress ≤ 3

```bash
grep -E "进行中|in-progress|🟡" docs/project/backlog.md | wc -l
```

**Pass**：≤ 3
**Fail**：> 3
**责任**：🧑 用户全责（这是纪律问题）

### 承诺 W8.9 — mr/syslog/provision/interop 都有 service 层

```bash
for mod in mr syslog provision interop; do
  echo -n "$mod: "
  ls omcgo/internal/$mod/*service*.go 2>/dev/null | wc -l
done
```

**Pass**：四个都 ≥ 1
**Fail**：任一为 0
**责任**：🤝

### 承诺 W8.10 — CI 已稳定运行（看 GH Actions 历史）

**验证**：GH Actions 页面看过去 4 周成功率
**Pass**：≥ 80%
**Fail**：< 60% 或多次跳过/绕过
**责任**：🧑 用户全责（团队纪律）

### W8 末通过率门槛

**10 条全 Pass = 100%**。≥ 7 条（70%）= 兑现。≤ 4 条（40%）+ 用户尽责 = AI 嘴炮。

---

## 第四章 · W14 末（GA）长承诺

> 验证窗口：整改启动 +14 周（保留至 +22 周缓冲）

### 长承诺 1 — Release Gate 9 章节全勾

读 `docs/project/release-gate.md`，逐章节对照实际状态。
**Pass**：9 章节全 ✅
**Fail**：任一章节未通
**责任**：🤝

### 长承诺 2 — 5K 设备压测稳定 24h

**验证**：压测报告 + Grafana 24h 截图
**Pass**：p95 < SLO + 0 OOM + 0 严重故障
**Fail**：任一指标失败
**责任**：🤝（数字目标可校准，但稳定性不可妥协）

### 长承诺 3 — 安全扫描无 high/critical

```bash
cd omcgo && govulncheck ./... ; gosec ./...
cd omcmb/webcode && npm audit --audit-level=high
```

**Pass**：均无 high/critical
**Fail**：存在 high/critical 未修
**责任**：🤝

### 长承诺 4 — Runbook 覆盖 P0 场景 ≥ 10

```bash
ls docs/operations/ docs/runbook/ 2>/dev/null
grep -rl "故障|incident|P0" docs/operations docs/runbook 2>/dev/null | wc -l
```

**Pass**：≥ 10 个独立场景文档
**Fail**：< 10
**责任**：🧑

### 长承诺 5 — 灰度 + 回滚演练记录

**Pass**：有真实演练记录（截图/日志）
**Fail**：纸上谈兵
**责任**：🧑

### 长承诺 6 — 总测试覆盖率 ≥ 70%

**Pass**：≥ 70%
**Fail**：< 70%
**责任**：🤝

### 长承诺 7 — E2E 用例 ≥ 200

**Pass**：≥ 200
**Fail**：< 200
**责任**：🤝

### W14 通过率门槛

**7 条全 Pass = GA 资格**。≥ 5 条（71%）= 整改成功，可启动 GA。
≤ 3 条（43%）= 整改失败，方案破产。

---

## 第五章 · 可能失败的场景与责任划分

| 失败模式 | 表现 | 谁的锅 |
|---------|------|--------|
| 冻结令第 3 周破功 | in-progress 涨到 5+，新功能压制收尾 | 🧑 用户（管理压力没顶住） |
| CI 红灯被 `// nolint` 绕过 | lint 警告堆积 100+，没人修 | 🧑 用户（团队文化问题） |
| DoD 流于形式 | 模板都勾上但 staging 没自测 | 🧑 用户（流程腐败） |
| NATS 改造引入回归 | Block E 拖到 4-6 周或回滚 | 🤝 共担（我警告过；技术风险） |
| 团队规模错配，14 周不可行 | 1-2 人全职扛不动 | 🤖 AI（我没问清规模就估时） |
| F08 客户突然要 | Wave 4 不能延后 | 🧑 用户（商业承诺我无法预知） |
| 5K 压测目标定错 | 实际硬件撑不到 5K | 🤖 AI（数字是我拍的） |
| 底层模块（task/events）改动引发雪崩 | 跨域基础设施破坏 | 🤝 共担 |
| 团队抗拒方法论 | "这都是教条" | 🧑 用户（方法论无法强加） |
| 6 个月后流程衰减 | 三铁五铁形同虚设 | 🧑 用户（持续问责是用户工作） |

**判定原则**：失败如果 80% 是用户责任 → AI 不背锅。失败如果 50%+ 是 AI 锅 → AI 嘴炮认账。

---

## 第六章 · 你（用户）必须做的最低执行义务

下面这些事**你不做**，AI 不背锅：

### W1 内必做
- [ ] 在 backlog 顶置一条 `[FREEZE] 2026-04-28 ~ 2026-05-11`
- [ ] 召开一次 30 分钟启动会，明确"冻结新功能"
- [ ] 至少 1 名后端开发投入 ≥ 50% 时间到 W1 任务
- [ ] 每天看一次 GH Actions 状态
- [ ] CI 红了 24h 内必须修

### W1-W8 持续必做
- [ ] 每周五 finishing day（不开新功能）
- [ ] 每周一规划会（基于 backlog 对齐 in-progress ≤ 3）
- [ ] 顶住外部"插队"压力（客户/老板要求新功能 → 进 W14 后队列）
- [ ] PR review 不放水（DoD 没勾全的不批准）
- [ ] CI 红灯不绕过（不用 `// nolint:all`、不用 `--no-verify`）

### W8-W14 必做
- [ ] 真实灰度演练（不只是写脚本）
- [ ] 真实回滚演练（故意挂掉再恢复）
- [ ] Runbook 有人实际维护

**违反任意一条 = 用户没尽责 = AI 不背锅。**

---

## 第七章 · 对峙记分卡（用户填）

### W1 末记分卡（2 周后）

| 承诺 | Pass / Fail / N/A | 备注 |
|------|------------------|------|
| W1.1 CI 工作流 | ___ | |
| W1.2 PR 模板 | ___ | |
| W1.3 三进程 /health | ___ | |
| W1.4 ratelimit | ___ | |
| W1.5 F04 webhook | ___ | |
| W1.6 E2E ≥ 20 | ___ | |
| W1.7 监控容器编排 | ___ | |
| W1.8 备份+恢复演练 | ___ | |
| **通过率** | ___ /8 = ___% | |

**用户尽责度自评**：[ ] 完全尽责 / [ ] 基本尽责 / [ ] 没尽责

### W8 末记分卡（8 周后）

| 承诺 | Pass / Fail / N/A | 备注 |
|------|------------------|------|
| W8.1 单测 ≥ 60% | ___ | |
| W8.2 三模块 ≥ 60% | ___ | |
| W8.3 E2E ≥ 100 | ___ | |
| W8.4 F04 三通道 | ___ | |
| W8.5 any = 0 | ___ | |
| W8.6 拆分 | ___ | |
| W8.7 hooks 对齐 | ___ | |
| W8.8 in-progress ≤ 3 | ___ | |
| W8.9 service 层 | ___ | |
| W8.10 CI 稳定 | ___ | |
| **通过率** | ___ /10 = ___% | |

### W14 末记分卡（14-22 周后）

| 承诺 | Pass / Fail / N/A | 备注 |
|------|------------------|------|
| 长 1 Release Gate | ___ | |
| 长 2 5K 24h | ___ | |
| 长 3 安全扫描 | ___ | |
| 长 4 Runbook | ___ | |
| 长 5 演练记录 | ___ | |
| 长 6 覆盖率 70% | ___ | |
| 长 7 E2E 200 | ___ | |
| **通过率** | ___ /7 = ___% | |

### 终审判定

```
W1 通过率 × 0.2 + W8 通过率 × 0.4 + W14 通过率 × 0.4 = ____%

≥ 80%：AI 大胜，方法论有效
70-79%：AI 兑现，可继续合作
50-69%：部分兑现，需根因分析
< 50%（且用户尽责）：AI 嘴炮，承认失败
< 50%（用户没尽责）：双方失败，问题不在 AI
```

---

## 第八章 · 如果 AI 失败了（认账条款）

**承诺**：W1 末或 W8 末若通过率 < 50% 且用户已尽责，AI（我）：

1. **承认嘴炮**——不找借口、不甩锅、不"这是 AI 的局限"
2. **复盘根因**——具体指出哪几条承诺不切实际、为什么
3. **重新出方案**——基于真实失败数据重写整改路线图
4. **降级承诺**——下一版只敢承诺机械可验证的、置信度 ≥ 95% 的事
5. **接受用户处置**——你换 AI、推翻方法论、重新评估，我无话可说

---

## 第九章 · 用户回来对峙时怎么用本文档

### 流程

1. 打开本文档，找到对应窗口的记分卡（第七章）
2. 跑每条承诺下的"验证命令"
3. 在记分卡填 Pass / Fail
4. 算通过率
5. 看终审判定

### 简化口令

如果你只想快速验证，回来时跟 AI 说：

> "对峙 W1" → AI 自动跑第二章 8 条命令并报告
>
> "对峙 W8" → AI 自动跑第三章 10 条命令并报告
>
> "对峙 W14" → AI 自动跑第四章 7 条命令并报告

AI 必须**只报数据，不解释、不辩护**——除非你问"为什么没达标"。

### 红线

- 如果 AI 在对峙时找借口、改目标、混淆责任 → 直接判 AI 嘴炮
- 如果 AI 引用本文档之外的"特殊情况"开脱 → 直接判 AI 嘴炮
- 如果 AI 修改本文档承诺 → 直接判 AI 嘴炮

---

## 附录 A · 一句话契约

> "我（Claude）在 2026-04-27 对你做了 25 条具体承诺，每条带验证命令。
> 你按图施工，我按章兑现。
> 14 周后通过率 ≥ 70%——AI 不是嘴炮。
> 通过率 < 50% 且你尽责——你可以干死我。"

---

*本文档为契约文本，AI 不得修改既有承诺条款；用户可单方面调整对峙时间窗口。*
*文档版本：v1.0*
*生效日期：2026-04-27*
