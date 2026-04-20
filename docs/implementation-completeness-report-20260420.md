# OMC 项目前后端实现完成度分析报告

> **报告日期**：2026-04-20  
> **分析范围**：omcgo 后端（494 Go 文件）/ omcmb 前端（393 TS/TSX 文件）  
> **分析维度**：逐模块、逐文件、逐流程，评估完成度并定位关键短板

---

## 0. 执行摘要（TL;DR）

| 层面 | 完成度 | 阶段 | 核心判断 |
|------|--------|------|---------|
| **后端功能域（F01-F10）** | 80% | Beta → RC | 核心协议栈与主流程齐备，缺通知渠道、多级聚合、OSS 标准协议栈 |
| **后端基础设施** | 78% | 接近商用 | Carrier 适配与组件层成熟；Events 缺 NATS、Notification 缺外部渠道 |
| **前端** | 79% | Beta | Device/System/Alarm/Login/Dashboard 可生产；Backup/Software 仅骨架 |
| **启动/部署** | 80% | 可一键起 | docker-compose + 脚本完整；Prometheus/Grafana 编排缺失 |
| **数据库迁移** | 78% | 主体完整 | 版本号跳跃（缺 000010, 000015-000018）需修复 |
| **E2E 测试** | 45% | 框架有用例 0 | 脚本 4459 行，`claim` 实际用例数为 0 |
| **综合** | **74%** | **Beta（封测可用，商用需补齐 5 项短板）** | — |

**五大短板**（按生产影响排序）：
1. **告警通知链路缺失**（F04）——邮件/短信/Webhook/移动端推送全无
2. **E2E 用例实现缺失**——452 个声称用例实际为 0，无法回归
3. **北向 OSS 协议栈缺失**（F08）——仅 REST JSON，无 SNMP/CORBA/MTOSI
4. **事件总线仅 SSE**——缺 NATS JetStream 分布式主题协调
5. **多级聚合与分析**——PM 仅设备级，MR 无地理聚合，无趋势/根因分析

---

## 1. 项目概况

```
omc/                        # 单一 git 仓库
├── omcgo/                  # 后端 Go 1.25 (494 .go 文件，~11 万行代码估算)
│   ├── cmd/                # 6 个入口：app / acs / worker / migrate / seed / omcctl
│   ├── internal/           # 28 个模块（10 功能域 + 基础设施）
│   ├── migrations/         # 19 个迁移文件（编号 000001-000023，有跳跃）
│   └── scripts/            # E2E / 压测 / RPC 工具
└── omcmb/webcode/          # 前端 React 19 + TS (393 .ts/.tsx 文件)
    └── src/
        ├── services/api/   # 28 个 API 服务（246 个 async 方法）
        ├── hooks/api/      # 23 个 React Query Hooks
        ├── pages/          # 18 个页面模块
        └── mock/           # 37 个 mock 文件
```

**部署形态**：模块化单体 + 独立 ACS 引擎 + 后台 Worker（三进程）。  
**规模目标**：10 万基站起步，100 万级预留。

---

## 2. 后端功能域逐模块分析

### 2.1 F01 — 南向接口 ACS（TR069）

**代码规模**：5162 行生产 + 3896 行测试（约 75% 测试占比）

| 文件 | 作用 | 行数 |
|------|------|------|
| `internal/acs/handler.go` | HTTP 请求分发、会话管理、Inform/RPC 路由 | 1656 |
| `internal/acs/session.go` | TR069 会话状态机（IDLE→INFORM_RECEIVED→PROCESSING→RPC_PENDING/RESPONSE→COMPLETE） | 204 |
| `internal/acs/server.go` | HTTP 服务器、TLS 配置 | 186 |
| `internal/acs/auth/` | CWMP 密码校验、摘要认证 | 592 |
| `internal/acs/connreq/` | Connection Request（UDP 唤醒+重试） | 766 |
| `internal/acs/stun/` | STUN 服务器/客户端（NAT 穿透） | 1560 |
| `internal/acs/rpc/dispatcher.go` | 11 个 RPC 注册与 SOAP 请求构建 | 254 |
| `internal/acs/cmdqueue/` | Redis 命令队列 | ~400 |

**RPC 方法覆盖（11/11）**：GetParameterValues、SetParameterValues、GetParameterNames、GetParameterAttributes、SetParameterAttributes、AddObject、DeleteObject、Download、Upload、Reboot、FactoryReset 全部完成。`GetRPCMethods` 非强制未实现。

