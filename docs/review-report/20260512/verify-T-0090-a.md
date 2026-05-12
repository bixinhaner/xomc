# Verify Report — T-0090-a (MML Console UX a)

**Task**: T-0090-a — MML Console FE-only ①②③④ 整改（操作类型差异化 MOD→修改值入口 + 删 3 字段 UI + 命令编码 Select→TextArea + 不允许选既有命令）
**Type**: feat / frontend / P2 / S (1-2d)
**Sprint**: sprint-11 (pull-forward 进 sprint-10 buffer 执行，sprint-10 三 deliverable 已全闭)
**Deps**: — (无)
**Owner**: Claude
**Date**: 2026-05-12

---

## §1. 改动面

| 文件 | 操作 | 行数变动 |
|------|------|---------|
| `omcmb/webcode/src/pages/mml/components/CommandCodeTextarea.tsx` | 新建 | +32 |
| `omcmb/webcode/src/pages/mml/components/OperationTypeWithModify.tsx` | 新建 | +66 |
| `omcmb/webcode/src/pages/mml/Console/components/AddTemplateModal.tsx` | 重写 | 286 → 140 行（净减 ~146 行 — 删 paramValues state / renderParamControl / dictionary hooks / commandCodeOptions 等不再需要的代码）|
| `omcmb/frontend-core/src/i18n/zh-CN/index.ts` | 增删 | +7 keys / -6 keys |
| `omcmb/frontend-core/src/i18n/en-US/index.ts` | 增删 | +7 keys / -6 keys |

新增 i18n keys：`commandCodeRequired` / `commandCodeTextareaPlaceholder` / `commandCodeTextareaTip` / `modifyValuesLabel` / `modifyValuesPlaceholder` / `modifyValuesTip` / `modifyValuesRequired`
删除 i18n keys：`selectOrInputCommandCode` / `selectExistingOrCustom` / `categoryGroup` / `selectCategory` / `productTypes` / `selectProductTypes` / `paramConfig`

---

## §2. 子项 ①②③④ 验收逐项核销

### ① 操作类型差异化（MOD → "修改值入口"）
- 实现：`OperationTypeWithModify.tsx` 通过 `Form.useWatch('operationType')` 监听；当值 === 'MOD' 时条件渲染 `Form.Item name="modifyValues"` + `Input.TextArea`（rows=4, monospace, 占位符示例 `CELL_INDEX=1`）
- AddTemplateModal `handleSubmit` 中调 `parseModifyValues()` 把 K=V 文本解析回 `parameters` dict 提交
- ✅ GWT-a 第 1 条 "选'修改值'操作类型 / Then UI 出现修改值入口" 满足

### ② 删 3 字段：参数配置 / 所属分类 / 适用产品类型
- AddTemplateModal 已删 `Form.Item name="categoryGroup"`（旧 L249-251）/ `Form.Item name="productTypes"`（旧 L253-260）/ "参数配置" section（旧 L262-278）
- 副带删除：`useDictionary` import / `useAllMMLCommands` import / `categoryOptions` useMemo / `productTypeOptions` useMemo / `paramValues` state / `handleParamChange` / `renderParamControl` / `editableParams` useMemo / `MMLCommand` 和 `MMLParam` type import
- **后端 schema 兼容**：`MMLCustomCommand` schema 4 字段 (categoryGroup / parameters / paramPaths / productTypes) 仍 required；提交时传 empty default (`'' / {} / [] / []`)，对后端无破坏；子任务 b 才真删 product_types column
- ✅ GWT-a 第 2 条 "参数配置 / 所属分类 / 适用产品类型 3 字段已删" 满足

### ③ 命令编码 input → textarea
- `CommandCodeTextarea.tsx` 渲染 `Input.TextArea`（rows=3, maxLength=200, monospace 12px）
- 旧 `Form.Item` 用 `<Select>` 现改为 `<CommandCodeTextarea>`
- ✅ GWT-a 第 3 条 "命令编码 input 改 textarea" 满足

