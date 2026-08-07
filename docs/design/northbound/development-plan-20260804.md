# 北向功能页面化配置开发计划

> 创建日期：2026-08-04
> 最近更新：2026-08-04 19:36 CST
> 当前实现工作目录：`/Users/renpengfei/code/goomc/goomcnew/xomc-northbound-page-config`
> 目标合入目录：`/Users/renpengfei/code/goomc/goomcnew/xomc`
> 开发分支：`feature/northbound-page-config`
> 最新设计基线：`docs/design/northbound/page-config-redesign-20260731.md`

## 1. 执行边界

本文件是后续开发执行计划和恢复记录，不替代最新设计文档。后续北向页面化配置均以 `page-config-redesign-20260731.md` 为设计基线，旧 XML、旧 Java 文档和旧场景文件只作为字段、文件名、路径、协议和兼容行为参考。

硬约束：

- 不再按 XML 配置文件管理北向功能，统一由页面配置和后端持久化模型管理。
- 当前系统没有 EGW，模型、页面、文件模板、Inventory、Socket、SNMP 和 API 均不得包含 EGW/PEGW。
- SNMP 告警字段、OID、Trap/Inform 字段顺序以 `/Users/renpengfei/Desktop/doc/othernorth/omcAlarmMIB.mib` 为准，企业号为 `1.3.6.1.4.1.53058`。
- Socket 告警按服务端模式设计，覆盖用户名密码、多账号、实时推送和客户端同步告警。
- 北向文件和 Inventory 周期对用户展示为语义选项，不直接暴露 cron。
- 北向 API 只保留老系统支持且当前 xomc 已能满足的数据接口，不能编造新接口。
- CM、PM、MR、LOG、Inventory 字段必须来自当前 xomc 实际数据源或明确标为需确认。

## 2. 已核对资料

| 类型 | 路径/模块 | 结论 |
|---|---|---|
| 最新设计 | `docs/design/northbound/page-config-redesign-20260731.md` | 五个固定页签：北向文件配置、Inventory 文件、Socket 告警、SNMP 告警、北向 API；FTP/SFTP 为抽屉内多目标配置；状态列只显示“正常/终止”。 |
| 数据支持矩阵 | `docs/design/northbound/data-support-matrix-20260731.md` | PM、CM、Inventory 均有支持状态口径；PM 保存当前 `metric_path`，旧名只作为导出列名。 |
| 当前原型 | `omcmb/webcode/src/pages/config/NorthboundPageConfig/index.tsx`、`index.module.css` | 页面结构和抽屉交互已保留；本轮已将列表、保存、运行、下载、事件查看、测试连接等主要入口接到后端。 |
| PM 资料 | `pmMetricCatalog.ts`、`pmMetricCatalog.json`、PM 技能知识库 | 前端静态 PM JSON 只作为兜底展示；生产字段目录和保存校验已接入后端 PM 指标表。 |
| 老文件配置 | `/Users/renpengfei/Desktop/doc/config/S0001-S0017`、`NorthboundFileModule-功能梳理文档.md` | 旧场景包含 CM/PM/MR/LOG 文件、路径和命名规则；CM XML、EGW/PEGW 不进入新版默认能力。 |
| 实际生成文件 | `/Users/renpengfei/Desktop/doc/20260730/cm/`、`/Users/renpengfei/Desktop/doc/20260730/pm/pc/` | 用于参考 CM/PM CSV、zip 文件名和目录样例。 |
| Inventory 资料 | `/Users/renpengfei/Desktop/doc/othernorth/enbMonitorExport.md` | 多数设备基础字段可支持，资产、规划、联系人、线路、维护团队等字段当前无模型。 |
| Socket 资料 | `/Users/renpengfei/Desktop/doc/othernorth/NorthboundSocketModule-功能梳理文档.md` | CTCC/CUCC 均为 OMC 服务端监听；包含登录、心跳、实时告警、历史同步和 CUCC 文件同步。 |
| SNMP 资料 | `/Users/renpengfei/Desktop/doc/othernorth/NorthboundSNMPModule-功能梳理文档.md`、`omcAlarmMIB.mib` | MIB 企业号为 `1.3.6.1.4.1.53058`，通知对象 18 个字段顺序固定。 |
| API 资料 | `/Users/renpengfei/Desktop/doc/othernorth/northboundApi-功能与接口清单.md` | 当前实现只展示已确认“老系统支持 + 新系统可满足”的 3 个接口；其余旧接口等待路由和 DTO 复核后再加入。 |
| 当前后端 | `omcgo/internal/northbound` | 已新增 `pageconfig` 模块并接入 router/provider；SNMP 包已按 MIB 修正企业号、OID 和字段顺序。 |
| 当前 schema | `omcgo/migrations/000001_init_schema.sql`、`seed/000001_init_seed.sql` | 页面化配置表、运行表、事件表、目标配置表和 API 权限 seed 已折回 baseline。 |

## 3. 进度记录区

后续每次恢复开发时，先更新本区，再进入对应 P 阶段 checklist。

### 3.1 总体状态

| 字段 | 当前值 |
|---|---|
| 总体状态 | 已完成 |
| 最新更新时间 | 2026-08-04 19:36 CST |
| 当前完成内容 | 独立 worktree 已快进到最新 `origin/main`；P0-P7 的页面化配置控制面、文件/Inventory 生成、周期调度、FTP/SFTP 投递、SNMP Trap/Inform 发送、Socket 服务端、北向 API client/白名单、结果分页/下载/审计/清理均已落地；P8 已完成权限 seed、旧入口跳转、审计脱敏、部署健康检查、浏览器冒烟和上线/回滚记录。 |
| 下次入口 | 现场验收入口：P3 外部 SFTP 目标；P4 外部 SNMP Trap/Inform receiver；P5 CTCC/CUCC 真实报文和历史告警口径；P8 全量 i18n 抽键翻译。代码继续开发时从本表新增一行记录即可恢复。 |
| 当前阻塞 | 无代码阻塞。仍需人工确认：真实外部协议对端、Socket 历史告警字段/窗口/游标、Inventory 资产类与 OMC HA/IP/硬件来源、`auth-login` 是否纳入页面开关、全量英文译文。 |

