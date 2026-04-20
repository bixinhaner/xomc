# OMC 开发流水线 Skill 设计方案（Dev Pipeline）

> **文档日期**：2026-04-20
> **作者**：Claude（整合 12 位专家视角）
> **背景**：部署流水线（L3 门控 + CI + branch protection）已建立——这是**最后防线**；但缺少一条把质量前置到编码阶段的**开发流水线**，导致缺陷在 PR/合入时才被发现，返工成本高。
> **定位**：以 skill 形式把既有 DoD / Release Gate / 专家 personas / 现有 skills 串成一条"从立项到提交"的分阶段门控流水线，让开发期就按节拍过关。
> **命名**：`/dev-pipeline`（或别名 `/devflow`）。

---

## 1. 为什么要做这条流水线

### 1.1 单靠部署流水线的缺口

| 环节 | 部署流水线能挡住吗？ | 成本 |
|------|-------------------|------|
| PRD 缺失、需求模糊 | 挡不住 | 代码写错了再改，最贵 |
| 运营商差异硬编码 | 部分（lint 规则） | review 打回，中等 |
| 设计过度 / 抽象过早 | 挡不住 | 长期技术债 |
| 没写 E2E 用例 | 挡不住（DoD 需人工勾选） | 放过去就烂账 |
| 迁移 up/down 不配对 | 可（check-migrations.sh） | 便宜 |
| 裸 panic / 字符串拼 SQL | 可（golangci-lint） | 便宜 |
| 观测埋点缺失 | 挡不住 | 线上盲发 |

结论：**部署流水线只能兜住"代码级硬约束"**，需求/设计/收尾等"软性但致命"的问题必须在**开发期**挡住，否则只能靠人自觉，和 CLAUDE.md §10.绝不要 里那些"靠纪律"条款同样脆弱。

### 1.2 与既有体系的关系（不是另起炉灶）

```
L0 Backlog（任务状态表 / 特性矩阵）            ← §11 新增
        ↓ 取任务（按依赖+优先级出队）
L1 角色层（CLAUDE.md §16：12 位专家）
        ↓ 被 skill 调用
L2 制品层（docs/project/：PRD / Sprint / DoD / Release Gate / Risk）
        ↓ 被 skill 读写
L3 硬门层（CI / branch protection / check-migrations.sh）
        ↑ 末端兜底
```

**新增的 dev-pipeline 不取代以上任何一层**，它只做一件事：按阶段**依次激活**正确的专家、产出正确的制品、调用正确的工具，让开发者不用凭记忆挨个过。

> **流水线的上游缺一不可**：如果没有一张"谁排第几、依赖谁、现在什么状态"的总表，流水线永远不知道下一条该做什么；所以新增 **L0 Backlog 层**，详见 §11。

---

## 2. 设计目标与非目标

### 2.1 目标

- **G1 阶段化**：把"接到需求→写代码→提交"拆成 7 个有显式入口条件与出口门控的阶段
- **G2 专家叠加**：每阶段自动激活 CLAUDE.md §16 对应专家视角（而不是靠人记）
- **G3 复用既有 skill**：`/e2e` `/review` `/commit` `/security-review` `/simplify` 作为节点调用，不重造
- **G4 制品驱动**：每阶段产出物挂到 `docs/project/` 既有目录，追得回、审得到
- **G5 可审计**：任意时点能问"这个分支当前在哪一阶段？哪个门没过？"（`/dev-pipeline audit` 模式）
- **G6 低摩擦**：hotfix / 单行修复允许**快速通道**（跳到 S4），但必须补 postmortem

### 2.2 非目标

- ❌ **不做仪式化 Scrum**——阶段之间可以几分钟内穿过，不强制会议
- ❌ **不做流水线 UI 工具**——全部 markdown + skill 输出
- ❌ **不做分支/PR 自动创建**——尊重用户对 git 操作的软约束（只 commit 不 push）
- ❌ **不强制 100% 全阶段**——典型的"bugfix、文档修复、重构小段"允许裁剪阶段（见 §6.2）

---

## 3. 多专家评议（12 位视角凑齐再动手）

> 每位专家就"这个 skill 如何才对我负责的领域有用"给一段观点。

### 3.1 架构专家（§16.1）
**看法**：流水线本身就是一个小型"分层系统"——阶段是层，门控是层间契约。**关键诉求**：阶段间只能向前传，不能跳；跳必须登记理由（同 Sprint 的 "加塞" 规则）。**担心**：skill 引入过多中间制品会让重构/小修成本飙升，**强制**为小改动设计"快速通道"。

### 3.2 Go 工程专家（§16.2）
**看法**：S3（开发）阶段必须把 `go build ./...` + `go test ./...` 设成**进入 S4 的硬门**——而不是等到 S5 review。**关键诉求**：skill 应在 S3 收尾时自动跑这两条命令，失败就原地打回，不让代码"先提交再修"。

### 3.3 TR-069 协议栈专家（§16.3）
**看法**：ACS 模块的改动有 TR-069 独有的陷阱（cwmpID 匹配、Empty Response、SOAP Fault 结构）。**关键诉求**：当 S2 设计阶段检测到改动涉及 `internal/acs/`，skill 需强制提示开发者回答"是否影响会话状态机 / RPC 编解码 / 厂商差异路径"三问，没回答不给进 S3。

### 3.4 电信业务专家（§16.4）
**看法**：OMC 70-85% 功能卡住的主因是**运营商差异没在需求期识别**，到编码期才发现适配器不够用。**关键诉求**：S0 立项阶段的 PRD 模板**必须**有"运营商差异矩阵"一节（已存在 TEMPLATE），skill 要把缺此节的 PRD 直接拒掉。

