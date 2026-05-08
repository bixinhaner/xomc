# verify T-0098-P4-02 — 产品中心导航 + i18n + super_admin 守卫

> **范围**：扩展 `User.isSuperAdmin` 派生字段、PrivateRoute super_admin 守卫、navConfig 一级菜单"产品中心"+ 5 子项、i18n（zh-CN/en-US）5 nav + 5 placeholder key、5 stub 页面、routes 注册（`/403` + 5 product 路由）。
> **wave-batched**：是（Skip S0/S1）

## 改动文件

### frontend-core（业务层）
| 文件 | 改动 |
|---|---|
| [src/types/system.ts](omcmb/frontend-core/src/types/system.ts) | `User` 接口加 `isSuperAdmin?: boolean` 派生字段 |
| [src/services/api/authApi.ts](omcmb/frontend-core/src/services/api/authApi.ts) | `BackendUser` 加 `source` 字段；`mapBackendUserToFrontend` 派生 `isSuperAdmin = source === 'builtIn'`（与后端 `user.IsSuperAdmin()` 同义） |
| [src/i18n/zh-CN/index.ts](omcmb/frontend-core/src/i18n/zh-CN/index.ts) | 新增 11 keys：1 一级菜单 + 5 子项 + 5 占位文案 |
| [src/i18n/en-US/index.ts](omcmb/frontend-core/src/i18n/en-US/index.ts) | 同上英文 |

### webcode（UI 壳）
| 文件 | 改动 |
|---|---|
| [src/router/PrivateRoute.tsx](omcmb/webcode/src/router/PrivateRoute.tsx) | 加 `requireSuperAdmin?: boolean` prop；非超管 → `<Navigate to="/403" replace />` |
| [src/router/routes.tsx](omcmb/webcode/src/router/routes.tsx) | 加 `withSuperAdmin(Component)` helper；注册 `/403` Forbidden + 5 product 路由（含 stub lazy import） |
| [src/components/Layout/Sidebar/navConfig.ts](omcmb/webcode/src/components/Layout/Sidebar/navConfig.ts) | `NavGroup` 加 `requireSuperAdmin?: boolean`；新增 `nav.product` 一级（icon `AppstoreAddOutlined`）+ 5 子项 |
| [src/components/Layout/Sidebar/NavMenu.tsx](omcmb/webcode/src/components/Layout/Sidebar/NavMenu.tsx) | 引入 `useUserStore` 派生 `isSuperAdmin`；`filteredNav` 按 `requireSuperAdmin` 过滤；注册 `AppstoreAddOutlined` 图标 |
| [src/pages/product/products/index.tsx](omcmb/webcode/src/pages/product/products/index.tsx) | stub（P4-03 替换） |
| [src/pages/product/param-model/index.tsx](omcmb/webcode/src/pages/product/param-model/index.tsx) | stub（P4-04 替换） |
| [src/pages/product/kpi-library/index.tsx](omcmb/webcode/src/pages/product/kpi-library/index.tsx) | stub（P4-05 替换） |
| [src/pages/product/alarm-library/index.tsx](omcmb/webcode/src/pages/product/alarm-library/index.tsx) | stub（P4-06 替换） |
| [src/pages/product/orphan-devices/index.tsx](omcmb/webcode/src/pages/product/orphan-devices/index.tsx) | stub（P4-07 替换） |

## super_admin 守卫拓扑

```
登录响应 backend User { source: 'builtIn' | 'admin' | 'LDAP', ... }
  ↓ (authApi.mapBackendUserToFrontend)
User { isSuperAdmin: source === 'builtIn', source, ... } 写入 userStore
  ↓
NavMenu 渲染：filteredNav 过滤掉 requireSuperAdmin 组（admin/operator/viewer 看不到产品中心）
  ↓
若用户绕过菜单直接访问 /product/* → PrivateRoute requireSuperAdmin 拦截 → /403
```

## 路径与 i18n key 映射

| 菜单 | 路径 | zh-CN | en-US |
|---|---|---|---|
| 一级 | — | 产品中心 | Product Center |
| 子项 | /product/products       | 产品管理       | Products |
| 子项 | /product/param-model    | 参数模型       | Parameter Models |
| 子项 | /product/kpi-library    | KPI 指标库     | KPI Library |
| 子项 | /product/alarm-library  | 告警库         | Alarm Library |
| 子项 | /product/orphan-devices | 孤儿设备       | Orphan Devices |

## 编译验证

```
$ cd omcmb/webcode && npm run typecheck
> webcode@0.0.0 typecheck
> tsc --noEmit
[exit 0, 0 错误]
```

✅ typecheck 全过

## 不在本子任务

- 不实现 5 个治理页面 UI（P4-03..P4-07 处理）
- 不评估 webcode-v2/v3（P4-08 处理）
- 旧 PrivateRoute 调用点不变（不传 requireSuperAdmin 视同 false，向后兼容）

## DoD

- [x] User 类型扩展 isSuperAdmin
- [x] authApi 派生 isSuperAdmin
- [x] PrivateRoute requireSuperAdmin prop
- [x] navConfig 加产品中心 + 5 子项
- [x] NavMenu 按 isSuperAdmin 过滤菜单
- [x] routes 注册 5 个 stub + /403
- [x] i18n zh-CN + en-US 各加 11 keys
- [x] 5 个 stub 页面文件
- [x] webcode typecheck 通过