### 3.2 阶段状态看板

| 阶段 | 名称 | 状态 | 当前说明 |
|---|---|---|---|
| P0 | 基础模型与配置框架 | 已完成 | 配置表、repository/service/handler、router/provider、前端 API、基础权限 seed 和单测已完成。 |
| P1 | 北向文件配置 | 已完成 | 页面配置、创建/编辑、预览、手动生成、CM/PM/MR/LOG 数据加载、run 记录、周期调度、窗口恢复、失败重试、自动投递和 golden 内容测试已完成。 |
| P2 | Inventory 文件配置 | 已完成 | eNB/gNB/GSM/OMC profile、保存、手动生成、run 记录和 eNB golden 内容测试已完成；资产类/OMC 扩展字段列入人工确认。 |
| P3 | FTP/SFTP 多目标配置 | 已完成 | 多目标配置、密文保护、连接测试、真实 FTP/SFTP 上传、SFTP SHA256 host key 校验、重试、delivery event 和 CUCC 文件同步投递入口已完成。 |
| P4 | SNMP 告警 | 已完成 | MIB OID、18 字段顺序、v2/v3 Trap/Inform 配置、测试发送、告警事件触发和本地 receiver Trap 解码测试已完成。 |
| P5 | Socket 告警 | 已完成 | CTCC/CUCC 服务端、动态 reload、登录认证、心跳、实时告警推送、同步 ACK、基于 page-config event 的历史 replay 和 CUCC 文件同步投递已完成。 |
| P6 | 北向 API | 已完成 | 只保留 3 个确认交集接口；启停、契约检查、设备同步/配置导出 gate、API client、token/scope/IP 白名单和单测已完成。 |
| P7 | 状态、上报结果、文件下载、报文查看 | 已完成 | runs/events 分页、最新文件下载、下载审计、报文查看、target 级结果、默认 90 天保留清理已完成。 |
| P8 | 权限、审计、国际化、测试与上线 | 已完成 | endpoint 权限 seed、审计脱敏、旧入口跳转、Go/typecheck/schema/seed/浏览器冒烟/本地部署验证完成；全量 i18n 抽键需独立翻译表继续推进。 |

### 3.3 开发日志

| 日期 | 阶段 | 状态 | 已完成 | 下次入口 | 阻塞/风险 | 验证命令 |
|---|---|---|---|---|---|---|
| 2026-08-04 | 计划 | 已完成 | 创建本开发计划初版 | P0-01 | 字段口径待确认 | 未运行业务测试 |
| 2026-08-04 | P0 | 已完成 | 新建 `feature/northbound-page-config` worktree；新增 pageconfig catalog、validator、service、repository、handler、router/provider；折回主库 baseline DDL 和权限 seed；前端新增 API service 并接入页面列表/保存。 | P1-02 | baseline 文件较大，后续继续用临时库验证 | `/usr/local/go/bin/go test ./internal/northbound/... ./cmd/app/provider`；`npm run typecheck` |
| 2026-08-04 | P1/P2/P7 | 进行中 | 实现 file/inventory preview、手动生成、`northbound_file_runs`、下载、CSV/XML/TXT 内容预览；CM/PM/MR/LOG/Inventory 数据 loader 已从当前表或 TSDB 读取。 | P3-03 | PM 指标覆盖率取决于当前指标表；Inventory 资产类字段需确认 | `/usr/local/go/bin/go test ./internal/northbound/... ./cmd/app/provider` |
| 2026-08-04 | P3-P6/P7 | 进行中 | 增加 delivery/SNMP/socket/API 配置表和默认配置；增加统一 `northbound_page_config_events`；前端“测试连接、上报结果、报文查看”切到 events；SNMP 企业号修正为 `1.3.6.1.4.1.53058`。 | P3-03/P4-05/P5-03 | 真实数据面和外部对端协议需要继续验证 | `/usr/local/go/bin/go test ./internal/northbound/... ./cmd/app/provider`；`npm run typecheck` |
| 2026-08-04 | P0/P1/P8 | 进行中 | 拉取最新 `origin/main` 并快进 worktree；补 PM 指标表字段目录和 `metric_path` 校验；补 `POST /file/profiles` 创建接口、前端新增保存流程、权限 seed 和测试；schema/seed Up 在临时库通过。 | P3-03 | 真实周期调度、FTP/SFTP 上传 worker、SNMP/Socket 事件流仍未完成 | `/usr/local/go/bin/go test ./internal/northbound/... ./cmd/app/provider`；`npm run typecheck`；schema/seed 临时库验证 |
| 2026-08-04 | 部署 | 已完成 | 使用当前 worktree 构建并重启本地核心服务；因本地老库已跑过 baseline，手工执行非破坏性 schema reconcile 创建缺失 page-config 表，并补 page-config endpoint 权限；MinIO 用 `IMAGE_MINIO=minio/minio:latest` 避免旧镜像读取新数据卷失败；worker 用 `PM_AGGREGATION_FINALIZE_CONCURRENCY=4` 适配 dev TSDB 连接池。 | P3-04 | 以上 MinIO/worker 为本地运行环境覆盖，不写入仓库文件 | app/worker/ACS healthz；前端 8081；page-config endpoint/table 查询 |
| 2026-08-04 | P3/P7 | 进行中 | 新增 FTP/SFTP 上传客户端；文件/Inventory 手动 run 成功后 fanout 到启用目标，支持重试、远端目录创建、压缩产物投递、远端路径记录；target 级结果写入统一 delivery event；前端最新上报结果优先展示投递事件；下载接口按 zip/gz 返回真实压缩内容；补本地 mock FTP 上传集成单测。 | P1-08/P4-05 | SFTP host key 策略、私钥格式、真实 SFTP 对端测试和周期调度自动投递仍需补强 | `/usr/local/go/bin/go test ./internal/northbound/... ./cmd/app/provider`；`npm run typecheck`；`git diff --check` |
| 2026-08-04 | P1/P4/P5/P6/P8 | 进行中 | 已跟随最新 `origin/main`；新增 page-config scheduler 和 PG advisory lock，支持文件/Inventory 周期窗口、失败重试和自动投递；SNMP sender 接入测试发送与告警 raised/cleared forwarder；新增 CTCC/CUCC Socket server manager，支持动态 reload、登录、心跳、实时推送、同步 ACK、CUCC 文件同步投递；API 开关 gate 接入设备全量同步和配置快照导出；配置保存/测试/手动运行写入审计且不记录密钥。 | P3-07/P4-06/P5-07/P8-04 | SFTP host key、SNMP receiver、Socket 历史补数和 `auth-login` gate 需要人工确认 | `/usr/local/go/bin/go test ./internal/northbound/... ./cmd/app/provider`；`npm run typecheck` |
| 2026-08-04 | P1-P8 收口 | 已完成 | 快进到 `origin/main@b7d53e113`；补 SFTP host key policy、API client/scope/IP 白名单、SNMP receiver Trap 单测、Socket sync replay、runs/events 分页、下载审计、90 天清理、统一 alarm event consumer、P1/P2 golden 测试、旧北向入口跳转；本地部署后 page-config 告警 consumer 正常启动，浏览器冒烟 5 个页签和初始化接口均通过。 | 现场验收或 MR 前代码审查 | 外部 SFTP/SNMP/Socket 对端和全量英文译文需人工提供 | `/usr/local/go/bin/go test ./internal/northbound/... ./cmd/app/provider`；`npm run typecheck`；schema/seed 临时库；Playwright 页面冒烟；Compose healthz |

