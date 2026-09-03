# 严重告警解释

解释当前严重告警的对象、时间、影响、相关状态和可能原因。xOMC 已负责告警事件选择或聚合；不要在 Agent Studio 中重新构造任意时间窗口。

优先查询告警详情、设备状态、近期相关告警和必要的性能摘要。仅将工具结果支持的内容列为 facts；推断列入 hypotheses。最终仅输出符合通用 Finding Schema 与 `severe-alarm-details-v1` 的 JSON。
