# Code Review — T-0090-a (MML Console UX a)

**Reviewer**: Claude (self-review)
**Date**: 2026-05-12
**Scope**: 5 files / 5 net-add components / FE-only frontend rework
**Related**: [`verify-T-0090-a.md`](./verify-T-0090-a.md)

---

## §1. Findings 严重度汇总

| Severity | Count |
|----------|-------|
| **CRITICAL (P0)** | 0 |
| **HIGH (P1)** | 0 |
| **MEDIUM (P2)** | 0 |
| **LOW (P3)** | 2 |
| **NOTE** | 3 |

无 P0/P1 issue。可以合入。

---

## §2. 逐文件审查

### 2.1 `omcmb/webcode/src/pages/mml/components/CommandCodeTextarea.tsx` (32 行，新建)

**优点**：
- 标准 Form.Item child 契约（`value?` + `onChange?`）— 与 antd `Form.Item` 的 `cloneElement` 自动注入兼容
- 无内部 state，纯受控组件
- maxLength=200 与旧 commandCode 字段 maxLength 持平
- 字体 monospace SFMono-Regular 12px 与 ScriptTaskDrawer 命令文本一致

**LOW-1**：默认 rows=3 可能在长命令场景偏短。**建议**：保留默认；调用方需要可通过 prop 覆盖（已支持）。无需修改。

### 2.2 `omcmb/webcode/src/pages/mml/components/OperationTypeWithModify.tsx` (66 行，新建)

**优点**：
- 接受 `form: FormInstance` prop — 显式依赖注入，便于 d 任务复用时传入自己的 form 实例
- `Form.useWatch(operationTypeName, form)` — antd v5 标准条件渲染模式
- `defaultOptions` memoized 依赖 `t`，避免 locale 稳定时重复重建
- 完整 10 个 operationType 选项（与旧 AddTemplateModal 一致）

**LOW-2**：调用方传入自定义 `options` prop 时不 memoize；若调用方每次渲染传新数组会引起 Select 重渲。**建议**：调用方自己 memoize 即可；不在本组件内强加 useMemo。无需修改。

**NOTE-1**：组件签名预留 `operationTypeName` / `modifyValuesName` 自定义字段名 — 是为 d 任务"私有命令页"中如果字段命名不同（如 `privateOperationType`）做的扩展点。当前用例未用，但保留接口低成本。

### 2.3 `omcmb/webcode/src/pages/mml/Console/components/AddTemplateModal.tsx` (286 → 140 行，重写)

**优点**：
- 净减 146 行，删除：`useAllMMLCommands` / `useDictionary` / `commandCodeOptions` / `matchedCommand` / `editableParams` / `paramValues` state / `renderParamControl` / `handleParamChange` 等不再需要的代码
- `parseModifyValues` 是顶层 pure function，可测试性高
- 提交 payload 明确传 empty default（categoryGroup=''/parameters={}/paramPaths=[]/productTypes=[]）+ 注释解释"T-0090-b 才真删 column"，把短期 schema 兼容与长期清理 decouple 清晰
- T-0097 messageApi 修复保留（`App.useApp()` 模式）

**NOTE-2**：`parseModifyValues` 中 `eq <= 0` 严格防止"=" 在首位的行被解析为空 key — 正确处理边界
**NOTE-3**：用户切换 MOD→其它 type→MOD 时 `modifyValues` 字段值保留（非 reset），属合理 UX（避免误删用户工作）。如需"切换即清空"语义，未来可在 OperationTypeWithModify 加 onChange 钩子

### 2.4 i18n 改动（zh-CN + en-US 对称）

- +7 keys / -6 keys 双向 symmetric ✅
- placeholder 含实例 (`CELL_INDEX=1` / `LTE_INTER_FREQ_DL_EARFCN=41390`) — 与 ScriptTaskDrawer L150-160 内嵌的 MMLTemplate.txt 示例风格一致
- 删除的 6 keys 经全工程 grep 确认仅 AddTemplateModal 引用 — 安全删除

---

## §3. DoD 逐项核销（`docs/project/dod.md` 通用 DoD + 模块特定）

### 编译与类型
- [N/A] 后端 `go build ./...`（FE-only）
- [N/A] 后端 `go test ./...`（FE-only）
- [N/A] 后端 `golangci-lint run`（FE-only）
- [✓] 前端 `cd omcmb/webcode && npx tsc --noEmit` 通过
- [✓] 前端 `npm run lint`：我改动的 5 文件 0 报错；全工程 225 pre-existing 与 T-0090-a 无关

### 测试
- [N/A] 新增单元测试 — 前端无单测约定（subtask 文件原文："纯 FE 低风险，typecheck PASS 即可"）
- [N/A] 成功 + 失败两路径 — 无单测载体
- [N/A] 新 REST 端点 E2E — 无新端点
- [N/A] 修复 bug 回归测试 — feat 而非 bug
- [N/A] 覆盖率不退 — 无单测变更
- [✓] 禁止禁用失败测试 — 未禁用任何测试

### 迁移与数据
- [N/A] 全部（无迁移文件）

### 代码规范
- [✓] 无遗留 TODO / FIXME / panic
- [N/A] `fmt.Errorf` 错误处理（FE-only）
- [N/A] Squirrel SQL（FE-only）
- [N/A] Carrier 接口（FE-only 无运营商差异）
- [✓] 后端响应 BackendXxx 映射 — MMLCustomCommand type 直接用，empty default 不破坏映射
- [✓] 无 `any` — 严格 TS，所有 props/state 显式类型
- [N/A] zap 结构化日志（FE-only）

### 文档
- [✓] PR 说明（commit body 待 S6 写）含 Why（"删 3 字段 / textarea 自定义 / MOD 修改值入口"）
- [✓] 关联 Backlog Task: T-0090-a
- [✓] 关联 PRD: subtask 文件 + backlog T-0090 Notes（七子项 + GWT）
- [N/A] CLAUDE.md / omcgo/CLAUDE.md 同步 — 无约定变更
- [N/A] Swagger / API 文档 — 无后端接口变更

### 流水线闭环
- [ ] commit footer 四元组（待 S6 执行时验）
- [ ] backlog.md Task 状态回写（待 S7 执行）
- [N/A] 快速通道 postmortem — T-0090-a 走完整 S0-S7（虽然 S0/S1/S2 通过 umbrella 继承，但未走 fast track）

### 安全
- [N/A] 新 API JWT — 无新端点
- [N/A] RBAC — 无敏感操作
- [✓] 输入校验 — Form.Item required rule + maxLength + parseModifyValues 容错
- [✓] 日志/错误不打印敏感 — 错误信息只含 error.message，不含密钥/token
- [N/A] 文件上传 — 不涉及

### 可观测性
- [N/A] Prometheus 指标 — FE-only
- [N/A] request_id / trace_id — FE-only（http 拦截器已注入）
- [N/A] context 取消 — FE-only

### 模块特定（前端）
- [✓] API/Hook/Store/Type 进 `frontend-core`：i18n keys 改 `frontend-core/src/i18n/*`；新组件在 `webcode/src/pages/mml/components/`（属 UI 壳，正确位置）
- [✓] 改 `frontend-core` 评估 v2/v3 影响：仅 i18n keys 增删（key 名独立、不破坏既有用法），v2/v3 编译不受影响（i18n 键缺失只会落 fallback，未来切换皮肤时如复用同 Modal 才需关注）

---

## §4. 结论

**APPROVE** — 无 P0/P1 issue；2 个 LOW 建议不强制修改；5 个 NOTE 仅为记录。

可以进 S6 commit。
