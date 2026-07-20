# System Config Functional Test Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:systematic-debugging` for every failed or unexpected result. This plan is executed inline in the current session; no subagent delegation is authorized.

**Goal:** Verify every user-facing System Config function and its storage, PM, logging, and runtime consumers without destructively changing the current environment.

**Architecture:** Tests are vertical slices through UI → API → `sys_configs` → runtime consumer. Existing automated tests verify behavior in isolation; browser and read-only runtime checks verify the deployed integration. Save behavior is tested through existing isolated tests or an unchanged-value request only when restoration and applied-state verification are available.

**Tech Stack:** React, Ant Design, Vitest, Playwright, Go test, Gin, PostgreSQL, TimescaleDB, MinIO, Docker Compose.

## Global Constraints

- Test the current `main` / `origin/main` baseline.
- Do not alter user data, credentials, retention periods, Agent connection state, or cleanup schedules.
- Do not rebuild or restart containers as part of a read-only test.
- Record PASS, PARTIAL, FAIL, or NOT TESTABLE for every feature.
- A rendered page is not sufficient evidence that its backend consumer works.
- Any failed test must be investigated to its failing layer before recommendations are written.

---

### Task 1: Establish Baseline and Test Inventory

**Files:**
- Read: `omcmb/webcode/src/pages/system/SystemConfig/`
- Read: `omcgo/internal/admin/`
- Read: `omcgo/internal/agentconfig/`
- Read: storage, retention, logger, device and transfer policy packages

- [ ] Confirm Git HEAD and dirty files.
- [ ] List all nine tabs, categories, keys, APIs, runtime consumers and existing tests.
- [ ] Record deployed container image age separately from source HEAD.

### Task 2: Run Automated Regression Tests

**Interfaces:**
- Consumes: public Go package APIs and frontend user behavior tests
- Produces: fresh pass/fail evidence and coverage gaps

- [ ] Run backend:

```bash
cd omcgo
go test ./...
```

- [ ] Run frontend type and lint checks:

```bash
cd omcmb
npm run typecheck
npm run lint
```

- [ ] Run all frontend unit/component tests:

```bash
cd omcmb
npm run test --workspace webcode
```

- [ ] Run relevant mock browser behavior:

```bash
cd omcmb/webcode
npx playwright test e2e/security-policy.spec.ts
```

- [ ] Run real-backend system smoke test against port 18081:

```bash
cd omcmb/webcode
SMOKE_API_TARGET=http://localhost:18081 npx playwright test --config playwright.smoke.config.ts e2e/smoke/system.spec.ts
```

### Task 3: Browser-Test Nine System Config Tabs

**Interfaces:**
- Consumes: `/system/config`, authenticated category queries
- Produces: actual DOM evidence for every tab

For each tab, verify:

- [ ] tab is reachable and selected;
- [ ] expected controls render with loaded values;
- [ ] no visible load error or console error;
- [ ] save controls are present and their enabled state matches load state;
- [ ] fields exposed by the UI match the backend-supported key set.

Tabs:

- [ ] Basic
- [ ] Security
- [ ] Device
- [ ] Storage
- [ ] ACS Transfer
- [ ] Agent
- [ ] PM Retention
- [ ] Retention / Backpressure
- [ ] Log Configuration

### Task 4: Test API and Security Boundaries

**Interfaces:**
- Consumes: public config endpoint, authenticated `sysConfig`, Agent admin DTO
- Produces: authorization and secret-handling results

- [ ] Anonymous request to protected config API returns 401.
- [ ] Public config API returns only intended keys.
- [ ] Authenticated admin can list every expected category.
- [ ] Determine whether a non-admin role can call config APIs using existing tests or an existing account, without creating users.
- [ ] Check whether generic responses contain keys classified as secrets.
- [ ] Verify unknown category, invalid category/key values and empty values are handled consistently by automated tests.

### Task 5: Test Runtime Consumers

**Interfaces:**
- Consumes: stored configuration and applied runtime state
- Produces: desired-vs-applied comparison

- [ ] Basic timezone matches worker cron interpretation.
- [ ] Security effective defaults match persisted and UI values.
- [ ] Device thresholds and enable switches match scanner behavior.
- [ ] ACS transfer snapshot matches stored values.
- [ ] Agent admin response masks token and runtime denylist remains present.
- [ ] PM retention values match TimescaleDB jobs.
- [ ] MinIO retention values match bucket lifecycle rules.
- [ ] Backpressure stored keys match ACS-supported keys.
- [ ] DB log retention jobs have recent successful executions.
- [ ] File log rotation settings match generated file sizes/archives as far as observable.

### Task 6: Test Capacity and Failure Visibility

- [ ] Record database table sizes and retention backlog indicators.
- [ ] Record MinIO object counts and sizes.
- [ ] Record application log directory size and archive pattern.
- [ ] Record host disk use, Docker images, volumes and build-cache use.
- [ ] Verify the system dashboard disk value against the host value.
- [ ] Verify failed configuration loads cannot be saved as default zero/false from source and existing tests.

### Task 7: Produce Test Report

**Files:**
- Create: `docs/review-report/20260720/TEST_cccbb2b3_system-config-functional.md`

- [ ] Record every feature as PASS, PARTIAL, FAIL, or NOT TESTABLE.
- [ ] Separate product defects from missing tests and environment drift.
- [ ] List critical findings with reproducible evidence.
- [ ] Rank immediate P0, near-term P1 and later P2 fixes.
- [ ] Include exact commands and fresh final verification results.
