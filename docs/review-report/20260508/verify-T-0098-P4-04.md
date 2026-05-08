# verify T-0098-P4-04 — 参数模型浏览器（3 Tabs）

> **范围**：实现 [/product/param-model](omcmb/webcode/src/pages/product/param-model) 页 — 3 Tabs（参数模型清单 / 默认映射 / 标准参数树）+ XML 导入 + storable 过滤 + {i} 占位符校验。
> **wave-batched**：是（Skip S0/S1）

## 新增文件

| 文件 | LOC | 职责 |
|---|---|---|
| [src/pages/product/param-model/index.tsx](omcmb/webcode/src/pages/product/param-model/index.tsx) | 92 | 3 Tabs 容器 + 顶部 XML 导入 / 刷新缓存按钮 |
| [src/pages/product/param-model/ModelsTab.tsx](omcmb/webcode/src/pages/product/param-model/ModelsTab.tsx) | 121 | Tab 1：参数模型清单 + 编辑元数据 + 删除 |
| [src/pages/product/param-model/MappingsTab.tsx](omcmb/webcode/src/pages/product/param-model/MappingsTab.tsx) | 252 | Tab 2：mappings + storable 过滤 + {i} 校验 + CRUD |
| [src/pages/product/param-model/StandardParamsTab.tsx](omcmb/webcode/src/pages/product/param-model/StandardParamsTab.tsx) | 178 | Tab 3：standard 参数 + 关键词 / entry_type 过滤 + Upsert |

## 功能清单

### Tab 1：参数模型清单
- 列：name (点击切换到 mappings Tab) / 总条目 / 对象数 / 参数数 / 加载源 (XML) / 激活 / 描述 / 操作（编辑 / 删除）
- 编辑元数据：仅可改 description / isActive；name 由 XML 决定
- 删除会级联删除 mappings（FK CASCADE，与后端一致）

### Tab 2：默认映射（依赖 Tab 1 选择 model）
- storable 过滤：全部 / 仅可存储 / 仅不可存储
- 关键词搜索：standardPath + privatePath
- **{i} 占位符校验**：standardPath 与 privatePath 的 `{i}` 计数 mismatch 时行首红色 `ExclamationCircleFilled` + Tooltip 显示具体计数差
- CRUD：Modal 表单（standardPath / privatePath / entryType / access / dataType / changeApplies / minValue / maxValue / softwareVersion / isStorable / isActive）

### Tab 3：标准参数树
- 列出全库 standard_params；关键词搜索 + entry_type 过滤
- Upsert：`useUpsertStandard()` 编辑模式禁用 standardPath 字段（不允许改主键）

### 顶部工具栏（页面级）
- XML 导入 / 重载：`POST /param-models/import-directory`
- 刷新缓存：`POST /param-models/cache/refresh`

## API 调用映射

| UI 操作 | Hook | 后端端点 |
|---|---|---|
| 模型列表 | useParamModelList | GET /param-models |
| 编辑元数据 | useUpdateParamModel | PUT /param-models/:name |
| 删除模型 | useDeleteParamModel | DELETE /param-models/:name |
| mappings 加载 | useParamMappings | GET /param-models/:name/mappings |
| 创建 mapping | useCreateMapping | POST /param-models/:name/mappings |
| 更新 mapping | useUpdateMapping | PUT /param-models/:name/mappings/:id |
| 删除 mapping | useDeleteMapping | DELETE /param-models/:name/mappings/:id |
| standard 加载 | useStandardParams | GET /param-models/standard |
| Upsert standard | useUpsertStandard | POST/PUT /param-models/standard[/:path] |
| 删除 standard | useDeleteStandard | DELETE /param-models/standard/:path |
| XML 导入 | useParamModelImportDirectory | POST /param-models/import-directory |
| 刷新缓存 | useParamModelCacheRefresh | POST /param-models/cache/refresh |

## 编译验证

```
$ cd omcmb/webcode && npm run typecheck
> webcode@0.0.0 typecheck
> tsc --noEmit
[exit 0, 0 错误]
```

✅ typecheck 全过

## 不在本子任务

- Tree 树形布局：Tab 3 当前用 Table 行扁平展示；树形渲染待 P5 阶段优化
- discovered mappings 视图：在产品详情抽屉中暴露（P4-03 字典引用段已留位）；本 Tab 不展示发现项
- XML 上传文件 picker：当前用 `import-directory` 按钮重新加载后端 datamodels/ 目录

## DoD

- [x] 3 Tabs 容器
- [x] Tab1 参数模型清单 + 编辑/删除
- [x] Tab2 默认映射 + storable 过滤 + {i} 校验 + CRUD
- [x] Tab3 标准参数树搜索 + Upsert
- [x] XML 导入 + 缓存刷新按钮
- [x] webcode typecheck 通过
