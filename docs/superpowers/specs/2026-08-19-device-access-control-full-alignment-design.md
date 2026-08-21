# 设备接入控制老项目能力完整对齐开发方案

> 状态：方案已评审，开发代码复核中；不得据此声明开发或验收完成
>
> 日期：2026-08-19
>
> 现状基线：`main@dea9ca5bc`，104 环境历史基线 `v100.0.0-20260819-0500`
>
> 业务基线：`old-omc-docs/feature-rebuild/01-基站接入控制.md`
>
> 验收基线：`docs/test/20260819-设备接入控制标准验收流程.md`

## 1. 背景与结论

104 已经完成接入判定、名单优先级、策略发布、普通任务门禁、拒绝后 RF 隔离、放行后 RF 恢复、决定证据和幂等动作等核心闭环，但尚不能宣布完整覆盖老项目能力。

本方案只处理老文档最低闭环中尚未对齐的内容，分为两类：

1. 必须补实现：批量治理、完整身份契约、动作可靠性、事件与审计闭环；
2. 必须补验收：规则维度反例、候选审核、关闭态真实 Inform、并发/重启、受限角色、原始 CWMP 报文。

邮件短信的具体通道、复杂多级审批、高级策略编排和 EGW/ECI 接入控制不进入本轮开发范围，但事件输出、通知状态关联和通知失败不影响安全动作属于本轮范围。

## 2. 目标与非目标

### 2.1 目标

- 等价覆盖老项目黑名单、白名单的模板下载、批量导入、追加、覆盖和批量删除能力；新系统的“删除”等价实现为可审计的批量停用。
- 支持名单及规则内 SN/TAC/ECGI/IP/GPS 维度的模板下载、导入、导出、预览、失败明细、原子提交和整批回滚。
- 建立可解释、可审计的统一设备身份快照，覆盖运营商、SN、设备编码、CloudKey 兼容信息、OUI、ProductClass、来源 IP 和 Inform 来源。
- 把设备动作补全为可恢复状态机，支持自动重试、离线等待、进程重启恢复、死信和人工修复。
- 输出完整接入事件，特别是 `DEVICE_ACCESS_ACTION_FAILED`，并与统一通知历史关联。
- 审计页面支持按 SN、运营商、时间、决定结果和原因码查询，展示决定证据、动作尝试和通知状态。
- 完成老文档要求的失败路径、并发路径、权限和原始 SOAP 验收。
- 收敛首次 Inform、BOOT、断链重连、上线恢复、名单/策略变更、候选审核和手工重评等全部触发入口。
- 固定特殊平台、名单、规则优先级、无匹配默认动作和系统异常失败策略，不再依赖代码顺序或数据库返回顺序。
- 保证电子围栏、位置移动等下游模块不能激活尚未通过或已被拒绝的设备。

### 2.2 非目标

- 不重写已经通过验收的策略求值器、任务门禁和 RF 所有权恢复逻辑。
- 不恢复老项目 JSP 页面或旧 V2/V3 皮肤。
- 不把 Redis 作为最终状态源。
- 不在本轮建设邮件、短信供应商通道；只完成事件和通知状态契约。
- 不纳入老项目独立 EGW/ECI 接入控制。
- 不允许通过导入直接修改已发布策略或物理删除历史安全记录。

## 3. 当前实现基线

当前 `deviceaccess` 模块已经具备：

- 运营商级业务总开关；
- deny、allow、revoked 名单及固定优先级；
- 策略草稿、发布、退役和规则版本；
- SN、TAC、ECGI、来源 IP、GPS 条件；
- `available/missing/collection_failed/system_error` 证据状态；
- 决定、条件检查、动作和 Outbox 持久化；
- RF SPV、GPV 回读、动作幂等键、尝试次数和人工重试；
- 未知设备候选审核；
- 运营商和设备组权限隔离；
- 接入状态、决定证据和动作页面。

方案评审时确认的主要缺口（大部分已有工作区实现，最终状态以开发复核报告和正式验收证据为准）：

- 名单只有单条写入，页面批量停用为客户端逐条调用；
- 没有导入模板、预览、追加/覆盖、失败明细和批次回滚；
- 接入身份快照未完整呈现设备编码和 CloudKey 兼容结果；
- 动作失败后主要依赖人工重试，没有独立尝试明细、自动退避、死信和重启扫描；
- 没有 `DEVICE_ACCESS_ACTION_FAILED` 事件；
- 审计筛选没有完整时间区间、结果/原因码和通知状态；
- 通知历史当前主要按告警关联，缺少接入决定/动作的通用关联字段。

### 3.1 老项目核心能力强制映射

本轮开发不以“新页面能用”为完成标准，而以老项目核心业务逐项等价为完成标准。老 JSP、老项目文档和 80 环境已确认的能力必须按下表闭环；允许改变技术实现和页面布局，不允许删除业务语义。

