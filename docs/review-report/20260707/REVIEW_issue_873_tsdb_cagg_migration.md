# Review: Issue #873 TSDB continuous aggregate migration

Fixed point: `main`
Diff reviewed: working tree diff for `omcgo/migrations/tsdb/000002_create_pm_metrics_hourly_cagg.sql`
Spec source: GitHub Issue #873

## Standards

No CRITICAL findings.

- The migration now has explicit `-- +goose Up` and `-- +goose Down` sections, matching `docs/project/dod.md` and `omcgo/CLAUDE.md` migration requirements.
- Multi-statement migration content is wrapped in `-- +goose StatementBegin` / `-- +goose StatementEnd`, matching the documented goose guidance for complex statements.
- The Down section removes the continuous aggregate policy before dropping the materialized view, satisfying the documented requirement that Down removes objects created by Up.
- No SQL string construction, application code, frontend code, or security-sensitive path is changed.

## Spec

No CRITICAL findings.

- Issue #873 asks for the TSDB continuous aggregate migration to become a valid bidirectional goose SQL migration. The diff adds the missing Up/Down sections and statement boundaries.
- Issue #873 asks for rollback support. The diff adds policy removal and materialized view drop in Down.
- Issue #873 asks to keep creation safe for existing data. The diff adds `WITH NO DATA` to avoid immediate historical materialization during migration.
- The change stays within scope: no PM query behavior, schema redesign, API, or frontend changes are included.

## Verification Notes

- `go build ./...` passed.
- `go test ./...` currently fails in unrelated existing test code; user confirmed these failures are out of scope for this migration-only fix.
- Empty-volume Docker deployment succeeded with this migration.
- TSDB migration down/up rehearsal succeeded: version 2 -> version 1 -> version 2.
