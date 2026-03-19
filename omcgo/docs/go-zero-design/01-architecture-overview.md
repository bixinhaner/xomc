# 01 — 架构总览

> 基于 go-zero 微服务框架的 OMC 系统架构设计

---

## 1. 设计哲学

### 1.1 混合架构策略

采用**「混合架构」**：管理面使用 go-zero 微服务框架，ACS 引擎使用 `net/http` stdlib 独立实现。

| 层面 | 策略 | 理由 |
|------|------|------|
| ACS 引擎（F01） | `net/http` stdlib + zRPC 接入 | go-zero 不支持 SOAP/XML，goctl 对 ACS 完全无效 |
| 管理面（F02-F10） | go-zero 原生服务 | REST API、RPC 服务、中间件链均可利用 goctl 代码生成 |
| 进程间通信 | go-zero zRPC（封装 gRPC） | 统一 RPC 框架，服务注册发现一体化 |
| 消息队列 | NATS JetStream（手动集成） | 保持轻量，不引入 Kafka/JVM 依赖 |
| 服务治理 | go-zero 内置（熔断/限流/降载/超时） | 管理面服务开箱即用 |

### 1.2 核心原则

1. **务实优先**：go-zero 擅长的地方用 go-zero，不擅长的地方不强行使用
2. **ACS 独立**：ACS 引擎是系统核心（~40% 代码），不受框架约束
3. **goctl 最大化**：管理面 REST API 和 RPC 接口全部使用 goctl 生成
4. **共享数据库**：10 域功能存在强引用关系，不做 database-per-service
5. **渐进拆分**：初期可合并部分服务，规模增长后再独立扩展

---

## 2. 系统架构图

```
                        ┌────────────────────────────────────────────────────┐
                        │              外部系统（OSS / 网管中心）              │
                        └───────────────────────┬────────────────────────────┘
                                                │ REST/gRPC
                        ┌───────────────────────┴────────────────────────────┐
                        │                 go-zero API Gateway                 │
                        │          JWT 认证 | 自适应限流 | 熔断 | 降载        │
                        │     device-api(:8080)  monitor-api(:8081)           │
                        │     admin-api(:8082)                                │
                        └──┬──────────────┬──────────────┬───────────────────┘
                           │              │              │
              ┌────────────┴──┐    ┌──────┴──────┐   ┌──┴────────────┐
              │  device-rpc   │    │  config-rpc │   │  admin-rpc    │
              │  :50051       │    │  :50052     │   │  :50055       │
              │  F06 设备管理  │    │  F02 数据模型│   │  RBAC/审计    │
              │  F09 自动开站  │    │  配置模板   │   │  固件管理     │
              │  F07 网元直连  │    │             │   │  F10 互操作   │
              └───────┬───────┘    └──────┬──────┘   └──────┬────────┘
                      │                   │                  │
              ┌───────┴───────┐    ┌──────┴──────┐          │
              │   pm-rpc      │    │  alarm-rpc  │          │
              │   :50053      │    │  :50054     │          │
              │   F03 PM/KPI  │    │  F04 告警   │          │
              │   F05 MR      │    │             │          │
              └───────┬───────┘    └──────┬──────┘          │
                      │                   │                  │
         ┌────────────┴───────────────────┴──────────────────┘
         │                    zRPC（gRPC）
         │
┌────────┴──────────────────────────────────────────────────────────────┐
│                         acs-rpc (zRPC :50050)                         │
│                 ACS 控制接口: QueueCommand / SendConnReq              │
├───────────────────────────────────────────────────────────────────────┤
│                      ACS Engine（独立进程）                            │
│              net/http :7547(HTTP) / :7548(HTTPS)                      │
│              SOAP/XML 协议栈 | 会话状态机 | 命令队列                    │
│              Digest/Basic 认证 | 设备级限流 | 全局准入控制              │
└────────────────────────────┬──────────────────────────────────────────┘
                             │ TR069/CWMP (SOAP/XML over HTTP)
                    ┌────────┴────────┐
                    │   CPE 基站设备   │
                    │  (10万 ~ 100万)  │
                    └─────────────────┘

                         ┌─────────────────────┐
                         │   Worker 后台进程     │
                         │   PM/MR 文件解析      │
                         │   KPI 计算引擎        │
                         │   告警关联分析        │
                         └──────────┬──────────┘
                                    │
         ┌──────────┬───────────────┼───────────────┬──────────┐
    ┌────┴────┐ ┌───┴───┐ ┌────────┴───────┐ ┌────┴────┐ ┌───┴───┐
    │PostgreSQL│ │ Redis │ │NATS JetStream  │ │  MinIO  │ │ etcd  │
    │+Timescale│ │Cluster│ │(事件/消息)     │ │(文件)   │ │(服务  │
    │  DB      │ │       │ │               │ │         │ │ 发现) │
    └─────────┘ └───────┘ └───────────────┘ └─────────┘ └───────┘
```

