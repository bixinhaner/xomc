# 设备分组、自定义告警、指标管理 - UI 样式重构方案

**日期**: 2026-04-16
**参考设计**: Vercel, Linear, Stripe
**核心目标**: 统一间距系统、增强呼吸感、优化视觉层次、支持暗色模式

---

## 📋 目录

1. [设计原则](#设计原则)
2. [间距系统](#间距系统)
3. [文件结构](#文件结构)
4. [实施步骤](#实施步骤)
5. [样式指南](#样式指南)
6. [响应式方案](#响应式方案)
7. [暗色模式](#暗色模式)
8. [验收标准](#验收标准)

---

## 设计原则

### 核心理念

| 原则 | 说明 | 应用示例 |
|------|------|---------|
| **统一间距** | 基于 4px 基础单位的倍数系统 | padding: 8px, 12px, 16px, 24px |
| **呼吸感** | 元素间留白充足，避免拥挤 | gap: 16px-20px, 统计项 padding: 12px 16px |
| **视觉层次** | 通过字号、字重、颜色建立层级 | 标题 17px/600, 正文 13px/400 |
| **微妙过渡** | 轻柔的动画，0.15-0.2s cubic-bezier | hover: transform translateX(2px) |
| **功能优先** | 布局服务于功能，非装饰 | 左侧导航宽度 260-300px |

### 设计模式参考

**Vercel 风格**:
- 极简主义，大量留白
- 柔和的灰度系统 (#fafafa, #f5f5f5)
- 微妙的阴影和边框

**Linear 风格**:
- 高密度的信息展示但不拥挤
- 精细的间距控制 (4px/8px/12px)
- 强调当前选中状态

**Stripe 风格**:
- 网格化布局，对齐精确
- 颜色的功能性使用（蓝色强调、绿色成功、红色危险）
- 卡片式设计的深度运用

---

## 间距系统

### 基础单位

```
基础单位: 4px
```

### 间距阶梯

| 名称 | 值 | 用途 | 示例 |
|------|-----|------|------|
| xs | 4px | 紧凑间距 | Icon 与文字间隙 |
| sm | 8px | 小间距 | 列表项垂直 gap |
| md | 12px | 中等间距 | 卡片内部 padding 垂直 |
| lg | 16px | 大间距（标准）| 卡片 padding, 容器 gap |
| xl | 24px | 超大间距 | 区块间距 |
| 2xl | 32px | 极大间距 | 页面级 padding |

### 应用规则

```
树面板 Header:    padding: 16px 16px 12px (lg lg md)
树节点:           padding: 10px 12px (md-lg)
搜索框:           padding: 12px 16px (md lg)
右侧内容区:       padding: 18px 22px (xl+)
卡片:             gap: 16px-20px (lg-xl)
统计项:           padding: 12px 16px (md lg)
表格行高:         min-height: 40px (10xl)
```

---

## 文件结构

### 新增文件

```
src/components/TreeListLayout/
  └── TreeListLayout.module.css          # 通用左右布局样式

src/pages/device/DeviceGrouping/
  └── DeviceGrouping.module.css          # 设备分组专用样式

src/pages/alarm/CustomAlarmStats/
  └── CustomAlarmStats.module.css        # 自定义告警专用样式

src/pages/performance/ThresholdConfig/
  └── ThresholdConfig.module.css         # 指标管理专用样式
```

### 样式文件关系

```
TreeListLayout.module.css (基础)
       ↓ import
DeviceGrouping.module.css (扩展)
CustomAlarmStats.module.css (扩展)
ThresholdConfig.module.css (扩展)
```

---

## 实施步骤

### Step 1: 导入基础样式

在各个页面组件中导入对应的 CSS Module:

```tsx
// 设备分组
import styles from './DeviceGrouping.module.css';

// 自定义告警
import styles from './CustomAlarmStats.module.css';

// 指标管理
import styles from './ThresholdConfig.module.css';
```

### Step 2: 应用容器类名

#### 设备分组页面

```tsx
// 左侧树面板
<div className={styles.deviceGroupTreePanel}>
  <div className={styles.treePanelHeader}>
    <span className={styles.treePanelTitle}>设备分组</span>
  </div>
  <div className={styles.treeSearchContainer}>
    {/* 搜索框 */}
  </div>
  <div className={styles.treeNodesContainer}>
    {/* 树节点 */}
    <div className={styles.groupNode}>
      <div className={styles.treeNodeContent}>
        <span className={styles.treeNodeIcon}>
          <FolderOutlined />
        </span>
        <span className={styles.treeNodeText}>北京分组</span>
      </div>
      <span className={styles.groupCountBadge}>128</span>
    </div>
  </div>
</div>

// 右侧内容区
<div className={styles.deviceListContainer}>
  <div className={styles.pageHeader}>
    <h2 className={styles.pageTitle}>北京分组</h2>
  </div>

  {/* 分组信息卡片 */}
  <Card className={styles.groupInfoCard}>
    <div className={styles.groupInfoHeader}>
      <span className={styles.cardHeaderTitle}>分组信息</span>
    </div>
    <div className={styles.groupInfoBody}>
      <div className={styles.groupInfoRow}>
        <span className={styles.groupInfoLabel}>设备数量</span>
        <span className={styles.groupInfoValue}>128</span>
      </div>
    </div>
  </Card>

  {/* 设备列表卡片 */}
  <Card className={styles.deviceTableCard}>
    <DataTable ... />
  </Card>
</div>
```

#### 自定义告警页面

```tsx
// 左侧树面板
<div className={styles.customAlarmTreePanel}>
  <div className={styles.treePanelHeader}>
    <span className={styles.treePanelTitle}>告警分组</span>
  </div>
  <div className={styles.treeSearchContainer}>
    {/* 搜索框 */}
  </div>
  <div className={styles.treeNodesContainer}>
    {/* 告警分组节点 */}
    <div className={styles.alarmGroupNode}>
      <div className={styles.treeNodeContent}>
        <AlertOutlined className={styles.alarmGroupNodeIcon} />
        <span className={styles.treeNodeText}>北京告警</span>
      </div>
      <div className={styles.groupActions}>
        {/* 编辑/删除按钮 */}
      </div>
    </div>
  </div>
</div>

// 右侧内容区
<div className={styles.customAlarmContentPanel}>
  <div className={styles.pageHeader}>
    <h2 className={styles.pageTitle}>北京告警</h2>
  </div>

  {/* 统计卡片 - 两行网格布局 */}
  <Card className={styles.statsCard}>
    <div className={styles.statsCardBody}>
      <div className={`${styles.statsItem} ${styles.statsItemActive}`}>
        <span className={styles.statsItemLabel}>总计</span>
        <span className={styles.statsItemValue}>256</span>
      </div>
      {/* 更多统计项... */}
    </div>
  </Card>

  {/* 快捷筛选 Pills */}
  <div className={styles.quickFilterPills}>
    <div className={`${styles.quickFilterPill} ${styles.quickFilterPillActive}`}>
      全部
    </div>
    <div className={styles.quickFilterPill}>严重</div>
    {/* 更多筛选项... */}
  </div>

  {/* 筛选器 */}
  <div className={styles.alarmFilterWrapper}>
    <FilterBar ... />
  </div>

  {/* 告警列表卡片 */}
  <Card className={styles.alarmListCard}>
    <DataTable ... />
  </Card>
</div>
```

#### 指标管理页面（改造为左右布局）

```tsx
// 主容器 - 改为左右布局
<div className={styles.thresholdConfigLayout}>
  {/* 左侧 KPI 分类导航 */}
  <div className={styles.kpiCategoryPanel}>
    <div className={styles.kpiCategoryHeader}>
      <span className={styles.kpiCategoryTitle}>KPI 分类</span>
    </div>
    <div className={styles.kpiCategoryList}>
      {/* 全部 KPI */}
      <div className={`${styles.kpiCategoryItem} ${styles.kpiCategoryAll} ${styles.kpiCategoryItemSelected}`}>
        <div className={styles.kpiCategoryContent}>
          <AppstoreOutlined className={styles.kpiCategoryIcon} />
          <span className={styles.kpiCategoryText}>全部指标</span>
        </div>
        <span className={styles.kpiCategoryCount}>7</span>
      </div>

      {/* 分类项 */}
      <div className={styles.kpiCategoryItem}>
        <div className={styles.kpiCategoryContent}>
          <PhoneOutlined className={styles.kpiCategoryIcon} />
          <span className={styles.kpiCategoryText}>接入类</span>
        </div>
        <span className={styles.kpiCategoryCount}>2</span>
      </div>

      {/* 更多分类... */}
    </div>
  </div>

  {/* 右侧阈值配置内容 */}
  <div className={styles.thresholdContentPanel}>
    <div className={styles.thresholdPageHeader}>
      <h2 className={styles.thresholdPageTitle}>接入类指标</h2>
      {/* 工具栏按钮 */}
    </div>

    {/* 当前分类信息 */}
    <Card className={styles.currentCategoryCard}>
      <div className={styles.currentCategoryHeader}>
        <span className={styles.currentCategoryTitle}>接入类指标</span>
      </div>
      <div className={styles.currentCategoryBody}>
        <div className={styles.currentCategoryStat}>
          <span className={styles.currentCategoryStatLabel}>配置阈值</span>
          <span className={styles.currentCategoryStatValue}>2</span>
        </div>
        {/* 更多统计... */}
      </div>
    </Card>

    {/* 快捷筛选 */}
    <div className={styles.thresholdQuickFilter}>
      <span className={styles.thresholdQuickFilterLabel}>状态:</span>
      <div className={styles.thresholdQuickFilterPills}>
        <div className={`${styles.quickFilterPill} ${styles.quickFilterPillActive}`}>全部</div>
        <div className={styles.quickFilterPill}>已启用</div>
        <div className={styles.quickFilterPill}>已禁用</div>
      </div>
    </div>

    {/* 阈值列表 */}
    <Card className={styles.thresholdListCard}>
      <DataTable ... />
    </Card>
  </div>
</div>
```

### Step 3: 更新组件结构

#### TreeListPageLayout 改造

如果使用现有的 `TreeListPageLayout` 组件，需要更新其样式支持：

```tsx
// 在 TreeListPageLayout 组件中
import treeListStyles from '@/components/TreeListLayout/TreeListLayout.module.css';

<TreeListPageLayout 
  tree={treePanel} 
  defaultTreeWidth={280}  // 调整默认宽度
  className={treeListStyles.treeListLayout}  // 添加布局类
>
  {contentPanel}
</TreeListPageLayout>
```

#### 自定义左右布局容器

对于需要更多控制的页面，可以完全自定义布局：

```tsx
<div className={styles.treeListLayout}>
  <aside className={styles.treePanelContainer}>
    {/* 左侧内容 */}
  </aside>
  <main className={styles.contentPanel}>
    {/* 右侧内容 */}
  </main>
</div>
```

### Step 4: 微调与验证

1. **间距检查**: 使用 CSS 验证工具确保间距符合规范
2. **响应式测试**: 在不同屏幕尺寸下测试布局
3. **暗色模式测试**: 切换主题验证样式
4. **浏览器兼容性**: 测试主流浏览器

---

## 样式指南

### 树面板

#### Header 样式

```css
.treePanelHeader {
  padding: 16px 16px 12px;  /* 上16 下12 左右16 */
  background: linear-gradient(to bottom, #fafafa, #ffffff);
  border-bottom: 1px solid #f5f5f5;  /* 更浅的边框 */
}
```

#### 搜索框

```css
.treeSearchInput {
  height: 32px;  /* 增加高度 */
  border-radius: 6px;
}
.treeSearchInput:focus {
  box-shadow: 0 0 0 2px rgba(22, 119, 255, 0.1);  /* 蓝色外发光 */
}
```

#### 树节点

```css
.treeNode {
  padding: 10px 12px;  /* 增加内边距 */
  min-height: 40px;  /* 增加最小高度 */
  border-radius: 6px;
}
.treeNode:hover {
  background: rgba(0, 0, 0, 0.04);
  transform: translateX(2px);  /* 微妙右移 */
}
.treeNodeSelected {
  background: linear-gradient(90deg, rgba(22, 119, 255, 0.08), rgba(22, 119, 255, 0.03));
}
```

### 统计卡片

#### 两行网格布局（推荐）

```css
.statsCardBody {
  display: grid;
  grid-template-columns: repeat(4, 1fr) repeat(5, 1fr);  /* 第一行4个，第二行5个 */
  gap: 12px;
  padding: 14px 18px;
}

@media (max-width: 1600px) {
  .statsCardBody {
    grid-template-columns: repeat(3, 1fr) repeat(6, 1fr);  /* 3+6布局 */
  }
}

@media (max-width: 1280px) {
  .statsCardBody {
    grid-template-columns: repeat(4, 1fr);  /* 单行4列布局 */
  }
}
```

#### StatItem 样式

```css
.statItem {
  padding: 12px 16px;  /* 充足内边距 */
  border-radius: 8px;
  background: #fafafa;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
.statItem:hover {
  transform: scale(1.02);  /* 微妙放大 */
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}
.statItem:active {
  transform: scale(0.98);  /* 点击缩小 */
}
```

### 卡片样式

```css
.contentCard {
  border-radius: 8px;
  border: 1px solid #f0f0f0;
  background: #ffffff;
  transition: all 0.2s ease;
}
.contentCard:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06),
              0 4px 16px rgba(0, 0, 0, 0.04);
  border-color: #e6e6e6;
}
```

---

## 响应式方案

### 断点定义

```css
/* 小屏幕 */
@media (max-width: 1280px) {
  .treePanelContainer { width: 220px-240px; }
  .contentPanel { padding: 14px 16px; gap: 14px; }
  .statsCardBody { grid-template-columns: repeat(4, 1fr); }
}

/* 中等屏幕（默认） */
@media (min-width: 1280px) and (max-width: 1600px) {
  .treePanelContainer { width: 260px-280px; }
  .contentPanel { padding: 16px 18px; gap: 16px; }
  .statsCardBody { grid-template-columns: repeat(3, 1fr) repeat(6, 1fr); }
}

/* 大屏幕 */
@media (min-width: 1600px) {
  .treePanelContainer { width: 280px-300px; }
  .contentPanel { padding: 18px 22px; gap: 18px; }
  .statsCardBody { grid-template-columns: repeat(4, 1fr) repeat(5, 1fr); }
}
```

### 移动端适配（可选）

对于需要支持移动端的场景：

```css
@media (max-width: 768px) {
  .treeListLayout {
    flex-direction: column;
  }
  
  .treePanelContainer {
    width: 100%;
    max-height: 300px;
    border-right: none;
    border-bottom: 1px solid #f0f0f0;
  }
  
  .contentPanel {
    padding: 12px 16px;
    gap: 12px;
  }
  
  .statsCardBody {
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }
}
```

---

## 暗色模式

### CSS 变量定义

```css
:root {
  --color-bg-layout: #f5f5f5;
  --color-bg-container: #ffffff;
  --color-border: #f0f0f0;
  --color-text: #262626;
  --color-text-secondary: #595959;
  --color-text-tertiary: #8c8c8c;
  --color-text-quaternary: #bfbfbf;
}

:global([data-theme='tech']) {
  --color-bg-layout: #0a0a0a;
  --color-bg-container: #141414;
  --color-border: #262626;
  --color-text: #e8e8e8;
  --color-text-secondary: #a6a6a6;
  --color-text-tertiary: #737373;
  --color-text-quaternary: #525252;
}
```

### 暗色模式适配示例

```css
/* 卡片 */
.contentCard {
  background: #ffffff;
  border-color: #f0f0f0;
}

:global([data-theme='tech']) .contentCard {
  background: #1f1f1f;
  border-color: #303030;
}

/* 树节点 hover */
.treeNode:hover {
  background: rgba(0, 0, 0, 0.04);
}

:global([data-theme='tech']) .treeNode:hover {
  background: rgba(255, 255, 255, 0.06);
}

/* 选中状态 */
.treeNodeSelected {
  background: linear-gradient(90deg, rgba(22, 119, 255, 0.08), rgba(22, 119, 255, 0.03));
}

:global([data-theme='tech']) .treeNodeSelected {
  background: linear-gradient(90deg, rgba(22, 119, 255, 0.15), rgba(22, 119, 255, 0.05));
}
```

---

## 验收标准

### 视觉验收

- [ ] 间距符合 4px 倍数系统
- [ ] 树节点行高 ≥ 40px，点击区域充足
- [ ] 统计卡片有足够 padding，不拥挤
- [ ] hover 状态有微妙动画（0.15-0.2s）
- [ ] 卡片边框、阴影符合设计规范

### 功能验收

- [ ] 树面板可折叠（可选功能）
- [ ] 树节点选中状态明显
- [ ] 响应式布局在不同屏幕下正常
- [ ] 暗色模式下颜色对比度符合 WCAG AA

### 性能验收

- [ ] CSS 文件大小合理（< 50KB）
- [ ] 无布局抖动（CLS < 0.1）
- [ ] 动画流畅（60fps）

### 兼容性验收

- [ ] Chrome/Edge 最新版本
- [ ] Firefox 最新版本
- [ ] Safari 最新版本

---

## 附录：快速参考

### 常用类名速查

| 类名 | 用途 | 所属文件 |
|------|------|---------|
| `.treeListLayout` | 左右布局容器 | TreeListLayout.module.css |
| `.treePanelContainer` | 左侧树面板 | TreeListLayout.module.css |
| `.treePanelHeader` | 树面板头部 | TreeListLayout.module.css |
| `.treeNode` | 树节点 | TreeListLayout.module.css |
| `.treeNodeSelected` | 选中状态 | TreeListLayout.module.css |
| `.contentPanel` | 右侧内容区 | TreeListLayout.module.css |
| `.contentCard` | 内容卡片 | TreeListLayout.module.css |
| `.statItem` | 统计项 | TreeListLayout.module.css |
| `.statsCardBody` | 统计网格 | CustomAlarmStats.module.css |

### 颜色速查

| 用途 | 亮色模式 | 暗色模式 |
|------|---------|---------|
| 布局背景 | #f5f5f5 | #0a0a0a |
| 容器背景 | #ffffff | #141414 |
| 边框 | #f0f0f0 | #303030 |
| 主要文字 | #262626 | #e8e8e8 |
| 次要文字 | #595959 | #a6a6a6 |
| 辅助文字 | #8c8c8c | #737373 |
| 主色 | #1677ff | #4096ff |
| 成功 | #52c41a | #73d13d |
| 警告 | #faad14 | #ffc53d |
| 错误 | #ff4d4f | #ff7875 |

---

**文档版本**: v1.0  
**最后更新**: 2026-04-16  
**维护者**: 前端团队
