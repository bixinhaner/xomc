# Button 按钮

## 概述
触发操作的交互元素，是系统中最基本的操作入口。

## 类型

| 类型 | 样式 | 用途 |
|------|------|------|
| Primary 主按钮 | 蓝色填充 `primary-600` | 主要操作：搜索、确认、新建 |
| Default 默认按钮 | 白色背景 + 灰色边框 | 次要操作：取消、重置、导出 |
| Danger 危险按钮 | 红色填充/红色边框 | 删除、清除等破坏性操作 |
| Text 文字按钮 | 无边框无背景 | 低优先级操作 |
| Link 链接按钮 | 蓝色文字无边框 | 表格行内操作、导航 |

## 尺寸

| 尺寸 | 高度 | 字号 | 内边距 | 用途 |
|------|------|------|--------|------|
| Small | 24px | 12px | 0 8px | 表格行内、紧凑区域 |
| Medium | 32px | 14px | 0 16px | **默认**，筛选栏、工具栏 |
| Large | 40px | 16px | 0 16px | 弹窗底部、表单提交 |

## 状态

| 状态 | Primary | Default | Danger |
|------|---------|---------|--------|
| Default | bg: `primary-600` | bg: white, border: `neutral-300` | bg: `status-error` |
| Hover | bg: `primary-700` | border: `primary-600` | bg: `#FF4D4F` |
| Active | bg: `primary-800` | border: `primary-700` | bg: `#D9363E` |
| Disabled | bg: `primary-300` | bg: `neutral-100`, color: `neutral-400` | bg: `#FFCCC7` |
| Loading | bg: `primary-600` + 旋转图标 | 同上 + 旋转图标 | 同上 |

## 变体
- **图标 + 文字**: 图标在文字左侧，间距 `space-1` (4px)
- **纯图标**: 正方形按钮，需配合 Tooltip
- **块级按钮**: `width: 100%`，用于弹窗底部

## OMC 使用场景
- 筛选栏: 搜索(Primary) + 重置(Default)
- 工具栏: 新建(Primary) + 导出(Default) + 批量操作(Default dropdown)
- 弹窗底部: 确认(Primary) + 取消(Default)
- 表格行内: 编辑(Link) + 删除(Link Danger)