**测试文件（16 个）**：handler/session/server/admission/ratelimit/authenticator/token/dispatcher/connreq/stun/pathutil 全覆盖。

**TODO/占位符**：零，无 panic/stub。

**完成度：88%** — 协议核心商用级；短板：AutonomousTransferComplete 仅框架实现；SOAP Fault 容错基础。

---

### 2.2 F02 — 数据模型与配置

**代码规模**：5621 行生产 + 3112 行测试（~55% 测试占比）

| 文件 | 作用 | 行数 |
|------|------|------|
| `internal/config/datamodel/handler.go` | 数据模型 REST（CRUD + 导入/导出 + 缓存刷新） | 404 |
| `internal/config/datamodel/importer.go` | XML/JSON 解析 + 三级回退解析 | 410 |
| `internal/config/datamodel/registry.go` | L1 内存 + L2 Redis 缓存 | 437 |
| `internal/config/datamodel/validator.go` | 参数树/OUI/Scope 校验 | 305 |
| `internal/config/datamodel/pg_repository.go` | CRUD + 生命周期 | 700+ |
| `internal/config/template/*.go` | 配置模板 CRUD + 应用 | 488+ |
| `internal/config/baseline/*.go` | 配置基线 CRUD + 批量应用 | 688+ |
| `internal/config/sync_handler.go` | 同步编排（模板 → 基线 → 任务） | 157 |

**REST 端点**：15+（datamodel 含 `resolve` 三级回退、`import-xml`、`activate/deprecate` 生命周期；template/baseline 完整 CRUD + apply）。

**三级回退**：Scope=product(OUI+ProductClass) → oui → carrier_default，L1 sync.Map / L2 Redis TTL 24h / L3 PG，跨实例版本号协调。

**测试文件（9 个）**：handler/service/importer/validator/path_matcher/iterator/cleaner/sync_handler。

**完成度：85%** — 三级缓存/三级回退/生命周期管理完整；短板：模板规则引擎和 Carrier 参数映射待补。

---

### 2.3 F03 — 性能管理（PM/KPI）

**代码规模**：4460 行生产 + 1861 行测试（约 42%）

**关键文件**：`handler.go`(430) / `threshold_handler.go`(110) / `collector/parser.go`(250, 3GPP 32.435 流式解析) / `collector/collector.go`(200) / `kpi/engine.go`(180) + `kpi/formula.go`(280) / `counter/pg_repository.go`(350, TimescaleDB) / `aggregation/aggregator.go`(30) / `metrics.go`(80)。

**REST 端点（12）**：计数器/KPI/任务/文件/阈值查询+CRUD 全实现。

**流程完整性**：
- ✅ 采集：MinIO 事件驱动 → 三协议支持（TR069 Upload/FTP/HTTP）
- ✅ 解析：3GPP 32.435 XML 流式
- ✅ 入库：TimescaleDB hypertable + 批量写入 + 压缩
- ✅ KPI 公式引擎（除零保护、加减乘除）
- ⚠️ **多级聚合仅到设备级**，站点/区域/网络级缺失
- ❌ **阈值告警未联动**：阈值已存但未触发告警事件
- ❌ FTP/HTTP Pull 采集器未实现（仅 MinIO 事件驱动）

**完成度：75%**

---

### 2.4 F04 — 告警管理

**代码规模**：4009 行生产 + 970 行测试

**关键文件**：`handler.go`(280) / `library_handler.go`(150) / `filter_handler.go`(120) / `engine.go`(250) / `receiver.go`(95, EventBus 订阅) / `filter_engine.go`(200, 6 种过滤+6 种动作) / `pg_alarm_store.go`(500) / `redis_store.go`(220) / `library_service.go`(180)。

**REST 端点（16）**：活跃/历史告警查询、统计、单告警确认/清除、批量操作、告警库 CRUD + i18n、过滤规则 CRUD。

**流程完整性**：
- ✅ 接收：EventBus 订阅 `device.alarm`
- ✅ 去重：Redis `alarm:active:{device_sn}:{code}` + DB 二重校验
- ✅ 关联规则：FilterEngine 6 类规则（设备/告警源/告警码/设备组），6 类动作（通知/压制/转发/升级/降级/自定义）
- ✅ 生命周期：激活→确认→清除，支持批量
- ✅ 存储：Redis 活跃 + TimescaleDB 历史
- ❌ **通知系统完全缺失**：规则 action="notify" 找不到下游通知实现（无邮件/短信/钉钉/企微）
- ❌ 与 PM 阈值告警未联动

