export const meta = {
  name: 'batch-task-loop',
  description: '多任务批量·无人值守串行：互不相关的多个任务逐单串行推进，每单内部 开发→检查→运行栈 回合制（死判+自判+对抗检查），每单独立分支+提交+PR，失败即跳过继续（不连累整批），支持软出口（已修不改/需人工决策）',
  phases: [
    { title: '准备' },
    { title: '任务' },
    { title: '收口' },
  ],
}

// ── 与旧 dev-check-loop 的区别（详见 ~/Documents/notes/知识库/开发方式-多任务批量-无人值守串行.md）──
//  旧脚本：一个功能的多个【相互依赖】阶段；账本传上下文；失败即停；末尾合成一个提交、一个分支。
//  本脚本：互不相关的多个【独立】任务；每单自成一体；失败即跳过继续；每单独立分支+提交+PR、允许 push。
//  内循环（开发→检查→运行栈，死判+自判+对抗检查）两者一致，原样复用。
//
// ── 参数（通过 Workflow args 传入）──
// {
//   batchId, title, repoRoot, ledgerPath, maxRounds, deployCmd, push(默认 true),
//   tasks: [
//     {
//       id,            // 展示用，如 '#191'
//       issue,         // GitHub issue 号（gh comment / Closes 用），如 191
//       name,          // 短名（分支 slug + 标签 + 存档点）
//       branchType,    // feat/fix/refactor/docs/chore（分支名前缀，缺省 fix）
//       slug,          // 分支名尾段，如 'role-delete-guard'
//       specDocs:[...], // 权威 spec（issue 正文落地的文件 / 路径 / 链接）
//       scope,          // 多行：做什么 / 不做什么
//       workdir,        // 该单 build/test 目录（绝对路径），缺省顶层 repoRoot
//       buildCmd, testCmd,
//       hardChecks:[{id,desc,expected}],  // 死判项（脚本比 实测==预期）
//       softChecks:[{id,desc}],           // 自判项（agent 判+强制留证据）
//       runSegments,    // 本单大致验哪几段
//       needsDeploy,    // true=要起运行栈(部署+DOM/复现)；false=纯单测(只跑 test 取死判值，不部署)。缺省 false
//       gate,           // 可选软出口：'diagnose-first'(复现不了可判已修) / 'design-decision'(遇设计取舍可停下出方案)
//     },
//     ...
//   ]
// }
const p = (typeof args === 'string' ? JSON.parse(args) : args) || {}
const batchId = p.batchId || 'BATCH'
const ledger = p.ledgerPath
const maxRounds = p.maxRounds || 4
const repoRoot = p.repoRoot || '/Users/shangyingbin/project/goomc'
const deployCmd = p.deployCmd || 'bash /Users/shangyingbin/project/omc-docker/docker-run.sh'
const allowPush = p.push !== false
const DEFAULT_BUILD = 'go build ./...'
const DEFAULT_TEST = 'go test ./...'
const tasks = Array.isArray(p.tasks) ? p.tasks : []

// ── 死判/自判结果 schema（与旧脚本一致）──
const DEV_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['summary', 'filesChanged', 'buildPassed', 'testPassed'],
  properties: {
    summary: { type: 'string', description: '本回合做了什么（业务语言）' },
    filesChanged: { type: 'array', items: { type: 'string' }, description: '改动/新增的文件路径' },
    buildPassed: { type: 'boolean', description: 'build 是否通过（真实结果）' },
    testPassed: { type: 'boolean', description: 'test 是否通过（真实结果）' },
    testEvidence: { type: 'string', description: 'build/test 的关键输出片段（exit code 或失败行）' },
    addressedIssues: { type: 'array', items: { type: 'string' }, description: '本回合修掉的上一轮问题（首轮为空）' },
    openQuestions: { type: 'array', items: { type: 'string' }, description: '留给检查方/用户的疑问' },
    softExit: {
      type: 'object',
      additionalProperties: false,
      required: ['kind', 'reason', 'evidence'],
      description: '软出口：仅当本单开了 gate 且你判定"这单不该硬改代码"时填，否则不填（kind=none）',
      properties: {
        kind: { type: 'string', enum: ['none', 'done-no-change', 'needs-input'], description: 'done-no-change=问题不复现/已被它单修掉，无需改码；needs-input=遇到只有人能定的设计取舍，需停下等决策' },
        reason: { type: 'string', description: '为什么判这个软出口' },
        evidence: { type: 'string', description: '支撑判断的真实原始证据（复现命令真实输出/接口真实 JSON/代码事实）；needs-input 时这里给方案选项' },
      },
    },
  },
}