### 3.5 数据与存储专家（§16.5）
**看法**：迁移是最容易出事的环节（近期 R-005/R-108 都是此类）。**关键诉求**：S4 本地验证必须跑 `scripts/check-migrations.sh` + **本地真执行 up/down 各一遍**（不只看编号），skill 要输出一段"回滚演练日志"作为制品。

### 3.6 前端专家（§16.6）
**看法**：前端最容易被"混到后端 PR 里顺便改"搞乱。**关键诉求**：skill 识别到同一变更同时动了 `omcgo/` 和 `omcmb/`，S2 设计阶段**强制拆分**为两个 PRD 条目或明确标注"联合变更"——否则审查责任不清。

### 3.7 测试专家（§16.7）
**看法**：E2E 覆盖率只进不退（DoD 已写死）。**关键诉求**：S4 验证阶段要计算"本次新增端点数 vs 新增 E2E 用例数"，**比值 < 1 不给过**（新端点必有新用例）。并在 S7 产出"E2E 覆盖增量"数字给 Release Gate 用。

### 3.8 安全合规专家（§16.8）
**看法**：auth/admin/rbac 改动走常规 review 不够。**关键诉求**：S5 审查阶段检测到 `internal/admin/` 或 `middleware/auth` 的改动，**自动追加** `/security-review` 为必经节点，而不是"建议"。

### 3.9 运维与可观测性专家（§16.9）
**看法**：新指标/新日志/新告警规则没埋点，线上盲发。**关键诉求**：S2 设计阶段要列出"本变更要加的 metric 名/label/告警规则"，S4 验证阶段 `grep` 确认这些名字在代码里真的出现过——凑不齐不给进 S5。

### 3.10 产品经理 PM（§16.10）
**看法**：S0 是我的主场。**关键诉求**：skill 启动时第一问必须是"对应哪个 PRD？"——找不到就原地停下，把开发者带去写 PRD（或拒绝，如果只是借"顺便"做）。PRD 七要素缺一不进 S1。

### 3.11 项目经理 PgM（§16.11）
**看法**：S1 是我的主场。**关键诉求**：skill 要能读 `docs/project/sprint/sprint-NN.md`，验证本次工作**真的在当前 Sprint 里**；不在就必须走"加塞登记"（记在 risk-register，或明确打进下个 Sprint）。依赖链缺口要自动标红。

### 3.12 QA / 发布经理（§16.12）
**看法**：S5 + S6 + S7 是我的主场。**关键诉求**：S6 提交阶段的 commit message footer **必须**带 `PRD: ...` `Sprint: ...` `Risk: ...`（任一为空需说明 N/A 原因），否则 `/commit` 拒绝。S7 收尾阶段自动更新 DoD 勾选状态和 E2E 增量账。

---

## 4. 七阶段流水线总览

```
       S0                S1               S2              S3
  ┌─────────┐       ┌─────────┐      ┌─────────┐     ┌─────────┐
  │  立项   │──────▶│  规划   │─────▶│  设计   │────▶│  开发   │
  │ Intake  │       │Planning │      │ Design  │     │ Impl.   │
  └─────────┘       └─────────┘      └─────────┘     └─────────┘
    PM 主持          PgM 主持        架构+领域专家     Go/前端专家
    PRD 就绪         Sprint 进栏     设计备忘/接口     增量代码
       │                │                │                │
       │                │                │                ▼
       │                │                │          ┌─────────┐
       │                │                │          │ 本地验证│ S4
       │                │                │          │  Verify │
       │                │                │          └─────────┘
       │                │                │        Testing+运维+数据专家
       │                │                │         build/test/lint/e2e
       │                │                │                │
       │                │                │                ▼
       │                │                │          ┌─────────┐
       │                │                │          │  审查   │ S5
       │                │                │          │ Review  │
       │                │                │          └─────────┘
       │                │                │          /review + 安全专家
       │                │                │           DoD 逐项勾选
       │                │                │                │
       │                │                │                ▼
       │                │                │          ┌─────────┐
       │                │                │          │  提交   │ S6
       │                │                │          │ Commit  │
       │                │                │          └─────────┘
       │                │                │            QA 主持
       │                │                │          /commit + footer
       │                │                │                │
       │                │                │                ▼
       │                │                │          ┌─────────┐
       └────────────────┴────────────────┴─────────▶│  收尾   │ S7
                                                    │ Handoff │
                                                    └─────────┘
                                                     QA+PgM 主持
                                                 Sprint/Risk/DoD 记账
                                                        │
                                                        ▼
                                                 ─────────────────
                                                  交接到部署流水线
                                                 （用户决定 push 时机）
```

每阶段包含 5 要素：**进入条件 · 激活专家 · 活动 · 产出物 · 出口门控**。

---

## 5. 各阶段详细设计

### S0 · 立项（Intake） — PM 主持

| 要素 | 内容 |
|------|------|
| **进入条件** | 用户/issue 提出新需求或新想法 |
| **激活专家** | §16.10 PM（主）+ §16.4 电信业务 + §16.1 架构（范围 sanity） |
| **活动** | 1) 澄清目标用户和场景；2) 识别运营商差异（CMCC/CTCC/CUCC）；3) 标注非目标；4) 列验收 Given/When/Then；5) 填 PRD 七要素 |
| **产出物** | `docs/project/prd/F{NN}-{slug}.md`（基于 `TEMPLATE.md`） |
| **出口门控** | ✅ PRD 七要素齐全（业务背景/用户故事/验收/运营商差异/非目标/依赖/度量）；✅ 运营商差异矩阵不能空；✅ 至少一条可测的验收标准 |
| **阻塞** | 缺任一要素 → 不进 S1；开发者坚持"先写再说" → skill 强制追问并拒绝 |
| **快速通道** | bugfix / 文案改 / 纯重构 允许跳 S0，但 S6 commit message 必须注明"无 PRD，类型：bugfix/refactor/docs" |

