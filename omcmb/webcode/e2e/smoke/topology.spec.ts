import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 拓扑管理域冒烟：三个页面在真实后端下正常渲染。
 *
 * 选择器依据（i18n 默认 zh-CN，正则兼容 en-US）：
 *   - /topology/gis-map  → src/pages/Topology/GISMapView/index.tsx
 *       左侧筛选面板：gis.filter.deviceStatus（设备状态）/ gis.filter.deviceGroup（设备组）
 *       / gis.status.onlineActive（在线激活）；右侧 OpenLayers 地图（.ol-viewport + canvas）。
 *   - /topology/domain   → src/pages/Topology/DomainManagement/index.tsx
 *       左侧 antd Tree（.ant-tree）+ 面板标题 nav.topology.domain（域管理）；
 *       初始未选中任何域 → 右侧空态提示 common.pleaseSelect（请选择）。
 *   - /topology/site     → src/pages/Topology/SiteManagement/index.tsx
 *       ListPageLayout 标题 nav.topology.site（站点管理）+ FilterBar 表单（.ant-form）
 *       + DataTable（.ant-table，真实栈数据稀疏，空表也有表头）。
 *   - /topology/settings → src/pages/Topology/TopologySettings/index.tsx
 *       纯前端静态设置表单（无后端依赖）：ListPageLayout 标题 nav.topology.settings（拓扑设置，
 *       与侧边栏菜单同名 → 收窄到 main 断言）+ 提示文案 topology.settings.hint（页面唯一）
 *       + antd Collapse 分组（.ant-collapse，布局算法/交互设置等）+ 保存按钮 common.save。
 */
test.describe('拓扑管理冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/topology/gis-map 渲染，筛选面板与 OpenLayers 地图 canvas 可见', async ({ page }) => {
    await expectPageRenders(page, '/topology/gis-map');

    // 左侧筛选面板：设备状态 / 设备组折叠面板标题 + 在线激活状态项
    await expect(page.getByText(/设备状态|Device Status/).first()).toBeVisible();
    await expect(page.getByText(/设备组|Device Group/).first()).toBeVisible();
    await expect(page.getByText(/在线激活|Online Active/).first()).toBeVisible();

    // OpenLayers 地图容器与 canvas（地图初始化可能较慢，放宽超时）
    await expect(page.locator('.ol-viewport').first()).toBeVisible({ timeout: 30_000 });
    await expect(page.locator('.ol-viewport canvas').first()).toBeVisible({ timeout: 30_000 });
  });

  test('/topology/domain 渲染，域树与未选中空态提示可见', async ({ page }) => {
    await expectPageRenders(page, '/topology/domain');

    // 左侧面板标题（nav.topology.domain：域管理）
    await expect(page.getByText(/域管理|Domain Management/).first()).toBeVisible();

    // antd Tree 骨架已渲染
    await expect(page.locator('.ant-tree').first()).toBeVisible();

    // 初始未选中任何域 → 右侧空态提示（common.pleaseSelect）
    await expect(page.getByText(/请选择|Please select/).first()).toBeVisible();
  });

  test('/topology/site 渲染，标题 / 筛选表单 / 站点表格可见', async ({ page }) => {
    await expectPageRenders(page, '/topology/site');

    // 页面标题（nav.topology.site：站点管理）
    await expect(page.getByText(/站点管理|Site Management/).first()).toBeVisible();

    // FilterBar 筛选表单
    await expect(page.locator('.ant-form').first()).toBeVisible();

    // 站点列表表格骨架（真实栈可能无数据，空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();

    // 新增按钮
    await expect(page.getByRole('button', { name: /新增|Add/ }).first()).toBeVisible();
  });

  test('/topology/settings 渲染，标题 / 提示文案 / 设置分组可见', async ({ page }) => {
    await expectPageRenders(page, '/topology/settings');

    // 页面标题（nav.topology.settings：拓扑设置）——与侧边栏菜单同名，收窄到 main
    await expect(
      page.locator('main').getByText(/拓扑设置|Topology Settings/).first(),
    ).toBeVisible();

    // 页面唯一提示文案（topology.settings.hint）
    await expect(
      page.getByText(/将影响拓扑图的显示效果|affect the topology display/).first(),
    ).toBeVisible();

    // antd Collapse 设置分组骨架已渲染
    await expect(page.locator('.ant-collapse').first()).toBeVisible();

    // 保存按钮（common.save）
    await expect(page.getByRole('button', { name: /保存|Save/ }).first()).toBeVisible();
  });
});