| 老项目核心能力 | 新系统等价方案 | 完成门槛 |
|---|---|---|
| 接入控制总开关 | 现有运营商运行时开关 | 开启执行决策；关闭时真实 Inform 兼容放行且不产生探测/动作 |
| 全部触发入口 | 统一接入重评协调器 | 首次 Inform、BOOT、重连、上线恢复、名单/策略变更、候选审核、手工重评均有稳定 trigger type 和幂等键 |
| 黑名单、白名单单条/多 SN 新增 | 现有名单写入扩展为服务端批量写入 | 多 SN 校验、幂等、运营商隔离和审计通过 |
| 黑白名单查询 | 扩展现有名单分页查询 | 支持名单类型、SN、设备名称、状态筛选，结果不跨运营商 |
| 白名单设备名称 | 名单查询关联资产名称，保留 ProductClass/型号 | 列表和停用确认同时展示 SN 与设备名称，未知资产明确显示未登记 |
| 名单模板下载 | 新增官方名单 CSV 模板 | 下载文件能直接回导 |
| 名单追加/覆盖导入 | 新增预览、append/replace 原子提交 | 重复、非法、部分错误、覆盖结果和失败明细通过 |
| 名单单条/批量删除 | 单条/批量停用，保留历史 | 一次服务端事务、二次确认、审计和重评通过 |
| 从接入结果加入黑/白名单 | 状态行快捷处置，复用名单 Service | 二次确认、权限、名单优先级、重评及 RF 动作闭环通过 |
| 规则新增、查看、编辑、删除、启停 | 现有草稿、发布、退役生命周期 | 新建/编辑/发布/退役及回滚均有版本证据 |
| 规则明确优先级 | 规则增加不可歧义的 `priority` 并支持草稿内排序 | 相同范围多规则按优先级稳定求值，不能依赖数据库或数组偶然顺序 |
| 规则指定 SN 范围 | 现有 list/prefix/range，加规则内 SN 维度导入导出 | 手工维护、追加、覆盖、清空、模板下载和导出通过 |
| TAC 规划值维护 | 现有 TAC 条件，加维度导入导出 | 手工、追加、覆盖、清空、导出及正反例通过 |
| ECGI 规划值维护 | 现有 ECGI 条件，加维度导入导出 | 手工、追加、覆盖、清空、导出及正反例通过 |
| IP 范围维护 | 增加 `ip_range` 操作符，同时保留现有 CIDR | 老 IP 起止范围无损表达，不能用单一 CIDR 近似，正反例通过 |
| GPS 范围维护 | 增加矩形 `within_bounds`，同时保留现有中心点/半径 | 老经纬度起止范围无损表达，不能转换成近似圆，导入导出及边界正反例通过 |
| “无邻区允许” | 规则级明确字段，作用于 TAC/ECGI 证据缺失 | 缺失可放行，不匹配不可借此放行 |
| “无 GPS 允许” | 现有 `allow_missing` 明确展示与审计 | 缺失可放行，读取失败/非法坐标按契约处理 |
| 规则内多维度组合 | 保持同一规则内 AND | TAC/ECGI/IP/GPS 任一必需条件失败即该规则失败 |
| 多规则匹配 | 保持启用规则之间 OR | 第一规则失败、第二规则成功时接受并记录第二规则 |
| 黑名单/白名单直通语义 | 黑名单、白名单均在普通接入规则之前终止求值 | 黑名单跳过模板并直接去激活小区；白名单跳过模板并直接允许激活；双名单冲突时黑名单优先 |
| 无规则匹配 | 固定拒绝 | 页面和决定历史展示 `no_applicable_rule`；未知设备需要人工处理时进入独立候选审核流程 |
| 特殊平台/免校验设备 | 通过身份/凭据策略显式配置，不硬编码 | 只对明确范围放行并记录绕过原因 |
| 系统异常安全策略 | 固定 `fail_closed` | 读取失败、系统错误达到采集期限后安全拒绝，任务门禁和 RF 语义明确 |
| 接入状态查询 | 扩展现有状态与决定页面 | 按规则、SN、控制方式、结果、时间查询 |
| 按规则查看接入详情 | 状态查询支持 `matched_rule_id/policy_version_id` 并从规则页跳转 | 展示该规则命中和失败的设备、证据与时间，运营商范围一致 |
| 规划值与上报值 | 保持 decision checks/evidence | 同屏展示规划值、实际值、来源、时间和失败原因 |
| 接受/拒绝统计 | 状态页增加当前筛选条件下汇总 | 汇总与列表使用同一筛选和权限范围 |
| 删除接入状态记录 | 改为可审计归档，不物理删除决定证据 | 默认列表隐藏归档项，可查询恢复；记录操作人、原因和时间 |
| 拒绝后设备去激活 | 现有异步 RF 隔离并补可靠性 | 决定、SPV、GPV 回读、失败/重试分开展示 |
| 位置/围栏联动 | 下游 RF 开启动作增加接入状态硬门禁 | 未接受、已拒绝或已吊销设备不能被围栏或位置流程重新激活 |
| 名单/策略变更后重新判断 | 现有重评 Outbox | 受影响设备异步重评，重复消息不重复动作 |
| 通知 | 统一事件与通知历史 | 通知失败不改变拒绝决定，也不阻塞 RF 安全动作 |
| 老名单和规则数据延续 | 使用同一批量治理能力提供旧数据迁移预演 | 数量、内容摘要、启停状态和运营商归属核对一致，差异有报告 |

老页面中的独立 EGW/ECI 接入控制继续明确排除，不得把它混入本次基站接入控制“遗漏项”。除该书面范围排除外，上表任一行未实现或未真实验收，都不能宣称覆盖老项目核心功能。

## 4. 总体架构

```text
Inform / 手工重评 / 策略变更 / 名单变更
                    |
                    v
             统一身份解析与快照
                    |
                    v
              策略求值与决定落库
                    |
             同事务写接入 Outbox
                    |
          +---------+----------+
          |                    |
          v                    v
    动作规划/执行器          接入事件消费者
          |                    |
   尝试明细/回读/重试       统一通知历史
          |
   成功 / 重试等待 / 死信
          |
          v
       审计聚合查询
```

