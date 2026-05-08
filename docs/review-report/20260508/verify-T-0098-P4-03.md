# verify T-0098-P4-03 — 产品管理页（最复杂）

> **范围**：实现 [/product/products](omcmb/webcode/src/pages/product/products) 全页 — 列表 + 4 段抽屉 + 全局匹配顺序浮窗 + 测试匹配实时面板。
> **wave-batched**：是（Skip S0/S1）

## 新增文件

| 文件 | LOC | 职责 |
|---|---|---|
| [src/pages/product/products/index.tsx](omcmb/webcode/src/pages/product/products/index.tsx) | 220 | 主页面：列表 + 工具栏 + MatchTester + 浮窗触发 |
| [src/pages/product/products/ProductDrawer.tsx](omcmb/webcode/src/pages/product/products/ProductDrawer.tsx) | 384 | 4 段抽屉（基本信息 / 字典引用 / 上传策略 / 正则模式）+ patterns 上下移 |
| [src/pages/product/products/MatchOrderDrawer.tsx](omcmb/webcode/src/pages/product/products/MatchOrderDrawer.tsx) | 67 | 全局匹配顺序浮窗（GET /products/match-order）|
| [src/pages/product/products/MatchTester.tsx](omcmb/webcode/src/pages/product/products/MatchTester.tsx) | 70 | 测试匹配实时面板（GET /products/match）+ 300ms debounce |

> 替换了 P4-02 的 stub `index.tsx`。

## 功能清单

### 列表（顶部工具栏 + 主表格）
- 搜索：keyword（产品名/厂商/描述）+ tech 下拉过滤
- 操作按钮：新增产品 / 全局匹配顺序 / 重载 XML / 刷新缓存
- 列：name / vendor / tech / 指标设备类型 / 指标平台 / 告警 NE 类型 / 上传开关 / 未知告警 / 设备数 / 操作（编辑 / 重置发现 / 删除）

### MatchTester（实时测试）
- Input 输入 productClass（debounce 300ms）→ `productApi.match(productClass)` 调 `GET /products/match`
- 命中：显示绿色 Tag + 命中产品名 + 命中规则 + global_order
- 未命中：橙色 Tag + 提示该 productClass 将进入孤儿设备列表

### MatchOrderDrawer（全局匹配顺序浮窗）
- Drawer width=720
- Table：sort_order / 产品名 / 正则规则（code 字体）/ 激活
- 数据来自 `useMatchOrder()` → `GET /products/match-order`

### ProductDrawer（4 段抽屉）
1. **基本信息** — name / description / vendor / tech / radioModes / 参数模型下拉（关联 useParamModelList）/ indicator_device_type / indicator_platform / alarm_ne_type
2. **字典引用** — 只读展示当前选中的参数模型 / 指标平台 / 告警 NE 类型；引用条目数 P4-04..P4-06 后再补
3. **上传策略** — enable_filetype11 Switch / device_attrs_override 5 列 Checkbox 矩阵（data_type 强制 disabled，与后端 `data_type=true is not allowed` 一致）/ enable_unknown_alarm Switch
4. **正则模式**（仅编辑态）— 添加 / 编辑 (Input onBlur 触发更新) / 删除 (Popconfirm) / 上下移（调 `PUT /:id/patterns/:patternId/move`）

## API 调用映射

| UI 操作 | Hook | 后端端点 |
|---|---|---|
| 列表加载 | useProductList | GET /products |
| 详情 | useProductDetail | GET /products/:id |
| 新增 | useCreateProduct | POST /products |
| 编辑 | useUpdateProduct | PUT /products/:id |
| 删除 | useDeleteProduct | DELETE /products/:id |
| 重置发现 | useResetDiscovered | DELETE /products/:id/discovered |
| 添加规则 | useCreatePattern | POST /products/:id/patterns |
| 编辑规则 | useUpdatePattern | PUT /products/:id/patterns/:patternId |
| 删除规则 | useDeletePattern | DELETE /products/:id/patterns/:patternId |
| 上下移 | useMovePattern | PUT /products/:id/patterns/:patternId/move |
| 测试匹配 | productApi.match | GET /products/match |
| 全局顺序 | useMatchOrder | GET /products/match-order |
| 重载 XML | useProductImportDirectory | POST /products/import-directory |
| 刷新缓存 | useProductCacheRefresh | POST /products/cache/refresh |
| 参数模型下拉 | useParamModelList | GET /param-models |

## 编译验证

```
$ cd omcmb/webcode && npm run typecheck
> webcode@0.0.0 typecheck
> tsc --noEmit
[exit 0, 0 错误]
```

✅ typecheck 全过

## 不在本子任务

- 字典引用段的"条目数 / 指标数 / 告警数"聚合统计：占位文案待 P4-04..P4-06 实现后端聚合 API 后补
- 浏览器手动验证：dev server `:3000` 起动后续验证（CLAUDE.md §8 商用级要求）；本次以 typecheck 为主门
- 单元测试 / Playwright E2E：本次 P4-03 不写

## DoD

- [x] 主页面 4 文件
- [x] 列表 + 搜索 + 工具栏齐全
- [x] MatchTester 实时（debounce 300ms）
- [x] MatchOrderDrawer 浮窗
- [x] ProductDrawer 4 段（基本/字典引用/上传策略/正则）
- [x] patterns 上下移调后端 move 接口
- [x] webcode typecheck 通过
