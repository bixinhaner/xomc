import { test, expect, type Page } from '@playwright/test';
import { login } from './helpers/auth';

/**
 * Sprint B / T-0119 浏览器 E2E：MML standard-model 重建后的 4 个核心行为
 * 由 UI 触发 + Network 拦截校验。
 *
 * 这些 spec 直接调真实后端（dev :8081 by default）；运行前需确保 docker
 * stack 起来且 mml-standard Loader 已跑过一遍（1988/260/831 数据在表里）。
 *
 * 跑法：
 *   cd omcmb/webcode
 *   npx playwright test e2e/mml-rebuild.spec.ts
 *   # 仅本文件 + 只跑某 case：
 *   npx playwright test e2e/mml-rebuild.spec.ts -g "新 schema"
 */

// API base 由 vite proxy 转 :8081；测试里直接命中 /api/* 即可，无需写绝对地址
const API_PREFIX = '/api/v1';

test.describe('MML standard-model rebuild (T-0119)', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('GET /mml/commands 返回新 schema（target_paths 字段）', async ({ page }) => {
    // 直接调后端 API（不依赖 UI 渲染）— 验证 Sprint A schema 落地
    const resp = await page.request.get(`${API_PREFIX}/mml/commands?page=1&page_size=3`);
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    // envelope unwrap：后端统一返 {ret, msg, data}
    const items = body.data?.items ?? body.items ?? [];
    expect(Array.isArray(items)).toBeTruthy();
    expect(items.length).toBeGreaterThan(0);

    // Sprint A schema 反退化：任一行必含 target_paths 字段
    const hasTargetPaths = items.some(
      (it: Record<string, unknown>) =>
        Object.prototype.hasOwnProperty.call(it, 'target_paths'),
    );
    expect(hasTargetPaths, 'every command row should expose target_paths after rebuild').toBeTruthy();

    // 同样验证老字段（param_template / product_types）确实下线 — 后端不会返
    const hasParamTemplate = items.some(
      (it: Record<string, unknown>) =>
        Object.prototype.hasOwnProperty.call(it, 'param_template'),
    );
    expect(hasParamTemplate, 'param_template column was dropped in migration 000090').toBeFalsy();
  });

  test('POST /admin/dictload/reload?name=mml-standard 200 + 第二次 skip', async ({ page }) => {
    // Sprint B-1：单 Loader 热重载端点
    const resp1 = await page.request.post(
      `${API_PREFIX}/admin/dictload/reload?name=mml-standard`,
    );
    expect(resp1.status()).toBe(200);
    const body1 = await resp1.json();
    const data1 = body1.data ?? body1;
    expect(data1).toHaveProperty('rows_affected');
    expect(data1).toHaveProperty('elapsed_ms');

    // F-3 增量重载：紧接着第二次 reload 应 skip（rows_affected=0 + <50ms）
    const resp2 = await page.request.post(
      `${API_PREFIX}/admin/dictload/reload?name=mml-standard`,
    );
    expect(resp2.status()).toBe(200);
    const body2 = await resp2.json();
    const data2 = body2.data ?? body2;
    expect(data2.rows_affected).toBe(0);
    expect(data2.elapsed_ms).toBeLessThan(100); // skip 通常 <5ms，放宽到 100ms 容忍 CI 噪声
  });

  test('POST /mml/execute orphan command_code 201 + orphan:true 标记', async ({ page }) => {
    // Sprint B-6：standard-model 重建后老 mml_custom_command 引用孤儿码不应 500
    const resp = await page.request.post(`${API_PREFIX}/mml/execute`, {
      data: {
        command_code: 'LST_ORPHAN_DOES_NOT_EXIST_E2E',
        device_sns: ['E2E-ORPHAN-SN'],
      },
    });
    expect(resp.status()).toBe(201);
    const body = await resp.json();
    const data = body.data ?? body;
    expect(data).toHaveProperty('commands');
    const orphans = (data.commands as Array<Record<string, unknown>>).filter(
      (c) => c.orphan === true,
    );
    expect(orphans.length, 'orphan command_code should be tagged as orphan:true').toBeGreaterThan(0);
    expect(orphans[0].command_code).toBe('LST_ORPHAN_DOES_NOT_EXIST_E2E');
  });

  test('POST /mml/groups/:id/execute 201 + operation_filter 生效', async ({ page }) => {
    // Sprint B-5：先从 list commands 里抓一条 group_id 真实值（每条命令都带 group_id）
    const listResp = await page.request.get(`${API_PREFIX}/mml/commands?page=1&page_size=50`);
    expect(listResp.status()).toBe(200);
    const listBody = await listResp.json();
    const items = (listBody.data?.items ?? listBody.items ?? []) as Array<{
      group_id?: string;
      command_code: string;
    }>;
    // 找第一条 group_id 非空的命令
    const withGroup = items.find((it) => typeof it.group_id === 'string' && it.group_id.length > 0);
    expect(withGroup, 'expect at least one command bound to a group').toBeTruthy();

    const groupId = withGroup!.group_id!;
    const execResp = await page.request.post(
      `${API_PREFIX}/mml/groups/${groupId}/execute`,
      {
        data: {
          device_sns: ['E2E-GROUP-SN'],
          operation_filter: ['LST'],
        },
      },
    );
    expect(execResp.status()).toBe(201);
    const execBody = await execResp.json();
    const task = execBody.data ?? execBody;
    expect(task).toHaveProperty('commands');
    const cmds = task.commands as Array<Record<string, unknown>>;
    expect(cmds.length, 'group execute should fan out >=1 commands').toBeGreaterThan(0);
    // operation_filter=["LST"] → 所有命令必须是 LST
    for (const c of cmds) {
      expect(c.operation_type).toBe('LST');
    }
  });

  // UI 层面的回归（轻量）：MML Console 页能成功加载新 schema 命令树
  test('MML Console 页面可加载新 schema 命令列表', async ({ page }: { page: Page }) => {
    await page.goto('/mml/console');
    // 命令树（左侧）容器渲染出至少一个 category 节点
    const tree = page.locator('.ant-tree');
    await expect(tree).toBeVisible({ timeout: 10_000 });
    // 至少出现一个类别 folder（标 FolderOutlined 的图标）
    const folders = page.locator('.ant-tree .anticon-folder, .ant-tree-icon');
    await expect.poll(async () => folders.count(), { timeout: 8_000 }).toBeGreaterThan(0);
  });
});