const CHECK_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['verdict', 'reRanBuild', 'reRanTest', 'blockingIssues', 'planConformance', 'acceptanceCoverage'],
  properties: {
    verdict: { type: 'string', enum: ['pass', 'fail'], description: 'pass=无阻塞问题且符合 spec 且 build/test 真过' },
    reRanBuild: { type: 'boolean', description: '检查方自己重跑了 build' },
    reRanTest: { type: 'boolean', description: '检查方自己重跑了 test' },
    buildTestEvidence: { type: 'string', description: '检查方重跑 build/test 的关键输出' },
    blockingIssues: {
      type: 'array',
      items: {
        type: 'object',
        additionalProperties: false,
        required: ['title', 'detail'],
        properties: {
          title: { type: 'string' },
          detail: { type: 'string', description: '问题 + 文件位置 + 建议修法' },
        },
      },
    },
    planConformance: {
      type: 'array',
      description: '逐条核对实现是否符合本单 spec（scope 的做什么/不做什么 + spec 指定结构）',
      items: {
        type: 'object',
        additionalProperties: false,
        required: ['scopeItem', 'conformance'],
        properties: {
          scopeItem: { type: 'string', description: 'scope 里的一条（做/不做）' },
          conformance: { type: 'string', enum: ['ok', 'deviation', 'over-scope', 'missing'], description: 'ok=符合 / deviation=偏离做法 / over-scope=越界做了不该做的 / missing=该做没做' },
          note: { type: 'string' },
        },
      },
    },
    derivedRisks: {
      type: 'array',
      items: { type: 'string' },
      description: '基于本回合真实改动反推、固定验收之外的额外风险点，逐条带核对结果',
    },
    nonBlockingNotes: { type: 'array', items: { type: 'string' } },
    acceptanceCoverage: {
      type: 'array',
      items: {
        type: 'object',
        additionalProperties: false,
        required: ['criterion', 'covered'],
        properties: {
          criterion: { type: 'string' },
          covered: { type: 'string', enum: ['yes', 'partial', 'no'] },
          note: { type: 'string' },
        },
      },
    },
    softExitVerdict: {
      type: 'object',
      additionalProperties: false,
      required: ['confirmed'],
      description: '仅当开发方提了 softExit 才填：对抗式核实其证据是否站得住',
      properties: {
        confirmed: { type: 'boolean', description: 'true=证据成立、认可该软出口；false=驳回（如"说不复现"实则能复现）→ 退回正常修复回合' },
        reason: { type: 'string', description: '认可/驳回的依据' },
      },
    },
  },
}

const RUNVERIFY_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['deployed', 'deployVia', 'segmentsRun', 'hardResults', 'softResults'],
  properties: {
    deployed: { type: 'boolean', description: '是否成功部署（needsDeploy=false 的纯单测单：未部署也回 true，并在 deployVia 注明"纯单测免部署"）' },
    deployVia: { type: 'string', description: '实际走的部署命令（护栏自证，必须是指定一键脚本）；纯单测单写"纯单测免部署，只跑 test"' },
    segmentsRun: { type: 'array', items: { type: 'string' }, description: '本回合实际跑了哪几段（跑测试/部署/建数据/查库/DOM）' },
    hardResults: {
      type: 'array',
      description: '死判项实测结果（脚本拿 actual 对回 hardChecks.expected 比对，你只回原始实测值，不要自己下结论）',
      items: {
        type: 'object',
        additionalProperties: false,
        required: ['id', 'actual', 'rawEvidence'],
        properties: {
          id: { type: 'string', description: '对应 hardChecks 的 id' },
          actual: { type: 'string', description: '实测值（原始数字/字符串/布尔，转字符串回）' },
          rawEvidence: { type: 'string', description: '支撑该实测值的原始证据：测试真实输出 / 库里真实那几行 / 接口真实 JSON' },
        },
      },
    },
    softResults: {
      type: 'array',
      description: '自判项判定（机器没法数的模糊视觉项，自己判但必须配真实原始证据）',
      items: {
        type: 'object',
        additionalProperties: false,
        required: ['id', 'verdict', 'rawEvidence'],
        properties: {
          id: { type: 'string', description: '对应 softChecks 的 id' },
          verdict: { type: 'string', enum: ['pass', 'fail'] },
          rawEvidence: { type: 'string', description: 'DOM 真实文字 / 截到的真实内容 / 接口真实 JSON，编不出就过不了' },
        },
      },
    },
    notes: { type: 'string' },
  },
}

