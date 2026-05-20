# Review — chore(frontend,migration): 产品中心菜单顺序调整

- 范围：
  - `omcmb/webcode/src/components/Layout/Sidebar/navConfig.ts`
  - `omcgo/migrations/seed/000134_reorder_product_center_menus.sql`
- Backlog：HOTFIX（用户临时菜单调整）
- 结论：**PASS**

## 背景

用户要求把"产品中心"下的"产品管理"子菜单下放到"告警库"下面。前端动态菜单从后端 `/api/v1/auth/menus` 读取（VITE_DYNAMIC_MENU=true），实际渲染顺序由 DB `menus.sort_order` 决定；navConfig.ts 是 fallback。两处都需要调整。

## 变更

### 1. navConfig.ts

调整 `product` 分组 `children` 数组顺序：

| 旧顺序 | 新顺序 |
|--------|--------|
| 产品管理 | 参数模型 |
| 参数模型 | KPI 指标库 |
| KPI 指标库 | 告警库 |
| 告警库 | 产品管理 |
| 孤儿设备 | 孤儿设备 |

### 2. seed/000134_reorder_product_center_menus.sql

5 个 `UPDATE menus SET sort_order = N` 语句，按新顺序调整 sort_order 1..5。

## 审查清单

### Migration

- [x] 版本号 134 = max(133) + 1，连续递增（migrations/ 最大 133 + seed/ 最大 129）
- [x] goose Up / Down 段配对
- [x] 无 DO 块/函数/触发器，不需要 StatementBegin/End
- [x] 仅 UPDATE，不涉及 DDL / 外键 / TimescaleDB
- [x] UUID 格式合规（8-4-4-4-12）
- [x] Down 段完整回滚 sort_order 到旧值
- [x] 仅按 id 定位单行 UPDATE，无 ON CONFLICT 需求

### Frontend

- [x] 仅数组顺序调整，无逻辑变化
- [x] 无类型变更，不影响 webcode-v2/v3

## 风险

低。改动仅影响菜单展示顺序，无 RBAC、路由、API 变化。

## 验证

- 浏览器 `:8081` 进入页面，展开"产品中心" → 顺序确认为 `参数模型 / KPI 指标库 / 告警库 / 产品管理 / 孤儿设备` ✓
- DB `SELECT name, sort_order FROM menus WHERE parent_id = '...001'` 返回新顺序 ✓
