# Review: MML Device Info Regroup

Date: 2026-07-21
Branch: feat/mml-device-info-regroup
Base: 818ca55
Scope: mml / migration
Result: PASS_WITH_WARNINGS

## Summary

本次审查覆盖 MML 分组命名补充、命令树隐藏 deprecated 命令、设备信息重分组 seed 合并迁移，以及对应 QA 明细文档。

## Findings

- CRITICAL: None.
- WARNING: `go test ./...` 在 `github.com/omcgo/omcgo/internal/paramsync` 的 `TestGPVFaultRecovererReleasesReplacementBeforeReturning` 失败，断言 outbox fallback 计数期望 1 实际 0。失败包不在本次 MML 改动路径内，但全量测试门未完全绿。
- INFO: `GOOSE_TABLE=goose_db_version_seed go run ./cmd/migrate up --path migrations/seed --dsn ...` 返回 `no migrations to run. current version: 13`，本地库 seed 版本高于新增 `000004`，因此 goose up 未实际执行该文件。

## Validation

- `go build ./...` — PASS.
- `go test ./internal/mml` — PASS.
- `go test ./...` — FAIL, limited to `internal/paramsync`.
- Seed SQL Up 段通过本地 PostgreSQL 事务演练并回滚：执行 `BEGIN; <Up SQL>; ROLLBACK;` 成功，涉及 UPDATE / INSERT / DELETE 均无 SQL 或约束错误。

## Notes

- 合并后的 seed 文件只保留一组 `-- +goose Up` 和一组 `-- +goose Down`，原 000004..000007 内容按源文件编号顺序拼接。
- Down 为 no-op，符合这批规范化/重建类 seed 的既有处理方式。
