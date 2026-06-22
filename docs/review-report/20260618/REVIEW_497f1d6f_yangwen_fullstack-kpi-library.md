# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-18 |
| 提交 | 497f1d6f (工作树未提交) |
| 作者 | yangwen |
| 范围 | fullstack-kpi-library (frontend-core + 三皮肤 + migration) |
| 变更文件数 | 10 改 + 3 新 + 1 迁移 |
| 新增行数 | ~1230 |

## 变更概要

Issue #525：三皮肤(v1/v2/v3)补齐 KPI 指标分组(功能集)维护 UI，接 frontend-core 已有 useCreateGroup/useUpdateGroup/useDeleteGroup hook（新建/编辑/删除分组 + 二次确认 + 内置组保护 + i18n）。frontend-core 补 IndicatorGroup.isBuildIn 字段 + mapGroup 映射 + 14 条 i18n(en/zh)。附带恢复分组种子迁移 000002（修复 6-08 dd991a44 误删）。

## 审查发现

### 🔴 CRITICAL (严重)
无

### 🟡 WARNING (警告)
无

### 🔵 INFO (建议)
1. **v1 `destroyOnClose` 已被 Antd5 弃用** — GroupsManageModal.tsx 两处 Modal 用 `destroyOnClose`，控制台有 deprecation warning（建议改 `destroyOnHidden`）。本次 /simplify 已修。
2. **v2 KpiLibrary 页面级既有硬编码中文** — KpiLibrary.tsx 原有「KPI 指标库/刷新/搜索平台」等未走 i18n，属本任务前既有状况；本次新增的分组维护文案全部走 t()，不扩大该既有问题。可另开任务收口。

## 详细分析

### 三个新 Modal/Dialog（v1/v2/v3）
- 三个 mutation hook 均正确 import + 调用，入参形状与 frontend-core 签名一致（v1/v3 mutateAsync，v2 mutate）。
- **内置组保护**：三皮肤均 `Boolean(row.isBuildIn)` 判定，编辑/删除按钮 disabled + 提示文案；v1 内置组删除走 disabled span（不挂 Popconfirm），v2/v3 disabled 按钮。
- **删除二次确认**：v1 Popconfirm；v2/v3 用 pendingDelete 状态 + 独立确认浮层（手写，因 shadcn/HUD 无 Popconfirm），均不直接删。
- **i18n**：三文件 CJK 仅出现在 JSX 注释，用户可见文本全走 t()/useT()。
- **类型安全**：无 any/as any；errMsg 用 AxiosError 类型化；unknown 收口。
- id 生成 crypto.randomUUID().replace(/-/g,'')，name 必填校验，maxLength 限制。

### frontend-core
- types/indicatorLibrary.ts：IndicatorGroup 加 isBuildIn?:boolean。
- indicatorLibraryApi.ts：BackendGroup 加 is_build_in?:string + mapGroup 补 isBuildIn:b.is_build_in==='1'。一处改、三皮肤共享。
- i18n en/zh 各补 product.kpi.group.* 14 key，双份一一对应。

### migration 000002_restore_indicator_groups.sql
- 幂等 ON CONFLICT DO NOTHING；id 与 XML/perf_indicators_*.group_id 全量对齐；Down 段无破坏性 DELETE（注释说明被 FK 引用不可回滚）。

## 代码质量回退检查
未发现回退：无删测试/删错误处理/降安全/引入 any/硬编码替代配置。

## 前后端一致性检查
后端 CRUD 端点(/indicator-groups GET/POST/PUT/DELETE)已存在，本次仅前端接入，契约一致（hook 入参 = 端点签名）。

## 配套更新提醒
- 文档：问题记录已在 docs/zhangguihua/ 留档；CLAUDE.md 无需改。
- 单测：前端组件，三皮肤 typecheck + Playwright 冒烟已验；如需可补组件测试（INFO）。
- E2E：分组 CRUD 走 UI，已 Playwright 冒烟三皮肤。

## 安全检查
未触 auth/middleware；删除/编辑经后端鉴权端点；无敏感数据日志。无需 /security-review。

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 0 |
| INFO | 2 |

**审查结论**: PASS_WITH_WARNINGS（实为仅 2 INFO；无 CRITICAL/WARNING，满足 ship P7 硬门）
