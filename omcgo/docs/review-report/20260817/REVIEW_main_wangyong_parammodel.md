# 多实例参数模型与 MML 命令修复审查报告

- 日期：2026-08-17
- 分支：`fix/dynamic-param-object-mappings`
- 基线：`origin/main`
- 关联 Issue：无（现场参数模型修复）
- 结论：`PASS_WITH_WARNINGS`

## 变更摘要

依据 BM、LTE、NR 设备全量参数集，将 BLN、BLQ、BM、BaiBNQ、ENB_DEFAULT_098、ENB_DEFAULT_181、MLN、MLQ 参数模型中的可写多实例对象统一改为 `{i}` 实例对象，并同步运行时模型与交付副本。BaiBNQ 的 SIB 参数取消固定实例 `1`，补充邻区、路由、地址、QoS、AMF 池和 Xn 地址映射对象。

同时修复接口绑定查询命令在全新数据库中未创建的问题，将 AMF 状态从 LTE 接入分组迁移到 NR 小区状态命令，并增加种子数据和设备技术制式过滤回归测试。

## 审查范围

- 运行时参数模型：`omcgo/data/param-mappings/*.xml`
- 参数模型交付副本：`omcgo/docs/param-model-delivery/xml/tr069-param-mapping/*.xml`
- 参数模型一致性测试：`omcgo/internal/config/parammodel/*_test.go`
- MML 命令过滤与种子测试：`omcgo/internal/mml/*_test.go`
- 初始种子：`omcgo/migrations/seed/000001_init_seed.sql`

## 审查结果

### CRITICAL

无。

### WARNING

1. 全量 `go test ./...` 存在一个与本次改动无关的既有失败：`TestBuiltinAlarmLibraries_GSMDefinitionsAreOwnedByGSM` 期望告警定义总数 442，实际为 443；失败文件不在本次改动范围内。

### INFO

1. 新增参考集一致性测试，校验设备 CSV 声明的动态对象在运行时模型和交付模型中均存在且为 `READ_WRITE`。
2. 接口绑定 LST 命令改为 `INSERT ... ON CONFLICT DO UPDATE`，兼容全新部署和重复执行。
3. AMF 状态只归属 NR 小区状态查询，5G 产品不再显示 LTE 接入分组中的 AMF 状态。
4. 参数模型头部 `totalEntries` 已随对象条目调整，并由现有模型解析测试校验。

## 验证记录

- `go build ./...`：通过。
- `go test -count=1 ./internal/config/parammodel ./internal/mml ./internal/product`：通过。
- `go test ./...`：除上述既有告警数量断言外，其余包通过；E2E 与 integration 包通过。
- `git diff --check --staged`：通过。
- `npm run build`：通过。
- Docker Compose app、ACS、worker、web 重建并重启：通过；`http://localhost:8081/` 返回 HTTP 200，参数模型加载 855 条映射。

## 结论

本次改动与设备侧多实例数据模型一致，运行时模型和交付副本同步，MML 分组与新库种子行为有回归覆盖，无阻断提交的问题。
