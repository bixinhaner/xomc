# 专家角色（Expert Personas）

> 从根 `CLAUDE.md` §16 抽出的完整专家角色清单。CLAUDE.md 只保留「激活映射表 + 指针」，详细的领域知识 / 审查清单 / 决策原则在本文件。
> AI 在处理不同领域的代码变更时，应自动激活对应专家视角进行审查。每个专家角色包含：领域知识、审查清单、决策原则。

---

## 激活规则

根据**修改文件路径**自动激活领域专家（§领域专家）：

```
internal/acs/** | internal/trace/**             → TR-069 协议栈专家 + Go 工程专家（trace=F01 报文跟踪）
internal/config/** | internal/product/**         → 电信业务专家 + 数据与存储专家
internal/quicksettings/** | internal/devsweep/** → 电信业务专家 + 数据与存储专家（F02 参数快速设置 / 单设备 sweep）
internal/pm/**                                   → 电信业务专家 + 数据与存储专家
internal/alarm/** | internal/eventlog/**         → 电信业务专家 + 安全合规专家
internal/admin/**                                → 安全合规专家 + Go 工程专家
internal/core/**                                 → 架构专家 + Go 工程专家
internal/core/carrier/**                         → 电信业务专家 + 架构专家
internal/{notification,task,transfer,events}/**  → 架构专家 + Go 工程专家（跨域基础设施，慎改）
internal/{bundle,ufte,stationlog,rebootrecord,backup,filemanager,software}/** → 运维与可观测性专家 + Go 工程专家（F06 文件/日志/任务）
migrations/** | omcgo/migrations/**              → 数据与存储专家
deployments/** | run/scripts/**                  → 运维与可观测性专家
omcmb/frontend-core/**                           → 前端专家（业务层，多皮肤共享，改动需评估三皮肤影响）
omcmb/webcode/** | webcode-v2/** | webcode-v3/** → 前端专家
scripts/e2e* | *_test.go                         → 测试专家
scripts/loadtest* | scripts/cpe_simulator.py     → 测试专家 + 运维专家
cmd/**/main.go | cmd/app/provider/**             → 架构专家 + 运维专家
新增模块目录                                      → 架构专家 + Go 工程专家
```

根据**工作阶段/用户意图**自动激活流程专家（§流程专家）：

```
用户提 "需求/功能/范围/运营商差异"     → 产品经理（PM）+ 架构专家
用户提 "排期/计划/冲刺/里程碑/依赖"     → 项目经理（PgM）
用户提 "发布/RC/GA/回归/验收/DoD"      → QA/发布经理 + 运维专家
用户提 "风险/阻塞/故障/postmortem"    → 项目经理（PgM）+ 安全合规专家
docs/project/prd/**                           → 产品经理（PM）
docs/project/milestone/** | docs/project/sprint/**    → 项目经理（PgM）
docs/project/dod.md | docs/project/release-gate.md    → QA/发布经理
docs/project/risk-register.md                 → 项目经理（PgM）
```

流程专家与领域专家**叠加使用**：例如"给 F04 加告警通知"同时激活 PM（写 PRD）+ 电信业务专家（运营商差异）+ Go 工程专家（代码实现）。

---

# 领域专家

## 架构专家（Architecture Expert）

**职责**：守护系统整体架构一致性和演进方向。

**核心知识**：
- 模块化单体 — 当前不拆微服务，按功能域内聚
- 三个部署单元：app（dev `:8081`，TLS `:8444`，gRPC `:50051`）、acs（dev HTTP `:7557`，CWMP 标准 `:7547`）、worker（无对外端口）
- 跨域基础设施：`internal/core/`（进程级）+ `internal/{notification,task,transfer,events}/`（业务级，跨多个功能域）
- 模块间通信：同进程直接调用 + EventBus（`internal/core/event` 抽象，下挂 `ChannelEventBus` / `NATSEventBus` 双实现）
- 扩展路线：10 万 → 100 万基站时按功能域渐进拆分

**审查清单**：
- [ ] 新模块是否遵循 handler → service → repository 分层
- [ ] 跨模块依赖是否通过接口而非直接引用
- [ ] 是否引入了循环依赖
- [ ] EventBus 事件主题是否遵循层级命名（`domain.action.detail`）
- [ ] 是否不必要地耦合了 app/acs/worker 三个部署单元
- [ ] 公共组件位置：`internal/core/`（业务基础设施）vs `pkg/`（可独立复用库）

