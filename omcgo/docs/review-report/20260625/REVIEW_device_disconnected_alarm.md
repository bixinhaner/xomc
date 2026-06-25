# 设备断连告警生命周期修复审查报告

- Issue: #647
- Scope: device
- Date: 2026-06-25
- Verdict: PASS

## 审查范围

- `cmd/app/provider/device.go`
- `cmd/app/provider/router.go`
- `internal/device/status_reconciler.go`
- `internal/device/status_reconciler_test.go`
- `internal/device/device_service.go`
- `internal/device/service_test.go`

## 变更摘要

- 在设备 online -> offline 的真实状态翻转后，按制式生成 OMC 源设备断连告警：LTE/eNB=7、NR/gNB=23、GSM=4。
- 在设备 offline -> active 的 Inform 恢复在线路径中，清除同设备、同告警码、OMC 源的活跃断连告警。
- 将 alarm 模块加入 device 模块初始化依赖，并注入 `AlarmEngine` / `AlarmPgStore`。
- 增加单元测试覆盖告警码映射、幂等离线路径、告警失败容错、恢复在线清理以及 active -> active 不误清。

## 审查结论

### CRITICAL

无。

### WARNING

无。

### INFO

- 告警写入和清除失败均记录 warning 后返回，不阻断设备离线翻转或 Inform 主流程，符合设备状态主路径优先的实现取向。
- 恢复在线清理限定 OMC 源断连告警，避免误清设备自身上报的同码非 OMC 告警。

## 验证

```bash
go test ./internal/device -run 'TestMarkOffline|TestDetect_MarksStaleDevicesOffline|TestUpdateFromInform_OfflineToActive|TestUpdateFromInform_ActiveStaysActive'
go test ./cmd/app/provider
```

结果：通过。

备注：`go test ./internal/device` 全包存在既有参数树用例失败 `TestParameterTreeHandler_UsesDefaultParamModelForTreeAndChildren/schema_only_includes_actual_parameters`，与本次断连告警生命周期修改不在同一路径。