---

## 3. 服务拆分策略

### 3.1 功能域到服务的映射

| 功能域 | 服务 | 部署单元 | 说明 |
|--------|------|---------|------|
| F01 南向接口（TR069） | acs-engine | 独立进程 | SOAP/XML 处理，不走 go-zero |
| F02 数据模型与配置 | config-rpc + device-api | RPC + API | goctl 生成 |
| F03 性能管理（PM/KPI） | pm-rpc + monitor-api + worker | RPC + API + Worker | KPI 计算在 worker |
| F04 告警管理 | alarm-rpc + monitor-api | RPC + API | goctl 生成 |
| F05 测量报告（MR） | pm-rpc + worker | RPC + Worker | 与 PM 合并 |
| F06 OMC-R 核心 | device-rpc + device-api | RPC + API | goctl 生成 |
| F07 网元直连 | device-rpc | RPC | 移动专有 |
| F08 北向/OSS 接口 | monitor-api | API | 北向推送 |
| F09 自动开站 | device-rpc | RPC | Saga 工作流 |
| F10 互操作测试 | admin-rpc + admin-api | RPC + API | goctl 生成 |

### 3.2 部署单元汇总（10 个）

| # | 部署单元 | 类型 | 框架 |
|---|---------|------|------|
| 1 | acs-engine | 独立服务 | net/http + zRPC |
| 2 | device-api | API 网关 | go-zero rest |
| 3 | monitor-api | API 网关 | go-zero rest |
| 4 | admin-api | API 网关 | go-zero rest |
| 5 | device-rpc | RPC 服务 | go-zero zRPC |
| 6 | config-rpc | RPC 服务 | go-zero zRPC |
| 7 | pm-rpc | RPC 服务 | go-zero zRPC |
| 8 | alarm-rpc | RPC 服务 | go-zero zRPC |
| 9 | admin-rpc | RPC 服务 | go-zero zRPC |
| 10 | worker | 后台进程 | go-zero ServiceGroup |

---

## 4. go-zero 特性与 OMC 需求映射

| OMC 需求 | go-zero 特性 | 覆盖度 | 备注 |
|---------|-------------|--------|------|
| REST API（~50 端点） | goctl api + rest 模块 | **完整** | 代码生成节省 ~2000 行 |
| 服务间 RPC | zRPC（gRPC 封装） | **完整** | 内置负载均衡、超时传播 |
| 服务注册与发现 | 内置 etcd 集成 | **完整** | 需新增 etcd 集群 |
| 自适应限流 | 内置 adaptive limiting | **管理面** | ACS 需自定义限流 |
| 熔断保护 | 内置 Google SRE 算法 | **管理面** | 防止级联故障 |
| 自适应降载 | 内置 CPU/并发感知降载 | **管理面** | 高负载自动拒绝请求 |
| JWT 认证 | 内置 middleware | **管理面** | 用于 REST API |
| 代码生成 | goctl（.api + .proto） | **~60%** | ACS 引擎无法使用 |
| 缓存自动化 | sqlc CachedConn | **部分** | 简单 CRUD 可用 |
| SOAP/XML 协议 | 不支持 | **零** | ACS 必须手写 |
| 有状态会话 | 不支持 | **零** | 会话状态机手动实现 |
| TimescaleDB 时序 | 不在生态中 | **零** | pgx/v5 手动集成 |
| 消息队列（NATS） | go-queue 仅支持 Kafka | **需适配** | NATS 手动集成 |
| 链路追踪 | 内置 OpenTelemetry | **完整** | 自动注入 trace |
| Prometheus 指标 | 内置自动采集 | **完整** | RPC/REST 自动打点 |

---

## 5. 与模块化单体方案的对比

