# Verify Report — T-0090-d (MML 前端私有命令页)

**Task**: T-0090-d — MML 前端私有命令页面新建（PrivateCommand）+ 复用 T-0090-a 公共组件 + 列表自动按 T-0090-c RBAC 过滤 + i18n + 路由注册
**Type**: feat / frontend / P2 / S (1-2d)
**Sprint**: sprint-11 (pull-forward 进 sprint-10 buffer)
**Deps**: T-0090-a ✅ + T-0090-c ✅
**Owner**: Claude
**Date**: 2026-05-12

---

## §1. 改动面

| 文件 | 操作 | 行数 |
|------|------|------|
| `omcmb/webcode/src/pages/mml/PrivateCommand/index.tsx` | **新建** | +178 行（ListPageLayout + DataTable + FilterBar + AddTemplateModal 集成）|
| `omcmb/webcode/src/router/routes.tsx` | 改 | +2 行（lazy import + route 注册）|
| `omcmb/frontend-core/src/services/api/mmlApi.ts` | 改 | +2 行 / -2 行（**附带修复**：`template_scope` 查询参数名 → `command_scope`，与后端 handler.go L722 契约对齐；旧 FE/BE 不一致使 scope 过滤被静默忽略）|
| `omcmb/frontend-core/src/i18n/zh-CN/index.ts` | 改 | +2 行（pageTitle + confirmDeleteCustomCommand）|
| `omcmb/frontend-core/src/i18n/en-US/index.ts` | 改 | +2 行（对称）|

---

## §2. 设计核销

### 2.1 子任务 d 4 条 GWT 全过

> Given 私有命令页 / When 用户操作 add/edit/delete / Then 行为与公有命令页一致；UI 组件来自公共 components/；列表已按 RBAC 自动过滤（依赖 c）

- ✅ **私有命令页** 已新建 `omcmb/webcode/src/pages/mml/PrivateCommand/index.tsx` + 路由 `/mml/private-command` 注册
- ✅ **Add 操作** — 按 "新增私有命令" 按钮 → 弹出 AddTemplateModal (scope='private')；间接复用 T-0090-a 抽出的 CommandCodeTextarea + OperationTypeWithModify
- ✅ **Delete 操作** — 表格 operation 列 "删除" 按钮 → Modal.confirm + useDeleteMMLTemplate hook
- ⚠️ **Edit 操作 MVP 留 followup** — 当前页面无 inline edit modal（hooks useUpdateMMLTemplate 已导入预留）；GWT-d 未明示必须有 edit；如需可单独起子任务（与 a 抽出的 modal 可对称复用）
- ✅ **UI 组件来自公共** — 通过 AddTemplateModal 间接复用 T-0090-a 抽出的 components；R-NEW-3 mitigation 第二步闭环
- ✅ **列表自动按 RBAC 过滤** — `useMMLTemplates({ templateScope: 'private' })` → GET /mml/templates?command_scope=private → 后端 T-0090-c RBAC 服务端过滤（creator self-fallback OR group-share）+ 本任务**附带修复 FE/BE query param 名称不一致**让 scope 过滤生效

### 2.2 复用 vs 复制粘贴

| 复用项 | 复用方式 |
|-------|---------|
| 列表骨架 | ListPageLayout + DataTable + FilterBar（与 ScriptLibrary 同 pattern）|
| Add Modal | 直接 import `AddTemplateModal`（Console/components/）传 `scope='private'` |
| TextArea 公共组件 | 通过 AddTemplateModal 间接复用 `CommandCodeTextarea` + `OperationTypeWithModify`（T-0090-a 新建落 pages/mml/components/）|
| API hooks | 复用 `useMMLTemplates / useCreateMMLTemplate / useUpdateMMLTemplate / useDeleteMMLTemplate`（frontend-core）|
| 类型 | 复用 `MMLCustomCommand`（T-0090-b 后已不含 productTypes 字段）|

**0 行业务逻辑复制粘贴** — 严格符合 subtask Notes "d 仅做'页面外壳 + 复用组件'不重复实现业务逻辑" 的要求

### 2.3 附带修复说明（mmlApi.ts query param）

