# MML 控制台命令选择按 Path 查询设计

## 背景

MML 控制台的“选择命令”弹框目前只按分组名称、命令名称和命令码筛选。标准命令已经通过 `/mml/group-tree` 返回 `target_paths`，前端适配为 `GroupTreeCommand.targetPaths`，因此用户可以在现有命令树数据上完成 Path 查询，无需新增后端接口或逐命令请求参数明细。

## 目标与范围

- 在命令选择弹框的现有搜索框中支持 Path 片段查询。
- 保留现有的命令名称、命令码、分组名称查询能力。
- 同时匹配标准命令的 `targetObject` 和 `targetPaths`。
- 自定义命令继续支持命令名称、命令码、分组名称查询，并增加其 `paramPaths` 查询。
- 搜索结果仍按原有“分组 → 命令”树展示；命中命令所在分组自动展开。
- 不新增 API、不修改命令选择后的参数加载和执行逻辑。

## 方案

在 `CommandSelectModal` 的现有 `useMemo` 筛选阶段扩展可搜索字段：

1. 标准命令将 `commandCode`、`displayName`、分组名称、`targetObject`、`targetPaths` 统一转为小写文本进行包含匹配。
2. 自定义命令将 `commandCode`、`commandName`、自定义分组名称、`paramPaths` 统一转为小写文本进行包含匹配。
3. 空关键字保持现有完整树；非空关键字只保留命中的命令，沿用现有树构建和自动展开行为。
4. `targetPaths` 或 `targetObject` 缺失时按空字符串处理，避免旧数据影响命令展示。

## 验证

- 在 `CommandSelectModal.test.tsx` 增加标准命令按 Path 筛选的回归测试。
- 运行命令选择弹框相关 Vitest 测试。
- 运行 `cd omcmb && npm run typecheck`。
- 部署前端到 113 自测环境，在 `/mml/console` 打开“选择命令”，输入一个已知标准 Path，确认只显示对应命令；清空搜索后确认命令树恢复。
