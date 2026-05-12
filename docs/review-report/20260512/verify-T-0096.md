# Verify Report — T-0096 (MML script 页面更新弹窗取消产品类型字段)

**Task**: T-0096 — ScriptTask 页面 (`/mml/script`) 新建+更新 Drawer 删 `productType` Form.Item，与 `/mml/console` "保存脚本" 弹窗 (T-0090-a 已删 productTypes) 对齐
**Type**: ref / frontend+F06/mml / P3 / S
**Sprint**: sprint-10 (pull-forward 续 T-0090 后)
**Deps**: T-0090 umbrella ✅ (刚 4/4 闭环)
**Owner**: Claude
**Date**: 2026-05-12

---

## §1. 改动面

| 文件 | 操作 | 行数 |
|------|------|------|
| `omcmb/webcode/src/pages/mml/ScriptTask/index.tsx` | 改 | -1 / -1 / -3 / +2 注释（净 -3 行）|

具体 3 处删除：
1. **L77-79** `TaskForm` interface — 删 `productType: string;` 字段
2. **L349** `openEditDrawer` setFieldsValue — 删 `productType: ''` 初始值
3. **L629-631** Drawer Form 体 — 删 `<Form.Item name="productType">` + 注释 +2 行（productTypeOptions 保留供 filter bar 使用）

---

## §2. Scope 边界守护

**KEEP (out of T-0096 scope)**：
- L156-160 `useDictionary('product_type')` + `productTypeOptions` useMemo — filter bar L275 仍消费（过滤既有任务）
- L271-278 filter bar `deviceType` Select options — 过滤既有 tasks 体验，subtask T-0096 仅"更新弹窗"scope
- mml_tasks 表后端 schema（product_type 列可能仍存在；本任务纯 UI 删除，不动 backend / migrations）

**API payload 检视**：L374-389 submit payload 本就**未传** productType 字段 — 表单 productType 一直是"前端孤儿字段"（创建/更新两路径），删除无后端影响

---

## §3. 出口门检查

| 检查项 | 结果 | 备注 |
|--------|------|------|
| `cd omcmb/webcode && npx tsc --noEmit` | ✅ PASS | 无输出 |
| `npx eslint webcode/src/pages/mml/ScriptTask/index.tsx`（omcmb root） | ⚠️ 3 pre-existing errors / 1 pre-existing warning | scriptFilterFields/handleScriptSearch/handleScriptReset unused vars + useCallback exhaustive-deps — 全部 pre-existing（与我改的 L77-79/349/629 无关），ScriptTask 现状 filter bar 占位但未在 JSX 中渲染 |
| 全工程 `npm run lint` | ⚠️ 225 pre-existing | 与 T-0090-a/b/c/d verify 同基线 |
| 无新增 TODO/FIXME/panic | ✅ | grep 验 |
| 无新增 `any` | ✅ | TaskForm 删字段不引入新类型 |
| 后端 build/test | N/A | FE-only |
| 新端点 R | 0 | 无新路由 |
| E2E claim 增量 E | 0 | subtask 无 e2e 要求；纯 UI 删除 |
| 迁移 up/down | N/A | 无迁移 |
| 新 metric/log 名 | N/A | 无新埋点 |

---

## §4. 子任务验收核销

> subtask T-0096 (backlog.md §4 Triaged 行)：MML script 页面「更新」弹窗取消产品类型字段（应与 /mml/console「保存脚本」弹窗一致）

- ✅ ScriptTask 页 `/mml/script` 的 Drawer（新建+更新双路径共用）已删 productType Form.Item
- ✅ 与 console "保存脚本" 弹窗（T-0090-a 已删 categoryGroup/productTypes/参数配置）对齐 — 都不再显示产品类型选择
- ✅ 反向依赖语义满足：subtask Notes 原文 "console 决定 productTypes 去留 → script 跟随"；T-0090 umbrella 选 "删" 路线，T-0096 follow 删

**遗留**（不在 subtask scope，保留 followup 空间）：
- ScriptTask filter bar 的 `deviceType` Select 仍含 productTypeOptions — 用于过滤"既有"任务的产品类型；新任务都不再写 productType 字段，长期 filter bar 也可清理（建议未来 sprint 单起 followup sub-task 同 P3）
- TaskRecord 页是否有类似 productType 字段？grep 显示仅 ScriptTask 有，TaskRecord 不涉

---

## §5. 用户回归路径

1. `cd omcmb/webcode && npm run dev`
2. 浏览器进 `/mml/script`
3. 验证：
   - 点 "新建任务" → Drawer 弹出 → form 字段中**无** "产品类型"
   - 选任务点 "更新" → Drawer 弹出 → 同样无 "产品类型"
   - filter bar 上方 "产品类型" 过滤（按 deviceType 名）**仍存在**（不在本任务 scope）

---

## §6. S4 出口门

- [✓] 所有命令绿（typecheck PASS / 改动文件未引入新 lint error）
- [N/A] E/R ≥ 1（无新端点）
- [N/A] 迁移双向演练（无迁移）
- [N/A] metric/log 名（无新埋点）
- [N/A] 累计型依赖阈值

**S4 PASS**。
