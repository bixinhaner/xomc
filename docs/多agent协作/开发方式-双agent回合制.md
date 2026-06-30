# 开发方式 · 双 agent 回合制 workflow

> 性质：开发方法论。**不入 git**，放 notes。
> 一句话：把一个任务交给两个 agent —— 一个**开发**、一个**检查**，靠一份**账本**跨回合传话，循环到检查通过为止。
> 配套脚本（可运行）：`~/Documents/notes/workflows/dev-check-loop.workflow.js`
> 触发：把这份文档（或说"按双 agent 回合制开发 T-XXXX"）给 Claude，它就按本法执行。

---

## 1. 何时用 / 何时不用

**适合**：单条开发线、范围清晰、能用 `go build`/`go test`/`typecheck` 等命令自证的后端或前端任务（一个 T-NNNN 级别的功能/重构）。

**不适合**：
- 还没锁设计（先 brainstorming）。
- 纯探查/读代码（用 Explore 即可，不需要开发+检查）。
- 多条互相冲突的开发线并行（那是另一套"多泳道并发"，见下方"与并发开发的关系"）。
- 单行 hotfix（直接改，杀鸡用牛刀）。

---

## 2. 核心结构：回合制接力 + 账本

```
准备：主线建账本（notes/tmp/<任务>-ledger.md）
回合 1 · 开发 agent：读账本+spec → 实现 → 自己跑 build/test → 把"改了啥/真实结果"写进账本
回合 1 · 检查 agent：读账本+spec+git diff → 自己重跑 build/test → 逐条对验收 → 把"问题清单"写进账本
        检查 verdict = pass？→ 收工
                       = fail？→ 进回合 2
回合 2 · 开发 agent：读账本里的问题 → 逐条修 → 重跑 → 更新账本
回合 2 · 检查 agent：复核 …
… 循环到 pass 或达到 maxRounds（默认 4）
```

**为什么要账本**：每个 agent 都是**全新的、没有上一回合的记忆**。账本（一个 notes/tmp/ 下的 md 文件）就是它们的共享记忆——开发写"我做了什么"，检查写"我发现什么问题"，下一回合的 agent 读账本就接得上。代码本身留在工作树上（跨回合可见），账本补的是"为什么/发现了什么"这层人话。

**为什么不用 git worktree 隔离**：worktree 会把开发和检查隔到两份不同的代码副本，互相看不见对方的改动。本法是**单线串行**（检查在开发到检查点后才看），不存在并行写冲突，所以直接在主工作树上改最简单。worktree 隔离是给"多条开发线同时改"用的，本法用不上。

---

## 3. 两个 agent 的职责红线

**开发 agent**：
- 在指定工作目录的主工作树直接改；只改代码与测试，**不 git commit**。
- 实现后**必须自己跑** build/test，记真实 exit code / 失败行，不许写"应该能过"。
- 关键逻辑补单测，覆盖成功 + 失败路径。
- 收尾把本回合追加进账本。

**检查 agent（价值核心 = 对抗式验证，默认怀疑）**：
- **自己重跑 build/test**，不信开发的口头声明。
- 逐条核对验收标准是否真被满足；查漏改（如"枚举要改两处只改一处"）、假通过的测试、边界遗漏。
- **关联本回合真实改动反推额外风险**：不只盯固定验收清单。脚本会把开发 agent 本回合的自报（摘要 + 改动文件）连同 `git diff` 一起喂给检查方，要它据此想清楚"这次实际动了什么 → 因此还有哪些没列进验收、却确实该验的点"（波及的调用方、落库丢键、漏掉的边界值等），逐条核对记入账本；构成阻塞的升级进问题清单。
- 不确定就判 fail——宁可多一轮，不放过。
- 只有"无阻塞问题 + 验收全覆盖 + build/test 真过"才判 pass。
- 收尾把问题清单追加进账本。

---

## 4. 范围划分（关键纪律）

workflow **只覆盖能用命令自证的层**：实现 + 单测 + `go build`/`go test`/`typecheck`。

**不塞进 workflow、在 workflow 之后做**：对运行栈的三段验证（重新部署 → curl/接口建数据 → 查 DB 指纹比对 → playwright 看 DOM）。