### ④ textarea 必须自定义、不允许选既有命令
- `CommandCodeTextarea.tsx` 不接收 options / 不渲染下拉 — 物理上无法"从下拉选既有命令"
- 配套删除 `useAllMMLCommands` 和 `commandCodeOptions` 派生代码
- 占位符明文："请输入自定义命令编码（不可选既有命令）" + `Form.Item extra` 二次提示
- ✅ GWT-a 第 4 条 "textarea 内容不能从下拉选既有命令" 满足

---

## §3. 出口门检查

| 检查项 | 结果 | 备注 |
|--------|------|------|
| `npx tsc --noEmit`（webcode） | ✅ PASS | 无输出 |
| `eslint` 我改动的 5 文件 | ✅ 0 errors / 0 warnings | 单独 `npx eslint <5 file>` 验证 |
| 全工程 `npm run lint` | ⚠️ 225 problems (163 errors / 62 warnings) | **全部 pre-existing**（topology/GISMapView, SiteManagement, TopologyCanvas, QueryProvider, routes 等），与 T-0090-a 无关 |
| 无新增 TODO/FIXME/panic | ✅ | grep 验证 |
| 无新增 `any` / `interface{}` | ✅ | 严格 TS，所有 props 显式 interface |
| 无新增 `if carrier ==` 硬编码 | N/A | FE-only |
| 后端 build / test | N/A | FE-only |
| 新端点 R | 0 | 无新路由 |
| E2E claim 增量 E | 0 | 无新端点 → E/R N/A |
| 迁移 up/down | N/A | 无迁移 |
| 新 metric/log 名 | N/A | 无新埋点 |
| 累计型 deps | N/A | Deps `—` |

---

## §4. 抽象组件对子任务 d/c 的复用价值（R-NEW-3 mitigation）

`pages/mml/components/` 新增 2 个共享组件：

| 组件 | 复用接口 | d 用法预期 |
|------|---------|-----------|
| `CommandCodeTextarea` | `{value?, onChange?, rows?, maxLength?, disabled?}` | d 的私有命令新建/编辑 modal 直接 `import` 用 |
| `OperationTypeWithModify` | `{form, operationTypeName?, modifyValuesName?, options?}` | 同上；接受自定义 `form` instance + 字段名可改 |

c 任务（后端 RBAC）不直接消费 FE 组件，但需保证后端 schema 接收的 `parameters` dict 与本任务 MOD 解析输出一致（已通过 `parseModifyValues` 保证）。

---

## §5. 用户回归确认（与 T-0097 同 pattern）

本任务**未启动 dev server 浏览器实测**，依据：
- subtask 文件第 1 行明示 "纯 FE 低风险，typecheck PASS 即可"
- T-0097（最近 MML Console 同模块 fix）commit 728ee1ed8 收尾原文："浏览器端实测留用户回归确认"
- 改动面边界明确（5 文件，无跨模块影响），typecheck + 5 文件 lint 净 0 已是足够静态证据

**建议用户回归路径**：
1. `bash run/scripts/start-all.sh`（或仅 `cd omcmb/webcode && npm run dev`）
2. 浏览器进 MML Console → 点 "保存脚本"（新增公有/私有命令 Modal）
3. 验证 4 条 GWT：
   - 命令编码控件已是 TextArea（无下拉）
   - 操作类型选 MOD 后下方出现 "修改值入口" TextArea（其它类型不出现）
   - 表单中无 "所属分类" / "适用产品类型" / "参数配置" 字段
   - 在 MOD 修改值入口里输入两行 `K=V` 保存 → 后端记录 parameters dict 应含两键

---

## §6. S4 出口门

- [✓] 所有命令绿（typecheck + 我文件 lint）
- [N/A] E/R ≥ 1（无新端点）
- [N/A] 迁移双向演练（无迁移）
- [N/A] metric/log 名全部能找到（无新埋点）
- [N/A] 累计型依赖阈值（无累计 deps）

**S4 PASS**。
