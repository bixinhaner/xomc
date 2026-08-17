# MML 参数模型数据类型校准审查报告

- 日期：2026-08-17
- 分支：`fix/mml-param-types-from-references`
- 基线：`origin/main`
- 关联 Issue：`#314`
- 结论：`PASS_WITH_WARNINGS`

## 变更摘要

依据 BM、GSM、LTE、NR 四份设备全量参数集的 `trpath.datatype`，按归一化后的 TR-069 路径校准 BM、BSC、BTS、BLN、BLQ、MLN、MLQ、BaiBNQ 共 8 个参数模型及其交付副本，共修正 1,347 个类型声明。新增公共 TR-069 数据类型到 SOAP XSD 类型映射，并在 ACS 标准路径转私有路径时使用具体产品模型覆盖下发类型和布尔值表示。

## 审查范围

- 参数模型：`omcgo/data/param-mappings/*.xml`
- 交付模型：`omcgo/docs/param-model-delivery/xml/tr069-param-mapping/*.xml`
- MML 下发：`omcgo/internal/mml/tr069_payload.go`
- ACS 路径翻译：`omcgo/internal/acs/path_translator.go`
- 公共类型映射：`omcgo/pkg/tr069/value_type.go`
- 一致性与回归测试：对应 `_test.go` 文件

## 审查结果

### CRITICAL

无。

### WARNING

1. 私有路径模式不经过标准路径到产品模型的 ACS 类型增强；本次保证常规标准路径 MML 下发链路，私有路径调用方仍需自行提供正确类型。
2. `U_LONG` 已支持 SOAP `xsd:unsignedLong` 下发，但前端录入和部分回读展示仍沿用既有通用字符串语义，未在本次后端模型校准范围内扩展专用控件。
3. 全量 `go test ./...` 存在两个与本次改动无关的基线失败：GSM 告警定义期望 442、实际 443；BLN 参数模型头部 `totalEntries=329`、实际条目 380。本次 XML 变更未增删参数条目。

### INFO

1. `ENB_DEFAULT_098.xml` 和 `ENB_DEFAULT_181.xml` 未修改，因为四份百佳设备参考集未覆盖这两个模型。
2. 参考 CSV 中无法匹配现有模型路径的条目不会改写模型；新增一致性测试持续校验所有已匹配路径。
3. 参数模型与交付副本同步修改，避免运行时模型和交付材料漂移。

## 验证记录

- `go build ./...`：通过。
- `go test ./pkg/tr069 ./internal/mml ./internal/acs ./internal/config/parammodel`：通过。
- TR-069 类型、MML payload、ACS 翻译及参考集一致性定向测试：通过。
- `git diff --check`：通过。
- 前端 `npm run build`：通过。
- Docker Compose web 栈重新构建并热重启：通过，页面 HTTP 200，核心容器运行正常。
- 真实在线设备 MML 只读自测：任务 `bf8fbb33-fcb0-465c-9119-e7b1b877f779` 完成，成功 1、失败 0；返回字段类型与 BaiBNQ 产品模型一致。

## 结论

改动满足本次“依据设备参考集校准参数模型并确保常规 MML 按产品模型类型下发”的范围，无阻断提交的问题。上述 WARNING 为已知边界和既有基线问题，建议后续独立跟踪。
