# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-07-10 19:20 |
| 提交 | 本地提交（本次审查范围） |
| 作者 | hezhenguo1046 |
| 范围 | fullstack-device |
| 变更文件数 | 19（含解析器、测试、v1 编辑体验和设计说明） |

## 变更概要

本次变更新增设备天线参数的运行时解析和只读 REST 接口，并由 GIS 地图选中设备后加载。前端增加 OpenLayers 方向线、固定 $120^\circ$ 覆盖扇面、设备弹窗扇区 Tab 以及方位角/机械下倾的异步任务编辑与本地预览；开发样例 SQL 保持本地辅助文件，不纳入提交。

## 审查发现

### 🔴 CRITICAL (严重)

无。

### 🟡 WARNING (警告)

无。

### 🔵 INFO (建议)

1. 新增 REST 接口尚未进入 [e2e_verify.sh](omcgo/scripts/e2e_verify.sh)。建议覆盖：合法设备、无权限设备、非法 UUID、设备不存在、无扇区参数和完整三扇区响应。
2. 前端真实 API 流程已通过浏览器冒烟验证，但没有自动化组件测试覆盖 [MapPopup.tsx](omcmb/webcode/src/components/GISMap/MapPopup.tsx) 的 Tab 切换、取消预览恢复和英文长标签布局。
3. Mock 模式下 [useTopology.ts](omcmb/frontend-core/src/hooks/api/useTopology.ts) 对扇区统一返回空数组，GIS mock 演示不会显示本功能；若 mock 是产品演示或前端回归入口，建议补充一台带三扇区的固定设备数据。

## 详细分析

### [antenna_sector.go](omcgo/internal/device/antenna_sector.go)

- 解析器支持未编号字段、字段后缀编号和 `CellConfig.<n>` 实例，并按扇区号稳定排序。
- 覆盖半径使用 $r = h / \tan(\theta)$，并在缺少必填字段或几何无效时拒绝生成扇面，方向/覆盖标志清晰。
- 非数值、`NaN` 与无穷大均会标记为缺失字段，不会生成 $0^\circ$ 方向或覆盖范围；对应解析测试已补齐。

### [device_info_handler.go](omcgo/internal/device/device_info_handler.go) 与 [device_service.go](omcgo/internal/device/device_service.go)

- `GET /devices/:id/antenna-sectors` 经过既有 `authorizeDeviceAccess`，无新增 IDOR 风险。
- Handler → Service → Repository 链路完整；参数查询使用 `param_group = 'antenna'` 的 Squirrel 参数化查询。
- 设备不存在返回 `404`，空参数组返回空数组，符合列表型只读接口语义。

### [deviceApi.ts](omcmb/frontend-core/src/services/api/deviceApi.ts) 与 [useTopology.ts](omcmb/frontend-core/src/hooks/api/useTopology.ts)

- 后端 snake_case DTO 经 `BackendAntennaSector` 显式映射为前端 camelCase 类型，避免 HTTP 封装仅解包信封造成字段丢失。
- 路径、设备 ID、响应字段和 React Query key `['devices', deviceID, 'antenna-sectors']` 与后端一致。
- Hook 位于共享 `frontend-core`，符合三皮肤业务层归属。

### [useOLMap.ts](omcmb/webcode/src/components/GISMap/useOLMap.ts) 与 [MapPopup.tsx](omcmb/webcode/src/components/GISMap/MapPopup.tsx)

- 扇区使用独立 VectorSource/Layer，在每次选中设备或数据变化时清空并重建，不会残留上一设备图形。
- `ol/sphere.offset` 的方位角传入弧度正确；覆盖扇面只在完整合法几何且缩放级别至少为 13 时绘制。
- 弹窗以 Ant Design Tabs 展示单个扇区，字段采用 Grid 分列；已通过中英文浏览器验证，英文长字段不再与数值重叠。

## 业务完整性检查

Handler、Service、Repository、路由注册、共享前端 API、React Query Hook、OpenLayers 图层和弹窗消费链路完整。无需数据库迁移：功能从既有 `device_parameters` 运行时读取。开发样例脚本可幂等更新已有参数记录。

## 业务影响范围检查

新增只读 API 和前端可选 GIS props，不改变既有设备详情、事件或共享数据库 schema。编辑复用既有参数写入任务链路，仅允许参数模型已支持的方位角和机械下倾进入任务。影响范围限定在设备参数分组 `antenna` 与 GIS 选中设备展示。

## 前后端一致性检查

- 路径一致：后端 `/devices/:id/antenna-sectors`，前端调用 `/devices/${id}/antenna-sectors`。
- 响应字段一致：`number`、覆盖/方向可用性、半径、缺失字段和 snake_case 参数经 mapper 完整转换。
- 无请求体、分页参数或新增错误码。

## 代码质量回退检查

未发现删除测试、移除错误处理、降级认证、引入 `any`、硬编码替代配置或 SQL 拼接等质量回退。

## 配套更新提醒

- **文档**：设计说明已存在；建议在正式 API 文档中登记新只读端点及字段含义。
- **单元测试**：非数值、非有限数值与字段源规范化测试已补充；仍建议补充新 REST 接口的 E2E 覆盖。
- **端到端测试**：建议为新端点补充认证、错误路径和完整响应断言。

## 安全检查

未发现新增认证、授权、注入或敏感数据暴露问题。接口复用设备组可见范围校验，数据库查询使用参数化 Squirrel 构建。

## 性能检查

单设备参数查询按设备 ID 和参数组过滤；地图层只保存当前选中设备的图形。未发现明显性能回归。需要在解析层拒绝无效数值，避免异常输入产生错误或过大的地理覆盖。

## 测试覆盖

- 已执行：`go build ./... && go test ./...` 通过。
- 已执行：`npm run typecheck` 通过，包含 `skin-parity` 与 v1/v2/v3 TypeScript 检查。
- 已执行：浏览器完成 GIS 扇区、Tab、固定 $120^\circ$ 扇面、编辑预览、取消恢复和参数布局冒烟。
- 缺口：接口 E2E、前端 Tab/不完整参数/取消恢复自动化测试。

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 0 |
| INFO | 3 |

**审查结论**: `PASS`