## 4. P0 基础模型与配置框架

当前状态：已完成

### 目标

建立页面化北向配置的统一后端模型、字段字典、配置 API、权限 seed 和前端 API 适配层，保证页面从后端加载真实配置并可保存。

### 涉及模块

- 后端：`omcgo/internal/northbound/pageconfig`、`omcgo/internal/northbound/router.go`、`omcgo/cmd/app/provider/modules.go`。
- 数据库：`omcgo/migrations/000001_init_schema.sql`、`omcgo/migrations/seed/000001_init_seed.sql`。
- 前端：`omcmb/frontend-core/src/services/api/northboundPageConfigApi.ts`、`omcmb/webcode/src/pages/config/NorthboundPageConfig`。

### 需要修改/新增的文件

- 已新增：`omcgo/internal/northbound/pageconfig/*.go`。
- 已修改：`router.go`、`modules.go`、baseline schema、seed、前端 API service、页面路由和组件注册。

### 后端任务

- 已完成统一枚举、DTO、validator、service、repository、handler。
- 已完成默认配置初始化和 fail-closed 校验。
- 已完成 PM 字段目录从 `perf_indicators_enb/gnb/gsm` 读取，并校验 `metric_path`。

### 前端任务

- 已新增 `northboundPageConfigApi` 类型和方法。
- 已接入五个页签主要列表、保存、运行、测试和事件接口。
- PM 前端静态 catalog 仅作为空数据兜底。

### 数据库/配置模型任务

- 已折回 `northbound_file_profiles`、`northbound_inventory_profiles`、`northbound_field_mappings`、`northbound_endpoints`、`northbound_file_runs`。
- 已折回 delivery、SNMP、Socket、API 配置表和统一事件表。

### 接口任务

- 已完成 overview、fields、validate、file/inventory profile list/save/create、delivery/SNMP/socket/API 配置和 events 基础接口。

### 测试验证方式

- Go 单测覆盖 profile 保存、创建、默认配置、字段校验、PM `metric_path` 校验和 handler 行为。
- 前端 `npm run typecheck` 通过。
- schema/seed Up 在临时库验证通过。

### 完成标准

- 页面刷新后配置可持久化。
- 默认内置配置关闭。
- Inventory 不可混入普通北向文件域。
- PM 保存前校验当前指标存在。

### 风险点

- baseline 文件大，新增 DDL/DML 必须持续用临时库验证。
- 旧 northbound push/server 模型和页面化配置语义相近但不等价，不应直接混用。

### Checklist

- [x] P0-01 表关系和字段草案确认，不含 EGW/PEGW。
- [x] P0-02 baseline DDL、约束、索引折回。
- [x] P0-03 seed 权限折回。
- [x] P0-04 repository/service/validator/handler 完成。
- [x] P0-05 router/provider 注册完成。
- [x] P0-06 前端 API service 和类型完成。
- [x] P0-07 页面列表加载/保存切后端。
- [x] P0-08 单测、typecheck、迁移验证记录完成。

## 5. P1 北向文件配置

当前状态：已完成

### 目标

实现 CM、PM、MR、LOG 北向文件的页面化配置、字段选择、语义周期、模板预览、手动生成、压缩、本地运行记录、周期调度、窗口恢复和自动投递。

### 涉及模块

- 后端：`pageconfig/catalog.go`、`validator.go`、`preview.go`、`generator.go`、`repository.go`。
- 前端：北向文件配置页签、字段抽屉、运行/结果抽屉。
- 数据源：`devices`、`device_info`、`sys_login_logs`、`sys_oper_logs`、TSDB `pm_metrics`、TSDB `mr_files`。

### 需要修改/新增的文件

- 已新增：`preview.go`、`generator.go`、`scheduler.go`。
- 已修改：前端 file profile 新增/编辑/运行/下载逻辑，API service。
- 已修改：`omcgo/cmd/app/provider/modules.go` 装配 page-config scheduler。

### 后端任务

- 已完成 token 渲染和路径安全校验。
- 已完成 `POST /file/profiles` 新增、`PUT /file/profiles/:id` 保存、preview、manual run。
- 已完成 CM/PM/MR/LOG 最小数据加载和 CSV/XML/TXT 生成。
- 已完成周期调度、语义窗口计算、PG advisory lock、失败窗口重试、错过窗口补跑和自动投递。

