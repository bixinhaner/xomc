# Code Review — 产品中心菜单 seed 补全

| 字段 | 值 |
|------|----|
| 日期 | 2026-05-13 |
| 范围 | `omcgo/migrations/seed/000087_seed_product_center_menus.sql` |
| 类型 | hotfix（T-0098-P4 wave 收官遗留） |
| Scope | migration |
| 关联 | 补 T-0098-P4-02（super_admin 守卫 + 5 stub）已 done 任务的 menus 表 seed 遗留 |
| 审查结论 | **PASS**（无 CRITICAL / WARNING） |

## 背景

T-0098-P4 wave 完成了产品中心 5 个治理页（产品 / 参数模型 / KPI 库 / 告警库 / 孤儿设备）+ frontend `routes.tsx` withSuperAdmin 守卫 + `NAV_CONFIG.requireSuperAdmin=true` 前端过滤。但**没人写对应的 menus 表 seed**，导致动态菜单分支（`VITE_DYNAMIC_MENU=true`，当前生效）从后端 `/api/v1/auth/menus` 读取菜单树时拿不到产品中心 5 项，侧边栏完全不见。

## 修复内容

1. **menus 表**：插入 1 directory (`/product`, sort=12) + 5 page menu（5 个治理页路由）
2. **role_menus 表**：仅绑定 admin (10000000-...001) 共 6 条；operator/viewer 不绑 = "仅超管可见"语义（与前端 `requireSuperAdmin` 一致）
3. **UUID namespace**：`aaaa0098-*`（T-0098 P4 助记），与既有 license `aaaa0009` / ops `aaaa000a` 区分
4. **icon**：directory 用 `AppstoreAddOutlined`（与 `navConfig.ts` 完全一致）

## 检查清单

### CRITICAL（必修）

| 项 | 结果 | 说明 |
|---|------|------|
| SQL 注入 / 字符串拼接 | ✅ | 纯字面量 INSERT/SELECT FROM VALUES，无变量插值 |
| 运营商硬编码 | ✅ | 与 carrier 无关 |
| 敏感信息泄漏 | ✅ | 无 |

### WARNING（建议修）

| 项 | 结果 | 说明 |
|---|------|------|
| 版本号连续性 | ✅ | seed/ 目录现有最大 `000086`，本次 `000087` 衔接，无空洞 |
| 幂等性 | ✅ | menus `ON CONFLICT (id) DO UPDATE`，role_menus `ON CONFLICT (role_id, menu_id) DO NOTHING`，可重复执行 |
| Down 段完整性 | ✅ | role_menus 6 条 DELETE + menus 6 节点 DELETE，与 Up 创建对象一一对应 |
| UUID 格式 | ✅ | 8-4-4-4-12 严格合规 |
| goose StatementBegin/End | N/A | 无 DO 块 / 函数定义，无需包裹 |
| TimescaleDB 压缩顺序 | N/A | 非 hypertable 改动 |
| 分区表 FK | N/A | menus/role_menus 非分区表 |
| 与现有 seed 重复 | ✅ | 数据库现状 `permission_key LIKE 'product%'` 为空，无冲突 |
| seed/schema goose 表分离 | ✅ | 已确认 docker-compose 中 `GOOSE_TABLE=goose_db_version_seed`，独立编号空间 |

### INFO

- 注释完整：包含背景、设计取舍（为何只绑 admin、为何不建 button 子节点）、命名约定、幂等性说明
- 参引：明确指向前端 `components/Layout/Sidebar/navConfig.ts` 和后端 `admin/service.go` 超管旁路逻辑，便于后续维护者快速理解一致性来源

## 运行验证

1. ✅ 应用到当前 docker 数据库（`omc-docker-postgres-1`，omcgo/omcgo）
2. ✅ `goose_db_version_seed` 标记 version 87 applied
3. ✅ 浏览器登录 admin/admin123 后侧边栏出现"产品中心"组（icon AppstoreOutlined）
4. ✅ 直接访问 `/product/products` 页面正常渲染列表（产品装配件列表 + 厂商/类型/参数模型/KPI 平台等列）
5. ✅ Console 零 error

## 后续 TODO

- **S7 补登记**：在 `docs/project/backlog.md` 增加 T-NNNN 条目，描述"补 T-0098-P4 wave 遗留：菜单 seed"，State=done，引用本次 commit hash

## 风险

- 无破坏性变更：纯新增 seed，不改任何既有数据
- 多实例并发执行安全：所有写入都有 ON CONFLICT 兜底
- 回滚安全：Down 段精确删除本次新增 12 条记录（6 menus + 6 role_menus），不影响其他数据
