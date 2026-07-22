# Review: MML catalog seed bindings

Date: 2026-07-22
Scope: migration
Branch: fix/mml-catalog-seed-bindings

## Result

PASS

## Reviewed Changes

- Consolidated reviewed MML catalog bindings into `omcgo/migrations/seed/000001_init_seed.sql`.
- Added and adjusted command/path bindings for GPS, DeviceInfo EU/RU/SAS/1588, WAN/IPv4/IPv6/VLAN/PPPoE, IPsec, MR, QOS, SIB, HALOB, and BWPDL coverage.
- Kept QOS and SIB command entries as direct children of the cell service group instead of visible second-level folders.
- Marked SAS and 1588 groups as first-level catalog groups.

## Findings

- CRITICAL: None.
- WARNING: The seed block is intentionally large because it consolidates multiple MML catalog corrections into the baseline seed. Future catalog changes should prefer smaller focused seed blocks.
- INFO: Audit CSV/Markdown outputs were refreshed outside this repository under `/Users/wangyong/OBJECT/Codex/outputs/mml-trpath-audit-20260722/`.

## Validation

- `awk '/^-- 2026-07-22 MML catalog reviewed path bindings merged from retired seed 000002\\./{flag=1;next}/^-- \\+goose Down/{flag=0}flag' omcgo/migrations/seed/000001_init_seed.sql | docker exec -i goomc-local-postgres-1 psql -U omcgo -d omcgo -v ON_ERROR_STOP=1` — PASS.
- `git diff --check` — PASS.
- BWPDL database check — PASS: total 59, `LST SH_SUB_01` 59, `MOD SH_SUB_01` 59, missing 0.
- SAS/1588/HALOB database spot check — PASS: groups active and expected command path counts present.

## Risk

- Runtime risk is limited to MML catalog seed data. Existing environments require the reviewed seed block to be applied or a fresh seed deployment to observe the catalog corrections.
