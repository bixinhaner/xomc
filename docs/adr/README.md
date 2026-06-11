# Architecture Decision Records (ADR)

> 本目录记录 OMC 项目**难以逆转、且有真实权衡**的架构决策。单上下文仓库：根级 `CONTEXT.md`（术语表）+ 本目录（决策记录）。详见 `docs/agents/domain.md`。

## 什么是 ADR

ADR（Architecture Decision Record）是一篇短文，记录**一个**架构决策的来龙去脉。判断一件事该不该写 ADR，看两条标准是否**同时**满足：

| 标准 | 含义 |
|------|------|
| **难以逆转** | 改回去要动很多代码 / 数据 / 部署，不是改一行配置能撤销的（选型、分层、协议、表结构语义） |
| **有真实权衡** | 存在被否的合理替代方案，当时是在多个有效选项中做了取舍，而非唯一解 |

只满足其一的不写：纯实现细节（变量命名、函数拆分）、行业既定无悬念的选择（用 UUID 主键）、随时可改的配置（超时值），都不进 ADR。

## 与本仓其它决策载体的分工

| 载体 | 装什么 |
|------|--------|
| `docs/adr/`（本目录） | 跨模块、长期生效、难逆转的**架构级**决策 |
| `omcgo/CLAUDE.md §1` | 后端架构决策**速记**（一句话结论 + 理由表），完整论证回链本目录 |
| `docs/design/*` | 单个特性的落地设计文档（含被拍板/被推翻的过程） |
| `docs/ref/*` | 踩坑案例库、历史考古、代码索引 |
| GitHub Issues | 活任务源（实现追踪），非决策记录 |

## ADR 格式

每篇固定五段：**Title / Status / Context / Decision / Consequences**。Status 取值 `Proposed` / `Accepted` / `Deprecated` / `Superseded by NNNN`。

下面 4 条为**回溯（retroactive）ADR** — 决策早已落地并在代码中生效，本目录成立时补记，Status 标 `Accepted（追认）` + 大致拍板日期。

## 索引

| 编号 | 决策 | Status | 大致日期 |
|------|------|--------|---------|
| [0001](0001-modular-monolith-not-microservices.md) | 模块化单体 + 独立 ACS 引擎，用 NATS，拒绝微服务 / go-zero / Kafka | Accepted（追认） | 项目立项期 |
| [0002](0002-squirrel-pgx-not-orm.md) | 数据库访问用 Squirrel + pgx，不用 ORM | Accepted（追认） | 项目立项期 |
| [0003](0003-t0098-data-dictionary-not-datamodel-fallback.md) | T-0098 数据字典重构：下线旧 datamodel 三级回退，改走 product 装配件 + ParamModel 字典 + Translator | Accepted（追认） | 2026-05-07/08 |
| [0004](0004-three-library-single-dir-sidecar.md) | 三库导入 XML 用单目录 + sidecar，不用 builtin/custom 双目录物理隔离 | Accepted（追认） | 2026-06-04 |
| [0005](0005-acs-horizontal-scale-stateless.md) | ACS 横向扩展去进程态：4 类会话副作用状态（准入计数 / 设备孤儿会话 / CR-URL / Digest nonce）全量接共享 Redis（Option B），不靠会话亲和 | Accepted | 2026-06-11 |
