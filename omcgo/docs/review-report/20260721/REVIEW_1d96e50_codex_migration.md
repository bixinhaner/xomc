# Review: MML catalog seed consolidation

结论：PASS_WITH_WARNINGS

## Scope

- `omcgo/migrations/seed/000001_init_seed.sql`

## Findings

### WARNING

- `go test ./...` 当前被既有 baseline 幂等性用例阻塞。失败行集中在本次合并块之前的既有 `INSERT INTO public.*` 段，例如 10756、10913、11007、11052 等；新增合并块已调整为 `ON CONFLICT DO NOTHING`，不再被该用例点名。

### INFO

- 本次把原独立 `000002_mml_catalog_group_cleanup.sql` 的 MML 目录清理逻辑合并进 consolidated seed `000001_init_seed.sql`，新环境初始化只需执行 001。
- 新增块使用先清理/更新再插入的方式，避免依赖页面手工修改，并符合 seed baseline 的静态幂等性规则。

## Verification

- `git diff --check`：通过
- `cd omcgo && go build ./...`：通过
- `000001_init_seed.sql` Up 段在当前 PostgreSQL 容器中 `BEGIN ... ROLLBACK` 执行：通过
- `cd omcgo && go test ./test/integration -run TestSeedBaselineHasOnConflict`：失败，剩余报错为既有 baseline INSERT 段缺少 `ON CONFLICT DO NOTHING`
- `cd omcgo && go test ./...`：失败，同上，失败 package 为 `github.com/omcgo/omcgo/test/integration`

## Risk

- 影响范围仅为 seed 初始化数据；不会改动运行时代码。
- 当前库若已记录 seed version 2，需要重建库或手动处理 goose seed 版本后，才能按纯 001 路径完整重放验证。
