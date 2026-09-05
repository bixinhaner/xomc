import { test, expect, type Page } from '@playwright/test';
import { randomUUID } from 'node:crypto';
import type { Assistant, AssistantRun, AssistantDefinition } from '../../../frontend-core/src/types/assistant';
const path = '/tests/assistant-preview/index.html';
const definition: AssistantDefinition = { name: '每日设备健康简报', goal: '检查当前可见设备和重要告警，区分已确认事实与待核实原因。', scope: { kind: 'visible' }, trigger: { kind: 'schedule', time: '09:00', timezone: 'Asia/Shanghai', weekdays: [1, 2, 3, 4, 5], conditions: [] }, operations: ['get.devices', 'get.alarms.active'], notify: 'findings', cooldownMinutes: 30 };
async function fixtures(page: Page) {
  const assistants: Assistant[] = []; const runs: AssistantRun[] = []; let conflict = false;
  await page.route('**/api/v1/**', async (route) => {
    const req = route.request(); const u = new URL(req.url()); const p = u.pathname; const method = req.method(); const body = req.postDataJSON() ?? {};
    let data: unknown;
    const send = (value: unknown) => route.fulfill({ contentType: 'application/json', body: JSON.stringify({ ret: 1, msg: 'ok', data: value }) });
    if (p.endsWith('/assistants/catalog')) return send({ connected: true, events: [], capabilities: [{ operationId: 'get.devices', title: '设备列表', description: '当前设备状态', path: '/api/v1/devices', deviceScoped: false }, { operationId: 'get.alarms.active', title: '当前告警', description: '当前告警列表', path: '/api/v1/alarms/active', deviceScoped: true }] });
    if (p.endsWith('/agent/assistants')) {
      if (method === 'POST') { const a: Assistant = { id: body.id, locale: body.locale, timezone: body.timezone, revision: 0, definition: null, messages: [], readiness: 'needs_input', questions: [], missingCapabilities: [], state: 'draft', publishedRevision: null, nextRunAt: null, lastRunAt: null, createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() }; assistants.push(a); data = a; } else data = assistants;
      return send(data);
    }
    const a = assistants.find((item) => p.includes(item.id));
    if (a) {
      if (p.endsWith('/messages')) {
        if (conflict) { conflict = false; return route.fulfill({ status: 409, contentType: 'application/json', body: JSON.stringify({ ret: 0, msg: 'ASSISTANT_REVISION_CONFLICT', data: null }) }); }
        a.definition = structuredClone(definition); a.revision++; a.readiness = 'ready'; a.messages.push({ role: 'user', text: body.message }, { role: 'assistant', text: '我会在工作日早上 9 点检查你有权限查看的设备和当前重要告警，只读分析，结果仅你可见。请先用真实数据试运行。' });
      } else if (p.endsWith('/draft')) { a.definition = body.definition; a.revision++; }
      else if (p.endsWith('/publish')) { a.state = 'active'; a.publishedRevision = a.revision; a.publishedDefinition = structuredClone(a.definition!); a.nextRunAt = '2026-09-07T01:00:00Z'; }
      else if (p.endsWith('/state')) { a.state = body.state; }
      else if (p.endsWith('/runs')) {
        if (method === 'POST') { const r: AssistantRun = { id: randomUUID(), assistantId: a.id, revision: a.revision, kind: body.kind, status: 'QUEUED', output: null, tools: [], attempts: 0, createdAt: new Date().toISOString(), startedAt: null, completedAt: null, readAt: null }; runs.unshift(r); return send(r); }
        for (const r of runs) if (r.status === 'QUEUED') { r.status = 'COMPLETED'; r.completedAt = new Date().toISOString(); r.output = { outcome: 'finding', title: '有 2 台设备值得优先关注', summary: '已检查当前可见范围中的设备与活动告警。两台设备存在重要告警；关联原因尚需进一步验证。', facts: [{ text: '设备 East-017 当前存在连接中断告警。', evidenceRefs: ['tool:get.alarms.active'] }, { text: '设备 East-031 当前处于离线状态。', evidenceRefs: ['tool:get.devices'] }], hypotheses: ['网络连接可能不稳定，但当前快照不足以证明故障原因。'], nextSteps: ['查看两台设备的连接状态与现场网络情况。'] }; r.tools = [{ id: 'tool1', operationId: 'get.devices', status: 'SUCCEEDED', createdAt: r.createdAt }]; }
        return send(runs.filter((r) => r.assistantId === a.id));
      }
      return send(a);
    }
    if (p.endsWith('/read')) return send(null);
    return route.fulfill({ status: 404, contentType: 'application/json', body: JSON.stringify({ ret: 0, msg: 'fixture route not found' }) });
  });
  return { assistants, runs, failNextPlan: () => { conflict = true; } };
}
async function createPlan(page: Page) {
  await page.getByRole('button', { name: '创建助手', exact: true }).first().click();
  await page.getByRole('textbox', { name: '发送需求' }).fill('工作日早上九点帮我检查设备和重要告警。');
  await page.getByRole('textbox', { name: '发送需求' }).press('Enter');
  await expect(page.getByRole('heading', { name: '每日设备健康简报', exact: true })).toBeVisible();
  // A saved title can render before the mutation finishes refreshing its queries.
  await expect(page.getByRole('button', { name: '用真实数据试运行' })).toBeEnabled();
}