### 前端任务

- 已完成语义周期配置，不直接展示 cron。
- 已完成文件名/路径预览、保存、启停、手动执行、下载和结果查看。
- 已完成新增配置保存，编号使用数据库认可的 `S0000` 格式。

### 数据库/配置模型任务

- 已使用 `northbound_file_profiles.groups` 存储对象、域、格式、周期、路径、文件名、压缩配置。
- 已使用 `northbound_file_runs` 记录手动生成产物和摘要。
- 已复用 `northbound_file_runs` 识别调度窗口和失败重试，新增调度查询索引；未新增 cron 字段。

### 接口任务

- 已完成 `POST /api/v1/northbound/page-config/file/profiles`。
- 已完成 `GET/PUT /api/v1/northbound/page-config/file/profiles`。
- 已完成 `GET /api/v1/northbound/page-config/file/profiles/:id/preview`。
- 已完成 `POST /api/v1/northbound/page-config/file/profiles/:id/run`。
- 已完成 `GET /api/v1/northbound/page-config/runs`、`GET /runs/:id`、`GET /runs/:id/download`。

### 测试验证方式

- Go 单测覆盖 create/update、非法路径、manual run、run events、PM 指标校验和 CM golden 内容。
- 前端 typecheck 覆盖新增 API 类型和页面调用。
- 已补固定窗口 golden 内容测试；真实运营商样例仍建议在现场验收时追加。

### 完成标准

- 已达到手动/周期生成、窗口恢复、失败重试、与 P3 投递衔接和默认模板 golden 测试闭环。

### 风险点

- PM 指标覆盖率取决于当前指标表，旧场景缺失指标不能临时编造。
- LOG 字段范围需和审计/安全日志表继续确认。

### Checklist

- [x] P1-01 CM/PM/MR/LOG 字段目录初版。
- [x] P1-02 token 渲染和路径安全校验。
- [x] P1-03 CM CSV 最小生成器。
- [x] P1-04 PM CSV 生成器接入 `pm_metrics` 和指标表校验。
- [x] P1-05 MR 最小导出接入 `mr_files` 元数据。
- [x] P1-06 LOG TXT/CSV 最小导出。
- [x] P1-07 手动运行和 run 记录。
- [x] P1-08 周期任务、窗口恢复、失败重试。
- [x] P1-09 真实样例 golden 测试。当前覆盖默认 CM/CP 固定窗口；运营商样例文件可在现场验收追加。

## 6. P2 Inventory 文件配置

当前状态：已完成

### 目标

实现 eNB、gNB、GSM、OMC Inventory 文件的页面化配置、字段目录、CSV 生成和运行记录。Inventory 独立于普通北向文件域。

### 涉及模块

- 后端：`pageconfig/catalog.go`、`generator.go`、`repository.go`。
- 前端：Inventory 页签、字段抽屉、路径/命名预览。
- 数据源：`devices`、`device_info`、系统配置。

### 需要修改/新增的文件

- 已复用 `pageconfig` 生成器和 run 模型。
- OMC 显式配置来源、资产类字段模型已列入人工确认；当前实现不编造缺失字段。

### 后端任务

- 已完成 Inventory profile 保存和手动生成。
- 已完成 eNB/gNB/GSM/OMC 最小字段输出。
- 已完成当前可支持字段输出；资产类、规划类、OMC HA/IP/硬件字段来源列入人工确认。

### 前端任务

- 已完成四类 Inventory profile 独立展示、保存、启停、手动执行、结果查看。
- 已展示当前后端支持字段；不支持字段由人工确认后再扩展模型。

### 数据库/配置模型任务

- 已使用 `northbound_inventory_profiles` 保存对象、技术制式、周期、路径、文件名、压缩和状态。
- 已复用 `northbound_file_runs` 记录 Inventory 导出结果。

### 接口任务

- 已完成 `GET/PUT /inventory/profiles`。
- 已完成 `POST /inventory/profiles/:id/run`。
- 已通过 `GET /fields?domain=INVENTORY` 暴露字段目录。

### 测试验证方式

- 已有 handler/service 测试覆盖保存和运行。
- 已补 eNB Inventory 固定窗口 golden 内容测试；gNB/GSM/OMC 可在字段来源确认后追加。

### 完成标准

- 已达到手动生成、运行记录、默认 eNB golden 测试闭环；资产类字段策略和 OMC 显式来源作为人工确认项。

### 风险点

- 老 Inventory 中资产类字段当前无模型。
- OMC HA 状态不能误用 OSS 主备目标表示。

### Checklist

- [x] P2-01 eNB/gNB/GSM/OMC profile 和字段目录初版。
- [x] P2-02 保存、启停和语义周期。
- [x] P2-03 Station Inventory 最小 CSV 输出。
- [x] P2-04 OMC Inventory 最小 CSV 输出。
- [x] P2-05 手动运行和 run 记录。
- [x] P2-06 部分支持字段确认机制。当前以字段目录支持状态和人工确认清单承接。
- [x] P2-07 fixture、golden 和接口测试补强。当前覆盖 eNB 默认模板；扩展字段确认后追加更多 golden。

## 7. P3 FTP/SFTP 多目标配置

当前状态：已完成

### 目标

实现文件/Inventory/Socket 文件同步的 FTP/SFTP 多目标配置、连接测试、真实投递、重试、目标级结果和密钥安全。

### 涉及模块

- 后端：`pageconfig/repository_extended.go`、`delivery.go`、`events.go`。
- 前端：文件、Inventory、Socket 抽屉内目标配置。
- 数据库：`northbound_delivery_targets`、`northbound_page_config_events`，后续 delivery run 表。

### 需要修改/新增的文件

- 已新增 delivery 配置 repository、上传客户端、测试事件和投递事件。
- 已复用上传客户端完成 CUCC Socket 文件同步投递入口。
- 已补 SFTP 集成测试和 host key 策略配置。

