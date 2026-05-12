# T-0090 拆分子任务（MML 控制台 UX 整改）

> 由 dev-pipeline `/pick T-0090` 在 sprint-10 执行 S2 阶段产出（2026-05-11）。
> umbrella T-0090 在 backlog.md §3 Active，State=`in_design`；本文件持有 4 个执行子任务。
>
> **来源 PRD**：backlog.md T-0090 Notes（B 方案二次扩展 7 子项）+ `frontend-multi-skin-plan-20260422.md`（前端架构约束）
> **关联 review**：`docs/review-report/20260430/verify-T-0094.md`（解锁 deps：category_group 列已就位）

---

## 0. S2 设计备忘（sprint-10 deliverable）

### 7 子项 → 4 sub-task 映射

| 子项 | 内容 | 落 sub-task |
|------|------|-----------|
| ① | 操作类型差异化（MOD → "修改值入口"）| **a** (FE-only) |
| ② | 删 3 字段：参数配置 / 所属分类 / 适用产品类型 | **a** (UI) + **b** (DB column) |
| ③ | 命令编码 input → textarea | **a** |
| ④ | textarea 必须自定义、不允许选已有命令 | **a** |
| ⑤ | 私有命令页面参考公有页面（功能一致）| **d** |
| ⑥ | 私有命令按当前管理员所属分组过滤（RBAC）| **c** (backend) |
| ⑦ | 公有命令不按管理员过滤 | **c** (默认行为，仅需测试覆盖) |

### 执行顺序 / 依赖图

```
T-0090-a (S, FE-only)  ──┐
                          ├──→ T-0090-c (M-L, backend RBAC)
T-0090-b (M, DB)  ───────┤        │
                          │        ↓
                          └──→ T-0090-d (S, FE 私有页面，复用 a 的组件)
```

- **a** 独立先做（纯 FE，1-2 天）
- **b** 独立（DB 破坏性，需谨慎评估历史数据；可与 a 并行）
- **c** 依赖 **a** 的 UI 组件 + **b** 的 schema（如果 productTypes 删则 RBAC query 也清掉相关 JOIN）
- **d** 依赖 **a**（复用页面组件）

### Risk 评估（S2 输出）

| Risk | 触发条件 | 缓解 |
|------|---------|------|
| **R-NEW-1** productTypes 删 = 破坏性 migration | drop column 历史数据有内容时 | b 任务必须先做 `SELECT id, product_types FROM mml_custom_command WHERE product_types != '[]'::jsonb` 评估；非空行需 backfill 策略（迁移到 category_group 还是直接丢弃）|
| **R-NEW-2** RBAC private 查询 query 重写错误 | c 任务漏过滤 → 私有命令跨用户泄露 | c 必须加单测验 (admin_a 不能看 admin_b 的 private 命令)；e2e 加 2 角色互见用例 |
| **R-NEW-3** d 复用 a 组件耦合 | a 组件抽象不足 → d 复制粘贴 | a 阶段把可复用的 Drawer/Form/CommandInput 抽到 `components/`，d 直接 import |
| **R-NEW-4** category_group 清理状态 | T-0094 misdiagnosis 后已确认列存在 | b 阶段决议：保留 column 现状 + 仅删 productTypes，不动 category_group（review T-0094 已锁结论）|

### 文件改动面预估

| sub-task | 后端 | 前端 | DB | 测试 |
|----------|------|------|-----|------|
| **a** | — | webcode/src/pages/mml/Console/ + components/ + ScriptTaskDrawer.tsx 改约 5 文件 | — | 前端无单测约定，靠 typecheck |
| **b** | model.go + pg_repository.go + handler.go 删 product_types 字段（36 处引用清扫）| webcode 同步删 productTypes form/field/state | migrations/0000NN_drop_product_types.sql | mml 后端 service_test 验删后 build/test 全过 |
| **c** | service.go + pg_repository.go private query JOIN role_device_groups | — | — | service_test 加 admin context 注入 + RBAC 跨用户隔离用例 |
| **d** | — | webcode/src/pages/mml/ 新增 PrivateCommand 页面 + 复用 a 的组件 | — | typecheck |

### 验收标准

| sub-task | Given/When/Then |
|----------|----------------|
| **a** | Given Console 页打开 / When 选"修改值"操作类型 / Then UI 出现修改值入口；参数配置 / 所属分类 / 适用产品类型 3 字段已删；命令编码 input 改 textarea；textarea 内容不能从下拉选既有命令 |
| **b** | Given DB 有 mml_custom_command 数据 / When migrate-up / Then product_types 列删除；mml backend build/test 全过；前端无 product_types 引用 |
| **c** | Given admin_a 属 group_A、admin_b 属 group_B，各创建 1 条 private 命令 / When admin_a GET /mml/custom-commands?command_scope=private / Then 仅返自己的 1 条；admin_b 不可见；GET command_scope=public 仍返全部（admin context 不影响 public）|
| **d** | Given 私有命令页 / When 用户操作 add/edit/delete / Then 行为与公有命令页一致；UI 组件来自公共 components/；列表已按 RBAC 自动过滤（依赖 c） |

