# OMC 旧库升级兼容与坐标查询修复设计

## 背景

78 环境由旧版迁移到最新基线后，`goose_db_version` 已到 29，但
`parameter_sync_requests` 缺少最新基线中的 durable routing 字段和配套表。
原因是历史 2–24 号迁移合并进 `000001` 后，后续功能继续直接修改基线；
已有数据库不会重新执行 `000001`。同时 `GetCoordinates` 使用无别名的
`devices`，却复用了引用 `d.deleted_at` 的统一软删除条件。

## 方案

1. 保持 `000001` 不变，新增 `000030` 幂等桥接迁移。
2. 桥接迁移只补 durable parameter-sync routing 所需的缺失列、表、约束和索引；
   所有建表、加列和索引操作可在旧库和全新库安全执行。
3. `GetCoordinates` 改为 `FROM devices d`，继续复用统一软删除条件。
4. 增加静态迁移契约测试和 SQL 生成测试，分别覆盖旧库升级入口与别名错误。
5. 在 78 上重新部署后确认 migration 30、所有健康端点、错误日志归零，并重新采样
   IO PSI、PostgreSQL 活跃查询及 NATS KPI backlog。

## 不采用的方案

- 直接修改 `000001`：只能修复全新安装，不能修复现有数据库。
- 仅在 78 手工 `ALTER TABLE`：无法保护后续升级环境，且不可审计。
- 重放已删除的 2–24 号迁移：会与已部分存在的对象冲突，升级风险更大。

## 成功标准

- `GetCoordinates` 生成的 SQL 包含 `FROM devices d`。
- 迁移 30 在已有对象和缺失对象两种状态下都可执行。
- 78 的 `parameter_sync_requests.source_event_id` 等字段和配套表存在。
- 最新版本的 app、ACS、worker、NATS、MinIO、Web 健康端点均为 HTTP 200。
- 新启动日志不再出现 `source_event_id does not exist` 或
  `missing FROM-clause entry for table "d"`。