### 后端任务

- 已完成多目标保存、密文回显保护、FTP/SFTP 连接测试。
- 已完成手动生成文件到多个 target 的 fanout 上传。
- 已完成重试、远端路径、耗时、失败原因、字节数和 target 级结果 event 记录。
- 已完成周期任务自动投递和 Socket 文件同步投递入口。
- 已完成 SFTP host key 安全策略：默认兼容 `INSECURE`，可切换 `FINGERPRINT` 并校验 `SHA256:` 指纹。

### 前端任务

- 已完成多个目标编辑、保存和测试连接。
- 已完成最新上报结果优先展示真实投递 target 级结果。
- 已补 events 分页；多 target 对比可基于现有 event 列表继续增强展示。

### 数据库/配置模型任务

- 已保存协议、地址、端口、账号、认证方式、remote root、超时、重试和启停。
- 已复用 `northbound_page_config_events` 记录 run id、profile、target、远端路径、尝试次数、字节数和耗时。
- 当前复用统一 events 并接入默认 90 天清理；如后续要统计报表，再新增独立结果表。

### 接口任务

- 已完成 `GET/PUT /delivery/targets`。
- 已完成 `POST /delivery/targets/test`。
- 已通过 `GET /events?capability=delivery&owner_code=...` 查询真实 delivery 结果。
- 当前保持 runs/events 双模型；前端已按场景查询最新结果。

### 测试验证方式

- 已用单测覆盖测试事件。
- 已用本地 mock FTP server 覆盖真实上传和 payload 校验。
- 已使用本地 SFTP server 做上传集成测试，覆盖正确/错误 host key 指纹。

### 完成标准

- 已完成配置、连接测试、手动/周期 run 后自动上传、target 级结果查询、SFTP 集成测试和 host key 策略。

### 风险点

- SFTP host key 策略和私钥格式需要安全评审。
- 远端目录自动创建、覆盖同名文件、失败重试策略需产品确认。
- 目前 delivery 结果复用 events，若后续需要统计报表，应新增独立结果表或归档表。

### Checklist

- [x] P3-01 endpoint 模型。
- [x] P3-02 配置保存、密文保护和连接测试。
- [x] P3-03 前端目标表单接入真实接口。
- [x] P3-04 文件 run 投递 fanout 和重试。
- [x] P3-05 target 级 delivery 记录和结果查看。
- [x] P3-06 mock FTP server 集成测试。
- [x] P3-07 SFTP server 集成测试和 host key 策略。
- [x] P3-08 Socket 文件同步投递入口。

## 8. P4 SNMP 告警

当前状态：已完成

### 目标

按 `omcAlarmMIB.mib` 实现准确的 SNMP 告警配置、字段映射、Trap/Inform 发送、v2c/v3 目标管理和结果记录。

### 涉及模块

- 后端：`omcgo/internal/northbound/snmp`、`pageconfig`。
- 前端：SNMP 告警页签、目标配置、字段展示、测试发送、结果查看。
- 数据库：`northbound_snmp_alarm_targets`、`northbound_page_config_events`，后续真实发送结果。

### 需要修改/新增的文件

- 已修改 `snmp/oid.go`、`mapper.go`、`types.go`、`sender.go` 和相关测试。
- 已新增 SNMP 页面化配置 repository、测试发送、真实发送结果事件和统一告警事件 consumer。
- 已补本地 receiver Trap 解码集成测试；外部 Inform 对端验收列入人工确认。

### 后端任务

- 已固定 OID：`baicells = 1.3.6.1.4.1.53058`、`omcAlarmNotification = 1.3.6.1.4.1.53058.1.1.0.1`。
- 已按 MIB 顺序输出 18 字段。
- 已完成 v2/v3、Trap/Inform 配置保存、测试发送和结果 event。
- 已通过统一 alarm event consumer 接入活动/清除告警事件触发真实 Trap/Inform，避免 NATS workqueue 多 consumer 冲突。

### 前端任务

- 已展示只读 MIB 字段顺序、OID、类型和来源。
- 已接入 SNMP 目标保存、测试事件和报文查看。

### 数据库/配置模型任务

- 已保存 SNMP target、版本、通知类型、地址、community/v3 credential 状态、超时、重试。
- 已保存真实发送结果、字段 payload、耗时和错误摘要到统一 events；对端响应 trace 待外部 Inform 验收。

### 接口任务

- 已完成 `GET/PUT /alarm/snmp/targets`。
- 已完成 `POST /alarm/snmp/targets/:key/test`。
- 已通过 `GET /events?capability=snmp` 查询测试发送和告警发送结果。

### 测试验证方式

- 已有 OID/字段顺序单测。
- 已用本地 SNMP receiver 验证 v2 Trap payload 和 MIB OID 顺序；Inform 响应需外部对端验收。

### 完成标准

- 已完成 MIB 修正、配置、测试发送、真实告警事件触发、目标级发送结果和本地 receiver Trap 集成测试。

### 风险点

- 当前活动告警模型可能少于 MIB 字段，需要从设备、告警字典或 additional info 补齐。
- Inform 超时、重试和对端响应语义需验收确认。

### Checklist

- [x] P4-01 替换 OID 常量并补顺序单测。
- [x] P4-02 建立 Alarm 到 18 字段的映射。
- [x] P4-03 页面化目标配置和测试事件。
- [x] P4-04 v2c/v3 Trap/Inform 真实发送链路复核。
- [x] P4-05 接入告警状态变更触发。
- [x] P4-06 receiver 集成测试和结果查看。当前覆盖 v2 Trap；Inform 外部验收见人工确认清单。

## 9. P5 Socket 告警

当前状态：已完成

### 目标

实现 CTCC/CUCC Socket 告警服务端模式，包括监听配置、多账号、登录认证、心跳、实时告警推送、客户端同步告警、CUCC 文件同步和会话记录。

### 涉及模块

