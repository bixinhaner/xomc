# 02 — 服务目录

> 10 个微服务的完整清单、依赖关系、路由规则与代码生成策略

---

## 1. 服务清单

### 1.1 完整服务注册表

| 服务名 | 类型 | 框架 | 端口 | 职责 | goctl 生成 |
|--------|------|------|------|------|-----------|
| acs-engine | 独立进程 | net/http + zRPC | 7547(HTTP), 7548(TLS), 50050(gRPC), 9090(metrics) | F01: TR069 SOAP/XML 协议处理 | 否（手动） |
| device-api | API 网关 | go-zero rest | 8080 | 设备/配置/拓扑/开站 REST API | 是 |
| monitor-api | API 网关 | go-zero rest | 8081 | PM/告警/MR/北向 REST API | 是 |
| admin-api | API 网关 | go-zero rest | 8082 | 用户/RBAC/审计/固件 REST API | 是 |
| device-rpc | RPC 服务 | go-zero zRPC | 50051 | F06 设备管理, F07 网元直连, F09 自动开站 | 是 |
| config-rpc | RPC 服务 | go-zero zRPC | 50052 | F02 数据模型, 配置模板, OUI 注册 | 是 |
| pm-rpc | RPC 服务 | go-zero zRPC | 50053 | F03 PM/KPI, F05 MR 查询 | 是 |
| alarm-rpc | RPC 服务 | go-zero zRPC | 50054 | F04 告警接收/去重/关联/生命周期 | 是 |
| admin-rpc | RPC 服务 | go-zero zRPC | 50055 | RBAC 用户管理, 审计日志, F10 互操作测试 | 是 |
| worker | 后台进程 | go-zero ServiceGroup | 9092(metrics) | PM/MR 文件解析, KPI 计算, 告警关联 | 部分 |

### 1.2 etcd 服务注册

```yaml
# 所有 go-zero RPC 服务在 etcd 注册
# Key 格式: /omcgo/{service_name}/{instance_id}
/omcgo/device.rpc/192.168.1.10:50051
/omcgo/config.rpc/192.168.1.10:50052
/omcgo/pm.rpc/192.168.1.10:50053
/omcgo/alarm.rpc/192.168.1.10:50054
/omcgo/admin.rpc/192.168.1.10:50055
/omcgo/acs.rpc/192.168.1.10:50050
```

---

## 2. 服务依赖关系图

```
                    ┌─────────────┐
                    │  device-api  │
                    │    :8080     │
                    └──┬──┬──┬────┘
                       │  │  │
          ┌────────────┘  │  └────────────┐
          ▼               ▼               ▼
    ┌───────────┐  ┌───────────┐   ┌───────────┐
    │device-rpc │  │config-rpc │   │ acs-rpc   │
    │  :50051   │──│  :50052   │   │  :50050   │
    └─────┬─────┘  └───────────┘   └───────────┘
          │
          │ (开站时调用)
          ├──────────────────────────────────┐
          ▼                                  ▼
    ┌───────────┐                     ┌───────────┐
    │config-rpc │                     │ acs-rpc   │
    │(模板匹配) │                     │(命令下发) │
    └───────────┘                     └───────────┘

                    ┌──────────────┐
                    │ monitor-api  │
                    │    :8081     │
                    └──┬──────┬───┘
                       │      │
          ┌────────────┘      └────────────┐
          ▼                                ▼
    ┌───────────┐                   ┌───────────┐
    │  pm-rpc   │                   │ alarm-rpc │
    │  :50053   │                   │  :50054   │
    └───────────┘                   └───────────┘

                    ┌─────────────┐
                    │  admin-api   │
                    │    :8082     │
                    └──────┬──────┘
                           │
                           ▼
                    ┌───────────┐
                    │ admin-rpc │
                    │  :50055   │
                    └───────────┘

                    ┌─────────────┐
                    │   worker    │
                    │  (后台进程)  │
                    └──┬──┬──┬───┘
                       │  │  │
    ┌──────────────────┘  │  └──────────────────┐
    ▼                     ▼                     ▼
 ┌──────┐          ┌───────────┐          ┌───────────┐
 │ NATS │          │ pm-rpc    │          │ alarm-rpc │
 │(消费)│          │(写入数据) │          │(告警处理) │
 └──────┘          └───────────┘          └───────────┘
```

### 依赖矩阵

| 调用方 → 被调用方 | device-rpc | config-rpc | pm-rpc | alarm-rpc | admin-rpc | acs-rpc |
|:-----------------|:----------:|:----------:|:------:|:---------:|:---------:|:-------:|
| device-api       | **调用** | **调用** | — | — | — | **调用** |
| monitor-api      | — | — | **调用** | **调用** | — | — |
| admin-api        | — | — | — | — | **调用** | — |
| device-rpc       | — | **调用** | — | — | — | **调用** |
| worker           | — | — | **调用** | **调用** | — | — |
| acs-engine       | **调用** | **调用** | — | — | — | — |