**完成度：80%**（功能实现完整但通知链路断层，实际不可用于运营）

---

### 2.5 F05 — 测量报告（MR）

**代码规模**：2364 行生产 + 574 行测试

**关键文件**：`handler.go`(448) / `collector/collector.go`(280) / `parser/mro_parser.go`(150) / `mrs_parser.go`(120) / `mre_parser.go`(140) / `collector/detector.go`(100, 文件名识别) / `pg_store.go`(206) / `pg_indicator_repository.go`(353)。

**REST 端点（7）**：文件列表/下载、数据查询、指标列表/统计、导出。

**流程完整性**：
- ✅ MRO（邻小区测量）/ MRS（周期测量）/ MRE（事件触发）三解析器齐全
- ✅ 指标入库 TimescaleDB
- ❌ **无数据分析**：仅存储，缺覆盖率/干扰分析/弱覆盖识别
- ❌ **无地理聚合**：仅设备级，站点/小区级缺失
- ❌ MRO 邻小区关系与拓扑未关联
- ❌ 无异常检测（干扰突增、覆盖黑洞）

**完成度：70%**

---

### 2.6 F06 — OMC-R 核心（12 子模块）

**F06 总计**：106 个 Go 文件，~38,729 行代码，26 个测试文件。平均完成度 **85%**。

#### 2.6.1 device（设备管理）— **88%**

39 文件 / 9124 行。REST 端点 22 个（CRUD+批量+状态+参数树+RF 开关+重启+回收站）。状态机：Discovered→Registered→Provisioning→Active→Maintenance/Offline/Decommissioned。批处理器（516 行）处理 Inform 批刷库。12 测试文件。

**短板**：RF 控制硬编码 LTE（TODO：Carrier 适配）；`status_history` 表字段未关联。

#### 2.6.2 admin（用户/RBAC/认证）— **92%**

45 文件 / 8493 行。REST 端点 25+。RBAC（Casbin）+ JWT + API-Key + 刷新令牌 + CAPTCHA + 限流 + 暴力破解防护 + 审计日志 + 数据权限（用户可见设备组，Redis TTL 5min）+ 菜单鉴权。9 测试文件。

**短板**：CAPTCHA 图形生成待补（仅端点）；API 权限自发现机制部分实现。

#### 2.6.3 topology（拓扑）— **85%**

18 文件 / 4790 行。REST 端点 18。2 级分组（L1 大区 + L2 子区）+ 站点关系 + 设备分配 + 拖拽排序 + 可视化节点/边/地理数据 + 统计聚合。3 测试文件。

**短板**：`rule_service` 中拓扑自动分组规则引擎未激活（TODO 注释）；不支持 >2 级深度分组。

#### 2.6.4 dashboard（仪表盘）— **95%**

4 文件 / 857 行。REST 端点 10（summary/alarm-trend/device-status/kpi-trend/region-stats/widgets/alarm-type-pie/kpi-time-series）。用户 widgets 布局 JSONB 持久化。2 测试文件。**无明显 TODO**（依赖 PM/Alarm 数据质量）。

#### 2.6.5 software（固件管理）— **85%**

10 文件 / 2047 行。MinIO 仓库 + 状态机（7 状态） + 批量升级 + 回滚 + 暂停/恢复/终止。3 测试文件。

**短板**：**灰度策略未实现**（分组/延时/百分比），仅支持全量批量。

#### 2.6.6 backup（配置备份）— **80%**

12 文件 / 2807 行。任务 + 调度（CronExpr） + FTP 配置 + 事件驱动执行 + Upload 命令推送。3 测试文件。

**短板**：`POST /ftp-configs/test` 占位符 "Connection test not implemented"。

#### 2.6.7 ops（运维工具）— **90%**

7 文件 / 2360 行。模板（多步骤工作流） + 任务（并发执行+状态机） + 命令记录（输出/耗时/操作人）。2 测试文件。

**短板**：缺限流、审批工作流等高级特性。

#### 2.6.8 report（报表）— **85%**

8 文件 / 2293 行。定义 + 异步生成器（收集 KPI/告警） + MinIO 存储（PDF/Excel/HTML/JSON）+ 手动触发 + 下载。2 测试文件。

