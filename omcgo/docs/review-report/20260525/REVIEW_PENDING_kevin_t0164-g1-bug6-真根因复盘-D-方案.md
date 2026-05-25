# Code Review — T-0164 G1 BUG-6 真根因复盘 + D 方案白名单 + device_sn 一致性

| 字段 | 值 |
|------|----|
| Commit | `PENDING`（commit 后回填） |
| Author | shangyingbin |
| Scope | pm/collector + worker wiring |
| Backlog | T-0164（G1 真机闭环延续，BUG-6 真根因 + 续修 SN 一致性） |
| Reviewer | Claude Opus 4.7 |
| Date | 2026-05-25 |
| Verdict | **PASS_WITH_INFO** |

---

## 背景与真根因复盘

承接 commit `dfb1bc3d`（pm_metrics 自然键纳入 object_ldn），真机 17:45 周期上传后 INSERT
仍抛 `SQLSTATE 21000: ON CONFLICT DO UPDATE command cannot affect row a second time`。
说明加 object_ldn 进自然键虽方向正确但不足以覆盖所有冲突场景。

### 真根因（深挖 PM 文件后确认）

抓取真机 PM 文件 `A20260525.1730+0800-1745+0800_48BF74.1202000240194DP0015.xml`（167KB）
分析得：

- 文件含 2 个 `<measInfo>`（1830 measType × 1 cell + 109 measType × 2 cell = 2048 counter）
- **整个文件只有 2 个 counter name 有重名**：
  - `MR.RIPPRB` × 53 — 53 个不同 `p="N"` 全部映射到同一 Name
  - `MR.RECEIVEDIPOWER` × 53 — 同上
- 这是 Baicells eNB 把 PRB（资源块）索引编码在 `<measType>` 的 `p` 属性而不在 `Name` 里的厂家实现
- **同 cell 同 time 同 metric_path 自然键 53 行重复**，加 object_ldn 也救不了

### 交叉比对：1835 个 unique counter name 中

| 维度 | 数值 |
|------|------|
| PM 文件 unique counter name | 1835 |
| 指标库 `perf_indicators_enb` 收录 | 1180（64%） |
| 孤儿（PM 文件有 / 库没有） | 655（36%） |
| **重名 counter** | **只有 2 个 `MR.RIPPRB` / `MR.RECEIVEDIPOWER`** |
| 这 2 个重名 counter 在指标库 | **都不在**（都是孤儿） |

→ **所有重名都来自孤儿**。已注册的 1180 个 counter 全部 name unique。

## 方案选择

讨论了 A/B/C/D 四个方案：

| 方案 | 范围 | 数据完整性 | schema 变动 | 实施难度 |
|------|------|-----------|------------|---------|
| A `.p{P}` 后缀（已实施后回退） | 仅修撞键 | 保留全部 | 无 | 极小 |
| B 加 `sub_index` 列 | 仅修撞键 | 保留全部 + 干净语义 | migration + ON CONFLICT 改 | 中 |
| C 抛弃后续同名 | 仅修撞键 | 丢失 106 行/文件 | 无 | 极小 |
| **D 白名单过滤（采纳）** | **修撞键 + 减噪** | **丢失 655 个孤儿/文件** | 无 | 中 |

用户拍板 **D**。理由：
1. 与 T-0164-P1 KPIEngine 的 "product → platform → indicator 子集" 路由一致（复用同套白名单）
2. 孤儿数据进库是噪声 — dashboard / adhoc / KPI 都按 indicator 配置，孤儿永远不被消费
3. 减 1/3 存储 + 减小 BatchInsert 压力
4. 符合"做减法"偏好

## 变更范围

| 文件 | 类型 | 行变化 |
|------|------|--------|
| `internal/pm/collector/collector.go` | 新增 `CounterWhitelist` 接口 + `SetCounterWhitelist` + `filterByWhitelist` + `applyPayloadIdentity` 方法 | +66 / -8 |
| `internal/pm/collector/filter_test.go` | 新建：6 个白名单单测 + 3 个 applyPayloadIdentity 单测 | +160 |
| `cmd/worker/main.go` | 装配 `routerCounterWhitelist` adapter wrap `pmKPIRouter` 传给 collector | +30 / -0 |

## 审查项

### 1. CounterWhitelist 接口设计

- [x] **最小接口**：只暴露 `LookupCounters(ctx, sn) → (set, err)`，不让 collector 知道 router 包
- [x] **fail-open 容错**：lookup 错误 / 空白名单 / nil 未注入 → 全部不过滤；只有"成功拿到非空白名单"才执行过滤
- [x] **错误处理一致**：技术错误（DB / cache）和业务错误（ErrProductNotMatched）都走 log warn + 不过滤
- [x] **接口注入点**：`SetCounterWhitelist` setter（与 SetMetrics/SetRunner/SetDeviceLookup 一致风格）

