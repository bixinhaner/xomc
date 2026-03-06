# 阴影与层级 (Shadows & Elevation)

## 阴影层级

| 层级 | Token | CSS 值 | 用途 |
|------|-------|--------|------|
| 0 | `shadow-none` | `none` | 平面元素：表格行、行内元素 |
| 1 | `shadow-sm` | `0 1px 2px 0 rgba(0,0,0,0.03), 0 1px 6px -1px rgba(0,0,0,0.02), 0 2px 4px 0 rgba(0,0,0,0.02)` | 卡片、侧边栏、Header 底边 |
| 2 | `shadow-md` | `0 3px 6px -4px rgba(0,0,0,0.12), 0 6px 16px 0 rgba(0,0,0,0.08), 0 9px 28px 8px rgba(0,0,0,0.05)` | 下拉框、Tooltip、Popover、浮动面板 |
| 3 | `shadow-lg` | `0 6px 16px -8px rgba(0,0,0,0.08), 0 9px 28px 0 rgba(0,0,0,0.05), 0 12px 48px 16px rgba(0,0,0,0.03)` | 弹窗（Modal）、抽屉（Drawer）、覆盖面板 |

## Z-Index 层级

| 层级 | z-index | 元素 |
|------|---------|------|
| 基础内容 | 0 | 页面内容、表格 |
| 浮动面板 | 10 | 地图浮动统计面板、拓扑工具栏 |
| 侧边栏 | 100 | 固定侧边栏 |
| Header | 100 | 固定顶栏 |
| 任务面板 | 200 | 底部任务面板 |
| 下拉/Popover | 1000 | Select 下拉、DatePicker 面板 |
| Tooltip | 1050 | 提示气泡 |
| 弹窗遮罩 | 1000 | Modal 半透明遮罩 |
| 弹窗内容 | 1000 | Modal 内容区 |
| 通知 | 2000 | Toast 通知 |

## 使用规范

### ✅ 正确做法
- 卡片容器使用 `shadow-sm`
- 下拉菜单使用 `shadow-md`
- 弹窗使用 `shadow-lg`
- 层级越高的元素使用越深的阴影

### ❌ 错误做法
- 不要给表格行添加阴影
- 不要给内联元素添加阴影
- 不要使用自定义 z-index 值打破层级系统
