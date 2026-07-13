# Review: task GPV summary duration

## Verdict

PASS

## Scope

- `omcgo/internal/task/pg_repository.go`
- `omcgo/internal/task/pg_repository_test.go`

## Summary

This change fixes `LatestSyncGPVSummaryByDevice` so the "latest sync GPV" duration is calculated from completed GPV tasks only. It prevents stale unfinished `sync-gpv-*` rows with the same `source_id` from pulling `MIN(created_at)` back by hours or days and inflating `wall_clock_seconds`.

## Findings

No CRITICAL findings.

## Checks

- SQL remains parameterized through `$1`; no string interpolation was introduced.
- The latest summary query now selects only rows with `completed_at IS NOT NULL`, so the reported `first_created_at`, `last_completed_at`, task count, and wall clock duration all describe a completed sync batch.
- Regression coverage inserts an old unfinished `sent` task and newer completed tasks under the same `source_id`, then verifies the summary ignores the unfinished task and reports the completed batch duration.

## Validation

- `go test ./internal/task -count=1` — passed
- `go build ./...` — passed

## Risk

Low. The visible behavior changes only the GPV sync summary used by status display. In-progress sync visibility still comes from queue length/pending status, not from this completed-summary query.
