# Database Migrations

迁移工具：`pressly/goose/v3`。版本记录在数据库 `goose_db_version` 表中。

## 文件格式

单文件格式（goose），包含 Up 和 Down 两个段落：

```sql
-- +goose Up
CREATE TABLE ...;

-- +goose Down
DROP TABLE IF EXISTS ...;
```

## 当前文件清单

| 文件 | 功能域 | 说明 |
|------|--------|------|
| 000001_extensions_functions.sql | 基础 | TimescaleDB 扩展、通用函数 |
| 000002_users_roles.sql | F06 | 用户、角色、权限 |
| 000003_devices.sql | F06 | 设备表（Hash 分区）、参数、任务 |
| 000004_data_models.sql | F02 | 数据模型定义 |
| 000005_config_pm.sql | F03 | PM 计数器、KPI（TimescaleDB hypertable） |
| 000006_alarms_mr_firmware.sql | F04/F05 | 告警、测量报告、固件 |
| 000007_system_infra.sql | F06 | MML、文件管理、备份、仪表盘 |
| 000008_nedirect_northbound.sql | F07/F08 | 网元直连、北向接口 |
| 000009_sys_admin.sql | F06 | 系统管理（字典、配置、菜单、API 端点） |
| 000011_optimize_indexes.sql | 全局 | 索引优化 |
| 000012_api_endpoints_and_data_perm.sql | F06 | API 端点管理、数据权限 |
| 000013_alarm_management_enhancement.sql | F04 | 告警管理增强 |
| 000014_mml_templates_and_audit.sql | F06 | MML 模板与审计 |
| 000019_alarm_clear_fields.sql | F04 | 告警清除字段 |
| 000020_mml_category_group_and_executor.sql | F06 | MML 分类、分组、执行器 |
| 000021_notifications.sql | F06 | 通知 |
| 000022_mml_param_library.sql | F06 | MML 参数库 |

## 种子数据

`seed/` 目录包含初始种子数据，紧接主目录版本号继续递增。

## 新增迁移

详见 `omcgo/CLAUDE.md` 第 5.5 节「数据库迁移规范」，包含版本号规则、goose 注解要求、自查清单等完整规范。
