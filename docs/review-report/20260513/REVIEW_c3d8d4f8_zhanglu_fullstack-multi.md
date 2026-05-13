# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-13 17:38 |
| 提交 | c3d8d4f8 |
| 作者 | zhanglu |
| 范围 | fullstack-multi |
| 变更文件数 | 13 |
| 新增行数 | +571 |
| 删除行数 | -11 |

## 变更概要

本次变更集中修复告警模块两条实际问题链路，并补齐相应验证与运行环境配套。后端新增 device_group 规则解析能力并给告警列表接口补上 event_type 过滤，前端补齐事件类型参数透传和 FilterBar 日期范围回填，同时修复 Docker web 镜像校验命令对 BusyBox grep 的兼容性。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

- [omcgo/internal/alarm/pg_store.go](omcgo/internal/alarm/pg_store.go): 当前 event_type 过滤通过 SQL 归一化表达式匹配多种别名，功能正确，但函数包裹列值会牺牲普通索引命中率；如果活动告警量继续增长，建议后续将 event_type 在写入侧规范化，或补一个可索引的规范化列。
- [omcmb/webcode/src/components/FilterBar/index.tsx](omcmb/webcode/src/components/FilterBar/index.tsx): 日期范围回填修复目前由类型检查和人工验证覆盖，后续可以补一条组件级用例，锁住 sessionStorage/initialValues 恢复 dayjs 的行为。

## 详细分析

### deployments/docker/Dockerfile.web

- 将 grep -P 改为 grep -E，修复 Alpine/BusyBox 下构建校验不兼容的问题。该修改保持原有“校验 index.html 引用的 JS 入口文件存在”的意图，没有引入行为回退。

### omcgo/internal/alarm/device_group_resolver.go

- 新增 DeviceGroupResolver 接口和 PostgreSQL 实现，查询通过 Squirrel 构造，错误包装完整，NoRows 路径安全返回 nil，符合告警模块现有存储层模式。

### omcgo/internal/alarm/filter_engine.go

- device_group 规则由原先的占位式 match-all 改为基于 resolver 的 fail-closed 语义，修复了设备组规则可能误匹配所有告警的风险。

### omcgo/cmd/app/provider/alarm.go

- 在 DI 入口注入 PgDeviceGroupResolver，补齐了 device_group 规则从运行时到存储层的闭环。

### omcgo/internal/alarm/handler.go

### omcgo/internal/alarm/store.go

### omcgo/internal/alarm/pg_store.go

- active/history 列表接口新增 event_type 查询参数绑定与过滤，前后端契约补齐。后端额外做了编码值、枚举值、TR-069 原始字符串三套别名归一化，能覆盖当前页面和历史存量数据两种来源。

### omcgo/internal/alarm/filter_engine_test.go

### omcgo/internal/alarm/engine_filter_integration_test.go

### omcgo/internal/alarm/pg_store_event_type_test.go

- 新增测试覆盖 device_group 规则匹配、resolver 缺失 fail-closed、event_type 别名归一化等关键路径，补上了本次核心行为的回归保护。

### omcmb/frontend-core/src/services/api/alarmApi.ts

- buildAlarmQuery 现在会透传 event_type，修复了当前告警页面事件类型筛选前端参数未发送的问题。

### omcmb/webcode/src/components/FilterBar/index.tsx

- 新增 hydrateFormValues，将 sessionStorage 和 initialValues 中的日期范围恢复为 dayjs 实例，修复日期筛选回填失效问题；改动范围局部，未改变其他字段恢复逻辑。

### .github/skills/omc-linux-dev-environment/SKILL.md

- 新增环境说明文档，内容与当前工作区已经验证过的 Linux + Docker 运行模式一致，未发现与现有仓库结构冲突的描述。

## 业务完整性检查

- Handler-Service-Repository 链路：通过
- DI / 路由注册：通过
- 前后端接口契约一致性：通过
- 测试覆盖：通过（后端新增单测/集成测试；前端完成类型检查）

## 验证记录

- `cd omcgo && go test ./internal/alarm` ✅
- `cd omcmb/webcode && npm run typecheck` ✅
- `cd /home/zhanglu/goomc && docker compose -f deployments/docker/docker-compose.yml build web` ✅
- `cd /home/zhanglu/goomc && docker compose -f deployments/docker/docker-compose.yml up -d web` ✅

## 审查结论

PASS_WITH_WARNINGS
