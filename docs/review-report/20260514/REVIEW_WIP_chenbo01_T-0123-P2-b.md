# Code Review Report — T-0123-P2-b

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-14 |
| 提交 | WIP（pre-commit；S6 完成后用真 hash rename 或新建） |
| 作者 | chenbo01（self-review per dev-pipeline §B5；`/review` 跳过，节约会话上下文） |
| 范围 | MML Console 前端 5 个新 UI 组件（MmlEditor / SubFieldChecklist / SubFieldInputList / InstancePicker / StepBar）|
| 变更文件数 | 14（10 新：5 .tsx + 5 .test.tsx；3 改：2 i18n + 1 setup；1 PRD 追加 §O；1 backlog 回写；1 verify report 新增）|
| 新增行数 | ~700（5 组件 ~345 + 5 测试 ~355；不含 PRD/backlog/verify）|
| 删除行数 | 0 |
| 关联 Task | T-0123-P2-b（依赖 T-0123-P2-a ✅ commit `3c760ce9`）|
| 关联 PRD | `docs/design/mml-restore-old-interaction-plan-20260514.md` v2 APPROVED §7.1-7.5 + §O |
| 关联 Risk | R-206 Mitigating（P2/3/4 后关闭）|

---

## 变更概要

5 个 React 函数组件，全部新增、不动既有；业务层（types / store / hooks / i18n）由 P2-a 已 ship，本期消费。

- **MmlEditor** — Step3 顶部 textbox + DO 按钮；onChange → store.setMmlTextDebounced（store 内 300ms 防抖 + 调 useParseMML.mutateAsync）；DO 扫 statements 中 OnReboot 命中 → Modal.confirm；execute 走 useExecuteStatements
- **SubFieldChecklist** — LST 视图：勾选行 + access icon（READ_ONLY=EyeOutlined / READ_WRITE=EditOutlined）+ OnReboot Tag；点击调 store.toggleSubField；sortOrder 升序
- **SubFieldInputList** — MOD/ADD 视图：MOD 过滤 READ_ONLY（PRD §7.3）；isRequired 字段红 `*`；constraintText hint；OnReboot Tag；onChange → store.setValue
- **InstancePicker** — RMV 视图：antd InputNumber min=0 step=1；GPV 自动探测延 P4（§O.9 待定点 1）
- **StepBar** — antd Steps 横向 4 步；纯 props 驱动，不订阅 store

---

## DoD 核查（通用 + 前端模块特定）

### 通用 — 编译与类型
- [✓] 前端 `npx tsc --noEmit` 通过（5 新 .tsx + 5 新 .test.tsx 全过）
- [✓] 前端 `npm run lint` 无新增告警（changed files ESLint = 0）
- [N/A] 后端 `go build` / `go test` / `golangci-lint` — 纯前端任务

### 通用 — 测试
- [✓] 新增代码包含单测（5 文件 16 case）
- [✓] 测试覆盖成功 + 失败两条路径（例：MmlEditor DO with/without OnReboot；SubFieldInputList MOD 过滤 vs ADD 保留；InstancePicker 正数 vs 空值）
- [N/A] 新 REST 端点 E2E — 无新端点
- [N/A] bug 回归测试 — 非 bugfix
- [✓] 测试覆盖率不降（pages/components 不在 vitest coverage.include 范围；新增 16 case 净增量）
- [✓] 未禁用任何失败测试

### 通用 — 迁移与数据
- [N/A] 无新迁移；无 DB 变更

### 通用 — 代码规范
- [✓] 0 处 TODO/FIXME/panic（grep 5 新 .tsx 文件命中 0）
- [N/A] 错误 wrap 规范 — 纯 TS；catch 块用 `e instanceof Error ? e.message : String(e)`
- [N/A] SQL Squirrel — 纯前端无 SQL
- [✓] 无 `if carrier == "cmcc/ctcc/cucc"` 硬编码（PRD §0.5 锁定三家一致）
- [✓] 后端响应类型有 `BackendXxx` → `mapBackendXxx` → `Xxx`（在 P2-a frontend-core 已建立；本期消费 `@core/types/mmlConsole` 已 typed Statement / SubFieldDef，无 `any`）
- [N/A] zap.String 结构化日志 — 纯前端

### 通用 — 文档
- [✓] PR 说明 Why 明确：恢复老 OMC MML 三栏交互，P2-b 落 5 个新组件作为 P2-c 三栏重构的零件
- [✓] 关联 Backlog Task `T-0123-P2-b`（已 §3 Active line 342，State=planned/Owner=Claude）
- [✓] 关联 PRD：`docs/design/mml-restore-old-interaction-plan-20260514.md` §7.1-7.5 + §O 设计备忘
- [N/A] CLAUDE.md 约定无变更
- [N/A] 对外 API / Swagger 无变更（无新端点）

