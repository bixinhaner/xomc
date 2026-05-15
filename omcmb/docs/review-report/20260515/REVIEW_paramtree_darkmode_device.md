---
date: 2026-05-15
author: shangyingbin
scope: device
type: fix
files:
  - omcmb/webcode/src/pages/device/DeviceDetail/ParameterTreeTab/ObjectTreePanel.css
  - omcmb/webcode/src/pages/device/DeviceDetail/ParameterTreeTab/ObjectTreePanel.tsx
  - omcmb/webcode/src/pages/device/DeviceDetail/ParameterTreeTab/index.tsx
verdict: PASS
---

# Review — 设备参数树 ObjectTreePanel 暗黑模式适配

## 背景

设备详情页 → 参数树 Tab 切换到暗色主题（`tech` / `cyberpunk`）后，
左侧"对象树"面板仍保持白底深色文字，与页面整体暗色调冲突。

## 根因

- ObjectTreePanel.css 内 ~20 处硬编码浅色 hex（背景 `#fff`、`#fafafa`、`#f5f5f5`、`#e6f4ff`；文字 `#262626`、`#8c8c8c`、`#bfbfbf` 等）
- `ParameterTreeTab/index.tsx` 外层 wrapper 内联写死 `background: '#fff'` 与 `border: '1px solid #e8e8e8'`
- `ObjectTreePanel.tsx` 内 `<mark>` 高亮与 `CaretDown/RightOutlined` 内联色硬编码
- 项目未启用 `cssVar: true`，AntD token 不暴露为 CSS 变量，plain CSS 文件无法直接消费 token

页面其他部分由 AntD 组件（Card / Tabs / Table / Tree）渲染，
自动消费 token 因此跟随主题——只剩此处仍是浅色。

## 修复方案

走 **方案 A：局部 useToken + CSS 变量桥接**。

- `ParameterTreeTab/index.tsx`：用 `theme.useToken()` 在外层 wrapper 注入一组 `--otp-*` CSS 变量到 inline style（涵盖背景 / 边框 / 文字 / 主色 / 成功 / 危险 / 滚动条），变量值取自 AntD 语义 token。同时 wrapper 自身的 background / border 改为消费 token。
- `ObjectTreePanel.css`：所有硬编码色值改为 `var(--otp-xxx, <原浅色>)`。保留原浅色作为 fallback 以确保浅色主题下零行为变化。
- `ObjectTreePanel.tsx`：
  - `highlightText()` 改为 `HighlightText` 组件，搜索高亮 `<mark>` 用 `token.colorWarningBg/Text`
  - 移除 CaretDown/RightOutlined 内联 `color: '#8c8c8c'`，让图标继承父级 `.virtuoso-tree-switcher` 的 `var(--otp-text-disabled)`

## 变量到 Token 映射

| CSS 变量 | AntD Token | 浅色 fallback |
|----------|------------|---------------|
| `--otp-bg` | `colorBgContainer` | `#fff` |
| `--otp-bg-soft` | `colorFillAlter` | `#fafafa` |
| `--otp-bg-hover` | `colorBgTextHover` | `#f5f5f5` |
| `--otp-bg-selected` | `controlItemBgActive` | `#e6f4ff` |
| `--otp-border` | `colorBorderSecondary` | `#f0f0f0` |
| `--otp-border-soft` | `colorSplit` | `#f5f5f5` |
| `--otp-text` | `colorText` | `#262626` |
| `--otp-text-secondary` | `colorTextSecondary` | `#8c8c8c` |
| `--otp-text-disabled` | `colorTextTertiary` | `#bfbfbf` |
| `--otp-primary` | `colorPrimary` | `#1677ff` |
| `--otp-primary-border` | `colorPrimaryBorder` | `#bae0ff` |
| `--otp-success` / `--otp-success-bg` | `colorSuccess` / `colorSuccessBg` | `#52c41a` / `#f6ffed` |
| `--otp-error` / `--otp-error-bg` | `colorError` / `colorErrorBg` | `#ff4d4f` / `#fff2f0` |
| `--otp-scrollbar` | `colorFill` | `#d9d9d9` |

## 审查清单

### 前端规范

- [x] **类型安全**：无 `any`；CSS 变量字面量索引使用 `[`--otp-bg` as string]: ...` 通过 TS 类型校验
- [x] **Hook 规则**：`useMemo(token)` 依赖完整，新增 React 顶层 `useMemo` 引入未引发 lint 警告
- [x] **token 来源唯一**：仅使用 AntD 官方语义 token（`theme.useToken()`），不混用项目自定义 token
- [x] **fallback 保底**：所有 CSS var 带 fallback，浅色主题（classic/fresh/minions/rmb/tiffany）零行为变化
- [x] **多皮肤兼容**：5 浅 + 2 暗共 7 个主题均自动适配；改动仅在 webcode UI 壳，不影响 frontend-core
- [x] **CSS 选择器无副作用**：仅修改色值，未改 layout / 选择器结构
- [x] **无新增依赖**：100% 复用 antd 已用 API
- [x] `tsc --noEmit` 通过；`npm run build` 通过

### 通用

- [x] 无硬编码 SN / device id 等业务数据
- [x] 无敏感信息泄漏
- [x] 无破坏性变更

## 真机验证

- 浏览器导航 `/device/detail/1202000240194DP0026 → 参数树` Tab
- 当前主题（暗色）下截图显示对象树面板背景、边框、文字、节点 hover 均与页面整体协调
- 浅色主题下回归：因 CSS var fallback 等同于原 hex，外观与修复前一致

## 结论

PASS。无 CRITICAL / WARNING。

## Out of Scope

- ChildParamTable / SyncStatusBar / ParameterEditModal 仅复用 AntD 组件，已通过 token 自动适配，本次无需改动
- 其他模块（告警、配置、运维）如有类似硬编码色值问题不在本次范围内；若发现可作为后续 hotfix 单独修复