const BRANCH_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['onBranch', 'cleanBase'],
  properties: {
    onBranch: { type: 'string', description: '当前所在分支名（应为本单新建的 feature 分支）' },
    cleanBase: { type: 'boolean', description: '是否确实从干净的 main 切出（无前一单残留改动）' },
    baseHead: { type: 'string', description: '切分支时 main 的 HEAD' },
    note: { type: 'string' },
  },
}

const DELIVER_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['outcome', 'backToMainClean'],
  properties: {
    outcome: { type: 'string', enum: ['pr-opened', 'closed-no-change', 'left-for-human', 'blocked'] },
    committed: { type: 'boolean' },
    commitHash: { type: 'string' },
    pushed: { type: 'boolean' },
    prUrl: { type: 'string', description: '开了 PR 则填链接' },
    issueCommented: { type: 'boolean', description: '是否在 issue 上留了说明/卡点/方案' },
    backToMainClean: { type: 'boolean', description: '收尾是否已 git switch main 且工作树干净（不带脏给下一单）' },
    note: { type: 'string' },
  },
}

// ── 死判比对：拿运行栈/单测 agent 回的实测值，对回本单断言，纯代码判等 ──
function judge(task, run) {
  const fails = []
  // needsDeploy=false 的纯单测单：不要求真部署
  if (task.needsDeploy && (!run || run.deployed !== true)) {
    fails.push({ kind: 'deploy', detail: `部署未成功（deployed=${run && run.deployed}，deployVia=${run && run.deployVia}）` })
  }
  const hardChecks = task.hardChecks || []
  const hardResults = (run && run.hardResults) || []
  for (const hc of hardChecks) {
    const r = hardResults.find(x => x.id === hc.id)
    if (!r) {
      fails.push({ kind: 'hard', id: hc.id, detail: `死判项 ${hc.id}「${hc.desc}」未回实测值` })
      continue
    }
    const a = String(r.actual).trim()
    const e = String(hc.expected).trim()
    if (a !== e) {
      fails.push({ kind: 'hard', id: hc.id, detail: `死判项 ${hc.id}「${hc.desc}」实测=${a} 预期=${e}（证据：${r.rawEvidence || '无'}）` })
    }
  }
  const softChecks = task.softChecks || []
  const softResults = (run && run.softResults) || []
  for (const sc of softChecks) {
    const r = softResults.find(x => x.id === sc.id)
    if (!r) {
      fails.push({ kind: 'soft', id: sc.id, detail: `自判项 ${sc.id}「${sc.desc}」未回判定` })
      continue
    }
    if (r.verdict !== 'pass') {
      fails.push({ kind: 'soft', id: sc.id, detail: `自判项 ${sc.id}「${sc.desc}」判 fail（证据：${r.rawEvidence || '无'}）` })
    }
  }
  return { pass: fails.length === 0, fails }
}

function specBlock(task) {
  return (task.specDocs || []).map((d, i) => `  ${i + 1}. ${d}`).join('\n') || '  (见 scope)'
}
function hardBlock(task) {
  return (task.hardChecks || []).map(h => `  - [死判 ${h.id}] ${h.desc}（预期：${h.expected}）`).join('\n') || '  (无)'
}
function softBlock(task) {
  return (task.softChecks || []).map(s => `  - [自判 ${s.id}] ${s.desc}`).join('\n') || '  (无)'
}
function branchName(task) {
  return `${task.branchType || 'fix'}/${task.issue || ''}-${task.slug || task.name}`
}
function gateBlock(task) {
  if (task.gate === 'diagnose-first') {
    return `\n【本单开了软出口 · diagnose-first】先复现再决定：若**确实复现**问题，正常修+补测；若**复现不了/已被它单修掉**，不要硬造改动——在 softExit 里填 kind='done-no-change' + 复现命令真实输出作证据。`
  }
  if (task.gate === 'design-decision') {
    return `\n【本单开了软出口 · design-decision】若遇到**只有人能拍板的设计取舍**（多种实现各有取舍、涉及数据库迁移口径等），不要替用户擅自定——能干的部分先干完，把取舍点在 softExit 里填 kind='needs-input' + 给出 2-3 个方案选项作 evidence。`
  }
  return ''
}