---

### S1 · 规划（Planning） — PgM 主持

| 要素 | 内容 |
|------|------|
| **进入条件** | S0 产出的 PRD 已存在 |
| **激活专家** | §16.11 PgM（主）+ §16.1 架构（依赖评估） |
| **活动** | 1) 识别依赖（代码模块 / 外部接口 / 前置 PR）；2) 登进当前或下个 Sprint；3) 指派 Owner（单人）；4) 若风险≥P1 则登 risk-register；5) 粗估时间盒（超 1 Sprint 需拆） |
| **产出物** | `docs/project/sprint/sprint-NN.md` 追加条目；必要时 `docs/project/risk-register.md` 新增行 |
| **出口门控** | ✅ Sprint 有对应条目；✅ Owner 明确；✅ 所有依赖项已在当前/更早 Sprint 排期；✅ 若涉跨模块变更，相关模块 owner 已知会 |
| **阻塞** | 依赖未排期 → 不进 S2；当前 Sprint 已满且非 P0 → 进下一 Sprint 登记后 skill 退出 |
| **快速通道** | hotfix 可跳 S1，直接进 S3，但 **必须**在 S7 补 risk-register 的 hotfix-postmortem |

---

### S2 · 设计（Design） — 架构 + 领域专家主持

| 要素 | 内容 |
|------|------|
| **进入条件** | S1 排期完成 |
| **激活专家** | §16.1 架构（主）+ 根据改动路径自动叠加（见 CLAUDE.md §16 激活规则） |
| **活动** | 按改动范围分别跑：<br>• **acs 改动** → TR-069 专家三问（会话状态机/RPC/厂商差异）<br>• **config/pm/alarm** → 电信+数据专家（Carrier 差异点、时序存储）<br>• **admin/middleware** → 安全专家（鉴权矩阵、审计）<br>• **迁移** → 数据专家（索引、hypertable、回滚策略）<br>• **前端** → 前端专家（类型契约、Mock 切换）<br>• **所有改动** → 运维专家（新 metric/log/告警名字） |
| **产出物** | 轻量设计备忘（挂 PRD 附录 or issue comment），含：<br>- 接口签名 / 路由 / 请求响应<br>- 迁移草案（上/下）<br>- Carrier 扩展点<br>- 新指标/日志名字清单 |
| **出口门控** | ✅ 接口契约明确（路由+请求+响应）；✅ 若动迁移，up/down 草案已写；✅ Carrier 差异点列出；✅ 观测埋点名字列出；✅ 无 3 处以上"待定" |
| **阻塞** | "待定"≥3 → 打回 S0 补澄清 |
| **快速通道** | 纯内部重构可省此阶段，但 S5 review 需勾"无行为变更" |

---

### S3 · 开发（Implementation） — Go / 前端专家主持

| 要素 | 内容 |
|------|------|
| **进入条件** | S2 设计备忘就绪 |
| **激活专家** | §16.2 Go / §16.6 前端（主）+ 领域专家（按路径） |
| **活动** | 1) 先写测试（成功+失败路径）；2) 按 handler → service → repository 分层落地；3) 每 30 分钟或每子任务 commit 一次（本地）；4) 遇阻 3 次停下重评（CLAUDE.md §9 "3 次尝试规则"）；5) 调 `/simplify` 做阶段性自检 |
| **产出物** | 多笔增量代码 + 对应单元测试 |
| **出口门控** | ✅ `go build ./...` 通过；✅ `go test ./...` 全绿；✅ 前端涉及时 `npx tsc --noEmit` 通过；✅ 无 TODO/FIXME/panic("not implemented") 残留；✅ 无 `any` / `if carrier == "cmcc"` |
| **阻塞** | build 或 test 不过 → 不进 S4；遇阻 3 次未解 → 回 S2 重评设计 |

---

### S4 · 本地验证（Local Verify） — Testing / 运维 / 数据专家主持

| 要素 | 内容 |
|------|------|
| **进入条件** | S3 代码就绪 |
| **激活专家** | §16.7 测试（主）+ §16.9 运维 + §16.5 数据 |
| **活动** | 1) `golangci-lint run`；2) 前端 `npm run lint`；3) 迁移 `bash scripts/check-migrations.sh`；4) **本地真跑一遍 migrate up + down**；5) 新端点补 E2E 用例到 `scripts/e2e_verify.sh`；6) 跑 `/e2e` 验证 ≥ 冒烟子集；7) grep 确认新 metric/log 名字已出现在代码中 |
| **产出物** | 本地验证报告（skill 打印到终端 + 可选写入 `docs/review-report/YYYYMMDD/verify-<hash>.md`） |
| **出口门控** | ✅ 所有命令绿；✅ **E2E 用例增量 ≥ 新增端点数**；✅ 迁移双向演练通过；✅ 设计阶段列的新指标/日志名全部在代码可找到 |
| **阻塞** | E2E 覆盖增量比值 < 1（新端点没补用例）→ 不给过；回滚失败 → 不给过 |

---

### S5 · 审查（Review） — 多专家主持

| 要素 | 内容 |
|------|------|
| **进入条件** | S4 验证全绿 |
| **激活专家** | §16 所有按路径匹配的专家 + §16.12 QA |
| **活动** | 1) 调 `/review`（按 scope 激活领域专家）；2) 若路径触及 `admin/` 或 `middleware/auth*` → 自动追加 `/security-review`；3) `/simplify` 最终查一遍；4) 逐项勾选 `docs/project/dod.md` |
| **产出物** | `docs/review-report/YYYYMMDD/REVIEW_<hash>_<author>_<scope>.md` + 勾好的 DoD 清单 |
| **出口门控** | ✅ review 报告无 P0/P1 未解；✅ DoD 清单每项打勾或 N/A（N/A 附理由）；✅ 安全审查（如触发）无 P0 发现 |
| **阻塞** | review 报告存在 P0 → 回 S3 修；DoD 有未勾项 → 回对应阶段补 |

