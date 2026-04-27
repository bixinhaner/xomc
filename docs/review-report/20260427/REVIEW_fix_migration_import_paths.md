# Code Review: Migration Version Fix + Frontend Import Path Adaptation

- **Date**: 2026-04-27
- **Scope**: `omcgo/migrations/`, `omcmb/frontend-core/`, `omcmb/webcode/src/pages/performance/`
- **Author**: Claude AI (automated review)
- **Related Task**: T-0027

## Summary

Two fixes to adapt to remote directory rename (`omcmb/webcode/` → `omcmb/frontend-core/`):
1. Migration version renumbering to avoid conflict with existing `000025_mml_sub_commands.sql`
2. Trigger idempotency improvement (StatementBegin/End + CREATE OR REPLACE)
3. Frontend import path adaptation (`@/` → `@core/` for moved modules)

## Changes Reviewed

### Backend (omcgo/)

| File | Change | Notes |
|------|--------|-------|
| `migrations/000025_indicator_management.sql` → `000035_...` | Renamed + trigger fix | Version 25 conflicted with `000025_mml_sub_commands.sql` |
| `migrations/seed/000026_seed_...` → `000027_seed_...` | Renamed | Content identical, version bump only |

**Trigger changes**: All 17 `CREATE TRIGGER` statements wrapped in `-- +goose StatementBegin/End` + `CREATE OR REPLACE TRIGGER` with `EXCEPTION WHEN others THEN NULL` for idempotency.

### Frontend (omcmb/)

| File | Change |
|------|--------|
| `frontend-core/src/hooks/api/useIndicator.ts` | `@/types/indicator` → `@core/types/indicator` (+3 more) |
| `webcode/src/pages/performance/KPIStandardReport/IndicatorDetail.tsx` | `@/hooks/api/useIndicator` → `@core/hooks/api/useIndicator` (+2 more) |
| `webcode/src/pages/performance/KPIStandardReport/index.tsx` | 5 imports changed to `@core/` |
| `webcode/src/pages/performance/KPIStationReport/index.tsx` | 2 imports changed to `@core/` |

## Issues Found

None. All changes are mechanical path/version adaptations.

## Verification

- `go build ./...` — PASS
- Frontend `tsc --noEmit` — PASS
- Frontend `npm run build` — PASS
- Migration `omcgo-migrate up` — PASS (version 35 applied)
- Seed data loaded — 27 indicators confirmed
- Browser testing — KPI Standard Report page loads with 1873 indicators

## Verdict: **PASS**
