# MML 用户模板表 — 命名规范化 + 所有权 FK 升级 多阶段方案

> 状态：**待审核（rename 部分），#3 Phase 1 已落地**
> 关联代码：`migrations/000133`、`migrations/000134`、`internal/mml/{model,handler,service,pg_repository}.go`
> 关联讨论：上一轮深度核查 §6 缺口清单

---

## 1. 背景

`mml_custom_command` 表的命名容易与 `mml_commands`（catalog 字典表）混淆，但二者本质是
"definition vs. instance" 两种不同领域实体（详见
`mml-custom-command-vs-mml-commands-architecture.md` 同期分析）。

历史成因：000014 初版表名为 `mml_templates`；000026 改名为
`mml_custom_command` 时未充分考虑与 `mml_commands` 的歧义。今天的代码里出现：

| 实体 | 含义 |
|---|---|
| `mml_commands` | 命令字典（catalog / definition） |
| `mml_custom_command` | 用户模板（user template / instance） |

并行还有一个所有权字段升级问题：`mml_custom_command.creator VARCHAR(100)`
（用户名字符串）正在被 `owner_user_id UUID`（外键到 `users.id`）替代。

本方案分两个独立但相关的改进，逐步推进。

---

## 2. 已完成（#1 + #3 Phase 1）

| Migration | 内容 | 状态 |
|---|---|---|
| `000133_mml_commands_drop_unused_visibility.sql` | 删除 `mml_commands.visibility` + `owner_user_id`（v2.3 方案预留但未实现的合表脚手架） | ✅ 已落地 |
| `000134_mml_custom_command_owner_user_id.sql` | 加 `mml_custom_command.owner_user_id UUID FK → users.id ON DELETE SET NULL` + 一次性 backfill + 索引 | ✅ 已落地 |
| `internal/mml/model.go` | `MMLCustomCommand` 加 `OwnerUserID *uuid.UUID` 字段 | ✅ |
| `internal/mml/pg_repository.go` | INSERT / scan 包含新列；Create 走 dual-write | ✅ |
| `internal/mml/handler.go` | `CreateTemplate / CloneTemplate` 从 gin ctx 取 user_id 注入 | ✅ |

**写路径已全部 dual-write**；新增行的 `owner_user_id` 在 caller 携带 JWT 用户上下文时
会正确写入，缺失时落 NULL（与历史脏数据兼容）。

**读路径不变**：visibility 过滤仍按 `creator (username)` + RBAC group-share 走。

---

## 3. 计划中（#2 表 rename + #3 Phase 2-3）

### 3.1 #3 Phase 2 — 读路径切换到 owner_user_id

**前置条件**：生产 backfill 跑完并验证

```sql
-- 验证 1：核对未命中比例
SELECT COUNT(*) FILTER (WHERE owner_user_id IS NULL AND creator IS NOT NULL AND creator <> '') AS unbacked,
       COUNT(*) AS total
FROM mml_custom_command;

-- 验证 2：抽样 mismatch
SELECT id, creator, owner_user_id, (SELECT username FROM users WHERE id = owner_user_id) AS resolved
FROM mml_custom_command
WHERE owner_user_id IS NOT NULL
ORDER BY random() LIMIT 50;
```

若 `unbacked / total < 1%`（可接受），进入 Phase 2。

**代码改动**：

1. `pg_repository.go` `PgCustomCommandRepository.List` 的 visibility 子句：
   ```go
   // 旧：creator == filter.Creator OR creator IN (group-share JOIN users.username)
   // 新：owner_user_id == filter.UserID OR owner_user_id IN (group-share JOIN users.id)
   if filter.UserID != nil && *filter.UserID != uuid.Nil {
       privateClauses = append(privateClauses, sq.Eq{"owner_user_id": *filter.UserID})
   }
   if len(filter.VisibleGroupIDs) > 0 {
       privateClauses = append(privateClauses, sq.Expr(
           `EXISTS (
               SELECT 1 FROM user_roles ur
               JOIN role_device_groups rdg ON rdg.role_id = ur.role_id
               WHERE ur.user_id = mml_custom_command.owner_user_id
                 AND rdg.group_id = ANY(?)
           )`,
           filter.VisibleGroupIDs))
   }
   ```
   注意：group-share 子查询省掉了 `JOIN users.username = creator` 这一跳，性能直接提升。

