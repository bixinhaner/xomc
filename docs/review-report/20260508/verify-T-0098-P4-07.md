# verify T-0098-P4-07 — 孤儿设备治理

> **范围**：实现 [/product/orphan-devices](omcmb/webcode/src/pages/product/orphan-devices) 页 — 列表 + 单台/批量绑定 + 触发重新匹配。
> **wave-batched**：是（Skip S0/S1）

## 新增文件

| 文件 | LOC | 职责 |
|---|---|---|
| [src/pages/product/orphan-devices/index.tsx](omcmb/webcode/src/pages/product/orphan-devices/index.tsx) | 142 | 列表 + 工具栏（全量重新匹配 / 批量绑定 / 刷新）|
| [src/pages/product/orphan-devices/BindProductModal.tsx](omcmb/webcode/src/pages/product/orphan-devices/BindProductModal.tsx) | 86 | 选产品对话框（单台 / 批量串行）|

## 列表页

- 数据源：`GET /products/orphan-devices?limit=200`
- 列：SN / OUI / productClass (Tag) / 运营商 / 厂商 / 最后 Inform / 操作（单台绑定）
- 工具栏：
  - **全量重新匹配**：`POST /products/orphan-devices/rematch`（Popconfirm 二次确认；后端扫描全部 orphan 按现有 ProductRegistry 重新匹配）
  - **批量绑定**：rowSelection 选中后打开 BindProductModal
  - **刷新**：refetch
- limit 由 pagination 的 pageSize 派生（>200 时上调）

## BindProductModal

- 单台 / 批量都走同一对话框
- Alert 显示当前要绑定的设备 SN / productClass
- 产品下拉：从 `useProductList()` 加载，showSearch + 显示 `name (vendor · TECH)`
- 批量串行调用单台 `PUT /products/orphan-devices/:deviceId/bind`（避免后端短时间内大量并发）
- 成功 / 失败统计后 message.success / warning

## API 调用映射

| UI 操作 | Hook | 后端端点 |
|---|---|---|
| 列表 | useOrphanDevices | GET /products/orphan-devices |
| 全量重新匹配 | useRematchOrphan | POST /products/orphan-devices/rematch |
| 单台 / 批量绑定 | useBindOrphan | PUT /products/orphan-devices/:deviceId/bind |
| 产品下拉 | useProductList | GET /products |

## 编译验证

```
$ cd omcmb/webcode && npm run typecheck
> webcode@0.0.0 typecheck
> tsc --noEmit
[exit 0, 0 错误]
```

✅ typecheck 全过

## 治理闭环

P4-06 未识别频次 Modal 与本页通过 productId 维度联动：

```
设备 Bootstrap → ProductRegistry 未命中 → 写孤儿表（本页可见）
                                       ↓ 配置 enable_unknown_alarm=true
告警接收 → AlarmDefinition 未命中 → fallback (is_unknown=true)
                                  ↓ unknown-stats 聚合
P4-06 未识别频次 Modal → productId 维度 → 跳本页（手工绑定）
```

## DoD

- [x] 列表展示 (limit=200 默认)
- [x] 单台绑定（行操作）
- [x] 批量绑定（rowSelection + 工具栏）
- [x] 全量重新匹配（Popconfirm 保护）
- [x] BindProductModal 支持产品搜索下拉
- [x] webcode typecheck 通过
