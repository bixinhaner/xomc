# 参数映射清理提交审查报告

## 结论

PASS

未发现阻塞提交的 CRITICAL 问题。

## 审查范围

- `omcgo/data/param-mappings/BaiBNQ.xml`
- `omcgo/data/param-mappings/BaiBNQ_trpath_review.md`
- `omcgo/data/param-mappings/param_mapping_cleanup_process.md`
- `omcgo/data/param-mappings/references/NR全量参数集.csv`

## 主要变更

- 基于 NR 全量参数集清理 `BaiBNQ.xml` 中 XML 独有且无子路径的叶子参数。
- 保留 XML 独有但有子路径的对象/表节点。
- 按 CSV 补齐并更新 `RRCTimers` 参数，路径按通用多实例泛化。
- 将 NR 全量参数集纳入项目目录 `param-mappings/references/`。
- 新增 BaiBNQ 清理 review 文档和后续站型复用流程文档。

## 验证

- XML 解析：通过。
- CSV 字段校验：`trpath.name` 存在。
- RRCTimers 校验：CSV 中 11 个 RRCTimers 参数按归一化规则全部匹配 XML。
- 独有叶子参数校验：剩余 XML 不匹配项均为有子路径的对象/表节点，独有叶子参数数量为 0。
- `FAPService.1.` 残留校验：BaiBNQ XML 中无残留。

## 风险与影响

- 影响范围限定在参数映射数据与流程文档，不涉及 Go/前端运行时代码。
- `NR全量参数集.csv` 为新增基准数据文件，后续站型清理依赖该路径。
- 未运行全量 Go 构建；本次为 XML/CSV 数据清理，已执行针对性结构和一致性校验。
