# Issue #227 告警实例跨通道关联与同步收敛设计

## 背景

设备 `120200055922C8B0068` 在 2026-07-30 出现 LMT 当前告警 2 条、OMC 当前告警 6 条的偏差。现场数据和 OMC 数据库记录表明：

- 15:55 的 `11109`、`11112` 由 `CurrentAlarm.3/.4` 写入。
- 15:59 的 `11189`、`11190` 由 `ExpeditedEvent` 写入。
- 16:24 基站产生 `11189`、`11190` 清除事件。
- 16:25 的 `11109`、`11112` 又由 `ExpeditedEvent` 写入。
- OMC 将同一业务告警在 `CurrentAlarm` 与 `ExpeditedEvent` 中的上报视为不同实例，形成重复活动告警。
- 15:55 后没有新的告警全量同步任务；后续 ExpeditedEvent 处理链未发布 `alarm.sync.requested`，旧告警无法依靠 CurrentAlarm 对账收敛。

2026-05-28 的提交 `9160cf73d` 将活动告警匹配键从 `AlarmIdentifier` 改为 `AlarmIdentifier + ManagedObjectInstance/AdditionalInformation`，用于支持同一设备上相同告警 ID 的并行实例。实际设备把 `CurrentAlarm.N`、`ExpeditedEvent`、`HistoryEvent.N` 作为上报容器路径，而非稳定业务对象路径，因此该匹配规则不能跨通道关联同一告警。

## 原流程与设计原则

原流程按 `DeviceSN + AlarmIdentifier` 处理 New、Changed、Cleared 和 CurrentAlarm 全量对账：

1. NewAlarm 查到活动告警则更新，否则新增。
2. ChangedAlarm 更新已有告警，查不到时按 NewAlarm 处理。
3. ClearedAlarm 查找活动告警，归档到历史表并从活动表删除。
4. CurrentAlarm 全量同步以设备当前列表为权威，执行新增、更新和清除。

本次保留上述流程和接口，不引入新的告警状态机，不改变活动表、历史表或 TR-069 协议模型。只修正实例关联规则、ExpeditedEvent 同步触发和既有重复数据的收敛方式。

业务上同一设备、同一 `AlarmIdentifier` 可能存在多个并行实例，因此不能完全退回 `DeviceSN + AlarmIdentifier`。实例键必须能区分真实业务对象，同时忽略 CurrentAlarm、ExpeditedEvent、HistoryEvent 的通道容器差异。

## 目标

- 同一业务告警从 CurrentAlarm 和 ExpeditedEvent 上报时只保留一个活动实例。
- 同一设备、同一告警 ID 在不同小区、射频或其他真实对象上可并行存在。
- ClearedAlarm 信息不足时不误清多个候选；能够安全清除唯一候选。
- 每批 ExpeditedEvent 处理后触发一次 CurrentAlarm 全量同步，保证最终一致性。
- 部署后一次全量同步可以把现场 6 条活动告警收敛到 LMT 当前 2 条。
- 保留原始 `managed_object_instance`、`additional_text` 和 `additional_information`，便于展示和排障。

## 非目标

- 不修改前端告警列表和设备列表计数逻辑。
- 不新增数据库表、字段或 `000002+` 迁移。
- 不改变告警确认、已读、人工清除、过滤规则或北向通知业务。
- 不按某几个告警 ID 写死 `11109`、`11112`、`11189`、`11190` 特例。
- 不在通用业务逻辑中散落运营商判断。

## 选定方案

### 1. 统一业务实例键

活动告警匹配键保持现有形式：

```text
AlarmIdentifier | StableObjectScope
```

但 `StableObjectScope` 不再直接使用原始 MOI，而由统一的身份解析器按以下优先级生成：

1. **真实管理对象 MOI**
   - 非 `Device.FaultMgmt.CurrentAlarm.*`
   - 非 `Device.FaultMgmt.ExpeditedEvent.*`
   - 非 `Device.FaultMgmt.HistoryEvent.*`
   - 例如 `Device.Radio.1`、`Device.Cell.1` 可直接使用。
2. **AdditionalInformation 中的稳定对象前缀**
   - 当字符串以 `AdditionalText` 对应的对象名开头，并带括号标识时，使用首个分号前的前缀。
   - 现场示例：`LTE0(73828545);S1setup fail...` 解析为 `LTE0(73828545)`。
