# Code Review — T-0164 G1 BUG-6 修复：pm_metrics 自然键纳入 object_ldn

| 字段 | 值 |
|------|----|
| Commit | `PENDING`（commit 后回填） |
| Author | shangyingbin |
| Scope | pm + migration |
| Backlog | T-0164（G1 真机闭环延续，BUG-6） |
| Reviewer | Claude Opus 4.7 |
| Date | 2026-05-25 |
| Verdict | **PASS_WITH_INFO** |

---

## 背景

承接 commit `70c382c5`（T-0164 G1 publish 链路修复），真机 BLQ
`1202000240194DP0015` 17:00 BJ 上传后 ACS publish ✓ / Collector parse 2048
counter ✓，但批量 INSERT `pm_metrics` 时撞：

> `SQLSTATE 21000: ON CONFLICT DO UPDATE command cannot affect row a second time`

retry 3 次进 DLQ，`pm_metrics` 0 行。

### 根因

唯一索引 `uq_pm_metrics_natural` 列清单：

```
(device_oui, device_sn, metric_path, granularity, end_time, time)
```

**不含 `object_ldn`**。PM 文件中同一 `<measType>` 下多个 `<measValue>`
携带不同 `MeasObjLdn`（cell-1 / cell-2 / ...）但 counter_name 相同时，
两行的自然键值完全一致 → 同一条 INSERT 语句 ON CONFLICT 二次命中同一目标行，
PG 直接拒绝。

---

## 变更范围

| 文件 | 类型 | 行变化 |
|------|------|--------|
| `migrations/000171_pm_metrics_natural_key_object_ldn.sql` | 新增 | +50 |
| `internal/pm/metrics/pg_repository.go` | 修改 | +24 / -10 |
| `internal/pm/metrics/pg_repository_test.go` | 修改 | +79 / -0 |
| `internal/pm/metrics/model.go` | 修改（注释） | +5 / -2 |

---

## 审查项

### 1. Migration 000171（数据与存储专家）

- [x] **版本号连续**：本地最大 `000170` → 新增 `000171`，无跳跃
- [x] **Up/Down 配对**：Down 段还原 NULLABLE + 旧索引 + 旧部分索引 WHERE
- [x] **StatementBegin/End 包裹**：纯 DDL 不含 PL/pgSQL DO 块，理论上可不包；保留包裹以利后续可能的扩展（与 §5.5.1 不冲突）
- [x] **TimescaleDB 唯一索引**：保留分区列 `time`，OK（§5.5.2）
- [x] **DEFAULT '' + NOT NULL 顺序**：先 `UPDATE` 填 NULL → `SET DEFAULT` → `SET NOT NULL`，避免 ALTER 拒绝
- [x] **部分索引修正**：原 `idx_pm_metrics_object_ldn WHERE object_ldn IS NOT NULL`
  在新约束下 WHERE 永真等于普通全列索引，重建为
  `WHERE object_ldn <> ''`（仅索引有 cell 信息的行），减索引体积
- [x] **CHECK 约束**：未引入新 CHECK，与 §5.5.4 一致
- [ ] **INFO（已知）**：pm_metrics 是 hypertable，`DROP INDEX` 和 `CREATE UNIQUE INDEX`
  会级联到所有 chunk；现网 0 行（撞 BUG-6 全部 DLQ），重建瞬时完成无锁风险

### 2. Repository 层（Go 工程专家）

- [x] **ON CONFLICT 子句**：列清单与新 UNIQUE 索引列序一致，便于人读
  （PG 按集合匹配，但保持列序减歧义）
- [x] **nil → '' 落盘兜底**：`var ldn interface{}` 改 `ldn := ""`，避免
  UNIQUE 中 `NULL ≠ NULL` 破坏幂等
- [x] **抽 `buildBatchInsertSQL`**：纯函数化便于单测；BatchInsert 保留
  resource lifecycle 职责（pool.Exec）
- [x] **`pmMetricsUpsertSuffix` 常量**：消除魔法字符串，单测可断言常量
- [x] **错误 wrap 一致**：`fmt.Errorf("...: %w", err)`
- [x] **无 SQL 字符串拼接**：仍走 Squirrel `Insert(...).Columns(...).Values(...)`

### 3. 单元测试（测试专家）

- [x] `Test_buildBatchInsertSQL_ON_CONFLICT_Includes_ObjectLDN` —
  BUG-6 回归：ON CONFLICT 子句必须含 `object_ldn`
- [x] `Test_buildBatchInsertSQL_MultiCell_NoNaturalKeyCollision` —
  三 cell 同 counter_name 落参数不合并、不含 NULL 字面量
- [x] `Test_buildBatchInsertSQL_NilObjectLDN_FallsBackToEmptyString` —
  nil 兜底为 `''`
- [x] 测试不依赖 DB / pool / context；执行确定性
- [x] 命名 `Test_<Function>_<Scenario>` 与文件其他测试一致

### 4. 聚合表是否同病（数据与存储专家 + 业务专家）

- [x] `aggregator.go:buildCountersSQL` 通过
  `GROUP BY (oui, sn, metric_path, statis_type)` 把多 cell 合并为单行
  + `MIN(object_ldn)` 取代表值 → 每 group 输出**一行**，不会撞键
- [x] `conflictTargetForTable` 各聚合表自然键：
  - `pm_metrics_hourly` 含 `time` 不含 ldn — OK（聚合表按设备级粒度）
  - `pm_metrics_daily/weekly/monthly` 走 PK — OK
  - `pm_group_metrics_*` 按 device_group_id 聚合 — OK
- **结论**：聚合表设计本就不需要 object_ldn 维度，不改

### 5. 安全合规

- [x] 未引入 SQL 注入面（Squirrel 参数化）
- [x] 未引入鉴权/日志泄露
- [x] 迁移操作幂等（DROP IF EXISTS）

### 6. 部署 & 真机验证

- [x] migration 000171 已应用 docker pg：`goose_db_version` 显示 171
- [x] `\d pm_metrics` 确认 object_ldn 列 NOT NULL DEFAULT '' + UNIQUE 索引已含 object_ldn
- [x] ACS `/healthz` 返回 200
- [ ] **待真机下个 15min PM 周期上传验证**：预期 `pm_metrics` 行数 > 0
  + `device_sn='1202000240194DP0015'`

---

## 结论

**PASS_WITH_INFO** — 修复方案清晰针对 BUG-6 根因；migration 风险可控
（0 行数据 + 索引重建瞬时）；单测覆盖足够。

真机闭环验证由 Operator 在 commit 后 15-30 分钟跟踪 `pm_metrics` 行数确认。
