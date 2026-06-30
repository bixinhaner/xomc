export const meta = {
  name: 'dev-check-loop',
  description: '三 agent 回合制 + 无人值守多阶段：每阶段 开发→检查→运行栈 回合制，账本跨回合传话，检查 gate 住部署，存档点提交，失败即停留现场',
  phases: [
    { title: '准备' },
    { title: '阶段' },
  ],
}

// ── 参数（通过 Workflow args 传入，见 ~/Documents/notes/知识库/开发方式-三agent回合制-无人值守多阶段-设计定稿-20260604.md §8）──
// {
//   taskId, title, workdir, ledgerPath, startHead, maxRounds, deployCmd,
//   stages: [
//     { name, specDocs:[...], scope, workdir(可选,覆盖顶层), buildCmd, testCmd,
//       hardChecks:[{id,desc,expected}], softChecks:[{id,desc}], runSegments },
//     ...
//   ]
//   workdir：顶层是默认 build/test 目录；前后端混合多阶段时每阶段可用 stage.workdir 覆盖
//   （如后端阶段 omcgo + go build/test、前端阶段 omcmb/webcode + npm typecheck/vitest）
// }
// args 可能以对象或 JSON 字符串形式投递，两种都兼容
const p = (typeof args === 'string' ? JSON.parse(args) : args) || {}
const taskId = p.taskId || 'TASK'
const ledger = p.ledgerPath
const maxRounds = p.maxRounds || 4
const startHead = p.startHead || '(未提供，主会话需自行确认回退锚点)'
const deployCmd = p.deployCmd || 'bash /Users/shangyingbin/project/omc-docker/docker-run.sh'
const DEFAULT_BUILD = 'go build ./...'
const DEFAULT_TEST = 'go test ./...'
const stages = Array.isArray(p.stages) ? p.stages : []

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
  },
}

const CHECK_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['verdict', 'reRanBuild', 'reRanTest', 'blockingIssues', 'planConformance', 'acceptanceCoverage'],
  properties: {
    verdict: { type: 'string', enum: ['pass', 'fail'], description: 'pass=无阻塞问题且符合实施计划且 build/test 真过' },
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
      description: '逐条核对实现是否符合本阶段实施计划（scope 的做什么/不做什么 + spec 指定结构）',
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
      description: '基于开发本回合实际改动反推出的、固定验收清单之外的额外风险点，逐条带核对结果',
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
  },
}

