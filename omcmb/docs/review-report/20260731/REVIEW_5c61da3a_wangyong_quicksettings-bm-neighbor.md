# BM 邻区新增参数补齐审查报告

## 结论

**PASS**

未发现 CRITICAL、WARNING 级问题。

## 审查范围

- `omcgo/data/param-mappings/BM.xml`
- `omcgo/data/quicksettings/BM.xml`
- `omcgo/internal/config/parammodel/model_test.go`
- `omcgo/internal/quicksettings/loader_test.go`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/MultiInstanceTable.tsx`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/neighborCellAdd.test.tsx`

## 关键检查

- BM 邻区 eNodeB 类型使用设备真实叶子 `NeighCellTypeContainer`，未复用 BLQ 私有叶子。
- `X2Flag` 映射与 BM 全量参数资料一致，为可写 `U_INT`，取值限制为 `0/1`。
- eNB ID 与 Cell ID 的拆分仅依赖 LTE ECI/CID，组合后仍写入原有 `CID`，未改变下发接口。
- 新增字段均通过 quicksettings 元数据提供类型、范围、必填和默认值，没有新增硬编码用户文案。
- 未涉及 SQL、迁移、运营商分支、认证权限或公共 API 契约。
- 回归测试覆盖 BM 厂商叶子映射、完整新增表单、数值约束和 X2 参数模型加载。

## 验证

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./...`：通过，包含 E2E 与集成测试。
- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb/webcode && npm test -- --run src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/neighborCellAdd.test.tsx`：4 个用例通过。
- `cd omcmb/webcode && npm run build`：生产构建通过。
- 本地 Docker Compose 已重建 app/web/acs/worker，`http://127.0.0.1:8081/` 返回 200。
- 真实 BM 设备 `120288069823C4B0060` 浏览器验证：新增邻区弹窗显示 9 项参数，QOffset/CIO 默认 `0`，eNodeB Type 默认 `Home`，X2 Flag 默认 `SON`。

## 风险与回滚

- 影响范围仅为 LTE BM 邻区新增表单和 BM 参数模型；其他型号继续按各自 quicksettings 元数据展示。
- 如需回滚，可整体回退本提交，不涉及数据库迁移或数据修复。