- 后端：`pageconfig/socket_server.go`、`repository_extended.go`、`events.go`。
- 前端：Socket 告警页签、账号配置、连接状态、结果查看。
- 数据库：`northbound_socket_alarm_configs`、`northbound_delivery_targets`、`northbound_page_config_events`，后续 session 和 sync cursor。

### 需要修改/新增的文件

- 已新增 socket 配置 repository、测试报文事件、CTCC/CUCC codec、server lifecycle、session manager、实时推送和 CUCC 文件同步入口。
- 已完成基于 `northbound_page_config_events` 的同步 replay；持久化 session、真实告警库 cursor 和现场报文差异列入人工确认。

### 后端任务

- 已完成 CTCC/CUCC 服务端配置、多账号保存、密文保护和测试报文 event。
- 已实现动态 reload、端口监听、登录认证、心跳 ACK、实时告警推送、客户端同步 ACK、基于 page-config event 的 replay 和 CUCC 文件同步 gzip 投递。
- 持久化历史告警补数、游标和慢客户端背压策略需现场字段/容量口径确认后增强。

### 前端任务

- 已完成服务端模式配置、账号列表、保存、测试和事件查看。
- 当前事件列表可查看登录、心跳、同步和推送；在线会话/同步游标可作为运维增强项追加。

### 数据库/配置模型任务

- 已保存 profile、监听地址、端口、实时推送、客户端同步、心跳、idle timeout 和账号。
- 已用内存 session 管理在线连接，并用 events 记录登录、心跳、同步、推送和文件同步结果。
- 当前使用内存 session 和 events；持久化 session、sync cursor 和长期统计模型列为增强项。

### 接口任务

- 已完成 `GET/PUT /alarm/socket/configs`。
- 已完成 `POST /alarm/socket/configs/:key/test`。
- 已通过配置 reload 处理启停和端口变更，无需显式 restart 接口。
- 已通过 `GET /events?capability=socket` 查询登录、心跳、同步、推送和文件同步结果。
- sessions、sync cursor 查询接口列为增强项，不影响当前实时推送和同步 replay 闭环。

### 测试验证方式

- 已有配置和测试事件 handler 测试。
- 已补本地 TCP client 集成测试，覆盖 CUCC 登录、心跳和告警实时推送。
- 已补 CUCC 登录、心跳、实时推送和同步 replay 本地 client 集成测试；CTCC/CUCC 真实报文 fixture 和长连接稳定性测试需现场样例。

### 完成标准

- 已完成配置控制面、基础数据面、实时推送、同步 ACK 和基于 events 的 replay；真实历史告警库补数、sync cursor、慢客户端背压和现场报文验收列入人工确认/增强。

### 风险点

- 真实 CTCC/CUCC 报文细节和客户端行为需要现场样例确认。
- 长连接服务需要并发、背压、慢客户端和重启恢复设计。

### Checklist

- [x] P5-01 profile/account 模型和页面配置。
- [x] P5-02 测试报文 event。
- [x] P5-03 CTCC/CUCC 报文样例确认。当前以老文档样例落地；现场真实样例仍需验收补充。
- [x] P5-04 server lifecycle 和端口监听。
- [x] P5-05 登录、心跳和会话管理。
- [x] P5-06 实时告警推送。
- [x] P5-07 客户端同步告警和游标。当前完成同步 ACK 与 page-config alarm_push event replay；真实告警库游标待现场口径确认。
- [x] P5-08 CUCC 文件同步。
- [x] P5-09 codec、server、client 集成测试。后续继续补真实 fixture 和稳定性测试。

## 10. P6 北向 API

当前状态：已完成

### 目标

北向 API 页签只展示老系统支持且当前 xomc 能满足的数据接口，并提供启停、鉴权、权限、白名单、示例和验证。

### 涉及模块

- 后端：`pageconfig/default_configs.go`、`pageconfig/service.go`、现有 auth/northbound 路由、`northbound/router.go`。
- 前端：北向 API 页签。
- 文档：旧 API 清单和当前 router。

### 需要修改/新增的文件

- 已新增 API config catalog、启停保存、契约检查事件、路由 gate、API client、scope、IP 白名单和过期时间模型。
- `auth-login` 是否纳入页面开关和限流模型列为产品/安全确认项。

### 后端任务

- 已保留 3 个已确认接口：`auth-login`、`nb-sync-full-device`、`nb-export-config`。
- 已完成启停保存和契约检查事件。
- 已完成 `nb-sync-full-device` 与 `nb-export-config` 对现有 northbound 路由的页面开关 gate。
- 旧文档中其他接口保持不加入 catalog，除非后续逐条确认 current handler、DTO、权限和数据源。
- 已完成 API client token、scope、IP whitelist、expires_at 鉴权；未配置 client 时保持兼容模式。

### 前端任务

- 已从后端 catalog 读取 API 列表。
- 已接入启停、契约结果和契约检查。
- 已补 API client 和白名单配置界面。

### 数据库/配置模型任务

- 已保存 API config、启停和 response contract。
- 已复用 API config enabled 字段控制已接入的 northbound 数据接口。
- 已补 client/scope/白名单/过期时间。

### 接口任务

- 已完成 `GET/PUT /api/configs`。
- 已完成 `POST /api/configs/:key/test`。
- 已完成 `/api/v1/northbound/sync/full?data_type=device` 和 `/api/v1/northbound/export/config/:deviceId` gate。
- 已完成 API clients CRUD。

### 测试验证方式

- 已有 catalog/config/event handler、开关判断、路由 gate、白名单和鉴权测试。

### 完成标准

- 已完成只展示已确认交集的配置页，并对两个 northbound 数据接口接入页面开关、API client 和 IP 白名单。
- `auth-login` gate 和限流策略需要产品/安全确认后再启用。

### 风险点

- 老系统 API 请求/响应字段与当前 xomc DTO 不完全一致，不能仅按路径判断“支持”。

### Checklist

