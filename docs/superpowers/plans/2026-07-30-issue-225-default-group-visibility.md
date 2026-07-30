# Issue 225 Default Group Visibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a role granted the default L2 device group see both devices physically assigned to that group and legacy devices without a membership row.

**Architecture:** Keep the default L2 UUID in the real visible-group set while also enabling the legacy ungrouped compatibility predicate. Reuse that single split contract in both group-only and group-plus-technology authorization paths so list SQL, SN SQL, and per-device authorization agree.

**Tech Stack:** Go, Squirrel, pgx, standard `testing`

## Global Constraints

- Preserve the `nil` superadmin, empty fail-closed, non-empty scoped-permission contract.
- Preserve technology filtering for device visibility grants.
- Use Squirrel for generated SQL; do not introduce ORM or string-concatenated query values.
- Do not add a database migration or change the frontend.

---

### Task 1: Lock the default-group compatibility contract with failing tests

**Files:**
- Test: `omcgo/internal/authz/authz_test.go`

**Interfaces:**
- Consumes: `SplitVisibleGroups`, `AuthorizeDeviceAccess`, `AuthorizeDeviceAccessByGrants`, `ApplyDeviceVisibilityFilter`, `ApplyDeviceVisibilityGrantsFilter`, `ApplyDeviceSNVisibilityFilter`, `VisibleSNSubquerySQL`
- Produces: Regression coverage requiring the default L2 UUID to remain a real group while also matching legacy ungrouped devices.

- [x] **Step 1: Add physical-default-membership authorization tests**

Add cases where both the device membership and the role grant contain `global.DefaultLevel2GroupID`; expect authorization to succeed.

- [x] **Step 2: Strengthen SQL tests**

For default-group visibility, assert both a `device_group_members` group predicate containing the default UUID and the legacy `NOT EXISTS` predicate.

- [x] **Step 3: Run the focused tests and verify RED**

Run:

```bash
GOCACHE=/private/tmp/xomc-go-build-issue225 go test ./internal/authz -run 'Test(AuthorizeDeviceAccess|AuthorizeDeviceAccessByGrants|ApplyDeviceVisibilityGrantsFilter|ApplyDeviceVisibilityFilter|ApplyDeviceSNVisibilityFilter|VisibleSNSubquerySQL)$'
```

Expected: failures showing that the default UUID is omitted from real group membership matching.

### Task 2: Unify default-group visibility semantics

**Files:**
- Modify: `omcgo/internal/authz/authz.go`
- Test: `omcgo/internal/authz/authz_test.go`

**Interfaces:**
- Consumes: `global.DefaultLevel2GroupID`
- Produces: `SplitVisibleGroups` returns the default UUID in `realGroups` and `includeUngrouped=true`; grant filters and intersection checks use the same contract.

- [x] **Step 1: Keep the default UUID in real groups**

Change `SplitVisibleGroups` so it appends every visible UUID, setting `includeUngrouped` additionally when it sees the default L2 UUID.

- [x] **Step 2: Reuse the split helper in grant SQL**

Replace the grant filter's custom loop with `SplitVisibleGroups(grant.GroupIDs)` and emit the legacy `NOT EXISTS` branch only when requested.

- [x] **Step 3: Match physical default membership in per-device authorization**

Build the grant visibility map from all grant group IDs, including the default L2 UUID.

- [x] **Step 4: Run focused tests and verify GREEN**

Run the Task 1 command and require exit code 0.

### Task 3: Verify affected consumers

**Files:**
- Verify: `omcgo/internal/authz`
- Verify: `omcgo/internal/device`
- Verify: `omcgo/internal/topology`

**Interfaces:**
- Consumes: unified authorization helpers
- Produces: evidence that list, per-device authorization, default-group assignment, and topology behavior remain compatible.

- [x] **Step 1: Run affected backend packages**

```bash
GOCACHE=/private/tmp/xomc-go-build-issue225 go test ./internal/authz ./internal/device ./internal/topology
```

- [x] **Step 2: Run formatting and inspect the diff**

```bash
gofmt -w omcgo/internal/authz/authz.go omcgo/internal/authz/authz_test.go
git diff --check
git diff -- omcgo/internal/authz/authz.go omcgo/internal/authz/authz_test.go
```

- [x] **Step 3: Confirm the worktree contains only scoped changes**

```bash
git status --short
```