名单和策略批量治理独立形成安全变更批次：

```text
模板下载 -> 上传解析 -> 预校验 -> 影响预览
         -> 原子提交 -> 重评 Outbox -> 可整批回滚
```

## 5. 设计一：批量名单与策略治理

### 5.1 业务契约

导入类型：

- `access_list`：deny、allow、revoked 名单；
- `rule_dimension`：某一策略草稿规则内的 SN、TAC、ECGI、IP 或 GPS 规划值；
- `policy_bundle`：完整策略草稿、规则和条件，属于增强能力，不作为老项目核心等价的前置条件。

导入模式：

- `append`：保留当前有效数据，对文件中的数据新增或幂等更新；
- `replace`：以文件内容形成新的完整有效集合，经确认后原子切换；
- `rollback`：恢复提交前集合或策略版本，并触发受影响设备重评。

导入失败处理：

- `strict`：存在任何非法行时禁止提交，作为新系统默认模式；
- `valid_only`：允许只提交预览中的合法行，保留完整失败明细，覆盖老项目“部分成功”能力；
- `replace + valid_only` 属于高风险组合，预览必须展示最终保留和将停用集合，并进行额外二次确认；
- 无论选择哪种模式，实际提交集合都在一个事务内原子生效，不能出现合法行只提交一半；
- 已提交的合法集合可以按批次整体回滚。

名单 `replace` 的作用域严格限定为“操作者运营商 + 当前名单类型”，覆盖黑名单不能改动白名单或吊销名单；规则维度 `replace` 只替换当前草稿、当前规则、当前维度。

批量删除统一实现为 `batch_disable`：

- 只允许停用调用者运营商范围内的有效条目；
- 整个请求在一个数据库事务内执行；
- 部分条目不存在或已停用时，预览阶段给出结果，提交阶段不产生半成功；
- 保留原记录、操作人、原因和批次关联。

页面手工新增同时支持单条和多 SN；多 SN 输入不再由前端拆成多次请求，而是规范化后一次提交服务端批次。状态页“加入黑名单/白名单”也必须调用同一名单 Service，禁止复制一套优先级、权限或重评逻辑。

### 5.2 文件格式

名单官方模板首先支持 CSV，保持与老项目实际操作一致：

```text
Serial Number,List Type,Reason,Valid From,Valid Until
```

约束：

- UTF-8，可兼容 UTF-8 BOM；
- 表头固定并支持本地化展示，服务端按稳定字段标识解析；
- 单文件默认上限 10,000 行、10 MiB，最终阈值进入配置；
- SN 去空格后按现有设备序列号规则校验；
- `List Type` 必须与页面选择一致，防止跨类型误导入；
- 日期统一按系统时区解析并保存为 UTC；
- 重复 SN、无效时间范围、非法类型和超限分别返回稳定错误码。

兼容门槛不是扩展名数量，而是老系统官方导出物能够直接迁移：80 下载的 `importAccessSN.csv`、老规则维度 XLSX 模板和老控制值 CSV 导出必须进入回归样本。若生产留存中确有二进制 `.xls`，增加受限兼容解析或离线转换工具，并在预览中标注转换结果；不能只因为老页面文件选择器声明支持 `.xls` 就引入未经样本验证的解析分支。

老项目按规则维度提供导入、导出，因此核心等价方案沿用维度模板：

- SN：`Serial Number`；
- TAC：`TAC`；
- ECGI：`ECGI`；
- IP：`Start IP`、`End IP`，服务端规范化为现有 CIDR/范围表达；
- GPS：`Longitude Range`、`Latitude Range`，服务端按确认后的坐标模型转换。

维度模板使用标准 XLSX，并支持 append/replace、预览、失败明细和回滚。导入目标必须是策略草稿中的具体规则，不能修改已发布版本。

完整策略包如评审决定建设，使用标准 XLSX 工作簿：

- `Policy`：策略名称、默认动作；
- `Rules`：规则 ID、名称、启停、SN 范围；
- `Conditions`：规则 ID、条件 ID、类型、操作符、期望值、必需标记、TTL；
- GPS 使用经度、纬度、半径和 `allow_missing` 独立列。

项目已依赖 `excelize/v2`，不新增同类库。完整策略导入只创建草稿，必须在现有策略编辑器复核后单独发布。

### 5.2.1 老规则范围的无损兼容

老项目的 IP 是任意起止地址，GPS 是经纬度矩形范围；当前新实现只有单一 CIDR 和中心点半径，两者不能保证无损互转。为确保核心业务等价，需要扩展规则模型：

```text
ConditionOperatorIPRange      = "ip_range"
ConditionOperatorWithinBounds = "within_bounds"
```

对应结构：

```text
IPRange  = { start_ip, end_ip }
GeoBounds = { min_longitude, max_longitude, min_latitude, max_latitude, allow_missing }
```

求值规则：

- IP 转为 128 位可比较地址后判断闭区间，IPv4 和 IPv6 不允许混用；
- GPS 对经纬度分别判断闭区间，边界值算匹配；
- 对跨越 180 度经线的范围必须显式表示或在预览阶段拒绝，不能静默交换端点；
- 新建规则仍可选择 CIDR 或半径围栏；导入老模板时保持原矩形/起止范围语义；
- 导出时按原操作符输出对应模板，不能把矩形导出为近似圆。

### 5.3 数据模型

新增 `device_access_import_batches`：