function branchPrompt(task) {
  return `你是「分支 agent」，为任务 ${task.id}（issue #${task.issue || '?'}）开工建分支。仓库根：${repoRoot}

只做一件事：从**干净的 main**切出本单的 feature 分支。
- 先确认工作树干净：\`git -C ${repoRoot} status --porcelain\` 应为空。**若非空**（前一单可能留了脏），说明收尾没干净——如实在 note 里报告，并 \`git -C ${repoRoot} switch main\`（不要 stash、不要丢码）后再判断。
- \`git -C ${repoRoot} switch main\`，记下 \`git -C ${repoRoot} rev-parse HEAD\` 作 baseHead。
- **不要 pull 远端**（除非另有指令）。
- \`git -C ${repoRoot} switch -c ${branchName(task)}\`。
- 返回当前分支名、是否干净切出、baseHead。`
}

function devPrompt(task, round, prevFail) {
  let fixSection
  if (prevFail && prevFail.source === 'check') {
    fixSection = `\n本回合是【修复回合 · 检查层】。上一轮检查方提的阻塞问题（逐条修掉）：\n${(prevFail.issues || []).map((b, i) => `  ${i + 1}. ${b.title} — ${b.detail}`).join('\n')}\n`
  } else if (prevFail && prevFail.source === 'runstack') {
    fixSection = `\n本回合是【修复回合 · 运行栈层】。检查层已过，但实测对不上验收，逐条修：\n${(prevFail.issues || []).map((d, i) => `  ${i + 1}. ${d}`).join('\n')}\n`
  } else if (prevFail && prevFail.source === 'softexit-rejected') {
    fixSection = `\n本回合：你上一轮提的软出口被检查方**驳回**了（理由：${prevFail.reason || '证据不成立'}）。说明这单确实需要改代码——按正常修复推进，别再走软出口。\n`
  } else {
    fixSection = `\n本回合是【首次实现】。\n`
  }
  return `你是「开发 agent」，负责任务 ${task.id}（issue #${task.issue || '?'}）「${task.name}」。工作目录：${task.workdir || repoRoot}
当前应已在分支 \`${branchName(task)}\` 上。

【先读账本】${ledger}
这是你和检查/运行栈 agent 的共享记忆。你没有上一回合的记忆，账本里有前情——先读它。若账本尚不存在，由你创建并写入抬头。
${fixSection}
【权威 spec（必读，照此实现）】
${specBlock(task)}

【本单范围】
${task.scope || '(见 spec)'}

【本单验收（你的实现要让这些可被验证）】
死判项（之后会被脚本按预期值比对）：
${hardBlock(task)}
自判项（之后由运行栈 agent 看真实证据判定）：
${softBlock(task)}
${gateBlock(task)}
【纪律】
- 在 ${task.workdir || repoRoot} 主工作树直接改；只改代码与测试，**绝不 git commit**（提交/PR 交给收尾 agent）。
- 实现后必须自己跑：\`${task.buildCmd || DEFAULT_BUILD}\` 和 \`${task.testCmd || DEFAULT_TEST}\`，记真实 exit code / 失败行，不许写"应该能过"。
- 关键逻辑补单测，覆盖成功 + 失败路径。
- 严守 scope 边界：不碰 scope 说"不做"的；spec 指定的结构/约束/迁移号严格照做，不臆造。

【收尾必做】把本回合"改了啥 / build & test 真实结果 / 修掉哪些上轮问题 / 遗留疑问 / （若有）软出口判断"**追加**写入账本 ${ledger}（带"${task.id} 回合${round}·开发"小标题），再返回结构化结果。`
}

