# 即插即用多小区参数配置审查报告

## 审查范围

- 即插即用公共参数与指定设备参数的多小区/BTS 列表编辑
- NR、LTE、GSM BM、GSM BSC Excel 模板、导入、导出与显式实例索引校验
- 策略参数物化、TR-069 实例路径解析与 XML 生成
- 中英文文案、设计文档及相关前后端测试

明确排除本地临时设备列表修复：

- `omcgo/internal/device/control_summary.go`
- `omcgo/internal/device/control_summary_test.go`

## 结论

PASS

未发现 CRITICAL、WARNING 或 INFO 级代码问题。

## 审查摘要

- 主实例控制列只用于选择小区/BTS 实例，不进入设备下发参数。
- 多行主实例必须提供正整数索引，并拒绝索引缺失、重复或与产品类型不匹配的文件。
- 子实例索引与主实例索引独立解析，覆盖 PLMN、邻区、载波、TRX、接口和隧道对象。
- LTE 使用小区编号选择 `FAPService` 实例；NR 使用 `CellConfig` 实例；GSM BM 与 BSC 分别使用 `GsmBTSCellDT` 和 `DeviceGSM.Bts` 实例。
- 公共配置和设备配置共用列表编辑器，支持新增、复制、删除、展开编辑，并保留按实例索引合并的导入数据。
- 未新增接口、数据库迁移、权限逻辑、字符串拼接 SQL、裸 `panic` 或不安全 HTML 渲染。

## 验证

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./internal/provision -count=1`：通过。
- `cd omcgo && go test ./... -count=1`：除既有 `internal/product` 的 BLN XML 元数据数量不一致（期望 380、实际 329）外，其余包通过；本次 `internal/provision`、E2E 与 integration 均通过。
- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb/webcode && npm test -- --run src/pages/device/PlugAndPlay`：27 个测试文件、138 个用例通过。
- `cd omcmb/webcode && npm run build`：通过，仅有既有 Vite 废弃配置与大 chunk 警告。
- `git diff --cached --check`：通过。
- 本地真实链路验证覆盖 Excel 文件导入、策略触发、XML 详情/下载一致性与下发任务创建；NR、LTE、GSM BM、GSM BSC 生成结果均与预期实例路径一致，XML 中无残留 `{i}` 或实例控制字段。

## 已知非本次阻塞项

- 仓库全量 Go 测试仍受既有 BLN 参数模型元数据数量不一致影响；该文件与本次 staged diff 无交集。
- LTE 与 GSM BM 的一次真实下发任务分别因模拟设备离线而过期/设备侧失败，但服务端生成 XML 已完成逐字节比对；NR 与 GSM BSC 任务完成。
