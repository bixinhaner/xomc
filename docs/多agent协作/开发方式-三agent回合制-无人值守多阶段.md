# 开发方式 · 三 agent 回合制 + 无人值守多阶段推进

> 性质：开发方法论母本（现行可用）。**不入 git**，放 notes。
> 一句话：有一份**多阶段实施计划**时，让一个总调度**无人值守逐阶段串行**推进——每阶段内部由**开发 / 检查 / 运行栈 / 存档点**四个 agent 回合接力，靠一份**账本**跨回合传话，**检查 gate 住部署**、**脚本死判**运行栈、**存档点**逐阶段提交、**失败即停留现场**；全部跑完回主会话**一次性收口验收 + 合成提交**。
> 配套脚本（可运行）：`~/Documents/notes/workflows/dev-check-loop.workflow.js`
> 母本前身：`知识库/开发方式-双agent回合制.md`（单任务两 agent，本法是它的多阶段无人值守演进）
> **触发**：我说「**按三 agent 回合制无人值守推进 T-XXXX**」/「**三 agent 多阶段跑 …**」，或把本文递给你 → 按本法执行。

---

## 1. 何时用 / 何时不用

**适合**：一份**已锁设计 + 已拆好的多阶段实施计划**（3-5 个可串行推进的阶段，每阶段范围清晰、能用命令自证 + 运行栈验证收口）。典型：一个功能的「后端骨架 → worker → 前端业务层 → 端到端」四阶段。

**不适合**：
- 还没锁设计（先 brainstorming）。
- 没拆成多阶段的单任务（用**双 agent 回合制**母本，不必上多阶段调度）。
- 单行 hotfix / 纯探查（直接做 / 用 Explore）。
- **首次或换了大改的脚本后**：先拿一个真实多阶段任务**小规模试跑**，跑顺再常态化。

**与双 agent 回合制的关系**：双 agent 是「把一个任务做扎实」；本法是「把一条多阶段实施计划无人值守跑完」。本法每阶段内部就是双 agent 那套对抗式回合，外面再套**多阶段串行 + 运行栈 agent + 存档点 + 死判**。

---

## 2. 整体形状：多阶段串行 + 阶段内三 agent 四环节

总调度读 `stages` 数组**按顺序串行**走，每阶段内部回合制：

```
阶段 N：
  回合 1..maxRounds（默认 4，②fail 与 ③fail 共用一套计数）：
    ① 开发 agent    读账本+spec → 实现/修 → 自跑 build/test → 写账本
    ② 检查 agent    对抗复核 build/test + 对照实施计划核符合性（强化职责）
          ├ fail → 下一回合（不部署，省一次重建）
          └ pass ↓
    ③ 运行栈 agent  部署 → 取证 → 回结构化实测值（只回证据不下结论）
          脚本死判：全部 hardChecks 实测==预期 且 全部 softChecks agent 判 pass
          ├ fail → 下一回合（开发读"运行栈哪没过"接着修）
          └ pass ↓
    ④ 存档点 agent  打一个极简 wip commit → 进下一阶段
  到 maxRounds 仍 fail → 整条停，留现场（账本+工作树+浏览器）→ 通知用户
```

关键设计：
- **检查 gate 住运行栈**：build/test 没过不白部署。
- **第④是 agent 个数不是回合数**：`/workflows` 显示「阶段 N · 4/4」是该阶段四个 agent（开发+检查+运行栈+存档点），**不是第 4 回合**——别误读，接手提示词里也写这句。
- **运行栈四段按需**：部署 → 建数据 → 查库指纹 → playwright 看 DOM，纯后端阶段自动跳过 DOM 段。

---

## 3. 混合验收：死判 + 自判（本法核心，破解"无人值守又不破铁律"）

用户铁律是「运行栈验证的判断留主线」，而主线做判断正是吃上下文的环节。破法：把判据尽量写成**可数预期**让脚本代码比对，只有机器没法数的才真下放给 agent 判。每阶段验收喂调度时拆两类：

- **死判项 `hardChecks`**：每条带业务描述 + 一个**预期值**（数字/字符串/布尔）。运行栈 agent 只回**实测值 + 那几行原始证据**，由**脚本代码**做 `实测==预期`。agent 不发结论 → 没有"信总结"空间 → 这段等于没破铁律，只是把"主线肉眼比对"换成"代码比对"。
- **自判项 `softChecks`**：图有没有画出来、滚动对不对这类机器没法数的，agent 自己判 pass/fail，但返回 schema **强制配一段真实原始证据**（库里真实行 / DOM 真实文字 / 接口真实 JSON），编不出过不了 schema。

**脚本判一阶段运行栈通过** = 全部死判项 `实测==预期` **且** 全部自判项 agent 判 pass。任一不满足 → 运行栈 fail → 回开发下一回合。