function checkPrompt(task, round, dev) {
  const changeSection = dev ? `\n【开发 agent 本回合自报（结合真实 \`git diff\` 核对，不全信自报）】\n- 摘要：${dev.summary || '(无)'}\n- 改动文件：${(dev.filesChanged || []).map(f => `\n    - ${f}`).join('') || ' (无)'}\n` : ''
  const softExitSection = (dev && dev.softExit && dev.softExit.kind && dev.softExit.kind !== 'none')
    ? `\n【⚠ 开发方提了软出口，必须对抗式核实】kind=${dev.softExit.kind}，理由：${dev.softExit.reason}，证据：${dev.softExit.evidence}\n你要判定该证据是否真站得住（如"说不复现"——你自己去复现一次看是不是真不复现；如"需人工决策"——确认是不是真的只有人能定、不是 agent 偷懒）。把结论填进 \`softExitVerdict\`：站得住 confirmed=true，否则 confirmed=false 并说明（脚本会据此退回正常修复回合）。\n`
    : ''
  return `你是「检查 agent」，对任务 ${task.id}（issue #${task.issue || '?'}）当前实现做**对抗式复核**。工作目录：${task.workdir || repoRoot}

【先读账本】${ledger}
开发 agent 刚写了本回合做了啥。读它 + 读权威 spec + 看真实代码改动（\`git diff\` / \`git status\`）。
${changeSection}${softExitSection}
【权威 spec】
${specBlock(task)}

【本单范围（scope）】
${task.scope || '(见 spec)'}

【本单验收】
死判项：
${hardBlock(task)}
自判项：
${softBlock(task)}

【你的职责 — 默认怀疑，不信开发的口头声明】
1. **自己重跑** \`${task.buildCmd || DEFAULT_BUILD}\` 和 \`${task.testCmd || DEFAULT_TEST}\`，记真实输出。
2. 逐条对验收核对：代码是否真覆盖？单测是否真测了成功+失败路径？有没有漏掉的 spec 要点？
3. **对照 spec 核符合性**（一等职责）：逐条读 scope + spec 指定结构，核对有没有 **偏离** / **越界** / **漏做**。逐条记入 \`planConformance\`；阻塞的升级进 \`blockingIssues\`。
4. **关联真实改动反推额外风险**：对照自报 + 真实 diff，想"实际动了什么 → 还有哪些没列进验收却该验的点"。逐条记入 \`derivedRisks\`；阻塞的升级进 \`blockingIssues\`。
5. 不确定就判 fail——宁可多一轮也不放过。只有"无阻塞 + 符合 spec + 验收覆盖 + build/test 真过"才判 pass。

【收尾必做】把"重跑结果 + 阻塞清单 + 符合性核对 + 反推风险 + 验收覆盖 +（若有）软出口核实"**追加**写入账本 ${ledger}（带"${task.id} 回合${round}·检查"小标题），再返回结构化结果。`
}

function runVerifyPrompt(task, round) {
  const deploySection = task.uiVerify
    ? `【本单为前端 UI 验证，免部署、免重建】**不要**跑 \`${deployCmd}\`、不要 docker build、不要碰任何部署目录。
- 前端 Vite dev server 已在 \`http://localhost:3000\` 运行（HMR 已开，开发 agent 本回合的前端改动已自动热更生效，无需重建）；后端栈已起、:3000 经 /api 代理到后端。
- 用 **playwright** 进 \`http://localhost:3000\` 做 DOM 取证（账号 admin / admin123）。**进页面走菜单逐级点击，禁止直接敲 URL**（带 URL query 参数的钻取场景例外——那正是被验对象）。
- deployed 回 true、deployVia 写"前端 :3000 playwright 免部署"。
- 死判/自判实测值来自真实 DOM（playwright snapshot 文字 / 元素属性 / network 响应），编不出就过不了。
- playwright 纪律：不主动截图、验完不主动关浏览器、停在最后一屏。`
    : task.needsDeploy
    ? `【⚠ 部署护栏 —— 你是空白上下文，以下铁律必须遵守】
- 部署**只跑这一条命令**：\`${deployCmd}\`（宿主编译 + docker 重建全栈）。
- **绝不碰** \`goomc/deployments/docker\` —— 碰错会删运行栈网络致全栈断网。
- 部署后验证：\`cd /Users/shangyingbin/project/omc-docker && docker compose ps\`（基础设施 healthy / 迁移 exited(0) / 应用 running）+ \`curl -fsS http://localhost:7557/healthz\`。app 健康经 :8081 nginx 验前端可达 + 经 :3000 验 API 代理。非健康视为未达可服务态，deployed=false。

【⚠ 环境坑】
- 明文 curl 登录被禁（前端走加密），\`POST /auth/login\` 明文会被拒。
- 内部 X-API-Key 对受限接口（如 /pm/adhoc/*）无权限。
- 要端到端验受限接口/UI：走前端 :3000 + playwright（账号 admin / admin123），或直接读 DB。
- playwright 纪律：不主动截图、验完不主动关浏览器、停在最后一屏。`
    : `【本单为纯单测，免部署】不要起 docker、不要碰任何部署目录。只在 ${task.workdir || repoRoot} 跑相关测试命令取死判实测值。
- deployed 回 true、deployVia 写"纯单测免部署，只跑 test"。
- 死判项的实测值来自测试真实输出（如跑 \`go test -run <用例名> -v\` 看 PASS/FAIL、或断言里的真实数值）。`
  return `你是「运行栈验证 agent」，对任务 ${task.id}（issue #${task.issue || '?'}）做**取证**。工作目录：${task.workdir || repoRoot}

检查 agent 已确认 build/test 层与 spec 符合性通过。现在取回真实证据供脚本死判。

【先读账本】${ledger}（了解本单做了什么、验收要点）

${deploySection}

【取证（本单大致验：${task.runSegments || (task.needsDeploy ? '部署 + 按需查库/DOM' : '跑相关单测取死判值')}）】
死判项 —— 测出真实实测值，**只回原始数字/字符串 + 那几行原始证据，不要自己下"通过/不通过"结论**（脚本会用预期值比对）：
${hardBlock(task)}
自判项 —— 机器没法数的自己判 pass/fail，但**必须配真实原始证据**，编不出就过不了：
${softBlock(task)}

【收尾必做】把"部署/测试结果 + 跑了哪几段 + 每个死判项实测值与证据 + 每个自判项判定与证据"**追加**写入账本 ${ledger}（带"${task.id} 回合${round}·运行栈"小标题），再返回结构化结果。`
}

