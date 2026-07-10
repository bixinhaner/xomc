# Task 12 report — 旧入口收口与安全清理工具

## Red/green evidence

- Red: `cd omcgo && /usr/local/go/bin/go test ./internal/task -run TestRedisTaskQueue_PurgeBySource -count=1 -v` failed because `RedisTaskQueue.PurgeBySource` was undefined.
- Green: the same focused queue test passes after implementing source-filtered SCAN/pipeline purge; dry-run and CWMP mapping deletion tests also pass.
- `cd omcgo && /usr/local/go/bin/go test ./internal/mml -run 'TestHandler_.*Script' -count=1` passes, including the removed `POST /mml/scripts` route returning 404.
- `cd omcgo && /usr/local/go/bin/go test ./cmd/omcctl -run 'TestMMLResetScripts' -count=1` passes, including confirmation guard and dry-run non-mutation.
- `cd omcmb && npm run typecheck` passes skin parity plus webcode, webcode-v2 and webcode-v3 typechecks.

## Safety evidence

- `PurgeBySource` traverses only task queue keys with Redis `SCAN`, reads details via pipeline, filters by serialized `Task.Source`, and removes only matching queue membership, detail hash, and optional CWMP mapping.
- `dryRun=true` increments `Matched` but performs no mutation.
- The implementation contains no `KEYS` or `FLUSHDB` calls.
- `omcctl mml reset-script-data` defaults to dry-run. Apply mode requires the exact `--confirm DELETE-MML-RUNTIME` token and reports `matched/deleted/skipped/errors`; any counted errors return a non-zero command error.

## Scope notes

- Removed the legacy browser `POST /mml/scripts` registration and converted its handler test to assert 404.
- Removed the old ScriptTaskDrawer file-upload/direct-task entry while retaining Console prefill parsing for immediate execution; script-library import/execution remains in the dedicated flow.