| 字段 | 含义 |
|---|---|
| `id` | 批次 ID |
| `carrier` | 服务端从操作者上下文取得的运营商 |
| `import_type` | `access_list/rule_dimension/policy_bundle` |
| `entry_type` | 名单类型，可空 |
| `target_policy_version_id/target_rule_id` | 规则维度导入目标，可空 |
| `dimension` | `sn/tac/ecgi/ip/gps`，可空 |
| `mode` | `append/replace` |
| `failure_policy` | `strict/valid_only` |
| `status` | `uploaded/validated/committing/committed/failed/rolled_back` |
| `source_filename` | 原始文件名 |
| `content_sha256` | 文件幂等摘要 |
| `total/valid/invalid/changed_count` | 统计 |
| `snapshot` | 提交前安全快照或版本引用 |
| `created_by/committed_by/rolled_back_by` | 操作人 |
| `created_at/committed_at/rolled_back_at` | 时间 |

新增 `device_access_import_rows`：

| 字段 | 含义 |
|---|---|
| `batch_id/row_number` | 批次和行号 |
| `raw_value` | 最小必要原始值，避免保存无关敏感信息 |
| `normalized_value` | 规范化结果 |
| `validation_status` | `valid/invalid/duplicate/no_change` |
| `error_code/error_message` | 失败明细 |
| `target_id` | 提交后的名单、规则或条件 ID |

现有名单条目增加可选的 `source_batch_id`；批量停用通过批次表保存目标集合。所有表和字段折回 `omcgo/migrations/000001_init_schema.sql`，同步更新基线一致性检查，不新增 `000002+`。

### 5.4 提交与回滚语义

预览阶段只解析、校验和计算影响，不修改生效数据。

提交阶段：

1. 锁定批次并校验状态为 `validated`；
2. 校验文件摘要、运营商范围和操作者权限；
3. `append` 执行幂等 upsert；
4. `replace` 在事务中停用旧集合并插入/启用新集合；
5. 写操作审计；名单提交同时写受影响设备重评 Outbox；
6. 名单提交后异步重评，不能在 HTTP 请求内同步重评所有设备；规则维度提交只更新草稿，必须等策略发布时才触发重评。

回滚阶段：

- 只允许回滚已提交且未被后续同范围批次覆盖的批次；
- 名单恢复提交前有效状态；
- 未发布草稿的规则维度导入直接恢复该草稿提交前快照；已发布策略不允许通过导入批次修改；
- 已发布策略的业务回滚通过创建并发布前一版本内容的新版本完成，不修改历史版本；
- 名单和已发布策略回滚产生审计、事件和设备重评；未发布草稿回滚只产生操作审计。

### 5.5 API

```text
GET  /api/v1/device-access/import-templates/:importType
POST /api/v1/device-access/imports/preview
GET  /api/v1/device-access/imports/:batchID
GET  /api/v1/device-access/imports/:batchID/errors
POST /api/v1/device-access/imports/:batchID/commit
POST /api/v1/device-access/imports/:batchID/rollback
POST /api/v1/device-access/access-list/batch-disable
```

关键约束：

- 请求体中的 `carrier` 一律忽略或拒绝，操作者运营商范围是唯一权威来源；
- preview、commit、rollback 分配不同 RBAC 权限；
- commit/rollback 支持 `Idempotency-Key`；
- 错误明细下载只允许访问本运营商批次；
- 文件解析和 SQL 均有数量上限，SQL 使用 Squirrel + pgx 参数化。

### 5.6 前端

在 `GovernancePanels.tsx` 的名单区域增加：

- 下载模板；
- 批量导入；
- 导入历史；
- 服务端批量停用；
- 预览弹窗：总数、有效、无效、重复、将新增、将停用；
- 覆盖模式二次确认；
- 失败明细下载；
- 已提交批次回滚入口。

在 `PolicyEditorModal` 的 SN、TAC、ECGI、IP、GPS 各维度增加模板下载、导入和导出。导入只能作用于当前草稿和当前规则，预览后展示差异；完整策略包导入如建设，则创建新草稿并展示整体差异。任何导入都禁止自动发布。

状态页面增加：

- SN、规则、策略版本、控制维度、决定结果、原因码和时间区间筛选；
- 当前筛选条件下接受/拒绝/待审核/吊销汇总；
- 从状态行直接加入黑名单或白名单；
- 从规则版本跳转查看该规则接入明细；
- 可审计归档，替代老项目物理删除接入结果；
- 手工刷新和有界轮询，抽屉打开时不得因刷新丢失当前上下文。

状态汇总使用独立的 `GET /device-access/states/summary`，复用状态列表完全相同的过滤器和授权谓词，避免前端按当前页自行计数。接入结果“归档”使用独立归档记录关联决定 ID，不修改原决定、当前接入状态或设备动作；当前生效决定不得从默认视图隐藏。

所有文案进入 i18n；Modal、toast、空态、错误和权限状态必须用真实浏览器验证。

## 6. 设计二：统一设备身份与 Inform 上下文

### 6.1 主键与快照

内部继续使用 `devices.id` 作为稳定主键，不把旧系统复合字段直接变成数据库主键。每次接入观察新增不可变身份快照：

```text
device_id
carrier
serial_number
device_code / small_cell_code
cloud_key
oui
product_class
remote_ip
inform_event
inform_time
request_id
identity_source
identity_status
identity_reason_code
```

身份快照与决定通过 `request_id/decision_id` 关联，历史决定不跟随设备表字段变化而改写。

### 6.2 解析规则

解析顺序固定为：

1. 已登记设备与凭据的强绑定；
2. 运营商适配器的 CloudKey/设备编码规则；
3. OUI + ProductClass 映射；
4. 未知设备候选。