function deliverPrompt(task, outcome, ctx) {
  const branch = branchName(task)
  const issue = task.issue || '?'
  // 关单关键字：整单修完用 Closes（合并即关）；子项/部分修复用 Refs（关联但不关整单）。缺省 Closes。
  const issueKeyword = task.issueKeyword || 'Closes'
  let body
  if (outcome === 'passed') {
    body = `本单三环节全过，把它**正式落地为一个 PR**：
1. 在 ${task.workdir || repoRoot} 所在仓库：\`git -C ${repoRoot} add -A\` → \`git -C ${repoRoot} commit\` 一个规范提交（Conventional Commits 中文描述，scope 对应模块；footer 带 \`${issueKeyword} #${issue}\`${issueKeyword === 'Refs' ? '（本单是 issue 的子项/部分修复，用 Refs 不用 Closes，**绝不能写 Closes**，合并不可误关整单）' : ''}）。**绝不 --no-verify、绝不 --force**。
2. ${allowPush ? `推送并开 PR：\`git -C ${repoRoot} push -u origin ${branch}\` → \`gh pr create\`（base main，body 含 \`${issueKeyword} #${issue}\` 与本单验收勾选）。回填 prUrl。` : '本批不 push（push=false）：只本地提交，pushed=false、不开 PR。'}
3. **收尾必做**：\`git -C ${repoRoot} switch main\` 且确认工作树干净（\`git status --porcelain\` 为空），backToMainClean=true，好让下一单从干净 main 切。
outcome 回 ${allowPush ? "'pr-opened'" : "'pr-opened'（注：未 push 但视为已落地提交）"}。`
  } else if (outcome === 'done-no-change') {
    body = `本单判定为**已修/不复现，无需改码**（软出口 done-no-change）。证据：${ctx.evidence || '见账本'}
1. 若工作树有零星改动（如只加了个回归测试），可 \`git -C ${repoRoot} add -A && git commit\` 后${allowPush ? ` push 开 PR` : ' 仅本地提交'}；若**完全无改动**，不要造提交。
2. 在 issue 上留结论：\`gh issue comment ${issue}\` 贴"复现不出/已被它单修复 + 真实证据"，建议关单（**不要自己关，留人工**）。issueCommented=true。
3. **收尾必做**：\`git -C ${repoRoot} switch main\`、工作树干净、backToMainClean=true。
outcome 回 'closed-no-change'。`
  } else if (outcome === 'needs-input') {
    body = `本单遇到**只有人能拍板的设计取舍**（软出口 needs-input）。方案选项：${ctx.evidence || '见账本'}
1. 能干完的部分（如子问题①）的改动，\`git -C ${repoRoot} add -A && git commit\` 到分支 ${branch}（提交信息注明"部分完成，②待决策"）${allowPush ? `，并 push（可开 draft PR）` : '，仅本地'}。
2. 在 issue 上留方案：\`gh issue comment ${issue}\` 贴清取舍点 + 2-3 个方案选项 + 各自利弊，请人定夺。issueCommented=true。
3. **收尾必做**：\`git -C ${repoRoot} switch main\`、工作树干净、backToMainClean=true。
outcome 回 'left-for-human'。`
  } else { // blocked
    body = `本单到 ${maxRounds} 回合仍未过（${ctx.reason || '见账本'}）。**不丢码、不连累整批**：
1. 把半成品提交到分支保命：\`git -C ${repoRoot} add -A && git commit -m "wip(${task.name}): 未通过，留半成品供人工接手"\`${allowPush ? `，并 push 分支 ${branch}（不开 PR 或开 draft）` : '，仅本地'}。
2. 在 issue 上留卡点：\`gh issue comment ${issue}\` 贴最后一轮失败原因 + 卡在哪。issueCommented=true。
3. **收尾必做**：\`git -C ${repoRoot} switch main\`、工作树干净、backToMainClean=true，让下一单不被带脏。
outcome 回 'blocked'。`
  }
  return `你是「收尾交付 agent」，处理任务 ${task.id}（issue #${issue}）的落地。仓库根：${repoRoot}，分支 ${branch}。

${body}

【铁律】绝不 \`--force\`、绝不 \`--no-verify\`、绝不直接动 main（只在本单 feature 分支提交）。pre-commit 失败修根因、不 --amend 绕过。
返回结构化结果。`
}