2. `service.go` `UpdateCustomCommand` / `DeleteCustomCommand` 的 ownership 校验（**当前完全缺失**）：
   ```go
   if !filter.IsSuperAdmin {
       if existing.OwnerUserID == nil || *existing.OwnerUserID != currentUserID {
           return ErrForbidden
       }
   }
   ```
   这一步同时**修复深度核查 §6 缺口 #1 + #2**（普通用户能改/删别人模板的越权 bug）。

3. 单测：补 RBAC test case 覆盖 (creator==self) / (group-share) / (super_admin bypass) 三路在 owner_user_id 路径上的语义一致性。

### 3.2 #3 Phase 3 — Deprecation 清理

**前置条件**：Phase 2 在生产稳定运行 ≥ 1 个 release（约 4 周）

**Migration 000XYZ**：
```sql
ALTER TABLE mml_custom_command
    ALTER COLUMN owner_user_id SET NOT NULL;

-- creator 暂保留供历史 audit 查询，但下个 release 可 DROP
COMMENT ON COLUMN mml_custom_command.creator IS
    'DEPRECATED: 历史用户名字符串字段。Phase 2 起读写均改走 owner_user_id。
     保留供历史 audit 行查询，下个 release 评估后 DROP。';
```

**Migration 000XYZ+1**（再下个 release）：
```sql
ALTER TABLE mml_custom_command DROP COLUMN creator;
```

### 3.3 #2 表 rename — `mml_custom_command` → `mml_user_templates`

**为什么单独排序在 #3 之后**：rename 涉及面虽广但是纯净的"显示层"改动；
而 #3 Phase 2 修复的是**真实安全漏洞**（越权 update/delete）。先后顺序由价值决定。

**Migration 000XYZ+2**（rename 与 deprecation 不互斥）：
```sql
-- +goose Up

-- 1. 表 rename
ALTER TABLE mml_custom_command RENAME TO mml_user_templates;

-- 2. 索引 rename（保持索引命名与表对齐）
ALTER INDEX IF EXISTS idx_mml_custom_command_command_code RENAME TO idx_mml_user_templates_command_code;
ALTER INDEX IF EXISTS idx_mml_custom_command_scope_creator RENAME TO idx_mml_user_templates_scope_creator;
ALTER INDEX IF EXISTS idx_mml_custom_command_product_types_gin RENAME TO idx_mml_user_templates_product_types_gin;
ALTER INDEX IF EXISTS idx_mml_custom_command_parameters_gin RENAME TO idx_mml_user_templates_parameters_gin;
ALTER INDEX IF EXISTS idx_mml_custom_command_owner RENAME TO idx_mml_user_templates_owner;

-- 3. 触发器 rename
ALTER TRIGGER trigger_mml_custom_command_updated_at ON mml_user_templates
    RENAME TO trigger_mml_user_templates_updated_at;

-- 4. 约束 rename
ALTER TABLE mml_user_templates RENAME CONSTRAINT chk_mml_custom_command_scope TO chk_mml_user_templates_scope;
ALTER TABLE mml_user_templates RENAME CONSTRAINT chk_mml_custom_command_op TO chk_mml_user_templates_op;
ALTER TABLE mml_user_templates RENAME CONSTRAINT fk_mml_custom_command_owner TO fk_mml_user_templates_owner;

-- 5. 向后兼容视图：旧表名作为 read-only 视图保留 1 个 release
CREATE OR REPLACE VIEW mml_custom_command AS SELECT * FROM mml_user_templates;
COMMENT ON VIEW mml_custom_command IS
    'DEPRECATED 兼容视图（rename to mml_user_templates）；下个 release DROP。
     新代码请直接引用 mml_user_templates 表。';

-- +goose Down
DROP VIEW IF EXISTS mml_custom_command;
ALTER TABLE mml_user_templates RENAME TO mml_custom_command;
-- 索引/触发器/约束 rename 逆向略，按上文反向执行
```

**配套 Go 代码改动**（约 19 处）：

