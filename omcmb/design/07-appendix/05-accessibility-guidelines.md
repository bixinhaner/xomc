# 无障碍设计指南 (Accessibility Guidelines)

> OMC 统一网管系统 - 附录

## 概述

无障碍设计 (Accessibility, 简称 a11y) 确保 OMC 系统可被所有用户使用，包括依赖键盘操作、使用屏幕阅读器或有视觉障碍的用户。本指南基于 WCAG 2.1 AA 级标准制定。

---

## 1. 键盘导航 (Keyboard Navigation)

### 1.1 基本原则

```
核心规则:
  - 所有可交互元素必须可通过键盘访问 (focusable)
  - Tab 顺序必须逻辑合理 (遵循视觉布局从上到下、从左到右)
  - 不能存在键盘陷阱 (keyboard trap) — 用户必须能 Tab 出任何组件
  - 自定义组件必须实现正确的键盘交互
```

### 1.2 可交互元素

| 元素 | 原生可聚焦 | 说明 |
|------|-----------|------|
| `<button>` | 是 | 按钮 |
| `<a href>` | 是 | 链接 |
| `<input>` | 是 | 输入框 |
| `<select>` | 是 | 下拉框 |
| `<textarea>` | 是 | 文本域 |
| `<div>` (自定义) | 否 | 需添加 `tabIndex={0}` 和 `role` |

### 1.3 Tab 顺序规范

```
推荐:
  - 使用自然 DOM 顺序 (不使用 tabIndex > 0)
  - 仅使用 tabIndex={0} (加入 Tab 序列) 和 tabIndex={-1} (编程聚焦)
  - 不使用 tabIndex > 0 (会打乱自然顺序)

OMC 页面 Tab 顺序:
  1. Header (Logo → 模块 Tab → 搜索 → 告警 → 任务 → 用户)
  2. Sidebar (菜单项从上到下)
  3. Content (面包屑 → 工具栏 → 筛选栏 → 表格 → 分页)
```

### 1.4 焦点陷阱 (Focus Trap)

```
Modal / Drawer 打开时:
  - 焦点限制在弹窗内 (不能 Tab 到弹窗外)
  - 使用 focus-trap-react 或类似库实现
  - 打开时: 焦点移到弹窗首个可聚焦元素
  - 关闭时: 焦点回到触发元素

实现:
  Tab 到最后一个元素 → 跳回第一个元素 (循环)
  Shift+Tab 到第一个元素 → 跳到最后一个元素 (循环)
```

---

## 2. 焦点指示器 (Focus Indicators)

### 2.1 视觉规范

```
焦点样式 (键盘导航时):
  outline: 2px solid #1890FF
  outline-offset: 2px

按钮焦点:
  ┌──────────────────┐
  │ ┌──────────────┐ │  ← 2px 蓝色外框, 偏移 2px
  │ │   保 存      │ │
  │ └──────────────┘ │
  └──────────────────┘

输入框焦点:
  border-color: #1890FF
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2)
```

### 2.2 :focus-visible

```css
/* 只在键盘导航时显示焦点指示 */
:focus-visible {
  outline: 2px solid #1890FF;
  outline-offset: 2px;
}

/* 鼠标点击不显示焦点指示 */
:focus:not(:focus-visible) {
  outline: none;
}

/* 高对比度模式适配 */
@media (forced-colors: active) {
  :focus-visible {
    outline: 2px solid CanvasText;
  }
}
```

### 2.3 禁用与跳过

```
规则:
  - disabled 元素: 不参与 Tab 序列
  - 装饰性元素: tabIndex={-1}
  - Skip Navigation: 页面顶部提供 "跳转到主要内容" 隐藏链接

Skip Link:
  <a href="#main-content" class="skip-link">
    跳转到主要内容
  </a>
  (平时隐藏, Tab 到时显示)
```

---

## 3. 颜色无障碍 (Color Accessibility)

### 3.1 不仅依赖颜色

```
核心规则: 信息传达不能仅靠颜色区分, 必须辅以文字或图标

告警级别 (正确做法):
  🔴 紧急 (Critical)    ← 颜色 + 图标 + 文字
  🟠 重要 (Major)       ← 颜色 + 图标 + 文字
  🟡 一般 (Minor)       ← 颜色 + 图标 + 文字
  🔵 提示 (Warning)     ← 颜色 + 图标 + 文字

告警级别 (错误做法):
  ● ● ● ●               ← 仅靠颜色区分, 色盲用户无法辨识

设备状态 (正确做法):
  ● 在线 (绿色圆点 + 文字)
  ● 离线 (红色圆点 + 文字)
  ● 未知 (灰色圆点 + 文字)

表单错误 (正确做法):
  红色边框 + 红色错误文字 + ⚠ 图标
  (不仅仅是红色边框)
```