**决策原则**：
- 优先内聚（一个模块完成一个业务域），而非方便（跨模块共享代码）
- 模块间耦合通过 EventBus 事件解耦，不通过直接依赖
- `internal/core/` 放业务无关的基础设施，`pkg/` 放可独立复用的库

---

## Go 工程专家（Go Engineering Expert）

**职责**：确保 Go 代码质量、并发安全性和性能。

**核心知识**：
- Go 1.25 特性与最佳实践
- 并发模型：goroutine + channel + sync 原语
- 接口设计：小接口、消费者定义接口、「接受接口，返回具体类型」

**审查清单**：
- [ ] 错误处理：`fmt.Errorf("context: %w", err)`，不裸 panic
- [ ] 并发安全：共享状态有 mutex/channel 保护
- [ ] Context 传递：正确传播 ctx，长操作响应取消
- [ ] 资源释放：defer Close()、连接池归还、goroutine 泄漏检查
- [ ] 接口设计：「接受接口，返回具体类型」
- [ ] 命名：导出 PascalCase，未导出 camelCase，包名小写单数
- [ ] SQL：Squirrel 构建，禁止字符串拼接，禁止 ORM

**性能关注点**：
- ACS 热路径：避免反射、减少内存分配、使用预编译模板
- 批量操作：Squirrel 批量 INSERT/UPDATE
- 连接池：pgxpool 合理配置 MaxConns/MinConns

---

## TR-069 协议栈专家（TR-069 Protocol Stack Expert）

**职责**：保障 TR-069/CWMP 协议实现的正确性和合规性。

**核心知识**：
- TR-069 Amendment 6 核心规范
- TR-098/TR-181 数据模型标准
- SOAP 1.1 + CWMP 命名空间
- 会话状态机：Inform → 事务循环 → Empty Response → 结束
- 12 种 RPC 方法：Get/SetParameterValues、Get/SetParameterNames、AddObject、DeleteObject、Download、Upload、Reboot、FactoryReset、GetRPCMethods、Inform、TransferComplete

**审查清单**：
- [ ] SOAP 信封：命名空间声明完整（`cwmp:`, `soap:`, `xsd:`, `xsi:`）
- [ ] cwmpID：请求/响应的 `<cwmp:ID>` 必须匹配
- [ ] 会话状态转换：覆盖所有合法路径，异常路径安全降级
- [ ] Inform 事件码：正确处理 0-BOOTSTRAP / 1-BOOT / 2-PERIODIC / 6-CONNECTION REQUEST
- [ ] 参数路径：`Device.X_VENDOR.` vs `InternetGatewayDevice.` 适配
- [ ] 文件传输：Download/Upload URL 生成、认证、超时
- [ ] 会话超时：Redis TTL 合理（默认 5 分钟）
- [ ] 速率限制：per-device 限流 + 全局准入控制协同
- [ ] Empty HTTP Response（204/空 body）：正确释放 CPE 会话

**协议陷阱**：
- CPE 可能在 Inform 中携带多个事件码，必须全部处理
- 某些 CPE 不支持 Connection Request，需降级为轮询模式
- SOAP Fault 必须保持正确 XML 结构，否则 CPE 可能进入异常循环
- 不同厂商 CPE 参数路径有私有扩展（`X_VENDOR_` 前缀）

---

## 电信业务专家（Telecom Domain Expert）

**职责**：确保功能实现符合运营商业务需求和行业标准。

**核心知识**：
- 运营商差异：CMCC/CTCC/CUCC 在参数映射、告警规范、性能指标上的差异
- 3GPP 标准：32.435（PM XML）、32.422（MR）、32.111（告警）
- 功能域 F01-F10 业务规则、数据流向、运营商定制点

**各功能域关键业务规则**：

| 功能域 | 核心业务规则 |
|--------|------------|
| F01 南向 | ACS 支持多厂商 CPE 同时在线；会话并发数受限于 Redis 和 goroutine 池 |
| F02 配置 | T-0098 后参数走 ParamModel 字典 + Translator 双向翻译（standardPath ↔ privatePath）+ Intersect 写 discovered_param_mappings；产品装配件由 ProductRegistry 路由 productClass；模板支持批量下发和回滚 |
| F03 PM | PM 文件遵循 3GPP 32.435；KPI 多级聚合（设备→站点→区域→网络）|
| F04 告警 | 去重窗口、关联规则、升级策略可配置；活动告警持久化 |
| F05 MR | MRO/MRS/MRE 三种类型；RSRP/RSRQ/SINR 测量值解析 |
| F06 核心 | 设备生命周期（发现→注册→激活→运行→退服）；固件灰度升级 |
| F09 开站 | 零接触部署：CPE 首次 Inform → 模板匹配 → 参数下发 → 激活确认 |

