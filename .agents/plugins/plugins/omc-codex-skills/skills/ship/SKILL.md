---
name: ship
description: OMC 一键交付流程，从想法或 GitHub Issue 推进到 feature 分支 PR。用户显式调用 $ship、要求交付 OMC Issue、执行 issue-to-PR 流水线、查看 status/audit、处理 ready-for-agent Issue，或要求从实现推进到构建、验证、审查、提交、推送、开 PR 时使用；严禁直接合并或推送 main。
---

# Ship

驱动 OMC 从需求或 GitHub Issue 到可审查 PR 的开发流程。本 skill 只做编排：把设计、实现、审查、交接委派给对应 skill 或工具，并在阶段之间执行硬门检查。

永远不要直接在 `main` 或 `master` 上提交、合并或推送。进入实现阶段后，必须在 feature 分支工作，并以 PR 作为交付终点。不要 force-push 共享分支。

## 调用参数

把 `$ship` 后面的参数解释为：

| 参数 | 含义 |
|---|---|
| 空 | 推断当前阶段，并自动推进到 PR。 |
| `status` | 只报告 GitHub Issue 看板分桶和当前分支阶段。 |
| `audit` | 只推断当前阶段、已过门、未过门和下一步动作。 |
| `#NN` | 交付指定 Issue；如果还不是 `ready-for-agent`，先分诊。 |
| `N` 或 `all` | 按优先级选择前 N 个或全部 open `ready-for-agent` Issue，每个 Issue 独立分支、独立 PR。 |
| `P1`..`P9` | 只跑指定阶段。别名：`align`、`spec`、`slice`、`triage`、`build`、`verify`、`review`、`commit`、`pr`、`close`。 |
| `--fast bugfix\|hotfix\|docs\|refactor` | 使用对应快速通道，但仍执行所有适用硬门，并且必须开 PR。 |

进入每个阶段前先输出：

```text
--- P<N> <阶段名> --- 委派: <skill/tool> 硬门: <gate list>
```

如果硬门失败，停止推进，报告失败硬门，并告诉用户修复后如何恢复，通常是再次运行 `$ship P<N>`。

## 阶段流水线

### P1 对齐

当需求模糊、跨模块或存在架构取舍时，使用 `$grill-with-docs`。

硬门：决策已经明确；当决策改变领域语言或架构时，同步更新 `CONTEXT.md` 或 `docs/adr/`。

### P2 立项

当需求已成形，可以发布为 GitHub Issue 时，使用 `$to-prd`。

硬门：PRD Issue 已存在，并带有 `ready-for-agent` 标签。

### P3 切片

当 PRD 包含多个可独立交付的垂直切片时，使用 `$to-issues`。

硬门：子 Issue 是可独立实现的 tracer-bullet 切片，并记录依赖顺序。

### P4 分诊

只对 P3 拆出的、尚未就绪的子 Issue 使用 `$triage`。

硬门：每个子 Issue 恰好有一个 category 标签、一个 state 标签；`ready-for-agent` 状态下包含 agent brief。

### P5 实现

实现使用 `$tdd`。如果硬 bug 或性能回归卡住，使用 `$diagnose`。

动代码前确认当前分支不是 `main` 或 `master`。如果当前在主分支，创建 `<type>/NN-<slug>` feature 分支，`type` 来自 Issue category：`feat`、`fix`、`refactor`、`docs` 或 `chore`。

硬门：

- `cd omcgo && go build ./...`
- `cd omcgo && go test ./...`
- 若有前端改动，按适用范围同步三皮肤：`webcode`、`webcode-v2`、`webcode-v3`；优先改 `omcmb/frontend-core` 共享业务层；然后执行 `cd omcmb && npm run typecheck`。
- 当前分支不是 `main` 或 `master`。

### P6 验证

选择最窄但足够的验证方式：E2E、ACS 压测、浏览器冒烟、启动后观察真实行为等。

硬门：

- 新端点要求 E/R >= 1：新增 E2E claim 数 / 新增 route 数至少为 1。
- 新增 metric/log 名称必须能被 grep 到。
- 前端改动需要对受影响皮肤连接真实后端做浏览器冒烟；出现 `未找到`、`加载失败`、`Error` 或 `pageerror` 判失败。
- 新迁移必须有配对 up/down SQL，并本地演练 up/down。

不适用的硬门标记为 `N/A`，不要静默跳过。

### P7 审查