test('desktop: real component creation, trial gate, publish, refresh, edit and conflict recovery', async ({ page }, info) => {
  await page.setViewportSize({ width: 1440, height: 1100 }); const store = await fixtures(page); const errors: string[] = []; page.on('pageerror', (e) => errors.push(e.message));
  await page.goto(path); await expect(page.getByRole('heading', { name: '从一件你想省心的事开始' })).toBeVisible(); await page.screenshot({ path: info.outputPath('01-desktop-empty.png'), fullPage: true });
  await createPlan(page); await expect(page.getByRole('button', { name: '确认启用', exact: true })).toBeDisabled(); await page.screenshot({ path: info.outputPath('02-desktop-builder.png'), fullPage: true });
  await page.getByRole('button', { name: '用真实数据试运行' }).click(); await expect(page.getByRole('heading', { name: '有 2 台设备值得优先关注' })).toBeVisible(); await page.screenshot({ path: info.outputPath('03-desktop-results.png'), fullPage: true });
  await page.getByRole('button', { name: '确认启用', exact: true }).first().click(); await page.getByRole('dialog').getByRole('button', { name: '确认启用', exact: true }).click(); await expect(page.getByRole('button', { name: '暂停', exact: true })).toBeVisible(); expect(store.assistants[0].publishedRevision).toBe(1);
  await page.reload(); await expect(page.getByRole('button', { name: '暂停', exact: true })).toBeVisible(); await page.getByRole('tab', { name: '配置助手' }).click();
  await page.getByRole('button', { name: '调整配置', exact: true }).click(); await page.getByRole('dialog').getByLabel('工作目标').fill('只分析当前重要告警，明确证据和待核实原因。');
  // Ant Design inserts spacing in two-character Chinese button labels.
  await page.getByRole('dialog').getByRole('button', { name: /^保\s*存$/ }).click(); await expect(page.getByText(/已发布的第 1 版仍按原方案运行/)).toBeVisible();
  store.failNextPlan(); const text = page.getByRole('textbox', { name: '发送需求' }); await text.fill('改成下午五点。'); await text.press('Enter'); await expect(text).toHaveValue('改成下午五点。'); await expect(page.getByRole('alert').filter({ hasText: /已在其他页面|版本|更新/ }).first()).toBeVisible();
  expect(errors).toEqual([]);
});

test('mobile: Chinese text, comfortable wrapping, keyboard access and no horizontal overflow', async ({ page }, info) => {
  await page.setViewportSize({ width: 390, height: 844 }); await fixtures(page); await page.goto(path); await createPlan(page);
  await expect(page.getByRole('heading', { name: '助手会这样工作' })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.screenshot({ path: info.outputPath('04-mobile-builder.png'), fullPage: true });
  await page.getByRole('button', { name: '用真实数据试运行' }).click(); await expect(page.getByRole('heading', { name: '有 2 台设备值得优先关注' })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.screenshot({ path: info.outputPath('05-mobile-results.png'), fullPage: true });
});
test('dark mode uses existing theme tokens', async ({ page }, info) => { await page.setViewportSize({ width: 1440, height: 1100 }); await fixtures(page); await page.goto(`${path}?theme=dark`); await createPlan(page); await page.screenshot({ path: info.outputPath('06-dark-builder.png'), fullPage: true }); });
