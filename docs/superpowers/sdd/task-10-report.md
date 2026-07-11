# Task 10 report — v2 MML script library

## Files

- `omcmb/webcode-v2/src/pages/mml/components/ScriptImportDialog.tsx`: TXT import/re-import, template download, server validation summary/issues, read-only plan table, error/warning filters and warning confirmation.
- `omcmb/webcode-v2/src/pages/mml/components/ScriptExecutionDialog.tsx`: server-snapshot execution form with immediate/suspended/scheduled/periodic modes, period fields and offline/failed retry fields; typed API validation/error handling.
- `omcmb/webcode-v2/src/pages/mml/ScriptTask.tsx`: v2 toolbar, detail re-import action, and execution/import dialog wiring; removed browser parser/task payload construction.
- `omcmb/webcode-v2/src/pages/mml/__tests__/ScriptImportDialog.test.tsx`: interaction coverage for upload, validation, warning confirmation, read-only behavior and disabled save on errors.
- `omcmb/webcode-v2/vitest.config.ts`, `src/test/setup.ts`, `package.json`, `omcmb/package-lock.json`: Vitest/jsdom/Testing Library entrypoint and aliases.

## Red/green verification

Red (before components existed):

```text
npm run test -- --run src/pages/mml/__tests__/ScriptImportDialog.test.tsx
FAIL — Failed to resolve import ../components/ScriptImportDialog
```

Green:

```text
cd omcmb/webcode-v2 && npm run test -- --run src/pages/mml/__tests__/ScriptImportDialog.test.tsx
Test Files 1 passed; Tests 2 passed

cd omcmb/webcode-v2 && npm run typecheck
PASS
```

`npm install --package-lock-only` was run from `omcmb` to update the shared lockfile.

## Concerns

- v2 has no shared Dialog primitive yet, so these dialogs use the existing Tailwind modal shell while preserving the same interaction contract.
- Error messages from typed `MMLScriptImportApiError` include HTTP status (notably 422/409) and retain server validation snapshots; no browser-side parser or command selector is used.
