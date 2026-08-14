# 代码审查报告 — License 容量限制修复（issue #316）

- **日期**: 2026-08-14
- **基线**: 835a98124（main）
- **范围**: license per-type 容量 gating、在线口径、降容清理、容量满告警、Inform Ack 语义、前端移除"查看 License"
- **审查人**: Claude（/commit Step 5 自动审查）
- **结论**: **PASS_WITH_WARNINGS**（无 CRITICAL，4 条 WARNING 均为已知取舍，见下）

## 变更清单

| 模块 | 文件 | 变更 |
|---|---|---|
| license | enforcer.go | EnforceCapacity 改 per-type gating（deviceType 入参 + 大小写不敏感匹配 lookupCapacity/upperKey）；未授权类型/空类型严格拒绝；容量满时经 AlertSink raise warning 告警（1h 时间窗去重） |
| license | pg_device_counter.go | CountDevices/CountDevicesByType 改**在线口径**（is_online=true AND deleted_at IS NULL）；按 alarm_ne_type 分组 |
| license | monitor.go | 新增 CapacityExhaustedIdentifier="40"（OMC 告警库）；CheckCapacity 后评估 raise/clear 容量满告警（与 enforcer 互补，exhaustedAlertActive 标志防重复 Clear） |
| license | system_license_service.go | Update 降容策略：允许降容上传 + Replace 后调 CapacityOffliner 把超容在线设备置离线（按 created_at 晚接入优先踢）；清理失败仅告警不回滚 |
| device | device_service.go | LicenseEnforcer 接口签名更新；resolveNEType（productClass→alarm_ne_type，orphan/未配置严格拒绝）；EnforceOnlineCapacity 统一离线→在线校验；OfflineExcessDevices（置离线 + **清 Redis 缓存**防 stale is_online 绕过）；CreateDevice 移除容量校验（初始离线不占容量）；RegisterFromInform/UpdateFromInform 接入点更新 |
| device | inform_handler.go | 4 个注册点对 license 可恢复拒绝 **Ack**（返回 nil，不重试单条消息也不 Term——容量可恢复，下次 periodic Inform 重新尝试）；batch 路径在 prepareDeviceUpdate 后补容量校验（堵生产主路径漏洞） |
| device | device_repository.go | OfflineExcessByTypeCapacity：两条静态参数化 SQL（ROW_NUMBER 按 created_at 排序踢超出部分 + 未授权类型全踢），RETURNING serial_number 供清缓存 |
| provider | modules.go / device.go | enforcer 注入 AlertSink；systemLicenseSvc 注入 CapacityOffliner；DeviceService 注入 ExcessOffliner |
| 告警库 | data/alarm-definitions/OMC.xml | 新增 identifier=40 "License容量已满"（Warning，totalCount 28→29） |
| 前端 | webcode SystemLicense/index.tsx | 移除"查看 License"按钮 + raw_content Modal（issue 第 2 点） |
| 文档 | docs/test/license-capacity-feature-e2e-test-plan | §2.2 容量口径改 per-type 在线口径；§10 风险表勾销已修复项 |

## 审查发现

### CRITICAL（0 条）

无。

### WARNING（4 条，均为已确认取舍）

1. **EnforceCapacity 每次调用触发 COUNT+GROUP BY 查询**（enforcer.go:291）
   离线→在线转换每次 Inform 都查全表分组计数。当前验证环境无压力问题；100K 设备 + 高频 Inform 场景下建议后续给 CountDevicesByType 加短 TTL（如 5s）缓存。**接受理由**：容量校验只发生在离线→在线边沿（非每条 Inform），稳态在线设备不触发。
2. **降容清理非原子**（system_license_service.go Update）
   Replace（新 license 落地）与 OfflineExcessDevices（踢超容）不在同一事务；清理失败时新 license 已生效、设备暂超容。**接受理由**：清理失败仅告警，被踢设备的下次 Inform 会走 EnforceOnlineCapacity 兜底拦住（已实测验证该兜底有效：被踢设备连续 4 次 periodic Inform 被拒上线）。
3. **orphan 设备（productClass 未登记）被严格拒绝上线**（device_service.go resolveNEType）
   fail-closed 语义：ne_type 解析失败 → 拒绝上线/注册。**接受理由**：产品负责人明确选择严格模式（license 未授权类型 + 类型未知均拒绝）；测试 TestDeviceService_RegisterFromInform_LicenseCapacity_ProductNotRegistered 锁定该行为。
4. **EnforceCapacity 接口签名破坏性变更**
   EnforceCapacity(ctx, additional) → EnforceCapacity(ctx, deviceType, additional)，所有实现/调用方已同步更新（编译通过 + 全测试通过），无遗留调用方。

### INFO

- OfflineExcessByTypeCapacity 逐类型循环执行 SQL：类型数 ≤6（eNB/gNB/EPC/CPE/EGW/UPS），可接受。
- 告警 identifier=40 已确认全局未被占用（grep 全部告警 XML）；loader 实测加载 29 行入库。
- 容量满告警 raise/clear 分属 enforcer（实时，1h 去重）与 monitor（hourly 兜底 + clear），无共享状态，靠 alarm engine (DeviceSN, AlarmIdentifier) 幂等去重收敛——实测 alarms_active 仅一条。

## 验证记录

- `go build ./...` 通过；`go vet`（license/device/cmd）通过；gofmt 全部合规
- `go test ./internal/license/... ./internal/device/...` 全部通过
- 前端 `tsc --noEmit -p tsconfig.app.json`：SystemLicense 无错误（仅预存 exceljs 报错）
- 实测（docker 环境，license eNB=1）：
  - 管理面创建第 3 台 ENB → 403 biz_code=12116（per-type 拒绝）
  - 容量满告警入库 identifier=40 / severity=4 / cn_name=License容量已满
  - 降容重传 → 201 + 晚接入设备被踢离线（按 created_at）
  - 被踢设备连续 4 次 periodic Inform 被 batch 路径拦截（kept offline by license capacity）
  - 原在线设备停机后 ~10-11 分钟（阈值 600s + 扫描 60s）被判定离线，容量空出
