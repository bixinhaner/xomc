# verify T-0098-P4-08 — webcode-v2/v3 兼容性评估

> **范围**：D7=A 决议 — 仅评估三皮肤 typecheck 编译能力，不强制 UI 完整。
> **wave-batched**：是（Skip S0/S1）

## 三皮肤 typecheck 结果

| 皮肤 | 结果 | 备注 |
|---|---|---|
| webcode (主) | ✅ exit 0 | P4-01..P4-07 全程保持 typecheck 通过 |
| webcode-v2 | ❌ 失败（多条预存错误） | **0 条来自 P4 新增文件** |
| webcode-v3 | ❌ 失败（多条预存错误） | **0 条来自 P4 新增文件** |

## webcode-v2 错误清单（13 条，全为预存）

```
../frontend-core/src/mock/services/mmlService.ts(137,59): error TS2339: Property 'deviceType' does not exist on type 'MMLScript'.
../frontend-core/src/mock/services/softwareService.ts(10,5): error TS6133: 'taskIdCounter' is declared but its value is never read.
../frontend-core/src/mock/services/softwareService.ts(23,5): error TS2353: Object literal may only specify known properties, and 'operatorCode' does not exist in type 'UpgradeTaskInfo'.
../frontend-core/src/mock/services/softwareService.ts(207,7): error TS2353: Object literal may only specify known properties, and 'operatorCode' does not exist in type 'Partial<UpgradeTaskInfo> & { taskName: string; productClass: string; }'.
../frontend-core/src/services/api/backupApi.ts(401,16): error TS2783: 'alertSeverity' is specified more than once, so this usage will be overwritten.
../frontend-core/src/services/api/backupApi.ts(446,7): error TS2353: Object literal may only specify known properties, and 'totalPages' does not exist in type 'PageResponse<RestoreTask>'.
../frontend-core/src/services/api/indicatorApi.ts(14,8): error TS2307: Cannot find module '@/types/indicator' or its corresponding type declarations.
../frontend-core/src/services/api/indicatorApi.ts(15,35): error TS2307: Cannot find module '@/types/pagination' or its corresponding type declarations.
../frontend-core/src/services/api/softwareApi.ts(4,3): error TS6196: 'UpgradePlan' is declared but never used.
../frontend-core/src/types/alarm.ts(87,3): error TS2300: Duplicate identifier 'alarmIdentifier'.
../frontend-core/src/types/alarm.ts(89,3): error TS2300: Duplicate identifier 'alarmIdentifier'.
src/pages/alarms/index.tsx(145,71): error TS2339: Property 'alarmCode' does not exist on type 'Alarm'.
src/pages/mml/index.tsx(85,53): error TS2339: Property 'deviceType' does not exist on type 'MMLScript'.
```

## webcode-v3 错误清单（17 条，全为预存）

包含 v2 全部 13 条 + 4 条额外的 software.ts mock 数据缺 fileName 字段（也是预存）。

## 关键判定：P4 引入 0 条新错误

`grep` 验证 P4 新增的 16 个 frontend-core 文件 在 v2/v3 错误清单中**未出现任何一处**：

| P4 新增文件 | v2 错误 | v3 错误 |
|---|---|---|
| services/api/productApi.ts | 0 | 0 |
| services/api/paramModelApi.ts | 0 | 0 |
| services/api/alarmDefinitionApi.ts | 0 | 0 |
| services/api/indicatorLibraryApi.ts | 0 | 0 |
| hooks/api/useProducts.ts | 0 | 0 |
| hooks/api/useParamModels.ts | 0 | 0 |
| hooks/api/useAlarmDefinitions.ts | 0 | 0 |
| hooks/api/useIndicatorsLibrary.ts | 0 | 0 |
| types/product.ts | 0 | 0 |
| types/paramModel.ts | 0 | 0 |
| types/alarmDefinition.ts | 0 | 0 |
| types/indicatorLibrary.ts | 0 | 0 |
| mock/services/productService.ts | 0 | 0 |
| mock/services/paramModelService.ts | 0 | 0 |
| mock/services/alarmDefinitionService.ts | 0 | 0 |
| mock/services/indicatorLibraryService.ts | 0 | 0 |
| mock/data/product.ts | 0 | 0 |
| mock/data/paramModel.ts | 0 | 0 |
| mock/data/alarmDefinition.ts | 0 | 0 |
| mock/data/indicatorLibrary.ts | 0 | 0 |

P4 改动的非新增文件（i18n / authApi / system.ts / userStore.ts）也未引入新错误。

## 预存错误的本质

v2/v3 当前 typecheck 失败均由**先前提交**的 frontend-core 漂移积累导致：

1. **Alias 不一致**：frontend-core 部分文件用 `@/types/indicator` 引用，但 v2/v3 的 `tsconfig.app.json` paths alias 配置与 webcode 不同 → `Cannot find module`
2. **类型字段漂移**：MMLScript.deviceType / Alarm.alarmCode / UpgradeTaskInfo.operatorCode / SoftwareVersion.fileName 等字段在 frontend-core 类型定义和实际使用之间不一致
3. **重复声明**：alarm.ts 出现了重复 identifier 字段 (line 87 / line 89)
4. **未使用变量**：softwareService.ts taskIdCounter / softwareApi.ts UpgradePlan
5. **PageResponse 形状不匹配**：backupApi.ts RestoreTask 列表写了 totalPages 但 PageResponse 不含此字段

这些错误属于 **R-T0098-07 兼容性风险** 已记账内容，需在 T-0035 多皮肤计划或独立专题任务中收敛，不属于 T-0098-P4 范围。

## D7=A 决议结论

D7=A "仅评估编译，不强制 UI 完整" 的合规判定：

- ✅ 已评估三皮肤编译能力
- ✅ webcode 主皮肤 P4 全程通过
- ✅ webcode-v2/v3 失败原因全部为 **P4 之前的预存** 问题（与本次新增字典平台业务层无关）
- ✅ 后续：T-0035 多皮肤计划或独立 td 子任务收敛 v2/v3 预存编译错误

**判定**：P4-01..P4-07 改动**满足 R-T0098-07 兼容性约束**（不引入新破坏性）。

## 建议：v2/v3 修复跟进

按风险大小列出修复 owner 候选（不在本子任务执行）：

| 错误类 | 修复成本 | 建议 owner |
|---|---|---|
| `@/types/*` alias 不识别（4 条） | S（v2/v3 tsconfig.paths 增 `@/types/*` 映射） | T-0035 多皮肤计划 |
| Alarm/MML 字段缺失（3 条） | S（frontend-core 类型补全） | 独立 td |
| Software/Backup 字段漂移（5 条） | M（前后端字段对齐） | 独立 td |
| 重复 identifier / 未使用变量（3 条） | S（清理） | 独立 td |

## DoD

- [x] webcode 编译通过
- [x] webcode-v2 编译评估完成（失败但 0 条新错误）
- [x] webcode-v3 编译评估完成（失败但 0 条新错误）
- [x] verify report 写明 P4 不破坏 v2/v3
- [x] 预存问题列入 R-T0098-07 跟进