禁止静默使用默认运营商。稳定结果码：

- `carrier_unresolved`；
- `device_code_missing`；
- `cloud_key_missing`；
- `identity_mismatch`；
- `identity_ambiguous`；
- `ownership_unverified`。

CloudKey 和设备编码是否必填由 `internal/core/carrier/` 中的运营商身份策略定义，不能在 `deviceaccess` 中散落运营商分支。缺失字段可以按策略进入拒绝或人工审核，但必须记录原因。

### 6.3 兼容原则

- 对不提供 CloudKey 的新设备型号允许运营商适配器声明“不要求”；
- 对老型号声明必填后，缺失不能被 OUI 推断静默放行；
- SN、设备编码冲突时不得自动合并资产；
- 未知设备审核通过后形成正式设备身份和名单变更，再触发重新评估。

### 6.4 统一触发契约

所有入口只负责提交标准化重评请求，由现有 Reevaluation Coordinator 统一加载最新策略和证据：

| 入口 | `trigger_type` | 作用范围 |
|---|---|---|
| 首次 Inform | `inform_first_seen` | 当前设备 |
| BOOT | `inform_boot` | 当前设备 |
| 断链重连/上线恢复 | `inform_reconnected` | 当前设备 |
| 普通周期 Inform | `inform_periodic` | 仅在证据、身份、策略或人工请求版本变化时形成新决定 |
| 名单提交/回滚/停用 | `access_list_changed` | 受影响 SN |
| 策略发布/回滚 | `policy_published` | 策略 SN 作用域内设备，后台分页展开 |
| 候选审核 | `candidate_reviewed` | 当前候选设备 |
| 身份/凭据变化 | `identity_changed` | 当前设备 |
| 手工重评 | `manual_reevaluation` | 当前设备 |

重评幂等键至少包含运营商、设备、触发事件 ID、策略版本和证据版本；同设备并发请求通过数据库锁或租约串行化。Redis 仅做短时抑制，丢失后不得造成重复设备动作。

### 6.5 例外、优先级和失败策略

特殊平台例外进入运营商接入配置，不在求值器中写产品名分支。配置包含匹配条件、优先级、原因、有效期、操作人和审计信息。黑白名单不是普通规则条件，而是名单直通分支：识别出运营商和 SN 后，命中黑名单直接形成拒绝决定并触发小区去激活，命中白名单直接形成允许决定并允许小区激活，二者都不得继续采集或计算 TAC、ECGI、IP、GPS 模板。为兼容老业务，求值顺序固定为：

```text
业务开关关闭兼容放行
→ 配置化平台例外
→ 解析运营商和 SN 最小身份
→ 吊销/资产退役
→ 黑名单直拒并去激活
→ 白名单直通并允许激活
→ 非名单设备身份与运营商完整校验
→ 按 priority 升序尝试启用规则
→ 运营商默认动作
```

名单直通仍必须产生 `denylist_matched` 或 `allowlist_matched` 决定证据，并分别保存小区去激活或允许激活/恢复的设备动作证据；“不走接入控制”只表示不进入普通模板求值，不能直接跳过而无记录。例外命中也必须产生 `bypass_profile_matched` 决定证据。规则 `priority` 在同一策略版本内唯一；发布前校验，页面支持排序和差异预览。

无规则匹配固定执行 `reject`，不提供无条件 `accept` 或无法闭环的通用人工复核。证据采集失败或系统错误先按有限重试和采集截止时间处理，耗尽后按 `failure_mode=fail_closed` 进入拒绝隔离；不得永久停留在 collecting 而没有终态和告警。未知设备的人工处理仅通过独立候选审核流程完成。

### 6.6 IP 与 GPS 证据口径

- 接入 IP 默认取 ACS 实际 TCP 对端地址；如经过可信代理，仅在代理白名单内解析标准转发头，并同时保存原始对端和解析地址；
- NAT 地址变化必须形成新证据版本并触发重评，不能覆盖历史决定证据；
- GPS 统一为 WGS84 十进制度，保存设备原值、归一化值和来源；
- 无效坐标、零值、越界、单位错误、坐标系未知、人工覆盖分别产生原因码；
- 人工位置覆盖是否可用于接入决定由运营商策略显式声明，不能自动沿用电子围栏的展示位置。

## 7. 设计三：设备动作可靠性

### 7.1 状态机

扩展动作状态：

```text
pending_dispatch
dispatching
verifying
retry_wait
succeeded
failed
dead
cancelled
```

保留现有 `failed` 以兼容 API 和历史数据：它表示不参加自动调度、但仍可人工重试的失败动作；新动作遇到可重试错误进入 `retry_wait`，耗尽次数进入 `dead`。部署时不自动重放历史 `failed` 动作，避免部署后突然操作设备。

### 7.2 动作尝试

新增 `device_access_action_attempts`：

- `action_id/attempt_no`；
- `phase`：baseline GPV、SPV、readback GPV；
- `device_task_id/command_key`；
- `status`：queued/sent/succeeded/failed/timeout/cancelled；
- `fault_code/failure_code/error_message`；
- `request_summary/response_summary`；
- `started_at/completed_at`。

主动作增加：

- `next_attempt_at`；
- `max_attempts`；
- `last_failure_code`；
- `dead_at`；
- `manual_repair_required`。

SOAP 原文仍使用现有 TR069 跟踪能力保存，不在接入动作表复制完整敏感报文；动作尝试只保存可审计摘要和跟踪关联 ID。

