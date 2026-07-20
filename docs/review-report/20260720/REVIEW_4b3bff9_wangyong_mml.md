# Code Review: MML Catalog Regroup And Stats Cleanup

Date: 2026-07-20
Branch: fix/mml-catalog-regroup-stats
Base: 4b3bff9
Scope: mml
Result: PASS_WITH_WARNINGS

## Reviewed Changes

- Pruned and normalized CMCC TD-LTE v2.3 MML catalog JSON data.
- Added seed migration `000003_prune_device_info_mml_sub_fields.sql` for device-info, software-version, management-server, and alarm command cleanup.
- Added QA archive for the MML device-info regrouping work.
- Updated MML admin/catalog UI to show target object rows for ADD/RMV commands without editable sub-fields.
- Fixed derived datamodel stats to match actual command and path counts.

## Findings

### WARNING

- `omcgo/migrations/seed/000003_prune_device_info_mml_sub_fields.sql` is intentionally forward-only. This matches the cleanup nature of the data change, but reviewers should treat rollback as restore-from-backup or follow-up seed repair rather than goose down reversal.

### INFO

- The datamodel `stats.commandLeavesTotal` and `stats.uniqueParams` now match actual content counts: 183 commands and 621 unique `treeNodeRefs` values.
- ADD/RMV object-path rows are rendered as synthetic read-only table rows and are not passed to edit/delete handlers.

## Verification

- `git diff --check` passed.
- `jq empty omcgo/data/mml-catalog/cmcc-tdlte-v2.3.json` passed.
- `jq empty omcgo/datamodels/mml-catalog/cmcc-tdlte-v2.3.json` passed.
- Datamodel stats consistency check passed: `ok=true stats.commands=183 actual.commands=183 stats.uniqueParams=621 actual.uniqueRefs=621`.
- `go test ./internal/mml ./internal/config/parammodel/mmlstandardloader` passed.
- `go build ./...` passed in `omcgo`.
- `npm run typecheck` passed in `omcmb/webcode`.

## Conclusion

No CRITICAL findings. The change is acceptable to commit and open an MR.