**运行栈验证用「混合」方式（默认套路，2026-05-30 用户拍板固化）**——取证交子 agent、判断留主线：

- **交给"取证子 agent"的体力活**：重新部署、建数据、触发任务、查库拿指纹、curl 拿响应、playwright 抓 DOM。这些机械、可并行、靠命令自证，子 agent 胜任，省主线上下文。
- **子 agent 只回"原始证据"，不回结论**：要它交回库里真实的几行、DOM 节点的真实文字、接口的真实 JSON——**证据先于结论**。
- **主线保留三件事**（不下放）：
  1. **对着原始证据下"通过/不通过"** ——运行栈验证是最后一道防线，恰因前面开发/检查 agent 多用内存桩、验不出落库丢键/滚动没真跑/过期没真删这类 DB 层缺陷；若把判断也下放成"另一个 agent 写报告"，等于又回到"信总结、没亲眼看指纹"。所以判断必须主线对着原始证据亲自做。
  2. **部署护栏显式交代** ——部署是破坏性操作（碰错 docker 目录会删运行栈网络致全栈断网），子 agent 是空白上下文、不带这条记忆，**必须在 prompt 里显式写明只走指定一键脚本**。
  3. **给用户的浏览器收口** ——子 agent 的输出回到的是主线、不是用户；浏览器是共享同一实例，子 agent 可停在最后一屏，但"已停在 X 页，请人工查看"这句话由主线转达。

**commit 时机**：workflow 全程只改代码不 commit；等主线做完运行栈三段验证（混合方式）、确认无误后，再由主线 commit（默认只 commit 不 push）。

---

## 5. 怎么调用

```
Workflow({
  scriptPath: '/Users/shangyingbin/Documents/notes/workflows/dev-check-loop.workflow.js',
  args: { …见下方参数清单… }
})
```

**args 参数清单**：

| 参数 | 含义 |
|---|---|
| `taskId` | 任务号，如 `'T-0182'`（也用于 agent 标签、账本默认名） |
| `title` | 任务名 |
| `workdir` | 跑 build/test 的目录（绝对路径） |
| `specDocs` | 权威 spec 文档路径数组（实施计划/决策记录/设计文档），agent 会自己读 |
| `ledgerPath` | 账本文件绝对路径，放 `~/Documents/notes/tmp/` |
| `scope` | 多行字符串：本任务做什么 / 不做什么 |
| `acceptance` | 验收标准数组，逐条 |
| `buildCmd` | 默认 `'go build ./...'` |
| `testCmd` | 默认 `'go test ./...'`（按任务收窄，如 `'go test ./internal/pm/...'`） |
| `maxRounds` | 默认 4 |

主线在调用前先 `mkdir -p` 账本目录即可（脚本不碰文件系统，由 agent 写账本）。

**`acceptance` 写法纪律（决定检查质量的上游）**：验收逐条按**业务级"前提 → 动作 → 预期"**写，颗粒度越细，检查方越无处可糊弄。

- ❌ 太粗：`告警通知能用`
- ✅ 够细：`CMCC 设备触发门限告警 → 应在 X 秒内发出邮件，且 Y 秒去重窗口内同源告警不重复发`

固定 `acceptance` 管"功能对不对"；检查方会再**结合本回合真实改动反推清单之外的额外风险**（见 §3 检查职责第 3 条），两者互补——所以 `acceptance` 不必穷举所有边界，把核心业务路径写细即可，边界遗漏交给反推那层兜。

---

## 6. 与"多泳道并发开发"的关系

- **本法（双 agent 回合制）** = 把**一个任务**做扎实（开发 + 对抗式检查）。
- **多泳道并发** = 把**多个互不冲突的任务**同时推进（先只读探查确认文件不撞车，再每条泳道一个 agent）。
- 两者可叠加：多泳道并发时，**每条泳道内部**都可以用本法（一对开发+检查）。但首次先用单任务把本法跑顺，再铺开到多泳道（避免一上来 N 对 agent 一起乱）。

---

## 7. 重开会话 / 上下文加载（取证可下放，判断留主线）

> 与 §4 运行栈验证同构：重开会话时"读懂现状"也走混合——**侦察可下放，承重判断留主线**。2026-05-30 用户拍板固化。

重开会话要重建后续整场要用的工作记忆，这件事有两半性质不同：

