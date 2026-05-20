# Review — chore(migration): 隐藏一级菜单"配置管理"

- 范围：`omcgo/migrations/seed/000135_hide_config_management_menu.sql`
- Backlog：HOTFIX（菜单临时调整）
- 结论：**PASS**

## 背景

一级菜单"配置管理"在 `omcmb/webcode/src/components/Layout/Sidebar/navConfig.ts` 中已注释隐藏（静态 fallback 视图），但 DB `menus` 表 `show_status='show'`，动态菜单接口 `/api/v1/auth/menus` 仍会返回这条 + 唯一子菜单"批量参数模板"，前端继续渲染。

## 变更

5 行 SQL，两条 UPDATE：

- 目录 `aaaa0120-0000-0000-0000-000000000001` (配置管理) → `show_status='hide'`
- 子菜单 `aaaa0120-1000-0000-0000-000000000001` (批量参数模板) → `show_status='hide'`

Down 段对称回滚。

## 审查清单

- [x] 版本号 135 = max(134) + 1，连续递增
- [x] goose Up / Down 段配对
- [x] 无 DO 块/函数，不需 StatementBegin/End
- [x] 仅按 id 定位的 UPDATE，无 DDL / 外键 / TimescaleDB
- [x] 不涉及 INSERT，无需 ON CONFLICT
- [x] UUID 格式合规
- [x] Down 完整回滚

## 风险

低。`show_status='hide'` 仅影响渲染，不删除记录、不动 role_menus 绑定，回滚直接置回 'show' 即恢复。