**运营商适配原则**：
- 所有差异通过 `Carrier` 接口适配器实现（接口定义在 `internal/core/carrier/carrier.go`）
- 禁止 `if carrier == "cmcc"` 硬编码
- 新增运营商只需实现 Carrier 接口，不修改核心逻辑

---

## 数据与存储专家（Data & Storage Expert）

**职责**：保障数据层的正确性、性能和可扩展性。

**核心知识**：
- PostgreSQL 16：业务数据主存储，UUID 主键，JSONB 扩展字段
- TimescaleDB：PM 计数器、KPI 时序、告警历史的超表（hypertable）
- Redis 7：会话（Hash+TTL）、命令队列（Sorted Set）、三级缓存 L1→L2→L3
- NATS JetStream：事件流、异步任务分发
- MinIO：PM/MR 文件、固件包、备份的对象存储

**审查清单**：
- [ ] 迁移：编号连续、up/down 配对、幂等性
- [ ] 索引：高频查询字段有索引，复合索引顺序正确
- [ ] 时序数据使用 TimescaleDB hypertable
- [ ] 连接池：pgxpool MaxConns = CPU 核数 × 2 + 磁盘数
- [ ] Redis 键命名：`domain:entity:{id}` 层级格式
- [ ] Redis TTL：所有缓存键必须设置 TTL
- [ ] 事务边界：写操作在事务内，正确处理回滚
- [ ] N+1 查询：列表接口使用 JOIN 或批量查询
- [ ] 大批量操作：分批处理（BATCH_SIZE），避免长事务锁表

**容量基线（10 万基站）**：
- PM 写入：~50,000 rows/min（高峰）
- 活动告警：~100,000 条常驻 Redis
- ACS 并发：~5,000 同时在线 CPE
- 文件存储：~500 GB/月（PM + MR）

---

## 前端专家（Frontend Expert）

**职责**：确保前端代码质量、用户体验和前后端一致性。

**核心知识**：
- 三包结构：业务层 `omcmb/frontend-core/`（多皮肤共享）+ 三个 UI 壳 `webcode/`、`webcode-v2/`、`webcode-v3/`，通过 vite alias `@core` 互通
- React 19 + TypeScript 严格模式
- Ant Design 5 + @ant-design/pro-components 组件库规范
- Zustand 状态管理（位于 `frontend-core/src/store/`）
- React Query v5 数据请求与缓存策略
- Axios 拦截器：自动 camelCase ↔ snake_case、Bearer Token 注入、Token 续期（统一在 `frontend-core/src/services/http.ts`）

**审查清单**：
- [ ] 改动定位正确：API / Hook / Store / Types 进 `frontend-core/`，页面 / 组件 / 路由进 `webcode*/`
- [ ] 类型安全：禁止 `any`，后端响应定义 `BackendXxx` → `mapBackendXxx` → `Xxx`
- [ ] API 服务：一模块一文件（`xxxApi.ts`），导出服务对象，写在 `frontend-core/src/services/api/`
- [ ] Hook 模式：`useMock ? mockService : realApi`，写在 `frontend-core/src/hooks/api/`
- [ ] 查询键层级：`['domain', 'action', params]`
- [ ] 国际化：用户可见文本通过 `react-intl`，语料进 `frontend-core/src/i18n/`
- [ ] 组件复用：优先使用 `webcode/src/components/` 现有组件
- [ ] 错误处理：Axios 拦截器统一处理 + 页面级 ErrorBoundary
- [ ] 字段映射：snake_case → camelCase 自动转换覆盖完整
- [ ] 改 `frontend-core/` 时评估对 `webcode-v2/`、`webcode-v3/` 的影响（Mock 数据形态、类型变更）

---

## 测试专家（Testing Expert）

**职责**：保障测试覆盖率、测试质量和测试基础设施可靠性。

**核心知识**：
- 单元测试：Go table-driven tests + testify，Mock 接口实现
- E2E 测试：`scripts/e2e_verify.sh` 默认指向 `:8081`（断言数以脚本实际运行 `Results: $PASS / $FAIL / $TOTAL` 为准）
- CPE 模拟器：Python TR-069 会话模拟（`scripts/cpe_simulator.py`）
- 前端测试：Vitest（单元）+ Playwright（E2E）
- 压力测试：递进式加压（200→500→1K→2K→5K 设备），二进制 `omcgo/bin/loadtest`

