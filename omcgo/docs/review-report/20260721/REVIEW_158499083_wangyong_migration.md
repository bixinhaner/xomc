# Review Report: MML seed business grouping and baseline consolidation

- Date: 2026-07-21
- Base: 158499083
- Author: wangyong
- Scope: migration
- Conclusion: PASS_WITH_WARNINGS

## Summary

This review covers the staged backend changes for consolidating schema/seed migrations into the 001 baseline, regrouping the MML command catalog by business meaning, and fixing new database deployment failures caused by a missing `refresh_mml_command_target_paths(uuid)` helper during seed execution.

## Findings

### WARNING

1. Existing databases that already recorded seed version 1 will not automatically replay the updated `000001_init_seed.sql`.
   - Files: `omcgo/migrations/seed/000001_init_seed.sql`
   - Impact: New deployments receive the corrected business catalog and compatibility helper. Existing deployments need either the already-applied manual sync or a future incremental migration if the same catalog changes must be replayed automatically outside this local environment.

### INFO

1. The command tree now filters deleted and deprecated groups, matching the existing deprecated-command behavior.
   - File: `omcgo/internal/mml/group_tree_repository.go`
   - Evidence: `g.deleted_at IS NULL` and `g.deprecated_at IS NULL` are applied before building the tree.

2. The interface binding group is intentionally flattened to two levels.
   - File: `omcgo/migrations/seed/000001_init_seed.sql`
   - Evidence: `MML350_G_INTERFACE_BINDING` contains direct query/modify command leaves for F1 and NG interface binding, while the intermediate F/NG technical groups are deleted.

3. The seed file defensively recreates `refresh_mml_command_target_paths(uuid)` before inserting sub-fields.
   - File: `omcgo/migrations/seed/000001_init_seed.sql`
   - Evidence: `CREATE OR REPLACE FUNCTION public.refresh_mml_command_target_paths(p_command_id uuid)` appears before seed data writes.

## Validation

- `cd omcgo && go test ./internal/mml` passed.
- `cd omcgo && go build ./...` passed.
- Fresh temporary database schema migration passed with `000001_init_schema.sql`.
- Fresh temporary database seed migration passed after deliberately dropping `refresh_mml_command_target_paths(uuid)`, confirming the new deploy failure path is covered.
- `git diff --staged --check` passed.

## Residual Risk

The biggest remaining risk is operational rather than code-level: because the requested consolidation moved follow-up data changes into 001, already-initialized databases will not pick them up through goose versioning unless they are manually synced or receive a later non-001 patch migration.
