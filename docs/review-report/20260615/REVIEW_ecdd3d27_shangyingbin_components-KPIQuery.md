# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-15 22:10 |
| 提交 | ecdd3d27（base，改动未提交） |
| 作者 | shangyingbin |
| 范围 | components-KPIQuery（前端 picker 弹窗） |
| 变更文件数 | 4 |
| 新增行数 | +88 |
| 删除行数 | -1 |
| 关联 Issue | #411 |

## 变更概要

修复「指标查询 → 查询模板」编辑时，指标/设备选择弹窗重开后回显上次残留选择的 bug。根因：两个 picker 组件（MetricPickerModal、DevicePickerModal）在调用页常驻不卸载（Modal 的 `destroyOnHidden` 只销毁弹窗 DOM、不重挂载组件），内部 `selected` 选中态仅首挂载初始化一次，重开时不随新 `initialSelected` 同步。修复：补渲染期幂等同步（监听 `open` 由关变开把 `selected` 重置为最新 `initialSelected`），与文件既有 `labelMap`/`deviceType` 同套路、不引入 useEffect。各补一条红→绿回归测试。

## 审查发现

### 🔴 CRITICAL (严重)

无。

### 🟡 WARNING (警告)

无。

### 🔵 INFO (建议)

- **同类残留（搜索关键词 / 当前页码 / 设备制式）未处理**：两个 picker 的 `keyword`/`page`/`searchDraft`（及 MetricPicker 非锁定态的 `deviceTypeState`）同样跨次打开残留，根因相同。但这些属浏览态、不构成「选错指标/设备」的正确性问题；MetricPicker 的 `deviceTypeState` 残留更是源码注释里有意保留（锁定态走派生值绕过）。本次按最小修复只动 `selected`，未扩面。
- **DeviceGroupPickerModal 同款潜在缺陷但为死代码**：`pages/performance/KPIQuery/components/DeviceGroupPickerModal.tsx` 有完全相同的 `selected` 写法，但全仓零处 JSX 引用（未挂载、不可达），故不在本 PR 修（改死代码属 scope creep），仅记录。

## 详细分析

### `omcmb/webcode/src/components/MetricPickerModal/MetricPickerModal.tsx`

新增渲染期同步块（`selected` useState 之后）：
```tsx
const [prevOpen, setPrevOpen] = useState(open);
if (prevOpen !== open) {
  setPrevOpen(open);
  if (open) setSelected(initialSelected);
}
```
- **渲染期 setState 正确性**：React 允许组件在渲染阶段对自身 setState（会在提交前就地重渲染、丢弃进行中的渲染输出）。`prevOpen !== open` 守卫确保仅在 open 跳变时触发，无限渲染不会发生。✓
- **不覆盖用户进行中勾选**：open 维持 true 期间 `prevOpen===open`，不再 setSelected，用户在弹窗内的勾选/取消不被打断；仅在「打开瞬间」同步一次。✓
- **模式一致性**：与同文件 `seenItems`/`seenIsEn`/`labelMap` 的渲染期幂等同步（第 98–111 行）完全同款，未引入 useEffect，风格一致。✓

### `omcmb/webcode/src/pages/performance/KPIQuery/components/DevicePickerModal.tsx`

同款修复（`selected`/`pasteText` useState 之后插入相同的 `prevOpen` 渲染期同步块）。批量粘贴 `handleBatchPaste` 基于当前 `selected` 合并，打开瞬间重置不影响其逻辑（粘贴发生在 open 期间，此时不再触发同步）。✓

### `*.test.tsx`（两份）

各新增一个 `describe('... 已选回显')`，用 `rerender` 模拟「open → 关闭(不卸载) → 换 initialSelected → 重开」，断言回显最新入参且不残留旧值。实测：撤掉修复时该测试 RED、加回修复 GREEN，原有制式测试（MetricPicker 3 条 / DevicePicker 2 条）不破。

## 业务完整性检查

纯前端组件交互修复，无 handler/service/repository/迁移/路由链路涉及。业务链路完整，无遗漏。

## 业务影响范围检查

改动局限在两个 picker 组件内部状态同步，不改 props 契约、不改 `onConfirm` 回调签名、不改调用方（KPIQuery）。对外行为仅「重开时正确回显」，无破坏性变更。变更范围可控，未发现跨模块影响。

## 前后端一致性检查

本次变更仅涉及前端（webcode v1），不涉及任何 API 路径/请求/响应契约。webcode-v2 无该模板编辑功能、webcode-v3 无该页面，三皮肤铁律满足（bug 仅存于 v1）。无后端同步需求。

## 代码质量回退检查

未发现代码质量回退。无删测试（新增 2 条回归测试）、无移除错误处理、无降级安全、无引入 any、无硬编码替配置、无弱化校验。

## 配套更新提醒

- **文档**: 无需更新（内部组件 bug 修复，不改 API/配置/部署）。
- **单元测试**: 已补充——两 picker 各 1 条回归测试，覆盖重开同步分支，红→绿经实测。
- **端到端测试**: 已做浏览器三段对齐实栈验证（指标 picker：编辑不同模板交替打开回显当前模板指标；设备 picker：编辑后从空主面板重开显示「未选择」不残留）。无新增 REST 端点，e2e_verify.sh 无需改。

## 安全检查

未触认证/授权/敏感数据/SQL/XSS 相关代码。未发现安全问题。

## 性能检查

渲染期 setState 有 `prevOpen` 守卫、仅 open 跳变时触发一次，无额外渲染开销。未发现性能问题。

## 测试覆盖

成功路径（重开回显最新）+ 残留防回归（不显示旧值）两侧均覆盖；原有制式联动测试保持通过。两 picker 测试文件合计 7 条全绿。

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 0 |
| INFO | 2 |

**审查结论**: `PASS`