**短板**：定时生成器（CronExpression 已存）未集成到 Worker。

#### 2.6.9 mml（命令行）— **95%**

13 文件 / 5145 行（代码量最大）。命令树 + 脚本 + 任务 + 模板（CRUD+克隆）+ 参数库 + 扇出引擎（MML→device_tasks）+ 结果聚合器 + 危险检查 + 审计。2 测试文件（行数多但比例偏低）。

**短板**：测试覆盖率 15% 偏低。

#### 2.6.10 filemanager（文件管理）— **75%**

6 文件 / 871 行。上传/下载（MinIO）+ 分发（Download RPC 推送）+ 目录浏览 + 元数据。1 测试文件。

**短板**：无配额/速率限制；service 层缺测试。

#### 2.6.11 syslog（系统日志）— **70%**

5 文件 / 582 行。系统日志查询 + NE 消息日志查询（级别/来源/时间/设备/方向过滤）。1 测试文件。

**短板**：**远程 syslog 转发缺失**；无滚动/采样策略。

#### 2.6.12 license（许可证）— **80%**

7 文件 / 1360 行。CRUD + 激活/吊销 + 导入 + 摘要统计。`MaxDevices/UsedDevices/ExpiryDate` 字段齐全。2 测试文件。

**短板**：**容量超限未拦截**、**过期不自动禁用**。

---

### 2.7 F07 — 网元直连（NE Direct）— **75%**

8 文件 / 2540 行。REST 端点 8（register/connect/disconnect/command/sessions/config/status/fault）。会话状态机（Active↔Disconnected）+ 超时关闭 + 故障转告警 + device/alarm 集成 + 命令队列对接。2 测试文件。

**短板**：无负载测试；无细粒度限流；无会话持久化 HA。

---

### 2.8 F08 — 北向/OSS 接口 — **70%**

16 文件 / 3054 行。REST 端点：export/{pm,alarms,config} + push/targets + sync/{full,incremental} + push/deadletter。**Push Engine**（HTTP 推送 + Outbox 模式 + Circuit Breaker + 签名认证 + 批量）。5 测试文件。

**短板（严重）**：**SNMP/CORBA/MTOSI/TMF 协议栈完全缺失**，仅 REST JSON 导出。不符合运营商 OSS 接入规范。

---

### 2.9 F09 — 自动开站（Zero-Touch Provisioning）— **82%**

15 文件 / 3886 行。REST 端点：templates/tasks/run/validate。完整闭环：设备发现 → Matcher（carrier+tech+productClass 三级匹配）→ GetParameterValues → SetParameterValues → Reboot。状态机 Pending→Processing→Completed→Failed。4 测试文件。

**短板**：无模板审计、回滚机制、第三方库存联动。

---

### 2.10 F10 — 互操作测试 — **68%**

10 文件 / 1874 行。REST 端点：test-cases/run/run/{category}/validate/{deviceId}。ConformanceTestRunner + Validator（参数树/数据模型/协议合规）。2 测试文件。

**短板**：**用例库稀疏（仅 RPC/Protocol/DataModel 三类）**，无运营商标准测试套件、性能/安全/多设备并发测试。

---

### 2.11 基础设施（Infrastructure）

| 模块 | 文件/行 | 完成度 | 说明 |
|------|--------|--------|------|
| `internal/core/` | 83 文件 / 9078 行 | **95%** | Middleware（CORS/Auth/Log/Metrics/Tracing）+ Carrier 三运营商适配器完整 + Components（PG/Redis/NATS/MinIO 健康检查+优雅关闭） |
| `internal/events/` | 3 文件 / 476 行 | **60%** | **仅 SSE Hub**，无 NATS JetStream 主题订阅，单进程事件总线 |
| `internal/notification/` | 5 文件 / 647 行 | **55%** | 仅内部通知 + SSE 推送；**邮件/短信/Webhook 全缺失** |
| `internal/task/` | 14 文件 / 4138 行 | **85%** | Redis Sorted Set 任务队列 + 优先级调度 + Prometheus 指标 + Connection Request 唤醒 |
| `internal/transfer/` | 2 文件 / 717 行 | **88%** | 订阅 AutonomousTransferComplete → 下载 → 事件分发（pm/mr/datamodel.file.received） |
| `global/` + `pkg/` | 常量/错误码/TR069 类型 | **90%** | — |

---

## 3. 前端逐模块分析（omcmb/webcode）

