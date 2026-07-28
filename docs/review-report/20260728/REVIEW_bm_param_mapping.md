# BM 参数映射清理审查报告

## 结论

PASS_WITH_WARNINGS

## 审查范围

- `omcgo/data/param-mappings/BM.xml`
- `omcgo/data/param-mappings/BM_trpath_review.md`
- `omcgo/data/param-mappings/references/BM全量参数集.csv`

## 变更摘要

- 按 `BM全量参数集.csv` 的 `trpath.name` 口径清理 `BM.xml` 独有叶子参数 13 个。
- 保留仍有子路径的 `Device.Services.FAPService.{i}.FAPControl.LTE.X_COM_LICENSE.Capacity.{i}` 表节点。
- 按 BM CSV 更新 `RRCTimers` 10 个参数的 `access`、`type`、`min`、`max`、`defaultValue`。
- 将 `totalEntries` 从 `545` 调整为 `532`。
- 新增 BM 清理 review 文档和 BM 全量参数集基准 CSV。

## 发现

### CRITICAL

无。

### WARNING

- `go test ./...` 失败在既有 `BaiBNQ.xml` 断言：`TestBaiBNQNguBindInterfaceAndFallbackAreWritable` 期望 `Device.FAP.NguIpBind1.BindInterface` 存在。当前分支未修改 `BaiBNQ.xml`，且 `origin/main` 中同样未包含该路径，判断为本次 BM 改动之外的既有测试/数据不一致。

### INFO

- `BM全量参数集.csv` 为新增基准文件，随本次提交纳入版本库，保证 review 文档引用路径可追溯。
- CSV 未提供 `changeApplies` 字段，本次更新 `RRCTimers` 时保留 XML 原值。

## 验证

- `python3` XML/CSV 专项校验：通过。`BM.xml` 可解析，XML 独有叶子参数为 0，`RRCTimers` 全部匹配 BM CSV。
- `git diff --check`：通过。
- `go build ./...`：通过。
- `go test ./...`：失败，原因见 WARNING，失败点不属于本次 BM diff。

## 风险与影响

- 影响范围限定在 BM 参数映射数据、对应清理说明和 BM CSV 基准数据。
- 未涉及 Go 业务代码、接口、数据库迁移或前端页面。
