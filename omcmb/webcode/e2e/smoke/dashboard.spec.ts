import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 仪表盘域冒烟：登录后 /dashboard 正常渲染，关键卡片可见。
 *
 * 选择器依据 src/pages/dashboard/index.tsx + i18n（默认 zh-CN，正则兼容 en-US）：
 *   - KPI 卡片标题：dashboard.totalDevices（总设备数 / Total Devices）等
 *   - 图表卡片标题：dashboard.deviceStatusByType / dashboard.alarmLevelStatistics
 *     （图表内容可能因后端无数据而显示 EmptyState，所以只断言卡片容器与标题）
 */
test.describe('仪表盘冒烟（真实后端）', { tag: '@smoke' }, () => {
  test('登录后 /dashboard 渲染，关键统计卡片与图表容器可见', async ({ page }) => {
    await smokeLogin(page);
    await expectPageRenders(page, '/dashboard');

    // Row 1：四张 KPI 统计卡片标题
    await expect(page.getByText(/总设备数|Total Devices/).first()).toBeVisible();
    await expect(page.getByText(/在线设备|Online Devices/).first()).toBeVisible();
    await expect(page.getByText(/活跃告警|Active Alarms/).first()).toBeVisible();

    // Row 3：两个图表 Card（标题来自 dashboard.deviceStatusByType / alarmLevelStatistics）
    await expect(
      page.locator('.ant-card-head-title').filter({ hasText: /设备状态分布|Device Status/ }),
    ).toBeVisible();
    await expect(
      page.locator('.ant-card-head-title').filter({ hasText: /告警级别统计|Alarm Level/ }),
    ).toBeVisible();

    // Row 4：快速入口卡片
    await expect(
      page.locator('.ant-card-head-title').filter({ hasText: /快速入口|Quick Access/ }),
    ).toBeVisible();
  });
});
