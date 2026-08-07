# F08 北向总体重设计（历史稿）

> 版本：2026-07-30
> 状态：已废弃，仅保留历史入口
> 当前推荐设计：`page-config-redesign-20260731.md`
> 数据支持附件：`data-support-matrix-20260731.md`

本稿早期尝试把旧 XML/Profile 方案、文件北向、Inventory、Socket、SNMP 和 API 统一进一个总体方案。后续评审结论已经调整为：

1. 北向运行配置以页面为唯一入口。
2. XML 和旧场景文件只作为内置模板种子，不作为运行配置。
3. 页面、模板、调度任务和 adapter 只覆盖当前 xomc 已有数据源。
4. 文件投递、Socket、SNMP、REST API 统一纳入能力开关、目标配置、运行记录和审计。

后续讨论和实现均以 `page-config-redesign-20260731.md` 为准；本文件不再维护旧范围细节。
