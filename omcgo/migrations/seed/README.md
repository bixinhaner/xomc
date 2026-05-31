# Seed Migrations

种子数据迁移目录，独立 goose 版本表 `goose_db_version_seed`，由 compose 中的 `migrate-seed` 服务在 `migrate-schema` 完成后执行。

当前只有一份 baseline：`000001_init_seed.sql`（~17 MB，全量 seed 数据快照，2026-05-31 从 `OMCGO_ENV=test` 环境 pg_dump 生成）。

新增 seed 迁移从 `000002_xxx.sql` 起递增。完整规范、合并流程、回滚说明见 `omcgo/migrations/README.md` 和 `omcgo/CLAUDE.md` §5.5。