3. **AdditionalText**
   - 例如 `LTE0`、`LTE1`。
4. **AdditionalInformation**
   - 作为兼容现有多实例行为的最后限定信息，进行首尾空白和连续空白归一化。
5. **空限定符**
   - 无任何对象信息时退化为原流程的 `AlarmIdentifier`。

身份解析器只负责生成关联键，不修改或删除原始 AdditionalInfo。

现场数据将得到：

```text
CurrentAlarm:   11109 | LTE0(73828545)
ExpeditedEvent: 11109 | LTE0(73828545)
```

两种通道因此关联为同一实例。若 LTE0、LTE1 或括号内对象标识不同，仍保留为不同活动实例。

### 2. NewAlarm 与 ChangedAlarm

- `Process`、Redis 去重和数据库回查统一使用业务实例键。
- CurrentAlarm 与 ExpeditedEvent 命中同一键时更新已有记录，不新增第二条。
- 更新时继续保留告警 ID、确认状态和原有 ID；AdditionalInfo 按现有方式合并，允许保存最近一次上报的原始 MOI。
- ChangedAlarm 查不到精确实例时，沿用现有逻辑转入 NewAlarm 处理。

### 3. ClearedAlarm 安全回退

ClearedAlarm 按三层规则处理：

1. 能生成业务实例键且精确命中时，只清除该实例。
2. 精确匹配失败后，查询同设备、同 `AlarmIdentifier` 的全部活动候选：
   - 只有一个候选：沿用原流程，清除该唯一候选。
   - 没有候选：保持幂等，不报业务错误，但记录 debug 结果。
   - 多个候选：不猜测、不清除；记录结构化 warning 和指标，并请求一次 CurrentAlarm 全量同步。
3. AutoClear 不再把“多候选无法判定”记录成普通成功。

该规则可处理 GPS/时钟类告警的清除文本与产生文本不同，同时避免误清同一告警 ID 的其他小区实例。

### 4. ExpeditedEvent 后触发全量同步

`ExpeditedEventReceiver` 对一批参数完成解析和处理后，只发布一次 `alarm.sync.requested`：

- 至少有一个合法 New、Changed 或 Cleared 事件被处理时发布。
- 无有效事件、设备不存在或解析失败时不发布。
- 单条事件处理失败不阻止其他事件；批次结束后仍发布一次同步请求，使 CurrentAlarm 对账承担兜底。
- 复用现有 Redis 同步锁、开放任务检查和 GPV 任务机制，不为每条告警创建任务。

该行为与普通 `AlarmReceiver` 处理完成后触发同步的原流程保持一致。

### 5. 全量同步与重复数据收敛

CurrentAlarm 仍是设备当前告警的最终权威。同步索引需要从 `map[key]*Alarm` 调整为能够识别同一业务键下多个本地活动记录：

- 远端不存在该键：清除该键下全部本地记录。
- 远端存在且本地只有一条：按原流程更新。
- 远端存在且本地有多条：
  1. 优先保留 `RaisedAt` 与远端最接近的记录。
  2. 无可比较时间时保留最新 `RaisedAt`，再以 `CreatedAt` 和 ID 稳定决胜。
  3. 更新保留记录。
  4. 其余记录通过正常归档路径清除，`cleared_by` 记为 `system:alarm_sync_duplicate`，`clear_note` 说明由 CurrentAlarm 对账清理重复实例。

不能直接覆盖 map 中的重复项，否则被覆盖的活动记录将永远无法进入 ToClear。

现场执行同步后：

- 当前 `11109/11112` 对应 16:25 的最新实例保留。
- 15:55 的旧 `11109/11112` 归档。
- CurrentAlarm 中不存在的 `11189/11190` 归档。
- OMC 活动告警收敛为 LMT 当前两条。

### 6. Redis 一致性

Redis 写入、查询和删除必须全部使用统一业务实例键：

- New/Update 使用业务实例键。
- 人工清除、AutoClear、ClearBySync 和重复清理均删除业务实例键。
- 现有仅按 `AlarmIdentifier` 删除的路径一并修正。
- PostgreSQL 仍是活动告警事实源；Redis 删除失败记录 warning，由现有 reconciler 最终修复，不回滚数据库清除。

## 组件边界

### AlarmIdentityResolver

放在告警域内，职责仅包括：

- 识别并忽略 FaultMgmt 临时容器 MOI。
- 归一化对象范围。
- 生成业务实例键。
- 提供同 ID 候选匹配所需的窄接口。

