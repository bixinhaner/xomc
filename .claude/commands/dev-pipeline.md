# 开发流水线 (Dev Pipeline)

将 L0 Backlog 中的任务按 S0→S7 七阶段门控推进到本地提交。部署流水线是最后防线，这个 skill 是**第一道防线**。

**完整设计**：`docs/project/dev-pipeline-design-20260420.md`
**任务源**：`docs/project/backlog.md`（唯一任务清单）

---

## 参数

`$ARGUMENTS` 支持以下形态，按下列顺序匹配：

| 参数 | 模式 | 说明 |
|------|------|------|
| 空 | full | 自动推断当前阶段，从该阶段接续（默认） |
| `status` | 子命令 | 打印 Backlog 仪表盘 + 健康度告警 |
| `audit` | 子命令 | 推断当前分支所处阶段、已过门、下一门 |
| `next [N]` | 子命令 | 出队前 N 条可做任务（默认 5） |
| `pick T-NNNN` | 子命令 | 选定任务进入 S0→S7 |
| `backlog add "<title>"` | 子命令 | 登记新想法到 Proposed |
| `triage` | 子命令 | 批量分诊 Proposed → triaged/deferred/rejected |
| `plan sprint-NN` | 子命令 | Sprint Planning：triaged → planned |
| `validate` | 子命令 | 检查依赖循环/孤儿/引用断裂 |
| `S0`..`S7` 或 `intake`/`planning`/`design`/`implement`/`verify`/`review`/`commit`/`handoff` | 单阶段 | 只跑一个阶段 |
| `--fast <bugfix\|hotfix\|refactor\|docs>` | 快速通道 | 启用裁剪规则（见 §C） |

---

## 执行步骤

### Step 1: 预检

1. 确认 `docs/project/backlog.md` 存在。不存在 → 提示"请先创建 Backlog（参考 dev-pipeline-design §11.12 Bootstrap）"并停止。
2. 解析 `$ARGUMENTS`，确定路由（子命令 / 单阶段 / full）。
3. `git rev-parse --abbrev-ref HEAD` 获取当前分支；`git status --porcelain` 获取工作区状态。
4. 记录时间戳（后续写入变更日志用）。

### Step 2: 路由分发

按参数匹配下面对应章节，执行完毕即结束：

- `status` → §A1
- `audit` → §A2
- `next [N]` → §A3
- `pick T-NNNN` → §A4（触发 S0→S7 主流程）
- `backlog add "..."` → §A5
- `triage` → §A6
- `plan sprint-NN` → §A7
- `validate` → §A8
- `S0..S7` 单阶段 → §B0..§B7
- full（默认）：跑 §A2 audit 推断阶段 → 从该阶段起按 §B 依次推进到 S7
- `--fast <type>` → 按 §C 裁剪规则跳过对应前置阶段，再进入 full

---

## §A 子命令章节

### §A1 status — 打印仪表盘

1. 读 `docs/project/backlog.md` §2 仪表盘段。
2. 按 schema 重新统计（防表格与实际行数不一致）：扫 §3/§4/§5/§6/§7/§8 分区，计数每个 state。
3. 健康度检查（dev-pipeline-design §11.11）：
   - `proposed` 积压 > 7 天 → 告警"本周未 triage"
   - `blocked` > 3 → 告警"开阻塞专题"
   - P0 任务数 > 当前 Sprint 容量 → 告警"优先级重排"
   - 单 Task `in_dev` > 5 天（比较 Updated 字段）→ 告警"3 次失败重评"
   - 仅含 done 未含 in_dev 且 Sprint 未结束 → 提示"Sprint 空转"
4. 输出格式：
   ```
   Backlog Status @ <日期>
   ────────────────────────────────────
   Total:       N     Done:      N
   In-flight:   N     Planned:   N
   Triaged:     N     Blocked:   N
   Proposed:    N     Deferred:  N  Rejected: N

   P0 风险关闭: X/5（<具体 Risk ID 列表>）
   E2E 累计用例: X/200
   当前 Sprint: sprint-NN（第 W 周）

   健康告警：
     [●] 无告警 / [△] <具体告警>
   ```

### §A2 audit — 阶段审计

**目的**：当前分支在 S0–S7 的哪个阶段？哪些门过了？下一步该做什么？

1. 扫描信号：
   - `git status --porcelain` 有未提交代码变更 → 可能 S3/S4
   - `git diff HEAD --stat` 非空 → 有工作在飞
   - `ls docs/review-report/$(date +%Y%m%d)/` 有当日 review 报告 → S5 已过
   - `git log -5 --format=%B` 中匹配 `PRD: .* Sprint: .* Risk: .*` footer → S6 已过
   - `go build ./...` 通过 → S3 出口门
   - `go test ./...` 全绿 → S4 首门