### sprint-11 执行计划（参考）

- 并行组：a + b（路径互斥，可发 2 sub-agent worktree）
- 串行组：a + b → c（依赖）→ d（依赖 a）
- 总 Est：S + M + M-L + S ≈ 6-7 工作日 = 1 个 Sprint 容量

---

## 1. Sub-task 表（13 列 schema 与 backlog.md §3 主表对齐）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0090-a | MML Console FE-only: 操作类型差异化（MOD→修改值入口）+ 删参数配置/所属分类/适用产品类型 UI section + 命令编码 input→textarea + textarea 自定义限制（不允许选既有命令）— **done 2026-05-12**（commit `bf71e518` 7 文件 +411/-198；抽 CommandCodeTextarea + OperationTypeWithModify 公共组件 → R-NEW-3 一半 mitigate（d 复用待落地）；后端 schema 兼容（empty default 保留至 b 真删 column））| feat | frontend | P2 | done | Claude | S (1-2d) | — | sprint-10 (pull-forward) | R-NEW-3（部分 mitigate）/ backlog T-0090 ①②③④ | 2026-05-12 | sprint-11 work pull-forward 进 sprint-10 buffer 执行；Closing Evidence → `docs/review-report/20260512/verify-T-0090-a.md` + `REVIEW_T-0090-a_chenbo01_mml.md` |
| T-0090-b | MML 后端 drop product_types 列 — migration 删 mml_custom_command.product_types + GIN 索引；扫净 ~29 处 Go 引用（model/service/handler/repo）+ FE 4 文件同步删 — **done 2026-05-12**（commit `86b19c3f`；live DB up/down/up 三次演练 / 6 行实测数据丢失符合 R-NEW-1 设计 / 精确边界守护 mml_custom_command 不动 mml_commands+indicator_definitions / migration 000087 重命名解 seed/000086 冲突 / Gin unknown-field silent ignore 保兼容老客户端）| ref | F06/mml+migration+frontend | P2 | done | Claude | M (2-3d) | — | sprint-10 (pull-forward) | R-NEW-1 已接受 / backlog T-0090 ② | 2026-05-12 | Closing Evidence → `docs/review-report/20260512/verify-T-0090-b.md` + `REVIEW_T-0090-b_chenbo01_mml.md`；解锁下游 T-0090-c (deps ready) |
| T-0090-c | MML 后端 RBAC 私有命令过滤 — service.go + pg_repository.go ListCustomCommands(scope=private) JOIN user→role_device_groups 派生当前 admin 可见 group_ids；公有命令路径不变；admin context 注入到 service 层（gin handler 转传 user_id）| feat | F06/mml+admin | P2 | triaged | — | M-L (3-4d) | T-0090-a, T-0090-b | sprint-11 | R-NEW-2（query 漏过滤泄露隐患）/ backlog T-0090 ⑥⑦ | 2026-05-11 | 复用 admin/permission_service.go 的 group_ids 派生逻辑；service_test 强制加 admin_a/admin_b 跨用户隔离用例；e2e_verify.sh +2 claim（admin_a 不可见 admin_b private + admin 看 public 全见）；query 重写需 review CRITICAL |
| T-0090-d | MML 前端私有命令页面新建 — webcode/src/pages/mml/PrivateCommand 复用 T-0090-a 抽出的 Drawer/Form/CommandInput 公共组件；功能与公有命令页一致；列表自动按 c 后端 RBAC 过滤；i18n + 路由注册 | feat | frontend | P2 | triaged | — | S (1-2d) | T-0090-a, T-0090-c | sprint-11 | R-NEW-3（与 a 组件耦合度）/ backlog T-0090 ⑤ | 2026-05-11 | 前置 a 必须做完且组件已抽到 components/；d 仅做"页面外壳 + 复用组件"不重复实现业务逻辑；typecheck + 多皮肤兼容性评估（webcode-v2/v3 需 review）|

---

## 2. 关联文档

- backlog umbrella: `docs/project/backlog.md` §3 T-0090
- sprint-10 任务来源: `docs/project/sprint/sprint-10.md`（本 S2 拆分是 sprint-10 唯一 T-0090 deliverable）
- T-0094 misdiagnosis verify: `docs/review-report/20260430/verify-T-0094.md`（解锁 category_group 决议）
- 前端架构: `docs/project/frontend-multi-skin-plan-20260422.md`（多皮肤约束）

---

## 3. sprint-11 sprint planning 时的建议

- 4 sub-task 全部 triaged → planned 升 sprint-11
- 推荐 commit 形态：a 1 commit + b 2 commit（migration + 36 处引用清扫）+ c 1-2 commit + d 1 commit = 总 5-6 commit / 1 个 PR（每 commit footer 各挂 `Backlog: T-0090-x`）
- T-0096 (MML script 弹窗) 跟随 T-0090-a 同 PR（语义已绑定 console productTypes 去留）
