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

  });

  test('用户信息与快速入口卡片暂时不渲染', async ({ page }) => {
    await smokeLogin(page);
    await expectPageRenders(page, '/dashboard');

    await expect(
      page.locator('main').getByText(/快速入口|Quick Access/, { exact: true }),
    ).toHaveCount(0);
    await expect(
      page.locator('main').getByText(/System Admin|系统管理员/, { exact: true }),
    ).toHaveCount(0);
  });

  test('侧边栏收起后菜单图标与 Logo 保持同一中心线', async ({ page }) => {
    await smokeLogin(page);
    await expectPageRenders(page, '/dashboard');

    await page.getByRole('button', { name: /收起侧边栏|Collapse Sidebar/ }).click();

    const sidebar = page.getByRole('complementary', { name: '导航菜单' });
    const collapsedMenu = sidebar.locator('.ant-menu-inline-collapsed');
    await expect(collapsedMenu).toBeVisible();
    await page.waitForTimeout(350);

    const logoBox = await sidebar.getByText('OMC', { exact: true }).boundingBox();
    const firstMenuIconBox = await collapsedMenu
      .locator(':scope > .ant-menu-item .anticon, :scope > .ant-menu-submenu > .ant-menu-submenu-title .anticon')
      .first()
      .boundingBox();

    if (!logoBox || !firstMenuIconBox) {
      throw new Error('无法测量侧边栏 Logo 或菜单图标位置');
    }

    const logoCenter = logoBox.x + logoBox.width / 2;
    const iconCenter = firstMenuIconBox.x + firstMenuIconBox.width / 2;
    expect(Math.abs(logoCenter - iconCenter)).toBeLessThanOrEqual(1);
  });

  test('侧边栏收起动画中菜单图标不会先向右大幅偏移再返回', async ({ page }) => {
    await smokeLogin(page);
    await expectPageRenders(page, '/dashboard');

    const iconCenters = await page.evaluate(async () => {
      const sidebar = document.querySelector<HTMLElement>('aside[aria-label="导航菜单"]');
      const collapseButton = document.querySelector<HTMLButtonElement>(
        'button[aria-label="Collapse Sidebar"], button[aria-label="收起侧边栏"]',
      );
      const firstMenuIcon = sidebar?.querySelector<HTMLElement>(
        '.ant-menu-item .anticon, .ant-menu-submenu-title .anticon',
      );

      if (!sidebar || !collapseButton || !firstMenuIcon) {
        throw new Error('无法获取侧边栏、收起按钮或菜单图标');
      }

      const samples: number[] = [];
      const readCenter = () => {
        const rect = firstMenuIcon.getBoundingClientRect();
        return rect.left + rect.width / 2;
      };

      samples.push(readCenter());
      collapseButton.click();

      await new Promise<void>((resolve) => {
        const startedAt = performance.now();
        const capture = () => {
          samples.push(readCenter());
          if (performance.now() - startedAt >= 450) {
            resolve();
            return;
          }
          requestAnimationFrame(capture);
        };
        requestAnimationFrame(capture);
      });

      return samples;
    });

    const startCenter = iconCenters[0];
    const endCenter = iconCenters.at(-1) ?? startCenter;
    const maxCenter = Math.max(...iconCenters);
    expect(maxCenter - Math.max(startCenter, endCenter)).toBeLessThanOrEqual(2);
  });
});