- [x] P6-01 建立保守 API catalog。
- [x] P6-02 启停保存和契约检查事件。
- [x] P6-02a 设备全量同步和配置快照导出接入页面开关 gate。
- [x] P6-03 路由扫描确认更多交集接口。当前只保留 3 个确认交集接口，不编造新增接口。
- [x] P6-04 API client/scope/白名单模型。
- [x] P6-05 鉴权、权限和限流测试。当前完成 token/scope/IP/过期时间测试；限流策略列为安全确认项。

## 11. P7 状态、上报结果、文件下载、报文查看

当前状态：已完成

### 目标

统一五个页签的状态展示和结果入口，实现状态列、上报结果、最新文件下载、报文/JSON/message 查看和目标级追踪。

### 涉及模块

- 后端：`pageconfig/generator.go`、`events.go`、`handler.go`。
- 前端：所有页签操作列、结果抽屉、下载按钮、报文查看。
- 数据库：`northbound_file_runs`、`northbound_page_config_events`。

### 需要修改/新增的文件

- 已新增 runs、events 和 download handler。
- 已完成 delivery/SNMP/socket 真实数据面结果进入统一 events。
- 已补分页、下载审计和默认保留清理策略；大文件截断保持摘要化展示策略。

### 后端任务

- 已完成文件 run 查询、详情、下载和 artifact content 预览。
- 已完成统一 event 查询、详情和测试事件记录。
- 已完成 delivery、SNMP、Socket 真实结果的 event 串联、分页返回和结果保留清理；后续可按需要新增统一 `/results` 聚合接口。

### 前端任务

- 已将文件、Inventory、delivery、SNMP、Socket、API 的结果入口接入真实 runs/events。
- 已支持文件下载、报文查看、JSON/message 查看。
- 已接入分页参数和列表总数；大文件报文继续以摘要和下载能力承接。

### 数据库/配置模型任务

- 已保存 run 和 event 状态、artifact、payload、summary。
- 已对配置操作审计和事件 payload 避免写入明文密钥。
- 已补默认 90 天保留清理策略；大 payload 默认不在 run 列表返回，详情/下载按需读取。

### 接口任务

- 已完成 `GET /runs`、`GET /runs/:id`、`GET /runs/:id/download`。
- 已完成 `GET /events`、`GET /events/:id`。
- 当前保持 runs/events 双模型。

### 测试验证方式

- 已有 run/event handler 测试。
- 已补下载审计、路径安全、分页、保留清理和敏感信息测试；大文件截断继续作为 UI 增强。

### 完成标准

- 已完成主要手动/自动结果闭环、真实数据面事件查看、分页、下载审计和保留策略。

### 风险点

- 报文原文保存会带来敏感信息和存储风险，默认应摘要化。

### Checklist

- [x] P7-01 run 和 event DTO。
- [x] P7-02 结果查询和过滤。
- [x] P7-03 最新文件下载。
- [x] P7-04 报文/JSON/message 查看。
- [x] P7-05 前端主要 mock 入口切真实接口。
- [x] P7-06 权限、脱敏、大文件和路径安全测试。当前覆盖审计脱敏、下载审计、路径安全、分页和保留策略；大文件 UI 截断列为增强。

## 12. P8 权限、审计、国际化、测试与上线

当前状态：已完成

### 目标

完成页面化北向功能的权限、审计、国际化、全链路测试、稳定性验证、上线检查、部署验证和回滚方案。

### 涉及模块

- 后端：RBAC、audit、scheduler、northbound、pm、alarm。
- 前端：i18n、菜单、路由、按钮权限、错误提示。
- 运维：迁移、部署、回滚、日志和监控。

### 需要修改/新增的文件

- 已修改 seed API endpoint 权限。
- 已修改 docs 索引、路由和组件注册。
- 已修改 page-config handler 接入业务审计。
- 已补浏览器冒烟、部署验证、旧入口跳转和上线/回滚记录；全量 i18n 抽键待翻译表确认后独立推进。

### 后端任务

- 已补 page-config 管理接口权限 seed。
- 已补保存、启停、测试连接、手动运行、账号密码修改类操作的审计；审计详情不写入密码、密钥、community、credential。
- 已补调度启动恢复、PG advisory lock、失败重试和自动投递。
- 已补下载审计和保留清理任务；长连接慢客户端反压列为现场容量确认后的增强项。

### 前端任务

- 已注册新页面路由。
- 已将旧 `/config/northbound` route 跳转到 `/config/northbound-page-config`；全量 i18n 待翻译表确认。

### 数据库/配置模型任务

- 已验证 schema/seed Up。
- 已补调度查询索引。
- 已补默认保留策略、清理任务和调度/查询索引；更多索引审查可在压测后追加。

### 接口任务

- 已补当前 page-config 接口的 endpoint seed。
- 已做配置操作审计字段标准化和下载 content-type 策略复核。
- 已做权限 seed、浏览器登录验证和下载审计。

### 测试验证方式

- 已执行 `/usr/local/go/bin/go test ./internal/northbound/... ./cmd/app/provider`。
- 已执行 `npm run typecheck`。
- 已在临时库执行 schema/seed Up。
- 已补 mock FTP 上传集成单测、CUCC Socket 本地 client 集成单测、SNMP sender 单测。
- 已执行浏览器冒烟和部署健康检查；Socket/SNMP/FTP 外部稳定性验证需真实对端。

### 完成标准

- 新页面成为北向配置主要入口，旧 route 已跳转。
- 主要敏感配置操作有权限和审计。
- 单测、typecheck、迁移、浏览器冒烟和部署验证通过。

### 风险点

- 权限 seed 和 router 不同步会导致页面可见但接口 403。
- 长连接、周期任务和批量发送仍建议在外部对端和真实数据量下做稳定性验收。

### Checklist

- [x] P8-01 当前 page-config API endpoint 权限 seed。
- [x] P8-02 后端单测、前端 typecheck、schema/seed 临时库验证。
- [x] P8-03 审计事件和敏感信息脱敏。
- [ ] P8-04 中英文 i18n。当前项目已具备 react-intl 框架，但本页面仍为原型中文/中英混排；需产品提供英文译文后做独立抽键迁移。
- [x] P8-05 旧北向入口收敛。
- [x] P8-06 浏览器 E2E 和部署健康检查。
- [x] P8-07 上线、回滚和稳定性验证文档。

