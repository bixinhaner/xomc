# Task 11 report — v3 STARFORGE MML script library

## Scope

- Added STARFORGE `ScriptImportDialog` for TXT upload/re-import, template download, server validation snapshots, read-only plan preview, severity filters, line report download, typed 422/409 handling, warning confirmation, Escape close, and visible validation loading text.
- Added `ScriptExecutionDialog` for immediate/suspended/scheduled/periodic execution, date/time normalization, offline and failed retry strategies, typed server validation, warning confirmation, Escape close, and visible submission loading text.
- Rewired the v3 script page toolbar/detail drawer to the server-authoritative import and execution dialogs. Removed browser-side parser/task payload construction from the route.
- Added Vitest/jsdom/Testing Library entrypoint and capability-alignment tests.

## Red/green verification

Red before implementation:

```text
cd omcmb/webcode-v3 && npm run test -- --run src/pages/mml/script/__tests__/ScriptImportDialog.test.tsx
FAIL — Failed to resolve import "../ScriptImportDialog" (dialog did not exist)
```

Green:

```text
cd omcmb && npm install --package-lock-only
cd omcmb/webcode-v3 && npm run test -- --run src/pages/mml/script/__tests__/ScriptImportDialog.test.tsx src/pages/mml/script/__tests__/ScriptExecutionDialog.test.tsx
PASS — 2 files, 6 tests
cd omcmb/webcode-v3 && npm run typecheck
PASS
```

The focused tests cover TXT upload, template download, read-only content (no browser editor), warning confirmation, error filtering and disabled save, replacement validation endpoint, immediate/suspended/scheduled/periodic controls, offline/failed retry fields, 409 warning confirmation, 422 mixed error handling, date/time normalization, Escape, and readable loading/status text.

## Concerns

- Vitest emits jsdom's expected `Not implemented: navigation to another Document` notice when the template-download anchor is clicked; the test still passes and does not rely on browser navigation.
- `cd omcmb && npm run skin-parity` PASS — v2/v3 each report 129 routes and 37 visible menus, aligned with v1.
