# /ship — 全流程编排器（One-Click Ship）

> **唯一职责：控制流程。** 一键从「一个想法 / 一个 GitHub Issue」贯穿到「feature 分支提交 + 开 PR 待合入」。
> 本 skill **只做调度 + 门控 + 委派**，自身**不实现**任何 PRD 模板 / 审查清单 / 测试逻辑 / 提交细节——那些归各子技能。
> 取代已下线的 `/dev-pipeline`（其 OMC 硬门精华已折叠进 §硬门；S0–S7 完整 rationale 见归档 `docs/project/dev-pipeline-design-20260420.md`）。
> **任务源 = GitHub Issues**（`gh` CLI，详见 `docs/agents/issue-tracker.md`）。

> 🔒 **铁律（不可绕过，最高优先级）：一切合入经 PR。** `/ship` **永不**直接 `commit` / `merge` / `push` 到 `main` / `master`；P5 起始终在 feature 分支（`<type>/NN-<slug>`）工作，P9 经 `gh pr create` 开 PR。**合入 main 由 review 通过后进行，ship 自身不 merge、不直推 main。** `--force` / force-push 改写共享分支永禁（settings.json 已 deny）。任一环节直接动 main = 流程失败，立即停下纠正。

---

## 参数

`$ARGUMENTS` 按下列顺序匹配：

| 参数 | 模式 | 说明 |
|------|------|------|
| 空 | full（默认） | 推断当前阶段，自动向前推进到 P9 开 PR |
| `status` | 子命令 | 打印 GitHub Issues 看板（按 triage 标签）+ 当前分支阶段 |
| `audit` | 子命令 | 推断当前阶段、已过门、下一步——**只报告不推进** |
| `#NN` / `NN` | 入口 | 从指定 GitHub Issue 起步：已 `ready-for-agent` 则直入 P5，否则先回 P4 分诊 |
| `P1`..`P9` 或 `align`/`spec`/`slice`/`triage`/`build`/`verify`/`review`/`commit`/`pr` | 单阶段 | 只跑一个阶段（别名↔阶段：align=P1 spec=P2 slice=P3 triage=P4 build=P5 verify=P6 review=P7 commit=P8 pr=P9，`close` 为 `pr` 同义） |
| `--fast <bugfix\|hotfix\|docs\|refactor>` | 快速通道 | 按 §快速通道裁剪前置阶段，再进入 full |

---

## 流水线（9 阶段，每阶段只委派 + 查门）

每进入一个阶段先 echo：`─── P<N> <名称> ───  委派: /<skill>  硬门: <list>`。
门未过 → **停下、报告、等修复**；修复后 `/ship P<N>` 重跑该阶段。

**分支铁律**：进入 P5 前若 `git branch --show-current` ∈ {`main`,`master`} → 先 `git switch -c <type>/NN-<slug>`（type 取 Issue category：feat/fix/refactor/docs/chore）。P5–P9 全程在该 feature 分支，**绝不**在 main 上 commit/merge；最终经 P9 的 PR 合入（见 §硬门 9）。

