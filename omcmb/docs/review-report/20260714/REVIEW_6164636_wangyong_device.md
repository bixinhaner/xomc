# Review Report: gNB Sync Source Quick Settings

- Date: 2026-07-14
- Author: wangyong
- Scope: device / quicksettings
- Base: 6164636
- Result: PASS

## Summary

This review covers the staged changes that add the BaiBNQ gNB sync source quick-settings group, wire it as a device-level outer group, and update the quick-settings form to support the required GNSS multi-select and forced-sync switch behavior.

## Findings

No CRITICAL findings.

## Checks

- Verified that `Device.DeviceInfo.iForcedSyncControlSwitch` is modeled as `U_INT` with `0..1` constraints and displayed as a switch.
- Verified that the forced-sync switch serializes to `1` / `0` for save and realtime validation.
- Verified that empty forced-sync values render as enabled by default and do not trigger an unintended write when unchanged.
- Verified that the removed PPS time offset quick-setting has no remaining gNB sync-source UI logic.
- Verified that gNB sync-source schema loading is limited to `Device.FAP.` and `Device.DeviceInfo.`.
- Verified that BM bitmask sync-source behavior remains routed through bitmask serialization.

## Validation

- `git diff --check` - pass
- `go test ./internal/quicksettings` - pass
- `npm run typecheck` in `omcmb` - pass
- Local Docker Compose hot redeploy - pass
- `curl -I --max-time 10 http://localhost:8081/` - `HTTP/1.1 200 OK`

## Risk

- The gNB sync-source UI relies on enum values from BaiBNQ quicksettings XML matching device-reported values exactly.
- PTP and GNSS field visibility depends on `PpsTimeMode` current/draft value; device-side non-standard values will hide dependent fields until mapped.