### 7.3 重试策略

- 系统异常、CWMP 超时、临时离线：可重试；
- SPV Fault 是否重试按 FaultCode 分类；
- GPV 回读不一致：允许有限重试，禁止直接标记成功；
- 身份变化、策略变化、业务开关关闭：取消旧动作并记录原因；
- 默认最多 3 次，建议退避 30 秒、2 分钟、10 分钟；
- 设备离线进入 `retry_wait`，设备下一次 Inform 后提前唤醒，不把离线等待当作成功；
- 达到上限进入 `dead`，发布动作失败事件并允许人工修复。

重试前必须重新读取当前接入状态、策略版本和业务开关；已经不再需要隔离/恢复时取消动作，不能执行过期安全命令。

### 7.4 恢复器

新增动作 Reconciler：

- 周期扫描到期的 `pending_dispatch/retry_wait`；
- 扫描超时的 `dispatching/verifying` 并查询关联任务终态；
- 使用数据库租约或 `FOR UPDATE SKIP LOCKED` 保证多 Worker 不重复处理；
- Worker 重启后从 PostgreSQL 恢复；
- Redis/NATS 只负责加速和通知，不是恢复依据。

### 7.5 人工修复

人工重试接口保留，并增加必填修复原因。死信动作可以：

- 重试原动作；
- 标记已由现场处理并执行 GPV 验证；
- 取消已不适用动作。

任何人工操作都要校验运营商、设备组和动作管理权限，记录操作审计。

### 7.6 下游位置与围栏门禁

电子围栏和位置移动模块在规划任何 RF 开启动作前，必须通过统一 Admission Reader 读取数据库中的当前接入状态：

- 只有 `accepted` 可以继续评估 RF 开启；
- `review_required/collecting/revalidating/rejected/revoked` 一律禁止激活；
- 接入控制业务开关关闭时按现有兼容行为放行，但记录门禁旁路原因；
- 围栏可以对已接受设备执行自身的隔离策略，但不能恢复由接入控制拥有的 RF 隔离；
- 接入状态变化事件触发下游重新评估，不能由下游直接改写接入决定。

## 8. 设计四：事件、通知与审计

### 8.1 事件

在事件主题中补充：

```text
device.access.action_failed
device.access.action_recovered
```

稳定业务事件名称：

```text
DEVICE_ACCESS_ACCEPTED
DEVICE_ACCESS_REJECTED
DEVICE_ACCESS_REVIEW_REQUIRED
DEVICE_ACCESS_REVOKED
DEVICE_ACCESS_ACTION_FAILED
DEVICE_ACCESS_ACTION_RECOVERED
```

事件最小载荷：

```json
{
  "event_id": "uuid",
  "request_id": "uuid",
  "decision_id": "uuid",
  "action_id": "uuid",
  "device_id": "uuid",
  "carrier": "CMCC",
  "serial_number": "...",
  "policy_version_id": "uuid",
  "reason_code": "gpv_readback_mismatch",
  "occurred_at": "UTC timestamp"
}
```

决定、动作终态和对应 Outbox 必须在同一数据库事务提交。通知失败只能更新通知状态，不能修改接入决定或动作状态。

### 8.2 通知关联

通用通知历史增加来源关联：

- `source_type`：alarm/device_access_decision/device_access_action；
- `source_id`；
- `event_id`；
- `correlation_id`。

尚未配置邮件短信规则时，页面显示 `not_configured`，不能误显示为发送成功；配置后展示 pending/sent/failed/dead_letter。

### 8.3 审计查询

新增 `device_access_decision_archives` 保存决定的展示归档信息：`decision_id`、`archived_by`、`reason`、`archived_at`、`restored_by`、`restored_at`。该表只影响历史列表可见性，不修改决定表、当前状态、任务门禁或设备动作。

扩展状态和动作查询：

- `started_at/ended_at`；
- `decision`；
- `reason_code`；
- `action_status`；
- `notification_status`；
- `policy_version_id`；
- `import_batch_id`。

详情页按时间线聚合：Inform 身份快照、证据采集、决定、重评、设备动作尝试、通知和人工修复。默认分页加载，不再固定只展示最近 50 条且不可继续查询。

## 9. 权限设计

新增或细化权限：

- 接入名单查询；
- 单条名单维护；
- 导入预览；
- 导入提交；
- 批次回滚；
- 批量停用；
- 策略草稿导入；
- 策略发布；
- 动作人工修复；
- 审计和失败明细下载。

所有服务端查询和修改都从认证上下文取得运营商与设备组范围。前端权限控制只用于交互提示，不能作为安全边界。

新增权限必须折回基线 seed 文件，并补 viewer/operator/admin 的 Casbin 测试和真实受限角色 403 验收。

## 10. 错误码与可观测性

### 10.1 稳定错误码

批量治理至少包含：

- `import_file_too_large`；
- `import_template_invalid`；
- `import_row_invalid`；
- `import_duplicate_identity`；
- `import_batch_state_conflict`；
- `import_rollback_conflict`；
- `import_carrier_scope_forbidden`。

设备动作至少包含：

- `cwmp_timeout`；
- `cwmp_spv_fault`；
- `gpv_readback_missing`；
- `gpv_readback_mismatch`；
- `device_offline`；
- `action_state_stale`；
- `action_retry_exhausted`。

错误信息在后端包装上下文，用户可见文案通过前端 i18n 映射，日志不记录完整凭据和不必要的敏感参数。

### 10.2 指标

建议增加：

