# Review: RBAC device group and system menu API permissions

## Scope

- Issue: #575
- Files reviewed:
  - `omcgo/internal/admin/permission_service.go`
  - `omcgo/internal/admin/permission_service_test.go`
  - `omcmb/frontend-core/src/hooks/api/useDevices.ts`
  - `omcmb/webcode/src/pages/system/RolePermission/index.tsx`

## Summary

This change fixes three related permission defects:

1. Device data permissions now expand selected parent groups to include child groups before device list filtering.
2. Device group mutations invalidate both device-page and role-management group query keys.
3. Role create/edit saves inferred read-only API endpoint permissions for selected system management menus.

## Findings

Result: PASS_WITH_WARNINGS

- CRITICAL: none.
- WARNING: `cd omcmb && npm run typecheck` currently fails on unrelated pre-existing frontend type errors in MML/MR/OPS/performance/software pages, mostly invalid Ant Design props such as `orientation` on `Space`, numeric Drawer `width`, and missing `react-resizable` declarations. The modified role-permission file has no VS Code diagnostics, and the Docker web build completed successfully during verification.
- INFO: Remaining 403 responses observed in browser verification are global header/dashboard polling endpoints (`notifications/unread-count`, `alarms/statistics`, `indicators`), not system management page business APIs. Treat as a separate product-policy follow-up if desired.

## Verification

- `cd omcgo && go test ./internal/admin -run 'TestPermissionService_'`: pass.
- `git diff --check`: pass.
- `cd omcmb && npm run typecheck`: fail due unrelated existing errors outside this change.
- Browser verification completed against Docker runtime:
  - `verify_max_user` sees 4 devices in device list.
  - Admin-created device group appears immediately in role management data-permission tree.
  - `verify_system_user` gets HTTP 200 for system management business APIs including users, roles, menus, audit logs, sysConfig, api-endpoints, dictionaries, and device-groups tree.

## Notes

- The change is intentionally scoped to RBAC/data-permission behavior and role save semantics.
- No database migration is required.