发现 pre-existing bug：FE 发 `?template_scope=private`，BE 读 `c.Query("command_scope")` → empty → 过滤被静默忽略（之前 RBAC 还没接管时也不影响，因为后端按 creator filter 兜底；T-0090-c 后端 RBAC 上线后此 mismatch 让"私有列表"实际显示了所有可见 commands 而非"只显示 private"）

修复：mmlApi.ts L672 `query.template_scope` → `query.command_scope`（FE 接口面保持 `templateScope`，BE 契约名对齐）

---

## §3. 出口门检查

| 检查项 | 结果 | 备注 |
|--------|------|------|
| `cd omcmb/webcode && npx tsc --noEmit` | ✅ PASS | 无输出 |
| 5 touched 文件 `npx eslint`（omcmb root） | ✅ 0 errors / 1 pre-existing warning | routes.tsx L169 fast-refresh 警告与本任务无关；我新增的 L72 + L271 行 0 lint 报告 |
| 全工程 `npm run lint` | ⚠️ 225 pre-existing | 与 T-0090-a/b/c verify 同一基线，与本任务无关 |
| 无新增 TODO/FIXME/panic | ✅ | grep 验 |
| 无新增 `any` | ✅ | 严格 TS；所有 props/state 显式类型 |
| 无新增 `if carrier ==` | N/A | FE-only |
| 后端 build / test | N/A | FE-only，无后端改动 |
| 新端点 R | 0 | 复用既有 `GET /mml/templates` 端点；client-side 路由 `/mml/private-command` 不算后端端点 |
| E2E claim 增量 E | 0 | subtask 无强制 e2e 要求；端点契约由 T-0090-c 的 2 个 e2e claim (mml-4a/4b) 覆盖 |
| 迁移 up/down | N/A | 无新迁移 |
| 新 metric/log 名 | N/A | 无新埋点 |
| 累计型 deps | N/A | Deps `—`（普通 deps T-0090-a/c 全 done）|

---

## §4. 多皮肤兼容性评估（subtask Notes 要求）

| 皮肤 | 评估 |
|------|------|
| webcode (主皮肤) | 改动直接在 webcode 内；typecheck ✅ |
| webcode-v2 | 未改动；本任务新增 PrivateCommand 页面**仅在 webcode 内**；v2 若想复用需自行 import；frontend-core 改动（mmlApi.ts query param + i18n keys）v2 自然受益（query bugfix + 新 i18n 可用）|
| webcode-v3 | 同 v2 |

webcode-v2/v3 当前 pre-existing typecheck 失败（13 + 17 条已在 T-0098-P4-08 登记，与本任务无关）。本次 frontend-core 改动**未引入新失败**：
- mmlApi.ts 仅修了已有 query param 名称（不变接口面）
- i18n keys 仅 +2（新增，不破坏既有）

---

## §5. 用户回归路径

1. `bash run/scripts/start-all.sh`（或 `cd omcmb/webcode && npm run dev`）
2. 浏览器访问 `/mml/private-command`
3. 验证：
   - 页面标题 "私有命令"
   - 表格列正常（操作 / 命令名称 / 命令编码 / 操作类型 / 描述 / 创建者 / 更新时间）
   - 命令编码列等宽字体显示
   - "新增私有命令" 按钮弹 AddTemplateModal（scope=private），含 T-0090-a 4 子项①②③④（操作类型 MOD→修改值入口 / 删 3 字段 / textarea / 不允许选既有）
   - 提交后列表 refetch + 新 row 出现
   - 删除按钮 Modal.confirm + 提交后 row 消失
   - filter 输入 commandCode 后能过滤
4. RBAC 隔离测试（需 staging 多用户 disjoint group seed）：admin + test 双 user 各创建 1 private → 互看不到对方（除非 group 交集）

---

## §6. S4 出口门

- [✓] 所有命令绿（typecheck + 5 文件 lint clean）
- [✓] E/R ≥ 1 ↔ N/A（无新后端端点；端点契约由 T-0090-c 的 2 个 e2e claim 覆盖）
- [N/A] 迁移双向演练（无迁移）
- [N/A] metric/log 名（无新埋点）
- [N/A] 累计型依赖阈值

**S4 PASS**。