### 3.2 对比度要求

```
WCAG 2.1 AA 级对比度要求:

普通文本 (< 18px 或 < 14px bold):
  最低对比度: 4.5:1

大文本 (>= 18px 或 >= 14px bold):
  最低对比度: 3:1

非文本元素 (图标、边框、控件):
  最低对比度: 3:1

OMC 系统颜色对比度验证:
  | 前景色 | 背景色 | 对比度 | 合规 |
  |--------|--------|--------|------|
  | #262626 (正文) | #FFFFFF | 14.7:1 | ✓ |
  | #8C8C8C (辅助) | #FFFFFF | 3.9:1  | ✓ (大文本) |
  | #1890FF (链接) | #FFFFFF | 3.5:1  | ✓ (大文本) |
  | #FF4D4F (错误) | #FFFFFF | 4.0:1  | ✓ (大文本) |
  | #52C41A (成功) | #FFFFFF | 3.2:1  | ✓ (大文本) |
  | #FFFFFF (按钮) | #1890FF | 3.5:1  | ✓ (大文本) |

工具推荐:
  - Chrome DevTools (Inspect → Accessibility)
  - WebAIM Contrast Checker
  - Axe DevTools 浏览器插件
```

---

## 4. 屏幕阅读器支持 (Screen Reader)

### 4.1 ARIA 标签

```html
<!-- 图标按钮必须有 aria-label -->
<button aria-label="关闭弹窗">
  <CloseOutlined />
</button>

<!-- 纯图标链接 -->
<a href="/alarm" aria-label="查看当前告警">
  <BellOutlined />
  <span class="badge">3</span>
</a>

<!-- 搜索输入框 -->
<input
  type="text"
  placeholder="搜索设备..."
  aria-label="搜索设备名称或IP地址"
/>

<!-- 加载状态 -->
<div aria-live="polite" aria-busy={loading}>
  {loading ? '数据加载中...' : null}
</div>
```

### 4.2 Role 属性

```html
<!-- 自定义导航 -->
<nav role="navigation" aria-label="主导航">
  <ul role="menubar">
    <li role="menuitem">设备管理</li>
    <li role="menuitem">告警管理</li>
  </ul>
</nav>

<!-- 自定义 Tab -->
<div role="tablist" aria-label="报表视图切换">
  <button role="tab" aria-selected="true">表格</button>
  <button role="tab" aria-selected="false">图表</button>
</div>
<div role="tabpanel">...</div>

<!-- 表格 -->
<table role="grid" aria-label="设备列表">
  <caption class="sr-only">设备列表, 共 156 条记录</caption>
  ...
</table>

<!-- 告警通知 -->
<div role="alert" aria-live="assertive">
  新告警: eNodeB-001 CELL_UNAVAILABLE
</div>
```

### 4.3 动态内容更新

```html
<!-- aria-live 用于动态更新区域 -->

<!-- 告警计数更新 (不打断用户) -->
<span aria-live="polite">
  当前告警: 紧急 2 条, 重要 15 条
</span>

<!-- 紧急通知 (立即朗读) -->
<div role="alert" aria-live="assertive">
  网络连接已断开
</div>

<!-- 表格数据更新 -->
<div aria-live="polite">
  搜索结果: 找到 23 条记录
</div>
```

### 4.4 隐藏装饰性内容

```html
<!-- 装饰性图标 (屏幕阅读器跳过) -->
<span aria-hidden="true">
  <DecorationIcon />
</span>

<!-- 仅屏幕阅读器可见 (视觉隐藏) -->
<span class="sr-only">
  当前页码: 第 3 页, 共 8 页
</span>

/* sr-only CSS */
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
```

---

## 5. 表单无障碍 (Form Accessibility)

### 5.1 Label 关联

```html
<!-- 正确: label 通过 htmlFor 关联 -->
<label htmlFor="deviceName">设备名称</label>
<input id="deviceName" type="text" />

<!-- 或使用 aria-label -->
<input type="text" aria-label="设备名称" />

<!-- 或使用 aria-labelledby -->
<span id="nameLabel">设备名称</span>
<input type="text" aria-labelledby="nameLabel" />
```

