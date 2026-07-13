# #37 KPI Name Localization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make KPI names follow the active Chinese/English locale in templates, web chart legends/tooltips, and asynchronous CSV exports.

**Architecture:** Keep stable KPI IDs and API fields, but select `display_name` from the request locale. Persist the export request locale inside the existing JSON params so the Worker can reproduce the request language without a schema migration. Include locale in shared React Query keys so all three skins refetch localized catalogs after a language change.

**Tech Stack:** Go 1.25, Gin, pgx, React 19, TypeScript, TanStack Query, Zustand, Docker Compose.

## Global Constraints

- English selects `en_name`, Chinese selects `cn_name`; each falls back to the other language and then the indicator code.
- Do not change KPI IDs, formulas, units, or stored template metric codes.
- Do not add a database migration; store export locale in the existing `params` JSON.
- Frontend behavior must live in `frontend-core` so all three skins share it.

---

### Task 1: Localize the KPI definition catalog

**Files:**
- Modify: `omcgo/internal/pm/handler.go`
- Test: `omcgo/internal/pm/handler_test.go`

**Interfaces:**
- Consumes: `appcontext.GetLocale(context.Context)` and `appcontext.LocaleEN`.
- Produces: locale-aware `kpiDefinitionItem.DisplayName` while preserving `Name = EnName` and `ID`.

- [ ] **Step 1: Write the failing English-locale handler test**

Add a test that injects `appcontext.LocaleEN` into the request context and asserts `display_name == "RRC Setup Success Rate"` when both English and Chinese names exist.

- [ ] **Step 2: Run the test and verify RED**

Run: `cd omcgo && go test ./internal/pm -run TestHandler_ListKPIDefinitions_EnglishLocale -count=1`

Expected: FAIL because the current handler returns the Chinese name.

- [ ] **Step 3: Implement locale-aware fallback**

Compute the request locale once and choose the display name with a focused helper:

```go
func localizedIndicatorName(loc appcontext.Locale, en string, cn *string, fallback string) string {
    cnName := derefOr(cn, "")
    if loc == appcontext.LocaleEN {
        if en != "" {
            return en
        }
        if cnName != "" {
            return cnName
        }
        return fallback
    }
    if cnName != "" {
        return cnName
    }
    if en != "" {
        return en
    }
    return fallback
}
```

- [ ] **Step 4: Run focused and package tests**

Run: `cd omcgo && go test ./internal/pm -count=1`

Expected: PASS.

---

### Task 2: Preserve locale across asynchronous export

**Files:**
- Create: `omcgo/internal/pm/export/locale.go`
- Test: `omcgo/internal/pm/export/locale_test.go`
- Modify: `omcgo/internal/pm/export/handler.go`
- Modify: `omcgo/internal/pm/export/runner.go`
- Modify: `omcgo/internal/pm/export/csv.go`
- Modify: `omcgo/internal/pm/export/generator.go`
- Modify: `omcgo/internal/pm/export/adhoc_label.go`
- Test: `omcgo/internal/pm/export/runner_test.go`
- Test: `omcgo/internal/pm/export/generator_test.go`
- Test: `omcgo/internal/pm/export/adhoc_label_test.go`

**Interfaces:**
- Produces: `withExportLocale(raw []byte, loc appcontext.Locale) ([]byte, error)`.
- Produces: `exportLocale(raw []byte) appcontext.Locale` with legacy/default fallback to `LocaleZH`.
- Consumes: task `params` JSON and passes the resolved locale into `newNameResolver`.

- [ ] **Step 1: Write failing locale JSON tests**

Cover these cases:

```go
withExportLocale([]byte(`{"task_id":"..."}`), appcontext.LocaleEN)
// preserves task_id and adds "locale":"en-US"

exportLocale([]byte(`{"locale":"en-US"}`)) == appcontext.LocaleEN
exportLocale([]byte(`{}`)) == appcontext.LocaleZH
exportLocale([]byte(`{"locale":"unknown"}`)) == appcontext.LocaleZH
```

- [ ] **Step 2: Run tests and verify RED**

Run: `cd omcgo && go test ./internal/pm/export -run 'TestWithExportLocale|TestExportLocale' -count=1`

Expected: build failure because the helpers do not exist.

- [ ] **Step 3: Implement the JSON locale helpers**

Use `map[string]json.RawMessage` to preserve all existing params, normalize to `en-US` or `zh-CN`, and return an error for malformed/non-object JSON. Missing or invalid stored locale must read as Chinese for backward compatibility.

- [ ] **Step 4: Capture locale in the create handler**

Before calling `Service.Create`, call:

```go
params, err = withExportLocale(params, appcontext.GetLocale(c.Request.Context()))
```

Return HTTP 400 when params cannot be merged as a JSON object.

