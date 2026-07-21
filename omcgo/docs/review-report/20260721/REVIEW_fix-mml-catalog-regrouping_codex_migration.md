# Review Report: MML catalog regrouping

- Date: 2026-07-21
- Branch: fix/mml-catalog-regrouping
- Scope:
  - omcgo/migrations/seed/000001_init_seed.sql
  - omcgo/scripts/mml_apply_config_updates_20260721.sql

## Result

PASS

## Summary

This change adjusts the MML350 catalog seed and update script to align parameter groups with the requested UI structure:

- Rename `Device.HaltReason` to restart reason and merge reason parameters into one query command.
- Split FAP-related entries into HANRU, 1588, NL, LAN configuration, and DNS configuration groups.
- Merge GPS read/write parameters into two flat commands: query GPS info and modify GPS info.
- Move Device.IP IPv4 address parameters into WAN IPv4 address configuration commands and remove the redundant Device.IP group entry.
- Merge FAP IPsec parameters into the existing IPsec query/modify commands and deprecate duplicate top-level entries.
- Add modify commands for writable parameters where the catalog previously only exposed query entries.

## Checks

- `git diff --check`: passed.
- SQL update script applied successfully against local `goomc-local` database during implementation.
- Local app/web containers restarted successfully with `deployments/docker/dc.sh restart app web`.
- `curl -I --max-time 10 http://localhost:8081/`: returned `HTTP/1.1 200 OK`.

## Notes

- No Go or frontend source files changed.
- Remaining risk is limited to catalog seed/update data correctness; live database checks confirmed command/path counts for the final GPS, WAN IPv4, DNS/LAN, restart reason, and IPsec group shapes.