使用 `$review`。如果改动触及 `internal/admin/` 或 `middleware/auth*`，追加安全审查。只有在能降低真实风险或复杂度时，才做简化/重构 pass。

硬门：

- 审查没有 CRITICAL 发现。
- 适用时逐项检查 `docs/project/dod.md`。
- 安全敏感路径已做安全审查。

CRITICAL 示例：运营商硬编码如 `if carrier == "cmcc"`、字符串拼接 SQL、裸 `panic`、公共接口扩宽为 `any` 或 `interface{}`、删除测试、降低安全性。

### P8 提交

在 feature 分支创建正常 commit。不要使用 `--no-verify`。如果 pre-commit 失败，修根因后正常提交。

硬门：commit 成功；交付 Issue 时提交信息包含 `Closes #NN`。如果快速通道跳过了阶段，在 commit body 记录 `Skip: P1,P2,...`。

### P9 PR

推送 feature 分支并使用 `gh pr create` 创建 PR。交付 Issue 时，PR body 必须包含 `Closes #NN`。

硬门：

- 当前分支不是 `main` 或 `master`。
- `git push -u origin <feature-branch>` 成功。
- `gh pr create` 返回 PR URL。
- 不合并 PR。review 和 merge 不属于本 skill。

如果工作无法完成，使用 `$handoff` 保存现场和下一步。

## 阶段推断

空参数 `$ship` 时，按下面顺序推断起始阶段：

1. 有暂存变更，且存在当日审查报告 -> P8。
2. 有未提交变更 -> 如果新端点或迁移需要验证则 P6，否则 P7。
3. 当前分支或近期 commit 引用 `#NN`，或存在 `ready-for-agent` Issue 且没有代码变更 -> P5。
4. 存在 `ready-for-human` Issue 且没有代码变更 -> 停止并报告需要人工实现。
5. 已有成形需求但没有 GitHub Issue -> P2；如果仍模糊则 P1。
6. 只有模糊想法 -> P1。

`audit` 模式只报告推断阶段和硬门状态，不推进。

## 快速通道

快速通道只减少前置仪式，不跳过适用的机械门或质量门，也永远不跳过 P9。

| 类型 | 可跳过 | 必跑 |
|---|---|---|
| enhancement | 无 | P1-P9 |
| bugfix | P1, P2, P3 | P4 轻量确认，P5-P9 |
| refactor | P1, P2 | P3 登记，P5-P9 |
| docs/config | P1, P2, P3, P6 | 适用的 P5，轻量 P7，P8，P9 |
| hotfix | P1, P2 | P5-P9，并补 postmortem 到 `docs/project/risk-register.md` |

对 `#NN`，Issue category 标签 `bug` 和 `enhancement` 分别映射到 bugfix 和 enhancement。`refactor`、`docs`、`hotfix` 必须显式传 `--fast`。

## 批量模式

对 `$ship N` 或 `$ship all`：

1. 查询 open `ready-for-agent` Issue：`gh issue list --label ready-for-agent --state open --json number,title,labels`。
2. 按优先级排序：`critical`、`high`、`medium`、`low`，同优先级按 Issue 编号升序。
3. 串行处理每个 Issue。从 `main` 起步，创建专属 feature 分支，执行 P5-P9，开一个独立 PR，再处理下一个。
4. 不自动 `git pull`，除非用户要求。
5. 如果某个 Issue 合理修复尝试后仍阻塞，在 Issue 评论阻塞点，保留半成品分支，在汇总中标记 blocked，然后继续下一个。

结束时给出逐 Issue 汇总：PR URL、blocked 原因或 skipped 原因。

## OMC 项目规则

- 从仓库根目录 `goomc/` 执行 git 命令。
- `AGENTS.md`、`.codex/` 以及无关本机 AI 配置默认不进 PR，除非用户明确要求。本 skill 自身位于 `.agents/skills/ship`，是用户明确要求的团队共享源码。
- 前端浏览器验证默认优先使用可见 Playwright + 系统 Chrome；除非用户要求关闭，否则验证结束保留最终浏览器现场。
- 按改动路径激活 `docs/expert-personas.md` 的专家视角，例如 `internal/acs/` 使用 TR-069 视角，`migrations/` 使用数据视角，`omcmb/` 使用前端视角。
- 不在这里复制子 skill 的完整清单。触发被委派 skill 时，读取并遵循它自己的 `SKILL.md`。
