# Code Review: KPI Indicator Management (Backend)

- **Date**: 2026-04-26
- **Scope**: `omcgo/internal/pm/indicator/`, `omcgo/cmd/app/provider/`, `omcgo/migrations/`
- **Author**: Claude AI (automated review)
- **Related Task**: T-0027

## Summary

Review of the KPI indicator management module — a new feature implementing indicator CRUD, group tree management, formula validation, batch enable/disable, and export for ENB/GSM/GNB device types.

## Files Reviewed (17 files, 17,359 lines)

| File | Lines | Notes |
|------|-------|-------|
| `handler.go` | 829 | HTTP handlers for all indicator operations |
| `service.go` | 502 | Business logic layer |
| `model.go` | 262 | Data models and request/response types |
| `repository.go` | 101 | Interface definitions |
| `pg_indicator_repository.go` | 580 | Indicator persistence |
| `pg_group_repository.go` | 214 | Group tree persistence |
| `pg_cust_name_repository.go` | 100 | Custom name persistence |
| `pg_enabled_repository.go` | 122 | Enabled indicator persistence |
| `pg_platform_formula_repository.go` | 154 | Formula relationship persistence |
| `pg_indicator_threshold_repository.go` | 87 | Threshold persistence |
| `pg_template_rel_repository.go` | 43 | Template association checks |
| `formula_validator.go` | 181 | Formula syntax + reference validation |
| `formula_validator_test.go` | 138 | 10 test cases for formula validation |
| `pm.go` (provider) | 84 | DI wiring |
| `router.go` (provider) | 437 | Route registration |
| `000025_indicator_management.sql` | 373 | DDL migration |
| `000026_seed_indicator_management.sql` | 13,636 | Seed data |

## Issues Found & Fixed

### CRITICAL (4 fixed)

| # | Issue | Fix |
|---|-------|-----|
| C1 | SQL injection via `sortBy`/`sortDir` in ORDER BY | Whitelist `allowedSortColumns` map + strict `sanitizeSortDir` |
| C2 | UUID truncation in `generateGroupID()` (8 chars) | Use full UUID string |
| C3 | ID generation race condition (MAX+1 without lock) | `pg_advisory_xact_lock` in transaction |
| C4 | `operatorCode` hardcoded empty string | Accept from request, fallback to "default" |

### WARNING (6 fixed)

| # | Issue | Fix |
|---|-------|-----|
| W1 | 8 locations missing `rows.Err()` check | Added to all `rows.Next()` loops |
| W2 | `refreshRedisCache` swallows Redis errors | Log via `s.logger.Warn` |
| W3 | API could create pseudo-built-in groups | `IsBuildIn: "0"` hardcoded in CreateGroup |
| W4 | `UpdateCounterName` bypasses built-in protection | Added `IsBuildIn == "1"` check |
| W5 | Empty update generates invalid SQL | Check `strings.Contains(query, "SET ")` |
| W6 | `isNumber` allows `1.2.3` | Strict validation with single-dot tracking |

### INFO (not fixed, tracked separately)

| # | Note |
|---|------|
| I1 | W7: `buildIDMap` full-table scan on every formula validation — performance optimization deferred |
| I2 | GNB handlers duplicate ENB handlers (~250 lines) — refactor when bandwidth allows |
| I3 | `indicatorColumnsWithoutAlias()` unused — minor cleanup |
| I4 | String-typed booleans (`"0"`/`"1"`) — database schema legacy |

## Verification

- `go build ./...` — PASS
- `go test ./internal/pm/indicator/...` — PASS (10/10 tests)
- Browser testing — PASS (10/10 features verified)

## Verdict: **PASS**
