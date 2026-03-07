# 实时更新 (Real-time Updates)

> OMC 统一网管系统 - 交互设计规范

## 概述

网管系统需要实时展示告警、设备状态、任务进度等动态数据。本文定义 WebSocket 推送与 Polling 降级的交互模式，以及各种实时数据的视觉更新方式。

---

## 1. WebSocket 告警推送

### 1.1 新告警到达

```
行为:
  1. WebSocket 收到新告警消息
  2. 告警列表: 新行从顶部插入 (push to top)
  3. 新行高亮: 背景色闪烁动画 (highlight flash)
  4. Dashboard: 告警计数更新 (数字递增动画)
  5. Header: 告警铃铛图标红点更新

新行高亮动画:
  - 背景色: @warning-color 20% → 透明
  - duration: 2s
  - 闪烁次数: 1 次 (渐变消失)
  - 高亮结束后恢复正常行样式
```

### 1.2 告警状态变更

```
行为:
  1. 告警确认/清除 → 更新对应行的状态列
  2. 状态列更新: 平滑文字切换 (无闪烁)
  3. 如果用户正在查看该告警的详情 Drawer → 同步更新

告警清除:
  - 当前告警列表: 行移除 (淡出动画 300ms)
  - 历史告警列表: 行状态更新为 "已清除"
```

### 1.3 告警音效 (可选)

```
紧急告警 (Critical): 可配置声音提醒
设置: 用户可在个人设置中开启/关闭告警音效
行为: 浏览器 Notification API 推送系统通知 (需用户授权)
```

---

## 2. WebSocket 任务进度推送

### 2.1 任务面板更新

```
行为:
  1. 任务状态变更 → 更新任务面板对应任务的状态和进度
  2. 进度条: 平滑更新 (CSS transition 500ms)
  3. 百分比数字: 数字滚动动画 (countUp)
  4. 状态文字: 直接替换

任务面板实时更新:
  ┌─────────────────────────────────────────┐
  │ 批量升级 (5台设备)                       │
  │ ████████████████░░░░░░  60% → 80%      │  ← 进度条平滑增长
  │ 状态: 进行中  耗时: 00:05:23            │  ← 耗时实时更新
  │ 当前: 正在升级 eNodeB-003 (3/5)         │  ← 当前步骤更新
  └─────────────────────────────────────────┘
```

### 2.2 任务完成通知

```
任务完成:
  ┌─────────────────────────────────────────────┐
  │ ✓ 批量升级已完成: 成功 4 项, 失败 1 项      │
  │                              [查看详情]      │
  └─────────────────────────────────────────────┘

  类型: Notification (右下角弹出, 非 Toast)
  自动关闭: 10s
  操作: [查看详情] → 导航到任务详情页

任务失败:
  ┌─────────────────────────────────────────────┐
  │ ✕ 配置同步失败: 设备 eNodeB-003 连接超时    │
  │                              [查看详情]      │
  └─────────────────────────────────────────────┘

  类型: Notification (右下角弹出)
  自动关闭: 不自动关闭
```

---

## 3. WebSocket Dashboard KPI 更新

### 3.1 数字更新动画

```
行为: KPI 数字变化时使用平滑过渡动画

示例:
  在线设备数: 1,234 → 1,235 (数字滚动递增)
  告警总数:   56 → 58 (数字滚动递增)
  CPU 使用率: 45.2% → 47.8% (数字平滑变化)

动画:
  - 类型: 数字滚动 (countUp/countDown)
  - duration: 500ms
  - timing: ease-out
  - 小数位数: 保持一致

颜色变化:
  - 数字增加: 短暂显示绿色 → 恢复正常色
  - 数字减少: 短暂显示红色 → 恢复正常色
  - 变化持续时间: 1s
```

### 3.2 图表实时更新

```
行为: Dashboard 中的实时趋势图

折线图:
  - 新数据点追加到右侧
  - X 轴向左滚动 (保持最近 N 个点)
  - 动画: 新点淡入 + 线条延伸, duration 300ms

仪表盘:
  - 指针平滑旋转到新值, duration 500ms

柱状图:
  - 柱子高度平滑变化, duration 300ms
```

---

## 4. Polling 降级策略

### 4.1 降级条件

```
WebSocket 不可用的情况:
  - WebSocket 连接失败
  - 防火墙/代理不支持 WebSocket
  - 浏览器兼容性问题

降级流程:
  1. 尝试建立 WebSocket 连接
  2. 连接失败 → 自动降级为 Polling 模式
  3. 后台持续尝试恢复 WebSocket (每 60s 一次)
  4. WebSocket 恢复 → 切回 WebSocket, 停止 Polling
```

### 4.2 Polling 间隔

| 数据类型 | Polling 间隔 | 说明 |
|---------|-------------|------|
| 告警列表 | 30s | 告警数据对实时性要求高 |
| 任务进度 | 10s | 用户正在关注任务时 |
| Dashboard KPI | 60s | 概览数据允许轻微延迟 |
| 设备状态 | 60s | 设备状态变化频率较低 |

