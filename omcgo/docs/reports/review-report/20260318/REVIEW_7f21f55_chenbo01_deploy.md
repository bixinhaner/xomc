# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-03-18 |
| 作者 | chenbo01 |
| 基准 | 7f21f55 |
| Scope | deploy |
| Type | fix |
| 文件数 | 2 |

## 变更概述

修复 docker compose 启动时数据库表不存在的问题。新增独立 `migrate` 服务在 app/worker/acs 之前执行数据库迁移，通过 `service_completed_successfully` 条件保证启动顺序。迁移目录通过 volume 挂载，无需重建镜像即可应用新迁移。

## 审查结果

**结论: PASS**

### 检查项

| # | 检查项 | 结果 | 说明 |
|---|--------|------|------|
| 1 | 服务依赖链 | ✅ | postgres(healthy) → migrate(completed) → app/worker/acs |
| 2 | Volume 路径 | ✅ | `../../migrations` 相对于 compose 位置正确指向 `omcgo/migrations/` |
| 3 | 只读挂载 | ✅ | `:ro` 防止容器意外修改迁移文件 |
| 4 | 幂等性 | ✅ | `omcgo-migrate up` 对已执行迁移输出 no change 后退出 |
| 5 | entrypoint 简化 | ✅ | 移除重复迁移逻辑，职责单一 |

### 零 CRITICAL / 零 WARNING

## 结论

修复方案正确，通过独立 migrate 服务和 `service_completed_successfully` 条件保证数据库 schema 就绪后应用才启动。

**审查结论: PASS**
