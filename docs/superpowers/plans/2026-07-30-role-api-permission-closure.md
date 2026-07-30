# Issue #225 Role API Permission Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore explicit role API permission configuration and exact role-copy behavior so Issue #225 passes through the real custom-role authorization flow without manual database grants.

**Architecture:** Keep the existing three-track authorization model: `role_menus` controls UI visibility, `role_api_permissions` controls endpoint access, and `role_device_groups` controls device visibility. Add a small pure frontend model for safe GET-only suggestions, restore the API permission tab and specialized save endpoint, and make `CopyRole` use a narrow binding repository with compensating cleanup on failure.

**Tech Stack:** Go 1.x, Gin, pgx, Squirrel, Casbin, React 19, TypeScript 6, Ant Design 6, TanStack Query, Vitest.

## Global Constraints

- Keep `RequireAPIPermission` strict and fail closed.
- Never grant the operator or built-in role baseline to a custom role.
- Only explicitly mapped `GET` endpoints may be suggested automatically.
- Editing an existing role must load exactly the endpoint IDs stored in `role_api_permissions`.
- `POST`, `PUT`, `PATCH`, and `DELETE` endpoints require explicit administrator selection.
- Use Squirrel + pgx for backend SQL; do not introduce an ORM or string-concatenated SQL.
- Wrap new backend errors with `fmt.Errorf("context: %w", err)`.
- User-visible frontend text must use i18n.
- Do not add or modify database migration files.
- Validate user-visible behavior with a real browser.

---

### Task 1: Make Role Copy Preserve API Grants and Fail Cleanly

**Files:**
- Modify: `omcgo/internal/admin/service.go`
- Modify: `omcgo/internal/admin/service_test.go`

**Interfaces:**
- Consumes: `PgRoleRepository.GetDeviceGroupData`, `SetDeviceGroupData`, `GetRoleApiEndpointIDs`, and `SetRoleApiEndpoints`.
- Produces: private `roleCopyBindingRepository` and `AdminService.copyRoleFailure(ctx, copiedID, cause) error`.

- [x] **Step 1: Write failing service tests for exact API permission copy**

Add callbacks to `mockRoleRepo` for the four binding operations and add tests with literal UUID fixtures:

```go
func TestAdminService_CopyRole_CopiesExactAPIPermissions(t *testing.T) {
    sourceID := uuid.MustParse("30000000-0000-0000-0000-000000000001")
    copiedID := uuid.MustParse("30000000-0000-0000-0000-000000000002")
    endpointIDs := []uuid.UUID{
        uuid.MustParse("40000000-0000-0000-0000-000000000001"),
        uuid.MustParse("40000000-0000-0000-0000-000000000002"),
    }
    // Configure source lookup, copied role creation, empty group/menu bindings,
    // and capture SetRoleApiEndpoints. Assert the captured role ID and exact
    // endpoint UUID order equal the literal endpointIDs slice.
}
```

Add separate tests for:

- an empty source endpoint set being written as an empty set;
- API endpoint read/write failure deleting the newly created role;
- cleanup deletion failure preserving both the copy error and cleanup error text;
- menu or device-group copy failure stopping before API policy persistence.

- [x] **Step 2: Run the focused tests and verify RED**

Run:

```bash
cd omcgo
go test ./internal/admin -run 'TestAdminService_CopyRole_' -count=1
```

Expected: FAIL because `CopyRole` skips API grants, ignores binding errors, and has no cleanup path.

- [x] **Step 3: Add the narrow binding interface and minimal copy implementation**

Add to `service.go`:

```go
type roleCopyBindingRepository interface {
    GetDeviceGroupData(context.Context, uuid.UUID) (*RoleDeviceGroupData, error)
    SetDeviceGroupData(context.Context, uuid.UUID, RoleDeviceGroupData) error
    GetRoleApiEndpointIDs(context.Context, uuid.UUID) ([]uuid.UUID, error)
    SetRoleApiEndpoints(context.Context, uuid.UUID, []uuid.UUID) error
}
```