// ════════════════ 主流程 ════════════════
phase('准备')
if (!tasks.length) {
  log('未提供 tasks，无事可做。')
  return { batchId, stopped: true, reason: 'no-tasks' }
}
log(`${batchId} 多任务批量·串行：${tasks.length} 单，每单≤${maxRounds} 回合，push=${allowPush}，账本 ${ledger}`)

const results = []

for (let ti = 0; ti < tasks.length; ti++) {
  const task = tasks[ti]
  const tLabel = `${task.id || task.name}（${ti + 1}/${tasks.length}）`
  phase(`任务:${task.id || task.name}`)
  log(`${tLabel} 开始 — 分支 ${branchName(task)}`)

  // ① 建分支（从干净 main 切）
  const br = await agent(branchPrompt(task), {
    label: `${task.id}-branch`, phase: `任务:${task.id || task.name}`, schema: BRANCH_SCHEMA,
  })
  if (!br || br.cleanBase !== true) {
    log(`${tLabel} ⚠ 未能从干净 main 切分支（${br && br.note || '未知'}）——跳过本单，避免污染`)
    results.push({ id: task.id, issue: task.issue, outcome: 'blocked', reason: '建分支失败/工作树不干净' })
    continue
  }

  // ② 内循环：开发→检查→运行栈，回合制
  let prevFail = null
  let finalRound = 0
  let innerOutcome = 'failed'          // passed | soft-done-no-change | soft-needs-input | failed
  let softCtx = {}
  let lastCheck = null
  let lastRun = null

  for (let round = 1; round <= maxRounds; round++) {
    finalRound = round

    const dev = await agent(devPrompt(task, round, prevFail), {
      label: `${task.id}-dev-r${round}`, phase: `任务:${task.id || task.name}`, schema: DEV_SCHEMA,
    })
    log(`${tLabel} 回合${round}·开发：build=${dev && dev.buildPassed} test=${dev && dev.testPassed}｜${(dev && dev.summary) || ''}`)

    const devSoftExit = dev && dev.softExit && dev.softExit.kind && dev.softExit.kind !== 'none' ? dev.softExit : null

    const check = await agent(checkPrompt(task, round, dev), {
      label: `${task.id}-check-r${round}`, phase: `任务:${task.id || task.name}`, schema: CHECK_SCHEMA,
    })
    lastCheck = check

    // 软出口：开发提了 + 检查核实通过 → 直接结束本单（不算失败）
    if (devSoftExit && check && check.softExitVerdict && check.softExitVerdict.confirmed === true) {
      innerOutcome = devSoftExit.kind === 'done-no-change' ? 'soft-done-no-change' : 'soft-needs-input'
      softCtx = { evidence: devSoftExit.evidence, reason: devSoftExit.reason }
      log(`${tLabel} 回合${round}·软出口确认：${devSoftExit.kind}`)
      break
    }
    // 软出口被驳回 → 退回正常修复回合
    if (devSoftExit && check && check.softExitVerdict && check.softExitVerdict.confirmed === false) {
      prevFail = { source: 'softexit-rejected', reason: check.softExitVerdict.reason }
      log(`${tLabel} 回合${round}·软出口被驳回 → 转正常修复`)
      continue
    }

    const nBlock = (check && check.blockingIssues && check.blockingIssues.length) || 0
    log(`${tLabel} 回合${round}·检查：verdict=${check && check.verdict}，阻塞 ${nBlock} 条`)
    if (!check || check.verdict !== 'pass') {
      prevFail = { source: 'check', issues: (check && check.blockingIssues) || [] }
      continue
    }

    // 检查 gate 通过才取证
    const run = await agent(runVerifyPrompt(task, round), {
      label: `${task.id}-run-r${round}`, phase: `任务:${task.id || task.name}`, schema: RUNVERIFY_SCHEMA,
    })
    lastRun = run
    const judged = judge(task, run)
    log(`${tLabel} 回合${round}·运行栈：deployed=${run && run.deployed}，死判 ${judged.pass ? 'PASS' : 'FAIL(' + judged.fails.length + ')'}`)
    if (!judged.pass) {
      prevFail = { source: 'runstack', issues: judged.fails.map(f => f.detail) }
      continue
    }

    innerOutcome = 'passed'
    break
  }

  // ③ 收尾交付（按内循环结局分流）
  let deliverOutcome
  if (innerOutcome === 'passed') deliverOutcome = 'passed'
  else if (innerOutcome === 'soft-done-no-change') deliverOutcome = 'done-no-change'
  else if (innerOutcome === 'soft-needs-input') deliverOutcome = 'needs-input'
  else deliverOutcome = 'blocked'

  const deliver = await agent(deliverPrompt(task, deliverOutcome, { ...softCtx, reason: (prevFail && prevFail.issues) ? JSON.stringify(prevFail.issues).slice(0, 300) : (softCtx.reason || '') }), {
    label: `${task.id}-deliver`, phase: `任务:${task.id || task.name}`, schema: DELIVER_SCHEMA,
  })
  log(`${tLabel} 收尾：outcome=${deliver && deliver.outcome} pr=${(deliver && deliver.prUrl) || '-'} backToMainClean=${deliver && deliver.backToMainClean}`)

  results.push({
    id: task.id,
    issue: task.issue,
    rounds: finalRound,
    innerOutcome,
    outcome: (deliver && deliver.outcome) || deliverOutcome,
    prUrl: (deliver && deliver.prUrl) || '',
    committed: deliver && deliver.committed,
    pushed: deliver && deliver.pushed,
    backToMainClean: deliver && deliver.backToMainClean,
  })

  // 收尾没回干净 main → 警示（下一单的分支 agent 也会再核一次）
  if (deliver && deliver.backToMainClean !== true) {
    log(`${tLabel} ⚠ 收尾未确认回到干净 main，下一单分支 agent 会再核`)
  }
}

// ════════════════ 收口汇总 ════════════════
phase('收口')
const opened = results.filter(r => r.outcome === 'pr-opened')
const noChange = results.filter(r => r.outcome === 'closed-no-change')
const human = results.filter(r => r.outcome === 'left-for-human')
const blocked = results.filter(r => r.outcome === 'blocked')
log(`${batchId} 批量完成：✅PR ${opened.length} · 🟰无需改动 ${noChange.length} · ✋待人工 ${human.length} · ⛔阻塞 ${blocked.length}（共 ${results.length}）`)

return {
  batchId,
  push: allowPush,
  results,
  summary: {
    prOpened: opened.map(r => ({ id: r.id, prUrl: r.prUrl })),
    closedNoChange: noChange.map(r => r.id),
    leftForHuman: human.map(r => r.id),
    blocked: blocked.map(r => r.id),
  },
  note: '交回主会话做收口自核：抽样重跑 build/test/typecheck、校迁移号、抽验关键自判项证据、按"UI 打到最终呈现层"过一眼要起栈的单；确认无误后通知用户人工验收（PR 待 review，blocked/待人工单需跟进）。',
}
