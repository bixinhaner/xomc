# 代码审查报告：MML 配置脚本整合与 standard_params 兼容迁移

- 日期：2026-07-21
- 作者：wangyong
- Scope：migration
- 结论：PASS

## 审查范围

- 新增 `migrations/000003_add_standard_params_updated_fields.sql`，为旧开发库补齐 `standard_params.updated_fields`。
- 新增 `cmd/migrate/standard_params_migration_test.go`，覆盖旧 schema 执行迁移后的字段默认值。
- 新增统一入口 `scripts/mml_apply_config_updates_20260721.sql`，合并原 MML apply / verify / rollback / reset 脚本。
- 删除被统一入口替代的旧 `scripts/mml_*.sql` 文件。

## 发现

### CRITICAL

无。

### WARNING

无。

### INFO

- 定向迁移测试首次真实连接本地 Postgres 时发现 `text[]` 不能直接 scan 到 `[]string`；已调整为读取数组字面量并断言为空数组，避免测试只在 skip 或缓存路径下通过。
- 统一 MML SQL 文件体量较大，review 重点检查了入口分支、事务边界、verify 使用 `ROLLBACK`、rollback/reset 分支和危险 SQL 模式；未发现字符串拼接 SQL、裸 `panic` 或运营商硬编码逻辑。

## 验证

- `go build ./...`：通过。
- `go test ./...`：通过。
- `OMCGO_TEST_DB_DSN='postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable' go test ./cmd/migrate -run TestStandardParamsUpdatedFieldsMigrationUpgradesLegacySchema -count=1 -v`：通过。
- 临时 schema 迁移演练：`go run ./cmd/migrate --dsn <temp search_path dsn> --path <temp migration dir> up` 后确认 `updated_fields:NO:'{}'::text[]`；随后 `down` 后确认列计数为 `0`。

## 影响与风险

- 影响后端迁移与 MML 配置维护脚本，不涉及前端三皮肤。
- 新迁移使用 `ADD COLUMN IF NOT EXISTS` / `DROP COLUMN IF EXISTS`，对已存在字段或回滚重复执行具有容错性。
- 统一 SQL 入口保留 verify、rollback、reset extension 和默认 apply 模式，降低多脚本分散执行的维护成本。
