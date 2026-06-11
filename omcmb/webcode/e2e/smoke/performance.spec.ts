import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 性能管理域冒烟（真实后端）：四个页面渲染不崩 + 关键骨架元素可见。
 *
 * 选择器依据：
 *   - /performance          → pages/performance/PmDashboard/PerformanceLayout.tsx
 *       左侧任务列表 Card 标题 = perf.dashboard.taskListTitle（任务 / Tasks）；
 *       右侧出图区数据依赖后端聚合任务，不断言具体图表内容。
 *   - /performance/query    → pages/performance/KPIQuery/index.tsx
 *       侧栏标题「查询模板」、公共/私有 Tabs、主区 Card「查询条件」「查询结果」
 *       （该页文案为硬编码中文，无 en-US 变体）。
 *   - /performance/kpi-standard → pages/performance/KPIStandardReport/index.tsx
 *       设备类型 Radio 页签 kpi.tree.enbSet（eNB指标集 / eNB Indicator Set）、
 *       perf.functionSet（指标功能集 / Function Set）、指标 DataTable（.ant-table）。
 *   - /performance/files    → pages/performance/PerformanceFiles/index.tsx
 *       页面标题 nav.performance.files（性能文件 / Performance Files）、
 *       FilterBar（.ant-form）+ DataTable（.ant-table）。
 *   - /performance/pm-adhoc → pages/performance/PmAdhoc/index.tsx
 *       两张 Card：perf.adhoc.cardBuiltin（内置聚合任务 / Built-in Aggregation Tasks）
 *       + perf.adhoc.cardCustom（自建聚合任务 / Custom Aggregation Tasks）、
 *       自建区「新建任务 / New Task」按钮（perf.adhoc.btnNewTask）、任务 Table 骨架。
 *   - /performance/pm-adhoc/new → pages/performance/PmAdhoc/PmAdhocWizard.tsx
 *       整页 5 步向导外壳 Card 标题 perf.adhoc.wizardTitle
 *       （新建自定义聚合任务 / New Custom Aggregation Task）、横向 Steps（.ant-steps）、
 *       第1步基本信息：任务名输入 + 「下一步 / Next」按钮（perf.adhoc.btnNext）。
 *   - /performance/device-view → pages/performance/PmDashboard/DeviceListPane.tsx
 *       条件区 Form：制式 Segmented（LTE/NR/GSM）+「出图 / Plot」按钮
 *       （perf.dashboard.btnPlot）；未出图时空态文案 perf.dashboard.emptyPickConditions
 *       （选择制式…/ Select technology…）——初始 UI 状态，非业务数据。
 *   - /performance/threshold → pages/performance/ThresholdConfig/index.tsx
 *       ListPageLayout 标题 nav.performance.threshold（门限配置 / Threshold Config，<h4>）、
 *       右上「新增 / Add」按钮（common.add）、门限 DataTable（.ant-table）骨架。
 *   - /performance/kpi-station → pages/performance/KPIStationReport/index.tsx
 *       ListPageLayout 标题 nav.performance.kpiStation（测量任务管理 / Measurement Task
 *       Management，<h4>）、FilterBar（.ant-form）+ 测量维护 DataTable（.ant-table）骨架。
 *   - /performance/charts → pages/performance/PerformanceCharts/index.tsx
 *       ListPageLayout 标题 nav.performance.charts（性能图表 / Performance Charts，<h4>）、
 *       FilterBar（.ant-form）+ KPI 快选按钮 kpi.rrcSetupSuccessRate（RRC建立成功率 /
 *       RRC Setup Success Rate）。
 *   - /performance/task-config → pages/performance/PerformanceTaskConfig/index.tsx
 *       ListPageLayout 标题 nav.performance.taskConfig（任务配置 / Task Config，<h4>）、
 *       右上「新增 / Add」按钮（common.add）、采集任务 DataTable（.ant-table）骨架。
 *
 * 注：threshold/charts/kpi-station/task-config 四项菜单在 navConfig.ts 已注释隐藏，
 * 仅保留直链路由；页面标题 <h4> 不与侧边栏菜单项撞文案，但仍统一收窄到 main 取 .first()。
 */
