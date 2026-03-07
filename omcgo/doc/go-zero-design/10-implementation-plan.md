# 10 — 实施计划

> 4 阶段 17 Sprint 分解、风险评估、工时对比、团队要求

---

## 1. 实施阶段总览

```
Phase 1: 基础建设        Phase 2: 核心功能        Phase 3: 数据管线       Phase 4: 北向与生产化
(6 Sprint, 34 人天)      (4 Sprint, 25 人天)      (3 Sprint, 18 人天)     (4 Sprint, 23 人天)
┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐
│ S1.1 脚手架 3d    │   │ S2.1 ACS RPC 8d  │   │ S3.1 PM/KPI 7d   │   │ S4.1 北向 5d     │
│ S1.2 基础设施 5d  │   │ S2.2 数据模型 6d  │   │ S3.2 告警 5d     │   │ S4.2 管理 5d     │
│ S1.3 公共包 5d    │   │ S2.3 运营商 5d    │   │ S3.3 MR 6d       │   │ S4.3 集成测试 8d  │
│ S1.4 goctl 3d    │   │ S2.4 开站 6d      │   │                  │   │ S4.4 K8s/生产 5d  │
│ S1.5 ACS 核心 10d │   │                  │   │                  │   │                  │
│ S1.6 设备注册 8d  │   │                  │   │                  │   │                  │
└──────────────────┘   └──────────────────┘   └──────────────────┘   └──────────────────┘

总计: ~100 人天（含分布式测试开销）
```

---

## 2. Sprint 详细分解

### Phase 1: 基础建设（目标：ACS 引擎接收 Inform 并注册设备）

#### S1.1 — 项目脚手架（3 人天）

| 任务 | 说明 |
|------|------|
| 初始化 monorepo | go.mod、目录结构、Makefile |
| 依赖引入 | go-zero v1.7+、pgx/v5、go-redis/v9、nats.go、etree 等 18 个核心依赖 |
| golangci-lint | 代码检查配置 |
| CI 基础 | GitHub Actions: lint + test |
| 配置模板 | 10 个 YAML 配置文件模板 |

#### S1.2 — 基础设施层（5 人天）

| 任务 | 说明 |
|------|------|
| PostgreSQL 连接 | pgxpool 封装、连接参数、健康检查 |
| TimescaleDB 验证 | 扩展安装验证、hypertable 创建 |
| Redis 连接 | 自动检测 Cluster/Standalone |
| NATS 连接 | JetStream 初始化、6 个 Stream 定义 |
| MinIO 连接 | Bucket 初始化（pm-files, mr-files, firmware, config-backup, logs） |
| etcd 连接 | go-zero etcd 配置验证 |
| docker-compose | 本地开发环境（PostgreSQL + Redis + NATS + MinIO + etcd） |
| 优雅关闭 | 信号处理、连接释放顺序 |

#### S1.3 — 公共包与 Proto 定义（5 人天）