| 阶段 | 委派技能 | 入口条件 | 硬门（过则推进） | HITL |
|------|---------|---------|----------------|------|
| **P1 对齐** | `/grill-with-docs` | 需求模糊 / 跨模块 / 有架构取舍 | 决策树各分支已定；`CONTEXT.md`/`docs/adr/` 同步 | ✋ 逐问确认 |
| **P2 立项** | `/to-prd` | 已对齐，需求成形 | PRD 发布为 GitHub Issue + 打 `ready-for-agent`（to-prd 自带，不再重复分诊） | ✋ 确认 seam |
| **P3 切片** | `/to-issues` | PRD 含 > 1 个垂直切片 | 切片为可独立领取的**子 Issue**（tracer-bullet），依赖序发布 | ✋ 确认粒度/依赖 |
| **P4 分诊** | `/triage` | **仅 P3 拆出的子 Issue** 未就绪 | 每子 Issue 恰好一个 category + 一个 state 标签；`ready-for-agent` 附 agent brief | ✋ 维护者拍板 |
| **P5 实现** | `/tdd`（卡死时 `/diagnose`） | 有 `ready-for-agent` 的 Issue（**先确保在 feature 分支，非 main**） | 见 §硬门 1–3、9（go build / go test / typecheck 退出码；分支非 main） | — AFK |
| **P6 验证** | `/e2e`、`/acs-stress-test`、`/verify` 或 `/run` | P5 门全绿 | 见 §硬门 4–5（E/R≥1、迁移双向演练） | — AFK |
| **P7 审查** | `/review`（触 auth 追加 `/security-review`）+ `/simplify` | P6 done | 见 §硬门 6–7（审查报告无 CRITICAL）；DoD 逐项勾选（`docs/project/dod.md`） | — AFK |
| **P8 提交** | `/commit` | P7 done | 见 §硬门 8；commit 成功；footer `Closes #NN` | — AFK |
| **P9 交付·PR** | （内联 `gh`）+ 卡续时 `/handoff` | P8 done | 见 §硬门 9；`git push -u origin <feature分支>` + `gh pr create`（body 含 `Closes #NN`）；输出 PR 链接；**不直接合 main**，合入待 review；给下一 Issue 建议 | ✋ 合并经 review |

> **不重复分诊**：P2 的 PRD Issue 已带 `ready-for-agent`（`/to-prd` 自带），P4 只 triage P3 拆出的子 Issue。**单切片 PRD**（不满足 P3 入口）从 P2 直接跳 P5，跳过 P3/P4。
> **专家视角**：每阶段按**改动文件路径**自动叠加 `docs/expert-personas.md` 的对应专家（如触 `internal/acs/`→TR-069，`migrations/`→数据，`omcmb/`→前端）。本 skill 不重述清单，只提醒激活。

---

## 入口推断（full 模式的「一键」逻辑）

**规则自上而下，首条命中即停**，从该 P 自动向前跑到 P9：

1. `git status --porcelain` 有暂存变更 + 存在当日审查报告 → **P8 提交**
2. 有未提交代码变更（`git diff HEAD --stat` 非空）→ 有新端点/迁移待验证则 **P6 验证**，否则 **P7 审查**
3. 当前分支名 / 近期 commit 含 `#NN`，或存在 `ready-for-agent` Issue 且无代码变更 → **P5 实现**
4. 存在 `ready-for-human` Issue 且无代码变更 → **停下报告**「需人工实现」，不自动进 P5
5. 对话中已有成形需求/方案但 GitHub 无对应 Issue → **P2 立项**（仍模糊则先 P1）
6. 对话中只有一句模糊想法 → **P1 对齐**

推断后先 echo 一行结论与起步阶段，再开跑。`audit` 模式只到这一步、不推进。

---

## 硬门（不可绕过，继承自 dev-pipeline 精华）

下游门依赖上游门，任一不过即停在当前阶段。**机械门** = ship 跑命令看退出码自己把关；**质量门** = 判断性、归 `/review`，ship 只认审查报告结论、**不自己 grep diff**。

1. 〔机械·P5〕`cd omcgo && go build ./...` 不过 → P5 起全部不过
2. 〔机械·P5〕`go test ./...` 不全绿 → P5 出口不过（成功 + 失败两条路径都要有）
3. 〔机械·P5〕前端涉及 `cd omcmb/webcode && npm run typecheck` 不过 → P5 不过
4. 〔机械·P6〕新端点 E/R < 1（E2E 新增 claim 数 / 新路由数）→ 不过；新 metric/log 名 `grep` 不到（存在性检查）→ 不过
5. 〔机械·P6〕新迁移 `.up.sql`/`.down.sql` 不配对、或本地 `migrate-up`/`migrate-down` 未各演练一次 → 不过
6. 〔质量·P7〕审查报告含 CRITICAL → 不过。CRITICAL 清单**归 `/review`**，含：`if carrier == "cmcc|ctcc|cucc"` 硬编码、字符串拼接 SQL、裸 `panic`、公共接口新增 `any`/`interface{}`、删测试 / 降安全等
7. 〔质量·P7〕触 `internal/admin/` 或 `middleware/auth*` 未跑 `/security-review` → 不过
8. 〔机械·P8〕`git commit --no-verify` → settings.json 已 deny，硬失败；pre-commit 失败修根因，不 `--amend`，新开一笔
9. 〔机械·P9〕**PR 铁律**：开 PR 前 `git branch --show-current` ∉ {`main`,`master`}（否则先建 `<type>/NN-<slug>` 并把提交移过去）；P9 出口 = `gh pr create` 返回 PR URL；**严禁** `git merge` / `git push` 直推 `main`、严禁 `--force`（settings.json deny）；PR 合入由 review 通过后进行，**ship 不自动 merge**