---

### S6 · 提交（Commit） — QA 主持

| 要素 | 内容 |
|------|------|
| **进入条件** | S5 通过 |
| **激活专家** | §16.12 QA（主）+ §16.11 PgM |
| **活动** | 1) 调 `/commit` 生成 Conventional Commits（中文描述）；2) footer 追加三元组：`PRD: <路径>` / `Sprint: <编号>` / `Risk: <编号或 N/A>` / `Review: <报告路径>`；3) pre-commit hook 失败 → 修根因，**严禁** `--no-verify`；4) 失败后不 `--amend`，新开一笔 commit |
| **产出物** | 一笔或多笔本地 commit（未 push） |
| **出口门控** | ✅ commit 成功；✅ footer 三元组齐全；✅ hash 与 message 回显给用户 |
| **阻塞** | hook 失败不绕 → 修到过为止 |
| **约束** | 遵循 CLAUDE.md §8.1 软约束：**只 commit 不 push**，push 必须等用户明确指令 |

---

### S7 · 收尾（Handoff） — QA + PgM 主持

| 要素 | 内容 |
|------|------|
| **进入条件** | S6 成功 |
| **激活专家** | §16.12 QA（主）+ §16.11 PgM |
| **活动** | 1) 告知用户"本地已提交（hash xxx），未推送远端"；2) 更新 `docs/project/sprint/sprint-NN.md` 条目状态为 Done；3) 若 S0 登过 risk-register 条目，更新状态或关闭；4) 累计 E2E 覆盖增量（供 Release Gate 取数）；5) 若走了快速通道（hotfix / 跳阶段），补一份简短 postmortem 登 risk-register |
| **产出物** | Sprint 状态更新 / risk 关闭 / E2E 账目更新 |
| **出口门控** | ✅ 状态更新到位；✅ 所有快速通道都补了 postmortem |
| **阻塞** | 无（此阶段只收尾，不阻断主线） |

---

## 6. Skill 调用模式

### 6.1 三种调用模式

| 模式 | 调用 | 行为 |
|------|------|------|
| **全流程** | `/dev-pipeline` | 识别当前状态（git status + 已有制品）推断阶段，从该阶段接着走 |
| **单阶段** | `/dev-pipeline S2` 或 `/dev-pipeline design` | 只执行一个阶段，返回该阶段门控结果 |
| **审计** | `/dev-pipeline audit` | 不执行任何阶段，只打印"当前在 S_N，通过/未通过的门，下一步该做什么" |
| **快速通道** | `/dev-pipeline --fast bugfix` | 允许跳 S0/S1/S2，但强制 S7 登 postmortem |

### 6.2 阶段裁剪规则

| 改动类型 | 允许裁剪 | 不可省略 |
|---------|---------|---------|
| 新功能 / 新端点 | 无 | S0–S7 全走 |
| bugfix | S0、S1、S2 可省 | S3、S4、S5、S6、S7 必须 |
| 纯重构（行为不变） | S0 可省 | S1 登记 + S2 轻量 + S3–S7 |
| 文档 / 配置 | S0、S1、S2、S4 可省 | S3 不适用；S5 走轻量；S6/S7 必须 |
| hotfix | S0、S1 可省 | S2 简 + S3–S6 + S7 **必补 postmortem** |

### 6.3 skill 实现产物

**本阶段先出设计（本文件）**，后续落地路线：

```
.claude/commands/dev-pipeline.md       # 新建 skill 主文件（对本方案的可执行化）
.claude/commands/dev-pipeline/         # 可选：阶段子指令
    ├── intake.md                      # S0
    ├── planning.md                    # S1
    ├── design.md                      # S2
    ├── implement.md                   # S3
    ├── verify.md                      # S4
    ├── review.md                      # S5（复用 /review）
    ├── commit.md                      # S6（复用 /commit）
    └── handoff.md                     # S7
docs/project/dev-pipeline-design-20260420.md  # 本方案文档（已生成）
```

skill 主文件的骨架（伪代码）：

```markdown
# /dev-pipeline [mode] [stage]

## 参数
- mode: full (默认) | audit | fast
- stage: S0..S7 或 intake|planning|design|implement|verify|review|commit|handoff

## 执行步骤
Step 1: 推断当前阶段
  - 看 git status / git diff / 已有 PRD / Sprint 条目
  - 在审计模式下打印推断结果后退出

Step 2: 按阶段循环执行
  对每个待执行阶段 N:
    a. 打印"进入 S{N}：激活专家 [...]"
    b. 执行活动清单（按路径自动叠加专家）
    c. 产出制品
    d. 跑出口门控检查（硬命令 + 软清单）
    e. 任一门不过 → 停下，报告哪个门、怎么补
    f. 全过 → 推进到下一阶段

Step 3: 终局报告
  - 本轮走过的阶段 / 每阶段产出路径 / 被跳过的阶段及其理由
```

---

## 7. 与既有体系的融合

| 既有制品 | 本方案中的角色 | 是否需要改动 |
|---------|-------------|-------------|
| `CLAUDE.md §16` 12 位专家 | 每阶段按激活规则调用 | 不改 |
| `docs/project/dod.md` | S5 门控的硬清单 | 不改 |
| `docs/project/release-gate.md` | S7 交接到它；E2E 增量数给它 | 不改 |
| `docs/project/risk-register.md` | S1/S7 读写 | 不改 |
| `docs/project/prd/*` | S0 产出；S2/S5/S6 引用 | 不改 |
| `docs/project/sprint/*` | S1/S7 读写 | 不改 |
| `.claude/commands/review.md` | S5 主入口 | 不改 |
| `.claude/commands/commit.md` | S6 主入口 | **小改**：footer 校验三元组 |
| `.claude/commands/e2e.md` | S4 调用 | 不改 |
| `.claude/commands/acs-stress-test.md` | 性能相关变更 S4/S5 调用 | 不改 |
| `scripts/check-migrations.sh` | S4 调用 | 不改 |
| CI / branch protection | S6 提交后进入它；本 skill 与之**平行**，不重复也不替代 | 不改 |

