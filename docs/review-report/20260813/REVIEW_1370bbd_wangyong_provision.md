# Code Review: Product-Aware Quick Settings Enum Values

- Date: 2026-08-13
- Base: `1370bbd`
- Author: `wangyong`
- Scope: `provision`, `quicksettings`, `PlugAndPlay`
- Result: **PASS_WITH_WARNINGS**

## Summary

This change makes product parameter mappings authoritative for quick-settings wire types, ranges, read-only state, and enum values. Plug-and-Play renders the selected product's enum values, while policy compilation validates them again and safely translates legacy LTE bandwidth values only when the target product explicitly supports the counterpart.

## Findings

### CRITICAL

None.

### WARNING

1. The repository-wide `go test ./...` gate still has the pre-existing `internal/product` failure: `BLN.xml` declares 380 entries while the parser reports 329. This change does not modify BLN product metadata. The affected `internal/provision` and `internal/quicksettings` packages pass.

### INFO

1. BLQ/BM/MLN/MLQ use numeric LTE bandwidth wire values (`25/50/75/100`), while ENB_DEFAULT_098/181 use prefixed values (`n25/n50/n75/n100`); the implementation now preserves this product-specific contract.
2. Legacy LTE bandwidth conversion is restricted to the known bandwidth value set and only succeeds when the target product declares the converted enum value.
3. Invalid enum values are rejected before task dispatch instead of being sent to the device.
4. The quick-settings endpoint remains backward-compatible when no parameter registry is injected, while the production router injects the registry and enriches both device- and param-model-based responses.

## Validation

- `cd omcgo && go build ./...` — passed.
- `cd omcgo && go test ./internal/provision ./internal/quicksettings -count=1` — passed.
- `cd omcgo && go test ./...` — pre-existing BLN metadata-count failure only.
- `cd omcmb && npm run typecheck` — passed.
- `cd omcmb/webcode && npm test -- --run src/pages/device/PlugAndPlay` — 26 files, 115 tests passed.
- Local Docker Compose web/app/acs rebuild and restart — passed.
- `curl -I http://localhost:8081/` — HTTP 200 with the rebuilt static bundle.

## Risk Assessment

- No database migration or public request contract change.
- Product mappings are now authoritative where available; deployments with incomplete mappings retain existing quick-settings metadata and the backend compiler still blocks invalid configured enums when product enum constraints exist.
