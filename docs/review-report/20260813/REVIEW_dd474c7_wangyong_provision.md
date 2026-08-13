# Code Review: Plug-and-Play Parameter Configuration Compatibility

- Date: 2026-08-13
- Base: `dd474c7`
- Author: `wangyong`
- Scope: `provision`, `PlugAndPlay`
- Result: **PASS_WITH_WARNINGS**

## Summary

This change aligns common and per-device parameter editors with product quick-settings metadata, adds row-level workbook download, preserves workbook TRPath mappings across export/import, and compiles imported workbook fields through sheet-qualified mappings with legacy alias compatibility.

## Review Findings

### CRITICAL

None.

### WARNING

1. The repository-wide `go test ./...` gate has a pre-existing failure in `internal/product`: `BLN.xml` declares 380 entries while the parser reports 329. This change does not modify BLN product metadata. The affected `internal/provision` package passes in full.

### INFO

1. Explicit workbook mappings are keyed by normalized `sheet + header`, preventing same-named columns in different sheets or instances from overwriting each other.
2. Strict mapped workbooks still reject unknown populated columns, while registered legacy aliases such as `CELL.*gNB ID` can safely fall back to the unique product definition.
3. Regression coverage includes concrete-path/template-path compatibility, duplicate column conflicts, cross-sheet names, all quick-settings aliases, workbook round trips, editor persistence, and common-network materialization.
4. Browser verification used the logged-in local UI and stopped before the final policy execution action, avoiding a real device configuration dispatch.

## Validation

- `cd omcgo && go build ./...` — passed.
- `cd omcgo && go test ./internal/provision -count=1` — passed.
- `cd omcmb && npm run typecheck` — passed.
- `cd omcmb/webcode && npm test -- --run src/pages/device/PlugAndPlay` — 26 files, 114 tests passed.
- `cd omcmb/webcode && npm run build` — passed before submission cleanup.
- Local Docker Compose app/web/acs services — running; `http://127.0.0.1:8081/` returned HTTP 200.
- Download `EXAMPLE-SN-001-参数配置.xlsx` and re-import with replacement mode — preview showed 0 added, 1 overwritten, 0 conflicts; import button enabled.
- Current stored BaiBNQ policy rows compiled successfully against database-backed product mappings without `SSB Frequency` or `CELL.*gNB ID` mapping errors.

## Risk Assessment

- Compatibility behavior is intentionally stricter for unknown workbook columns and more permissive only for unique, registered legacy aliases.
- Final execution against the selected online device was not triggered during verification.
