# Review Report

## Summary

- Scope: MML seed catalog and incremental MML config update script.
- Result: PASS_WITH_WARNINGS
- Reviewer: Codex
- Date: 2026-07-23

## Staged Files

- `omcgo/migrations/seed/000001_init_seed.sql`
- `omcgo/scripts/mml_apply_config_updates_20260721.sql`

## Findings

### CRITICAL

None.

### WARNING

- `bash omcgo/scripts/check-migrations.sh` still reports pre-existing migration numbering issues: main migrations skip `000002`, and `migrations/000001_init_schema.sql` conflicts by version id with `migrations/seed/000001_init_seed.sql`. This change did not add a new migration file or introduce a new version conflict, but the repository-level checker exits non-zero until the existing numbering policy is cleaned up.

### INFO

- Fresh-environment seed state was verified after local Docker volume reset and rebuild. `DefaultIpRoute` has both LST/MOD commands under `chapter:SL` / `WAN口配置参数管理`; `MML350_G_DEVICE_ETHERNET` and `chapter:SD` are absent.
- Incremental behavior was verified by simulating the old catalog state and applying the focused SQL for this change. The result matched the target state.
- The full historical `mml_apply_config_updates_20260721.sql` still has older precondition checks for legacy commands that are absent from the current consolidated baseline before reaching this change's later block. The focused incremental block for this change is idempotent and passed.

## Validation

- `git diff --check` — PASS
- `go build ./...` from `omcgo/` — PASS
- `go test ./...` from `omcgo/` — PASS
- `bash omcgo/scripts/check-migrations.sh` — FAIL, existing migration numbering/version warnings as described above
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh down -v && OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build` — PASS
- `curl -I --max-time 10 http://localhost:8081/` — PASS, `HTTP/1.1 200 OK`

## Conclusion

No blocking issues found in the staged change. The remaining migration-check warning is existing repository state, not introduced by this task.
