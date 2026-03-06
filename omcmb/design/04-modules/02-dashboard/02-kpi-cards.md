# KPI 快速统计卡片 (KPI Cards)

## 页面用途

Dashboard 第一行的 4 个关键指标卡片，提供设备总数、小区总数、EU 总数、RU 总数的快速概览。每张卡片包含图标、标签、数值、趋势箭头和对比文本。

## 布局模板

内嵌于 Dashboard 页面 Row 1，使用 `Row > Col span={6}` 四列等宽布局。

## 线框描述

### 单张 KPI 卡片结构

```
┌─────────────────────────────────────┐
│                                     │
│  ┌──────┐   设备总数                 │
│  │ Icon │   Device Total            │
│  │ 48px │                           │
│  └──────┘   256                     │
│             ↑↑↑ font-size: 36px     │
│                                     │
│             ▲ 较昨日 +12 (2.3%)      │
│                                     │
└─────────────────────────────────────┘
```

### 4 张卡片并排

```
┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐
│ 📡         │ │ 📶         │ │ 🖥          │ │ 📻         │
│ 设备总数    │ │ 小区总数    │ │ EU 总数     │ │ RU 总数     │
│   256      │ │  1,024     │ │   512      │ │   768      │
│ ▲ +12 较昨日│ │ ▲ +5 较昨日 │ │ ▼ -2 较昨日 │ │ → 0 较昨日  │
└────────────┘ └────────────┘ └────────────┘ └────────────┘
```

## 卡片数据规格

| 卡片     | 英文标识          | 图标                   | 图标背景色   | API 字段              |
|---------|------------------|----------------------|------------|----------------------|
| 设备总数  | Device Total     | `ClusterOutlined`    | #1890FF    | `deviceCount`        |
| 小区总数  | Cell Total       | `DeploymentUnitOutlined` | #52C41A | `cellCount`          |
| EU 总数  | EU Total         | `DesktopOutlined`    | #722ED1    | `euCount`            |
| RU 总数  | RU Total         | `WifiOutlined`       | #FA8C16    | `ruCount`            |

## 卡片样式规格

### 卡片容器

| 属性         | 值                                |
|-------------|-----------------------------------|
| 高度         | 120px                             |
| 背景色       | #FFFFFF                           |
| 圆角         | 8px                               |
| 阴影         | 0 2px 8px rgba(0, 0, 0, 0.06)    |
| 内边距       | 20px 24px                         |
| Hover 效果   | box-shadow 加深, translateY(-2px)  |
| 点击效果     | cursor: pointer                   |
| 布局方式     | flex, row, space-between          |

### 图标区域

| 属性         | 值                              |
|-------------|--------------------------------|
| 容器尺寸     | 48 x 48 px                     |
| 圆角         | 12px                           |
| 背景色       | 各卡片对应色 opacity 0.1         |
| 图标尺寸     | 24px                           |
| 图标颜色     | 各卡片对应主色                   |

### 文字区域

| 元素         | 样式                                      |
|-------------|------------------------------------------|
| 标签文字     | font-size: 14px; color: #8C8C8C; line-height: 22px |
| 数值文字     | font-size: 36px; font-weight: 700; color: #262626; line-height: 44px |
| 数值格式     | 千分位分隔符 (例: 1,024)                    |

### 趋势指示区域

| 元素         | 样式                                         |
|-------------|----------------------------------------------|
| 容器         | margin-top: 4px; font-size: 12px              |
| 上升箭头     | `CaretUpOutlined`, color: #52C41A (绿色)      |
| 下降箭头     | `CaretDownOutlined`, color: #FF4D4F (红色)    |
| 持平箭头     | `MinusOutlined`, color: #8C8C8C (灰色)        |
| 对比文字     | color: #8C8C8C; 格式: "较昨日 +{diff} ({percent}%)" |
| 正数差值     | 显示 "+{n}", 绿色                              |
| 负数差值     | 显示 "-{n}", 红色                              |
| 零差值       | 显示 "0", 灰色                                 |

## 数据接口

```typescript
interface KPICardData {
  deviceCount: KPIMetric;
  cellCount: KPIMetric;
  euCount: KPIMetric;
  ruCount: KPIMetric;
}

interface KPIMetric {
  current: number;          // 当前值
  previous: number;         // 昨日同期值
  diff: number;             // 差值 (current - previous)
  diffPercent: number;      // 变化百分比
  trend: 'up' | 'down' | 'flat';  // 趋势方向
}

// API 调用
GET /api/dashboard/kpi?deviceType={deviceType}
```

## 动画效果

| 场景         | 动画                                       |
|-------------|-------------------------------------------|
| 首次加载     | 数字从 0 滚动到实际值 (countUp, 800ms, easeOut) |
| 数据刷新     | 数字从旧值滚动到新值 (countUp, 400ms)          |
| 卡片出现     | 从下方淡入 (translateY(20px) → 0, opacity 0 → 1, 300ms, 依次延迟 100ms) |
| Hover        | box-shadow 加深 + translateY(-2px), 200ms    |

## 状态处理

| 状态         | 处理方式                                     |
|-------------|---------------------------------------------|
| 加载中       | Skeleton 骨架屏: 图标圆形骨架 + 两行文字骨架   |
| 加载失败     | 显示 "--" 代替数值, Tooltip "数据加载失败"     |
| 数值为 0     | 正常显示 "0", 不隐藏卡片                      |
| 超大数值     | >= 10000 显示 "1.2万", >= 1000000 显示 "120万"|

## 跨模块导航

| 点击卡片     | 导航目标                    | 携带参数                    |
|-------------|----------------------------|----------------------------|
| 设备总数     | `/device/list`             | `deviceType={currentType}` |
| 小区总数     | `/device/ne`               | `neType=cell`              |
| EU 总数     | `/device/ne`               | `neType=EU`                |
| RU 总数     | `/device/ne`               | `neType=RU`                |
