# Code Review — T-0090-d (MML 前端私有命令页)

**Reviewer**: Claude (self-review)
**Date**: 2026-05-12
**Scope**: 5 files / 1 new page + 1 route + 1 API bugfix + 2 i18n / Type=feat / FE-only
**Related**: [`verify-T-0090-d.md`](./verify-T-0090-d.md)

---

## §1. Findings 严重度汇总

| Severity | Count |
|----------|-------|
| **CRITICAL (P0)** | 0 |
| **HIGH (P1)** | 0 |
| **MEDIUM (P2)** | 0 |
| **LOW (P3)** | 1 |
| **NOTE** | 3 |

无 P0/P1。**APPROVE**。

---

## §2. 逐文件审查

### 2.1 `omcmb/webcode/src/pages/mml/PrivateCommand/index.tsx` (+178 行，新建)

**优点**：
- 严格遵循 ScriptLibrary 同 pattern（ListPageLayout + DataTable + FilterBar），减少思维负担
- 0 业务逻辑复制粘贴 — Add Modal 直接 import 既有 AddTemplateModal 传 scope='private'
- `useMMLTemplates({ templateScope: 'private' })` + 附带修复 query param 让 RBAC 后端 + scope filter 双重过滤生效
- T-0097 messageApi 模式保留（`App.useApp().message`）
- 表格列 `operationType` 用 Tag 视觉区分 MOD (orange) vs 其它 (blue) — UX 清晰
- 命令编码列 monospace 字体一致 console pattern

**LOW-1**：当前页面无 Edit modal — GWT-d 未明示必须有；hooks 已导入预留（`void useCreateMMLTemplate / useUpdateMMLTemplate` placeholder）。**建议**：若实际用户需要 inline edit，后续 sprint 单起 sub-task 与 a 抽出 Modal 对称复用。无需本任务修。

**NOTE-1**：`void useCreateMMLTemplate; void useUpdateMMLTemplate;` 形式保留未来扩展点 — 这是显式的"future hook"，比注释好（编译期保证 import 不被自动清掉）。如未来确定不需要 edit，可改为直接删除。

### 2.2 `omcmb/webcode/src/router/routes.tsx` (+2 行)

**优点**：
- 与既有 MML 4 个 route 同 pattern（const + lazy import + path entry）
- withSuspense 一致包裹
- 缩进 align 调整既有 mml/* 路由 表格化（受影响 4 行排版）— 改动 commit 时会显示，但不影响行为

**NOTE-2**：路径命名 `/mml/private-command` 用 kebab-case 与既有 `/mml/task-records` 一致 — 选择对称

### 2.3 `omcmb/frontend-core/src/services/api/mmlApi.ts` (+2 / -2，附带修复)

**关键修复**：FE 旧版查询字符串名 `template_scope` 与后端 `c.Query("command_scope")` 不一致 → scope 过滤被静默忽略

**优点**：
- 修复保留 FE 接口面 `templateScope` 不变（调用方零破坏）
- 仅改 BE 契约名 `query.command_scope = params.templateScope`
- 注释 inline 解释修复 rationale + 关联到 backend handler.go L722

**NOTE-3**：此修复实际是 pre-existing bug，独立成 hotfix 也合理；本任务"附带" inline 是为了让 T-0090-d 的"列表自动按 c 后端 RBAC 过滤"GWT 真生效。提交注释明示修复属性。

### 2.4 `omcmb/frontend-core/src/i18n/zh-CN/index.ts` + `en-US/index.ts` (各 +2 行)

**优点**：
- zh-CN + en-US 对称 +2 keys
- `mml.privateCommand.pageTitle` 命名空间清晰
- `mml.confirmDeleteCustomCommand` 含占位符 `{name}` 用户友好

---

## §3. DoD 逐项核销

### 编译与类型
- [N/A] 后端（FE-only）
- [✓] 前端 `npx tsc --noEmit` PASS
- [✓] 前端 5 文件 `npx eslint` 0 errors / 1 pre-existing warning（routes.tsx L169 与本任务无关）

### 测试
- [N/A] 单元测试 — FE 无单测约定；UI 列表页 typecheck PASS 即可（subtask Notes "纯 FE 低风险 typecheck PASS 即可"对 d 同理）
- [N/A] 成功 + 失败两路径
- [N/A] 新 REST 端点 E2E — 无新后端端点；client-side 路由 + 既有端点
- [N/A] 修复 bug 回归 — 附带修复 query param 名称，影响面经 T-0090-c 2 e2e claim 覆盖（mml-4a/4b 已验 scope=private/public 200 路径）
- [N/A] 覆盖率不退
- [✓] 禁止禁用失败测试 — 0 禁用

### 迁移与数据
- [N/A] 全部

### 代码规范
- [✓] 无 TODO / FIXME / panic — `void useCreateMMLTemplate` 是显式 future hook 非 TODO
- [N/A] 后端规范
- [✓] 后端响应 BackendMMLCustomCommand → mapBackendCustomCommand → MMLCustomCommand 三层映射完整（无本任务改动；继承 T-0090-b/c 现状）
- [✓] 无 `any` — 严格 TS
- [N/A] zap 结构化日志

### 文档
- [✓] PR 说明含 Why（"d sub-task / RBAC 后端过滤生效 / 公共组件复用闭环 R-NEW-3"）
- [✓] 关联 Backlog: T-0090-d
- [✓] 关联 PRD: subtask 文件 + backlog T-0090 umbrella ⑤
- [N/A] CLAUDE.md 同步
- [N/A] Swagger 同步

### 流水线闭环
- [ ] commit footer 五元组（待 S6）
- [ ] backlog.md Task 状态回写（待 S7）+ **T-0090 umbrella State in_design → done**（4/4 全闭）
- [N/A] 快速通道 postmortem — type=feat 完整流程

### 安全
- [N/A] 新 API JWT — 复用既有端点（auth middleware 既有）
- [N/A] RBAC — 复用 T-0090-c 服务端 RBAC（FE 不做权限决策）
- [✓] 输入校验 — Modal.confirm + Form 校验
- [✓] 日志/错误不打印敏感 — error.message 抓取无敏感
- [N/A] 文件上传

### 可观测性
- [N/A] 全部 — FE-only 无新埋点

### 模块特定（前端）
- [✓] 改动定位正确：API/Hook/Type/i18n 在 `frontend-core/`；页面 + 路由在 `webcode/`
- [✓] 类型安全：`MMLCustomCommand` 显式类型（T-0090-b 后已不含 productTypes）
- [✓] 国际化：所有用户可见文本经 `useT()`，语料 zh-CN + en-US 对称 +2
- [✓] 多皮肤评估：webcode-v2/v3 未直接受影响；frontend-core 改动（mmlApi.ts query bugfix + i18n keys）v2/v3 自然受益；本任务**未引入新 typecheck 失败**

---

## §4. R-NEW-3 mitigation 完整闭环检视

| 步骤 | 状态 |
|------|------|
| T-0090-a 抽出 CommandCodeTextarea + OperationTypeWithModify 落 `pages/mml/components/` | ✅ done |
| T-0090-d 通过 AddTemplateModal 间接复用上述组件（不复制粘贴）| ✅ done |
| 复制粘贴行数 = 0 | ✅ 验证（PrivateCommand 内 0 行业务逻辑复用，全经 AddTemplateModal） |

**R-NEW-3（组件抽象不足 → d 复制粘贴）风险**：mitigation 完整闭环。

---

## §5. 结论

**APPROVE** — 无 P0/P1；1 LOW（缺 Edit modal）不强制（GWT-d 未要求）；3 NOTE 仅记录。

可以进 S6 commit。