---

## 8. 落地路线图（建议）

| 阶段 | 时间 | 内容 | Owner |
|------|------|------|-------|
| P1 审阅 | D0–D2 | 本方案内部过一遍，收集各专家追加条款 | PgM |
| P2 实现主骨架 | D3–D7 | 写 `.claude/commands/dev-pipeline.md`，跑通 `audit` 模式 | Claude + 用户 |
| P3 阶段子指令 | D8–D14 | 分两批落地 S0/S1/S2，S3/S4/S5/S6/S7 | Claude + 用户 |
| P4 试运行 | D15–D28 | 在 F04-alarm-notification 或下一个新特性上试跑，收集摩擦点 | PgM + PM |
| P5 小改 commit skill | 并行 | footer 三元组校验 | QA |
| P6 回顾 | D28 | 调参：裁剪规则、快速通道边界、专家叠加顺序 | 全员 |

**成功标准（3 个月后回看）**：
- 至少 3 个新功能从 S0 走完到 S7
- E2E 覆盖增量账目无"新端点 0 用例"记录
- risk-register 中"hotfix 未补 postmortem"条目 = 0
- DoD 未勾项导致 PR 打回的次数显著下降

---

## 9. 风险与缓解

| 风险 | 等级 | 缓解 |
|------|------|------|
| 开发者嫌阶段多，绕过 skill | M | 快速通道 + 审计模式（`audit`）做事后对齐；DoD 仍是 PR 硬门 |
| 阶段之间推断错误（skill 判错当前在 Sx） | M | `audit` 模式先输出推断依据，人工可 override |
| 专家叠加导致一次执行过慢 | L | 并行激活 + 短评化（每专家 200 字内） |
| 文档/制品过度膨胀 | M | 明确"轻量设计备忘即可"，不要求独立文档；裁剪规则写死 |
| PRD 模板过严导致 bugfix 被挡 | L | 快速通道覆盖；bugfix 默认跳 S0 |

---

## 10. 附录：一眼看懂的决策树

```
你要动代码吗？
├─ 新功能/新端点 ────────────────▶ 走全流程 S0→S7
├─ 修 bug ──────────────────────▶ 跳 S0/S1/S2，走 S3→S7
├─ 纯重构（行为不变）────────────▶ 跳 S0，S1 登记一行，S2 轻量，S3→S7
├─ 改文档/配置 ─────────────────▶ 只走 S3 适用部分 + S5 轻量 + S6 + S7
└─ hotfix ──────────────────────▶ 跳 S0/S1，S2 简，S3→S7；S7 强制 postmortem

每一阶段最后问自己 4 个字：**门过了吗？**
门没过 → 不准往下走。所有硬门的集合 = 开发流水线的值。
```

---

**结语**：这条流水线不是为了堆流程，而是把"我们已经同意了的规则"（CLAUDE.md §10 + DoD + Release Gate + §16 专家视角）**在开发期**就自动应用，不等到部署期才暴露。部署流水线是最后防线，开发流水线是**第一道防线**——两道线夹起来，质量才守得住。

---

## 11. 补充：L0 Backlog 层（流水线的上游）

> 本章回应"流水线需要一个任务状态表来驱动"的关键诉求。
> 流水线是**消费者**，Backlog 是**生产线**——没有 Backlog，流水线就是空转。

### 11.1 现状缺口

当前 `docs/project/` 有：
- `sprint/sprint-NN.md` — 只描述**当前 2 周**要做什么
- `milestone/2026Q2-to-RC.md` — 只描述**季度/里程碑**粒度
- `risk-register.md` — 登记风险，不登记"想做的事"
- `prd/*.md` — 写出来的特性，没写的不在这里

缺了中间一层：**所有想法（特性/bug/技术债/文档/hotfix）汇合、分诊、排序、等待出队**的地方。
没这一层会出现：
- 想法丢在 chat 里，再问起记不得
- 两个 Sprint 之间出现"真空期"，不知道拉什么
- 依赖关系藏在脑子里，换人就断
- 优先级靠口头，不同人不同排法

### 11.2 Backlog 的位置

```
       提出                           出队
┌──────────────┐      ┌─────────┐      ┌────────────────┐
│  Source：    │─────▶│   L0    │─────▶│ dev-pipeline   │
│ user / bug   │ 登记  │ Backlog │ 取   │  S0→S1→...→S7  │
│ review / QA  │      │         │      └────────────────┘
└──────────────┘      └─────────┘
                          ↕
                 每周 triage + 每 Sprint plan
                 （PM / PgM / QA 三角色轮值）
```

**关键职责**：
- **入口**：任何来源的"可能要做的事"都先进 Backlog，**不允许**跳过 Backlog 直接进 S0
- **分诊**：打类型、定优先级、估工作量、识别依赖
- **排序**：按优先级 + 依赖关系 + Sprint 容量出队
- **状态追踪**：每个任务有明确状态，流水线推进时回写
- **出口**：按优先级顺序把任务喂给 `/dev-pipeline`

### 11.3 一个 Task 的 Schema

每个任务有一行 schema（`docs/project/backlog.md` 主表字段）：

