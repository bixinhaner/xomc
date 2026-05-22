# 代码审查报告

- **审查对象**：`omcgo/migrations/` 三个 000146 重号文件去重
- **基线 commit**：`2d7bdd77`（HEAD before fix）
- **作者**：shangyingbin
- **审查时间**：2026-05-22
- **审查类型**：HOTFIX（部署阻塞修复）
- **结论**：**PASS**

---

## 1. 背景

Docker 部署执行 `migrate-schema` 时 goose panic：

```
panic: goose: duplicate version 146 detected:
  /etc/omcgo/migrations/000146_topo_nodes_performance_indexes.sql
  /etc/omcgo/migrations/000146_devices_last_param_sync_failed.sql
```

实际 `omcgo/migrations/` 下同时存在 **3 个 000146**：

| 提交时间 (2026-05-21) | 提交哈希 | 文件 |
|---|---|---|
| 14:28:05 | dc518a15 | 000146_topo_nodes_performance_indexes.sql |
| 14:36:59 | 7da9fcc2 | 000146_ufte_task_types_sort_order.sql |
| 14:40:27 | 0d3d6601 | 000146_devices_last_param_sync_failed.sql |

三笔 PR 在同一天先后合入 main，各自基于不同的版本号头部，撞车在 000146。`migrations/seed/`、`migrations/` 之间编号独立，本次冲突仅在 `migrations/` 内。

T-0080 是同类先例（000038 → 000049 rename hotfix），处理方式一致。

## 2. 修复方案

按 commit 时间先后递增，保留最早提交的 000146，后续两笔顺延：

| 原文件 | 新文件 |
|---|---|
| 000146_topo_nodes_performance_indexes.sql (dc518a15) | 000146 保持 |
| 000146_ufte_task_types_sort_order.sql (7da9fcc2) | → 000147 |
| 000146_devices_last_param_sync_failed.sql (0d3d6601) | → 000148 |

`git mv` 操作，`similarity index 100%` —— SQL 内容零改动。

## 3. 审查项

### 3.1 版本号唯一性 ✓
```
000145_migrate_logcollect_rows_to_new_tables.sql
000146_topo_nodes_performance_indexes.sql
000147_ufte_task_types_sort_order.sql
000148_devices_last_param_sync_failed.sql
```
连续递增、无重复。

### 3.2 内容完整性 ✓
`git diff --cached` 显示 `similarity index 100%`，仅文件名变化，SQL 语句、`-- +goose Up/Down`、约束、索引全部原样。

### 3.3 goose 执行顺序影响 ✓
三个迁移之间无 schema 依赖（分别是 topology 索引、ufte sort_order、devices 列新增），重排执行顺序不影响最终 schema 等价性。已在容器内验证：`migrate-schema` 与 `migrate-seed` 均 `exited (0)`。

### 3.4 跨目录冲突 — 不在本次范围
`check-migrations.sh` 报 37 处 `migrations/` 与 `migrations/seed/` 同号告警属于历史遗留，goose 分目录运行不冲突，由 release-gate 集中清理（与 T-0080 处理边界一致）。

### 3.5 回滚 ✓
若需回滚，goose down 按新版本号倒序执行即可（单文件 Up/Down 模式）。

## 4. 部署验证

修复后重跑 `bash /Users/shangyingbin/project/omc-docker/docker-run.sh`：

- `migrate-schema`: exit 0
- `migrate-seed`: exit 0
- app/acs/worker/web: running
- `curl http://localhost:8081/healthz` → 200
- `curl http://localhost:7557/healthz` → 200

## 5. 后续建议（S7 补登记）

- backlog 补登一个 `T-NNNN` hotfix 任务记账（同 T-0080 模式）
- 长期：CI 加迁移版本号唯一性预检（`check-migrations.sh` 已有逻辑，可在 pre-merge 启用阻塞模式），避免三笔 PR 并行合入再撞车

## 6. 风险

无。文件改名不改 SQL，goose 重排无副作用，部署已验证。
