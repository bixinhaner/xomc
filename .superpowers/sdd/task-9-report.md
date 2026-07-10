# Task 9 report — v1 MML TXT script-library UI

## Files

- Added `omcmb/webcode/src/pages/mml/ScriptTask/ScriptImportModal.tsx` for new/re-import flows, server validation, warning confirmation, template download, and save gating.
- Added `omcmb/webcode/src/pages/mml/ScriptTask/ScriptImportPreview.tsx` for server-authoritative summary, read-only plan rows, line issues, error/warning filters, and issue report download.
- Added `omcmb/webcode/src/pages/mml/ScriptTask/ScriptExecutionDrawer.tsx` for task name, execution mode/time, retry options, warning confirmation retry, and visible validation issues.
- Reworked `omcmb/webcode/src/pages/mml/ScriptTask/index.tsx` to expose TXT import, read-only details, re-import/download/edit-basic-info/delete/execute actions; removed browser content editor, command selector, and execution-mode selector from the page.
- Added interaction coverage in `ScriptImportModal.test.tsx` and `ScriptExecutionDrawer.test.tsx`.
- Marked Task 9 progress complete in `docs/superpowers/plans/2026-07-10-mml-script-txt-import.md`.

## Tests and commands

1. Red phase (before implementation):
   - `cd omcmb/webcode && npm run test -- --run src/pages/mml/ScriptTask/__tests__`
   - Failed as expected because the two new component modules did not exist.
2. Green interaction suite:
   - `cd omcmb/webcode && npm run test -- --run src/pages/mml/ScriptTask/__tests__`
   - Result: 3 files, 7 tests passed.
3. Type check:
   - `cd omcmb/webcode && npm run typecheck`
   - Result: TypeScript exited 0.

## Concerns

- Ant Design emits existing jsdom `getComputedStyle` and deprecation warnings during tests; no test failures or TypeScript errors remain.
- The shared layer currently exposes only the generic validation mutation, so re-import validation calls the reviewed `mmlApi.validateScriptReplacement` directly; save/execute continue through the Task 8 mutation hooks.

## Reviewer follow-up

- Typed 409 execution validation warnings now remain in the drawer, open the warning confirmation, and retry with `confirmWarnings=true`.
- Every upload clears the prior validation snapshot first; a failed replacement upload cannot reuse an older token or preview.
- Issue cards and report downloads follow the selected all/error/warning filter.
- Scheduled execution now requires `scheduledAt`; periodic execution renders and validates date range plus time, mapping them to `periodStart`, `periodEnd`, and `periodTime`.
- Added regression coverage for typed warning rejection, stale upload clearing, issue-card filtering, and scheduled/periodic form boundaries.

Follow-up verification:

- `cd omcmb/webcode && npm run test -- --run src/pages/mml/ScriptTask/__tests__` — 4 files, 12 tests passed.
- `cd omcmb/webcode && npm run typecheck` — TypeScript exited 0.