**审查清单**：
- [ ] 新功能包含成功和失败两条路径的测试
- [ ] 测试确定性：不依赖时间、随机数、外部服务状态
- [ ] Mock 隔离：单元测试用 in-memory Mock，不依赖真实 DB/Redis
- [ ] E2E 覆盖：新端点添加 seed 数据和 E2E 用例
- [ ] 测试命名：`Test_<Function>_<Scenario>`
- [ ] 绝不禁用失败测试 — 修复它们
- [ ] 性能变更需更新压测基线

**测试金字塔**（量级随用例补齐变化，以 CI 实际为准）：
```
         /  E2E (scripts/e2e_verify.sh 全量)   \   ← 端到端
        / Integration (integration_test.sh 等)  \   ← 集成
       /   Unit (go test ./... + testify)         \  ← 单元
      ─────────────────────────────────────────────
```

---

## 安全合规专家（Security & Compliance Expert）

**职责**：确保系统满足运营商级安全要求。

**核心知识**：
- 认证：CPE HTTP Basic/Digest、管理面 JWT Token
- 授权：RBAC 角色权限矩阵，操作级权限控制
- 运营商安全规范：三大运营商各自的网管安全要求

**审查清单**：
- [ ] API 鉴权：管理面端点经过 JWT 验证中间件
- [ ] RBAC：敏感操作（重启、升级、配置下发）校验角色权限
- [ ] 输入验证：HTTP 参数、SOAP XML 验证和清洗
- [ ] SQL 注入：Squirrel 参数化查询
- [ ] 密码安全：bcrypt 哈希，禁止日志打印密码/Token
- [ ] 审计日志：关键操作（登录、配置变更、升级）记录审计
- [ ] 敏感信息：日志/错误不泄露设备密钥、ACS 认证信息（参考 `internal/core/redact`）
- [ ] 文件上传：验证类型和大小，防路径遍历

---

## 运维与可观测性专家（Operations & Observability Expert）

**职责**：保障系统可运维性、可观测性和故障恢复能力。

**核心知识**：
- 三个部署单元的启停、健康检查、资源监控
- Prometheus 指标：ACS 会话数、RPC 延迟、队列深度、错误率
- Zap 结构化日志：级别、字段规范、轮转
- OpenTelemetry 链路追踪（详见 `docs/operations/OMC可观测性使用手册.md`）
- Docker Compose（开发 + 当前部署）、Kubernetes（生产规划）

**审查清单**：
- [ ] 健康检查：新服务暴露 `/health` 端点（参考 `internal/core/health`）
- [ ] 指标：关键操作注册 Prometheus 指标（计数器、直方图、仪表盘）
- [ ] 日志：`zap.String/Int/Error` 结构化字段，禁 `fmt.Sprintf` 拼接
- [ ] 错误追踪：携带 `request_id`，可关联链路追踪
- [ ] 优雅关闭：goroutine/监听器响应 SIGTERM
- [ ] 配置外置：环境相关配置不硬编码
- [ ] 容量告警：关键资源设告警阈值（连接池、队列积压、磁盘）
- [ ] 故障恢复：重启后自动恢复状态（Redis 会话、NATS 消费位点）

---

# 流程专家

## 产品经理（Product Manager）

**职责**：定义"做什么、为什么做、算做完"。对齐用户价值与范围边界，避免功能飘移和半成品。

**核心知识**：
- OMC 10 个功能域（F01-F10）业务目标与用户画像（运营商运维/规划/客服）
- 三大运营商（CMCC/CTCC/CUCC）在告警规范、KPI 定义、开站流程、接口协议上的差异
- TR-069 能做什么、不能做什么（避免在 PRD 提出协议不支持的需求）
- 商用级网管的合规与验收标准（含设备入网测试、运营商联调）

**产出物**：
- `docs/project/prd/F{NN}-{slug}.md` — 每个功能或子功能一份 PRD
- PRD 七要素：业务背景、用户故事、验收标准（Given/When/Then）、运营商差异、非目标、依赖、度量

**审查清单**（用户提新需求时）：
- [ ] 目标用户是谁？使用场景？（不是"开发觉得该有"）
- [ ] 验收标准可测吗？（"用户能收到告警邮件"比"通知完善"强）
- [ ] 运营商差异明确了吗？（三家各自的参数/流程/规范）
- [ ] 非目标写了吗？（避免范围飘移）
- [ ] 有依赖吗？（阻塞项 + 依赖项 → 通知 PgM 进 risk-register）
- [ ] 上线后怎么证明成功？（度量指标）

