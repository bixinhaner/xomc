# 0001 — 模块化单体 + 独立 ACS 引擎，用 NATS，拒绝微服务 / go-zero / Kafka

## Status

Accepted（追认 retroactive）。决策于项目立项期落地，本目录成立时补记。

## Context

OMC 是面向运营商的小基站 TR-069/CWMP 无线网管，10 万基站起步、预留 100 万级扩展，含 10 个功能域（F01-F10）、43 项子功能。立项时需要在整体架构形态与若干基础选型上定调：

- **架构形态**：微服务 vs 单体。10 个功能域之间跨域交互密集（设备状态、配置、PM、告警、开站彼此联动），过早拆微服务会引入分布式事务与大量跨服务调用的复杂度。
- **ACS 引擎的位置**：TR-069 是 SOAP/XML over HTTP 的**有状态会话**协议（Inform → 事务循环 → Empty Response），并发模型、扩展节奏与管理面 REST API 截然不同。
- **Web/RPC 脚手架**：是否采用 go-zero 这类一体化框架做代码生成。go-zero 只面向 JSON/Protobuf，无法处理 ACS 的 SOAP/XML，且其生态不含 NATS，ACS 无法用 goctl 生成。
- **消息中间件**：事件流 / 异步任务分发选 NATS 还是 Kafka。

## Decision

1. **模块化单体**：F02-F10 管理面收敛在单个 Go 进程（`omcgo-app`），按功能域内聚为 `internal/` 下的扁平模块，模块间走同进程直接调用 + EventBus 解耦，不拆微服务。
2. **ACS 独立部署**：TR-069 引擎拆为独立进程 `omcgo-acs`（标准 CWMP 端口 `:7547`），与 app 通过 gRPC 通信，可独立水平扩展。再加一个后台工作进程 `omcgo-worker`。三个部署单元（app / acs / worker，端口见根 `CLAUDE.md §5`）。
3. **拒绝 go-zero**：管理面用 Gin，ACS 用 `net/http` stdlib（完全控制请求生命周期），不引入一体化代码生成框架。
4. **选 NATS（JetStream）而非 Kafka**：作为 EventBus 的多实例实现（`NATSEventBus`）与异步任务分发。单进程场景另有 `ChannelEventBus`（进程内 Go channel）。
5. **演进路线**：保持模块化单体到 10 万规模无虞；逼近 100 万规模时按功能域**渐进拆分**，EventBus 的双实现抽象（`internal/core/event/`）使拆分时事件链路无需重写。

允许一项受限例外：瞬时状态通知可使用 Core NATS 做非持久化 fan-out，但主题必须避开持久化 WorkQueue/EventBus，数据库必须保持权威来源。该通道不得承载命令、任务分发或任何需要历史重放的事件。

## Consequences

**正向**：

- 跨域调用是函数调用，无网络开销、无分布式事务；本地开发与调试简单。
- ACS 的有状态会话与高并发压力与管理面隔离，互不拖累，可单独扩缩容。
- 选型轻量：NATS Go 原生、运维成本远低于 Kafka，吞吐对当前规模足够。
- EventBus 接口 + 双实现（Channel / NATS）让"单体内事件"与"跨实例事件"用同一套 Subject 命名（`domain.action.detail`），为未来拆分留出无痛迁移路径。

**代价 / 约束**：

- app 进程承载 F02-F10 全部模块，任一模块的严重故障可能影响整进程 —— 必须靠 goroutine 隔离、模块边界纪律、EventBus 解耦来控制爆炸半径。
- 渐进拆分是未来必经的一步，模块间一旦出现隐性强耦合（直接引用而非走接口/事件）会加大拆分成本，需在 review 中持续守护"跨模块依赖走接口/EventBus"。
- 放弃 go-zero 的代码生成红利，handler/service/repository 分层需手写并靠规范保持一致。

> 后端速记见 `omcgo/CLAUDE.md §1`。三个部署单元与端口见根 `CLAUDE.md §5`，EventBus 见 `omcgo/CLAUDE.md §4.5`。