| 字段 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `ID` | ✅ | `T-NNNN` | 全局唯一，递增（如 T-0001） |
| `Title` | ✅ | 一句话 | 动词开头（"实现 F04 告警邮件通知"） |
| `Type` | ✅ | enum | feature / bugfix / tech-debt / refactor / docs / hotfix |
| `Domain` | ✅ | enum | F01-F10 / infra / process |
| `State` | ✅ | enum | 见 §11.4 状态机 |
| `Priority` | ✅ | enum | P0 / P1 / P2 / P3（见 §11.5） |
| `Owner` | ⚪ | string | 分诊后填；P0/P1 必填 |
| `Est` | ⚪ | 天 | 粗估工作量（<0.5 / 0.5-1 / 1-3 / 3-5 / >5 需拆） |
| `Deps` | ⚪ | `T-xxxx, T-yyyy` | 阻塞依赖（这些 Done 了才可进 S1） |
| `Risks` | ⚪ | `R-xxx` | 关联 risk-register 条目 |
| `PRD` | ⚪ | 路径 | S0 产出后回填 |
| `Sprint` | ⚪ | `sprint-NN` | S1 排期后回填 |
| `Created` | ✅ | YYYY-MM-DD | 进池日期 |
| `Updated` | ✅ | YYYY-MM-DD | 最近状态变更日期 |
| `Closed` | ⚪ | YYYY-MM-DD | S7 完成日期 |

**补充原则**：
- 小任务（bugfix / docs，Est < 0.5 天）只需主表一行，不开详情文件
- 中大任务（feature / 跨模块 / P0/P1）在 `docs/project/backlog/T-NNNN-<slug>.md` 开详情文件（模板见 §11.8）
- Task 与 PRD 关系：**1 个 PRD 可拆成多个 Task**（如 F04 告警通知可拆"模型+API"、"NATS 订阅"、"前端 UI"三个 Task，共享一个 PRD）

### 11.4 状态机

```
                      ┌────────────┐
                      │  proposed  │ 刚进池，等分诊
                      └─────┬──────┘
                            │ triage 后
                ┌───────────┼────────────┐
                ▼           ▼            ▼
         ┌──────────┐  ┌──────────┐  ┌──────────┐
         │ triaged  │  │ deferred │  │ rejected │
         └────┬─────┘  └──────────┘  └──────────┘
              │ 进 Sprint
              ▼
         ┌──────────┐
         │ planned  │ S1 完
         └────┬─────┘
              │
              ▼
         ┌──────────┐
         │in_design │ S2
         └────┬─────┘
              ▼
         ┌──────────┐        ┌─────────┐
         │ in_dev   │◀──────▶│ blocked │（任何 in_* 都能转 blocked）
         └────┬─────┘        └─────────┘
              ▼
         ┌──────────┐
         │in_review │ S5
         └────┬─────┘
              ▼
         ┌──────────┐
         │   done   │ S7 关闭
         └──────────┘
```

**状态转移规则**：
- `proposed → triaged`：每周一次 triage 例行（见 §11.6）；PM/PgM/QA 共同判决
- `triaged → planned`：Sprint Planning 时（`/dev-pipeline plan` 子命令）
- `planned → in_design` 到 `done`：由 `/dev-pipeline` 推进时自动回写
- `* → blocked`：依赖未完成/外部阻塞；必须在 risk-register 有对应条目
- `blocked → in_*`：阻塞解除后回到原状态
- `triaged → deferred/rejected`：显式判决，不影响池子整洁

### 11.5 优先级定义（P0–P3）

| 级别 | 含义 | 判断 | 示例 |
|------|------|------|------|
| **P0** | **阻塞 RC/GA** | 不做无法发布 | F04 告警通知缺失（已列 milestone）、R-005 迁移编号不连续 |
| **P1** | **本里程碑必须** | 不做就得挪到下一里程碑 | F08 北向 OSS 协议接入、Notification 基础设施 |
| **P2** | **应该做** | 可挪 1 个里程碑内消化 | 日志字段扩展、少量 UI 优化 |
| **P3** | **想做** | 无硬时间约束，填 buffer 用 | 工具脚本、文档润色 |

**出队规则**（`/dev-pipeline next` 的排序逻辑）：
```
1) 过滤 State ∈ {triaged, planned}
2) 过滤 Deps 全部 done
3) 按 Priority 升序（P0 优先）
4) 同级按 Created 升序（先进先出）
5) 返回前 N 条（默认 5）
```

### 11.6 Triage 节奏

| 节奏 | 触发 | 参与角色 | 动作 |
|------|------|---------|------|
| **每日** | 开始工作前 | 执行者 | `/dev-pipeline status` 看概览；`/dev-pipeline next` 取下一任务 |
| **每周 Triage** | 每周一上午 | PM + PgM + QA | 将 `proposed` 批量判决为 `triaged/deferred/rejected`；补齐字段 |
| **每 Sprint Plan** | Sprint 首日 | PgM 主持 | 从 `triaged` 挑选进入本 Sprint，转 `planned`；Sprint 容量 = `(2 周 × Owner 可用天数) × 0.8 buffer` |
| **随手登记** | 任意时刻 | 任何人 | 发现新想法直接在 backlog.md 加一行 `proposed`（等 triage） |

### 11.7 依赖图 & 阻塞处理

```
T-0001 (Notification 基础设施)  ←───┐
T-0002 (F04 告警订阅 API)       ←───┤── 被依赖
T-0008 (F04 告警邮件渠道) ──────────┘─→ 依赖 T-0001 + T-0002 都 done
```

**自动化处理**：
- 进 S1 时 skill 检查 `Deps` 全为 `done`，否则 → `blocked` + 在 risk-register 登一行
- 循环依赖检测（skill `/dev-pipeline validate`）
- 关键路径高亮（长依赖链 P0 任务标红）

### 11.7.1 累计型依赖豁免（贯穿型任务特殊规则）

