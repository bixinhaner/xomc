# CONTEXT.md — OMC 统一语言术语表

> 本文件是 OMC 项目的**统一语言（ubiquitous language）词典**：集中定义每个领域词在本项目里"指什么"，让人与 AI 用同一套词汇沟通。
> 这是一份**术语表，不含实现细节** —— 只解释概念，不写代码路径、不教怎么实现（那些归 `CLAUDE.md` / `omcgo/CLAUDE.md`）。
> **Agent 探索代码前应先读本文件**，建立领域心智模型再动手。
>
> 按主题分组。每条 1-2 句。

---

## 协议与南向（TR-069）

| 术语 | 含义 |
|------|------|
| **TR-069 / CWMP** | Broadband Forum 定义的 CPE 广域网管理协议（CPE WAN Management Protocol），OMC 管理小基站的核心南向协议，承载于 SOAP/XML over HTTP。 |
| **ACS** | Auto-Configuration Server，自动配置服务器，即网管侧的 TR-069 服务端，负责接收设备上报、下发配置与指令。OMC 的 ACS 是独立部署单元。 |
| **CPE** | Customer Premises Equipment，用户驻地设备，在 OMC 语境下即被管理的小基站本身（TR-069 客户端）。 |
| **Inform** | CPE 主动发起会话时发送的首条 RPC，携带设备标识与触发原因（事件码）。每次会话都以 Inform 开场。 |
| **Inform 事件码** | Inform 中标明触发原因的码值，可同时携带多个：**BOOTSTRAP**（首次接入/恢复出厂后初次上报）、**BOOT**（设备重启）、**PERIODIC**（周期性定时上报）、**CONNECTION REQUEST**（响应网管主动连接请求）。 |
| **SOAP / XML** | Inform 与所有 RPC 的报文封装格式：SOAP 1.1 信封 + CWMP 命名空间，body 为 XML。OMC 发送侧用预编译模板，接收侧流式解析。 |
| **RPC** | Remote Procedure Call，TR-069 定义的远程方法调用，分 ACS→CPE 与 CPE→ACS 两个方向（见下表）。 |
| **GPV（GetParameterValues）** | 读取设备参数当前值的 RPC。 |
| **SPV（SetParameterValues）** | 写入/修改设备参数值的 RPC。 |
| **GPN（GetParameterNames）** | 枚举设备参数树节点名的 RPC。 |
| **SPN（SetParameterAttributes）** | 设置参数属性（如主动上报通知标志）的 RPC。 |
| **AddObject / DeleteObject** | 在参数树中动态增删对象实例的 RPC（如新增一个邻区配置实例）。 |
| **Download / Upload** | 触发 CPE 从指定 URL 下载（固件/配置）或向指定 URL 上传（日志/配置/PM/MR 文件）的 RPC。 |
| **Reboot / FactoryReset** | 远程重启 / 恢复出厂设置的 RPC。 |
| **TransferComplete** | CPE 完成一次 Download/Upload 后回报结果的 RPC（CPE→ACS 方向）。 |
| **Connection Request** | 网管主动"敲门"通知 CPE 立即发起会话的机制（HTTP 请求设备侧 URL）；部分设备不支持，需降级为轮询。 |
| **Session（会话）** | 一次完整的 TR-069 交互，从 Inform 开始、经若干 RPC 事务、以空响应结束。会话是有状态的（状态机驱动），状态存于 Redis 并带 TTL。 |

---

## 设备与无线制式

