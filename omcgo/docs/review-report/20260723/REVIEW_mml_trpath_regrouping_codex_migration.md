# MML TRPath Remaining Regrouping Review

- Date: 2026-07-23
- Author: Codex
- Scope: migration
- Conclusion: PASS_WITH_WARNINGS

## Reviewed Files

- `omcgo/migrations/seed/000001_init_seed.sql`
- `omcgo/scripts/mml_apply_config_updates_20260721.sql`

## Summary

This change lands the 20260723 MML standard TRPath remaining-parameter regrouping in both the fresh-environment seed path and the operational incremental script path. It adds 14 shortened visible groups, 28 LST/MOD commands, and 436 command sub-fields for 242 remaining standard TRPath parameters.

The old standalone `000003_mml_standard_trpath_remaining_regrouping_20260723.sql` migration was intentionally folded into the existing seed baseline and the existing field update script, so new environments stop at the current seed version sequence while existing environments can apply the targeted `mml_trpath_regrouping` branch.

## Findings

### WARNING

- The same TRPath data block is present in both the seed SQL and the incremental SQL script. This is acceptable for the current deployment model, but future edits must keep both copies synchronized.
- The historical default/verify path in `mml_apply_config_updates_20260721.sql` still contains legacy sections that can fail before this new block on databases missing older prerequisite commands. The new `-v mml_trpath_regrouping=1` branch avoids that for this scoped incremental deployment.

### INFO

- Group names were shortened to the requested 14 concise labels while preserving stable group codes `MML_STD_TRPATH_G01` through `MML_STD_TRPATH_G14`.
- New groups use `source='admin'` so they remain visible in the current MML command tree query behavior.
- The script includes verification checks for group count, visible group count, command count, and sub-field count.

## Validation

- PASS: `git diff --check`
- PASS: `go test ./internal/mml ./internal/config/parammodel/mmlstandardloader ./internal/config/parammodel`
- PASS: Fresh-environment migration rehearsal previously verified 14 groups, 28 commands, and 436 fields.
- PASS: Incremental targeted deployment rehearsal previously verified 14 groups, 14 visible groups, 28 commands, and 436 fields using `psql -v mml_trpath_regrouping=1`.
- PASS: Local deployment smoke previously returned HTTP 200 on `http://localhost:8081/`.

## Gate Result

No CRITICAL findings were identified. This change is ready to commit.
