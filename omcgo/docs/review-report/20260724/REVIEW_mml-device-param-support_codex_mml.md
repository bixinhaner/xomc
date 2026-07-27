# Review: MML device parameter support

Date: 2026-07-24
Author: Codex
Scope: mml

## Summary

This review covers the device-specific MML command filtering fix, BaiBNQ top-level DeviceInfo standard path corrections, and the seed SQL fold-in.

## Findings

Result: PASS

- CRITICAL: none
- WARNING: none
- INFO: The seed SQL fix-up is compatible with existing rows and is harmless on empty seed state; BaiBNQ XML remains the fresh-import source of truth.

## Validation

- `go test -count=1 ./internal/config/parammodel -run 'TestBaiBNQTopLevelDeviceInfoUsesStandardPaths|TestBaiBNQGNBNameIsWritableNRCommonPath|TestBaiBNQNguBindInterfaceAndFallbackAreWritable'` - passed
- `go test -count=1 ./internal/mml -run 'TestBuildGroupTreeFilteredByDevice|TestBuildFlatGroupTreeFilteredByDevice|TestGetCommandSubFields_Device'` - passed
- `go test -count=1 ./cmd/app/provider` - passed
- `go build ./...` - passed
- Fresh deployment migration self-test: schema version 4, seed version 2, tsdb version 1 - passed
- Incremental migration self-test: no migrations to run for schema, seed, and tsdb - passed