2. 识别当前 Task：
   - 从 `git log -20 --format=%B` 抓 `T-NNNN` 引用
   - 未抓到 → 询问用户 or 参考 backlog `in_dev` 状态任务
3. 对照 Task.State 和信号交叉判断：
   - Task.State = planned，无代码变更 → S2 design 待进
   - Task.State = planned，有设计备忘 + 代码变更 → S3 in_dev
   - Task.State = in_dev，build+test 全绿 → S4 verify 待进
   - Task.State = in_review → S5 进行中
   - 近期 commit 有完整 footer → S6/S7
4. 输出：
   ```
   Audit Result
   ────────────────────────────────────
   当前 Task: T-NNNN <title>（State: <state>）
   推断阶段: S<N> <name>

   已过门:
     [✓] S0 (PRD 存在: <path>)
     [✓] S1 (Sprint: <sprint>, Owner: <owner>)
     ...

   当前阶段未过门:
     [ ] <门 1>
     [ ] <门 2>

   下一步建议: <具体动作>
   ```
5. audit 模式**不执行**推进，只报告。

### §A3 next — 出队

1. 读 backlog §3 Active + §4 Triaged 两表。
2. 读 §6 Done 建立 done-set。
3. 识别**累计型上游**（设计 §11.7.1）：扫描 §3 Active 中 `Sprint` 含 `..` 跨度 且 `Est=XL` 的 Task → 累计型集合。
4. 过滤：
   - `State ∈ {triaged, planned}`（排除 in_*, blocked）
   - 每个 `Deps` 字段内的依赖逐个判断：
     - 普通 `T-xxxx` → 必须在 done-set，否则此 Task 入"等待清单"
     - `T-xxxx@累计≥N` 或上游属累计型集合 → 若上游 State ∈ {in_dev, in_review} 且 Progress（从 Notes 读或默认 0）≥ 阈值 N → 视为已满足；否则入"等待清单"并标 `⚠️ 累计未达`
5. 主候选排序：Prio 升序 → Updated 升序（FIFO）；累计型依赖放行的候选末尾加 `⚠️ 累计型` 标签
6. 等待清单排序：Prio 升序，附注阻塞依赖（区分 `(done-pending)` / `(cumulative-short)`）
7. 取候选前 N（默认 5），输出：
   ```
   ✅ 可做任务（Top N，按 Prio → FIFO）:
   # | ID | Title | Prio | Sprint | Deps | Est
   1 | T-xxxx | ... | P0 | sprint-01 | — | M
   5 | T-0025 | RC 冻结 | P0 | sprint-07 | T-0006@累计≥150 | M  ⚠️ 累计型

   ⏸ 等待解除依赖:
   - T-xxxx 等待 T-yyyy (done-pending)
   - T-zzzz 等待 T-0006@累计≥150 (cumulative-short: 当前 0/150)

   操作: /dev-pipeline pick T-xxxx 即可开始
   ```

### §A4 pick T-NNNN

1. 在 backlog 主表查找 T-NNNN；不存在 → 停，提示"未登记"。
2. 验证 State ∈ {triaged, planned, in_*}：否则拒绝（done/deferred/rejected 不再工作）。
3. 验证 Deps 全 done：否则转 blocked 并提示。
4. 根据 Task.Type 决定裁剪（见 §C）。
5. 按 §B0→§B7 依次进入，每阶段开始时 echo：
   ```
   ─── 进入 S<N> <stage-name>（T-NNNN） ───
   激活专家: <personas list>
   硬门: <list>
   制品: <output path>
   ```
6. 每阶段门未过 → 停下、报告、等用户修复；修复后可 `/dev-pipeline S<N>` 重跑该阶段。

### §A5 backlog add "<title>"

1. 读 backlog.md，扫描最大 `T-NNNN`。
2. 生成下一 ID = `T-$(printf %04d $((max+1)))`。
3. 在 §5 Proposed 表追加一行：
   ```
   | T-NNNN | <title> | <user> | <today> | — |
   ```
4. 更新 §2 仪表盘 proposed 计数。
5. 更新 §10 变更日志追加一行。
6. 输出：`✅ 已登记 T-NNNN，等待下周 Triage（或执行 /dev-pipeline triage）`。

### §A6 triage