---

## 3. API 网关路由规则

### 3.1 device-api (:8080)

| 方法 | 路径 | 下游 RPC | 说明 |
|------|------|---------|------|
| GET | `/api/v1/devices` | device-rpc.ListDevices | 设备列表（支持过滤） |
| GET | `/api/v1/devices/:id` | device-rpc.GetDevice | 设备详情 |
| PUT | `/api/v1/devices/:id` | device-rpc.UpdateDevice | 更新设备信息 |
| DELETE | `/api/v1/devices/:id` | device-rpc.DeleteDevice | 删除设备 |
| POST | `/api/v1/devices/:id/reboot` | acs-rpc.QueueCommand | 重启设备 |
| POST | `/api/v1/devices/:id/factory-reset` | acs-rpc.QueueCommand | 恢复出厂 |
| POST | `/api/v1/devices/:id/connection-request` | acs-rpc.SendConnectionRequest | 主动拉设备 |
| GET | `/api/v1/devices/:id/parameters` | device-rpc.GetDeviceParameters | 读取参数 |
| POST | `/api/v1/devices/:id/parameters` | acs-rpc.QueueCommand | 写入参数 |
| GET | `/api/v1/topology/tree` | device-rpc.GetTopologyTree | 拓扑树 |
| GET | `/api/v1/topology/groups` | device-rpc.ListGroups | 设备组列表 |
| POST | `/api/v1/topology/groups` | device-rpc.CreateGroup | 创建设备组 |
| PUT | `/api/v1/topology/groups/:id` | device-rpc.UpdateGroup | 更新设备组 |
| DELETE | `/api/v1/topology/groups/:id` | device-rpc.DeleteGroup | 删除设备组 |
| POST | `/api/v1/topology/groups/:id/devices` | device-rpc.AssignDeviceToGroup | 分配设备 |
| GET | `/api/v1/datamodels` | config-rpc.ListDataModels | 数据模型列表 |
| GET | `/api/v1/datamodels/:id` | config-rpc.GetDataModel | 数据模型详情 |
| POST | `/api/v1/datamodels/import` | config-rpc.ImportDataModel | 导入数据模型 |
| POST | `/api/v1/datamodels/:id/activate` | config-rpc.ActivateDataModel | 激活数据模型 |
| GET | `/api/v1/templates` | config-rpc.ListTemplates | 配置模板列表 |
| POST | `/api/v1/templates` | config-rpc.CreateTemplate | 创建配置模板 |
| GET | `/api/v1/provisioning/tasks` | device-rpc.ListProvisioningTasks | 开站任务列表 |
| GET | `/api/v1/provisioning/tasks/:id` | device-rpc.GetProvisioningTask | 开站任务详情 |
| POST | `/api/v1/provisioning/tasks/:id/retry` | device-rpc.RetryProvisioning | 重试开站 |

### 3.2 monitor-api (:8081)

| 方法 | 路径 | 下游 RPC | 说明 |
|------|------|---------|------|
| GET | `/api/v1/pm/counters` | pm-rpc.QueryCounters | PM 计数器查询 |
| GET | `/api/v1/pm/kpi` | pm-rpc.QueryKPI | KPI 查询 |
| GET | `/api/v1/pm/kpi/definitions` | pm-rpc.GetKPIDefinitions | KPI 定义列表 |
| POST | `/api/v1/pm/export` | pm-rpc.TriggerExport | 触发 PM 导出 |
| GET | `/api/v1/alarms/active` | alarm-rpc.ListActiveAlarms | 活跃告警列表 |
| GET | `/api/v1/alarms/history` | alarm-rpc.ListHistoryAlarms | 历史告警查询 |
| POST | `/api/v1/alarms/:id/acknowledge` | alarm-rpc.AcknowledgeAlarm | 确认告警 |
| POST | `/api/v1/alarms/:id/clear` | alarm-rpc.ClearAlarm | 清除告警 |
| GET | `/api/v1/alarms/statistics` | alarm-rpc.GetAlarmStatistics | 告警统计 |
| GET | `/api/v1/mr/reports` | pm-rpc.QueryMeasurementReports | MR 报告查询 |
| GET | `/api/v1/northbound/pm` | pm-rpc.QueryCounters | 北向 PM 接口 |
| GET | `/api/v1/northbound/alarms` | alarm-rpc.ListActiveAlarms | 北向告警接口 |
| POST | `/api/v1/northbound/sync` | pm-rpc.TriggerExport | 北向全量同步 |

### 3.3 admin-api (:8082)