- [ ] **Step 5: Make Runner use the stored locale**

Replace Worker-context locale lookup in `buildSource` with:

```go
loc := exportLocale(task.Params)
```

This ensures both dashboard and adhoc exports resolve column names using the locale captured at creation.

- [ ] **Step 6: Add a Runner regression test**

Create an adhoc task with `"locale":"en-US"`, capture the name-resolution SQL, and assert it contains `COALESCE(NULLIF(en_name, ''), cn_name)` even though `buildSource` runs with `context.Background()`.

- [ ] **Step 7: Run export tests**

Before the package run, add failing tests that require `Start Time`, `End Time`, `Technology`, `Measurement Object`, and localized dimension headers for English layouts. Pass stored locale through `csvLayout`; retain Chinese defaults for legacy tasks and direct writer callers.

Run: `cd omcgo && go test ./internal/pm/export -count=1`

Expected: PASS.

---

### Task 3: Isolate frontend KPI caches by locale

**Files:**
- Modify: `omcmb/frontend-core/src/hooks/api/usePerformance.ts`
- Test: `omcmb/webcode/src/pages/performance/usePerformanceQueryKeys.test.ts`

**Interfaces:**
- Consumes: `useAppStore((s) => s.locale)`.
- Produces: locale-bearing query keys for KPI list, full KPI catalog, and indicator candidates.

- [ ] **Step 1: Write the failing query-key test**

Export focused pure key builders from `usePerformance.ts` and assert that `zh-CN` and `en-US` produce different keys for KPI list, full catalog, and indicator candidates.

- [ ] **Step 2: Run the test and verify RED**

Run: `cd omcmb && npm test --workspace webcode -- usePerformanceQueryKeys.test.ts`

Expected: FAIL because locale is absent from the current key builders/hooks.

- [ ] **Step 3: Add locale to shared hooks**

For `useKPIList`, `useAllKPIs`, and `useIndicatorCandidates`, read the locale and append it to each query key:

```ts
const locale = useAppStore((s) => s.locale);
queryKey: ['performance', 'kpis', params, locale]
```

Use the equivalent key shape for the other two hooks. The request interceptor already sends the same locale through `Accept-Language`.

- [ ] **Step 4: Re-run the focused frontend test**

Run: `cd omcmb && npm test --workspace webcode -- usePerformanceQueryKeys.test.ts`

Expected: PASS.

- [ ] **Step 5: Verify all three skins compile and stay in parity**

Run: `cd omcmb && npm run skin-parity`

Expected: PASS.

Run: `cd omcmb && npm run typecheck`

Expected: PASS for v1, v2, and v3.

---

### Task 4: Full verification and Docker acceptance

**Files:**
- No production files beyond Tasks 1-3.

**Interfaces:**
- Consumes: the complete implementation.
- Produces: build/test evidence and a real English CSV/API result.

- [ ] **Step 1: Run backend verification**

Run: `cd omcgo && go build ./...`

Run: `cd omcgo && go test ./...`

Expected: all packages PASS.

- [ ] **Step 2: Rebuild and start Docker services**

Run: `docker compose -p omc -f deployments/docker/docker-compose.yml up -d --build app worker web`

Expected: app/web are running and worker is healthy.

- [ ] **Step 3: Verify the live English KPI catalog**

Call `/api/v1/pm/kpi/definitions?device_type=GSM` with `Accept-Language: en-US` and assert `KGSM0143.display_name == "Cell Availability Rate"`.

- [ ] **Step 4: Verify the live English asynchronous export**

Create an isolated adhoc fixture using `KGSM0143`, create an export through the API with `Accept-Language: en-US`, wait for success, download CSV, and assert the complete header is `Start Time,End Time,Product,Cell Availability Rate` with no Chinese field names.

- [ ] **Step 5: Clean acceptance fixtures**

Delete the temporary main-DB task/export/job rows, TSDB result row, MinIO object, and Worker temporary CSV. Confirm all fixture counts are zero.

---

### Task 5: GitLab integration

**Files:**
- Commit all code, tests, design, and plan files for #37.

**Interfaces:**
- Produces: merged GitLab MR closing #37.

- [ ] **Step 1: Review the final diff and status**

Run: `git diff --check`

Expected: no output.

- [ ] **Step 2: Commit the implementation**

Commit message: `fix(pm): 修复 KPI 名称英文环境本地化`

- [ ] **Step 3: Push and create GitLab MR**

Push `fix/issue-37-kpi-locale`, create an MR targeting `main`, include verification evidence, and add `Closes #37`.

- [ ] **Step 4: Merge and synchronize main**

Merge only the reviewed SHA, remove the source branch, switch to `main`, and fast-forward from `origin/main`.