| 术语 | 含义 |
|------|------|
| **小基站** | 低功率、小覆盖的蜂窝基站统称，OMC 的核心管理对象。按覆盖与功率细分为下面三类。 |
| **皮基站（Pico）** | 覆盖更小、功率更低的小基站，典型用于室内热点/企业场景。 |
| **微基站（Micro）** | 覆盖介于宏站与皮站之间的小基站，典型用于城区补盲/容量补充。 |
| **eNB** | E-UTRAN NodeB，LTE（4G）制式的基站网元。 |
| **gNB** | Next Generation NodeB，5G NR 制式的基站网元。 |
| **GSM** | 2G 制式，OMC 也覆盖部分 GSM 小基站（如 BSC/BTS）。 |
| **LTE** | Long Term Evolution，4G 制式。OMC 内部制式常量 `lte`。 |
| **NR (SA)** | New Radio（Standalone），5G 独立组网制式。OMC 内部制式常量 `nr`。 |
| **neType（网元类型）** | 区分设备所属网元种类的标识，告警定义按 neType 归属（不同网元类型有不同的告警字典）。 |

---

## 运营商

| 术语 | 含义 |
|------|------|
| **cmcc** | 中国移动（China Mobile）的运营商代号。 |
| **ctcc** | 中国电信（China Telecom）的运营商代号。 |
| **cucc** | 中国联通（China Unicom）的运营商代号。 |
| **运营商差异** | 三家在参数映射、告警规范、KPI 定义、开站流程、接口协议上的不同；OMC 把所有差异收敛到 Carrier 适配点，业务层不感知具体运营商。 |
| **Carrier 接口** | 运营商适配的统一抽象。每家运营商是一个适配器实现，新增运营商即新增适配器，核心逻辑不变；严禁在业务代码里硬编码"如果是某运营商"。 |

---

## 功能域 F01–F10

| 编号 | 功能域 | 一句话职责 |
|------|--------|-----------|
| **F01** | 南向接口（TR-069） | ACS 引擎，处理 SOAP/XML 协议与设备会话。 |
| **F02** | 数据模型与配置 | 参数模型字典、产品装配件、配置模板与基线。 |
| **F03** | 性能管理（PM/KPI） | 采集计数器、计算 KPI、时序存储与多级聚合。 |
| **F04** | 告警管理 | 告警接收、去重、关联、生命周期管理。 |
| **F05** | 测量报告（MR） | MRO/MRS/MRE 测量文件的采集与解析。 |
| **F06** | OMC-R 核心 | 设备、用户、拓扑、固件、备份、仪表盘、运维、报表、MML、文件、日志、许可等运维主体功能。 |
| **F07** | 网元直连 | 网元与网管之间的直连通道。 |
| **F08** | 北向/OSS 接口 | 向上游 OSS 系统开放数据与事件。 |
| **F09** | 自动开站 | 设备零接触发现、模板匹配、配置下发、激活。 |
| **F10** | 互操作测试 | 设备联调与协议一致性验证。 |

---

## F02 参数模型与产品装配件

| 术语 | 含义 |
|------|------|
| **参数模型字典（ParamModel）** | 描述某类设备参数空间的字典：包含默认参数映射与标准路径元属性，是 OMC 理解设备参数的基准。 |
| **产品装配件（product）** | 把"产品类 + 参数模型 + KPI 平台 + 告警网元类型 + 上传开关"聚合到一起的产品级配置单元，是设备落到正确字典的入口。 |
| **ProductRegistry** | 产品装配件的路由器：用全局正则匹配设备上报的 productClass，定位到对应 product。 |
| **productClass** | 设备在 Inform 中上报的产品类标识，是路由到产品装配件的匹配键。 |
| **standardPath** | 标准化参数路径，OMC 内部统一使用、与厂商无关的参数命名（模板/SPV/GPV 输入侧用它）。 |
| **privatePath** | 厂商专有参数路径，设备实际识别的命名（下发与持久化用它）。 |
| **Translator** | standardPath ↔ privatePath 的双向翻译器，屏蔽厂商参数命名差异。 |
| **param_mappings** | 参数模型的**默认映射**集合（standardPath ↔ privatePath 的基准对照）。 |
| **discovered_param_mappings** | 从设备实际上传的参数 XML 中"发现"并落库的映射，按产品 + 软件版本索引，精度高于默认映射，匹配时优先采用。 |

---

## F03/F05 性能与测量

