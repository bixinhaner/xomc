# Code Review — T-0155 设备详情 tab 共享槽位 + URL 内部 tab 状态

- **日期**：2026-05-19
- **范围**：`omcmb/frontend-core/src/store/tabStore.ts` / `types/common.ts` / `omcmb/webcode/src/components/Layout/TabBar/{TabItem,index}.tsx` / `omcmb/webcode/src/pages/device/DeviceDetail/index.tsx`
- **作者**：Claude
- **Reviewer**：Claude（self-review，前端专家视角）
- **关联 Task**：T-0155

---

## 变更概要

1. `useTabStore.openTab` 命中同 key 时合并 `path/label/labelRaw/closable`，支持"同槽位复用显示不同记录详情"（如所有设备详情共用 `device-detail` 这个 tab key，按 SN 替换内容）
2. `TabItem` 接口新增可选字段 `labelRaw?: boolean`，标记 label 已本地化、跳过 `react-intl` 翻译，避免"label 含动态 SN/名称"时报 missing translation
3. `TabBar/TabItem.tsx` 与 `TabBar/index.tsx` overflow 菜单两处渲染时识别 `labelRaw`
4. `DeviceDetail/index.tsx`：内部 tab `activeTab` 从 `useState('basic')` 改为 URL `?tab=` 单一真相源；新增 useEffect 注册 `device-detail` 共享 tab 到 TabBar，path 带 `location.search` 保留深链接
5. quickSettings 内部 tab 加 `forceRender: true` 保活子组件状态（与 T-0156 协同；纯内部 tab 切换不卸载组件）

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| 类型安全 | 新增字段 `labelRaw?: boolean` 可选，向后兼容 | ✅ |
| 类型安全 | 全程无 `any` / `interface{}` | ✅ |
| 业务层归属 | API/Hook/Store/Types 在 `frontend-core/`，UI 在 `webcode/` | ✅ |
| 单一真相源 | 内部 tab 状态去 useState → URL，刷新 / 卸载不丢 | ✅ |
| 多皮肤影响 | `frontend-core` 的 TabItem / tabStore 改动是新增可选字段 + 行为兼容 | ✅（webcode-v2 / v3 无破坏性影响） |
| i18n | `labelRaw` 设计正确支持已本地化的动态拼接（如 `${t('common.detail')} · ${SN}`） | ✅ |
| 路由 / 跳转 | `useSearchParams({ replace: true })` 不污染历史栈 | ✅ |

---

## 发现

### CRITICAL
- 无

### WARNING
- W-01：`DeviceDetail.useEffect` 的依赖项含 `t`（来自 `useT()`）。`useT` 包装 `useIntl().formatMessage`，多数 react-intl 实现下函数引用稳定，但**不绝对**。最坏情况：每次 render `t` 都是新引用 → `useEffect` 反复 fire → 每次都调一次 `openTab`。
  - **影响**：`openTab` 内部对同 key 做 merge（已在本次改动中加入），不会真正插入重复 tab，只会重写一遍 store。性能损耗微弱，无功能 bug。
  - **建议**：后续若性能 profiling 发现热路径，可去掉 `t` 依赖，改用 ref 缓存上一次 label 字符串。本次接受现状。

### INFO
- I-01：`forceRender: true` 给 quickSettings 内部 tab 是 T-0148/T-0146 引入的同步性优化，本 commit 顺带保留；与 T-0156 store 持久化方案配合使用——store 是"跨页面卸载"兜底，`forceRender` 是"同页面内部 tab 切换"优化，两者并存合理。
- I-02：`openTab` 合并的注释清晰说明了"复用槽位"语义；将来若新增 tab 类型（如 `topology-detail` 也共享一个槽位）可复用此模式。

---

## 测试

- Playwright 跑过设备详情 tab 共享槽位场景（先前会话）：进入不同 SN 复用同一个 tab、URL `?tab=quickSettings` 深链接生效、刷新页面 URL 还原 — 全通过
- `npm run typecheck` 通过

---

## 结论

**PASS_WITH_WARNINGS**（W-01 为可观测性建议，不阻塞合入）

可合入。
