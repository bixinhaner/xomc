# Code Review — T-0096 (ScriptTask Drawer 删 productType)

**Reviewer**: Claude (self-review)
**Date**: 2026-05-12
**Scope**: 1 file / 3 lines net deletion / Type=ref / FE-only
**Related**: [`verify-T-0096.md`](./verify-T-0096.md)

---

## §1. Findings 严重度汇总

| Severity | Count |
|----------|-------|
| **CRITICAL (P0)** | 0 |
| **HIGH (P1)** | 0 |
| **MEDIUM (P2)** | 0 |
| **LOW (P3)** | 1 |
| **NOTE** | 2 |

**APPROVE** — 极小 refactor。

---

## §2. 审查

**核心改动**（ScriptTask/index.tsx 3 处）：
- TaskForm interface 删 `productType: string` 字段
- openEditDrawer setFieldsValue 删 `productType: ''` 初始值
- Drawer Form 体删 `<Form.Item productType>` block + 注释 +2 行说明保留 productTypeOptions 给 filter bar

**Scope 边界守护**：仅删除"更新弹窗"内的 productType UI；filter bar + productTypeOptions 完整保留（subtask 明示反向 deps T-0090 — console 决定 productTypes 去留 → script 跟随）

**API contract 检视**：submit payload 本就未传 productType（L374-389 createTaskMutation 调用前的 payload 对象不含此字段）— 表单字段是"孤儿"，删除无后端影响

**LOW-1**：filter bar 仍含 productTypeOptions 过滤选项；长期看若新任务都不写 productType，filter 也无意义；建议**未来单独 sub-task** 同 P3 清理（不在本任务 scope）

**NOTE-1**：注释 `T-0096：删 productType Form.Item 与 console「保存脚本」弹窗对齐` inline 在 Drawer 内保留 — 解释"为什么这里突然不见 productType"；运维 6 个月后看代码不会困惑

**NOTE-2**：ScriptTask 内 pre-existing 3 lint errors (scriptFilterFields/handleScriptSearch/handleScriptReset unused) 与本任务无关；这些 vars 占位但 filter bar 似未在 JSX 中真实渲染 — 属本 page 早期 refactor 残余，不在本任务责任面

---

## §3. DoD 逐项核销

### 编译与类型
- [N/A] 后端（FE-only）
- [✓] 前端 typecheck PASS
- [⚠️] 前端 lint：本任务文件 3 pre-existing errors（与本任务无关）；改动行未引入新 error

### 测试
- [N/A] 新增单测 — 纯 UI 删除，无新行为；subtask 文件 Notes "纯 FE 低风险 typecheck PASS 即可" 适用
- [N/A] 成功 + 失败两路径
- [N/A] 新 REST 端点 E2E — 无新端点
- [N/A] 修复 bug 回归 — type=ref 行为不变
- [✓] 禁止禁用失败测试 — 0 禁用

### 迁移与数据
- [N/A] 全部（无迁移）

### 代码规范
- [✓] 无 TODO / FIXME / panic
- [N/A] 后端规范
- [✓] 无 `any`
- [N/A] zap / Carrier / SQL

### 文档
- [✓] PR 说明含 Why（"与 T-0090-a console 弹窗对齐 / 关闭 productTypes UI 残留"）
- [✓] 关联 Backlog: T-0096
- [N/A] 关联 PRD — ref 不需要 PRD（per §C）
- [N/A] CLAUDE.md / Swagger 同步

### 流水线闭环
- [ ] commit footer 五元组（待 S6）
- [ ] backlog.md Task 状态回写（待 S7：T-0096 triaged → done，移出 §4 Triaged）
- [N/A] 快速通道 postmortem — type=ref 走 S1+S2-S7 完整

### 安全
- [N/A] 全部（FE-only 删字段，无新攻击面）

### 可观测性
- [N/A] 全部

### 模块特定（前端）
- [✓] 改动定位正确：ScriptTask 页面在 `webcode/src/pages/mml/ScriptTask/`
- [✓] 多皮肤评估：v2/v3 无影响（本任务仅改 webcode 单文件，不动 frontend-core）

---

## §4. 结论

**APPROVE** — 无 P0/P1；1 LOW (filter bar 长期清理) 留 future；2 NOTE 仅记录。

可以进 S6 commit。