### 通用 — 流水线闭环（S6/S7 硬门）
- [pending S6] commit footer 五元组将齐全（见 `verify-T-0123-P2-b.md` §5）
- [pending S7] backlog Task 状态待 in_review → done 回写
- [N/A] 非快速通道（feat 走全套 S0-S7，无 hotfix 裁剪）

### 通用 — 安全
- [N/A] 无新 API 端点
- [N/A] 无敏感操作
- [✓] 用户输入校验：InstancePicker InputNumber min=0 + precision=0；SubFieldInputList isRequired 状态可视化
- [N/A] 路径遍历 / SQL 注入 — 纯前端无后端边界
- [✓] 日志 / 错误不打印敏感信息（MmlEditor 错误捕获只取 `error.message`，未透传完整对象）
- [N/A] 文件上传 — 无

### 通用 — 可观测性
- [N/A] Prometheus 指标 — 前端组件按 PRD §O.10 延 P4 统一接入
- [N/A] request_id — 前端组件依赖 axios 拦截器，已在 `http.ts` 全局植入
- [N/A] context 取消 — React Query 自动管理 mutation/query 生命周期

### 前端模块特定（`omcmb/webcode/**`）
- [✓] API 服务连真实后端（useParseMML / useExecuteStatements 走 mmlApi 真 axios；P2-a 阶段未配 mock console service 已 acknowledged）
- [N/A] Hook `useMock ? mockService : realApi` — 本期组件不直接消费 mock；mock 切换在 hooks 层（已在 frontend-core/services/apiSwitch.ts，本期消费侧无需感知）
- [✓] 查询键层级 `['domain', 'action', params]`（继承 P2-a：`['mml', 'console', 'group-tree', ...]` / `['mml', 'console', 'sub-fields', ...]`）
- [✓] 用户可见文本通过 `react-intl`（全部 9 新 + 复用 P2-a 既有 keys 均经 useT；硬编码字符串仅 fallback：`'DO'` 等运营商不区分静态）
- [✓] 错误页面级 ErrorBoundary 已在上层 router 注册；MmlEditor 内部捕获 execute 错误 → antd message.error 反馈

---

## 简洁性自查（替代 `/simplify`）

| 维度 | 结果 |
|------|------|
| 单一职责 | ✓ 5 组件各司其职（编辑器 / 勾选 / 输入 / 实例选 / 步骤栏），无 mixed 责任 |
| 文件大小 | 最大 MmlEditor 127 行（PRD §O.1 估 ~120 命中），其余均 < 100 行；远低于 800 max |
| 嵌套深度 | ≤ 3（条件渲染 + 列表 map）；无 >4 levels |
| 函数大小 | 内联 helper `hasOnRebootHits` 6 行；所有 handler ≤ 30 行 |
| 抽象适当 | 5 组件 = 5 文件，不提"通用 SubFieldRenderer"基类（PRD §7.3 元数据差异化在组件内 if 即可，未到需要抽象的 3 处相似阈值）|
| 不变性 | ✓ store 内部全 spread；组件不持有本地状态（除 React Query mutation 状态）|
| 可读性 | 每文件顶部 import 分组（react → antd → @core → @/）；prop interface 与默认 export 紧邻 |

---

## 多皮肤兼容声明（PRD §7.5）

- 5 组件全部 `@core/*` alias 引用业务层 → 业务层无依赖增量 → webcode-v2/v3 不受本任务影响
- 实测 v2/v3 typecheck pre-existing fail（与 P2-b 无关）→ 详 `verify-T-0123-P2-b.md` §3 处置建议
- 组件文件仅落 `webcode/src/`，**不进 frontend-core**（按 §7.5 锁定）

---

## 发现的问题 / Action Items

| 严重度 | 描述 | 处置 |
|--------|------|------|
| LOW | InstancePicker GPV 探测延后 | 已记 PRD §O.9 待定点 1，P4 接入 |
| LOW | MmlEditor parseErrors 渲染用 `<ul>` 列表但 alert 内含 i18n stub 在测试里看不到清楚文本 | 测试已断言 alert 容器存在 + i18n key 字面命中；运行时 react-intl 会正确插值 |
| INFO | webcode-v2/v3 typecheck pre-existing fail | 不属本任务范围；建议进 backlog 新登记或并入 P2-d 收尾 |

**P0 / P1 严重度问题：0** —— S5 出口门可过。

---

## 出口门结论

- [✓] review 报告无 P0/P1 未解
- [N/A] 安全审查未触发（无 admin/middleware/auth*）
- [✓] DoD 每项打勾或 N/A（带理由）

**S5 PASS**。下一步 S6 commit。
