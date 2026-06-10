import { test, expect } from '@playwright/test';
import { expectPageRenders, smokeLogin } from './helpers';

/**
 * 产品中心域冒烟（真实后端，super_admin 守卫路由）：
 *   /product/products       产品装配件列表（ProductRegistry）
 *   /product/param-model    参数模型清单（ModelsTab 列表态）
 *   /product/kpi-library    KPI 指标库（SummaryTab 平台一级列表）
 *   /product/orphan-devices 孤儿设备只读列表（数据稀疏时为 Empty 态）
 *
 * 路由均为 withSuperAdmin(PrivateRoute requireSuperAdmin) 包裹，admin 账号当前可访问。
 * 选择器依据 src/pages/product/<页面>/index.tsx + frontend-core/src/i18n/{zh-CN,en-US}：
 *   - SearchInput 包装 antd Input.Search，placeholder 可直接断言；
 *   - 不断言具体业务数据，只断言搜索框 / 操作按钮 / 表格骨架（空表也有表头）。
 */
test.describe('产品中心冒烟（真实后端）', { tag: '@smoke' }, () => {
  test.beforeEach(async ({ page }) => {
    await smokeLogin(page);
  });

  test('/product/products 渲染，搜索框 + 新增产品按钮 + 产品表格可见', async ({ page }) => {
    await expectPageRenders(page, '/product/products');

    // toolbar：搜索框（product.products.searchPh）+ 新增产品按钮（product.products.createBtn）
    await expect(
      page.getByPlaceholder(/名称 \/ 厂商|name \/ vendor/).first(),
    ).toBeVisible();
    await expect(
      page.getByRole('button', { name: /新增产品|New product/ }),
    ).toBeVisible();

    // 产品列表表格骨架（空表也有表头）
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/product/param-model 渲染，列表态搜索框 + 导入 XML 按钮 + 模型清单表格可见', async ({ page }) => {
    await expectPageRenders(page, '/product/param-model');

    // 列表态 toolbar：搜索框（product.paramModel.searchPh）+ 导入 XML（common.importXml）
    await expect(
      page.getByPlaceholder(/名称 \/描述|name\/description/).first(),
    ).toBeVisible();
    await expect(
      page.getByRole('button', { name: /导入 XML|Import XML/ }),
    ).toBeVisible();

    // ModelsTab 参数模型清单表格
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/product/kpi-library 渲染，平台搜索框 + 导入 XML 按钮 + 平台汇总表格可见', async ({ page }) => {
    await expectPageRenders(page, '/product/kpi-library');

    // 列表态 toolbar：搜索框（product.kpi.summary.searchPh）+ 导入 XML（common.importXml）
    await expect(
      page.getByPlaceholder(/名称 \/ 描述|name \/ description/).first(),
    ).toBeVisible();
    await expect(
      page.getByRole('button', { name: /导入 XML|Import XML/ }),
    ).toBeVisible();

    // SummaryTab（平台，制式）一级列表表格
    await expect(page.locator('.ant-table').first()).toBeVisible();
  });

  test('/product/orphan-devices 渲染，搜索框可见，表格或空态二选一', async ({ page }) => {
    await expectPageRenders(page, '/product/orphan-devices');

    // 搜索框（product.orphan.searchPh，双语都含 "SN / OUI"）
    await expect(page.getByPlaceholder(/SN \/ OUI/).first()).toBeVisible();

    // 页面在无孤儿设备时渲染 Empty（product.orphan.emptyState），有数据时渲染 Table
    await expect(
      page.locator('.ant-table').or(page.locator('.ant-empty')).first(),
    ).toBeVisible();
  });
});