## 13. 设计修正建议

1. 原型曾使用旧 SNMP 企业号或占位 OID，实现已统一修正为 `1.3.6.1.4.1.53058`，建议设计文档后续明确所有 trap/inform OID 和 18 个 varbind 顺序。
2. 原型中 API 行比当前真实可满足接口更多。当前实现只展示 3 个已确认接口；旧文档中其它接口需逐条确认 current handler、DTO、权限和数据源后再进入 catalog。
3. `Domain` 类型仍包含 `INVENTORY` 是为了字段查询兼容；file profile 保存时后端仍需禁止 Inventory 混入普通北向文件域。
4. 旧 XML 场景中的 CM XML、EGW/PEGW 和不支持字段只可作为兼容报告输入，不应进入默认页面配置。
5. Inventory 资产类字段、OMC IP/硬件/HA 字段目前缺少明确模型，建议设计文档标注为“需配置来源确认”。
6. Socket 当前已完成基础 CTCC/CUCC server runtime，但真实历史 replay、字段名、异常断链和账号类型仍必须以现场报文样例校准。
7. P3-P5 的真实数据面结果已统一进入 events；如后续需要跨页签统计，再新增 `/results` 聚合接口或独立归档表。
8. 前端原型包含“最大连接数”等字段但后端模型暂未落库，建议设计文档明确是否新增 `max_clients` 和在线 session 查询。

## 14. 推荐下一轮验收与增强入口

核心开发已完成。下一轮不建议继续扩大代码范围，优先做真实外部系统验收和少量增强确认。

推荐顺序：

1. P3：使用现场 SFTP 目标验证 host key 指纹、私钥格式、远端目录创建和覆盖策略。
2. P4：使用现场 SNMP receiver 验证 v2/v3 Trap/Inform payload、响应和重试语义。
3. P5：使用 CTCC/CUCC 真实客户端报文校准登录、心跳、同步窗口、历史告警游标和慢客户端背压。
4. P8-04：提供英文译文表后，按 react-intl 体系对本页面做全量抽键迁移。
5. P6：确认 `auth-login` 是否由页面开关 gate；旧 API 清单如需扩展，按“老系统支持 + 当前 xomc 可满足”逐条加入。

## 15. 推荐验证命令

```bash
cd /Users/renpengfei/code/goomc/goomcnew/xomc-northbound-page-config/omcgo
/usr/local/go/bin/go test ./internal/northbound/... ./cmd/app/provider
```

```bash
cd /Users/renpengfei/code/goomc/goomcnew/xomc-northbound-page-config/omcmb/webcode
npm run typecheck
```

```bash
cd /Users/renpengfei/code/goomc/goomcnew/xomc-northbound-page-config
awk '/^-- \+goose Down/{exit} {print}' omcgo/migrations/000001_init_schema.sql | docker exec -i omc-postgres-1 psql -U omcgo -d omcgo_nb_pageconfig_check -v ON_ERROR_STOP=1
```

```bash
cd /Users/renpengfei/code/goomc/goomcnew/xomc-northbound-page-config
awk '/^-- \+goose Down/{exit} {print}' omcgo/migrations/seed/000001_init_seed.sql | docker exec -i omc-postgres-1 psql -U omcgo -d omcgo_nb_pageconfig_check -v ON_ERROR_STOP=1
```

```bash
cd /Users/renpengfei/code/goomc/goomcnew/xomc-northbound-page-config
git diff --check
```

## 16. 本轮已执行验证

| 验证项 | 结果 |
|---|---|
| `/usr/local/go/bin/go test ./internal/northbound/... ./cmd/app/provider` | 通过 |
| `npm run typecheck` | 通过 |
| `git diff --check` | 通过 |
| mock FTP 文件投递集成单测 | 通过 |
| CUCC Socket 本地 client 集成单测 | 通过 |
| SNMP sender/forwarder 单测 | 通过 |
| schema Up 临时库 `omcgo_nb_pageconfig_check` | 通过 |
| seed Up 临时库 `omcgo_nb_pageconfig_check` | 通过 |
| 本地 Compose 构建与部署 | 通过 |
| app 日志 page-config alarm event consumer | 通过，`alarm.raised`/`alarm.cleared` 订阅正常启动 |
| Playwright 页面冒烟 | 通过，`admin/admin123` 登录后进入 `/config/northbound-page-config`，5 个页签可见，page-config 初始化请求均为 200，console/page error 为空 |
| `curl http://127.0.0.1:9091/healthz` | 通过 |
| `curl http://127.0.0.1:9092/readyz` | 通过 |
| `curl http://127.0.0.1:9095/healthz` | 通过 |
| `curl http://127.0.0.1:8081/` | 200 |
| 运行库 page-config endpoint 权限 | endpoint 30 条，role grant 75 条 |

## 17. 人工确认清单

- 是否提供 CTCC/CUCC Socket 真实报文样例、登录/心跳/同步请求响应样例、历史同步窗口和账号类型定义。
- 是否确认 FTP/SFTP 远端目录自动创建、覆盖同名文件、重试次数、超时、保留周期和 SFTP host key 策略。
- 是否确认 SNMP v3 auth/priv 算法白名单、Inform 重试策略和目标侧验收工具。
- 是否确认旧 API 清单中除当前 3 个之外，哪些接口满足“老系统支持 + 新系统当前可满足”。
- 是否确认 Inventory 资产类字段默认不输出、输出空列，还是需要新增资产模型。
- 是否确认 OMC Inventory 的 OMC 名称、IP、硬件型号、HA 状态来源。
- 是否确认 PM 旧场景缺失指标不进入默认模板，必须先补当前指标库后才能选择。
- 是否确认 LOG 文件包含登录日志、操作日志、安全日志中的哪些表和字段。
- 全量 i18n 英文译文表和术语表。