Store the asserted interface on `AdminService` during construction. Update `CopyRole` to:

1. create the non-built-in role;
2. read and write device-group data;
3. read and write menu IDs;
4. read and write the exact API endpoint IDs last;
5. return the copied role only after all binding operations succeed.

API permissions are written last so a later menu/group failure cannot leave stale in-memory Casbin policies. On any binding error, call `roleRepo.Delete(ctx, copied.ID)` and return a wrapped error. If deletion fails, wrap both errors with enough context to identify the original failure and cleanup failure.

- [x] **Step 4: Run focused and related backend tests and verify GREEN**

Run:

```bash
cd omcgo
gofmt -w internal/admin/service.go internal/admin/service_test.go
go test ./internal/admin -run 'TestAdminService_CopyRole_|TestPgRoleRepositoryPolicy' -count=1
```

Expected: PASS.

---

### Task 2: Add a Tested GET-Only API Suggestion Model

**Files:**
- Create: `omcmb/webcode/src/pages/system/RolePermission/roleApiPermissionModel.ts`
- Create: `omcmb/webcode/src/pages/system/RolePermission/roleApiPermissionModel.test.ts`

**Interfaces:**
- Consumes: `Menu[]` and `ApiEndpoint[]`.
- Produces:
  - `inferReadApiEndpointIds(menuIds: string[], menus: Menu[], endpoints: ApiEndpoint[]): string[]`
  - `applyNewApiSuggestions(selectedIds: string[], suggestedIds: string[], alreadySuggestedIds: string[], autoSelectedIds: string[]): { selectedIds: string[]; suggestedIds: string[]; autoSelectedIds: string[] }`

- [x] **Step 1: Write failing pure-model tests**

Use complete literal menu and endpoint fixtures. Cover:

```ts
it('suggests only GET endpoints from explicitly mapped menu groups', () => {
  expect(inferReadApiEndpointIds(
    ['menu-system-role'],
    roleMenuFixture,
    endpointFixture,
  )).toEqual(['endpoint-get-roles', 'endpoint-get-menus']);
});

it('never suggests write endpoints', () => {
  expect(inferReadApiEndpointIds(
    ['menu-system-role'],
    roleMenuFixture,
    endpointFixture,
  )).not.toContain('endpoint-put-role');
});

it('keeps administrator selections when applying new suggestions', () => {
  expect(applyNewApiSuggestions(
    ['endpoint-explicit-post'],
    ['endpoint-suggested-get'],
    [],
  )).toEqual({
    selectedIds: ['endpoint-explicit-post', 'endpoint-suggested-get'],
    suggestedIds: ['endpoint-suggested-get'],
  });
});

it('does not re-add a suggestion the administrator removed', () => {
  expect(applyNewApiSuggestions(
    [],
    ['endpoint-suggested-get'],
    ['endpoint-suggested-get'],
  )).toEqual({
    selectedIds: [],
    suggestedIds: ['endpoint-suggested-get'],
  });
});
```

Also assert that an unmapped menu returns `[]` and duplicate IDs are removed without reordering the administrator's existing selections.

- [x] **Step 2: Run the model test and verify RED**

Run:

```bash
cd omcmb
npm test --workspace webcode -- roleApiPermissionModel.test.ts
```

Expected: FAIL because the model file and exported functions do not exist.

- [x] **Step 3: Implement the minimal pure model**

Move the existing historical `READ_API_GROUPS_BY_MENU_KEY` concept into the new file. Traverse selected menu descendants, collect only explicitly mapped `apiGroup` names, then return endpoint IDs satisfying:

```ts
endpoint.method.toUpperCase() === 'GET' &&
endpoint.apiGroup !== undefined &&
mappedGroups.has(endpoint.apiGroup)
```

