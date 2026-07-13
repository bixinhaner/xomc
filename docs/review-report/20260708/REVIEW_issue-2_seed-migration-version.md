# Review: Issue #2 seed migration version

## Scope

- Issue: #2 修复 seed migration 重复版本号导致部署失败
- Branch: fix/2-seed-migration-version
- Diff: rename `omcgo/migrations/seed/000007_add_direct_standard_params_to_mml_tree.sql` to `omcgo/migrations/seed/000009_add_direct_standard_params_to_mml_tree.sql`

## Standards

- Hard findings: none.
- The staged change preserves SQL contents and moves the later seed migration to the next available version.
- Seed migration filenames are unique and sequential from `000001` through `000009`.
- Operational note: renaming an already-recorded goose seed migration can be risky in environments that applied the old version, but main currently has duplicate `000007` files, so the rename is required to prevent goose startup panic.

## Spec

- Findings: none.
- Matches Issue #2 requirement to keep `000007_indicator_process_number.sql` and assign the MML standard parameter binding seed migration a unique sequential version.
- No scope creep: SQL contents and schema are unchanged.

## Verification

- `go build ./...`: pass.
- `go test ./...`: pass.
- Seed duplicate-version check: pass.
- `000009_add_direct_standard_params_to_mml_tree.sql` contains `-- +goose Up` and `-- +goose Down`.
- Seed migration down/up rehearsal against local Docker Postgres: pass, version 9 -> 8 -> 9.
- Docker deploy via `/Users/shangyingbin/project/omc-docker/docker-run.sh`: pass.
- `docker compose ps -a`: migration containers exited 0; app/acs/worker/web running.
- `curl -fsS http://localhost:9091/healthz`: pass.
- `curl -fsS http://localhost:7557/healthz`: pass.
- Browser smoke at `http://localhost:8081/login`: pass, title `OMC NMS`, no page errors.

## Notes

- `golangci-lint run` was not executed because `golangci-lint` is not installed on this host.
- `bash scripts/check-migrations.sh --strict` currently exits early on an existing main schema migration with an empty Down body under `set -euo pipefail`; this is pre-existing and unrelated to the seed rename.