### 3.1 API 服务层（services/api/* 共 28 文件，246 async 方法）

**全部直连真实 HTTP 后端**，通过 `http.ts` 拦截器（Bearer Token、camelCase↔snake_case、401 刷新队列）。

| 梯度 | API 数 | 代表模块 |
|------|--------|---------|
| **高频（15+ 方法）** | 5 | adminApi(57) / mmlApi(24) / alarmApi(22) / deviceApi(21) / topologyApi(19) |
| **中频（8-14 方法）** | 6 | pmApi / backupApi / opsToolsApi / datamodelApi / deviceParameterApi / dashboardApi |
| **轻量（≤7 方法）** | 17 | authApi(3) / systemApi(1) / configSyncApi(3) / licenseApi(6) / templateApi(5) / provisionApi(4) / interopApi(4) 等 |

### 3.2 页面模块（src/pages/* 共 18 个）

#### 成熟页面（90%+，可交付）

| 页面 | 文件数 | 完成度 | 说明 |
|------|--------|--------|------|
| **Login** | 1 | 98% | 登录/记住凭据/Token 刷新/重定向完整 |
| **Error（404/403）** | 3 | 95% | 错误边界完整 |
| **System** | 24 | 94% | 用户/角色/菜单/字典/LDAP/安全策略/通知 11 子模块 |
| **Device** | 36 | 92% | 最成熟：列表/分组/NE/交付/资源/规则/回收站/导入导出 |
| **Alarm** | 11 | 91% | 活跃/历史/统计/规则/支持库/同步 7 页 |
| **Dashboard** | 1 | 88% | KPI 卡片+设备统计+告警分布+GIS+导航 |

#### 完善页面（75-90%）

| 页面 | 文件数 | 完成度 | 短板 |
|------|--------|--------|------|
| **Config** | 15 | 85% | 部分列表加载 TODO |
| **MML** | 19 | 78% | 命令树/脚本编辑完整 |
| **File** | 7 | 76% | 7 个检索模块 |
| **Topology** | 6 | 75% | GIS（Leaflet）+ 画布（D3），交互优化空间 |
| **Performance** | 11 | 72% | 图表数据部分硬编码，查询模板等待后端 |

#### 中等页面（55-75%）

| 页面 | 文件数 | 完成度 | 短板 |
|------|--------|--------|------|
| **OpsTools** | 5 | 62% | 高级诊断缺失 |
| **Report** | 4 | 58% | 模板构建交互不全 |
| **License** | 3 | 55% | 激活/二维码交互待完善 |
| **Software** | 5 | 52% | **缺升级进度/回滚/固件验证交互** |
| **Backup** | 5 | 48% | **仅列表，新增/编辑/执行业务逻辑空白** |

### 3.3 基础设施

| 模块 | 文件/说明 | 完成度 |
|------|----------|--------|
| `services/http.ts` | Token 注入 + camel↔snake + 401 刷新队列 + 参数过滤 | **88%**（缺超时提示/重试） |
| `services/apiSwitch.ts` | `VITE_USE_MOCK` 全量切换 | **72%**（无热切换/灰度） |
| `store/*.ts` | Zustand：user/app/alarm/task/tab | **80%** |
| `mock/*` | 16 数据集 + 17 service + 2 WebSocket mock | **84%** |
| `types/*.ts` | 13 文件，100+ interface，backend↔frontend 映射 | **86%** |
| `router/routes.tsx` | 35+ 路由懒加载 + PrivateRoute | **82%**（无动态权限过滤） |
| `hooks/api/*` | 23 React Query Hooks | **79%**（无乐观更新） |

**前端总完成度：79%**

---

## 4. 启动 / 迁移 / 测试 / 部署

### 4.1 二进制入口 — **85%**

- `cmd/app/main.go` + `bootstrap.go` + `provider/*`：16 模块 DI + 拓扑排序 + 双健康检查（`/healthz` 2s / `/readyz` 5s 组件级）+ 优雅关闭（30s 超时，HTTP→NATS→Redis→DB）。路由 490 行接近维护上限。
- `cmd/acs`：独立进程，协议日志 JSON 轮转，Upload/Download + STUN + Connection Request + Post-Session Wake 完整（**95%**）。
- `cmd/worker`：事件驱动，6 订阅者（PM 收集+KPI+告警+MR+Transfer+Backup+Report）。**无重试/死信队列**（**90%**）。
- `cmd/migrate`（Goose）/ `cmd/seed` / `cmd/omcctl` / `scripts/rpctool`：工具链齐全。