`applyNewApiSuggestions` must preserve the selected list first, append each suggestion at most once
per create-drawer session, and remember all suggestions already offered. This lets an administrator
remove a suggested endpoint without a later menu change silently adding it back.
The final implementation also tracks which selections were automatic, removing only stale automatic
GET grants when the corresponding menu is unchecked while preserving explicit administrator grants.

- [x] **Step 4: Run the model tests and verify GREEN**

Run:

```bash
cd omcmb
npm test --workspace webcode -- roleApiPermissionModel.test.ts
```

Expected: PASS.

---

### Task 3: Restore Explicit API Permission Configuration and Persistence

**Files:**
- Modify: `omcmb/webcode/src/pages/system/RolePermission/index.tsx`
- Create: `omcmb/webcode/src/pages/system/RolePermission/rolePermissionSave.ts`
- Create: `omcmb/webcode/src/pages/system/RolePermission/rolePermissionSave.test.ts`
- Modify: `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- Modify: `omcmb/frontend-core/src/i18n/en-US/index.ts`
- Test: `omcmb/webcode/src/pages/system/RolePermission/roleApiPermissionModel.test.ts`

**Interfaces:**
- Consumes: `apiPermissionApi.listEndpoints`, `getRolePermissions`, `setRolePermissions`, and Task 2 model functions.
- Produces: API permission tab state, exact create/edit persistence, and i18n validation key `role.pleaseSelectApiPermission`.

- [x] **Step 1: Restore API endpoint query, state, tree, and exact role loading**

In `RolePermission/index.tsx`:

- import `useQuery`, `ApiEndpoint`, `apiPermissionApi`, and the Task 2 model;
- restore `selectedApiEndpointIds`, `expandedApiGroupKeys`, and `apiCheckStrictly`;
- add `suggestedApiEndpointIds` to remember which automatic suggestions have already been offered
  during the current create-drawer session;
- query all API endpoints through `apiPermissionApi.listEndpoints`;
- group endpoints by `apiGroup` and render Method + path leaf nodes;
- on edit/view, call `getRolePermissions(role.id)` and assign the returned endpoint IDs directly;
- do not run suggestion merging while loading an existing role.

- [x] **Step 2: Apply safe suggestions during create-mode menu selection**

When menu checked keys change in the create drawer:

1. resolve the selected menu IDs;
2. call `inferReadApiEndpointIds`;
3. call `applyNewApiSuggestions` with the current selected and already-suggested endpoint IDs;
4. write both returned arrays to state.

Do not call either suggestion function from edit/view loading. Reset `suggestedApiEndpointIds` when
the create drawer closes or saves successfully.

- [x] **Step 3: Restore API validation and specialized persistence**

Before create or edit submission:

```ts
if (selectedApiEndpointIds.length === 0) {
  message.warning(t('role.pleaseSelectApiPermission'));
  return;
}
```

On create and edit, serialize the currently displayed endpoint selection exactly:

```ts
await setRoleMenusMut.mutateAsync({ roleId: newRole.id, menuIds });
await setRoleApiPermissions.mutateAsync({
  roleId: newRole.id,
  endpointIds: selectedApiEndpointIds,
});
await setRoleDeviceGroupsMut.mutateAsync({
  roleId: newRole.id,
  deviceGroupIds: selectedDeviceGroupIds,
  networkTypes: selectedNetworkTypes,
});
```

Keep existing `try/catch` behavior so success UI runs only after all specialized requests resolve.
The final implementation routes this sequence through the tested
`saveRolePermissionBindings` orchestrator, which stops immediately on a failed API write.

- [x] **Step 4: Restore API permission tabs and reset state**

Add API permission entries between menu and data tabs for create, edit, and view drawers. Reset selected and expanded API state in every close/success cleanup path. Add:

```ts
'role.pleaseSelectApiPermission': '请选择 API 权限'
'role.pleaseSelectApiPermission': 'Please select API permissions'
```

to the existing zh-CN and en-US role sections.

- [x] **Step 5: Run frontend tests, typecheck, and lint**

Run:

```bash
cd omcmb
npm test --workspace webcode -- roleApiPermissionModel.test.ts
npm run typecheck
npm run lint -- --quiet
```

Expected: all commands PASS.

---

### Task 4: Full Verification, Browser Regression, and Commit

**Files:**
- Modify: `docs/superpowers/plans/2026-07-30-role-api-permission-closure.md` only to check completed steps.
- Verify all files changed by Tasks 1–3.

**Interfaces:**
- Consumes: completed backend and frontend behavior.
- Produces: a clean, tested branch with browser evidence and no manual API-permission SQL.

- [x] **Step 1: Run full relevant automated verification**

Run:

```bash
cd omcgo
go test ./...
cd ../omcmb
npm test --workspace webcode -- roleApiPermissionModel.test.ts
npm run typecheck
npm run lint -- --quiet
git diff --check
```

Expected: PASS.

Final evidence: backend `go test ./...` passed outside the filesystem sandbox (the suite's
miniredis/httptest/STUN cases require temporary local listeners); the two RolePermission Vitest files
passed (11 tests), TypeScript typecheck and Vite production build passed, scoped ESLint passed, and
`git diff --check` passed. The repository-wide lint command still reports two
pre-existing unrelated errors in `runtimeClient.test.ts` and
`mmlConsoleStore.ts`; neither file is modified by this plan.

- [x] **Step 2: Rebuild and restart local services from this worktree**

Run the project Docker backend build and V1 Vite frontend using the existing local ports. Confirm:

```bash
curl -sS -i --max-time 5 http://localhost:18081/healthz
curl -sS -I --max-time 5 http://127.0.0.1:3001/
```

Expected: both return HTTP 200.

- [x] **Step 3: Execute the real-browser positive and negative scenario**

Using the in-app browser:

1. log in as administrator;
2. create a new custom role with menu, explicit API, and default-group LTE/NR data permissions;
3. create a normal user assigned only to that role;
4. log in as the normal user;
5. verify dashboard summary and device list requests return 200;
6. verify the known physical LTE and NR default-group devices are visible;
7. verify legacy ungrouped devices remain visible;
8. invoke an unselected write endpoint and verify HTTP 403;
9. copy the role and verify exact menu, API, and data permission parity;
10. do not execute manual `INSERT INTO role_api_permissions`.

Post-review browser regression also verified that removing the dashboard menu reduced its automatic
GET grants from 15 to 0, the edit confirmation remained disabled until all three role-binding reads
completed, and the rebuilt backend copied the source role's exact `46 / 741` API grants plus
eNB/gNB/default-group data permissions.

Independent review found and verified fixes for stale async role responses, duplicate full-chain
saves, stale automatic grants, final copied-role read compensation, and canceled-request cleanup.
The final re-review reported no Critical or Important findings.

- [x] **Step 4: Review the final diff and commit**

Confirm only scoped files changed, then commit with:

```bash
git add omcgo/internal/admin/service.go \
  omcgo/internal/admin/service_test.go \
  omcmb/webcode/src/pages/system/RolePermission/index.tsx \
  omcmb/webcode/src/pages/system/RolePermission/roleApiPermissionModel.ts \
  omcmb/webcode/src/pages/system/RolePermission/roleApiPermissionModel.test.ts \
  omcmb/webcode/src/pages/system/RolePermission/rolePermissionSave.ts \
  omcmb/webcode/src/pages/system/RolePermission/rolePermissionSave.test.ts \
  omcmb/frontend-core/src/i18n/zh-CN/index.ts \
  omcmb/frontend-core/src/i18n/en-US/index.ts \
  docs/superpowers/plans/2026-07-30-role-api-permission-closure.md
git commit -m "fix(rbac): 恢复自定义角色 API 权限闭环"
```
