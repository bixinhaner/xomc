# Review Report

- Base: `8e92d05`
- Author: `wangyong`
- Scope: `device`
- Date: `2026-07-14`
- Result: `PASS`

## Summary

本次变更覆盖设备列表、设备详情和设备分组列表的前端展示逻辑：

- 设备列表新增 `MME连接状态` 列。
- 离线设备的展示态统一派生为小区激活状态 `inactive/Deactive`、RF 状态 `OFF`。
- 设备详情头部和小区表复用同一展示派生逻辑。
- 设备分组列表移除固定 `scroll.y=100`，改由 `DataTable.autoFitHeight` 根据容器高度计算纵向滚动，修复每页 100 条时无滚动条的问题。
- 增加状态派生和分组列表滚动配置的回归测试。

## Findings

### CRITICAL

None.

### WARNING

None.

### INFO

- 设备分组列表滚动属于视觉布局行为，当前单测锁定的是 DataTable 配置层，建议部署后在真实浏览器里把每页切到 100 条做一次手工确认。
- `displayActivationStatus*` 与 `displayRFStatus*` 是展示层派生函数，保留原始 `activationStatusOf` / `rfStatusOf` 语义不变，避免影响筛选选项和非展示场景。

## Validation

- `npm run typecheck`
- `npm run test --workspace webcode -- --run frontend-core/src/utils/__tests__/activationStatus.test.ts frontend-core/src/utils/__tests__/rfStatus.test.ts src/pages/device/DeviceGrouping/DeviceListPanel.test.tsx`
- `npm run build`
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web`
- `curl -I --max-time 10 http://localhost:8081/` -> `HTTP/1.1 200 OK`

## Decision

PASS. No blocking issues found.
