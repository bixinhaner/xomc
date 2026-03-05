# OMC Go 详细设计文档索引

> 本目录包含 OMC Go 项目全部 21 个详细设计文档，按 4 阶段实施路线图组织。
> 每个文档可直接指导对应模块的编码实现。

---

## 文档总览

| 编号 | 文档 | 功能域 | 实施阶段 | 说明 |
|------|------|--------|---------|------|
| 01 | [项目脚手架](01-project-scaffolding.md) | — | Phase 1 | Go module、目录结构、Makefile、cmd 入口、配置管理 |
| 02 | [基础设施层](02-infrastructure-layer.md) | — | Phase 1 | PostgreSQL/Redis/NATS/MinIO 连接、健康检查、优雅关闭 |
| 03 | [公共领域模型](03-common-domain-models.md) | — | Phase 1 | Device/Alarm/Parameter 等领域模型、错误类型、中间件 |
| 04 | [事件总线](04-event-bus.md) | — | Phase 1 | EventBus 接口、Channel/NATS 双实现、事件目录 |
| 05 | [运营商抽象层](05-carrier-abstraction.md) | — | Phase 2 | Carrier 接口、CarrierRegistry、三家运营商适配器 |
| 06 | [TR069 协议库](06-tr069-protocol-library.md) | F01 | Phase 1 | pkg/tr069 类型、CWMP 错误码、SOAP 模板、XML 工具 |
| 07 | [ACS 引擎](07-acs-engine.md) | F01 | Phase 1 | HTTP 服务器、会话状态机、RPC 方法、命令队列、ConnReq |
| 08 | [数据模型与配置](08-data-model-config.md) | F02 | Phase 2 | 三级回退解析、三级缓存、数据模型导入、配置模板 |
| 09 | [设备管理](09-device-management.md) | F06 | Phase 1 | 设备注册、状态机、心跳检测、拓扑管理 |
| 10 | [自动开站](10-provisioning.md) | F09 | Phase 2 | 开站状态机、模板匹配、配置下发编排、批量开站 |
| 11 | [性能管理](11-performance-management.md) | F03 | Phase 3 | PM 采集管线、XML 解析、KPI 计算、时间聚合 |
| 12 | [告警管理](12-alarm-management.md) | F04 | Phase 3 | 告警接收/去重/关联/生命周期/转发/存储 |
| 13 | [测量报告](13-measurement-reports.md) | F05 | Phase 3 | MR 采集、MRO/MRS/MRE 解析、存储 |
| 14 | [软件管理](14-software-management.md) | F06 | Phase 4 | 固件版本管理、远程升级编排、批量升级 |
| 15 | [北向/OSS 接口](15-northbound-oss.md) | F08 | Phase 4 | PM/告警/配置北向接口、数据推送、全量/增量同步 |
| 16 | [网元直连接口](16-ne-direct.md) | F07 | Phase 4 | 网元直连接口（移动专有） |
| 17 | [用户管理与 RBAC](17-admin-rbac.md) | F06 | Phase 4 | 用户管理、角色权限、JWT 认证、操作审计 |
| 18 | [互操作测试](18-interop-testing.md) | F10 | Phase 4 | 一致性测试框架、协议/数据模型验证 |
| 19 | [可观测性](19-observability.md) | — | Phase 1 | zap 日志、Prometheus 指标、OpenTelemetry 追踪 |
| 20 | [部署架构与运维](20-deployment.md) | — | Phase 1→4 | Docker、docker-compose、K8s、HPA/KEDA、规模化 |
| 21 | [数据库迁移](21-database-migrations.md) | — | Phase 2 | golang-migrate 策略、迁移文件清单、种子数据 |

---

## 实施阶段映射

### Phase 1 — 基础建设

> 目标：ACS 引擎能接收 Inform 并注册设备

```
01-project-scaffolding     项目脚手架、构建工具
02-infrastructure-layer    基础设施连接层
03-common-domain-models    公共领域模型与错误类型
04-event-bus               事件总线（进程内 channel）
06-tr069-protocol-library  TR069 协议类型与 SOAP 工具
07-acs-engine              ACS 引擎核心
09-device-management       设备注册与状态管理
19-observability           日志/指标/追踪基础
20-deployment (20a)        Dockerfile + docker-compose
```

### Phase 2 — 核心功能

> 目标：完整设备管理和自动开站流程

```
05-carrier-abstraction     运营商抽象层（CMCC 优先）
08-data-model-config       数据模型三级回退、配置模板
10-provisioning            自动开站状态机
21-database-migrations     全量迁移文件
```