> **写验收的纪律（决定质量的上游）**：**能转死判的尽量写成死判**。首跑教训——「webcode-v2/v3 仍能编译」当时写成了 softCheck，agent 报喜判 pass，实为 typecheck FAIL；这种"退出码=0"本就该是 hardCheck（expected=0）。`softChecks` 只留真正机器数不出来的视觉/交互项。

---

## 4. 四个 agent 的职责红线

**① 开发 agent**：在阶段工作树直接改、只改代码与测试**不 commit**；实现后**必须自跑** build/test 记真实结果；关键逻辑补单测；收尾写账本。（同双 agent 母本）

**② 检查 agent（对抗式，默认怀疑）**：自己重跑 build/test 不信开发口头；逐条核对验收 + 反推额外风险 + 找阻塞 + 不确定判 fail。**本法强化一等职责**：**对照实施计划核符合性**——逐条读本阶段 `scope`（做什么/不做什么）+ spec 指定结构，核对实现有没有**偏离**（说照搬某模式却另起炉灶）/**越界**（动了 scope 说"不做"的）/**漏做**（scope 说"做"的没做），逐条记 `planConformance`，阻塞项升级进 `blockingIssues`。
> 首跑印证有效：T4 第 1 回合检查 agent 揪出"前端下钻白名单被后端结构体静默丢弃"真实缺陷判 fail，逼出第 2 回合修正——对抗式检查的价值真兑现。

**③ 运行栈 agent（空白上下文，护栏必须写死进 prompt）**：部署 → 取证 → 回 `RUNVERIFY_SCHEMA`（hardResults 实测值+证据 / softResults 判定+证据 / 部署护栏自证）。**只回原始证据，死判项绝不自己下结论**。部署护栏见 §6。

**④ 存档点 agent**：阶段过后打一个 wip commit（`git add -A` + `wip(<stage>): 阶段存档点`），唯一作用是让下阶段 `git diff` 干净，**不是最终历史**。
> 存档点保持 `git add -A` 不改路径白名单——前提是用户人工保证同一时间只有一个会话在改这棵树（见 §7，2026-06-05 拍板）。万一约定没守住，检查 agent + 收口 grep 验兜底。

---

## 5. 三处 git 分工（脚本是 JS 沙箱，跑不了 git）

1. **记 HEAD（主会话）**：调 workflow **前** `git rev-parse HEAD`，传进 `args.startHead`（脚本透传写账本回合0），作最后回退锚点。
2. **存档点 commit（④ agent）**：每阶段过了打 `wip(<stage>): 阶段存档点`。
3. **合成提交（主会话）**：全部阶段跑完，流水线 return 给主会话 → 主会话读完整账本 → `git reset --soft <startHead>`（改动全留不丢码）→ 合成**一个**功能提交配好 message → **只 commit 不 push**（push 留用户人工验证通过后分步 `pull --rebase` → push）。

---

## 6. 部署护栏（必须写死进运行栈 agent 的 prompt）

第③ agent 不带"部署目录铁律"记忆，prompt 显式写明：
- 部署**只跑** `bash /Users/shangyingbin/project/omc-docker/docker-run.sh`（宿主编译 + docker 重建全栈）。
- **绝不碰** `goomc/deployments/docker`（碰错会删运行栈网络致全栈断网）。
- 部署后验证：`cd /Users/shangyingbin/project/omc-docker && docker compose ps`（基础设施 healthy / 迁移 exited(0) / 应用 running）+ `curl -fsS http://localhost:8081/healthz`、`curl -fsS http://localhost:7557/healthz`。
- 端到端走前端 :3000 + playwright（admin/admin123）。**环境坑**：明文 curl 登录被禁（前端走加密）；内部 X-API-Key 对受限接口（如 `/pm/adhoc/*`）无权 → 端到端验受限接口走 playwright 或直接读 DB。

---

## 7. 独占工作树 + 独占运行环境（由用户人工保证，2026-06-05 拍板）

本法默认**调度独占当前 git 工作树 + 独占那套唯一的 dev 运行环境**（一套 docker 栈 + 一个库 + 固定端口）。**为什么不靠技术隔离**：worktree 只能隔代码、隔不开单例运行环境（两个无人值守任务一部署照样抢同一套服务/库/端口），所以"同时只跑一个"这条约束躲不掉；既然躲不掉，再上 worktree / 存档点路径白名单就是多花成本买半个好处。

**用户已拍板的执行约定（替代一切技术护栏）**：
- **由用户人工保证**：执行三 agent 回合制期间，同一时间只有一个会话在改代码 / 用部署环境。发起前用户自行确认没有其他开发会话或手工编辑在同棵树 / 同套环境上跑。
- 因此：**存档点 `git add -A` 保持不变**（不改路径白名单）；**不引入 worktree 隔离**。两项均**决定不做**。
- 兜底仍在：检查 agent 会报"工作区混入未申报改动"、主会话收口会 grep 验提交未污染（§9 第 3 项）——万一约定没守住也能当场抓出，但这是兜底不是依赖。

---

## 8. 失败即停 + 留现场

任一阶段到 maxRounds 仍没过 → **整条停**，不回滚上阶段、不跳过往下冲（避免拿烂地基继续）。脚本 return `{stopped:true, failedStage, lastVerdict}`；账本每回合已写、工作树留未提交实现、浏览器停最后一屏。主会话据此通知用户。

---

## 9. 主会话收口自核五项（首跑验证有效，固化为收尾纪律）

流水线 return 后，主会话**亲自**做（不下放，这是最后一道防线）：
1. **重跑 build/test/typecheck**——首跑正是靠这条才发现 v2/v3 typecheck 实为 FAIL（agent 报喜判 pass）。
2. **校迁移版本号连续**（若动了迁移）。
3. **grep 验提交未污染并发任务**（见 §7）。
4. **关键 softChecks 抽样复核**——对图有没有画、内容对不对这类自判项，抽关键的亲自看一眼证据，别全信 agent。
5. **按"UI 打到最终呈现层"过一眼**端到端最终呈现（即便运行栈 agent 已三段对齐）。
五项全过 → 合成提交（只 commit 不 push）→ 通知用户人工验收。

---

## 10. 输入结构 + 怎么调用

```js
Workflow({
  scriptPath: '/Users/shangyingbin/Documents/notes/workflows/dev-check-loop.workflow.js',
  args: {
    taskId, title,
    workdir,        // 默认 build/test 目录（绝对路径）
    ledgerPath,     // 账本绝对路径，放 ~/Documents/notes/tmp/
    startHead,      // 主会话调用前取的 HEAD（git rev-parse HEAD）
    maxRounds,      // 每阶段回合上限，默认 4
    deployCmd,      // 默认 'bash /Users/shangyingbin/project/omc-docker/docker-run.sh'
    stages: [
      {
        name,                       // 阶段名（标签 + 存档点 commit message）
        specDocs: [ ... ],          // 本阶段权威 spec 路径数组
        scope,                      // 多行：本阶段做什么 / 不做什么
        workdir,                    // 可选，覆盖顶层（前后端混合：后端阶段 omcgo、前端阶段 omcmb/webcode）
        buildCmd,                   // 默认 'go build ./...'（前端阶段 'npm run typecheck'）
        testCmd,                    // 默认 'go test ./...'，按阶段收窄
        hardChecks: [ { id, desc, expected } ],   // 死判项（脚本比对 实测==预期）
        softChecks: [ { id, desc } ],             // 自判项（agent 判 + schema 强制留证据）
        runSegments,                // 提示本阶段大致验哪几段（部署/建数据/查库/DOM），agent 按需取舍
      },
      // ...
    ],
  },
})
```

**调用前主会话三件事**：① `git rev-parse HEAD` 取 `startHead`；② `mkdir -p` 账本目录（脚本不碰文件系统，账本由 agent 写）；③ 确认无其他会话在改这棵树（§7）。

---

## 11. 调用样例（KPI 导出首跑，2026-06-04，四阶段全过 → 合成提交 228152df）

```js
Workflow({
  scriptPath: '/Users/shangyingbin/Documents/notes/workflows/dev-check-loop.workflow.js',
  args: {
    taskId: 'KPI-EXPORT',
    title: 'KPI 数据导出',
    workdir: '/Users/shangyingbin/project/goomc/omcgo',
    ledgerPath: '/Users/shangyingbin/Documents/notes/tmp/kpi-export-ledger.md',
    startHead: '<主会话 git rev-parse HEAD 取的值>',
    maxRounds: 4,
    deployCmd: 'bash /Users/shangyingbin/project/omc-docker/docker-run.sh',
    stages: [
      { name: 'T1-后端骨架', workdir: '…/omcgo', specDocs:[…], scope:'做：export 模块+迁移…\n不做：worker 生成', buildCmd:'go build ./...', testCmd:'go test ./internal/pm/...',
        hardChecks:[{id:'mig-seq', desc:'迁移版本号连续', expected:true}], softChecks:[], runSegments:'部署+查表结构' },
      { name: 'T2-worker流式生成', /* … */ },
      { name: 'T3-前端业务层+两Tab', workdir:'…/omcmb/webcode', buildCmd:'npm run typecheck',
        hardChecks:[{id:'tsc', desc:'webcode typecheck 退出码', expected:0}], /* … */ },
      { name: 'T4-导出按钮+端到端', /* hardChecks: 新建任务 DB count=1 等; softChecks: CSV 中文不乱码 */ },
    ],
  },
})
```

> 首跑结果：T1/T2/T3 各 1 回合、T4 2 回合全过；账本 `~/Documents/notes/tmp/kpi-export-ledger.md`，Run `wf_3b381442-ee4`。

---

## 12. 一句话

机制骨架已试跑验证可用——对抗式检查真抓 bug、死判比对不靠嘴、失败即停不带病下冲；唯一真风险"共享工作树跨任务污染"靠 §7 约定 + 存档点路径白名单根治，自判项放水靠 §9 主会话收口抽验兜底。