test.describe('性能管理冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/performance PM 仪表盘渲染，任务列表卡片可见', async ({ page }) => {
    await expectPageRenders(page, '/performance');

    // 左侧 240px 任务列表 Card（标题 = perf.dashboard.taskListTitle）
    await expect(
      page.locator('.ant-card-head-title').filter({ hasText: /^(任务|Tasks)$/ }).first(),
    ).toBeVisible();

    // 右侧出图区：选中任务后是图表面板，无任务则是空态卡片——两态都至少有一张 Card
    await expect(page.locator('main .ant-card').first()).toBeVisible();
  });

  test('/performance/query 指标查询渲染，模板侧栏与查询条件/结果卡片可见', async ({ page }) => {
    await expectPageRenders(page, '/performance/query');

    // 模板侧栏标题 + 公共/私有 Tabs（页面硬编码中文）
    await expect(page.getByText('查询模板').first()).toBeVisible();
    await expect(page.locator('.ant-tabs').first()).toBeVisible();

    // 主区两张 Card：查询条件（含表单）/ 查询结果
    await expect(
      page.locator('.ant-card-head-title').filter({ hasText: '查询条件' }),
    ).toBeVisible();
    await expect(
      page.locator('.ant-card-head-title').filter({ hasText: '查询结果' }),
    ).toBeVisible();
    await expect(page.locator('main .ant-form').first()).toBeVisible();
  });

  test('/performance/kpi-standard 指标库渲染，设备页签/功能集树/指标表格可见', async ({ page }) => {
    await expectPageRenders(page, '/performance/kpi-standard');

    // 左侧树面板：设备类型 Radio 页签 + 「指标功能集」标题
    await expect(page.getByText(/eNB指标集|eNB Indicator Set/).first()).toBeVisible();
    await expect(page.getByText(/指标功能集|Function Set/).first()).toBeVisible();

    // 指标 DataTable 骨架（空表也有表头）
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/performance/files 性能文件渲染，过滤栏与文件表格可见', async ({ page }) => {
    await expectPageRenders(page, '/performance/files');

    // 页面标题（nav.performance.files）
    await expect(page.getByText(/性能文件|Performance Files/).first()).toBeVisible();

    // FilterBar（Form）+ DataTable 骨架
    await expect(page.locator('main .ant-form').first()).toBeVisible();
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/performance/pm-adhoc 自定义聚合渲染，内置/自建任务卡片与新建按钮可见', async ({ page }) => {
    await expectPageRenders(page, '/performance/pm-adhoc');

    // 两张分区 Card 标题（perf.adhoc.cardBuiltin / cardCustom）
    await expect(
      page
        .locator('.ant-card-head-title')
        .filter({ hasText: /内置聚合任务|Built-in Aggregation Tasks/ }),
    ).toBeVisible();
    await expect(
      page
        .locator('.ant-card-head-title')
        .filter({ hasText: /自建聚合任务|Custom Aggregation Tasks/ }),
    ).toBeVisible();

    // 自建区「新建任务」按钮（perf.adhoc.btnNewTask）+ 任务表骨架（空表也有表头）
    await expect(
      page.locator('main').getByRole('button', { name: /新建任务|New Task/ }),
    ).toBeVisible();
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/performance/pm-adhoc/new 新建向导渲染，向导标题/步骤条/下一步按钮可见', async ({ page }) => {
    await expectPageRenders(page, '/performance/pm-adhoc/new');

    // 整页向导外壳 Card 标题（perf.adhoc.wizardTitle）
    await expect(
      page
        .locator('.ant-card-head-title')
        .filter({ hasText: /新建自定义聚合任务|New Custom Aggregation Task/ }),
    ).toBeVisible();

    // 横向 5 步 Steps 步骤条骨架
    await expect(page.locator('main .ant-steps').first()).toBeVisible();

    // 「下一步」按钮（perf.adhoc.btnNext，向导底部导航）
    await expect(
      page.locator('main').getByRole('button', { name: /^(下一步|Next)$/ }),
    ).toBeVisible();
  });

  test('/performance/device-view 设备性能查看渲染，制式切换/出图按钮/空态提示可见', async ({ page }) => {
    await expectPageRenders(page, '/performance/device-view');

    // 条件区：制式 Segmented（LTE/NR/GSM，perf.dashboard.fieldTech）
    await expect(page.locator('main .ant-segmented').first()).toBeVisible();
    await expect(
      page.locator('main .ant-segmented-item').filter({ hasText: 'LTE' }).first(),
    ).toBeVisible();

    // 「出图」按钮（perf.dashboard.btnPlot）
    await expect(
      page.locator('main').getByRole('button', { name: /出图|Plot/ }).first(),
    ).toBeVisible();

    // 初始未出图：空态引导文案（perf.dashboard.emptyPickConditions，UI 状态非业务数据）
    await expect(
      page.locator('main').getByText(/选择制式|Select technology/).first(),
    ).toBeVisible();
  });

  test('/performance/threshold 门限配置渲染，标题/新增按钮/门限表格可见', async ({ page }) => {
    await expectPageRenders(page, '/performance/threshold');

    // ListPageLayout 页面标题 <h4>（nav.performance.threshold）
    await expect(
      page.locator('main').getByRole('heading', { name: /门限配置|Threshold Config/ }).first(),
    ).toBeVisible();

    // 右上「新增」按钮（common.add；PlusOutlined 图标使无障碍名为「plus 新增」，故不锚定）
    await expect(
      page.locator('main').getByRole('button', { name: /新增|Add/ }).first(),
    ).toBeVisible();

    // 门限 DataTable 骨架（空表也有表头）
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/performance/kpi-station 测量任务管理渲染，标题/过滤栏/测量表格可见', async ({ page }) => {
    await expectPageRenders(page, '/performance/kpi-station');

    // ListPageLayout 页面标题 <h4>（nav.performance.kpiStation）
    await expect(
      page
        .locator('main')
        .getByRole('heading', { name: /测量任务管理|Measurement Task Management/ })
        .first(),
    ).toBeVisible();

    // FilterBar（Form）+ 测量维护 DataTable 骨架
    await expect(page.locator('main .ant-form').first()).toBeVisible();
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });

  test('/performance/charts 性能图表渲染，标题/过滤栏/KPI 快选按钮可见', async ({ page }) => {
    await expectPageRenders(page, '/performance/charts');

    // ListPageLayout 页面标题 <h4>（nav.performance.charts）
    await expect(
      page.locator('main').getByRole('heading', { name: /性能图表|Performance Charts/ }).first(),
    ).toBeVisible();

    // FilterBar（Form）骨架
    await expect(page.locator('main .ant-form').first()).toBeVisible();

    // KPI 快选按钮区：RRC建立成功率（kpi.rrcSetupSuccessRate）按钮
    await expect(
      page
        .locator('main')
        .getByRole('button', { name: /RRC建立成功率|RRC Setup Success Rate/ })
        .first(),
    ).toBeVisible();
  });

  test('/performance/task-config 采集任务配置渲染，标题/新增按钮/任务表格可见', async ({ page }) => {
    await expectPageRenders(page, '/performance/task-config');

    // ListPageLayout 页面标题 <h4>（nav.performance.taskConfig）
    await expect(
      page.locator('main').getByRole('heading', { name: /任务配置|Task Config/ }).first(),
    ).toBeVisible();

    // 右上「新增」按钮（common.add；PlusOutlined 图标使无障碍名为「plus 新增」，故不锚定）
    await expect(
      page.locator('main').getByRole('button', { name: /新增|Add/ }).first(),
    ).toBeVisible();

    // 采集任务 DataTable 骨架（空表也有表头）
    await expect(page.locator('main .ant-table').first()).toBeVisible();
  });
});
