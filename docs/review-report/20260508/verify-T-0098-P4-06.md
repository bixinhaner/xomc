# verify T-0098-P4-06 — 告警库管理（单页 + 详情抽屉 + 未识别频次）

> **范围**：实现 [/product/alarm-library](omcmb/webcode/src/pages/product/alarm-library) 页 — 列表 + 详情抽屉 + 未识别频次 Modal。
> **wave-batched**：是（Skip S0/S1）

## 新增文件

| 文件 | LOC | 职责 |
|---|---|---|
| [src/pages/product/alarm-library/index.tsx](omcmb/webcode/src/pages/product/alarm-library/index.tsx) | 198 | 列表页 + 工具栏 + 4 个过滤器 |
| [src/pages/product/alarm-library/AlarmDefinitionDrawer.tsx](omcmb/webcode/src/pages/product/alarm-library/AlarmDefinitionDrawer.tsx) | 145 | 详情抽屉 + CRUD 表单 |
| [src/pages/product/alarm-library/UnknownStatsModal.tsx](omcmb/webcode/src/pages/product/alarm-library/UnknownStatsModal.tsx) | 60 | 未识别频次 Modal |

## 列表页

- 列：identifier (Tag, 未识别加 `未识别 fallback` Tag) / cnName / enName / neType / severityCode (颜色 Tag) / eventType / isShow / 操作
- 4 个过滤器：keyword / neType / severityCode (从 severity-levels API 拉) / **isUnknown (全部/已识别/未识别 fallback)**
- 工具栏：未识别频次 Modal / 重载 XML / 刷新缓存 / 新增定义

## 详情抽屉

- identifier 编辑态禁用（主键不可改）
- 严重级别从 GET /alarm-severity-levels 加载下拉，显示 "code - 中文 / English"
- 11 字段：identifier / neType / cnName / enName / severityCode / eventType / cnProbableCause / enProbableCause / cnSuggestion / enSuggestion / isShow

## 未识别频次 Modal

- GET /alarm-definitions/unknown-stats?productId=&days=
- 列：productId / productName / identifier / count / lastSeenAt
- 过滤：productId 输入 + days 下拉（1/7/30）

## API 调用映射

| UI 操作 | Hook | 后端端点 |
|---|---|---|
| 列表 | useAlarmDefinitionList | GET /alarm-definitions |
| 严重级别下拉 | useAlarmSeverityLevels | GET /alarm-severity-levels |
| 创建 | useCreateAlarmDefinition | POST /alarm-definitions |
| 更新 | useUpdateAlarmDefinition | PUT /alarm-definitions/:identifier |
| 删除 | useDeleteAlarmDefinition | DELETE /alarm-definitions/:identifier |
| 未识别频次 | useUnknownAlarmStats | GET /alarm-definitions/unknown-stats |
| 重载 XML | useAlarmDefinitionImportDirectory | POST /alarm-definitions/import-directory |
| 刷新缓存 | useAlarmDefinitionCacheRefresh | POST /alarm-definitions/cache/refresh |

## 与 AlarmSupportLibrary 并存

- 业务用户搜索视角：webcode/src/pages/alarm/AlarmSupportLibrary（旧 alarm_libraries 表）
- 网管员治理视角：本 P4-06（新 alarm_definitions 表，P1-04 DDL）
- 两表 P5-06（D2=B 决议）DROP 旧 alarm_libraries 后合一

## 编译验证

```
$ cd omcmb/webcode && npm run typecheck
> webcode@0.0.0 typecheck
> tsc --noEmit
[exit 0, 0 错误]
```

✅ typecheck 全过

## DoD

- [x] 单页列表 + 4 过滤
- [x] 详情抽屉 11 字段 CRUD
- [x] 严重级别从只读 API 拉下拉
- [x] 未识别频次 Modal（productId + days 过滤）
- [x] is_unknown 过滤
- [x] webcode typecheck 通过