### 4.2 数据库迁移 — **78%**

**19 个迁移文件**，版本号 **000001-000023** 但有跳跃：

| 文件 | 内容 | 状态 |
|------|------|------|
| 000001 | TimescaleDB/uuid/update 函数 | ✅ |
| 000002 | 用户/角色/RBAC | ✅ |
| 000003 | 设备（分区：carrier+tech） | ✅ |
| 000004 | 数据模型 | ✅ |
| 000005 | 配置+PM | ✅ |
| 000006 | 告警+MR+固件（hypertable） | ✅ |
| 000007 | 字典/审计 | ✅ |
| 000008 | 网元直连+北向 | ✅ |
| 000009 | API 端点+权限 | ✅ |
| **000010** | **缺** | ❌ |
| 000011 | 索引优化 | ✅ |
| 000012 | API 端点+数据权限 | ✅ |
| 000013 | 告警增强 | ✅ |
| 000014 | MML 模板+审计 | ✅ |
| **000015-000018** | **缺** | ❌ |
| 000019 | 告警清除字段 | ✅ |
| 000020 | MML 分类/分组 | ✅ |
| 000021 | 通知模块 | ✅ |
| 000022 | MML 参数库 | ✅ |
| 000023 | 设备任务 + MML 桥接 | ✅ |

**风险**：版本号跳跃会让 goose 在严格模式下失败或产生歧义。部分文件 Down 段过简单。

### 4.3 E2E 测试 — **45%**

- `scripts/e2e_verify.sh`：4459 行，**框架完整但用例实现为 0**（grep `claim` = 0）。CLAUDE.md 宣称的 "~452 用例" 未实现。
- `scripts/cpe_simulator.py`：2200+ 行，完整 CPE 模拟（12 RPC + 中移 EventCode 101-107 + 告警/PM/MR 模拟 + 批量并发）。**压测工具已可用**。
- `scripts/loadtest-benchmark.sh`：4 轮阶梯（100→3000 并发，2000→30000 设备）。
- `cmd/worker`/`scripts/seed_e2e_testdata.sql`：种子数据覆盖有限。

**严重问题**：**无法做回归**。

### 4.4 部署 & CI — **80%**

- `docker-compose.yml`：6 服务 + migrations，依赖链（postgres → migrate-schema → migrate-seed → acs/app/worker）+ healthcheck（5 服务）+ 多阶段 Dockerfile。
- `run/scripts/{start-all,restart-all,start-deps,start-backend}.sh`：本地一键启动齐全（含 design-baseline worktree :3001 对比）。
- Prometheus Metrics + OpenTelemetry 链路追踪已集成，**但缺 Prometheus/Grafana/AlertManager 容器编排**。
- `Makefile`：build/test/migrate/seed/docker/race 目标齐全（**95%**）。

---

## 5. 综合评估矩阵

### 5.1 后端功能域完成度矩阵

| 功能域 | 采集 | 解析 | 入库 | 聚合 | 查询 | 规则 | 通知 | 分析 | 总体 |
|--------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| F01 ACS | ✅ | ✅ | ✅ | — | ✅ | ✅ | — | — | **88%** |
| F02 配置 | — | ✅ | ✅ | — | ✅ | ◐ | — | — | **85%** |
| F03 PM | ✅ | ✅ | ✅ | ◐ | ✅ | — | — | ❌ | **75%** |
| F04 告警 | ✅ | — | ✅ | — | ✅ | ✅ | ❌ | ❌ | **80%** |
| F05 MR | ✅ | ✅ | ✅ | ❌ | ✅ | — | — | ❌ | **70%** |
| F06 核心 | — | — | ✅ | ✅ | ✅ | ◐ | ❌ | ✅ | **85%** |
| F07 NE 直连 | ✅ | — | ✅ | — | ✅ | — | — | — | **75%** |
| F08 北向 | — | — | ✅ | — | ✅ | ◐ | ◐ | — | **70%** |
| F09 开站 | ✅ | ✅ | ✅ | — | ✅ | ✅ | — | — | **82%** |
| F10 互操作 | — | — | ✅ | — | ✅ | — | — | ◐ | **68%** |

### 5.2 五大关键短板（按生产阻塞度）

