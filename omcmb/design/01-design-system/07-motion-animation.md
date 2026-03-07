# 动效规范 (Motion & Animation)

## 时长

| Token | 时长 | 缓动函数 | 用途 |
|-------|------|---------|------|
| `duration-fast` | 100ms | `ease-out` | 悬浮态变化、图标过渡 |
| `duration-normal` | 200ms | `ease-in-out` | 按钮按下、Toggle 切换、下拉展开 |
| `duration-slow` | 300ms | `ease-in-out` | 弹窗开关、侧边栏展开/收起、面板滑动 |
| `duration-slower` | 400ms | `ease-in-out` | 页面过渡、图表动画 |

## 具体动效

### 侧边栏展开/收起
- 属性: `width`
- 从: 48px → 240px / 240px → 48px
- 时长: `duration-slow` (300ms)
- 缓动: `ease-in-out`

### 弹窗打开
- 属性: `opacity` + `transform`
- 从: opacity 0 + scale(0.95) → opacity 1 + scale(1)
- 时长: `duration-slow` (300ms)
- 缓动: `ease-out`

### 弹窗关闭
- 属性: `opacity` + `transform`
- 从: opacity 1 + scale(1) → opacity 0 + scale(0.95)
- 时长: `duration-normal` (200ms)
- 缓动: `ease-in`

### 下拉菜单展开
- 属性: `max-height` + `opacity`
- 时长: `duration-normal` (200ms)
- 缓动: `ease-in-out`

### 表格行悬浮
- 属性: `background-color`
- 时长: `duration-fast` (100ms)
- 缓动: `ease-out`

### 图表数据更新
- 属性: 柱状图高度 / 折线位移
- 时长: `duration-slower` (400ms)
- 缓动: `ease-in-out`

### Toast 通知
- 进入: 从右侧滑入 + 淡入，`duration-slow`
- 停留: 自动消失前 3s（成功）/ 5s（错误）
- 退出: 向右滑出 + 淡出，`duration-normal`

## 原则

1. **功能性优先**: 动效服务于操作反馈，不做纯装饰动效
2. **快速响应**: 用户触发的交互动效不超过 300ms
3. **一致性**: 相同类型的动效使用相同的时长和缓动
4. **尊重偏好**: 检测 `prefers-reduced-motion`，减弱或关闭动效
