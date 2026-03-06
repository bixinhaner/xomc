# 设计 Token 汇总 (Design Tokens Reference)

所有设计 Token 的快速查找表。详细说明请参考各子文件。

## 色彩 Token

| Token | 值 | CSS 变量 | 用途 |
|-------|-----|---------|------|
| `primary-50` | `#EBF5FF` | `--color-primary-50` | 选中行背景 |
| `primary-100` | `#D6EBFF` | `--color-primary-100` | 浅色强调背景 |
| `primary-200` | `#ADD6FF` | `--color-primary-200` | 悬浮态背景 |
| `primary-300` | `#85C1FF` | `--color-primary-300` | 禁用态主色 |
| `primary-400` | `#5CACFF` | `--color-primary-400` | 链接悬浮色 |
| `primary-500` | `#3396FF` | `--color-primary-500` | 活动链接色 |
| `primary-600` | `#1677FF` | `--color-primary-600` | **主操作色** |
| `primary-700` | `#0958D9` | `--color-primary-700` | 主按钮悬浮 |
| `primary-800` | `#003EB3` | `--color-primary-800` | 主按钮按下 |
| `primary-900` | `#002C8C` | `--color-primary-900` | 深色强调 |
| `neutral-50` | `#FAFAFA` | `--color-neutral-50` | 页面背景 |
| `neutral-100` | `#F5F5F5` | `--color-neutral-100` | 卡片背景、斑马纹 |
| `neutral-200` | `#F0F0F0` | `--color-neutral-200` | 分割线 |
| `neutral-300` | `#D9D9D9` | `--color-neutral-300` | 禁用态边框 |
| `neutral-400` | `#BFBFBF` | `--color-neutral-400` | 禁用态文字 |
| `neutral-500` | `#8C8C8C` | `--color-neutral-500` | 次要文字 |
| `neutral-600` | `#595959` | `--color-neutral-600` | 正文文字 |
| `neutral-700` | `#434343` | `--color-neutral-700` | 强调正文 |
| `neutral-800` | `#262626` | `--color-neutral-800` | 标题文字 |
| `neutral-900` | `#1F1F1F` | `--color-neutral-900` | 侧边栏背景 |
| `neutral-950` | `#141414` | `--color-neutral-950` | 侧边栏最深 |
| `severity-critical` | `#F5222D` | `--color-severity-critical` | 紧急告警 |
| `severity-major` | `#FA8C16` | `--color-severity-major` | 主要告警 |
| `severity-minor` | `#FAAD14` | `--color-severity-minor` | 次要告警 |
| `severity-warning` | `#1890FF` | `--color-severity-warning` | 警告告警 |
| `status-success` | `#52C41A` | `--color-status-success` | 在线/成功 |
| `status-error` | `#F5222D` | `--color-status-error` | 离线/失败 |
| `status-processing` | `#1677FF` | `--color-status-processing` | 进行中 |
| `status-warning` | `#FA8C16` | `--color-status-warning` | 异常 |
| `status-inactive` | `#8C8C8C` | `--color-status-inactive` | 未知/禁用 |

## 字体 Token

| Token | 值 | CSS 变量 | 用途 |
|-------|-----|---------|------|
| `font-xs` | `12px` | `--font-size-xs` | 辅助文字、徽标 |
| `font-sm` | `13px` | `--font-size-sm` | 紧凑表格、时间戳 |
| `font-base` | `14px` | `--font-size-base` | **默认正文** |
| `font-md` | `16px` | `--font-size-md` | 区段标题 |
| `font-lg` | `18px` | `--font-size-lg` | 页面副标题 |
| `font-xl` | `20px` | `--font-size-xl` | 页面标题 |
| `font-2xl` | `24px` | `--font-size-2xl` | KPI 数值 |
| `font-3xl` | `32px` | `--font-size-3xl` | Dashboard 大数字 |
| `font-4xl` | `40px` | `--font-size-4xl` | 极少使用特大数字 |

## 间距 Token

| Token | 值 | CSS 变量 | 用途 |
|-------|-----|---------|------|
| `space-0.5` | `2px` | `--space-0-5` | 微间距 |
| `space-1` | `4px` | `--space-1` | 图标与文字间距 |
| `space-2` | `8px` | `--space-2` | 表格单元格内边距 |
| `space-3` | `12px` | `--space-3` | 输入框内边距 |
| `space-4` | `16px` | `--space-4` | **标准间距** |
| `space-5` | `20px` | `--space-5` | 表单字段间距 |
| `space-6` | `24px` | `--space-6` | 页面水平内边距 |
| `space-8` | `32px` | `--space-8` | 大区段分隔 |
| `space-10` | `40px` | `--space-10` | Dashboard 卡片间距 |
| `space-12` | `48px` | `--space-12` | Header 高度 |
| `space-16` | `64px` | `--space-16` | 页面外边距 |

## 阴影 Token

| Token | 值 | 用途 |
|-------|-----|------|
| `shadow-none` | `none` | 平面元素 |
| `shadow-sm` | `0 1px 2px rgba(0,0,0,0.03)...` | 卡片、侧边栏 |
| `shadow-md` | `0 3px 6px rgba(0,0,0,0.12)...` | 下拉框、提示 |
| `shadow-lg` | `0 6px 16px rgba(0,0,0,0.08)...` | 弹窗、抽屉 |

## 圆角 Token

| Token | 值 | 用途 |
|-------|-----|------|
| `radius-none` | `0` | 全宽元素 |
| `radius-xs` | `2px` | 标签、小徽标 |
| `radius-sm` | `4px` | **默认**: 输入框、按钮、卡片 |
| `radius-md` | `6px` | 大卡片、面板 |
| `radius-lg` | `8px` | 弹窗 |
| `radius-xl` | `12px` | Dashboard KPI 卡片 |
| `radius-full` | `9999px` | 头像、状态点 |
