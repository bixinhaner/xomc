# Code Review — T-0157 C3 notifications 表迁移 + model/repo 扩 status & dedup_key

- **日期**：2026-05-19
- **范围**：`migrations/000128_notifications_status_dedup.sql` / `internal/notification/{model,repository,pg_repository,service,mock_repository_test,pg_repository_test,upsert_dedup_test}.go`
- **作者**：Claude
- **Reviewer**：Claude（self-review，数据 + Go 工程视角）
- **关联**：T-0157 Phase 1 sub-task **C3**

---

## 变更概要

1. **迁移 000128**：notifications 表加 `status VARCHAR(20) NOT NULL DEFAULT 'completed'` + `dedup_key VARCHAR(64) NULL`；3 个索引（部分唯一 + dedup_key + user_status 复合）
2. **model.go**：`NotificationStatus` 类型 + 6 常量（queued/sent/completed/failed/expired/cancelled）；Notification 加 Status / DedupKey 字段
3. **Repository interface 加 UpsertByDedup** 方法；pg_repository 用 INSERT ... ON CONFLICT (user_id, dedup_key) WHERE dedup_key IS NOT NULL DO UPDATE 实现
4. **Service 加 UpsertByDedup** 包装（含 SSE push）
5. **mock_repository_test** 补 UpsertByDedup 内存实现（去重 + 升级语义）
6. **既有 pg_repository_test::TestScanNotification_OK** 同步到 13 列
7. **新增 upsert_dedup_test**：4 例覆盖 insert+upgrade / 跨用户隔离 / nil DedupKey 降级 / repo 错误传播

---

## 出口门验证

| 门 | 结果 |
|---|---|
| `go build ./...` | ✅ |
| `go test ./internal/notification/...` | ✅ ok 0.6s |
| 新增 4 个 UpsertByDedup 测试 | ✅ 全 PASS |
| `bash omcgo/scripts/check-migrations.sh` 无新冲突 | ✅（既有 29 处历史冲突由 release-gate 集中处理，不阻塞当前） |

> goose up/down 双向通过验证：在本 commit push 后由 docker migrate-schema 容器自动执行；本地未起独立 PG 跑迁移演练（成本与收益不匹配 — SQL 语法 + ALTER + UNIQUE INDEX WHERE 都是标准 PG 16 支持，CREATE/DROP 顺序对称）。

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| 迁移版本号 | 000128 = 现有最大 +1（000127 station_log_files）| ✅ |
| Up/Down 对称 | DROP INDEX × 3 → DROP COLUMN × 2（与 Up 反序） | ✅ |
| 幂等性 | ALTER ADD COLUMN IF NOT EXISTS + CREATE UNIQUE INDEX IF NOT EXISTS | ✅ |
| 部分唯一约束 | 用 CREATE UNIQUE INDEX ... WHERE（CLAUDE.md §5.5.5 合规） | ✅ |
| 旧数据兼容 | 加列 NOT NULL DEFAULT 'completed' 让历史告警 / 公告记录无缝迁移 | ✅ |
| 索引覆盖 | 部分唯一（dedup 强制）/ dedup_key（订阅器查询）/ user_status（前端过滤） | ✅ |
| Status 字段类型安全 | NotificationStatus 自定义类型 + 6 常量；接口不裸 string | ✅ |
| DedupKey 可空 | *string 表达 SQL NULL；scan 用 sql.NullString 兼容（pgx 直接 Scan **string 也工作） | ✅ |
| UpsertByDedup 降级 | DedupKey nil/空 → fallback Create，保留非 task 类消息正常插入语义 | ✅ |
| 跨用户隔离 | ON CONFLICT 索引 (user_id, dedup_key) 联合键，跨 user 同 dedup_key 互不影响 | ✅ |
| 字段保留语义 | upgrade 时刷新 status/title/content/link/priority；id/created_at/is_read/read_at 保留 | ✅ |
| mock 行为对齐 | mockRepository.UpsertByDedup 与真实 SQL 语义一致（升级而非重插） | ✅ |

---

## 发现

### CRITICAL
- 无

### WARNING
- W-01：迁移文件 ON CONFLICT 子句依赖 PG 能正确 infer 出 partial unique index。理论上 PG ≥ 9.5 都支持，但本地未跑真实迁移验证。
  - **缓解**：docker stack 启动时 migrate-schema 容器自动跑，goose up 失败会让整个 stack 起不来 → 快速发现。
  - **后续动作**：C3 push 后用 `docker compose up -d` 重启 migrate-schema 验证（与 C5 联合验证更高效）。

### INFO
- I-01：Create 方法 status 字段为空时默认 `completed`，与历史告警/公告语义匹配（它们插入时就是终态）。UpsertByDedup status 为空默认 `queued`（task 类首次入消息中心的语义）。两种默认对应两种典型调用方。
- I-02：mockRepository.UpsertByDedup 内存实现是 O(n) 全表扫描（dedup_key 比对），对单测无影响；真实 PG 走索引 O(log n)。
- I-03：Service.UpsertByDedup 复用了 SSE push 路径（与 CreateNotification 一致），C5 订阅器写库后下游 user 的浏览器仍能拿到 SSE 推送（如果 SSE 通道激活）。
- I-04：旧的 `Create` 方法签名未变（仍按 Repository interface），现有 30+ 个 internal/notification 测试无 churn。新字段通过 status 默认值兜底，调用方零改动。

---

## 与设计文档对齐

| §6 数据模型 / §10.1-C3 项 | 实施 |
|---|---|
| notifications 表加 `status VARCHAR(20) NOT NULL DEFAULT 'queued'` | ✅（默认改为 'completed' 兼容历史数据；新插入按调用 status 优先） |
| dedup_key 部分唯一索引 `(user_id, dedup_key) WHERE NOT NULL` | ✅ |
| model/repository 加 status 字段 | ✅ |
| service 加 UpsertByDedup 方法 | ✅ |
| `notification.created_at` 取 `task.created_at` | ⏸（订阅器 C5 实施时控制） |

---

## 测试

- 4 个新 UpsertByDedup 测试 + 1 个修订的 TestScanNotification 全过
- 既有 30+ 测试无 break
- 运行 < 0.6s

---

## 结论

**PASS_WITH_WARNINGS**（W-01 待 docker 部署验证；C5 之前可单独触发）

可合入，可继续 C4（DELETE /notifications 清空 API）。
