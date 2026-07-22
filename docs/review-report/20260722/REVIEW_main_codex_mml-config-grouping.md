# Review Report: MML 配置分组与命令排序

Date: 2026-07-22
Branch: fix/mml-config-grouping
Scope: mml / migration / frontend

## Result

PASS

## Findings

No CRITICAL findings.

## Notes

- MML 命令排序改为按逻辑名称聚合，再按 ADD/RMV/MOD/LST 排序；后端树、扁平树和前端管理目录保持一致。
- seed 与部署 SQL 增补 DNS、LAN、SignallingTrace、NeighborList、GSM 小区命令整合和 HALOB 同级展示等配置；部署 SQL 均使用幂等 upsert/update 形态。
- seed 幂等静态测试原先只识别 `ON CONFLICT DO NOTHING`，本次同步扩展为识别 `ON CONFLICT DO UPDATE` 多行 upsert，匹配当前 seed 的实际写法。

## Verification

- `cd omcgo && go test ./internal/mml` — PASS
- `cd omcgo && go build ./...` — PASS
- `cd omcgo && go test ./test/integration -run TestSeedBaselineHasOnConflict -v` — PASS
- `cd omcgo && go test ./test/integration` — PASS
- `cd omcgo && go test ./...` — PASS
- `cd omcmb && npm run typecheck` — PASS
- Local PostgreSQL check: `SF_HALOB` and `chapter:SF` both have `nlevel(path)=1`; `LST SF_HALOB` / `MOD SF_HALOB` remain under `HALOB参数管理`.

## Risk

- MML seed SQL change volume is large because the baseline contains generated catalog data; rollback should use the deployment SQL blocks or revert the seed change as a whole.
- UI ordering depends on logical names being present; backend fallback to logical code remains in place for commands without localized logical names.
