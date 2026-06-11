import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';
import { waitForPageLoad } from '../helpers/navigation';

/**
 * 设备管理域冒烟（真实后端）：只验证页面渲染不崩 + 关键骨架元素，
 * 不做深度交互（深度用例归 mock 套件）。
 *
 * 选择器依据：
 *   - /device/list    → src/pages/device/DeviceList/index.tsx
 *                       （FilterBar=.ant-form、StatisticsPanel 标签
 *                        device.count.total=总数/Total、common.hasAlarm=有告警/Has Alarm、
 *                        DataTable=.ant-table）
 *   - /device/group   → src/pages/device/DeviceGrouping/（GroupTreePanel 标题
 *                       nav.device.group=设备分组/Device Groups，树根节点 common.all=全部/All，
 *                       右侧 DeviceListPanel 的 .ant-table）
 *   - /device/recycle → src/pages/device/RecycleBin/index.tsx（FilterBar 占位
 *                       recycle.searchPlaceholder=基站编码 / MAC地址 / Serial Number / MAC Address，
 *                       common.import=导入/Import 按钮，.ant-table）
 *   - /device/detail/<SN> → src/pages/device/DeviceDetail/index.tsx（头部 Card：
 *                       common.back=返回/Back + SN 文本；Tabs：common.detail=详情/Detail、
 *                       device.parameterTree=参数树/Parameter Tree）
 *   - /device/plug-and-play → src/pages/device/PlugAndPlay/index.tsx（F09 开站入口；
 *                       页头 provision.plugAndPlay=即插即用/Plug and Play + common.add=新增/Add，
 *                       区块标题 provision.policyList=策略列表/Policy List、
 *                       provision.executeStatus=执行状态/Execution Status，双 .ant-table）
 *   - /device/abnormal-reboot → src/pages/device/AbnormalReboot/index.tsx（重启记录统一列表，
 *                       事件日志/异常重启已合并为单列表无 Tabs；标题
 *                       page.rebootRecords.title=重启记录/Reboot Records，FilterBar 占位
 *                       log.exception.devCodeNameIp=请输入设备编码\/名称\/IP / Enter device code\/name\/IP，
 *                       工具栏 log.event.statistics=统计/Statistics + log.export=导出/Export；
 *                       旧路径 /log/exception 由路由 Navigate 跳转到此）
 *   - /device/monitor → src/pages/device/OnlineMonitoring/index.tsx（卡片网格，非表格！
 *                       页头 Title nav.device.monitor=在线监控/Online Monitor + WifiOutlined；
 *                       筛选区：search Input 占位 common.placeholder=请输入/Please enter、
 *                       刷新按钮 common.refresh=刷新/Refresh、合计 table.total=合计/Total；
 *                       设备以 Card 网格渲染，空数据落 common.noData=暂无数据/No Data，不断言 .ant-table）
 *   - /device/ne → src/pages/device/NEManagement/index.tsx（ListPageLayout 网元列表，
 *                       标题 nav.device.ne=网元管理/NE Management，extra 刷新 common.refresh +
 *                       新增 common.add=新增/Add；FilterBar=.ant-form + DataTable=.ant-table）
 *
 * 真实栈数据稀疏：不断言任何具体业务数据，列表空表也有表头（.ant-table 可见即可）。
 */
