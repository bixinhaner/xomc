# Dashboard GSM Network Type Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make GSM a canonical, visible, and correctly filtered Dashboard KPI technology for issues #50 and #58.

**Architecture:** Seed data establishes the canonical `network_type` values and upgrades legacy data. The dictionary service enforces the machine-value contract on writes. The Dashboard forwards the selected technology into the KPI time-series API and PM Aggregator query so its values use the same scope as PM.

**Tech Stack:** Go, pgx/PostgreSQL goose seed migrations, React/TypeScript, Vitest.

## Global Constraints

- Canonical machine values are exactly `lte`, `nr`, and `gsm`; `GSM` remains display text only.
- Preserve existing generic dictionary behavior for types other than `network_type`.
- Use a forward seed migration; do not modify an already applied seed baseline.
- Keep the existing Dashboard dictionary-driven visibility and GSM KPI layouts.
- Do not use issue-closing keywords in commits or MR descriptions before test-environment acceptance.

---

### Task 1: Normalize `network_type` seed data

**Files:**
- Create: `omcgo/migrations/seed/000002_normalize_network_type_gsm.sql`
- Test: local PostgreSQL seed migration verification query

**Consumes:** `sys_dictionaries.type`, `sys_dictionary_details` active-value uniqueness index.

**Produces:** One active `gsm` item under `network_type`, preserving the display label `GSM`.

- [ ] **Step 1: Write the migration before changing data behavior**

```sql
-- Resolve the dictionary by type; do not depend on baseline numeric IDs.
-- If both values exist, soft-delete legacy `GSM` first, then normalize it only
-- when no canonical value exists. Insert canonical `gsm` when neither exists.
```

- [ ] **Step 2: Run the migration against a database containing only `GSM`**

Run: `GOOSE_TABLE=goose_db_version_seed go run ./cmd/migrate up --path migrations/seed`

Expected: the active detail queried with `WHERE type='network_type'` has `value='gsm'`, not `GSM`.

- [ ] **Step 3: Complete the idempotent SQL branches**

```sql
UPDATE sys_dictionary_details ... SET value = 'gsm'
WHERE value = 'GSM' AND NOT EXISTS (... value = 'gsm' ...);
UPDATE sys_dictionary_details ... SET deleted_at = NOW()
WHERE value = 'GSM' AND EXISTS (... value = 'gsm' ...);
INSERT INTO sys_dictionary_details (..., label, value, status, sort, ...)
SELECT ..., 'GSM', 'gsm', true, 3, ...
WHERE NOT EXISTS (... value = 'gsm' ...);
```

- [ ] **Step 4: Verify all three historical states**

Run a query for: missing GSM, legacy-only `GSM`, and both values.

Expected: each result has one active `gsm` detail and no active `GSM` detail.

### Task 2: Enforce the dictionary machine-value contract

**Files:**
- Modify: `omcgo/internal/admin/dictionary_service.go:198-317`
- Modify: `omcgo/internal/admin/dictionary_service_test.go`

**Consumes:** dictionary lookup by `SysDictionaryID`, `CreateDictionaryDetailRequest`, `UpdateDictionaryDetailRequest`.

**Produces:** `validateNetworkTypeDictionaryValue(dictionaryType, value)` invoked on create and update.

- [ ] **Step 1: Add failing service tests**

```go
func TestDictionaryServiceCreateNetworkTypeRejectsUppercaseGSM(t *testing.T) {
    // dictionary type network_type, request value GSM
    // expect a business validation error
}

func TestDictionaryServiceUpdateNetworkTypeAcceptsLowercaseGSM(t *testing.T) {
    // dictionary type network_type, request value gsm
    // expect repository update with gsm
}

func TestDictionaryServiceAllowsOtherDictionaryValues(t *testing.T) {
    // a non-network_type dictionary may retain arbitrary existing values
}
```

- [ ] **Step 2: Run only the new tests and confirm RED**

Run: `cd omcgo && go test ./internal/admin -run 'TestDictionaryService(CreateNetworkTypeRejectsUppercaseGSM|UpdateNetworkTypeAcceptsLowercaseGSM|AllowsOtherDictionaryValues)' -count=1`

Expected: the uppercase case currently succeeds, so the test fails for the intended missing validation.

- [ ] **Step 3: Add minimal validation**

```go
func validateNetworkTypeDictionaryValue(dictType, value string) error {
    if dictType != "network_type" { return nil }
    switch value {
    case "lte", "nr", "gsm": return nil
    default: return commonerrors.NewBusinessError(7014,
        "network_type value must be one of: lte, nr, gsm", nil)
    }
}
```

For updates, resolve the effective dictionary after applying any requested dictionary-ID change, then validate the effective value before repository persistence.

- [ ] **Step 4: Run the focused tests and confirm GREEN**

Run: same command as Step 2.

Expected: PASS.

### Task 3: Propagate Dashboard technology to the PM query