1. 列出 §5 Proposed 所有条目。
2. 逐条与用户交互（每条必答）：Type / Domain / Prio / Est / Deps / Risks / 判决（triaged/deferred/rejected，后两者需填 Reason）
3. 判决为 `triaged` → 移到 §4 Triaged；`deferred` → §7；`rejected` → §8。
4. 从 §5 Proposed 删除。
5. 更新仪表盘 + 变更日志。

### §A7 plan sprint-NN

1. 检查 `docs/project/sprint/sprint-NN.md` 是否存在；不存在 → 基于 `TEMPLATE.md` 新建。
2. 列出 §4 Triaged，按 Prio 升序。
3. 与用户逐条确认"进入 sprint-NN ? (y/n)"。
4. 计算 Sprint 容量：`2 周 × Owner 可用天数 × 0.8 buffer`。
5. 选中者：State `triaged → planned`；回填 Sprint + Owner；写入 sprint-NN.md。
6. 若任一选中 Task 的 Deps 不在本 Sprint 或更早 → 拒绝，提示重排。
7. 更新 backlog 主表 + 仪表盘 + 变更日志。

### §A8 validate

1. 解析 backlog 所有 Task 的 Deps → 建 DAG。
2. 检测：
   - **循环依赖**：DFS 找回边
   - **孤儿依赖**：Deps 指向不存在的 ID
   - **状态悖论**：Task.State = planned 但 Deps 未 done；Task.State = done 但 Deps 未 done。**例外**：累计型依赖（设计 §11.7.1，上游 Sprint 跨度+XL 或 `T-xxx@累计≥N` 语法）不计悖论
   - **Sprint 错位**：Task 进了 sprint-X，Deps 在 sprint-Y 且 Y > X。累计型上游的 Sprint 取其结束 Sprint 判定
   - **PRD 空缺**：feat 类型且 Prio ∈ {P0, P1} 但 PRD 字段空
   - **累计阈值无效**：`T-xxx@累计≥N` 语法中 N 必须为正整数；上游必须确为累计型
3. 输出违规列表，每条附修复建议；全过 → `✅ Backlog 依赖关系一致`。

---

## §B 阶段章节

每阶段详细 rationale 见 `dev-pipeline-design §5.S0..S7`，此处仅列操作清单。

### §B0 S0 intake — 立项（PM 主持）

**进入条件**：Task.State ∈ {triaged, planned}
**激活专家**：`§16.10 PM` + `§16.4 电信` + `§16.1 架构`
**活动**：
1. 按 Task.Type 判断：`bugfix`/`docs`/`refactor` → 提示可跳 S0；`feat`/`hot`/`proc` → 继续
2. 查 Task.PRD 字段：有路径 → 读+校验；空 → 基于 `docs/project/prd/TEMPLATE.md` 起草
3. 七要素检查：业务背景 / 用户故事 / 验收（Given-When-Then）/ 运营商差异矩阵 / 非目标 / 依赖 / 度量
4. 运营商差异矩阵非空（CMCC/CTCC/CUCC 至少一列有内容或明确"无差异，本条一致"）
5. 回写 Task.PRD = 路径

**出口门**：
- [ ] PRD 七要素全
- [ ] 运营商差异矩阵非空
- [ ] ≥1 条可测验收（含 Then）

**阻塞**：缺任一 → 停，不进 S1。

### §B1 S1 planning — 规划（PgM 主持）

**进入条件**：S0 done
**激活专家**：`§16.11 PgM` + `§16.1 架构`
**活动**：
1. 读 Task.Deps，逐个在 Done 区验证
2. 估 Task.Est（如未估）
3. 查目标 Sprint 剩余容量
4. 指派 Owner（单人）
5. 若涉及 Risk ≥ P1 → 更新 `risk-register.md`
6. 回写 Task.Sprint / Task.Owner；State: `triaged → planned`

**出口门**：
- [ ] Sprint 条目新增
- [ ] Owner 明确
- [ ] Deps 全 done
- [ ] 若 P0/P1，risk-register 有对应条目

### §B2 S2 design — 设计（架构 + 领域专家）

**进入条件**：S1 done
**激活专家**：按改动路径叠加
**活动**：
1. 路径触发：
   - `internal/acs/` → TR-069 三问（会话状态机/RPC/厂商差异）
   - `internal/config|pm|alarm/` → 电信+数据专家
   - `internal/admin/` 或 `middleware/auth*` → 安全专家
   - `migrations/` → 数据专家（索引/hypertable/回滚）
   - `omcmb/` → 前端专家
2. 产出设计备忘：接口签名 / 路由 / 迁移 up/down 草案 / Carrier 扩展点 / 新 metric+log 名字清单
3. 跨模块（omcgo + omcmb）→ 明确"联合变更"或强制拆分
4. State: `planned → in_design`