| # | 短板 | 影响 | 建议 |
|---|------|------|------|
| 1 | **告警通知链路**（F04/Notification） | ⚠️ 运营致命：告警无法送达 | 补邮件/短信/Webhook/移动端推送 |
| 2 | **E2E 用例实现**（scripts/e2e_verify.sh） | ⚠️ 无法回归、无法守门 | 补齐 452 用例（估 ~200 人天） |
| 3 | **F08 OSS 协议栈** | 不满足运营商接入 | SNMP/CORBA/MTOSI 适配 |
| 4 | **NATS JetStream 事件总线** | 单进程限制、无分布式协调 | Events 改 JetStream，弃 SSE-only |
| 5 | **多级聚合 + 数据分析** | PM/MR 价值打折 | 站点/区域/网络级 KPI、趋势/根因 |

### 5.3 次级短板

- **数据库迁移版本号跳跃**（缺 000010、000015-000018）
- **F06 灰度升级策略缺失**（software 模块仅全量）
- **F10 用例库稀疏**（仅 3 类）
- **拓扑自动分组规则引擎未激活**（rule_service TODO）
- **License 容量/过期未拦截**
- **syslog 远程转发缺失**
- **worker 无重试/死信**
- **前端 Backup/Software 仅骨架**
- **Prometheus/Grafana 编排缺失**
- **CAPTCHA 图形生成待补**
- **RF 控制硬编码 LTE（Carrier 适配未贯穿）**

---

## 6. 阶段定位与路线建议

### 6.1 当前阶段：**Beta（封测可用）**

- ✅ 可一键部署、核心业务链可演示、主流程有测试
- ⚠️ 不能回归（E2E 用例 0）、告警不可达用户、OSS 标准协议缺
- ❌ 商用级门槛：可观测性链不全、灰度/容量/限流策略稀疏

### 6.2 到 RC（预生产）的最小收敛项

1. **[P0]** 补告警通知渠道（邮件 + 短信 + Webhook 三件套） — 2 周
2. **[P0]** 补 E2E 用例核心 200 项 — 6 周
3. **[P0]** 修复数据库迁移版本号跳跃 — 1 天
4. **[P1]** NATS JetStream 事件总线改造 — 3 周
5. **[P1]** Prometheus/Grafana/AlertManager 编排 + SLA 大盘 — 1 周
6. **[P1]** Software 灰度升级策略（分组/延时/百分比） — 2 周
7. **[P2]** 前端 Backup/Software 业务逻辑补齐 — 2 周
8. **[P2]** License 容量超限拦截 + 过期禁用 — 3 天

**估算**：**~14 周**可从 Beta 收敛至 RC。

### 6.3 到 GA（商用）的持续项

- F08 OSS 协议栈（SNMP/CORBA/MTOSI）：视客户要求分阶段补齐
- F10 运营商标准测试套件对齐
- 多级聚合与分析面板（覆盖率/干扰/趋势）
- 分布式事务/跨机房容错/自动 failover
- 审计日志完整化、密钥轮换、DLP 策略

---

## 7. 代码质量参考指标

| 指标 | 数值 |
|------|------|
| 后端 Go 文件 | 494 |
| 后端估算代码行 | ~110,000 行 |
| 后端测试文件 | ~128 个，核心模块覆盖 60-75%，F07-F10 覆盖 50-70% |
| 前端 TS/TSX 文件 | 393 |
| 前端 API 方法 | 246 个 async 方法 |
| 前端 Mock 文件 | 37 个 |
| 数据库迁移 | 19 个（编号 000001-000023） |
| 运营商适配 | CMCC / CTCC / CUCC 全适配 |
| TR069 RPC 覆盖 | 11/11 强制方法 |

---

## 8. 结论

**一句话结论**：OMC 是一个**代码体量充足（11 万行后端 + 393 前端文件）、架构清晰（模块化单体 + 独立 ACS + Worker）、核心协议与业务主流程工程完整度高（F01/F06 商用水准）的 Beta 阶段系统**，但**告警通知链路、E2E 回归用例、OSS 标准协议栈、事件总线分布式化、数据分析聚合**五个短板决定了它距离 **RC 还需 ~14 周** 的补齐工作。

**生产风险判断**：**中等**。可在单运营商小规模（≤1 万基站）试点，但大规模商用前必须补齐前三项 P0 短板。

---

*报告生成：2026-04-20 by Claude Opus 4.7 (1M context)。基于 7 个并行子 Agent 的逐模块调研汇总。*
