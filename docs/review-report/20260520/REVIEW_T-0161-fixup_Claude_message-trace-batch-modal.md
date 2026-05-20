# 代码审查报告 — T-0161 fixup: MessageTrace 批量删除 Modal.confirm 修复

| 字段 | 值 |
|------|------|
| 任务 | T-0161 followup（批量删除 Modal 不显示） |
| 范围 | frontend (webcode/pages/ops/MessageTrace) |
| 改动量 | 1 文件 |
| 审查结论 | **PASS** |

---

## 1. 变更概览

T-0161 主体（commit `3803a4ce`）实现批量删除时用了 antd 静态 API `Modal.confirm({...})`。浏览器实测发现点击批量删除按钮 → 任务 selectedKeys 正常 → handler 触发 → 但 Modal 没渲染。

**根因**：antd 5 + React 19 严格模式下，从 `import { Modal } from 'antd'` 直接调用 `Modal.confirm()` 拿不到 ConfigProvider 上下文（含 theme token / locale / messageContext），导致 Modal 内部 render 时早期 bail-out。Antd 5 已废弃静态 API 通过 `<App>` wrapper 提供的 `App.useApp()` hook 替代。

**修复**：
- import 加 `App`，删除静态 `message` import
- 组件顶部 `const { modal, message } = App.useApp();`
- `Modal.confirm({...})` → `modal.confirm({...})`
- 现有所有 `message.success/warning/error/info` 调用自动走 hook 实例（变量名同名，不需改 call site）

Popconfirm 是组件式（不依赖静态 API），单删行内 Popconfirm 一直正常工作不受影响。

## 2. 审查检查项

| 项 | 结果 | 备注 |
|----|------|------|
| 前端 typecheck | ✅ | `npm run typecheck` PASS |
| `<App>` wrapper 存在 | ✅ | `providers/ThemeProvider.tsx` 已 wrap `<App>` |
| `<Modal>` JSX 组件仍 import | ✅ | 创建任务/查看报文等 dialog 仍用 `<Modal>` JSX 形式 |
| message.* call sites 同名透传 | ✅ | 不需改 8 处 message.success/error/info/warning |
| 行内 Popconfirm 单删未受影响 | ✅ | 浏览器实测 popconfirm OK，删 1 行 audit_logs 落 trace_delete |

## 3. 部署验证（已完成）

| 测试 | 结果 |
|------|------|
| 勾 2 个任务 → 批量删除按钮启用 | ✅ |
| 点击 → Modal "批量删除"标题 + "将删除已选中的 2 个任务及其全部报文，且不可恢复，确认？" 展示 | ✅ |
| 确认 → message "删除完成：成功 2 / 失败 0" | ✅ |
| DB trace_tasks 22 → 20 | ✅ |
| audit_logs 新增 trace_delete 记录 | ✅ |

---

**审查人**：Claude Opus 4.7  
**日期**：2026-05-20  
**结论**：PASS
