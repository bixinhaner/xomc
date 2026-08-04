# ACS 会话指标语义修正设计

## 背景与问题

当前 `acs_active_sessions` 由单个 ACS 进程维护的本地会话 ID 集合驱动。集合中的记录最长
保留 5 分钟，并由定时回收器清理，因此在 Inform 流量较高时，该指标更接近“最近 5 分钟
内由本进程跟踪过的会话数”，不是实时并发会话数。现网曾出现该指标约 2.5 万，而共享准入
控制器统计的 `acs_global_active_sessions` 只有几十的情况，容易被误判为 ACS 接近 30,000
会话上限。

真实的全局实时并发来源已经存在：共享准入控制器的 `Current` 结果每 5 秒写入
`acs_global_active_sessions`。本次修正不改变准入、会话生命周期和 5 分钟本地记录回收行为，
只纠正可观测性契约。

## 目标

- 明确区分全局实时并发与进程本地短期跟踪记录。
- 保留 `acs_active_sessions`，避免已有外部查询立即失效，但使其不再表达错误语义。
- 让 Dashboard、告警、巡检和压测脚本统一使用权威的全局指标。
- 通过测试锁定指标值、帮助文本、注册关系和本地回收语义。

## 非目标

- 不修改 ACS 30,000 全局会话上限、限流参数或会话 TTL。
- 不修改准入控制器、Redis 数据结构、Session 存储和业务请求流程。
- 不调整本地会话跟踪集合的 5 分钟回收周期。
- 不修改历史归档、旧设计记录中的指标名称；它们继续作为当时实现的历史材料。

## 指标契约

| 指标 | 修正后的语义 | 数据来源 | 兼容策略 |
| --- | --- | --- | --- |
| `acs_global_active_sessions` | 当前所有 ACS 实例共享的实时已准入会话数 | 共享准入控制器 `Current` | 权威指标，Dashboard、告警和运维脚本必须使用 |
| `acs_active_sessions` | `acs_global_active_sessions` 的兼容别名 | 与全局指标相同的单次 `Current` 采样 | 保留名称；HELP 标明 deprecated，并引导使用全局指标 |
| `acs_local_tracked_sessions` | 当前进程本地保留的会话 ID 数，包含最长 5 分钟的待回收记录 | `localActiveSessions` | 新增诊断指标；不得用于实时并发、容量或准入判断 |

同一次刷新必须先读取一次共享准入值，再把相同值写入
`acs_global_active_sessions` 和 `acs_active_sessions`，避免两次读取之间产生瞬时差异。

## 数据流与实现边界

1. `refreshGlobalActiveSessions` 每 5 秒读取一次共享准入控制器。
2. 读取结果同时写入权威全局指标和兼容别名，二者始终相等。
3. `trackActiveSessionAt`、`untrackActiveSession` 和 `reapLocalActiveSessions` 只维护
   `acs_local_tracked_sessions`，不再直接修改 `acs_active_sessions`。
4. 本地集合仍用 `LoadOrStore`、`LoadAndDelete` 保证同一 ID 只增减一次，防止重复完成或跨实例
   遗留会话把指标减成负数。
5. Prometheus 注册器同时注册三个指标；不引入数据库、配置或外部 API 变化。

## 运维查询与文档迁移

- 现有 Grafana Dashboard 和运行时告警已经查询 `acs_global_active_sessions`，保持不变并用测试或
  静态检查锁定。
- 当前使用的健康检查、压测基准脚本改查 `acs_global_active_sessions`。
- 当前可观测性、部署和 ACS 压测文档使用全局指标描述实时并发；需要解释本地记录时，只使用
  `acs_local_tracked_sessions`。
- 历史设计、阶段报告和归档文档不批量重写，避免篡改历史上下文。

## 兼容性与发布风险

- 继续暴露 `acs_active_sessions`，已有 PromQL、采集规则和第三方集成不会因时间序列消失而报错。
- 旧查询看到的数值会从“本地 5 分钟跟踪量”切换为“全局实时并发量”。这是有意的语义纠正，
  HELP 中必须明确弃用和替代指标。
- 新增一个无标签 Gauge，时间序列成本可忽略，不增加高基数风险。
- 部署后首次 5 秒刷新前两个全局指标可能为初始值 0；这与当前
  `acs_global_active_sessions` 的启动行为一致。

## 测试策略

按 TDD 顺序先补失败测试，再做最小实现：

- 指标构造测试验证三个 Gauge 均注册、名称和 HELP 契约正确。
- 全局刷新测试验证一次刷新后 `acs_active_sessions == acs_global_active_sessions`，且值来自共享
  准入控制器。
- 本地跟踪测试验证新增指标的增、减、重复完成和跨实例遗留保护。
- 本地回收测试验证超过 5 分钟的记录只影响本地诊断指标，不改变全局指标及兼容别名。
- 脚本契约测试或静态检查验证当前健康检查与压测脚本不再读取旧指标。
- 运行 ACS 定向测试、相关脚本测试和完整 Go 测试；发布验证阶段检查实际 `/metrics` 输出。

## 验收标准

- 实际 `/metrics` 中 `acs_active_sessions` 与 `acs_global_active_sessions` 数值相等。
- `acs_local_tracked_sessions` 独立反映本地集合规模，并可高于实时全局并发而不触发容量误判。
- `acs_active_sessions` HELP 明确标记 deprecated，`acs_local_tracked_sessions` HELP 明确说明它不是
  实时并发。
- 当前 Dashboard、告警、健康检查、压测脚本和运维文档均以
  `acs_global_active_sessions` 作为并发及容量判断依据。
- 会话准入、完成、跨实例回收、30,000 上限和既有业务行为没有变化。
- 定向测试与完整测试通过，部署后无重复指标注册、负值、采集失败或新增业务告警。