**Files:**
- Modify: `omcmb/frontend-core/src/services/api/dashboardApi.ts`
- Modify: `omcmb/frontend-core/src/hooks/api/useDashboard.ts`
- Modify: `omcmb/webcode/src/pages/dashboard/DashboardKPIModules.tsx`
- Modify: `omcgo/internal/dashboard/handler.go`
- Modify: `omcgo/internal/dashboard/service.go`
- Modify: `omcgo/internal/dashboard/kpi_network_query.go`
- Test: existing Dashboard API/hook tests and `omcgo/internal/dashboard/*_test.go`

**Consumes:** `TechnologyType`, Dashboard KPI request query parameters, `pmaggregator.QueryRequest.Technologies`.

**Produces:** request parameter `technology=gsm` and aggregator request `Technologies: []model.Technology{model.TechGSM}`.

- [ ] **Step 1: Add failing frontend and backend tests**

```ts
expect(http.get).toHaveBeenCalledWith('/dashboard/kpi-time-series', {
  params: expect.objectContaining({ technology: 'gsm' }),
});
```

```go
func TestBuildNetworkKPIDailySeriesRequestFiltersTechnology(t *testing.T) {
    request := buildNetworkKPIDailySeriesRequest([]string{"KGSM0101"}, []model.Technology{model.TechGSM}, start, end)
    require.Equal(t, []model.Technology{model.TechGSM}, request.Technologies)
}
```

- [ ] **Step 2: Run focused tests and confirm RED**

Run frontend: `cd omcmb && npx vitest run <dashboard-api-test-file>`.

Run backend: `cd omcgo && go test ./internal/dashboard -run TestBuildNetworkKPIDailySeriesRequestFiltersTechnology -count=1`.

Expected: current request types and builders do not accept technology, so both tests fail.

- [ ] **Step 3: Implement the narrow parameter flow**

```text
DashboardKPIModules(technology)
  -> Dashboard data hook / dashboardApi query params
  -> handler validates lte|nr|gsm
  -> service request
  -> buildNetworkKPISeriesRequest(..., technologies, ...)
  -> pmaggregator.QueryRequest.Technologies
```

Use the canonical technology conversion already defined in the backend model/constants; reject unsupported API values with a 4xx validation response.

- [ ] **Step 4: Run focused tests and confirm GREEN**

Run the commands from Step 2.

Expected: PASS; LTE and NR call sites compile after their technology argument is supplied.

### Task 4: Lock homepage GSM visibility and perform end-to-end regression

**Files:**
- Modify: `omcmb/webcode/src/components/dashboard/__tests__/useTechnologyDictionary.test.tsx`
- Modify: relevant Dashboard tests under `omcmb/webcode/src/pages/dashboard/__tests__/`

**Consumes:** canonical dictionary details and Dashboard KPI hook API.

**Produces:** regression coverage proving that a `gsm` detail renders a GSM option and disabled/uppercase values do not silently become valid.

- [ ] **Step 1: Add a failing visibility test**

```tsx
it('returns the enabled lowercase gsm dictionary detail as a GSM option', () => {
  // mock network_type details: lte, nr, gsm(label GSM)
  expect(result.current.options.map(({ value }) => value)).toEqual(['lte', 'nr', 'gsm'])
})
```

- [ ] **Step 2: Run the hook test and confirm its baseline behavior**

Run: `cd omcmb && npx vitest run webcode/src/components/dashboard/__tests__/useTechnologyDictionary.test.tsx --environment node`.

Expected: PASS once seed/API data is canonical; this test locks the user-visible #50 result without weakening strict input semantics.

- [ ] **Step 3: Run complete scoped validation**

Run:

```bash
cd omcgo && go test ./internal/admin ./internal/dashboard && go build ./...
cd omcmb && npm run typecheck
cd omcmb && npx vitest run webcode/src/components/dashboard frontend-core/src --environment node
```

Expected: all targeted tests, backend build, and frontend typecheck pass.

- [ ] **Step 4: Perform database and browser acceptance**

Verify the active `network_type` values are `lte,nr,gsm`, open the local Dashboard, select GSM, and confirm both hourly and daily network requests contain `technology=gsm` and return GSM KPI series.

- [ ] **Step 5: Commit the implementation**

```bash
git add omcgo/migrations/seed/000002_normalize_network_type_gsm.sql \
  omcgo/internal/admin omcgo/internal/dashboard \
  omcmb/frontend-core omcmb/webcode \
  docs/design/dashboard-gsm-network-type-issues-50-58-20260713.md \
  docs/superpowers/plans/2026-07-13-dashboard-gsm-network-type.md
git commit -m "fix(dashboard): 支持 GSM KPI 制式"
```

## Self-review

- Seed migration, validation, request propagation and user-visible GSM coverage all map to a design acceptance item.
- The migration explicitly treats missing, uppercase-only and duplicate case states.
- Technology remains lower-case in each interface; only labels display `GSM`.
- The plan deliberately retains strict frontend filtering, so invalid data remains observable instead of being masked.