const RUNVERIFY_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['deployed', 'deployVia', 'segmentsRun', 'hardResults', 'softResults'],
  properties: {
    deployed: { type: 'boolean', description: '是否成功部署（走指定一键脚本，ps/healthz 通过）' },
    deployVia: { type: 'string', description: '实际走的部署命令（护栏自证，必须是指定一键脚本）' },
    segmentsRun: { type: 'array', items: { type: 'string' }, description: '本回合实际跑了哪几段（部署/建数据/查库/DOM）' },
    hardResults: {
      type: 'array',
      description: '死判项实测结果（脚本会拿 actual 对回 hardChecks.expected 比对，你只回原始实测值，不要自己下结论）',
      items: {
        type: 'object',
        additionalProperties: false,
        required: ['id', 'actual', 'rawEvidence'],
        properties: {
          id: { type: 'string', description: '对应 hardChecks 的 id' },
          actual: { type: 'string', description: '实测值（原始数字/字符串/布尔，转成字符串回）' },
          rawEvidence: { type: 'string', description: '支撑该实测值的原始证据：库里真实那几行 / 接口真实 JSON / 命令真实输出' },
        },
      },
    },
    softResults: {
      type: 'array',
      description: '自判项判定（机器没法数的模糊视觉项，你自己判，但必须配真实原始证据）',
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

// ── 死判比对：拿运行栈 agent 回的实测值，对回本阶段断言，纯代码判等 ──
function judgeRunStack(stage, run) {
  const fails = []
  if (!run || run.deployed !== true) {
    fails.push({ kind: 'deploy', detail: `部署未成功（deployed=${run && run.deployed}，deployVia=${run && run.deployVia}）` })
  }
  const hardChecks = stage.hardChecks || []
  const hardResults = (run && run.hardResults) || []
  for (const hc of hardChecks) {
    const r = hardResults.find(x => x.id === hc.id)
    if (!r) {
      fails.push({ kind: 'hard', id: hc.id, detail: `死判项 ${hc.id}「${hc.desc}」运行栈未回实测值` })
      continue
    }
    const a = String(r.actual).trim()
    const e = String(hc.expected).trim()
    if (a !== e) {
      fails.push({ kind: 'hard', id: hc.id, detail: `死判项 ${hc.id}「${hc.desc}」实测=${a} 预期=${e}（证据：${r.rawEvidence || '无'}）` })
    }
  }
  const softChecks = stage.softChecks || []
  const softResults = (run && run.softResults) || []
  for (const sc of softChecks) {
    const r = softResults.find(x => x.id === sc.id)
    if (!r) {
      fails.push({ kind: 'soft', id: sc.id, detail: `自判项 ${sc.id}「${sc.desc}」运行栈未回判定` })
      continue
    }
    if (r.verdict !== 'pass') {
      fails.push({ kind: 'soft', id: sc.id, detail: `自判项 ${sc.id}「${sc.desc}」agent 判 fail（证据：${r.rawEvidence || '无'}）` })
    }
  }
  return { pass: fails.length === 0, fails }
}

phase('准备')
log(`${taskId} 三 agent 多阶段：${stages.length} 阶段，每阶段≤${maxRounds} 回合，账本 ${ledger}，回退锚点 ${startHead}`)

function specBlock(stage) {
  return (stage.specDocs || []).map((d, i) => `  ${i + 1}. ${d}`).join('\n') || '  (见 scope)'
}
function hardBlock(stage) {
  return (stage.hardChecks || []).map(h => `  - [死判 ${h.id}] ${h.desc}（预期：${h.expected}）`).join('\n') || '  (无)'
}
function softBlock(stage) {
  return (stage.softChecks || []).map(s => `  - [自判 ${s.id}] ${s.desc}`).join('\n') || '  (无)'
}

function devPrompt(stage, round, prevFail) {
  let fixSection
  if (prevFail && prevFail.source === 'check') {
    fixSection = `\n本回合是【修复回合 · 检查层】。上一轮检查方在 build/test/符合性层提的阻塞问题（必须逐条修掉）：\n${(prevFail.issues || []).map((b, i) => `  ${i + 1}. ${b.title} — ${b.detail}`).join('\n')}\n`
  } else if (prevFail && prevFail.source === 'runstack') {
    fixSection = `\n本回合是【修复回合 · 运行栈层】。检查层已过，但部署后实测对不上验收（落库/接口/DOM 层问题），逐条修：\n${(prevFail.issues || []).map((d, i) => `  ${i + 1}. ${d}`).join('\n')}\n`
  } else {
    fixSection = `\n本回合是【首次实现】。\n`
  }
  return `你是「开发 agent」，负责实现任务 ${taskId} 的阶段「${stage.name}」（${p.title || ''}）。工作目录：${stage.workdir || p.workdir}

【先读账本】${ledger}
这是你和检查/运行栈 agent 的共享记忆。你没有上一回合的记忆，账本里有前情——先读它。若账本尚不存在，由你创建并写入抬头（含回退锚点 startHead=${startHead}）。
${fixSection}
【权威 spec（必读，照此实现）】
${specBlock(stage)}

【本阶段范围】
${stage.scope || '(见 spec)'}

【本阶段验收（你的实现要让这些可被验证）】
死判项（部署后会被脚本按预期值比对）：
${hardBlock(stage)}
自判项（部署后由运行栈 agent 看真实证据判定）：
${softBlock(stage)}

【纪律】
- 在 ${stage.workdir || p.workdir} 主工作树直接改（不开 worktree）；只改代码与测试，**绝不 git commit**。
- 实现后必须自己跑：\`${stage.buildCmd || DEFAULT_BUILD}\` 和 \`${stage.testCmd || DEFAULT_TEST}\`，记真实 exit code / 失败行，不许写"应该能过"。
- 关键逻辑补单测，覆盖成功 + 失败路径。
- 严守 scope 边界：不碰 scope 说"不做"的范围；spec 指定的结构/约束/迁移号严格照做，不臆造。

【收尾必做】把本回合"改了啥 / build & test 真实结果 / 修掉了哪些上轮问题 / 遗留疑问"**追加**写入账本 ${ledger}（带"阶段「${stage.name}」回合${round}·开发"小标题），再返回结构化结果。`
}

function checkPrompt(stage, round, dev) {
  const changeSection = dev ? `\n【开发 agent 本回合自报（结合真实 \`git diff\` 核对，不全信自报）】\n- 摘要：${dev.summary || '(无)'}\n- 改动文件：${(dev.filesChanged || []).map(f => `\n    - ${f}`).join('') || ' (无)'}\n` : ''
  return `你是「检查 agent」，对任务 ${taskId} 阶段「${stage.name}」的当前实现做**对抗式复核**。工作目录：${stage.workdir || p.workdir}

【先读账本】${ledger}
开发 agent 刚写了本回合做了啥。读它 + 读权威 spec + 看真实代码改动（\`git diff\` / \`git status\`）。
${changeSection}
【权威 spec】
${specBlock(stage)}

【本阶段范围（scope）】
${stage.scope || '(见 spec)'}

【本阶段验收】
死判项：
${hardBlock(stage)}
自判项：
${softBlock(stage)}

【你的职责 — 默认怀疑，不信开发的口头声明】
1. **自己重跑** \`${stage.buildCmd || DEFAULT_BUILD}\` 和 \`${stage.testCmd || DEFAULT_TEST}\`，把真实输出记下来。
2. 逐条对验收核对：代码是否真覆盖？单测是否真测了成功+失败路径？有没有漏掉的 spec 要点（枚举/约束改两处只改一处、迁移 up/down 未配对）？
3. **对照实施计划核符合性**（一等职责）：逐条读 scope 的做什么/不做什么 + spec 指定结构，核对实现有没有 **偏离**（说照搬某模式却另起炉灶）/ **越界**（动了 scope 说"不做"的范围）/ **漏做**（scope 说"做"的没做）。逐条记入 \`planConformance\`；构成阻塞的升级进 \`blockingIssues\`。
4. **关联本回合真实改动反推额外风险**：对照开发自报 + 真实 diff，想清楚"这次实际动了什么 → 还有哪些没列进验收、却确实该验的点"（波及的调用方、落库丢键、漏掉的边界值）。逐条记入 \`derivedRisks\`；阻塞的升级进 \`blockingIssues\`。
5. 不确定就判 fail——宁可多一轮也不放过。只有"无阻塞问题 + 符合实施计划 + 验收覆盖 + build/test 真过"才判 pass。

【收尾必做】把"重跑 build/test 真实结果 + 阻塞清单 + 符合性核对 + 反推风险 + 验收覆盖"**追加**写入账本 ${ledger}（带"阶段「${stage.name}」回合${round}·检查"小标题），再返回结构化结果。`
}

function runVerifyPrompt(stage, round) {
  return `你是「运行栈验证 agent」，对任务 ${taskId} 阶段「${stage.name}」做**部署 + 取证**。工作目录：${stage.workdir || p.workdir}

检查 agent 已确认 build/test 层与实施计划符合性通过。现在要把系统真跑起来、取回真实证据。

【先读账本】${ledger}（了解本阶段做了什么、验收要点）

【⚠ 部署护栏 —— 你是空白上下文，以下铁律必须遵守】
- 部署**只跑这一条命令**：\`${deployCmd}\`（宿主编译 + docker 重建全栈）。
- **绝不碰** \`goomc/deployments/docker\` —— 碰错会删运行栈网络致全栈断网。
- 部署后验证：\`cd /Users/shangyingbin/project/omc-docker && docker compose ps\`（基础设施 healthy / 迁移 exited(0) / 应用 running）+ \`curl -fsS http://localhost:8081/healthz\` 与 \`curl -fsS http://localhost:7557/healthz\`。非 200 视为未达可服务态，deployed=false。

【⚠ 环境坑】
- 明文 curl 登录被禁（前端走加密），\`POST /auth/login\` 明文会被拒。
- 内部 X-API-Key 对受限接口（如 /pm/adhoc/*）无权限。
- 要端到端验受限接口/UI：走前端 :3000 + playwright（账号 admin / admin123），或直接读 DB。
- playwright 纪律：不主动截图、验完不主动关浏览器、停在最后一屏。

【取证（本阶段大致验：${stage.runSegments || '部署 + 按需查库/DOM'}）】
死判项 —— 去测出真实实测值，**只回原始数字/字符串 + 那几行原始证据，不要自己下"通过/不通过"结论**（脚本会用预期值比对）：
${hardBlock(stage)}
自判项 —— 机器没法数的，你自己判 pass/fail，但**必须配真实原始证据**（DOM 真实文字 / 接口真实 JSON），编不出就过不了：
${softBlock(stage)}

【收尾必做】把"部署结果 + 跑了哪几段 + 每个死判项的实测值与证据 + 每个自判项的判定与证据"**追加**写入账本 ${ledger}（带"阶段「${stage.name}」回合${round}·运行栈"小标题），再返回结构化结果。`
}

function archivePrompt(stage) {
  return `你是「存档点 agent」，任务 ${taskId} 阶段「${stage.name}」已通过三环节。工作目录：${stage.workdir || p.workdir}

只做一件事：在 ${stage.workdir || p.workdir} 所在 git 仓库打一个**存档点 commit**，让下一阶段看到的 \`git diff\` 干净（只含本阶段改动）。
- 跑：\`git add -A\` 然后 \`git commit -m "wip(${stage.name}): 阶段存档点"\`。
- 这**不是最终历史**，只是临时存档点；**绝不 push**，绝不 \`--no-verify\`，绝不 \`--force\`。
- 若没有可提交改动，如实返回 committed=false。

返回结构化结果即可。`
}

const ARCHIVE_SCHEMA = {
  type: 'object',
  additionalProperties: false,
  required: ['committed'],
  properties: {
    committed: { type: 'boolean' },
    commitHash: { type: 'string' },
    note: { type: 'string' },
  },
}

if (!stages.length) {
  log('未提供 stages，无事可做。请按设计定稿 §8 备好多阶段输入。')
  return { taskId, stopped: true, reason: 'no-stages' }
}

const stageSummaries = []

for (let si = 0; si < stages.length; si++) {
  const stage = stages[si]
  const stageLabel = `阶段${si + 1}/${stages.length}「${stage.name}」`
  phase(`阶段:${stage.name}`)
  log(`${stageLabel} 开始`)

  let prevFail = null
  let stagePassed = false
  let finalRound = 0
  let lastCheck = null
  let lastRun = null

  for (let round = 1; round <= maxRounds; round++) {
    finalRound = round

    const dev = await agent(devPrompt(stage, round, prevFail), {
      label: `${taskId}-${stage.name}-dev-r${round}`, phase: `阶段:${stage.name}`, schema: DEV_SCHEMA,
    })
    log(`${stageLabel} 回合${round}·开发：build=${dev && dev.buildPassed} test=${dev && dev.testPassed}｜${(dev && dev.summary) || ''}`)

    const check = await agent(checkPrompt(stage, round, dev), {
      label: `${taskId}-${stage.name}-check-r${round}`, phase: `阶段:${stage.name}`, schema: CHECK_SCHEMA,
    })
    lastCheck = check
    const nBlock = (check && check.blockingIssues && check.blockingIssues.length) || 0
    log(`${stageLabel} 回合${round}·检查：verdict=${check && check.verdict}，阻塞 ${nBlock} 条`)

    if (!check || check.verdict !== 'pass') {
      prevFail = { source: 'check', issues: (check && check.blockingIssues) || [] }
      continue
    }

    // 检查 gate 通过才部署
    const run = await agent(runVerifyPrompt(stage, round), {
      label: `${taskId}-${stage.name}-run-r${round}`, phase: `阶段:${stage.name}`, schema: RUNVERIFY_SCHEMA,
    })
    lastRun = run
    const judged = judgeRunStack(stage, run)
    log(`${stageLabel} 回合${round}·运行栈：deployed=${run && run.deployed}，死判 ${judged.pass ? 'PASS' : 'FAIL(' + judged.fails.length + ')'}`)

    if (!judged.pass) {
      prevFail = { source: 'runstack', issues: judged.fails.map(f => f.detail) }
      continue
    }

    // 三环节全过 → 存档点
    const arch = await agent(archivePrompt(stage), {
      label: `${taskId}-${stage.name}-archive`, phase: `阶段:${stage.name}`, schema: ARCHIVE_SCHEMA,
    })
    log(`${stageLabel} 存档点：committed=${arch && arch.committed} ${(arch && arch.commitHash) || ''}`)
    stagePassed = true
    break
  }

  stageSummaries.push({ name: stage.name, passed: stagePassed, rounds: finalRound })

  if (!stagePassed) {
    log(`${stageLabel} 到 ${maxRounds} 回合仍未过 → 整条停，留现场`)
    return {
      taskId,
      startHead,
      stopped: true,
      failedStage: stage.name,
      failedAtIndex: si,
      rounds: finalRound,
      lastCheck,
      lastRun,
      stageSummaries,
      note: '失败即停：不回滚上阶段、不跳过。账本已记每回合，工作树留未提交实现，浏览器停最后一屏。主会话据此通知用户。',
    }
  }
}

return {
  taskId,
  startHead,
  passed: true,
  stageSummaries,
  note: '全部阶段通过。交回主会话：git reset --soft <startHead>（改动全留）→ 合成一个功能提交配好 message → 只 commit 不 push。',
}