**问题**：某些 Task 天然跨多个 Sprint 累计推进（典型：T-0006 E2E 用例补齐，Sprint-01→07 累计 ≥200 条）。若下游 Task 等它 `done` 才放行，会导致下游永远进不了 Sprint-06/07——死锁。

**识别方式**（两条中任一即视为"累计型"）：
1. `Sprint` 字段含 `..` 跨度（如 `sprint-01..07` / `sprint-02..04`）且 `Est = XL`；
2. Deps 引用形如 `T-NNNN@累计≥X`（显式里程碑式依赖）。

**豁免规则**：
- 下游 Task 在 S1 Planning 校验依赖时：
  - 若上游为累计型 + 当前 `State ∈ {in_dev, in_review}` + 已承诺覆盖的累计里程碑大于等于下游声明阈值 → **放行**进 `planned`，并在 Task.Notes 追加 `⚠️ cumulative-dep:T-NNNN@累计≥X`。
  - 否则与普通 Deps 同样处理（转 `blocked`）。
- 下游在 S4 verify 时新增一项硬门：`Cumulative Dep Check` — 实际累计数 ≥ 声明阈值。未达则 S4 不过。
- 上游累计 Task 每次 Sprint 回顾更新"当前累计值"到 Task.Notes（字段 `Progress: 累计 NN`），便于下游校验。

**典型案例**：
| 下游 | 上游 | 语义 | 校验时机 |
|------|------|------|---------|
| T-0025（RC 冻结 + 冒烟） | T-0006@累计≥150 | RC 前至少 150 条 E2E 才能宣布冻结 | S4 verify |
| T-0024（Gate 演练） | T-0023（压测基线）[非累计] | 常规 done 依赖 | S1 planning |

**自动化要点**：
- `/dev-pipeline next` 识别累计型上游 → 放入候选并标 `⚠️ 累计型`（不入等待清单）。
- `/dev-pipeline validate` 对累计型依赖不报"状态悖论"（Task=planned 但 Deps 未 done 本不允许，但累计型例外）。

### 11.8 存储格式（具体落地）

**A. 主索引 `docs/project/backlog.md`**（单一 source of truth）：

```markdown
# OMC 需求池（Backlog）

> 更新时间：2026-04-20
> 统计：proposed 12 / triaged 8 / planned 5 / in_* 3 / done (本季) 47

## Active（triaged + planned + in_*）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Updated |
|----|-------|------|--------|------|-------|-------|-----|------|--------|---------|
| T-0001 | 实现 Notification 基础设施 | feature | infra | P0 | in_dev | @alice | 3 | - | sprint-07 | 2026-04-18 |
| T-0002 | F04 告警订阅 API | feature | F04 | P0 | planned | @bob | 2 | T-0001 | sprint-08 | 2026-04-15 |
| T-0008 | F04 告警邮件渠道 | feature | F04 | P1 | triaged | - | 2 | T-0001,T-0002 | - | 2026-04-20 |
| ... |

## Proposed（待 triage）

| ID | Title | Type | Proposed By | Created |
|----|-------|------|-------------|---------|
| T-0042 | 告警导出 CSV | feature | @user | 2026-04-19 |
| ... |

## Done（最近 30 天）

| ID | Title | Domain | Closed |
|----|-------|--------|--------|
| T-0007 | 修补 5 个 reserved_placeholder 迁移 | infra | 2026-04-19 |
| ... |

## Deferred / Rejected

| ID | Title | State | Reason |
|----|-------|-------|--------|
| ... |
```

**B. 详情文件 `docs/project/backlog/T-NNNN-<slug>.md`**（仅 P0/P1/复杂任务）：

```markdown
---
id: T-0008
title: F04 告警邮件渠道
type: feature
domain: F04
priority: P1
state: triaged
owner:
est_days: 2
deps: [T-0001, T-0002]
risks: [R-012]
prd: docs/project/prd/F04-alarm-notification.md
sprint:
created: 2026-04-20
updated: 2026-04-20
---

## 动机（为什么要做）
...

## 验收（Acceptance，粗版；细化归 S0 的 PRD）
- [ ] 告警触发后 1 分钟内收到邮件
- [ ] 邮件模板支持 CMCC/CTCC/CUCC 品牌差异

## 依赖
- T-0001：Notification 基础设施（提供 channel 抽象）
- T-0002：告警订阅 API（提供订阅者数据）

## 状态变迁日志
- 2026-04-20 proposed → triaged（PM+PgM+QA Triage 会议）
```

### 11.9 与 S0–S7 的对接（每阶段读写什么）

| 阶段 | 读 Backlog | 写 Backlog |
|------|-----------|-----------|
| （进 S0 前） | `/dev-pipeline next` 出队 1 条，检查 State=triaged 或 planned、Deps 全 done | — |
| **S0 Intake** | 读 Task 的 Type/Domain/验收粗版，用于起草 PRD | PRD 产出后，回填 `PRD` 字段；State 不变（triaged 继续） |
| **S1 Planning** | 读 Deps / Est | 回填 `Sprint` / `Owner`；State: `triaged → planned` |
| **S2 Design** | 读 PRD 路径 | State: `planned → in_design` |
| **S3 Impl** | — | State: `in_design → in_dev` |
| **S4 Verify** | — | (保持 in_dev) |
| **S5 Review** | — | State: `in_dev → in_review` |
| **S6 Commit** | 读 `PRD / Sprint / Risks` 字段填入 commit footer | — |
| **S7 Handoff** | — | State: `in_review → done`；回填 `Closed` 日期 |
| （任何阶段）遇阻 | — | State: `* → blocked`；同步 risk-register |

### 11.10 Skill 新增子命令（补充 §6.1）