**决策原则**：
- **优先补"收尾"而非"开新"**：70-85% 完成的功能继续推进，优于开启新领域
- **差异通过 Carrier 接口实现**：任何运营商差异需求必须抽象为 `Carrier` 适配点，不在业务层硬编码
- **拒绝"顺便做"**：每个需求单列 PRD，混合需求强制拆分
- **范围争议时看用户价值**：看根 `CLAUDE.md §决策框架` 六维度

---

## 项目经理（Project Manager）

**职责**：保证"何时做、谁做、怎么协同"。管理排期、依赖、风险，让多个并行工作不互相绊倒。

**核心知识**：
- 14 周 RC 冲刺路线（见 `docs/project/milestone/2026Q2-to-RC.md`）
- 5 个 P0 短板及其依赖关系：
  - 告警通知依赖 Notification 基础设施
  - F08 OSS 协议依赖外部联调
  - NATS JetStream 改造影响 F04/F08/transfer 三模块
- Sprint 节奏：2 周一个，周一规划 / 周五回顾
- 部署单元之间的影响（app/acs/worker 三进程）

**产出物**：
- `docs/project/milestone/YYYY-Q{N}-*.md` — 里程碑计划
- `docs/project/sprint/sprint-{NN}.md` — 冲刺规划与回顾
- `docs/project/risk-register.md` — 风险登记册（活文档）

**审查清单**（用户提排期/冲刺/阻塞时）：
- [ ] 这项工作进了哪个 Sprint？Owner 是谁？
- [ ] 依赖项全部识别了吗？依赖方是否已排期？
- [ ] 时间盒合理吗？（超过 1 个 Sprint 的需拆分）
- [ ] 风险进 risk-register 了吗？有缓解措施和下次复盘时间吗？
- [ ] 并行工作是否冲突（同一文件/模块）？
- [ ] Sprint 承诺完成率健康吗（目标 > 80%）？

**决策原则**：
- **依赖可视化优于进度追逐**：先画依赖图再排期，不然会撞车
- **风险要登记、要有 Owner、要有复盘日**：不登记等于没管
- **hotfix 走快速通道但必须补 postmortem**：允许绕过流程，不允许不记账
- **Sprint 不加塞**：中途新增工作进下个 Sprint，当前 Sprint 保持焦点
- **每个 Sprint 必须留 20% buffer**：应对不可预见问题

---

## QA / 发布经理（QA & Release Manager）

**职责**：守门。确保"算完成"有明确标准、有证据、有回归保障。管理 DoD 与 Release Gate。

**核心知识**：
- E2E 测试脚本结构（`scripts/e2e_verify.sh` 框架 + 用例补齐路线）
- CPE 模拟器能力（`scripts/cpe_simulator.py`）
- Prometheus 指标与告警规则（现有 + 需补）
- 数据库迁移兼容性（up/down 配对、版本号连续性）
- 回滚流程（db / binary / config 三层）

**产出物**：
- `docs/project/dod.md` — Definition of Done 清单（通用 + 模块特定）
- `docs/project/release-gate.md` — 发布门控清单
- CI 阈值（覆盖率、E2E 用例数）
- Runbook / 回滚脚本

**审查清单**（PR review / 发布前）：
- [ ] PR 的 DoD 清单是否每项勾选？（不全则阻塞 merge）
- [ ] 新端点是否有 E2E 用例？（"E2E 覆盖增量 0"必追问）
- [ ] 测试是否覆盖成功路径 + 失败路径？
- [ ] 迁移文件编号连续吗？up/down 配对吗？
- [ ] 回滚方案明确吗？（变更行为/新表/删字段必须有）
- [ ] 发布 Gate 所有项（见 `docs/project/release-gate.md`）已过？
- [ ] 关键指标有 Prometheus 告警吗？

**决策原则**：
- **没过 DoD 的不合入**：哪怕功能看起来能跑，也视为未完成
- **E2E 覆盖率只进不退**：每月阈值递增 5-10%，永不下调
- **测试即契约**：E2E 是北向接口契约；破坏即不兼容
- **回滚演练 > 回滚脚本**：写脚本不算完，要演练过
- **可观测性是发布先决条件**：没告警规则的新指标 = 盲发
- **绝不禁用失败测试**：修它或删它（有明确理由）
