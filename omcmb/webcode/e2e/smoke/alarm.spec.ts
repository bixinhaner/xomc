import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';
import { navigateTo, waitForPageLoad } from '../helpers/navigation';

/**
 * 告警管理域冒烟（真实后端）：四个页面渲染不崩 + 关键骨架元素可见。
 *
 * 选择器依据（i18n 默认 zh-CN，正则兼容 en-US）：
 *   - /alarm/current    → src/pages/alarm/CurrentAlarms（ListPageLayout title = nav.alarm.current
 *                         「当前告警 / Active Alarms」+ DataTable(.ant-table)）
 *   - /alarm/history    → src/pages/alarm/HistoricalAlarms（title = nav.alarm.history
 *                         「历史告警 / Historical Alarms」+ DataTable(.ant-table)）
 *   - /alarm/statistics → src/pages/alarm/AlarmStatistics（标题 nav.alarm.statistics +
 *                         图表 Card：alarm.stats.distribution「告警存量分布」/
 *                         alarm.stats.trend「告警变化趋势」——图表内容可能因无数据而 Empty，
 *                         只断言卡片标题）
 *   - /alarm/rules      → src/pages/alarm/AlarmRules（title = nav.alarm.rules
 *                         「告警规则 / Alarm Rules」+ DataTable(.ant-table)）
 *   - /alarm/sync       → src/pages/alarm/AlarmSync（ListPageLayout title = nav.alarm.sync
 *                         「告警同步 / Alarm Sync」+ DataTable(.ant-table)；标题与侧边栏菜单
 *                         同名，取 first）
 *   - /alarm/custom-stats → src/pages/alarm/CustomAlarmStats（TreeListPageLayout：左侧树面板
 *                         标题 alarm.customAlarmGroup「自定义告警分组 / Custom Alarm Groups」
 *                         + 右侧 DataTable(.ant-table)；树面板标题仅出现在页面主体，不撞菜单）
 *
 * 不断言具体业务数据（真实栈数据稀疏）；空表也有表头，.ant-table 可见即可。
 */
test.describe('告警管理冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/alarm/current 当前告警列表渲染', async ({ page }) => {
    await expectPageRenders(page, '/alarm/current');

    // 页面标题（侧边栏菜单同名，取 first）
    await expect(page.getByText(/当前告警|Active Alarms/).first()).toBeVisible();
    // 告警列表表格骨架（空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/alarm/history 历史告警列表渲染', async ({ page }) => {
    await expectPageRenders(page, '/alarm/history');

    await expect(page.getByText(/历史告警|Historical Alarms/).first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/alarm/statistics 告警统计图表卡片渲染', async ({ page }) => {
    // 统计页有多个聚合查询 + 自动刷新 Spin，首屏可能偏慢，放宽等待
    await navigateTo(page, '/alarm/statistics');
    await waitForPageLoad(page, 30_000);

    // 复用 expectPageRenders 的崩溃断言（此时已加载完成，等待开销很小）
    await expectPageRenders(page, '/alarm/statistics');

    // 页面标题（侧边栏菜单同名，取 first）
    await expect(page.getByText(/告警统计|Alarm Statistics/).first()).toBeVisible();

    // 图表 Card 标题（图表内容可能是 Empty/Spin 三态，只断言卡片标题）
    await expect(
      page
        .locator('.ant-card-head-title')
        .filter({ hasText: /告警存量分布|Alarm Distribution/ })
        .first(),
    ).toBeVisible();
    await expect(
      page
        .locator('.ant-card-head-title')
        .filter({ hasText: /告警变化趋势|Alarm Change Trend/ })
        .first(),
    ).toBeVisible();
  });

  test('/alarm/rules 告警规则列表渲染', async ({ page }) => {
    await expectPageRenders(page, '/alarm/rules');

    await expect(page.getByText(/告警规则|Alarm Rules/).first()).toBeVisible();
    // 副标题（alarm.rulesDesc）
    await expect(
      page.getByText(/配置告警过滤规则|Configure alarm filtering rules/).first(),
    ).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/alarm/sync 告警同步任务列表渲染', async ({ page }) => {
    await expectPageRenders(page, '/alarm/sync');

    // 页面标题（侧边栏菜单同名，取 first）
    await expect(page.getByText(/告警同步|Alarm Sync/).first()).toBeVisible();
    // 同步任务列表表格骨架（空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/alarm/custom-stats 自定义统计（树+表）渲染', async ({ page }) => {
    await expectPageRenders(page, '/alarm/custom-stats');

    // 左侧自定义告警分组树面板标题（仅页面主体出现，不撞侧边栏菜单）
    await expect(
      page.getByText(/自定义告警分组|Custom Alarm Groups/).first(),
    ).toBeVisible();
    // 右侧告警列表表格骨架（空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });
});
