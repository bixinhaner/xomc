# /e2e -- End-to-End Verification Command

Run the full E2E test suite against a running omcgo-app instance.

## Test Coverage

Total: ~301 test cases across Sprint 0-9 milestones (S1-S56 sections).

### Sprint 0 (S1-S4): Infrastructure & Health
- S1: Health check endpoints (`/healthz`, `/readyz`)
- S2: Auth login/token flow
- S3: Device CRUD basics
- S4: Alarm list & filter

### Sprint 1 (S5-S8): Device & Alarm Core
- S5: Device list with pagination and filtering
- S6: Device detail and update
- S7: Alarm acknowledge and lifecycle
- S8: Alarm severity filtering

### Sprint 2 (S9-S14): Config & Firmware
- S9: Config template CRUD
- S10: Firmware version management
- S11: Upgrade task creation and status
- S12: Device group management
- S13: User CRUD
- S14: RBAC role assignment

### Sprint 3 (S15-S18): PM / KPI / MR / Audit
- S15: PM counter query
- S16: KPI definition and calculation
- S17: MR file and record query
- S18: Audit log query

### Sprint 4 (S19-S24): Dashboard / Rules / Thresholds / Logs
- S19-S20: Extended alarm and PM queries
- S21: Dashboard summary
- S22: Alarm rules CRUD
- S23: KPI thresholds CRUD
- S24: System & NE message logs

### Sprint 5 (S25-S30): DataModel / Admin / Advanced Queries
- S25-S30: Data model CRUD, admin user management, advanced filtering

### Sprint 6 (S31-S38): Extended Admin / Group / Device / Config Sync
- S31: Admin role CRUD extended
- S32: Admin user management extended
- S33: Role assignment
- S34: PM threshold CRUD extended
- S35: Data model extended operations
- S36: Group CRUD extended
- S37: Device extended operations (stats, parameters, reboot)
- S38: Config sync (push/pull/status)

### Sprint 7 (S39-S46): Dashboard Trends / Config Sync / DataModel / Regression
- S39: Dashboard alarm trend (3 cases) -- alarm-trend endpoint with day range, date field validation
- S40: Dashboard device status (2 cases) -- device-status endpoint with status counts
- S41: Dashboard KPI trend (3 cases) -- kpi-trend endpoint with time/value fields, required param validation
- S42: Dashboard region stats (2 cases) -- region-stats endpoint with array response
- S43: Config sync integration (3 cases) -- push/pull/status with Sprint 7 specific payloads
- S44: Device parameters (2 cases) -- device parameter retrieval and items validation
- S45: DataModel resolve (2 cases) -- datamodel list with items check
- S46: Sprint 7 regression (3 cases) -- dashboard/summary, healthz, CORS headers

### Sprint 8 (S47-S52): Backup / File Manager / MML Console
- S47: Backup task CRUD (6 cases) -- list, create, get, cancel, verify cancelled, delete
- S48: Backup schedule CRUD (4 cases) -- list, create, update, delete
- S49: File management (6 cases) -- list, multipart upload, get detail, download with Content-Disposition, type filter, delete
- S50: MML commands (4 cases) -- list commands (3 seed), get command detail, execute command, get task status
- S51: MML scripts (3 cases) -- create script, list scripts, delete script
- S52: MML task history (2 cases) -- list tasks, verify task from S50 in list

### Sprint 9 (S53-S56): Frontend Integration & Full Regression
- S53: Backup frontend integration (3 cases) -- paginated task list with total, create+delete cleanup, paginated schedule list
- S54: File management frontend integration (3 cases) -- paginated file list with total, file_type filter, file detail by id
- S55: MML frontend integration (3 cases) -- paginated command list, execute command with task_id, task detail with status
- S56: Full regression smoke (6 cases) -- healthz, dashboard summary, devices list, alarms list, alarm-trend, CORS headers

Sprint 9 provides comprehensive frontend integration validation ensuring all Sprint 6-9 API endpoints respond correctly when called with the pagination and filtering patterns used by the frontend application. The full regression section (S56) acts as a final smoke test confirming Sprint 0-9 core endpoints remain functional end-to-end.

## Prerequisites

1. omcgo-app running on `localhost:8080`
2. Database migrated: `make migrate-up`
3. Seed data loaded: `psql "$DSN" -f scripts/seed_e2e_testdata.sql`

## Usage

```bash
# Default (localhost:8080)
./scripts/e2e_verify.sh

# Custom base URL
./scripts/e2e_verify.sh http://my-server:8080
```

## Seed Data

Test data is managed in `scripts/seed_e2e_testdata.sql` and includes:
- 5 test devices (various carriers/technologies/statuses)
- 8+ active alarms (4 severity levels) + 3 Sprint 7 alarm trend records
- 3 config templates
- 3 firmware versions, 2 upgrade tasks
- 3 device groups with hierarchy
- PM counters, KPI definitions/values, MR files/records
- Audit logs, alarm rules, KPI thresholds, system/NE message logs
- 2 backup tasks, 2 backup schedules, 3 managed files, 1 MML script (Sprint 8)