若后续出现运营商特有解析规则，通过 `internal/core/carrier/` 的可选适配能力注入，通用解析器不出现 `if carrier == ...`。

### AlarmEngine

保持 New、Changed、Clear 的原有职责，只改为消费统一身份解析结果，并实现 ClearedAlarm 唯一候选回退。

### AlarmSyncProcessor

继续执行 parse → diff → apply，只扩展 diff 结果以显式携带重复清理项，避免索引覆盖。

### ExpeditedEventReceiver

保持事件解析和分派职责，新增批次级同步请求发布，不直接执行 GPV。

## 错误处理与可观测性

增加以下结构化日志：

- `alarm clear exact match missed`
- `alarm clear resolved by unique identifier fallback`
- `alarm clear ambiguous; sync requested`
- `alarm duplicate reconciled by current alarm sync`
- `expedited alarm sync request published`

日志至少包含 `device_sn`、`alarm_identifier`、业务实例键、候选数量和涉及的活动告警 ID。

增加或复用指标统计：

- 精确清除未命中次数。
- 唯一候选回退清除次数。
- 多候选清除歧义次数。
- 全量同步清理重复活动告警数量。

同步请求发布失败记录 warning，但不回滚已处理的 ExpeditedEvent；后续 periodic Inform 和人工 Alarm Sync 仍可恢复一致性。

## 测试设计

### 身份解析单元测试

- `CurrentAlarm.3` 与 `ExpeditedEvent` 使用相同 `LTE0(73828545)` 时得到相同键。
- `CurrentAlarm.3` 与 `CurrentAlarm.4` 不能仅因表项序号不同而成为不同实例。
- `LTE0(73828545)` 与 `LTE1(73828546)` 得到不同键。
- `Device.Radio.1` 与 `Device.Radio.2` 得到不同键。
- AdditionalInformation 的首尾和连续空白不影响键。

### Engine 单元测试

- CurrentAlarm 后收到等价 Expedited NewAlarm 只更新、不重复。
- ClearedAlarm 精确命中时只清除对应实例。
- 清除信息缺失且同 ID 只有一个候选时成功清除。
- 清除信息缺失且同 ID 有多个候选时不误清，并请求同步。
- Redis 写入和各清除路径使用相同业务实例键。

### Receiver 单元测试

- 一批多个 ExpeditedEvent 只发布一次同步请求。
- 部分事件失败时仍处理其余事件并发布一次同步请求。
- 无有效事件时不发布。

### Sync 单元与集成测试

- 远端一条、本地同业务键两条时保留正确记录并归档重复项。
- 远端无该键时清除该键的全部本地记录。
- 同 ID 不同小区实例不会相互合并。
- 使用本次现场六条数据库记录和两条 CurrentAlarm 构造回归夹具，结果必须为活动两条、归档四条。

### 验证命令

- 告警包相关单元测试。
- `go test ./internal/alarm/...`
- 受影响的 carrier 适配测试。
- `go test ./internal/core/carrier/...`
- `go build ./...`
- 在测试环境触发一次 ExpeditedEvent 和一次 Alarm Sync，浏览器验证活动告警数量与 LMT 一致。

## 部署与现场处理

1. 部署修复版本，不在部署脚本中直接修改告警数据。
2. 对目标设备触发一次 Alarm Sync。
3. 验证 OMC 活动告警从 6 条收敛为 LMT 当前数量。
4. 检查被清理的四条记录已进入历史告警，且 `cleared_by/clear_note` 能区分自动清除和重复收敛。
5. 观察至少一个 ExpeditedEvent 周期，确认其后出现新的告警同步任务。
6. 若同步失败，保留活动记录并按现有任务失败机制排查，不执行批量直接删除。

## 验收标准

- 同一业务告警跨 CurrentAlarm 与 ExpeditedEvent 不重复。
- 同一告警 ID 的不同业务对象实例可并行存在。
- ClearedAlarm 不因上报通道 MOI 不同而残留唯一活动告警。
- 多候选清除不误伤，最终由 CurrentAlarm 同步收敛。
- ExpeditedEvent 后可观察到且每批最多一个告警同步任务。
- 本次现场回归数据从 6 条收敛为 2 条，历史中保留四条清理记录。
- 无数据库结构变化，无前端改动，无告警 ID 特例。