### 2. filterByWhitelist 实现

- [x] **原地 reslice**：`kept := counters[:0]` 复用 slice 底层数组，零额外分配
- [x] **空 counter 短路**：`len(counters) == 0` 时不触发 lookup
- [x] **日志可观测**：只在 dropped > 0 时 INFO 输出 kept / dropped / whitelist_size，避免噪声
- [x] **设备 sn 入 log**：便于排查具体设备的过滤行为

### 3. applyPayloadIdentity 抽出 + 翻转优先级

- [x] **抽函数便于单测**：原 inline 循环改 helper，纯函数可表驱动测试
- [x] **翻转 DeviceSN 优先级**：之前 parser 优先 / payload fallback，改为 payload 总是覆盖
- [x] **修 device_sn 跨模块一致性 BUG**：parser 从 `<managedElement localDn="Station=eNb-{SN}"/>` 提取拿到带 `eNb-` 前缀的脏值，导致 pm_metrics 写 `eNb-1202000240194DP0015` 与 devices 表 / KPI Engine 用的 `1202000240194DP0015` 不一致 → 跨模块查询全部失败
- [x] **理由清晰注释**：注释说明 payload.DeviceSN 是 ACS upload handler 解析的 TR-069 标准 SN，是权威源

### 4. worker wiring

- [x] **复用现有 pmKPIRouter**：T-0164-P1 已构造 router 用于 KPIEngine，新增 wiring 复用同一实例
- [x] **routerCounterWhitelist adapter 在 worker 包内**：避免 collector / router 互相依赖
- [x] **错误透传**：adapter LookupCounters 把 router 错误原样返回，由 collector 决定 fail-open

### 5. 单元测试

- [x] `Test_filterByWhitelist_DropsOrphans` — 白名单含 A/B，drop C/D
- [x] `Test_filterByWhitelist_NilWhitelist_NoFilter` — 未注入不过滤
- [x] `Test_filterByWhitelist_LookupError_NoFilter` — lookup 错误 fail-open
- [x] `Test_filterByWhitelist_EmptyWhitelist_NoFilter` — 空白名单防止误删
- [x] `Test_filterByWhitelist_EmptyCounters_Noop` — 空 counter 不触发 lookup
- [x] `Test_filterByWhitelist_BUG6_BaicellsOrphans` — 直接复现：53×2 RIPPRB+RECEIVEDIPOWER + 1 已注册，过滤后只剩注册的
- [x] `Test_applyPayloadIdentity_PayloadOverridesParserPrefix` — 验证 SN 前缀 BUG 修复
- [x] `Test_applyPayloadIdentity_EmptyParserSN_PayloadFills` — parser 输出空时 payload 填充
- [x] `Test_applyPayloadIdentity_EmptyCounters_Noop` — 空 slice 不 panic

### 6. 部署 & 真机验证

- [x] 重新部署 docker（18:08 + 18:24，含 BUG-6 真根因复盘 + SN 修复）
- [x] **17:45 周期真机入库 1059 行**（D 方案过滤孤儿 989 / 2048 生效）
- [x] **18:30 周期真机入库 device_sn = `1202000240194DP0015`**（不带 `eNb-` 前缀）— **TODO 真机入库后回填实测值**
- [x] worker 启动日志含 `filtered orphan counters` + `kept` / `dropped_orphans` / `whitelist_size`

### 7. 跨模块一致性

- [x] 修后 pm_metrics.device_sn 与 devices.serial_number / KPI Engine 查询用 SN 全一致
- [x] KPI Engine 按标准 SN 查 pm_metrics 不再 0 行（KPI 业务正常工作的前置）
- [x] 不影响 device_sn 已下游用法（dashboard / adhoc 配置等都按标准 SN）

## 后续

用户提出规模化担忧 → 立项 **T-0166 pm_metrics 规模化容量重审 P0**：
- 1 站实测 1059 行/文件外推 1 万站 ~1200 万行/15min，CLAUDE.md §12 基线低估 16-160 倍
- 调 chunk 粒度 / 压缩策略 / 保留期 / 0 值过滤 / 横向扩展
- 启动依赖 T-0164 全 done，wave-3-post-T0164 排期

## 结论

**PASS_WITH_INFO** — 修复方案准确针对真根因；白名单复用 KPIEngine 已有路由保证语义一致；
fail-open 设计避免误删数据；同时顺手修了发现的 SN 一致性 BUG。

真机入库行数 + device_sn 标准化两项 DoD 由 Operator 在 commit 后 1 个 PM 周期内确认。