### 4.3 智能 Polling

```
规则:
  - 页面可见 (document.visibilityState === 'visible') 时正常 Polling
  - 页面不可见 (切到其他 Tab) 时暂停 Polling
  - 页面重新可见时立即触发一次请求 + 恢复 Polling
  - 用户手动刷新时重置 Polling 计时器
```

---

## 5. 实时数据视觉指示器

### 5.1 脉冲圆点 (Pulse Dot)

```
┌─────────────────────────────────────┐
│ ● 实时  最后更新: 14:30:25          │
└─────────────────────────────────────┘

脉冲动画:
  - 绿色圆点: 6px 直径
  - 外圈脉冲: 12px → 24px, opacity 1 → 0
  - duration: 2s
  - iteration: infinite

含义:
  ● 绿色脉冲: WebSocket 连接正常, 实时数据流通
  ● 橙色脉冲: Polling 模式, 有轻微延迟
  ● 灰色静止: 非实时数据 (静态页面)
```

### 5.2 最后更新时间戳

```
格式: "最后更新: HH:MM:SS"
位置: 数据区域右上角 或 表格标题栏
更新: 每次数据刷新后更新时间戳

示例:
  最后更新: 14:30:25 (实时)    ← WebSocket 模式
  最后更新: 14:30:00 (30s前)   ← Polling 模式, 显示相对时间
```

---

## 6. Header 连接状态指示器

### 6.1 交互规范

```
位置: 全局 Header 右侧区域

状态:
  ● 已连接          ← 绿色圆点 + 文字
  ● 连接中...       ← 橙色圆点 + 文字 (重连中)
  ● 已断开 [重连]   ← 红色圆点 + 文字 + 重连链接

Tooltip (hover 显示详情):
  ┌─────────────────────────────────┐
  │ WebSocket 连接状态              │
  │ 状态: 已连接                    │
  │ 服务器: ws://omc.example.com   │
  │ 连接时间: 2024-01-15 08:30:00  │
  │ 延迟: 12ms                     │
  └─────────────────────────────────┘
```

---

## 7. 数据冲突处理

### 7.1 场景

```
多用户同时操作同一条数据:
  用户 A 正在编辑设备参数
  用户 B 同时修改了该设备的参数
  WebSocket 推送了 B 的修改

处理方式:
  方案 A - 提示用户 (推荐):
    ┌──────────────────────────────────────────────────┐
    │ ⚠ 该设备的配置已被其他用户修改                     │
    │   修改人: 张三, 修改时间: 14:30:25                 │
    │   [刷新数据]  [继续编辑 (可能覆盖)]                │
    └──────────────────────────────────────────────────┘

  方案 B - 乐观锁:
    提交时检测版本号冲突 → 提示 "数据已被修改，请刷新后重试"
```

---

## 8. WebSocket 消息格式

### 8.1 统一消息结构

```typescript
interface WSMessage {
  type: 'alarm' | 'task' | 'kpi' | 'device_status' | 'heartbeat';
  action: 'create' | 'update' | 'delete';
  timestamp: string;   // ISO 8601
  data: any;           // 业务数据
}

// 示例: 新告警
{
  type: 'alarm',
  action: 'create',
  timestamp: '2024-01-15T14:30:25.000Z',
  data: {
    alarmId: 'ALM-20240115-001',
    severity: 'critical',
    deviceName: 'eNodeB-001',
    alarmTitle: 'CELL_UNAVAILABLE',
    // ...
  }
}
```

### 8.2 心跳机制

```
客户端 → 服务端: PING (每 30s)
服务端 → 客户端: PONG

规则:
  - 连续 3 次 PING 无 PONG 响应 → 判定连接断开
  - 触发重连逻辑 (参考 02-error-states.md 重连策略)
```

---

## 9. 实现参考

### WebSocket 管理

```typescript
// 推荐: 封装 WebSocket 管理器
class WSManager {
  private ws: WebSocket | null = null;
  private reconnectTimer: number = 0;
  private reconnectAttempts: number = 0;

  connect(url: string) {
    this.ws = new WebSocket(url);
    this.ws.onopen = () => this.onConnected();
    this.ws.onclose = () => this.onDisconnected();
    this.ws.onmessage = (e) => this.onMessage(e);
  }

  private onDisconnected() {
    // 自动重连 (exponential backoff)
    const delay = Math.min(2 ** this.reconnectAttempts * 1000, 30000);
    this.reconnectTimer = setTimeout(() => this.connect(this.url), delay);
    this.reconnectAttempts++;
  }
}
```

### Polling 管理

```typescript
// 推荐: 使用 TanStack Query (React Query)
useQuery({
  queryKey: ['alarms'],
  queryFn: fetchAlarms,
  refetchInterval: 30000,               // 30s Polling
  refetchIntervalInBackground: false,    // 后台不刷新
  refetchOnWindowFocus: true,            // 窗口聚焦时刷新
});
```
