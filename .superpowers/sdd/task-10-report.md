# Task 10 report — v2 MML script library

## Files

- `omcmb/webcode-v2/src/pages/mml/components/ScriptImportDialog.tsx`: TXT import/re-import, template download, server validation summary/issues, read-only plan table, error/warning filters and warning confirmation.
- `omcmb/webcode-v2/src/pages/mml/components/ScriptExecutionDialog.tsx`: server-snapshot execution form with immediate/suspended/scheduled/periodic modes, period fields and retry fields; typed API validation/error handling.
- `omcmb/webcode-v2/src/pages/mml/ScriptTask.tsx`: toolbar, detail re-import action and dialog wiring; removed browser parser/task payload construction.
- `omcmb/webcode-v2/src/pages/mml/__tests__/ScriptImportDialog.test.tsx`: upload, validation, warning confirmation, read-only behavior and disabled-save interaction coverage.
- `omcmb/webcode-v2/vitest.config.ts`, `src/test/setup.ts`, `package.json`, `omcmb/package-lock.json`: Vitest/jsdom/Testing Library setup.

## Red/green verification

Red before implementation:

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

`npm install --package-lock-only` was run from `omcmb`.

Follow-up regression verification:

```text
cd omcmb/webcode-v2 && npm run test -- --run src/pages/mml/__tests__/ScriptImportDialog.test.tsx src/pages/mml/__tests__/ScriptExecutionDialog.test.tsx
Test Files 2 passed; Tests 5 passed

cd omcmb/webcode-v2 && npm run typecheck
PASS
```

The execution dialog now opens confirmation for warning-only rejected 409 responses, keeps mixed 422 error/warning responses in the error state, and normalizes periodic HTML date/time values to backend ISO/date-time formats.

Additional focused coverage now verifies replacement endpoint selection, template download invocation, actual severity filtering of preview rows, Escape close behavior for both dialogs, per-line execution issues, periodic date/time normalization, and scheduled datetime normalization. Latest focused run: 2 files / 9 tests passed; typecheck passed.

## Concerns

- v2 has no shared Dialog primitive, so dialogs use the existing Tailwind modal shell.
- Typed 422/409 API errors preserve server validation snapshots; warning-only 409 responses are confirmed explicitly, while mixed error+warning responses remain visible without confirmation.

## UI primitive follow-up

- Replaced the import preview’s raw plan table with the shared `@/components/ui/table` primitives.
- Replaced the visible execution-mode control with the shared `@/components/ui/select` primitive while retaining native test/form value synchronization.
- Focused MML tests: 2 files / 9 tests passed; `npm run typecheck` passed.
- Final accessibility follow-up: the hidden native Select test bridge is `aria-hidden` with `tabIndex=-1`; the visible execution-mode control remains the shared Select primitive.