| 文件 | 改动 |
|---|---|
| `internal/mml/pg_repository.go` | ~13 处 SQL 字符串 `"mml_custom_command"` → `"mml_user_templates"` |
| `internal/mml/service.go` | 1 处注释提及 |
| `internal/mml/model.go` | 类型 `MMLCustomCommand` 是否同步改名为 `MMLUserTemplate`？**建议保留**（避免连锁改动）；表名归表名，Go 类型归 Go 类型 |
| `internal/mml/repository.go` | 接口名 `CustomCommandRepository` 同上 |

**滚动部署窗口**：
- T+0：Migration 上 → 表已 rename，旧名走兼容视图，旧 binary 仍能读但不能写视图
- T+1：app 重启 → 新 binary 用新表名直读直写
- T+1 release：删除兼容视图（独立 migration）

### 3.4 时序与建议节奏

```
本期（已完成）：
  ✅ #1 drop dead columns         ← 000133
  ✅ #3 Phase 1 add FK + dual-write ← 000134

下个 release：
  → 跑生产 backfill 验证（§3.1 SQL）
  → #3 Phase 2: read 切 owner_user_id + 修 Update/Delete 越权 ✨ 顺手修真 bug
  → 单测补齐

再下个 release（如稳）：
  → #3 Phase 3: owner_user_id NOT NULL + creator 标 deprecated
  → #2 表 rename + 兼容视图

更后续：
  → DROP creator 列
  → DROP 兼容视图
```

---

## 4. 风险与回滚

| 风险 | 缓解 |
|---|---|
| **Phase 1 backfill 漏掉用户名变更后的孤儿** | Phase 1 不阻塞 — owner_user_id 为 NULL 时读路径继续走 creator；运维可视化未命中比例 |
| **Phase 2 读切换后 group-share JOIN 性能退化** | 新 SQL 省了 `users.username = creator` 这一跳；新加的 `idx_mml_custom_command_owner` 覆盖 owner_user_id 查询；预期更快 |
| **rename 期间旧 binary 写视图失败** | 建立兼容 VIEW 是 SELECT-only；旧 binary 在窗口期能读、不能写；窗口期 ≤ deploy 间隔，可接受 |
| **Phase 3 NOT NULL 升级时仍存在 NULL 行** | 升级前 SQL 验证 `WHERE owner_user_id IS NULL` = 0；不为 0 则人工修复或推迟 |
| **回滚** | 每个 migration 自带 Down；除 Phase 3 NOT NULL 外都可平滑回滚 |

---

## 5. 决策清单（待审核确认）

| # | 决策点 | 默认建议 |
|---|---|---|
| D1 | Phase 2 改 SQL 时是否同时修 Update/Delete 越权（深度核查 §6 #1+#2 缺口）？ | ✅ 同 PR 修，避免两次涉及同模块文件的合并冲突 |
| D2 | rename 时 Go 类型 `MMLCustomCommand` / `CustomCommandRepository` / `CustomCommandFilter` 是否同步改名？ | ❌ 不改 — 表名归表名、Go 类型归 Go 类型，避免 50+ 个调用点联动 |
| D3 | 兼容 VIEW 保留多久？ | 1 个 release（约 4 周），独立 migration 删除 |
| D4 | Phase 3 `creator` 列何时 DROP？ | rename 之后再下个 release（确认无审计查询依赖） |

---

## 6. 关联资料

- 深度核查报告（上一轮对话）— 命名分析 §6、CRUD 缺口 §6
- `omcgo/migrations/000014_mml_templates_and_audit.sql` — 初版表（彼时叫 mml_templates）
- `omcgo/migrations/000026_mml_table_restructure.sql` — 改名到 mml_custom_command（造成今日歧义的源头）
- `omcgo/migrations/000095_mml_command_catalog_v2.sql` — mml_commands 升级为关系图节点
- `omcgo/migrations/000132_mml_console_v23_catalog.sql` — 加 visibility / owner_user_id 到 mml_commands（已由 000133 撤销）
- `omcgo/migrations/000133_mml_commands_drop_unused_visibility.sql` — 本期 #1
- `omcgo/migrations/000134_mml_custom_command_owner_user_id.sql` — 本期 #3 Phase 1