- 导入预览、提交、失败、回滚批次数；
- 各失败码导入行数；
- 接入决定按运营商、结果和原因码计数；
- 动作 pending/retry_wait/dead 数；
- 动作执行时延和尝试次数；
- 动作失败事件与通知失败数；
- Reconciler 扫描、恢复和租约冲突数。

## 11. 测试方案

### 11.1 单元测试

- CSV/XLSX 模板解析、规范化、重复、非法值、大小和行数限制；
- append/replace/rollback 差异计算；
- 身份解析优先级、字段缺失、冲突、共享 OUI；
- 动作错误分类、退避、上限、状态变化取消；
- 事件载荷、Outbox 幂等和通知隔离；
- 所有列表和排序字段参数化。

### 11.2 PostgreSQL 集成测试

- 预览不修改线上数据；
- replace 原子提交，任一步失败整批回滚；
- 两个并发 commit 只有一个成功；
- 回滚与后续批次冲突检查；
- 决定/动作/Outbox 同事务；
- 多 Worker 使用 `SKIP LOCKED` 不重复执行；
- 进程重启后未完成动作可恢复；
- 运营商和设备组越权均为 403。

### 11.3 前端测试

- 导入模式和预览统计；
- 覆盖二次确认；
- 失败明细和回滚状态；
- 批量停用只提交一次服务端请求；
- 筛选条件和分页；
- 动作时间线、重试、死信和通知状态；
- 无权限、错误、空态和加载态。

### 11.4 真实浏览器验收

- 官方模板下载、追加、覆盖、重复/非法/部分错误、批次回滚；
- TAC、ECGI、IP、GPS 分维度匹配、不匹配、缺失、读取失败和多项失败；
- 无邻区允许和无 GPS 允许；
- 多规则 OR 和规则内 AND；
- 未知设备候选放行与拒绝；
- 总开关关闭后的真实 Inform；
- CWMP 超时、SPV Fault、GPV 不一致、人工修复；
- 设备离线重连、重复/并发 Inform、Worker 重启；
- 首次 Inform、BOOT、周期 Inform、断链重连、名单/策略变更、候选审核的触发矩阵；
- 特殊平台例外、规则显式优先级、固定默认拒绝和 fail_closed；
- 已拒绝设备命中围栏恢复条件时仍不得 RF 开启；
- 受限角色服务端 403；
- 通知失败时决定和 RF 安全动作不受影响；
- 原始 Inform、GPV、SPV 和回读 SOAP 留档。

## 12. 发布与兼容策略

### 12.1 部署顺序

1. 合入基线 Schema/seed、后端模型和只读兼容代码；
2. 合入批量治理、动作 Reconciler 和事件能力；
3. 合入前端入口和审计查询；
4. 在测试环境执行数据结构与权限验证；
5. 先对单运营商、授权测试设备灰度；
6. 104 完成全量标准验收后再决定正式放行。

### 12.2 安全开关

建议新增独立运行时开关：

- `device_access_batch_governance_enabled`；
- `device_access_action_auto_retry_enabled`；
- `device_access_notification_events_enabled`。

功能发布初期可以关闭自动重试，只观察旧失败动作和新动作分类；确认不会误操作设备后再对单运营商开启。业务总开关关闭时仍不得产生新的探测、接入决定或 RF 动作。

### 12.3 历史数据

- 不批量重放历史失败动作；
- 不物理删除旧名单和决定；
- 旧 `failed` 动作保留原状态，仅新动作进入完整自动重试状态机；
- 通知关联字段为空时页面显示“历史未关联”；
- 历史身份缺少 CloudKey/设备编码时明确显示“未采集”，不反推伪造证据。

### 12.4 老项目数据迁移

生产替换前使用批量治理接口完成一次受控迁移，不让新服务运行时直接依赖老数据库：

1. 从老系统按运营商导出有效黑名单、白名单、规则、SN、TAC、ECGI、IP、GPS 和启停状态；
2. 生成文件 SHA-256、条数统计和业务范围清单；
3. 在新系统执行 preview，只产生差异报告；
4. 处理非法、重复、跨运营商和身份无法映射的数据；
5. 先导入名单，再导入规则草稿；
6. 人工复核 IP 起止范围、GPS 矩形范围和“无邻区/无 GPS”标记；
7. 发布策略前进行新旧双算比较，不下发设备动作；
8. 差异清零或完成书面裁决后，再对单运营商授权设备灰度；
9. 保存迁移批次、差异报告和回滚点。

迁移验收必须比较运营商维度的总数、启用数、内容摘要、规则条件数和抽样判定结果，不能只看导入接口返回成功。

## 13. 实施工作包与建议提交顺序

### WP0：业务契约冻结

- 确认名单 CSV 字段、replace 语义和回滚冲突规则；
- 确认各运营商设备编码/CloudKey 必填策略；
- 确认无邻区、无 GPS 和读取失败的决定语义；
- 确认动作重试 FaultCode 分类和退避参数。

交付：评审结论和接口契约，不改业务行为。

### WP1：批量名单治理

- Schema、Repository、Service、Handler；
- 模板下载、预览、提交、失败明细、回滚；
- 服务端批量停用；
- 多 SN 手工新增、名单 SN/设备名称查询、状态行快捷加入黑白名单；
- 权限、审计、Outbox 和前端页面。

### WP2：规则维度导入和身份闭环

- SN/TAC/ECGI/IP/GPS 维度 XLSX 模板、导入/导出并更新草稿；
- 可选的完整策略包导入/导出；
- 身份快照和运营商身份策略；
- 全部接入触发入口收敛、配置化平台例外、规则 priority 和默认/失败策略；
- 候选审核后的身份落库和重评证据。

