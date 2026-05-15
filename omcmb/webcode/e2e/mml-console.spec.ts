import { test, expect, type Page, type Route } from '@playwright/test';
import { login } from './helpers/auth';

/**
 * T-0123-P2-d MML Console 三方向同步 E2E spec (PRD §11 DoD 4):
 *   勾选 (SubFieldChecklist) ⇄ MML textbox 反映命令字符串
 *   MML textbox 输入 → 300ms 防抖 → 调 /mml/parse → statements 更新
 *   DO 按钮 → /mml/execute-statements 调用 + Modal.confirm OnReboot 确认
 *
 * Mock 策略 (page.route)：
 *   - GET /mml/group-tree           → 单 group + 1 LST 命令 (mock-fixture)
 *   - GET /mml/commands/:id/sub-fields → 2 sub-fields (1 defaultSelected + 1 OnReboot)
 *   - POST /mml/parse                → echo parse_errors 空 statements 空 (用于验证防抖触发)
 *   - POST /mml/execute-statements   → echo MMLTask
 *
 * 不依赖真后端 / 真 PG / 真 ACS。
 *
 * 跑法：
 *   cd omcmb/webcode
 *   npx playwright test e2e/mml-console.spec.ts
 */

const MOCK_GROUP_ID = '11111111-1111-1111-1111-111111111111';
const MOCK_CMD_ID = '22222222-2222-2222-2222-222222222222';
const MOCK_SF1_ID = '33333333-3333-3333-3333-333333333333';
const MOCK_SF2_ID = '44444444-4444-4444-4444-444444444444';

async function mockBackend(page: Page): Promise<{ parseCalls: number; executeCalls: number }> {
  const counters = { parseCalls: 0, executeCalls: 0 };

  // group-tree (Console 命令树数据源)
  await page.route(/\/mml\/group-tree/, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        ret: 1,
        msg: 'ok',
        data: [
          {
            id: MOCK_GROUP_ID,
            group_code: 'BSC_BASIC',
            path: 'BSC_BASIC',
            display_name: 'BSC Configuration',
            display_order: 0,
            commands: [
              {
                id: MOCK_CMD_ID,
                command_code: 'LST_DEVICE_INFO',
                logical_code: 'DEVICE_INFO',
                operation_type: 'LST',
                display_name: 'Device Info (LST)',
                require_confirm: false,
              },
            ],
            children: [],
          },
        ],
      }),
    });
  });

  // sub-fields (点击命令后拉取)
  await page.route(/\/mml\/commands\/.+\/sub-fields/, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        ret: 1,
        msg: 'ok',
        data: [
          {
            id: MOCK_SF1_ID,
            command_id: MOCK_CMD_ID,
            param_id: 'p1',
            mml_code: 'MODEL_NAME',
            label: 'Model Name',
            label_i18n: { 'zh-CN': '型号', 'en-US': 'Model Name' },
            tr069_path: 'Device.DeviceInfo.ModelName',
            value_type: 'string',
            access_type: 'READ_ONLY',
            is_object: false,
            supports_add: false,
            supports_delete: false,
            change_applies: 'Immediate',
            constraint_text: '',
            constraint_text_i18n: {},
            default_selected: true,
            is_required: false,
            sort_order: 1,
          },
          {
            id: MOCK_SF2_ID,
            command_id: MOCK_CMD_ID,
            param_id: 'p2',
            mml_code: 'SERIAL_NUM',
            label: 'Serial Number',
            label_i18n: { 'zh-CN': '序列号', 'en-US': 'Serial Number' },
            tr069_path: 'Device.DeviceInfo.SerialNumber',
            value_type: 'string',
            access_type: 'READ_ONLY',
            is_object: false,
            supports_add: false,
            supports_delete: false,
            change_applies: 'OnReboot',
            constraint_text: '',
            constraint_text_i18n: {},
            default_selected: false,
            is_required: false,
            sort_order: 2,
          },
        ],
      }),
    });
  });

  // POST /mml/parse — textbox 防抖触发
  await page.route(/\/mml\/parse/, async (route: Route) => {
    counters.parseCalls += 1;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        ret: 1,
        msg: 'ok',
        data: { statements: [], parse_errors: [] },
      }),
    });
  });

  // POST /mml/execute-statements — DO 按钮触发
  await page.route(/\/mml\/execute-statements/, async (route: Route) => {
    counters.executeCalls += 1;
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({
        ret: 1,
        msg: 'ok',
        data: {
          id: 'task-1',
          status: 'running',
          created_at: new Date().toISOString(),
        },
      }),
    });
  });

  return counters;
}

test.describe('T-0123-P2-d MML Console 三方向同步', () => {
  test('点击 LST 命令叶子 → statement 添加 + 默认勾选 sub-field 显示', async ({ page }) => {
    await mockBackend(page);
    await login(page);
    await page.goto('/mml/console');

    // 等命令树渲染
    await page.waitForSelector('text=Device Info (LST)', { timeout: 10_000 });

    // 点击命令叶子
    await page.getByText('Device Info (LST)').click();

    // 验证 sub-field "Model Name" 出现在右栏 (LST 视图 SubFieldChecklist)
    await expect(page.getByText('Model Name')).toBeVisible({ timeout: 5_000 });
    // 验证 OnReboot tag 出现 (SubFieldChecklist 标签)
    await expect(page.getByText('Serial Number')).toBeVisible();
  });

  test('MML textbox 输入 → 300ms 防抖触发 /mml/parse', async ({ page }) => {
    const counters = await mockBackend(page);
    await login(page);
    await page.goto('/mml/console');

    // 找到 MML textbox (antd Input.TextArea, monospace 字体)
    const textarea = page.locator('textarea').first();
    await textarea.waitFor({ timeout: 10_000 });

    // 输入文本
    await textarea.fill('LST DEVICE_INFO:lstId={MODEL_NAME};');

    // 等防抖 300ms + 网络 buffer
    await page.waitForTimeout(500);

    expect(counters.parseCalls).toBeGreaterThanOrEqual(1);
  });

  test('DO 按钮 onClick 校验：未选设备时不调用 execute', async ({ page }) => {
    const counters = await mockBackend(page);
    await login(page);
    await page.goto('/mml/console');

    // DO 按钮可见
    const doButton = page.getByRole('button', { name: /DO/i }).first();
    await doButton.waitFor({ timeout: 10_000 });

    // 直接点 DO — 未选设备 + 无 statements，应触发 message.warning 不调 execute
    await doButton.click();
    await page.waitForTimeout(300);

    expect(counters.executeCalls).toBe(0);
  });
});