| 维度 | 模块化单体（当前方案） | go-zero 微服务（本方案） |
|------|---------------------|------------------------|
| **部署单元** | 3 个（acs, app, worker） | 10 个 |
| **代码生成** | 无，全部手写 | ~60% 代码 goctl 生成 |
| **服务治理** | 手动集成 | 内置（限流/熔断/降载/超时） |
| **服务发现** | 不需要 | etcd（新增依赖） |
| **ACS 引擎** | net/http（完全控制） | 同左，通过 zRPC 接入 |
| **开发工时** | ~87 人天 | ~92 人天（+6%） |
| **测试复杂度** | 低（进程内调用） | 高（跨服务集成测试） |
| **运维复杂度** | 低（3 进程 + 4 中间件） | 高（10 进程 + 5 中间件） |
| **年维护成本** | 60-86 人天 | 100-155 人天（+70%） |
| **独立扩展** | 仅 ACS 可独立扩展 | 每个服务可独立扩展 |
| **技术栈学习** | Go 标准库 + 成熟组件 | 需额外学习 go-zero/goctl |
| **适合团队** | 1-5 人小团队 | 5+ 人团队 |
| **适合规模** | 10万起步，渐进扩展 | 直接面向 100万级 |

### 5.1 go-zero 方案的优势

1. **代码生成效率**：goctl 从 .api/.proto 生成 handler/types/routes/server/client，管理面开发效率提升 ~40%
2. **内置服务治理**：限流、熔断、降载、超时链式传播开箱即用
3. **独立扩展性**：热点服务（如告警处理、PM 查询）可独立扩展 Pod 数量
4. **故障隔离**：单服务故障不影响其他服务（如 pm-rpc 挂掉不影响设备管理）
5. **自动可观测**：zRPC/REST 自动注入 trace 和 metrics

### 5.2 go-zero 方案的劣势

1. **ACS 引擎无收益**：占 40% 代码量的核心模块无法使用 go-zero 特性
2. **运维复杂度翻倍**：10 个部署单元 vs 3 个，需要完善的 CI/CD
3. **新增 etcd 依赖**：需额外运维 3 节点 etcd 集群
4. **分布式事务**：自动开站工作流跨 device-rpc、config-rpc、acs-rpc，需 Saga 模式
5. **联调调试困难**：跨服务日志追踪、断点调试比进程内复杂得多
6. **年维护成本 +70%**：更多进程 = 更多 Docker 镜像 / K8s 配置 / 监控告警

### 5.3 适用场景建议

| 场景 | 推荐方案 |
|------|---------|
| 团队 1-3 人，100K 规模起步 | 模块化单体 |
| 团队 5+ 人，直接面向 100万级 | go-zero 微服务 |
| 团队有丰富 go-zero 经验 | go-zero 微服务 |
| 需要快速上线 MVP | 模块化单体 |
| 长期演进，需频繁独立部署 | go-zero 微服务 |

---

## 6. 消息队列决策：保留 NATS

| 维度 | NATS JetStream | Kafka（go-queue） |
|------|---------------|-------------------|
| 与 go-zero 集成 | 手动集成 | go-queue 原生支持 |
| 运维复杂度 | Go 单二进制 | JVM 生态 + ZooKeeper |
| 吞吐量 | 足够（10万基站） | 过剩 |
| 内存占用 | ~100MB | ~1GB+ |
| 启动时间 | 秒级 | 分钟级 |
| 本项目需求匹配度 | 高（事件驱动为主） | 中（大数据流处理更适合 Kafka） |

**结论**：保留 NATS JetStream，通过手动 wrapper 集成到 go-zero 生态。不引入 Kafka 的 JVM 依赖。

---

## 7. 技术栈总览

| 组件 | 选型 | 用途 |
|------|------|------|
| 语言 | Go 1.22+ | |
| 微服务框架 | go-zero v1.7+ | 管理面 REST/RPC |
| HTTP（ACS） | `net/http` stdlib | TR069 SOAP/XML |
| HTTP（管理面） | go-zero rest 模块 | REST API |
| RPC | go-zero zRPC | 服务间通信 |
| 服务发现 | etcd 3.5+ | go-zero 内置 |
| XML/SOAP | `encoding/xml` + `text/template` + `beevik/etree` | ACS 专用 |
| 消息队列 | NATS JetStream | 事件驱动 |
| 数据库 | PostgreSQL 16 + TimescaleDB | 关系型 + 时序 |
| 缓存 | Redis 7 | 会话/缓存/队列 |
| 对象存储 | MinIO | 文件存储 |
| 代码生成 | goctl | .api/.proto → 框架代码 |
| 日志 | go-zero 内置（或 zap） | 结构化日志 |
| 指标 | go-zero 内置 Prometheus | 自动采集 |
| 链路追踪 | go-zero 内置 OpenTelemetry | 分布式追踪 |
| 配置 | go-zero conf + YAML | 配置管理 |
| 数据库迁移 | golang-migrate/migrate/v4 | Schema 版本 |
| 参数验证 | go-playground/validator/v10 | 结构体校验 |
| 定时任务 | robfig/cron/v3 | 周期任务 |