### WP3：动作可靠性

- 动作尝试表、状态机和错误分类；
- 自动退避、离线唤醒、重启 Reconciler；
- 死信和人工修复；
- SOAP 跟踪关联。
- 位置/电子围栏 RF 开启动作的接入状态门禁与所有权测试。

### WP4：事件、通知与审计

- action failed/recovered 事件；
- 通知历史来源关联；
- 时间、结果、原因和通知筛选；
- 规则维度详情、筛选汇总和接入结果归档；
- 统一详情时间线。

### WP5：标准验收

- 自动测试和真实浏览器全矩阵；
- 104 受控设备验收；
- 原始 SOAP、API、数据库、页面和设备状态证据；
- 清理测试数据并形成正式验收报告。
- 执行老数据迁移预演和新旧判定双算对账。

每个 WP 独立提交，提交信息使用 Conventional Commits 中文描述；不把无关工作树改动纳入提交。

### 13.1 预计代码改动面

后端接入控制模块：

- `omcgo/internal/deviceaccess/model.go`：导入、身份、动作状态和筛选模型；
- `evaluator.go`、`policy_service.go`：`ip_range/within_bounds` 无损求值与校验；
- 新增 `import_service.go`、`import_repository.go`、`import_parser.go`；
- `http_handler.go`：导入、回滚、批量停用和审计筛选 API；
- `list_service.go`、`policy_service.go`：批次提交和异步重评；
- `action.go`、`action_repository.go`：动作状态机、尝试记录和事务终态；
- 新增 `action_reconciler.go`：自动重试与重启恢复；
- `management.go`：时间、结果、通知和批次查询；
- `actor_resolver.go`、`access_gate.go`：统一身份上下文；
- `omcgo/internal/core/carrier/`：运营商身份必填策略；
- `omcgo/internal/core/event/subjects.go`：动作失败/恢复主题；
- Worker provider：注册 Reconciler 和事件消费者。

数据和权限：

- `omcgo/migrations/000001_init_schema.sql`：批次、行明细、动作尝试和关联字段；
- `omcgo/migrations/seed/000001_init_seed.sql`：新 API 权限；
- 对应基线 Schema、seed 和 Casbin 测试。

前端：

- `omcmb/frontend-core/src/services/api/deviceAccessApi.ts`；
- `omcmb/frontend-core/src/hooks/api/useDeviceAccess.ts`；
- `omcmb/webcode/src/pages/device/AccessControl/GovernancePanels.tsx`；
- `PolicyEditorModal.tsx`、`AccessStatesPanel.tsx`、`AccessActionsPanel.tsx`；
- V1 语言资源和对应组件/API 测试。

### 13.2 开发分支与工作树策略

当前本地 `main` 比 `origin/main` 落后，且工作树存在与本功能无关的用户改动。评审通过后应从包含现有接入控制实现的 `origin/main` 创建 `codex/device-access-full-alignment` 开发分支，并隔离现有脏文件；不得把离线地图、部署文档和本地配置等无关改动带入本功能提交。开始开发前再次确认远端基线是否有更新，未经用户授权不主动 pull/push。

## 14. 完成定义

只有同时满足以下条件，才能宣布“完整覆盖老项目接入控制核心能力”：

- 名单模板下载、追加、覆盖、失败明细、批量停用和整批回滚通过；
- 名单多 SN 手工新增、SN/设备名称查询、状态行快捷加入黑白名单通过；
- 规则内 SN/TAC/ECGI/IP/GPS 模板下载、追加、覆盖、导出和清空通过；
- 老 IP 起止范围和 GPS 经纬度矩形范围无损求值通过；
- 可选的完整策略包导入如建设，必须生成草稿且发布生命周期通过；
- 身份快照、身份冲突和运营商解析结果可审计；
- 全部触发入口、特殊平台例外、规则 priority、默认动作和系统异常失败策略通过；
- TAC、ECGI、IP、GPS 完整正反例和特殊分支通过；
- 候选放行/拒绝真实闭环通过；
- 动作超时、Fault、回读不一致、自动重试、死信、人工修复和离线重连通过；
- 并发 Inform、重复事件和 Worker 重启不产生重复动作；
- 拒绝/吊销设备不能被位置或电子围栏流程重新激活；
- viewer/operator/admin 服务端权限矩阵通过；
- 动作失败事件产生，通知失败不影响接入决定和安全动作；
- 关键场景保存原始 CWMP SOAP 及数据库、API、页面、设备状态证据。
- 老名单和规则迁移预演完成，新旧数量、内容摘要、启停状态和抽样判定结果一致。

## 15. 评审决策点

评审需要明确以下四项，确认后才能进入开发：

1. **导入格式**：推荐名单用 CSV，规则内 SN/TAC/ECGI/IP/GPS 用与老项目对应的标准 XLSX；完整策略包导入作为可选增强，是否接受该边界。
2. **身份必填规则**：各运营商/设备型号哪些必须提供设备编码和 CloudKey，缺失时选择拒绝还是人工审核。
3. **动作自动重试策略**：推荐新动作最多 3 次，30 秒/2 分钟/10 分钟退避，历史失败动作不自动重放。
4. **回滚边界**：推荐后续同范围批次已经提交后，旧批次禁止一键回滚，只允许基于当前状态生成新的反向变更批次。

评审通过后，开发从 WP1 开始；WP0 未确认的内容不得由开发人员自行猜测。