test.describe('设备管理冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/device/list 设备列表：统计面板 + 筛选栏 + 表格骨架', async ({ page }) => {
    await expectPageRenders(page, '/device/list');

    // StatisticsPanel：总数 / 有告警 两个标签（在线/离线文案与表格行内容重复，不取）
    await expect(page.getByText(/总数|Total/).first()).toBeVisible();
    await expect(page.getByText(/有告警|Has Alarm/).first()).toBeVisible();

    // FilterBar 筛选表单 + 查询按钮（icon-only，accessible name 被图标 aria-label
    // 覆盖为 "search"，故按 title 属性定位）+ 重置链接
    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(
      page.locator('button[title="查询"]').or(page.locator('button[title="Query"]')).first(),
    ).toBeVisible();
    await expect(page.getByText(/重置|Reset/).first()).toBeVisible();

    // 表格骨架（真实后端数据可空，空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/device/group 设备分组：分组树 + 设备表格', async ({ page }) => {
    await expectPageRenders(page, '/device/group');

    // 左侧 GroupTreePanel：面板标题 + Tree 根节点「全部」
    await expect(page.getByText(/设备分组|Device Groups/).first()).toBeVisible();
    const tree = page.locator('.ant-tree').first();
    await expect(tree).toBeVisible();
    await expect(tree.getByText(/全部|All/).first()).toBeVisible();

    // 右侧 DeviceListPanel：设备表格骨架
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/device/recycle 回收站：筛选栏 + 导入按钮 + 表格骨架', async ({ page }) => {
    await expectPageRenders(page, '/device/recycle');

    // FilterBar：基站编码 / MAC 地址搜索框
    await expect(
      page.getByPlaceholder(/基站编码|Serial Number/).first(),
    ).toBeVisible();

    // 工具栏「导入」按钮（与筛选同行，靠右）
    await expect(page.getByRole('button', { name: /导入|Import/ }).first()).toBeVisible();

    // 回收站表格骨架（空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/device/plug-and-play 即插即用：策略列表 + 执行状态 + 新增按钮', async ({ page }) => {
    await expectPageRenders(page, '/device/plug-and-play');

    // 页头「新增」按钮（ListPageLayout extra，新建策略入口）
    await expect(page.getByRole('button', { name: /新增|Add/ }).first()).toBeVisible();

    // 上下两个区块标题：策略列表 / 执行状态
    const main = page.locator('main');
    await expect(main.getByText(/策略列表|Policy List/).first()).toBeVisible();
    await expect(main.getByText(/执行状态|Execution Status/).first()).toBeVisible();

    // 策略表 + 任务表 双表格骨架
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/device/abnormal-reboot 重启记录：筛选栏 + 统计/导出按钮 + 旧路径跳转', async ({ page }) => {
    await expectPageRenders(page, '/device/abnormal-reboot');

    // FilterBar：设备编码/名称/IP 搜索框
    await expect(
      page.getByPlaceholder(/请输入设备编码|Enter device code/).first(),
    ).toBeVisible();

    // 工具栏：统计（按设备聚合弹窗入口）+ 导出 按钮
    await expect(page.getByRole('button', { name: /统计|Statistics/ }).first()).toBeVisible();
    await expect(page.getByRole('button', { name: /导出|Export/ }).first()).toBeVisible();

    // 重启记录表格骨架（真实后端数据可空，空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();

    // 旧路径 /log/exception 兼容跳转（路由级 Navigate replace）
    await page.goto('/log/exception');
    await expect(page).toHaveURL(/\/device\/abnormal-reboot/);
  });

  test('/device/detail/<SN> 设备详情：从列表第一行进入，头部与 Tabs 渲染', async ({ page }) => {
    await expectPageRenders(page, '/device/list');

    // 取列表第一行；真实后端无设备时跳过（冒烟不造数据）
    const firstRow = page.locator('.ant-table-tbody tr.ant-table-row').first();
    const hasData = await firstRow.isVisible({ timeout: 10_000 }).catch(() => false);
    test.skip(!hasData, '真实后端设备列表为空，跳过详情页冒烟');

    // 行内第一个 <a> 即 SN 列的 Typography.Link（SN 列 fixed left，位于 IP 链接之前）
    const snLink = firstRow.locator('a').first();
    const sn = (await snLink.innerText()).trim();
    await snLink.click();

    await expect(page).toHaveURL(/\/device\/detail\//);
    // 详情页聚合查询（设备 + 参数 + 告警）较慢，放宽等待
    await waitForPageLoad(page, 30_000);

    // 未落入"无数据"降级分支（Alert: 暂无数据 / No Data + SN）
    await expect(page.locator('.ant-alert-error')).not.toBeVisible();

    // 头部 Card：返回按钮 + 当前设备 SN
    await expect(page.getByRole('button', { name: /返回|Back/ }).first()).toBeVisible();
    await expect(page.getByText(sn).first()).toBeVisible();

    // Tabs：详情 / 参数树 两个固定页签（quickSettings/license 按设备类型动态出现，不断言）
    await expect(page.locator('.ant-tabs').first()).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /详情|Detail/ }).first(),
    ).toBeVisible();
    await expect(
      page.locator('.ant-tabs-tab').filter({ hasText: /参数树|Parameter Tree/ }).first(),
    ).toBeVisible();
  });

  test('/device/monitor 在线监控：页头标题 + 筛选搜索框 + 刷新按钮（卡片网格非表格）', async ({ page }) => {
    await expectPageRenders(page, '/device/monitor');

    const main = page.locator('main');

    // 页头 Title「在线监控」（侧边栏菜单同名，收窄到 main 取首个）
    await expect(main.getByText(/在线监控|Online Monitor/).first()).toBeVisible();

    // 筛选区搜索框（占位「请输入」）
    await expect(
      main.getByPlaceholder(/请输入|Please enter/).first(),
    ).toBeVisible();

    // 刷新按钮（图标 + 文案）
    await expect(
      main.getByRole('button', { name: /刷新|Refresh/ }).first(),
    ).toBeVisible();

    // 合计标签（设备网格头部统计；本页是 Card 网格，不断言 .ant-table）
    await expect(main.getByText(/合计|Total/).first()).toBeVisible();
  });

  test('/device/ne 网元管理：页头标题 + 新增按钮 + 筛选栏 + 表格骨架', async ({ page }) => {
    await expectPageRenders(page, '/device/ne');

    const main = page.locator('main');

    // ListPageLayout 标题「网元管理」（侧边栏菜单同名，收窄到 main 取首个）
    await expect(main.getByText(/网元管理|NE Management/).first()).toBeVisible();

    // extra 工具栏：新增按钮
    await expect(main.getByRole('button', { name: /新增|Add/ }).first()).toBeVisible();

    // FilterBar 筛选表单 + DataTable 表格骨架（真实后端数据可空，空表也有表头）
    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  // —— smoke round 5 追加：device 域剩余固定路由（列表/表单/向导页）——

  test('/device/register 设备注册向导：标题 + Steps 步骤条 + 返回/下一步按钮', async ({ page }) => {
    await expectPageRenders(page, '/device/register');

    const main = page.locator('main');

    // ListPageLayout 标题「设备注册」（侧边栏菜单同名，收窄到 main 取首个）
    await expect(main.getByText(/设备注册|Device Registration/).first()).toBeVisible();

    // 三步向导 Steps（基础信息 / IP地址 / 提交）
    await expect(page.locator('.ant-steps').first()).toBeVisible();

    // extra 返回按钮 + 第一步「下一步」按钮
    await expect(main.getByRole('button', { name: /返回|Back/ }).first()).toBeVisible();
    await expect(main.getByRole('button', { name: /下一步|Next/ }).first()).toBeVisible();
  });

  test('/device/commission 开通管理：新增按钮 + 筛选栏 + 表格骨架', async ({ page }) => {
    await expectPageRenders(page, '/device/commission');

    const main = page.locator('main');

    // ListPageLayout extra 新增按钮（侧边栏菜单同名「开通管理」，按钮用 role 定位）
    await expect(main.getByRole('button', { name: /新增|Add/ }).first()).toBeVisible();

    // FilterBar 筛选表单 + DataTable 表格骨架（页面内置 mock 数据，空表也有表头）
    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/device/handover 交接管理：新增按钮 + 筛选栏 + 表格骨架', async ({ page }) => {
    await expectPageRenders(page, '/device/handover');

    const main = page.locator('main');

    // ListPageLayout 标题「交接管理」（侧边栏同名，收窄到 main 取首个）
    await expect(main.getByText(/交接管理|Handover Management/).first()).toBeVisible();

    // extra 新增按钮
    await expect(main.getByRole('button', { name: /新增|Add/ }).first()).toBeVisible();

    // FilterBar 筛选表单 + DataTable 表格骨架
    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/device/stats 资源统计：页头标题 + 汇总统计卡 + 分类饼图卡', async ({ page }) => {
    await expectPageRenders(page, '/device/stats');

    const main = page.locator('main');

    // 页头 Title「资源统计」（侧边栏菜单同名，收窄到 main 取首个）
    await expect(main.getByText(/资源统计|Resource Statistics/).first()).toBeVisible();

    // 汇总区四张 Statistic 卡（总数等）；本页为 Card + 图表网格，不断言 .ant-table
    await expect(page.locator('.ant-statistic').first()).toBeVisible();

    // 分类卡标题「产品类型」（饼图卡），证明图表区块渲染
    await expect(main.getByText(/产品类型|Product Type/).first()).toBeVisible();
  });

  test('/device/import 导入导出：导入卡 + 导出卡 + 上传拖拽区', async ({ page }) => {
    await expectPageRenders(page, '/device/import');

    const main = page.locator('main');

    // ListPageLayout 标题「导入导出」（侧边栏同名，收窄到 main 取首个）
    await expect(main.getByText(/导入导出|Import & Export/).first()).toBeVisible();

    // 上传拖拽区（antd Upload Dragger）
    await expect(page.locator('.ant-upload-drag').first()).toBeVisible();

    // 导出卡上的「导出」按钮
    await expect(main.getByRole('button', { name: /导出|Export/ }).first()).toBeVisible();
  });

  test('/device/plug-and-play/add 新增即插即用策略：页头标题 + 基础信息卡 + 返回/确定', async ({ page }) => {
    await expectPageRenders(page, '/device/plug-and-play/add');

    const main = page.locator('main');

    // 页头标题「新增 Policy / Add Policy」
    await expect(main.getByText(/新增\s*Policy|Add\s*Policy/).first()).toBeVisible();

    // 基础信息卡（Card 标题 common.basicInfo）
    await expect(main.getByText(/基础信息|Basic Info/).first()).toBeVisible();

    // 表单骨架 + 页头「取消」按钮（非查看模式才渲染 取消/确认 一组；
    // antd 对纯两字 CJK 按钮会插入字间空格，故 "取 消" 用 \s* 容错）
    await expect(page.locator('.ant-form').first()).toBeVisible();
    await expect(main.getByRole('button', { name: /取\s*消|Cancel/ }).first()).toBeVisible();
  });
});
