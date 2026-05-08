# verify T-0098-P4-05 — KPI 指标库 5 Tabs（吸收 T-0095）

> **范围**：实现 [/product/kpi-library](omcmb/webcode/src/pages/product/kpi-library) 页 — 5 Tabs (ENB / GSM / GNB / 启用 / 单位) + 详情抽屉全平台公式 CRUD。
> **wave-batched**：是（Skip S0/S1）
> **D10=A 吸收 T-0095**：T-0095 KPI 标准库管理在本子任务实现，不再独立工作

## 新增文件

| 文件 | LOC | 职责 |
|---|---|---|
| [src/pages/product/kpi-library/index.tsx](omcmb/webcode/src/pages/product/kpi-library/index.tsx) | 67 | 5 Tabs 容器 + XML 导入/缓存刷新 |
| [src/pages/product/kpi-library/IndicatorTab.tsx](omcmb/webcode/src/pages/product/kpi-library/IndicatorTab.tsx) | 95 | ENB/GSM/GNB 共用，按 deviceType 加载（GNB 隐藏 indicator_level 列） |
| [src/pages/product/kpi-library/IndicatorDrawer.tsx](omcmb/webcode/src/pages/product/kpi-library/IndicatorDrawer.tsx) | 196 | 详情抽屉 + 全平台公式 CRUD |
| [src/pages/product/kpi-library/EnabledIndicatorsTab.tsx](omcmb/webcode/src/pages/product/kpi-library/EnabledIndicatorsTab.tsx) | 123 | 启用指标按 deviceType + operatorCode 切换 |
| [src/pages/product/kpi-library/IndicatorUnitsTab.tsx](omcmb/webcode/src/pages/product/kpi-library/IndicatorUnitsTab.tsx) | 130 | 单位 CRUD |

## 5 Tabs 功能

| Tab | 内容 |
|---|---|
| ENB | 共享 IndicatorTab(deviceType='ENB')，列：id / cnName / enName / 分组 / 计数器类型 / **级别** / 单位 / 启用 / 操作（详情 / 删除）|
| GSM | 共享 IndicatorTab(deviceType='GSM') |
| GNB | 共享 IndicatorTab(deviceType='GNB')，**隐藏 indicator_level 列**（schema 不同）|
| 启用指标 | 按 deviceType + operatorCode (cmcc/ctcc/cucc/default) 切换；表格 rowSelection 选中后调 setEnabled |
| 单位定义 | 单位 CRUD（id / enName / cnName）|

## 详情抽屉（IndicatorDrawer）

- 顶部 Descriptions 显示指标元数据（10 字段，bordered）
- 中部 "全平台公式" 表格 — 列：platform_name (Tag) / formula (code) / 操作
- 新增 / 编辑公式 Modal：platform 字段（编辑时禁用主键）+ formula TextArea
- **公式校验**：前端基础括号匹配（"(" depth ≤ 0 或末尾不为 0 → 报错"括号不匹配"）；后端二次完整语法验证
- 删除公式：Popconfirm

## API 调用映射

| UI 操作 | Hook | 后端端点 |
|---|---|---|
| 列表加载 | useIndicatorList | GET /indicators?deviceType= |
| 详情公式 | useFormulas | GET /indicators/:id/formulas?deviceType= |
| 新增/更新公式 | useUpsertFormula | POST /indicators/:id/formulas?deviceType= |
| 删除公式 | useDeleteFormula | DELETE /indicators/:id/formulas/:platform?deviceType= |
| 删除指标 | useDeleteIndicator | DELETE /indicators/:id?deviceType= |
| 启用指标列表 | useEnabledIndicators | GET /enabled-indicators?deviceType=&operatorCode= |
| 启用/停用 | useSetEnabledIndicators | PUT /enabled-indicators?... |
| 单位列表 | useIndicatorUnits | GET /indicator-units |
| Upsert 单位 | useUpsertUnit | POST /indicator-units |
| 更新单位 | useUpdateUnit | PUT /indicator-units/:id |
| 删除单位 | useDeleteUnit | DELETE /indicator-units/:id |
| XML 导入 | useIndicatorImportDirectory | POST /indicators/import-directory |
| 刷新缓存 | useIndicatorCacheRefresh | POST /indicators/cache/refresh |

## D10=A T-0095 吸收

- T-0095 "KPI 标准库管理" 原计划独立实现 KPI 治理 UI；本子任务的 5 Tabs + 详情抽屉公式 CRUD 已覆盖该范围
- T-0095 在 backlog 主表标记 done，关联 `T-0098-P4-05` 收尾时一并关闭

## 编译验证

```
$ cd omcmb/webcode && npm run typecheck
> webcode@0.0.0 typecheck
> tsc --noEmit
[exit 0, 0 错误]
```

✅ typecheck 全过

## 不在本子任务

- 公式可视化编辑器（拖拽 token）：当前仅 TextArea；后续 V2 优化
- 历史版本对比：当前仅当前版本；P5 阶段考虑

## DoD

- [x] 5 Tabs 容器
- [x] ENB/GSM/GNB 共用 IndicatorTab
- [x] GNB 隐藏 indicator_level 列
- [x] 启用指标按 operatorCode 维度切换
- [x] 单位 CRUD
- [x] 详情抽屉 + 全平台公式 CRUD（前端括号校验）
- [x] webcode typecheck 通过
- [x] D10=A T-0095 吸收（本子任务完成即 T-0095 done）
