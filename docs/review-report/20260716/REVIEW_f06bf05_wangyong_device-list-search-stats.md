# Review: device-list-search-stats

- Reviewer: Codex
- Author: wangyong
- Base: f06bf05
- Scope: device
- Result: PASS

## Summary

本次变更修复设备列表搜索分页参数与后端契约不一致、COUNT 因设备分组 JOIN 重复计数、以及“当前告警”统计口径误用“有告警设备数”的问题。前端设备列表继续使用筛选条件下的后端 stats，后端将 `alarmed` 改为活动告警总条数，与列表行内 `active_alarm_count` 加总一致。

## Findings

无 CRITICAL 问题。

## Checks

- Backend SQL: 使用 squirrel 参数化查询与既有 filter 拼装逻辑，未新增字符串拼接用户输入。
- Count semantics: `ListDevicesWithInfo` 的总数改为 `COUNT(DISTINCT d.id)`，避免 `device_group_members` 一对多放大。
- Alarm semantics: `ComputeListStats` 使用每设备活动告警子查询并外层 `SUM(active_alarm_count)`，口径与列表行内告警数量一致。
- Frontend API: `page_size` / `sort_by` / `sort_dir` 映射后端 snake_case 参数，避免后端使用默认分页导致长短搜索数量异常。
- Refresh behavior: 告警轮询仍触发设备列表 invalidation，但统计卡片不再用全局告警数覆盖筛选 stats。
- Mock behavior: mock service 增加 `searchText` 多字段包含搜索，测试覆盖“长词结果不会多于短词”。

## Validation

- `go test ./internal/device` — PASS
- `go build ./cmd/app` — PASS
- `npm run typecheck` in `omcmb` — PASS
- `npx vitest run ../frontend-core/src/services/api/__tests__/deviceApi.test.ts ../frontend-core/src/mock/services/__tests__/deviceService.test.ts` — PASS, 2 files / 26 tests
- `npm run build` in `omcmb/webcode` — PASS
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web` — PASS
- `curl -I --max-time 10 http://localhost:8081/` — PASS, HTTP 200
- Post-deploy API smoke: `GET /api/v1/devices?page=1&page_size=20` as admin returned `total=7`, `stats.alarmed=21`, row alarm sum `21`.

## Residual Risk

- `stats.alarmed` 字段名沿用历史名称，但语义已明确为活动告警总条数；前后端注释已同步，避免后续误读。
- 本地数据库曾缺少 `recycle_type` / `recycle_executor` 迁移列，已在本地补齐用于恢复环境；本次提交不包含数据库数据修复。