**出口门**：
- [ ] 接口契约明确
- [ ] 迁移草案（若涉）
- [ ] Carrier 差异点列出
- [ ] 观测埋点名字列出
- [ ] 待定点 < 3

### §B3 S3 implement — 开发（Go/前端专家）

**进入条件**：S2 done
**激活专家**：`§16.2 Go` / `§16.6 前端` + 领域专家
**活动**：
1. 先写测试（成功+失败两条路径）
2. 按 `handler → service → repository` 分层落地
3. 每子任务后增量 commit（本地）
4. 每 2 次迭代后调 `/simplify` 自检
5. 遇阻 3 次停下重评（CLAUDE.md §9）
6. State: `in_design → in_dev`

**出口门**（硬命令）：
- [ ] `go build ./...` 通过
- [ ] `go test ./...` 全绿
- [ ] 前端 `npx tsc --noEmit` 通过（若涉）
- [ ] 无新增 TODO/FIXME/panic("not implemented")
- [ ] 无新增 `if carrier == "cmcc|ctcc|cucc"` 硬编码
- [ ] 公共接口无新增 `any` / `interface{}`

### §B4 S4 verify — 本地验证（测试+数据+运维）

**进入条件**：S3 门全绿
**活动**：
1. `golangci-lint run`
2. 前端涉及：`cd omcmb/webcode && npm run lint`
3. 迁移涉及：`bash scripts/check-migrations.sh` + 本地 `migrate-up` + `migrate-down` 各一次
4. 新端点：新路由数 R vs E2E 新增 claim 数 E → **E/R ≥ 1**
5. 新 metric/log：每个 `grep -rn <name>` 返回 ≥ 1
6. **累计型依赖核销**（若 Task.Deps 含 `@累计≥N`）：读上游 Task.Notes 的 `Progress: 累计 NN`，NN ≥ N 才算过；否则停
7. 产出 `docs/review-report/YYYYMMDD/verify-<hash>.md`

**出口门**：
- [ ] 所有命令绿
- [ ] E/R ≥ 1（无新端点则 N/A）
- [ ] 迁移双向演练通过（无迁移则 N/A）
- [ ] metric/log 名全部能找到
- [ ] 累计型依赖阈值全达标（无则 N/A）

### §B5 S5 review — 审查

**进入条件**：S4 done
**活动**：
1. 调 `/review` 主入口
2. 路径触及 `internal/admin/` 或 `middleware/auth*` → **强制**追加 `/security-review`
3. 调 `/simplify` 最终查一遍
4. 逐项勾选 `docs/project/dod.md`
5. State: `in_dev → in_review`

**出口门**：
- [ ] review 报告无 P0/P1 未解
- [ ] 安全审查（若触发）无 P0
- [ ] DoD 每项打勾或 N/A（附理由）

### §B6 S6 commit — 提交（QA 主持）

**进入条件**：S5 done
**活动**：
1. 调 `/commit`
2. **Footer 三元组校验**（本 skill 独有）：必须含
   ```
   PRD: <路径> 或 N/A (<原因>)
   Sprint: <sprint-NN>
   Risk: <R-NNN 或 ->
   Backlog: T-NNNN
   Review: <report 路径或 N/A (skipped per §C <type>)>
   ```
3. pre-commit hook 失败 → 修根因，**严禁** `--no-verify`
4. 失败后**不**使用 `--amend`，新开一笔 commit

**出口门**：
- [ ] commit 成功
- [ ] Footer 三元组齐全
- [ ] hash + message 回显

### §B7 S7 handoff — 收尾（QA + PgM）

**进入条件**：S6 done
**活动**：
1. 输出"本地已提交（hash xxx），未推送远端。如需推送请明确告知"
2. 更新 sprint-NN.md 条目 → Done
3. 更新 backlog 主表：Task.State `in_review → done`；填 Closed
4. 追加到 §6 Done 分区（带 Closing Evidence）
5. 更新 §10 变更日志
6. 关联 Risk mitigation 完成 → 更新 risk-register
7. 走快速通道 → **强制**补 postmortem 到 risk-register
8. E2E 覆盖增量记账到 §2 仪表盘

**出口门**：
- [ ] Sprint 状态更新
- [ ] Backlog Task 关闭
- [ ] 快速通道均补 postmortem

---

## §C 快速通道 / 裁剪规则