- **可下放给子 agent 的"现状侦察"**：git/迁移/部署当前状态、"哪些文件实现了某能力"、相关文档章节定位、dev 环境是否在跑。这些是可并行的扇出搜索，子 agent 读摘要回结论、不把文件原文灌进主线上下文，省的是主线上下文预算。
- **留主线亲自读的"承重材料"**：权威 spec、账本、以及**接下来要改 / 要判其正确性的那几段代码**。这部分细节直接喂给后续决策，交给子 agent 只会得到**有损压缩**，而丢掉的恰是 bug 藏身处。

**为什么判断不能下放**（与 §4 同一条教训）：T-0184 开工时"全网/设备组两个维度根本没实现"，是主线亲读两个文件的枚举、发现不一致才抓出来的；若只拿子 agent 一句"维度定义在这两处"，低估会被盖住、整任务按错误工作量开工。

**两条具体纪律**：

1. **不要二次摘要已 curated 的文档**：TODO.MD 的"重开提示词"、账本"回合 0"本就是手工压缩好的高信号摘要。子 agent 再去总结它们 = 摘要的摘要，越压越糊、收益极低。要扇出的是对**代码和运行状态**的侦察，不是把这些已 curated 的文档再总结一遍。
2. **承重判断主线复核**：即便子 agent（或账本回合 0）给了摘要，主线对**决策所依赖的那几条承重事实**仍要亲自复核一遍再动手。

**操作顺序**：先主线亲读"重开提示词 + 账本"（便宜、高信号、手工curated）→ 据此判断哪些是"承重事实"要自己复核、哪些是"侦察体力活"可扇出 → 再决定要不要起侦察子 agent。

---

## 附录 A · T-0182 调用样例（本法首试）

```js
Workflow({
  scriptPath: '/Users/shangyingbin/Documents/notes/workflows/dev-check-loop.workflow.js',
  args: {
    taskId: 'T-0182',
    title: '聚合任务模型与维度扩展（基础）',
    workdir: '/Users/shangyingbin/project/goomc/omcgo',
    specDocs: [
      '/Users/shangyingbin/Documents/notes/PM功能设计/pm-redesign-impl-plan.md',
      '/Users/shangyingbin/Documents/notes/PM功能设计/pm-redesign-决策记录.md',
      '/Users/shangyingbin/project/goomc/docs/design/pm-metric-aggregation-dashboard-redesign-20260529.md',
    ],
    ledgerPath: '/Users/shangyingbin/Documents/notes/tmp/T0182-ledger.md',
    scope: [
      '做：任务表(pm_tasks)加字段 technology/is_builtin/expire_days；维度枚举两处(adhoc 模型 + aggregator 查询)加 product/band；',
      '    aggregator 加"产品维度"分组(照搬 device_group 模式按 product_id 分组)；制式过滤(QueryRequest 加 Technologies + JOIN devices 过滤 + 建任务拒跨制式)；',
      '    迁移 000220(加字段 + 扩 dimension CHECK 约束含 product/band，up/down 配对)；配置加全局开关 pm.storage.store_all_metrics(默认 true)，仅存所选时落库前按 task.metric_paths 过滤。',
      '不做：band 维度的聚合实现(拆到 T-0183，本任务只把 band 加进枚举/约束)；前端(向导在 T-0185)；多粒度(保留数组但 create 校验 len==1，不动 executor 循环)。',
    ].join('\n'),
    acceptance: [
      '建"产品-LTE"任务，结果按产品分组正确落库(真机 1202000240194DP0015，product_id=9259a43e-…，tech=lte)',
      '制式过滤生效：LTE 任务范围里选不到 NR 设备，后端也拒跨制式',
      '存储开关两档行为：全存=落全部指标、仅存所选=只落所选 N 个',
      '关键聚合单测覆盖成功 + 失败(空维度、跨制式)',
    ],
    buildCmd: 'go build ./...',
    testCmd: 'go test ./internal/pm/...',
    maxRounds: 4,
  },
})
```

> 注：上面 ① ② ③ 的"对运行栈验证"按 §4 由主线在 workflow 之后做（重新部署 → curl 建任务 → DB 指纹比对）；workflow 阶段聚焦实现 + 单测(④) + build/test 自证。