| 术语 | 含义 |
|------|------|
| **PM** | Performance Management，性能管理。设备按 3GPP 规范周期产出性能计数器文件，OMC 采集解析后入时序库。 |
| **KPI** | Key Performance Indicator，由原始计数器按公式计算得到的关键性能指标，支持设备→站点→区域→网络多级聚合。 |
| **指标库（Indicator）** | KPI 计算所依据的指标定义字典（指标名、公式、聚合方式等），按平台组织。 |
| **MR** | Measurement Report，测量报告。设备上报的无线测量数据文件，分 MRO/MRS/MRE 三类。 |
| **MRO / MRS / MRE** | 三种测量报告类型（Original / Statistics / Event 等口径），OMC 分别采集解析。 |
| **RSRP** | Reference Signal Received Power，参考信号接收功率，衡量信号强度的测量值。 |
| **RSRQ** | Reference Signal Received Quality，参考信号接收质量，衡量信号质量的测量值。 |
| **SINR** | Signal to Interference plus Noise Ratio，信号与干扰加噪声比，衡量信号纯净度的测量值。 |

---

## F04 告警

| 术语 | 含义 |
|------|------|
| **告警定义（alarm definition）** | 告警字典：规定每种告警的编码、含义、级别、处理规则等。按 neType（网元类型）归属，不同网元类型有各自的告警集合。 |
| **活动告警** | 当前未清除、处于活跃状态的告警实例，需常驻可快速查询的存储。 |
| **告警生命周期** | 一条告警从产生（raised）→ 确认（acknowledged）→ 清除（cleared）的状态流转。 |

---

## 字典三库与导入范式

| 术语 | 含义 |
|------|------|
| **装配件三库** | F02/F03/F04 共享同一套字典导入范式的三个字典库：**ParamModel**（参数模型）、**Indicator**（指标/KPI）、**Alarm**（告警定义）。 |
| **单目录 + sidecar** | 三库导入的统一范式：内置（builtin）与自定义（custom）字典文件同住一个目录，来源靠一个 sidecar 空标记文件区分（仅 custom 可删，builtin 不可删）。 |
| **builtin / custom** | 字典来源标记：builtin 为随系统出厂的内置字典，custom 为用户上传的自定义字典。 |

---

## 跨域基础设施

| 术语 | 含义 |
|------|------|
| **EventBus** | 事件总线，模块间异步解耦的统一抽象。事件主题按 `领域.动作.细节` 分层命名；底层有进程内与跨实例两种实现。 |
| **统一任务队列（task）** | 所有需要派发到设备的指令（MML、固件、配置同步、Reboot 等）的统一队列，维护"任务 ↔ CWMP 会话 ID"映射与完成回调。 |
| **UFTE** | 统一文件传输/任务引擎（Unified File Transfer/task Engine），承载备份、日志采集、配置恢复等"产出文件"类的输出型任务。 |
| **Notification（通知中心）** | 实时告警、批量同步、邮件/SMS/Webhook 等对外通知的统一出口。 |
| **Transfer（文件传输桥）** | Download/Upload 与 ACS、应用之间的文件传输中介，服务 PM/MR/软件升级/备份等场景。 |

---

## F09 自动开站与北向

| 术语 | 含义 |
|------|------|
| **自动开站 / 零接触部署** | 设备首次上电首次 Inform 后，由系统自动完成"发现→模板匹配→参数下发→激活确认"，无需现场人工逐项配置。 |
| **北向 / OSS** | 北向接口指 OMC 向上游 OSS（Operations Support System，运营支撑系统）开放数据与事件的方向（相对"南向"面向设备而言）。 |

---

## 系统架构

| 术语 | 含义 |
|------|------|
| **模块化单体** | OMC 后端的架构选择：单一进程内按功能域内聚成模块，不拆微服务；规模增长时按功能域渐进拆分。 |
| **三部署单元** | 后端的三个独立进程：**app**（主应用，承载 F02–F10 与管理面 API）、**acs**（独立的 TR-069 ACS 引擎，可水平扩展）、**worker**（后台工作进程，处理 PM/MR 文件与 KPI 计算等）。 |
