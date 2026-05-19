# Code Review — T-0157 C9 消息中心 UI（铃铛 Popover + 列表）

- **日期**：2026-05-19
- **范围**：`webcode/components/Layout/Header/index.tsx`（铃铛 Popover 接入 + Badge unreadCount）/ `webcode/components/NotificationCenter/index.tsx`（新组件含 NotificationCenter + NotificationItem + uiVisuals）
- **作者**：Claude
- **Reviewer**：Claude（self-review，前端视角）
- **关联**：T-0157 Phase 1 sub-task **C9**

---

## 变更概要

1. **Header 铃铛**：从占位 `<button>` 改为 antd `<Popover trigger="click">`；Badge count 由 `useNotificationUnreadCount()` 驱动（10s 轮询）；overflowCount=99 → "99+"
2. **NotificationCenter 组件**：
   - 头部：标题"消息中心" + "全部已读"+"清空" 两个文字按钮（disabled when empty 或 busy）
   - 列表：antd `<List>` 渲染 useNotificationCenter 返回的 items；最大 440px 高度内滚动；空态 "暂无消息"
   - 单条 NotificationItem：5 态视觉（蓝旋/绿√/红×/黄⏰/灰⊘）+ 未读左侧蓝条+加粗 + 已读灰色 + 详情区折叠 3 行 + 时间 Tooltip 显示绝对时间
3. **点击行为**：mark-read + navigate(linkTo) + 关闭 Popover
4. **i18n**：本期硬编码中文（§11 L-03 范围外；O-03 后端 locale 渲染未触及前端的"消息中心 / 全部已读 / 清空"等 UI 文案）

---

## 出口门验证

| 门 | 结果 |
|---|---|
| `omcmb/webcode npm run typecheck` | ✅ tsc --noEmit 通过 |
| dayjs relativeTime 插件 + zh-cn locale 已 import | ✅ |
| 5 态视觉映射函数 uiVisuals 与 §4.4 表对齐 | ✅ |

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| 业务层归属 | UI 在 webcode，数据消费走 @core/hooks/api/useNotificationCenter | ✅ |
| antd 一致性 | Popover / Badge / List / Empty / Button / Tooltip 全部用 antd 5 既有组件 | ✅ |
| 性能 | List 数据 ≤20 条无需虚拟化；Popover 关闭后 query 仍在缓存（不重 fetch） | ✅ |
| 受控开关 | notifOpen state 让点击条目可以从内部关闭 Popover | ✅ |
| Empty 状态 | items.length === 0 时显示 Empty 占位；按钮也 disabled | ✅ |
| 错误处理 | markAll / clearAll 异常用 message.error；markRead 静默（用户跳转优先） | ✅ |
| 已读视觉 | 未读：左侧蓝条 + 加粗 + 白底；已读：灰底 + 不加粗 + opacity 0.75 | ✅ |
| 时间显示 | 相对时间 + Tooltip 绝对时间（hover 可见精确时刻） | ✅ |

---

## 发现

### CRITICAL / WARNING
- 无

### INFO
- I-01：副标题最初一稿带了"系统 / 设备 / 相对时间"拼接，含两段无用的三元，已简化为只显示相对时间（标题已含"设备 SN"）。
- I-02：dayjs zh-CN locale 全局 import 后会污染其他 dayjs 用户。但 webcode 其他位置如 alarm/log 也希望中文相对时间，副作用是预期。如需改 en-US，用户切语言时 i18n 层会 re-format（V2 优化）。
- I-03：消息条目左侧蓝条 + 加粗的未读视觉来自 §4.2 设计，已实现。`Item.tsx` 没单独拆文件 — 内联在 index.tsx 让 100 行内的内聚组件不分散；后续维护超出 200 行再拆。
- I-04：mock 模式下 4 条预填消息（queued/completed/failed/expired）覆盖五态主要分支，开发期视觉一目了然。
- I-05：clearAll 没加二次确认对话框。一键清空对用户来说不可逆，但符合 antd 文字按钮的常规行为（带 `danger` 红色 + 文字明确）。如需 Popconfirm 防误点，在用户反馈后再加。

---

## 与设计对齐

| §4.2 项 | 实施 |
|---|---|
| 360px × 480px Popover | ✅ width=360, maxHeight=440 给头部留 40 |
| 头部"全部已读"/"清空" | ✅ |
| 列表按 createdAt 倒序 | ✅ useNotificationCenter sortField=created_at,sortOrder=descend |
| 空态"暂无消息" | ✅ |
| 状态图标 5 色 | ✅ uiVisuals |
| 未读蓝条+加粗 / 已读灰色 | ✅ |
| 点击 markRead + navigate | ✅ |

---

## 结论

**PASS**

可合入，可继续 C10（快速设置 mutation 触发 invalidate + Tag 改读 useDeviceTaskStatus，端到端 E2E）。
