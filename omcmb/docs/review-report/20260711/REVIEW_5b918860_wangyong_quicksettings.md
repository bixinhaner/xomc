# Review: quicksettings NGU local address fallback

## 结论

PASS

## 范围

- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/CellParameterForm.tsx`
- `omcgo/data/param-mappings/BaiBNQ.xml`
- `omcgo/internal/config/parammodel/model_test.go`

## 检查项

- 前端运行时路径选择：通过设备实参判断是否支持 `Device.FAP.NguIpBind*`，不会仅因映射表存在而误走老路径。
- 前端表单保存：老路径模式保存接口路径，fallback 模式保存 IP 值，符合两类设备参数语义。
- 后端参数映射：`Device.FAP.NguIpBind1.BindInterface` 和 `Device.LAN_HostConfigManagement.IPInterface.NgapMgmt.NguLocalIpAddrList` 均为可写映射。
- 回归覆盖：新增 BaiBNQ 映射校验测试，覆盖主路径和 fallback 路径的可写校验。

## Findings

无 CRITICAL / WARNING。

## 验证

- `go test ./internal/config/parammodel ./internal/quicksettings ./internal/device` — 通过。
- `npm run typecheck`（`omcmb`）— 通过。
- 本地热重启后 `http://localhost:8081/` 返回 200。
- 欧版设备无 `Device.FAP.NguIpBind*` 实参时，下发 `Device.LAN_HostConfigManagement.IPInterface.NgapMgmt.NguLocalIpAddrList=172.19.3.81`，任务完成且设备返回 `SetParameterValuesResponse Status=0`。

## 风险

- 未在具备 `Device.FAP.NguIpBind*` 实参的真实设备上重新下发验证；当前逻辑保留原路径行为，并由映射测试覆盖本地校验。