| 任务 | 说明 |
|------|------|
| common/model | 常量（CarrierCode, Technology, DeviceStatus, AlarmSeverity） |
| common/errorx | 业务错误码（1000-6999） |
| common/result | 统一 API 响应格式 |
| common/middleware | JWT、RBAC、API Key 中间件 |
| common/carrier | Carrier 接口、CarrierRegistry、空适配器桩 |
| pkg/tr069 | TR069 类型、事件码、CWMP 错误码 |
| pkg/soap | SOAP Envelope、预编译模板、流式解析 |
| api/proto/*.proto | 6 个 Proto 文件定义 |

#### S1.4 — goctl 代码生成（3 人天）

| 任务 | 说明 |
|------|------|
| .api 文件编写 | device.api、monitor.api、admin.api |
| goctl api 生成 | 3 个 API 网关框架代码 |
| goctl rpc 生成 | 5 个 RPC 服务 + 1 个 ACS 控制服务框架代码 |
| goctl model 生成 | 9 张可生成表的 CRUD |
| 生成脚本 | scripts/goctl-gen.sh |
| 验证 | 所有生成的服务可编译启动 |

#### S1.5 — ACS 引擎核心（10 人天）

| 任务 | 说明 |
|------|------|
| HTTP Server | net/http 服务器、TLS 支持 |
| SOAP 解码 | xml.Decoder 流式解析 Inform 消息 |
| SOAP 编码 | text/template 预编译 InformResponse |
| 会话状态机 | Redis-backed SessionStore、6 个状态转换 |
| 请求管线 | 限流 → 准入 → 认证 → 解析 → 分发 → 响应 |
| CPE 认证 | HTTP Digest + Basic |
| 设备级限流 | rate.Limiter per device |
| 全局准入 | atomic 并发计数器 |
| zRPC 服务端 | AcsControl gRPC 服务、etcd 注册 |
| Prometheus 指标 | acs_active_sessions, acs_inform_total 等 |
| Inform 处理 | Bootstrap/Periodic/ValueChange/Alarm 事件分发 |
| NATS 事件发布 | device.inform.* 事件 |

#### S1.6 — 设备管理基础（8 人天）

| 任务 | 说明 |
|------|------|
| device-rpc Logic | RegisterDevice、GetDevice、ListDevices、UpdateDevice |
| 设备状态机 | TransitionStatus、validTransitions 矩阵 |
| 心跳更新 | UpdateHeartbeat（Redis + PostgreSQL） |
| device-api Logic | 调用 device-rpc 的 REST 处理逻辑 |
| ACS → device-rpc | Bootstrap Inform → RegisterDevice 调用链验证 |
| 数据库迁移 | 000001_create_devices 迁移文件 |
| 端到端验证 | CPE Inform → ACS → device-rpc → PostgreSQL 全链路 |

---

### Phase 2: 核心功能（目标：完整设备管理 + 自动开站）

#### S2.1 — ACS RPC 全量方法（8 人天）

| 任务 | 说明 |
|------|------|
| GetParameterValues | SOAP 模板 + 响应解析 |
| SetParameterValues | 参数写入 + fault 处理 |
| GetParameterNames | 参数树遍历 |
| AddObject / DeleteObject | 实例管理 |
| Download / Upload | 文件传输（固件/配置/日志） |
| Reboot / FactoryReset | 设备控制 |
| 命令队列 | Redis Sorted Set CRUD |
| Connection Request | 主动拉设备连接 |
| zRPC QueueCommand | 管理面 → ACS 命令入队 |

#### S2.2 — 数据模型与配置（6 人天）

| 任务 | 说明 |
|------|------|
| config-rpc Logic | ResolveDataModel（三级回退） |
| 三级缓存 | L1 sync.Map + L2 Redis + L3 PostgreSQL |
| 缓存失效 | NATS 事件 + Redis INCR cache_version |
| 数据模型 CRUD | Import/Activate/Deprecate/List |
| 配置模板 | MatchTemplate（优先级匹配） |
| OUI 注册表 | goctl model CRUD |
| 数据库迁移 | data_model_definitions, config_templates, oui_registry |
| 种子数据 | cmcc_lte, cmcc_nr 默认数据模型 |

#### S2.3 — 运营商适配器（5 人天）

| 任务 | 说明 |
|------|------|
| CMCC 适配器 | 参数映射、设备验证、KPI 公式、开站步骤 |
| CTCC 适配器 | 参数映射、接口版本差异处理 |
| CUCC 适配器 | 5G NR 参数映射、北向接口格式 |
| 注册验证 | CarrierRegistry 自动加载 + 单元测试 |
| 集成测试 | 3 个运营商的 Inform 解析验证 |

#### S2.4 — 自动开站（6 人天）

| 任务 | 说明 |
|------|------|
| 开站状态机 | 7 状态转换 + provisioning_tasks 表 |
| Saga 编排器 | device-rpc 协调 config-rpc + acs-rpc |
| 模板匹配 | 优先级匹配（product_class → carrier → 全局） |
| 命令生成 | 从模板生成 Get/Set/Download/Reboot 命令序列 |
| 验证步骤 | GetParameterValues + 预期值比对 |
| 失败重试 | RetryProvisioning、错误记录 |
| 拓扑管理 | device_groups CRUD + 设备分配 |
| REST API | 开站任务列表/详情/重试端点 |

---

### Phase 3: 数据管线（目标：PM/告警/MR 全链路）

#### S3.1 — PM 采集与 KPI 引擎（7 人天）

| 任务 | 说明 |
|------|------|
| PM NATS 消费者 | pm.file.received 事件消费 |
| PM XML 解析器 | xml.Decoder 流式解析 |
| 批量写入 | pgx CopyFrom → TimescaleDB pm_counters |
| KPI 引擎 | 公式注册、计数器查询、表达式计算 |
| LTE KPI | RRC 成功率、E-RAB 成功率、PRB 利用率等 |
| NR KPI | 5G RRC 成功率、NGAP 成功率等 |
| pm-rpc Logic | QueryCounters、QueryKPI、GetKPIDefinitions |
| 时间聚合 | TimescaleDB 连续聚合（小时/天） |
| 数据库迁移 | pm_counters + kpi_values hypertable |

#### S3.2 — 告警管理（5 人天）

| 任务 | 说明 |
|------|------|
| 告警 NATS 消费 | alarm.raised 事件消费 |
| 告警引擎 | 去重 + 关联 + 抑制 三阶段处理 |
| 活跃告警 | alarms_active 表 + Redis 缓存 |
| 历史归档 | 清除时迁移到 alarms_history (TimescaleDB) |
| alarm-rpc Logic | ListActive、ListHistory、Acknowledge、Clear |
| 告警统计 | 按严重度/设备/时间段统计 |
| 数据库迁移 | alarms_active + alarms_history hypertable |

#### S3.3 — 测量报告（6 人天）

| 任务 | 说明 |
|------|------|
| MR NATS 消费者 | mr.file.received 事件消费 |
| MRO/MRS/MRE 解析 | 三种报告类型的 XML 解析器 |
| 批量写入 | measurement_reports hypertable |
| pm-rpc MR 查询 | QueryMeasurementReports |
| monitor-api MR 端点 | REST API |
| 数据库迁移 | measurement_reports hypertable |

---

### Phase 4: 北向与生产化（目标：OSS 对接 + 100K 级验证）

#### S4.1 — 北向/OSS 接口（5 人天）

| 任务 | 说明 |
|------|------|
| 北向 REST API | PM 导出、告警查询、配置快照 |
| 推送引擎 | NATS 订阅 → HTTP POST 到 OSS |
| API Key 认证 | 北向接口独立认证 |
| 全量同步 | 按需导出所有 PM/告警数据 |
| 联通北向适配 | 按联通规范实现特有接口 |

#### S4.2 — 管理功能（5 人天）

| 任务 | 说明 |
|------|------|
| admin-rpc Logic | 用户 CRUD、角色 CRUD、权限管理 |
| JWT 登录 | Login → 生成 JWT Token |
| 审计日志 | 所有变更操作记录 |
| 固件管理 | 版本注册、MinIO 文件上传 |
| 批量升级 | 升级任务创建 → Download 命令批量下发 |
| 互操作测试 | 测试用例执行框架 |
| 数据库迁移 | users, roles, permissions, audit_logs |

#### S4.3 — 集成测试与负载测试（8 人天）

| 任务 | 说明 |
|------|------|
| 跨服务集成测试 | docker-compose 启动全部 10 个服务 |
| 端到端测试 | Inform → 注册 → 开站 → PM 采集 → 告警 全流程 |
| 负载测试 | 模拟 100K 设备并发 Inform |
| ACS 压测 | 验证单实例 200 并发会话 |
| 服务间调用延迟 | zRPC 链路延迟基准测试 |
| 故障注入 | 单服务宕机、Redis 断连、NATS 不可用场景 |
| Bug 修复 | 测试发现的问题修复 |

#### S4.4 — K8s 部署与生产加固（5 人天）

| 任务 | 说明 |
|------|------|
| 10 个 Dockerfile | 多阶段构建优化 |
| K8s Manifests | Deployment + Service + HPA |
| Ingress 路由 | 路径分发到 3 个 API 网关 |
| TLS | ACS HTTPS + API TLS 终止 |
| 监控仪表盘 | Grafana 仪表盘（ACS/设备/PM/告警） |
| 告警规则 | Prometheus AlertManager 规则 |
| 文档 | 运维手册、部署指南 |

---

## 3. 风险评估

| # | 风险 | 严重度 | 概率 | 缓解策略 |
|---|------|:------:|:----:|---------|
| R1 | SOAP/XML 不被 goctl 支持 | 高 | 确定 | ACS 独立进程，不用 goctl，已在架构中解决 |
| R2 | 自动开站跨服务分布式事务 | 高 | 高 | Saga 编排模式 + 状态持久化 + 重试机制 |
| R3 | etcd 集群故障 | 高 | 低 | 3 节点 HA + 已注册服务有本地缓存 |
| R4 | TimescaleDB 不在 go-zero 生态 | 中 | 确定 | 手动 pgx/v5 集成，不用 goctl model |
| R5 | NATS 不在 go-queue 生态 | 中 | 确定 | 手动集成 NATS client，不用 go-queue |
| R6 | goctl 再生成覆盖手写代码 | 中 | 中 | 严格 gen/custom 分离 + Git diff 检查 |
| R7 | 10 个服务的联调调试困难 | 中 | 高 | 投资 Jaeger 分布式追踪 + 结构化日志 |
| R8 | 跨服务集成测试复杂 | 中 | 高 | docker-compose 一键启动 + 集成测试框架 |
| R9 | 团队 go-zero 学习曲线 | 中 | 中 | go-zero 中文文档丰富 + go-zero-looklook 参考 |
| R10 | Docker 镜像数量多，CI/CD 慢 | 低 | 高 | 并行构建 + Docker layer 缓存 |

---

## 4. 工时对比

### 4.1 开发工时

| 模块 | 模块化单体（人天） | go-zero 微服务（人天） | 差异 |
|------|:-----------------:|:-------------------:|:----:|
| 脚手架 & 基础设施 | 8 | 11 | +3（etcd + goctl 配置） |
| 公共包 & Proto | 10 | 8 | -2（goctl 生成减少手写） |
| ACS 引擎 | 18 | 18 | 0（不用 go-zero） |
| 设备管理 | 10 | 8 | -2（goctl model） |
| 数据模型 | 9 | 9 | 0（JSONB 需手写） |
| 运营商适配 | 5 | 5 | 0 |
| 自动开站 | 6 | 8 | +2（Saga 跨服务编排） |
| PM/KPI | 7 | 7 | 0（TimescaleDB 手写） |
| 告警管理 | 5 | 5 | 0 |
| 测量报告 | 6 | 6 | 0 |
| 北向/OSS | 5 | 5 | 0 |
| 管理/RBAC | 5 | 4 | -1（goctl model） |
| **开发小计** | **94** | **94** | **0** |

### 4.2 测试工时

| 测试类型 | 模块化单体（人天） | go-zero 微服务（人天） | 差异 |
|---------|:-----------------:|:-------------------:|:----:|
| 单元测试 | 25 | 25 | 0 |
| 集成测试 | 10 | 18 | +8（跨服务集成） |
| E2E 测试 | 5 | 8 | +3（10 个服务启动） |
| 负载测试 | 5 | 8 | +3（分布式场景） |
| **测试小计** | **45** | **59** | **+14** |

### 4.3 基础设施 & 部署

| 工作项 | 模块化单体（人天） | go-zero 微服务（人天） | 差异 |
|-------|:-----------------:|:-------------------:|:----:|
| Dockerfile | 3 | 5 | +2（10 vs 3） |
| docker-compose | 2 | 3 | +1 |
| K8s Manifests | 5 | 8 | +3（10 个 Deployment） |
| 监控/告警 | 3 | 5 | +2（更多服务指标） |
| **基础设施小计** | **13** | **21** | **+8** |

### 4.4 总计

| 维度 | 模块化单体 | go-zero 微服务 | 差异 |
|------|:---------:|:------------:|:----:|
| 开发 | 94 | 94 | 0 |
| 测试 | 45 | 59 | +14 |
| 基础设施 | 13 | 21 | +8 |
| **总计** | **152** | **174** | **+22 (+14%)** |

### 4.5 年维护成本

| 维度 | 模块化单体 | go-zero 微服务 | 差异 |
|------|:---------:|:------------:|:----:|
| 基础设施运维 | 15 | 30 | +15 |
| Bug 修复 | 30 | 40 | +10（跨服务排查） |
| 功能迭代 | 20 | 20 | 0 |
| 依赖升级 | 5 | 10 | +5 |
| **年维护总计** | **70** | **100** | **+30 (+43%)** |

---

## 5. 团队技能要求

### 5.1 必备技能

| 技能 | 说明 | 适用范围 |
|------|------|---------|
| Go 语言 | 1.22+ 泛型、goroutine、channel | 全部 |
| go-zero 框架 | goctl、zRPC、rest 模块、配置 | 管理面服务 |
| gRPC / Protobuf | Proto3 语法、流式 RPC | 服务间通信 |
| PostgreSQL | SQL、索引优化、JSONB、分区表 | 数据层 |
| TimescaleDB | 超表、连续聚合、压缩策略 | PM/告警时序 |
| Redis | Cluster、Sorted Set、Pub/Sub、TTL | 缓存/会话/队列 |
| NATS JetStream | Stream、Consumer Group | 事件驱动 |
| TR069/CWMP | SOAP/XML、Inform、RPC 方法 | ACS 引擎 |
| Docker / K8s | 容器化、Deployment、HPA、Ingress | 部署 |

### 5.2 推荐团队配置

| 角色 | 人数 | 职责 |
|------|:----:|------|
| 架构师/Tech Lead | 1 | 整体架构、ACS 引擎、技术决策 |
| 后端开发（go-zero） | 2-3 | 管理面服务开发、goctl 工作流 |
| 后端开发（数据管线） | 1 | PM/告警/MR Worker、TimescaleDB |
| DevOps | 1 | Docker/K8s、CI/CD、监控 |
| 测试 | 1 | 集成测试、E2E、负载测试 |
| **合计** | **6-7** | |

---

## 6. 里程碑检查点

| 里程碑 | Sprint | 验收标准 |
|--------|:------:|---------|
| M1 ACS 通信 | S1.5 | CPE 发送 Inform → ACS 解析 → 返回 InformResponse |
| M2 设备注册 | S1.6 | Bootstrap Inform → ACS → device-rpc → PostgreSQL 写入设备 |
| M3 完整 RPC | S2.1 | 管理面可通过 API 触发 Get/Set/Download/Reboot 等命令 |
| M4 自动开站 | S2.4 | 新设备 → 自动匹配模板 → 配置下发 → 验证 → 激活 |
| M5 数据管线 | S3.3 | PM 文件 → KPI 计算 → 告警处理 → 全部可通过 API 查询 |
| M6 生产就绪 | S4.4 | 100K 负载测试通过、K8s 部署、监控完善 |

---

## 7. 与模块化单体方案的关键差异总结

| 维度 | 模块化单体 | go-zero 微服务 |
|------|-----------|---------------|
| 首次交付时间 | ~152 人天 | ~174 人天（+14%） |
| 年维护成本 | ~70 人天 | ~100 人天（+43%） |
| 最小团队规模 | 2-3 人 | 5-7 人 |
| 部署复杂度 | 低（3 进程） | 高（10 进程 + etcd） |
| 独立扩展能力 | 有限 | 完整 |
| goctl 代码生成收益 | 无 | ~60% 管理面代码 |
| 适合初创阶段 | **是** | 否 |
| 适合大团队长期演进 | 否 | **是** |
| 适合直接面向 100万级 | 否（需拆分） | **是** |
