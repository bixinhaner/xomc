# 字体排版 (Typography)

## 字体族 (Font Family)

### 主字体（无衬线，中文优先）

```css
font-family: -apple-system, BlinkMacSystemFont,
  'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei',
  'Helvetica Neue', Helvetica, Arial, sans-serif;
```

### 等宽字体（代码、命令、序列号）

```css
font-family: 'SFMono-Regular', Consolas, 'Liberation Mono',
  Menlo, Courier, monospace;
```

**等宽字体使用场景**: MML 命令、设备 SN、IP 地址、JSON 日志内容、参数路径、告警码

## 字号 (Font Size)

基准: `14px` (0.875rem)

| Token | rem | px | 用途 |
|-------|-----|-----|------|
| `font-xs` | 0.75 | 12 | 辅助文字、帮助文字、徽标计数、表格次要信息 |
| `font-sm` | 0.8125 | 13 | 紧凑表格内容、时间戳文字 |
| `font-base` | 0.875 | **14** | **默认** — 正文、表单标签、表格单元格、导航项、按钮 |
| `font-md` | 1.0 | 16 | 区段标题、卡片标题、弹窗副标题 |
| `font-lg` | 1.125 | 18 | 页面副标题、突出标签 |
| `font-xl` | 1.25 | 20 | 页面标题 |
| `font-2xl` | 1.5 | 24 | Dashboard KPI 当前值 |
| `font-3xl` | 2.0 | 32 | Dashboard 大数字 |
| `font-4xl` | 2.5 | 40 | Dashboard 特大数字（极少使用） |

## 字重 (Font Weight)

| Token | 值 | 用途 |
|-------|-----|------|
| `font-regular` | 400 | 正文、表格单元格、表单输入 |
| `font-medium` | 500 | 标签、导航项、副标题、标签文字 |
| `font-semibold` | 600 | 区段标题、按钮文字、卡片标题、表头 |
| `font-bold` | 700 | 页面标题、KPI 数值、强调标题 |

## 行高 (Line Height)

| Token | 值 | 用途 |
|-------|-----|------|
| `leading-tight` | 1.25 | 大标题（font-2xl 及以上） |
| `leading-compact` | 1.375 | 表格行、密集列表、紧凑徽标 |
| `leading-normal` | 1.5 | **默认** — 正文、表单、标签 |
| `leading-relaxed` | 1.75 | 描述段落、帮助文字、Tooltip 内容 |

## 排版组合示例

### 页面标题
- 字号: `font-xl` (20px)
- 字重: `font-bold` (700)
- 颜色: `neutral-800`
- 行高: `leading-tight`

### 区段标题
- 字号: `font-md` (16px)
- 字重: `font-semibold` (600)
- 颜色: `neutral-800`
- 行高: `leading-normal`

### 表格表头
- 字号: `font-base` (14px)
- 字重: `font-semibold` (600)
- 颜色: `neutral-800`
- 行高: `leading-compact`
- 背景: `neutral-100`

### 表格内容
- 字号: `font-base` (14px)
- 字重: `font-regular` (400)
- 颜色: `neutral-600`
- 行高: `leading-compact`

### 表单标签
- 字号: `font-base` (14px)
- 字重: `font-regular` (400)
- 颜色: `neutral-700`
- 行高: `leading-normal`

### KPI 大数字
- 字号: `font-2xl` ~ `font-3xl` (24-32px)
- 字重: `font-bold` (700)
- 颜色: `neutral-800`
- 行高: `leading-tight`

### 辅助/帮助文字
- 字号: `font-xs` (12px)
- 字重: `font-regular` (400)
- 颜色: `neutral-500`
- 行高: `leading-relaxed`
