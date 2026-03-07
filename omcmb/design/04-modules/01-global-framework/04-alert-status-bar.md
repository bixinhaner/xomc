# 实时告警状态栏 (Alert Status Bar)

## 页面用途

在 Header 右侧区域展示实时告警等级统计，通过 WebSocket 推送保持数据实时更新。提供一目了然的系统健康状态概览，并支持点击快速跳转到对应等级的告警列表。

## 布局模板

内嵌于 Header 组件内部，不独立占用页面空间。

## 线框描述

```
+---------------------------------------------------+
|                                                   |
|  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ |
|  │ 紧急  5  │ │ 主要 12  │ │ 次要 28  │ │ 警告 45  │ |
|  └─────────┘ └─────────┘ └─────────┘ └─────────┘ |
|                                                   |
+---------------------------------------------------+
   ↑ red       ↑ orange    ↑ yellow    ↑ blue
```

### 单个角标详细结构

```
+------------------+
| [●] 紧急    5    |
+------------------+
  ↑    ↑       ↑
  圆点  标签   计数

  圆点: 6px 实心圆, 与背景色同色
  标签: 中文等级名称
  计数: font-weight: 600
```

## 四级告警颜色系统

| 等级   | 英文标识    | 背景色    | 文字色    | 圆点色    | 排列顺序 |
|-------|-----------|----------|----------|----------|---------|
| 紧急   | Critical  | #FF4D4F  | #FFFFFF  | #FF4D4F  | 1 (最左) |
| 主要   | Major     | #FA8C16  | #FFFFFF  | #FA8C16  | 2       |
| 次要   | Minor     | #FADB14  | #333333  | #FADB14  | 3       |
| 警告   | Warning   | #1890FF  | #FFFFFF  | #1890FF  | 4 (最右) |

> 颜色系统详细规范参见 `04-alarm-management/09-severity-color-system.md`

## 角标样式规格

```css
.alarm-badge {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
  user-select: none;
}

.alarm-badge:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.alarm-badge:active {
  transform: translateY(0);
}

.alarm-badge .count {
  font-weight: 600;
  margin-left: 4px;
}

.alarm-badge-group {
  display: flex;
  gap: 8px;
  align-items: center;
}
```

## 计数动画规格

| 场景               | 动画效果                                    |
|-------------------|--------------------------------------------|
| 计数增加            | 数字放大弹跳 (scale 1.0 → 1.3 → 1.0, 300ms) |
| 计数减少            | 无特殊动画，直接更新                           |
| 紧急告警新增         | 整个角标闪烁 2 次 (opacity 闪烁, 600ms)        |
| 首次加载            | 从 0 滚动至实际值 (countUp 动画, 500ms)        |
| 计数为 0            | 角标 opacity: 0.45, 不可点击                  |
| 计数超过 999        | 显示 "999+"                                 |
| 计数超过 9999       | 显示 "9999+"                                |

## WebSocket 实时更新

### 连接配置

```typescript
interface AlarmWebSocketConfig {
  url: string;               // ws://host/ws/alarm-count
  reconnectInterval: 3000;   // 断线重连间隔 3s
  maxReconnectAttempts: 10;  // 最大重连次数
  heartbeatInterval: 30000;  // 心跳间隔 30s
}
```

### 消息格式

```json
// 服务端推送消息
{
  "type": "alarm_count_update",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "critical": 5,
    "major": 12,
    "minor": 28,
    "warning": 45
  },
  "deviceType": "eNB"    // 对应当前全局设备类型上下文
}
```

### 连接状态处理

| 连接状态       | UI 表现                                      |
|--------------|---------------------------------------------|
| 连接中        | 角标显示 Skeleton 加载态                       |
| 已连接        | 正常显示实时数据                                |
| 断线重连中     | 计数显示 "--", 角标灰色, Tooltip "连接中断，正在重连..." |
| 重连失败       | 显示 "⚠ 连接失败" 提示文字, 提供手动刷新按钮       |
| 心跳超时       | 触发重连逻辑                                   |

### 数据同步策略

```
初始化流程：
1. 页面加载 → 发送 REST API 请求获取当前告警计数 (HTTP GET /api/alarm/count)
2. REST 返回后渲染初始数据
3. 同时建立 WebSocket 连接
4. WebSocket 连接成功后，后续通过推送更新
5. 设备类型切换时，发送 WebSocket 消息请求新设备类型的数据

刷新策略：
- WebSocket 推送: 实时更新
- 定时轮询兜底: 每 60s 调用 REST API 校准数据 (防止推送遗漏)
- 设备类型切换: 立即调用 REST API + 重新订阅 WebSocket 频道
```

## 点击交互

| 点击目标     | 导航行为                                          |
|------------|--------------------------------------------------|
| 紧急角标     | 跳转 `/alarm/current?severity=critical`           |
| 主要角标     | 跳转 `/alarm/current?severity=major`              |
| 次要角标     | 跳转 `/alarm/current?severity=minor`              |
| 警告角标     | 跳转 `/alarm/current?severity=warning`            |
| 计数为 0    | 不可点击, `cursor: not-allowed`                    |

**跳转行为细节：**

- 如果当前已在告警列表页，仅更新筛选条件，不重新打开 Tab
- 如果当前在其他页面，新开一个告警列表 Tab（或切换到已存在的告警 Tab 并更新筛选）
- 跳转后，告警列表的等级筛选器自动选中对应等级

## Tooltip 信息

Hover 角标时显示 Tooltip：

```
+---------------------------------+
| 紧急告警                         |
| 当前数量: 5                      |
| 最近 1 小时变化: +2              |
| 最新: eNB-001 光模块故障          |
|       2 分钟前                   |
| 点击查看全部紧急告警 →            |
+---------------------------------+
```

## 跨模块导航

| 来源          | 目标                 | 触发方式                    |
|--------------|---------------------|-----------------------------|
| 告警角标       | 告警管理 > 当前告警   | 点击角标, 携带 severity 参数  |
| Tooltip 链接  | 告警管理 > 告警详情   | 点击 Tooltip 中的最新告警     |