无对应改动的门标 `N/A`（如无新端点则 E/R = N/A）。门 1–3→P5、4–5→P6、6–7→P7、8→P8、9→P9，无孤儿门。

---

## 快速通道（裁剪规则，继承 dev-pipeline §C）

`#NN` 读 Issue 的 category 标签（**仅 `bug`/`enhancement` 两种**）自动选 bugfix / enhancement 行；refactor/docs/hotfix 行无对应 triage 标签，只能经 `--fast <type>` 显式触发：

| 类型 | 可省 | 必过 |
|------|------|------|
| enhancement（新功能） | 无 | P1–P9 |
| bugfix | P1, P2, P3 | P4 轻（确认 Issue 已就绪）+ P5–P9 |
| refactor（行为不变） | P1, P2 | P3 登记 + P5–P9 |
| docs / config | P1, P2, P3, P6 | P5 适用 + P7 轻 + P8, P9 |
| hotfix | P1, P2 | P5–P8 + P9（**强制补 postmortem 到 `docs/project/risk-register.md`**） |

裁剪的阶段必须在 P8 commit body 的 `Skip: P1,P2,...` 字段记录。
> refactor「行为不变」**不豁免硬门 6**：若触及 `migrations/` 仍须过迁移双向演练。
> **P9（PR）任何快速通道都不可裁剪**：hotfix / docs 一律经 PR 合入，紧急时在 PR 上加急 review，但**绝不直推 main**。

---

## §子命令

### status
查 GitHub Issues（`gh issue list`）按 triage 标签分桶计数（`needs-triage`/`needs-info`/`ready-for-agent`/`ready-for-human`；`wontfix` 与已关闭不计入活跃看板），列当前分支推断阶段。仅终端输出。

### audit
执行「入口推断」全过程，输出当前 Issue、推断阶段、已过/未过门、下一步建议。**不推进。**

---

## 与子技能的调用约定

| 被调技能 | 阶段 | 触发 |
|---------|------|------|
| `/grill-with-docs` | P1 | 需求模糊 / 跨模块 / 架构取舍 |
| `/to-prd` | P2 | 需求成形，发 PRD Issue |
| `/to-issues` | P3 | PRD 拆垂直切片 |
| `/triage` | P4 | 子 Issue 状态机推进 |
| `/tdd` | P5 | 红-绿-重构实现 |
| `/diagnose` | P5 | 硬 bug / 性能回归卡死 |
| `/e2e` · `/acs-stress-test` | P6 | 新端点 / acs 性能敏感 |
| `/verify` · `/run` | P6 | 跑起来观察真实行为 |
| `/review` · `/security-review` · `/simplify` | P7 | 审查 / 安全 / 精简 |
| `/commit` | P8 | 生成审查报告 + 规范化提交 |
| `/handoff` | P9 | 工作未完，压缩交接 |

**原则**：`/ship` 不重造任何既有功能，只作编排者按序调用、在阶段间查门。

---

## 常见场景

```
/ship                 # 一键：从当前状态自动推进到开 PR（在 feature 分支，不碰 main）
/ship status          # 看 GitHub Issues 看板 + 当前阶段
/ship audit           # 这分支到哪了？哪些门过了？下一步？
/ship #42             # 领取 Issue #42（就绪则直入实现，否则先分诊）→ feature 分支 → 开 PR
/ship --fast bugfix #42   # bugfix 快速通道（仍经 PR 合入，不可裁剪 P9）
/ship pr              # 只跑 P9：push feature 分支 + gh pr create
/ship P7              # 只重跑审查阶段
```