### Phase 3 — 数据管线

> 目标：PM/告警/MR 数据全链路

```
11-performance-management  PM 采集 → KPI 计算 → 时序存储
12-alarm-management        告警全生命周期管理
13-measurement-reports     MR 文件采集与解析
```

### Phase 4 — 北向与规模化

> 目标：OSS 对接、10 万级验证、生产加固

```
14-software-management     固件升级管理
15-northbound-oss          北向 OSS 数据接口
16-ne-direct               网元直连（移动专有）
17-admin-rbac              用户管理与权限控制
18-interop-testing         互操作测试框架
20-deployment (20b/20c)    K8s 部署 + HPA/KEDA
```

---

## 功能域 → 文档交叉索引

| 功能域 | 子功能 | 对应文档 |
|--------|--------|---------|
| **F01** 南向接口 | TR069 协议处理 | [06](06-tr069-protocol-library.md) |
| | ACS 引擎 | [07](07-acs-engine.md) |
| **F02** 数据模型与配置 | 数据模型管理、配置模板 | [08](08-data-model-config.md) |
| **F03** 性能管理 | PM 采集、KPI 计算 | [11](11-performance-management.md) |
| **F04** 告警管理 | 告警接收/处理/转发 | [12](12-alarm-management.md) |
| **F05** 测量报告 | MRO/MRS/MRE 解析 | [13](13-measurement-reports.md) |
| **F06** OMC-R 核心 | 设备管理 | [09](09-device-management.md) |
| | 固件升级 | [14](14-software-management.md) |
| | 用户管理/RBAC | [17](17-admin-rbac.md) |
| **F07** 网元直连 | 直连接口 | [16](16-ne-direct.md) |
| **F08** 北向/OSS | OSS 数据接口 | [15](15-northbound-oss.md) |
| **F09** 自动开站 | 开站流程编排 | [10](10-provisioning.md) |
| **F10** 互操作测试 | 一致性验证 | [18](18-interop-testing.md) |

---

## 文档间依赖关系

```
01-scaffolding ─────────────────────────────────────────────
  │
  ├── 02-infrastructure ──┐
  ├── 03-domain-models ───┤
  ├── 06-tr069-library    │
  ├── 19-observability    │
  └── 20-deployment       │
                          │
  02 + 03 ──────────→ 04-event-bus
  03 ───────────────→ 05-carrier
                          │
  02 + 04 + 06 ────→ 07-acs-engine
  02 + 03 + 05 ────→ 08-data-model
  02 + 03 + 04 + 07 → 09-device
                          │
  07 + 08 + 09 ────→ 10-provisioning
  02 + 04 + 05 ────→ 11-pm
  02 + 04 + 07 ────→ 12-alarm
  02 + 04 ─────────→ 13-mr
  07 + 09 ─────────→ 14-software
  11 + 12 + 08 ───→ 15-northbound
  05 + 09 ─────────→ 16-ne-direct
  02 + 03 ─────────→ 17-admin-rbac
  07 + 08 ─────────→ 18-interop
  02 ──────────────→ 21-migrations
```

---

## 各文档标准结构

每个详细设计文档遵循以下标准结构（按模块需要裁剪）：

1. **概述** — 模块定位、核心职责、交互关系
2. **接口设计** — Go interface、REST API、gRPC 定义
3. **数据模型** — 结构体、DB Schema、Redis 数据结构
4. **详细设计** — 核心算法/流程、状态机、序列图
5. **运营商差异** — 三家差异点及适配方式
6. **实施子阶段** — 可独立交付的小步骤
7. **文件清单** — 需创建的源代码文件路径
8. **参考** — 关联规范文档、feature docs

---

## 关联文档

| 文档 | 路径 | 关系 |
|------|------|------|
| 后端架构设计 | `doc/architecture/backend-design.md` | 上游：所有详细设计的架构依据 |
| 功能索引 | `doc/功能索引.md` | 上游：43 项子功能完整清单 |
| 功能域详情 | `doc/features/01~10-*.md` | 上游：各功能域需求详情 |
| 接口拓扑 | `doc/architecture/interface-topology.md` | 上游：接口协议栈与数据流 |
| 框架选型 | `doc/architecture/framework-comparison.md` | 参考：为何不用 go-zero |
| CLAUDE.md | `CLAUDE.md` | 参考：编码规范、命名规范、目录结构 |
