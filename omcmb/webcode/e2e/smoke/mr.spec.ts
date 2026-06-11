import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 测量报告（MR）域冒烟：登录后各 MR 页面在真实后端下正常渲染。
 *
 * 选择器依据：
 *   - /mr/indicators     → src/pages/mr/Indicators/index.tsx
 *       标题 nav.mr.indicators（MR指标 / MR Indicators）+ 副标题 mr.indicatorsSubtitle，
 *       FilterBar(.ant-form) + DataTable(.ant-table)，列头 mr.indicatorName / mr.indicatorCode
 *   - /mr/files          → src/pages/mr/Files/index.tsx
 *       搜索框 placeholder mr.searchDeviceSn（搜索设备SN / Search device SN），
 *       批量下载按钮 bundle.batchDownload（批量下载 / Batch Download），DataTable(.ant-table)
 *   - /mr/device-mapping → src/pages/mr/DeviceMapping/index.tsx
 *       标题/副标题/列头为硬编码中文（无 i18n），FilterBar(.ant-form) + DataTable(.ant-table)
 *   - /mr/reports        → src/pages/mr/Reports/index.tsx
 *       标题 nav.mr.reports（MR报表 / MR Reports，与侧边栏菜单同名）+ 唯一副标题 mr.reportsSubtitle，
 *       FilterBar(.ant-form) + DataTable(.ant-table)，列头 mr.reportName / mr.analysisType
 *   - /mr/variables      → src/pages/mr/Variables/index.tsx
 *       标题 nav.mr.variables（变量管理 / Variable Management，与侧边栏菜单同名）+ 唯一副标题 mr.variablesSubtitle，
 *       FilterBar(.ant-form) + DataTable(.ant-table)，列头 mr.varName / mr.valueRange
 *
 * 仅断言容器 / 标题 / 表格骨架（真实栈数据稀疏，空表也有表头），不断言业务数据。
 */
test.describe('测量报告冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/mr/indicators 渲染：标题、筛选表单与指标表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/mr/indicators');

    const main = page.locator('main');
    // 页面标题 + 副标题（副标题唯一，标题与侧边栏菜单同名故收窄到 main 并取 first）
    await expect(main.getByText(/MR指标|MR Indicators/).first()).toBeVisible();
    await expect(
      main.getByText(/查看和管理MR测量指标定义|View and manage MR measurement indicator definitions/),
    ).toBeVisible();

    // FilterBar（antd Form）与 DataTable（antd Table）骨架
    await expect(main.locator('.ant-form').first()).toBeVisible();
    const table = main.locator('.ant-table').first();
    await expect(table).toBeVisible();
    // 关键列头
    await expect(table.getByText(/指标名称|Indicator Name/).first()).toBeVisible();
    await expect(table.getByText(/指标编码|Indicator Code/).first()).toBeVisible();
  });

  test('/mr/files 渲染：搜索框、批量下载按钮与设备文件表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/mr/files');

    const main = page.locator('main');
    // 顶部筛选区：设备SN 搜索框（placeholder 来自 mr.searchDeviceSn）
    await expect(
      main.getByPlaceholder(/搜索设备SN|Search device SN/).first(),
    ).toBeVisible();

    // 工具栏：批量下载按钮（bundle.batchDownload，未勾选时 disabled 但可见）
    await expect(
      main.getByRole('button', { name: /批量下载|Batch Download/ }).first(),
    ).toBeVisible();

    // 按设备聚合的文件列表表格骨架（空表也有表头）
    const table = main.locator('.ant-table').first();
    await expect(table).toBeVisible();
    await expect(table.getByText(/文件数|File Count/).first()).toBeVisible();
  });

  test('/mr/device-mapping 渲染：标题、筛选表单与映射表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/mr/device-mapping');

    const main = page.locator('main');
    // 标题/副标题为页面源码硬编码中文（与 locale 无关）
    await expect(main.getByText('设备小区映射').first()).toBeVisible();
    await expect(main.getByText('管理设备与小区的MR数据采集映射关系')).toBeVisible();

    // FilterBar 与映射列表表格骨架
    await expect(main.locator('.ant-form').first()).toBeVisible();
    const table = main.locator('.ant-table').first();
    await expect(table).toBeVisible();
    // 关键列头（硬编码中文）
    await expect(table.getByText('小区名称').first()).toBeVisible();
    await expect(table.getByText('采样间隔(分钟)').first()).toBeVisible();
  });

  test('/mr/reports 渲染：标题、筛选表单与报表表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/mr/reports');

    const main = page.locator('main');
    // 标题与侧边栏菜单同名（MR报表 / MR Reports），收窄到 main 并取 first；副标题唯一
    await expect(main.getByText(/MR报表|MR Reports/).first()).toBeVisible();
    await expect(
      main.getByText(/查看MR数据分析报告|View MR data analysis reports/),
    ).toBeVisible();

    // FilterBar（antd Form）与 DataTable（antd Table）骨架
    await expect(main.locator('.ant-form').first()).toBeVisible();
    const table = main.locator('.ant-table').first();
    await expect(table).toBeVisible();
    // 关键列头（报表名称同时是筛选项 label，故收窄到 table）
    await expect(table.getByText(/报表名称|Report Name/).first()).toBeVisible();
    await expect(table.getByText(/分析类型|Analysis Type/).first()).toBeVisible();
  });

  test('/mr/variables 渲染：标题、筛选表单与变量表格骨架可见', async ({ page }) => {
    await expectPageRenders(page, '/mr/variables');

    const main = page.locator('main');
    // 标题与侧边栏菜单同名（变量管理 / Variable Management），收窄到 main 并取 first；副标题唯一
    await expect(main.getByText(/变量管理|Variable Management/).first()).toBeVisible();
    await expect(
      main.getByText(/管理MR测量报告采集变量参数|Manage MR measurement report collection variables/),
    ).toBeVisible();

    // FilterBar（antd Form）与 DataTable（antd Table）骨架
    await expect(main.locator('.ant-form').first()).toBeVisible();
    const table = main.locator('.ant-table').first();
    await expect(table).toBeVisible();
    // 关键列头（变量名同时是筛选项 label，故收窄到 table）
    await expect(table.getByText(/变量名|Variable Name/).first()).toBeVisible();
    await expect(table.getByText(/取值范围|Value Range/).first()).toBeVisible();
  });
});
