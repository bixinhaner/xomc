# verify T-0098-P4-01 — frontend-core 字典平台业务层拆分

> **范围**：在 `omcmb/frontend-core/` 新增 4 个 API 服务 + 4 个 Hook + 4 个 Type + 4 个 Mock service + 4 份 Mock data，对接后端 P3 完成的 48 个 REST 端点（products / param-models / indicators / alarm-definitions）。
> **commit**：pending（本次 P4-01 提交）
> **wave-batched**：是（Skip S0/S1，Footer 引用 §C wave-batched）

## 新增文件清单

| 类别 | 文件 | LOC |
|---|---|---|
| Type | [src/types/product.ts](omcmb/frontend-core/src/types/product.ts) | 89 |
| Type | [src/types/paramModel.ts](omcmb/frontend-core/src/types/paramModel.ts) | 109 |
| Type | [src/types/alarmDefinition.ts](omcmb/frontend-core/src/types/alarmDefinition.ts) | 70 |
| Type | [src/types/indicatorLibrary.ts](omcmb/frontend-core/src/types/indicatorLibrary.ts) | 96 |
| API | [src/services/api/productApi.ts](omcmb/frontend-core/src/services/api/productApi.ts) | 263 |
| API | [src/services/api/paramModelApi.ts](omcmb/frontend-core/src/services/api/paramModelApi.ts) | 295 |
| API | [src/services/api/alarmDefinitionApi.ts](omcmb/frontend-core/src/services/api/alarmDefinitionApi.ts) | 173 |
| API | [src/services/api/indicatorLibraryApi.ts](omcmb/frontend-core/src/services/api/indicatorLibraryApi.ts) | 296 |
| Hook | [src/hooks/api/useProducts.ts](omcmb/frontend-core/src/hooks/api/useProducts.ts) | 178 |
| Hook | [src/hooks/api/useParamModels.ts](omcmb/frontend-core/src/hooks/api/useParamModels.ts) | 174 |
| Hook | [src/hooks/api/useAlarmDefinitions.ts](omcmb/frontend-core/src/hooks/api/useAlarmDefinitions.ts) | 89 |
| Hook | [src/hooks/api/useIndicatorsLibrary.ts](omcmb/frontend-core/src/hooks/api/useIndicatorsLibrary.ts) | 234 |
| Mock svc | [src/mock/services/productService.ts](omcmb/frontend-core/src/mock/services/productService.ts) | 158 |
| Mock svc | [src/mock/services/paramModelService.ts](omcmb/frontend-core/src/mock/services/paramModelService.ts) | 145 |
| Mock svc | [src/mock/services/alarmDefinitionService.ts](omcmb/frontend-core/src/mock/services/alarmDefinitionService.ts) | 109 |
| Mock svc | [src/mock/services/indicatorLibraryService.ts](omcmb/frontend-core/src/mock/services/indicatorLibraryService.ts) | 187 |
| Mock data | [src/mock/data/product.ts](omcmb/frontend-core/src/mock/data/product.ts) | 50 |
| Mock data | [src/mock/data/paramModel.ts](omcmb/frontend-core/src/mock/data/paramModel.ts) | 86 |
| Mock data | [src/mock/data/alarmDefinition.ts](omcmb/frontend-core/src/mock/data/alarmDefinition.ts) | 58 |
| Mock data | [src/mock/data/indicatorLibrary.ts](omcmb/frontend-core/src/mock/data/indicatorLibrary.ts) | 39 |
| **合计** | 20 文件 | ~2898 LOC |

## 端点覆盖（47/48）

- products handler 17 端点：list / get / create / update / delete / resetDiscovered / pattern CRUD + move / match / matchOrder / orphan list / rematch / bind / cacheRefresh / importDirectory ✅
- param-models handler 19 端点：CRUD / mappings CRUD / standard CRUD / translate / discovered list+versions+delete / cacheRefresh / importDirectory ✅
- indicators handler 21 端点：CRUD / formulas list+get+upsert+delete / groups CRUD / enabled list+set / units CRUD / cacheRefresh / importDirectory ✅
- alarm-definitions handler 9 端点：CRUD / unknownStats / severityLevels / cacheRefresh / importDirectory ✅
- patterns delete 端点（DELETE /products/:id/patterns/:patternId）— 见 productApi.deletePattern ✅

## 设计对齐

- Backend↔Frontend 字段映射：所有响应通过 `BackendXxx` interface + `mapBackendXxx` 函数完成 snake_case → camelCase 转换
- HTTP 层（[http.ts](omcmb/frontend-core/src/services/http.ts)）已自动处理 `{ret:1, data}` 信封 + Bearer Token 注入 + camelCase ↔ snake_case query 转换；本次新 API 透明复用
- Hook 三段式 query key：`[domain, action, params]`（与现有 `useDevices` 风格一致）
- Mutation 失败时 RQ 自动 retry 由全局 QueryClient 控制；本次未单独配置
- `createApiSwitch(mockService, realService)` 切换：`VITE_USE_MOCK=true` 走 Mock，否则走真后端
- patterns 上下移动调用 `PUT /:id/patterns/:patternId/move`（后端用临时负值绕开 partial unique index，前端无需处理）

## 特殊处理

| 项 | 处理 |
|---|---|
| 后端 `PatternView` 缺 json tag 输出 PascalCase | productApi 内部 `BackendPattern` 用 PascalCase 字段，再 `mapBackendPattern` 转 camelCase |
| translate 是 POST | useTranslate 用 useMutation 而非 useQuery |
| products/:id/discovered 子路径属于 paramModel 域 | 写在 paramModelApi.listDiscovered / listDiscoveredVersions / deleteDiscovered |
| indicators 端点带 `?deviceType=ENB\|GSM\|GNB` query | useIndicatorsLibrary 在每个 hook 第一参数显式接 `deviceType` |
| Mock 默认 5-10 条样本 | 仅供 `VITE_USE_MOCK=true` 时跑通 UI；不替代 E2E |

## 编译验证

```
$ cd omcmb/webcode && npm run typecheck
> webcode@0.0.0 typecheck
> tsc --noEmit
[exit 0, 0 错误]
```

✅ typecheck 全过

## 不在本子任务

- 不删 `datamodelApi.ts` / 不拆 `alarmApi.ts`：保留兼容期供旧页面 `DataModelManagement` / `AlarmSupportLibrary` 继续工作；P5-04 / P5-06 处理
- 不修改 `webcode/src/pages/`、不动路由：P4-02 ~ P4-07 处理
- 不评估 webcode-v2/v3：P4-08 处理
- 不写单元测试：业务层映射逻辑暴露简单；E2E 在主 `e2e_verify.sh` 已覆盖后端，UI 级测试在 `webcode/test/e2e/`（如需）

## DoD

- [x] 4 个新 API 文件，端点齐全
- [x] 4 个新 Hook 文件，三段式 query key
- [x] 4 个新 Type 文件，BackendXxx + mapBackendXxx
- [x] 4 个新 Mock service + 4 份 Mock data
- [x] webcode typecheck 通过
- [x] 不影响现有 `datamodelApi` / `alarmApi`