### 5.2 错误提示关联

```html
<!-- 错误消息通过 aria-describedby 关联 -->
<label htmlFor="ipAddr">IP 地址</label>
<input
  id="ipAddr"
  type="text"
  aria-invalid="true"
  aria-describedby="ipAddr-error"
/>
<span id="ipAddr-error" role="alert">
  IP 地址格式不正确
</span>
```

### 5.3 必填字段

```html
<!-- 使用 aria-required -->
<label htmlFor="name">
  <span aria-hidden="true" class="required-star">*</span>
  设备名称
</label>
<input id="name" type="text" aria-required="true" />
```

---

## 6. 动画与运动 (Motion)

### 6.1 尊重用户偏好

```css
/* 尊重系统级 "减少动画" 设置 */
@media (prefers-reduced-motion: reduce) {
  /* 禁用所有过渡动画 */
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }

  /* Skeleton 不闪烁 */
  .skeleton-shimmer {
    animation: none;
  }

  /* Modal 不缩放 */
  .modal-enter {
    transform: none;
    opacity: 1;
  }
}
```

### 6.2 自动播放限制

```
规则:
  - 不自动播放视频或音频
  - 告警声音默认关闭, 由用户主动开启
  - 不使用持续闪烁的动画 (可能引发光敏性癫痫)
  - 闪烁频率不超过 3 次/秒 (WCAG 2.3.1)
```

---

## 7. OMC 特定无障碍要求

### 7.1 告警系统

```
- 告警级别: 颜色 + 图标 + 文字三重标识
- 新告警到达: aria-live="polite" 通知屏幕阅读器
- 紧急告警: aria-live="assertive" 立即朗读
- 告警声音: 可由用户配置开启/关闭
- 告警列表: 表格有 caption 和 aria-label
```

### 7.2 表格

```
- 每个表格有 aria-label 描述其用途
- 排序状态: aria-sort="ascending" / "descending" / "none"
- 选中状态: aria-selected="true"
- 展开状态: aria-expanded="true" / "false"
- 分页: aria-label 描述当前页和总页数
```

### 7.3 树形控件

```
- role="tree" 和 role="treeitem"
- 展开/折叠: aria-expanded
- 层级: aria-level
- 选中: aria-selected
- 多选: aria-checked (三态: true/false/mixed)
```

### 7.4 拓扑图

```
- 提供文字替代描述 (aria-label 或伴随表格)
- 拓扑图不能是唯一的数据查看方式
- 提供列表/表格视图作为替代
```

---

## 8. 测试与验证

### 8.1 自动化测试

```
工具:
  - axe-core: 自动检测常见无障碍问题
  - eslint-plugin-jsx-a11y: JSX 中的 a11y 规则
  - @testing-library/react: 鼓励按角色查询 (getByRole)

集成到 CI:
  - 每次 PR 运行 axe 检测
  - 检测到 Critical 或 Serious 问题阻止合并
```

### 8.2 手动测试

```
键盘测试:
  - 拔掉鼠标, 仅用键盘完成所有操作
  - 检查 Tab 顺序是否合理
  - 检查焦点指示器是否可见
  - 检查 Modal 内焦点是否被正确限制

屏幕阅读器测试:
  - macOS: VoiceOver (内置)
  - Windows: NVDA (免费)
  - 检查所有控件是否被正确朗读
  - 检查动态更新是否被通知

色盲模拟:
  - Chrome DevTools → Rendering → Emulate vision deficiencies
  - 检查信息是否仅通过颜色传达
```

### 8.3 验证清单

```
☐ 所有可交互元素可通过键盘访问
☐ Tab 顺序逻辑合理
☐ 焦点指示器清晰可见
☐ Modal 使用焦点陷阱
☐ 所有图片有 alt 文本 (或 aria-hidden)
☐ 所有图标按钮有 aria-label
☐ 表单 label 正确关联
☐ 错误消息通过 aria-describedby 关联
☐ 颜色对比度满足 WCAG AA
☐ 信息不仅靠颜色传达
☐ 动态内容使用 aria-live
☐ 尊重 prefers-reduced-motion
☐ 表格有正确的 ARIA 属性
☐ Skip Navigation 链接存在
```