| 改动类型 | 可省阶段 | 必过阶段 |
|---------|---------|---------|
| feat（新功能/端点） | 无 | S0–S7 |
| bugfix | S0, S1, S2 | S3, S4, S5, S6, S7 |
| refactor（行为不变） | S0 | S1 登记 + S2 轻 + S3–S7 |
| docs / config | S0, S1, S2, S4 | S3 适用 + S5 轻 + S6, S7 |
| hotfix | S0, S1 | S2 简 + S3–S6 + S7（**强制 postmortem**） |
| proc（流程/工具链） | S0 可省 | S1 登记 + S2 轻 + S3–S7 |

判定方式：`pick T-NNNN` 读 Task.Type 自动裁剪；或 `--fast <type>` 显式。

**裁剪的阶段**必须在 S6 commit footer 的 `Skip: S0,S1,...` 字段记录。

---

## §D 公共规则

### D1 专家激活（与 `CLAUDE.md §16` 对齐）

- **按路径**：触及 `internal/acs/` 必激活 TR-069；`migrations/` 必激活数据；…（见 CLAUDE.md §16 激活规则）
- **按阶段**：S0→PM；S1→PgM；S5/S6/S7→QA；阶段可叠加路径触发

### D2 制品路径约定

| 阶段 | 产出 | 路径 |
|------|------|------|
| S0 | PRD | `docs/project/prd/F{NN}-{slug}.md` |
| S1 | Sprint 条目 | `docs/project/sprint/sprint-NN.md` |
| S2 | 设计备忘 | PRD 附录（追加"## 设计备忘"节） |
| S4 | 验证日志 | `docs/review-report/YYYYMMDD/verify-<hash>.md` |
| S5 | 审查报告 | `docs/review-report/YYYYMMDD/REVIEW_<hash>_<author>_<scope>.md` |
| S6 | commit | 本地 git log |
| S7 | 状态回写 | backlog §6 + Sprint + risk-register |

### D3 Commit Footer 三元组（S6 硬门）

```
<type>(<scope>): <中文简要描述>

<body>

PRD: docs/project/prd/F04-alarm-notification.md
Sprint: sprint-01
Risk: R-001
Backlog: T-0007
Review: docs/review-report/20260420/REVIEW_abc1234_author_F04.md

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
```

- PRD 空允许（bugfix/docs）但需填 `N/A (type=bugfix)`
- Sprint / Risk / Backlog 不允许空
- Review 无报告则填 `N/A (skipped per §C <type>)`

### D4 硬门合集（不可绕过）

- `go build ./...` 不过 → 下游全部不过
- `go test ./...` 不过 → S4 起不过
- `git commit --no-verify` → settings.json deny
- 新迁移 up/down 不配对 → S4 不过
- 新端点 E/R < 1 → S4 不过
- review 报告有 P0 → S5 不过
- DoD 未打勾且未 N/A → S5 不过

---

## §E 与既有 skill 的调用约定

| 被调 skill | 何时调 | 触发阶段 |
|-----------|--------|---------|
| `/review` | S5 入口 | §B5 |
| `/security-review` | admin / middleware/auth* 触及 | §B5 自动追加 |
| `/simplify` | S3 中期 + S5 末端 | §B3 / §B5 |
| `/commit` | S6 入口 | §B6 |
| `/e2e` | S4 验证新端点 | §B4 |
| `/acs-stress-test` | 性能敏感 (`internal/acs/`) | §B4 / §B5 |

**原则**：本 skill 不重造既有功能，只作编排者调用。

---

## §F 输出与日志

- 每次执行后终端输出阶段推进摘要
- 可选写入 `docs/review-report/YYYYMMDD/dev-pipeline-<hash>.log`
- `audit` / `status` 仅终端输出，不落盘

---

## 常见场景速查

**场景 1：接手新工作不知道做什么**
```
/dev-pipeline next
→ 看 Top 5 → /dev-pipeline pick T-xxxx
```

**场景 2：分支做了一半接着走**
```
/dev-pipeline audit
→ 看当前阶段与未过门 → /dev-pipeline S<N> 单阶段推进
```

**场景 3：临时记想法**
```
/dev-pipeline backlog add "F04 告警导出 CSV"
```

**场景 4：hotfix**
```
/dev-pipeline --fast hotfix pick T-xxxx
→ 跳 S0/S1，直接 S2 简 + S3→S7 + 强制 postmortem
```

**场景 5：看整体状态**
```
/dev-pipeline status
```

**场景 6：Backlog 自洽检查**
```
/dev-pipeline validate
```

---

**完整 rationale / 专家评议 / bootstrap 策略**：`docs/project/dev-pipeline-design-20260420.md`
**任务状态表**：`docs/project/backlog.md`
**DoD 硬门清单**：`docs/project/dod.md`