| 方法 | 路径 | 下游 RPC | 说明 |
|------|------|---------|------|
| GET | `/api/v1/users` | admin-rpc.ListUsers | 用户列表 |
| POST | `/api/v1/users` | admin-rpc.CreateUser | 创建用户 |
| PUT | `/api/v1/users/:id` | admin-rpc.UpdateUser | 更新用户 |
| DELETE | `/api/v1/users/:id` | admin-rpc.DeleteUser | 删除用户 |
| POST | `/api/v1/auth/login` | admin-rpc.Login | 登录 |
| POST | `/api/v1/auth/refresh` | admin-rpc.RefreshToken | 刷新 Token |
| GET | `/api/v1/roles` | admin-rpc.ListRoles | 角色列表 |
| POST | `/api/v1/roles` | admin-rpc.CreateRole | 创建角色 |
| PUT | `/api/v1/roles/:id/permissions` | admin-rpc.UpdatePermissions | 更新权限 |
| GET | `/api/v1/audit/logs` | admin-rpc.ListAuditLogs | 审计日志 |
| GET | `/api/v1/firmware` | admin-rpc.ListFirmware | 固件列表 |
| POST | `/api/v1/firmware` | admin-rpc.UploadFirmware | 上传固件 |
| POST | `/api/v1/firmware/:id/upgrade` | device-rpc + acs-rpc | 批量升级 |
| GET | `/api/v1/oui` | config-rpc.ListOUI | OUI 注册表 |
| POST | `/api/v1/interop/tests` | admin-rpc.RunInteropTest | 互操作测试 |
| GET | `/api/v1/interop/results` | admin-rpc.GetInteropResults | 测试结果 |

---

## 4. goctl 代码生成策略

### 4.1 可生成与需手写的边界

| 层面 | goctl 可生成 | 需手动编写 |
|------|-------------|-----------|
| API Handler | handler 框架代码、路由注册、类型定义 | 业务逻辑（logic 层） |
| RPC Server | server 框架、client 代码、pb 类型 | 业务逻辑（logic 层） |
| Model（简单表） | CRUD + 缓存 | 复杂查询、JSONB、分区表 |
| Dockerfile | 基础镜像配置 | 多阶段构建优化 |
| K8s YAML | 基础 Deployment | HPA、PVC、ConfigMap |

### 4.2 goctl model 适用性

| 表名 | goctl model | 说明 |
|------|:-----------:|------|
| devices | 部分 | 分区表需手写，简单 CRUD 可生成 |
| device_parameters | 否 | 复合主键 + 批量操作 |
| device_groups | **是** | 简单 CRUD，可带缓存 |
| device_group_members | 否 | 联合查询 |
| data_model_definitions | 否 | JSONB parameter_tree + 三级回退 |
| config_templates | **是** | 简单 CRUD |
| oui_registry | **是** | 简单查询 + 缓存 |
| provisioning_tasks | **是** | 简单 CRUD + 状态更新 |
| pm_counters | 否 | TimescaleDB hypertable |
| kpi_values | 否 | TimescaleDB hypertable |
| alarms_active | 否 | 复杂查询 + 生命周期 |
| alarms_history | 否 | TimescaleDB hypertable |
| measurement_reports | 否 | TimescaleDB hypertable |
| users | **是** | 简单 CRUD + 缓存 |
| roles | **是** | 简单 CRUD |
| permissions | **是** | 简单 CRUD |
| audit_logs | 否 | 仅追加 + 时间范围查询 |
| firmware_versions | **是** | 简单 CRUD |
| upgrade_tasks | **是** | 简单 CRUD + 状态更新 |

**统计**：19 张表中 9 张可用 goctl model 生成（~47%），其余需手写 SQL。

### 4.3 代码生成命令示例

```bash
# API 服务生成
goctl api go -api service/device/api/device.api \
    -dir service/device/api/ \
    --style goZero

# RPC 服务生成
goctl rpc protoc api/proto/device.proto \
    --go_out=service/device/rpc/pb \
    --go-grpc_out=service/device/rpc/pb \
    --zrpc_out=service/device/rpc/ \
    --style goZero

# Model 生成（带缓存）
goctl model pg datasource \
    -url="postgres://omcgo:password@localhost:5432/omcgo?sslmode=disable" \
    -table="users" \
    -dir="service/admin/model" \
    -cache \
    --style goZero

# Dockerfile 生成
goctl docker -go service/device/rpc/device.go

# K8s 部署清单
goctl kube deploy -name device-rpc \
    -namespace omcgo \
    -image omcgo/device-rpc:latest \
    -port 50051
```

### 4.4 goctl 再生成保护策略

go-zero 的 goctl 有覆盖风险。保护策略：

1. **严格分层**：生成代码在 `handler/`、`types/`、`server/`；手写代码在 `logic/`、`svc/`、`model/`
2. **Logic 永不覆盖**：goctl 生成 logic stub 仅在首次创建，已存在则跳过
3. **自定义 model**：goctl model 生成 `*_gen.go`（可覆盖），自定义扩展写在 `*_custom.go`
4. **版本控制**：所有生成文件纳入 Git，diff 即可发现意外覆盖