| 子命令 | 作用 |
|--------|------|
| `/dev-pipeline backlog add <title>` | 快速登记新想法（默认 `proposed`） |
| `/dev-pipeline triage` | 遍历 `proposed`，逐条引导 PM/PgM/QA 判决（batch 模式） |
| `/dev-pipeline plan sprint-NN` | 从 `triaged` 池按优先级 + 容量挑选进 Sprint |
| `/dev-pipeline next` | 出队：返回**当前可做**的 Top N 任务（默认 5） |
| `/dev-pipeline pick T-NNNN` | 手动选定，进入全流程 S0→S7 |
| `/dev-pipeline status` | 打印 Backlog 统计与健康度（proposed 积压、blocked 未解、超期 Sprint 条目等） |
| `/dev-pipeline validate` | 检查循环依赖、孤儿任务、Deps 指向未登记 ID、被 Sprint 引用但状态 `proposed` 等 |

**关键：`pick` / `next` 才会触发 S0–S7 主流程**，其他子命令只动 Backlog，不走流水线。

### 11.11 健康度指标（Backlog 本身的 DoD）

Backlog 不是登记就完事的，它也有自己的健康阈值（由 `/dev-pipeline status` 输出）：

| 指标 | 目标 | 超阈行为 |
|------|------|---------|
| `proposed` 积压天数 > 7 | 不允许 | skill 提醒"本周未 triage"，PgM 必须安排 |
| `blocked` > 3 条 | 不允许 | 开"阻塞专题"评审 |
| P0 数 > Sprint 容量 | 不允许 | 强制重新优先级 / 推迟里程碑 |
| 单 Task `in_dev` > 5 天 | 警告 | 启动"3 次失败"重评（CLAUDE.md §9） |
| Sprint 完成率 < 80% | 警告 | 回顾会上必查根因 |

### 11.12 Bootstrap 策略（从 0 建这张表）

**不要求一次建全**。按以下节奏 bootstrap：

| 阶段 | 动作 | 产出 |
|------|------|------|
| **Day 0** | 新建空 `docs/project/backlog.md`（带表头 + 3 个分区） | 主表骨架 |
| **Day 1-2** | PM/PgM/QA 三人各用 30 分钟，把**当前知道的** P0/P1 事项逐条登进主表（不强求完整字段） | 20-40 条初始 Task |
| **Day 3** | 从现有 `sprint/` `milestone/` `risk-register.md` 反向索引出已承诺但未登记的工作 → 补齐 Backlog | Backlog 与既有制品对齐 |
| **Day 4-7** | 试运行 `/dev-pipeline next` + `triage` + `plan`，同时保留旧路径（允许不走 skill 直接做） | 工具磨合 |
| **Week 2 起** | 强制：所有新工作必须先在 Backlog 登记才能进 S0 | Backlog 成为入口 |
| **Month 1 末** | 回顾 Backlog 使用情况，调 Schema / 优先级定义 / Triage 频率 | 稳定版本 |

### 11.13 与既有制品的关系梳理（避免重复）

| 制品 | 与 Backlog 的关系 |
|------|-----------------|
| `sprint/sprint-NN.md` | Sprint 是 Backlog 的**视图**——从 Backlog 筛 `sprint=sprint-NN` 即可生成；Sprint 不再手工维护"做什么"，只记录冲刺目标与回顾 |
| `milestone/*.md` | Milestone 是 Backlog 的**聚合**——按里程碑筛 P0/P1 任务形成；Milestone 写高层叙事，不重复任务清单 |
| `prd/*.md` | PRD 被一个或多个 Task 引用（多对一）；PRD 只写"做什么"，不写"什么时候做"（时间归 Sprint） |
| `risk-register.md` | Risk 与 Task 互相交叉引用：Task 可能会 mitigate 某 Risk，Risk 也可能 spawn 新 Task |
| `dod.md` | Backlog 不改 DoD；但 Backlog 的 `done` 状态回写前，DoD 清单必须全绿（S5 门控） |

**原则**：Backlog 是**唯一任务清单**，其他文件**引用而不重复**；去掉 Sprint/Milestone 里重复的"待办"字段。

### 11.14 举一个完整的例子

假设用户说"我想加一个告警导出 CSV 的功能"：

```
Day 1 (随手登记):
  /dev-pipeline backlog add "F04 告警导出 CSV"
  → 生成 T-0042, state=proposed, 其他字段空
  → backlog.md 主表 proposed 分区多一行

Day 3 (周一 triage):
  /dev-pipeline triage
  → 逐条过 proposed 项；T-0042 被判为：
      type=feature, domain=F04, priority=P2, est=1 天,
      deps=[], state=triaged
  → 不开详情文件（P2 且简单）

Day 15 (Sprint-09 首日 Planning):
  /dev-pipeline plan sprint-09
  → Sprint-09 还有 3 天容量，从 triaged 池按 P0→P1→P2 排
  → T-0042 被选中，owner=@alice, sprint=sprint-09, state=planned

Day 17 (Alice 上班):
  /dev-pipeline next
  → 返回本人当前最高优先级可做的：T-0042
  /dev-pipeline pick T-0042
  → 进入 S0：因 type=feature，要求 PRD
  → 起草 docs/project/prd/F06-alarm-export-csv.md（属 F06 ops 子域）
  → S0 完，PRD 回填，state 仍为 planned

Day 17-18: S1→S7 依次推进，state 自动回写
Day 18: S7 关闭 → state=done, closed=2026-05-07
```

全程**人只做判断，不维护多份冗余表**。

---

> 至此 §11 补齐了流水线上游。**核心结论**：Backlog 是流水线唯一的"任务源"，没有它就不启动流水线；有了它，流水线就变成机械的出队-加工-回写过程，人的注意力回到真正值钱的地方——分诊、优先级、依赖判断。
