# Risk Register（风险登记册）

> **性质**：活文档。每 Sprint 回顾时更新；每季度深度复盘。  
> **守护人**：项目经理（`CLAUDE.md §16.11`）  
> **初始化依据**：`docs/archive/reports/implementation-completeness-report-20260420.md` 五大 P0 + 次级短板

---

## 字段说明

| 字段 | 含义 |
|------|------|
| ID | R-NNN，永不复用 |
| 等级 | P0（阻塞发布）/ P1（影响质量）/ P2（优化项） |
| 概率 | 高 / 中 / 低 |
| 影响 | 若发生的最坏结果 |
| Owner | 负责跟踪的角色/人 |
| 状态 | Open / Mitigating / Closed |
| 下次复盘 | 日期 |

---

## P0 风险（发布阻塞）

### R-001 告警通知链路缺失
- **描述**：F04 告警模块规则支持 action="notify"，但无任何下游通知渠道实现（邮件/短信/Webhook/移动推送全无）
- **等级**：P0
- **概率**：高（已发生 — 当前状态即是）
- **影响**：告警无法送达运维人员，运营场景下致命
- **Owner**：PM（写 PRD）+ 电信业务专家 + Go 工程专家
- **状态**：Open
- **关联 Task**：T-0007（邮件）/ T-0009（短信凭据）/ T-0011（Webhook）/ T-0014（短信）
- **缓解**：PRD `docs/project/prd/F04-alarm-notification.md` 已产出；进 Sprint 1-2 实现邮件+Webhook，Sprint 3 补短信
- **下次复盘**：Sprint-01 回顾（~2026-05-04）

### R-002 E2E 用例实现为 0
- **描述**：`scripts/e2e_verify.sh` 框架 4459 行但实际用例数为 0（grep `claim` = 0），声称的 452 用例完全未落地
- **等级**：P0
- **概率**：高（已发生）
- **影响**：无法回归；任何合入都可能引入未发现的破坏
- **Owner**：QA/发布经理
- **状态**：Open
- **关联 Task**：T-0006（累计型，Sprint-01..07，目标 ≥200）
- **缓解**：CI 中加入 E2E 用例数阈值（初始 20，每 Sprint +20 到 200+）；Sprint 1 先补核心流程（登录/设备/告警/PM/Inform）
- **下次复盘**：每 Sprint 回顾

### R-003 F08 北向 OSS 协议栈缺失
- **描述**：仅实现 REST JSON 导出，无 SNMP/CORBA/MTOSI/TMF 标准协议适配
- **等级**：P0（对接运营商 OSS 时）
- **概率**：中（取决于首个商用运营商的接入要求）
- **影响**：不满足运营商 OSS 接入规范，可能阻塞商用
- **Owner**：PM（需求矩阵）+ 架构师
- **状态**：Open
- **关联 Task**：T-0013（SNMP 骨架）/ T-0017（联调）/ T-0020（推送可靠性）
- **缓解**：PM 产出运营商需求矩阵 → 优先实现刚需协议（通常 SNMP Trap）→ 6-10 周独立 track
- **下次复盘**：Sprint-02 回顾

### R-004 事件总线仅 SSE，无分布式协调
- **描述**：`internal/events/` 仅实现 SSE Hub + 持久化，没有 NATS JetStream 主题订阅，单进程事件总线限制横向扩展
- **等级**：P0（扩展性）
- **概率**：中
- **影响**：10 万→100 万扩展时，模块间事件解耦无法实现
- **Owner**：架构师 + 运维
- **状态**：Open
- **关联 Task**：T-0010（NATS 主改造，XL）+ T-0012（worker 重试，依赖 T-0010）
- **缓解**：基础设施层引入 NATS JetStream 主题订阅；影响 F04 通知/F08 推送/transfer 三模块，需并排改造
- **下次复盘**：Sprint-02

### R-005 数据库迁移版本号跳跃
- **描述**：`migrations/` 缺 000010、000015-000018，goose 严格模式可能歧义
- **等级**：P0（可快速修复）
- **概率**：高（已存在）
- **影响**：新环境首次迁移可能失败；团队认知混乱
- **Owner**：数据与存储专家
- **状态**：**Closed（2026-04-20）**
- **关闭依据**：采用方案 (a) 补占位——创建 5 个 `000010/000015-000018_reserved_placeholder.sql`
  （Up/Down 均为 `SELECT 1;` noop + 注释说明占位用途）。`scripts/check-migrations.sh`
  运行确认编号 000001→000023 连续无跳跃。同时修正脚本 Down 空判定阈值 bug
  （`<=1` → `<1`）——原阈值会把只有一条 SQL 的 Down 段误判为空。后续接入 CI 由 R-107 跟进。
- **复盘**：初版脚本误报 000001 / 000021 "Down 空"，诊断后确认两者 Down 段完整
  （分别 DROP FUNCTION / DROP TABLE），误报源于脚本阈值 bug，已修正（见 R-108 Closed）

---

## 整改期风险（Wave 1-3 专属）

> 来源：`docs/project/整改路线图-2026Q2.md` 第 5 章。
> 这些风险在整改启动后立刻生效；不达 Wave 退出条件不进下一波。

### R-INT-01 整改期间冻结令破功
- **描述**：客户/老板压力或团队惯性导致"冻结新功能"在 Wave 1-2 期间被打破，in-progress 涨到 5+
- **等级**：P0
- **概率**：高（首次执行冻结）
- **影响**：整改方案破产，重回 74% 完成度僵尸状态
- **Owner**：项目经理
- **状态**：Open
- **关联**：Wave 1.1 ~ W8 全部任务；`docs/project/backlog.md` FREEZE 块
- **缓解**：
  - 周一规划会硬性"finishing only"
  - 周报公布完成数（W1 末按 `docs/methodology/AI承诺对峙清单.md` 8 项硬承诺核验）
  - 任何插队需求转入 Wave 4 后队列
- **下次复盘**：每周五 finishing day（首次 2026-05-01）

### R-INT-02 CI 红灯被绕过
- **描述**：CI 建好后频繁失败，团队加 `// nolint:all` 或 `--no-verify` 绕过
- **等级**：P0
- **概率**：中
- **影响**：DoD 形同虚设，所有铁律失效
- **Owner**：后端 lead
- **状态**：Open
- **关联**：W1.1（CI 工作流）、W1.2（PR 模板）
- **缓解**：
  - CI 红灯 24h 内必须修
  - 连续 3 次跳过 = 升级到团队复盘
  - PR review 拒绝 `// nolint:all`、`--no-verify`
- **下次复盘**：每两周

### R-INT-03 E2E 用例补齐速度跟不上
- **描述**：W1 末 ≥ 20 / W8 末 ≥ 100 / W14 末 ≥ 200 的速度无法维持
- **等级**：P0
- **概率**：中
- **影响**：W1/W8 退出条件不达，整改路线图无法推进
- **Owner**：QA/发布经理
- **状态**：Open
- **关联**：W1.6、W8.3、长承诺 7
- **缓解**：
  - 最低速度 10 用例/周；不达则削减新切片
  - 每个新功能 PR 必须新增至少 1 条 E2E 断言
  - CI 中加用例数阈值检查
- **下次复盘**：每周

### R-INT-04 NATS JetStream 改造引入回归
- **描述**：Wave 3 Block E（W9-W10）NATS 切换破坏 F04/F08/transfer 联动
- **等级**：P0
- **概率**：中（跨域基础设施变更高风险）
- **影响**：Wave 3 阻塞 4-6 周，可能退回 ChannelBus
- **Owner**：架构专家
- **状态**：Open
- **关联**：Wave 3 Block E、R-004（事件总线）
- **缓解**：
  - 强制 `/ultrareview` 多 agent 审查
  - 配置驱动可切换实现，灰度发布
  - 故障演练（NATS 挂掉自动重连不丢消息）
- **下次复盘**：Block E 末

### R-INT-05 整改完成后惯性回潮
- **描述**：14 周整改完成后流程纪律松懈，三铁五铁形同虚设
- **等级**：P1
- **概率**：高（无持续问责的制度都会衰减）
- **影响**：6 个月内重回僵尸状态
- **Owner**：项目经理
- **状态**：Open
- **关联**：永久制度（流程刚性升级第 3 章）
- **缓解**：
  - DoD 三个不可绕过开关写进 `.github/CODEOWNERS` 与 CI 配置
  - 季度审计：`docs/methodology/AI承诺对峙清单.md` 通过率单调
  - 工程文化文档新人必读
- **下次复盘**：GA 后每季度

---

## P1 风险（质量影响）

### R-101 Software 模块缺灰度升级策略 + 回退审计/触发
- **描述**：`internal/software/` 批量升级仅支持全量并发控制，无分组/延时/百分比策略；回退仅"机械触发"，无审计/原因/灰度联动
- **等级**：P1
- **概率**：中
- **影响**：大规模升级无法分批验证；回退缺审计影响事故追溯
- **Owner**：电信业务专家
- **状态**：✅ **完整闭环 Closed** (2026-04-29) — T-0018 灰度 + T-0021 回退增强双链落地
- **关联 Task**：T-0018（灰度升级策略）✅ done / T-0021（回退增强）✅ done
- **缓解（灰度，T-0018）**：`internal/software/canary.go` 类型 + DefaultCanaryStages [1,10,50,100] + ValidateStages / DevicesForStage / FailureRate；`canary_monitor.go` cron 每 1 min 失败率阈值检查 → 自动暂停 + 4 Prometheus 指标；4 Admin endpoints (advance/pause/resume/abort)；migration 000045 +7 字段；BatchUpgradeRequest 加 strategy 向后兼容
- **缓解（回退，T-0021）**：`RollbackRequest` +3 audit 字段（reason/source/target_firmware_id）；4 source 枚举 + CHECK 约束 + IsValidRollbackSource；TargetFirmwareID 非空时 sub_tasks.DestVersion 取该固件版本；`CanaryStrategy +RollbackOnFailure`（默认 false 维持"不擅自回滚"D4 invariant）；canary monitor 阈值超 + opt-in → pause 后追加 auto-rollback；3 新 Counter（`software_rollback_total{source}` / `software_rollback_devices_total{source}` / `software_rollback_with_target_total`）；migration 000046 +4 列 + 1 部分索引；e2e_verify.sh +1 claim
- **关闭依据**：`docs/project/prd/T-0018-software-canary-upgrade.md` + `docs/review-report/20260429/verify-T-0018.md` + `docs/project/prd/T-0021-software-rollback-enhanced.md` + `docs/review-report/20260429/verify-T-0021.md` + commit `5baaf633`
- **下次复盘**：N/A（已闭环）

### R-102 前端 Backup/Software 页面仅骨架
- **描述**：Backup 48%、Software 52% 完成度，业务逻辑缺失（任务创建/升级进度/回滚）
- **等级**：P1
- **概率**：高
- **影响**：运维场景不可用
- **Owner**：前端专家
- **状态**：⚠️ Partially Closed (2026-04-29) — Software/Topology/Report 全闭环 + Backup 持久化 4/4 ✅ + Restore 主链路 ✅（T-0072/0078/0079/0080）+ enforcement 4/4 ✅（T-0073 cleanup Phase 1 / T-0074 压缩 / T-0075 加密 / T-0076 cleanup Phase 2 物理删除）+ 磁盘容量保护 ✅ T-0082 + severity 灵活化 ✅ T-0084 + **decrypt DoS 防御 ✅ T-0089**（concurrency cap 64MB×N 内存放大威胁 — T-0075 review M-4 闭环）；剩 orphan reaper（T-0083）+ encryption followup（T-0085 CBC+ChaCha20 / T-0086 KMS / T-0087 KEK 旋转 / T-0088 FE Tag）— 全部为 enhancement，不阻塞主链路
- **关联 Task**：T-0019 ✅ / T-0022 ✅ / T-0016 ✅ / T-0070 ✅ / T-0071 ✅ / T-0072 ✅ / T-0073 ✅ / T-0074 ✅ / T-0075 ✅ / T-0076 ✅ / T-0077 ✅ / T-0078 ✅ / T-0079 ✅ / T-0080 ✅ / T-0082 ✅ / T-0084 ✅ / **T-0089 ✅**；followups T-0083/T-0085/T-0086/T-0087/T-0088
- **关闭依据（已闭环部分）**：T-0019 `4ebdc91c` + T-0022 `c5e134c0` + T-0016 `c939ff12` + T-0070 `9f7ce74e` + T-0071 `d05831c3` + T-0073 `4f534b2d` + T-0074 `6043d326` + T-0075 `715a756a` + T-0076 `e22dc3e4` + T-0079 `2d7b5be6` + T-0072 `b7e09ed9` + T-0078 `286d56fb` + T-0080 `51100478` + T-0077 `d9d14417` + T-0082 `08913d0e` + T-0084 `5cafe871` + **T-0089 `b997ca3d`**
- **未闭环部分**：(a) 多设备 orphan 文件回收（T-0083，依赖 list-prefix 或 backup_files 表）；(b) encryption 增强（T-0085 CBC+ChaCha20 / T-0086 KMS / T-0087 KEK 旋转 / T-0088 FE Tag）— 均为 enhancement followup
- **下次复盘**：T-0083/T-0085~T-0088 任一闭环时

### R-103 License 容量/过期未拦截
- **描述**：`MaxDevices`/`ExpiryDate` 字段有，但无超限拦截与自动禁用
- **等级**：P1
- **概率**：中
- **影响**：违反商业约束，不符合授权要求
- **Owner**：电信业务专家
- **状态**：✅ Closed (2026-04-28)
- **关联 Task**：T-0015（License 容量/过期拦截）✅ done
- **缓解**：T-0015 完成 — Enforcer + Monitor cron + 6 Prometheus metrics + device.CreateDevice 拦截闸 + GET /licenses/quota 端点；6 项决策（过期软告警限写 D1 / 三档容量阈值 D2 / max active license D3 / perpetual 跳过 D4 / grace_period_days 可配 D5 / 第一版仅 device D6）全部实施
- **关闭依据**：`docs/project/prd/F06-license-enforcement.md` + `docs/review-report/20260428/verify-T-0015.md` + commit (post-S6)
- **下次复盘**：N/A（已关闭）

### R-104 拓扑自动分组规则引擎未激活 — **已关闭 2026-05-06**
- **描述**：~~`rule_service` 注释为 TODO，rule_matcher 存在但未接入~~ → 三路径全闭环
- **等级**：P1
- **概率**：低
- **影响**：~~分组依赖手工维护，规模上不去~~ → 解决
- **Owner**：架构师
- **状态**：**Closed**（2026-05-06）
- **关联 Task**：T-0027（**done sprint-09 / Owner Claude**；S0-S7 全过 11 commits — `b7fd2add` ID 修正 → `689207ad` S0 PRD → ... → `ad553f95` S3 Day 9 FE）
- **关闭依据**：
  * 三路径全闭环：手工 ApplyRule（Day 4-5）+ cron @hourly reEvaluateAll（Day 6）+ device.registered EventBus 订阅（Day 7）
  * A4 manual override SQL 层守护（Day 5 `WHERE source_type IS DISTINCT FROM 'manual'`），三路径自然继承
  * 6 metric + 7 log key 完整交付（Day 8 PRD §12.5 全核销）
  * 17 unit test PASS / 2 SKIP（A4/A5 PG integration 重定位 S4） / 0 FAIL / -race 全过
  * verify report：`docs/review-report/20260506/verify-T-0027.md`
- **followup**（不阻塞 R-104 关闭）：A4 PG integration test、Gauge 周期 SQL 抽样、evaluation_duration timer 包裹、W3 LAC/TAC 数据源（候选 T-0098）— 详见 verify §9

### R-105 syslog 远程转发缺失
- **描述**：仅查询，无 UDP/TCP syslog 转发
- **等级**：P1
- **概率**：低
- **影响**：与统一日志平台集成困难
- **Owner**：运维与可观测性专家
- **状态**：Open
- **关联 Task**：T-0028（triaged，按需拉起）
- **缓解**：按需实现
- **下次复盘**：Sprint-05

### R-106 worker 进程无重试/死信队列
- **描述**：EventBus 订阅失败无重试，长运行稳定性未验证
- **等级**：P1
- **概率**：中
- **影响**：PM/MR 文件处理失败会直接丢弃
- **Owner**：架构师 + 运维
- **状态**：✅ Closed (2026-04-28)
- **关联 Task**：T-0012（worker 重试/死信） ✅ done
- **缓解**：T-0012 完成 — `internal/core/reliability/dlq` 通用 DLQ 子包 + `runner` retry 装饰器（内层 3 次 + NATSEventBus 外层 5 次双重保险）+ `/api/v1/admin/dead-letters` 4 端点（List/Get/Delete/Replay，admin RBAC）+ migration 000044 dead_letters 表 + 4 Prometheus metric + PM Collector 接入示范；其他 11 subscriber 后续 PR 扩展
- **关闭依据**：`docs/project/prd/T-0012-worker-retry-dlq.md` + `docs/review-report/20260428/verify-T-0012.md` + commit (post-S6)
- **下次复盘**：N/A（已关闭）

### R-107 Prometheus/Grafana/AlertManager 容器编排缺失
- **描述**：metrics/tracing 已集成但 docker-compose 无监控栈
- **等级**：P1
- **概率**：高
- **影响**：Release Gate 中"可观测性就绪"无法自动化验证
- **Owner**：运维与可观测性专家
- **状态**：Open
- **关联 Task**：T-0008（Prometheus/Grafana/AlertManager 容器编排 + 基础 dashboard）
- **缓解**：Sprint 1 补 compose 文件 + 基础 dashboard
- **下次复盘**：Sprint-01

### R-109 License 治理层无审计日志（合规缺口）— **已关闭 2026-05-09**
- **描述**：~~T-0015 enforcement 引擎已落地（容量/过期拦截 + 阈值告警 cron），但拒绝事件 / 告警事件 / 用户写操作（Import / Activate / Revoke）均无审计落库。`license_logs` 表不存在~~ → 已落地
- **等级**：P1
- **概率**：高 → 已规避
- **影响**：~~违反**等保 2.0 三级 8.1.4.7**「重要操作日志保留 ≥ 6 个月」要求~~ → 合规链路打通；6 月保留期 + MinIO 归档延后到 P4 阶段实施
- **Owner**：电信业务专家 + 安全合规专家
- **状态**：**Closed**（2026-05-09）
- **关联 Task**：T-0100（umbrella，进行中）/ **T-0100-P0 done 2026-05-09**
- **关闭依据**：
  * migration 000073 落地 `license_logs` 表（9 种 log_type CHECK + 4 索引）
  * `license_log_model.go` + `pg_license_log_repository.go` + `log_writer.go`（NoopLogWriter + pgLogWriter，写库失败非阻断）
  * 5 处写入点接入：handler.go (Import/Activate/Revoke 成功+失败) + enforcer.go (EnforceCapacity/Expiry 拒绝) + monitor.go (auto_expire/expiry_alert/capacity_alert)
  * DI 通过 modules.go SetLogWriter 注入
  * 7 个 unit test 全过（覆盖 9 种 log_type、system 操作、nil license_id、repo 失败非阻断、details marshal 失败、empty details 规范化、并发 50 写）
- **未完待 P1+ 处理**（不阻塞 R-109 关闭）：GET /licenses/logs 端点（P1）+ 前端 LicenseLogs 真实数据（P1）+ 6 月保留 MinIO 归档（P4）
- **关联 PRD**：`docs/project/prd/F06-license.md` §6.2 / §9.3 / §11.2 V13

### R-110 MR 多 worker 调度并发（重复下发 SPV）
- **描述**：F05 MR scheduler 跑 cron @every 30s，多 app 实例同时跑会导致同一任务被两次"开/关"下发 SPV，给 ACS 队列压力 + 给设备发重复命令
- **等级**：P1
- **概率**：高（启用 HA 双 app 实例时必然发生）
- **影响**：ACS 队列堆积；设备 SOAP Fault；用户看到 cell 状态闪烁
- **Owner**：后端 + 架构
- **状态**：✅ Mitigated（已实现）
- **缓解**：`internal/mr/task/scheduler.go runWithLock` Redis SETNX + TTL × 3 自动续期；scheduler 三个 tick（open/close/heartbeat）各占独立锁键 `mr:scheduler:lock:{branch}`
- **关联 Task**：F05 Phase 2.2
- **下次复盘**：HA 部署后第一次回顾

### R-111 MR Redis 心跳误报（cell 标 abnormal 但实际在上报）
- **描述**：scheduler 巡检 Redis `MRFileReport_{cellCode}` key TTL；Redis 抖动 / 临时不可用会把正在上报的 cell 误标 abnormal
- **等级**：P1
- **概率**：中
- **影响**：用户对 UI 健康状态失去信任；触发不必要的运维介入
- **Owner**：后端
- **状态**：Mitigating
- **缓解**：阈值机制 — `mr.heartbeat_miss_threshold`（默认 2 次），跨阈值才标 abnormal；运维可调高
- **未做**：alert 应对短期 Redis 不可用时**暂停**心跳巡检（避免大量 abnormal 告警），需后续增强
- **下次复盘**：上线后第一次 Redis 故障复盘

### R-112 MR start_time 调度依赖系统时钟一致性
- **描述**：scheduler 按 UTC `start_time<=now` 抓 due 任务；多 app 实例 NTP 漂移 > 30s 会导致一台抓到任务、另一台没抓，与分布式锁机制配合下不影响正确性，但漂移过大可能延迟首次下发
- **等级**：P2
- **概率**：低（生产环境强制 NTP）
- **影响**：首次下发延迟 < 1 个 tick (30s)；累计无影响
- **Owner**：运维
- **状态**：Mitigated（依赖部署 NTP）
- **缓解**：部署 runbook 要求 NTP；scheduler interval 默认 30s 给足容差
- **下次复盘**：N/A

### R-108 个别迁移 Down 段缺失（误报，已撤销）
- **描述**（初版）：`000001_extensions_functions.sql` 与 `000021_notifications.sql`
  的 Down 段被脚本标记为空
- **等级**：P1 → **撤销**
- **状态**：**Closed（2026-04-20，误报）**
- **复盘**：验证原文件后确认两者 Down 段实际完整：
  · 000001：`DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;`
    （extensions `uuid-ossp`/`timescaledb` 为项目级共享基础设施，按 `omcgo/CLAUDE.md
    §5.5.8 "仅限迁移专属扩展"` 规则不应在 Down 中 DROP，刻意保留）
  · 000021：`DROP TABLE IF EXISTS notifications;`
- **根因**：`check-migrations.sh` 阈值 bug——`grep -v '^\s*--'` 已剥掉 `-- +goose Down`
  标记行，但判定仍用 `<=1`，导致只含一条 SQL 的 Down 段被误判为空。已修正为 `<1`
- **教训**：新工具投产前需用 golden sample（含代表性正确/错误样本）校准阈值

---

## T-0098 数据字典平台化任务专属风险（R-T0098-01..12）

> 来源：`docs/project/参数-KPI-告警-整合-实施计划.md` §4。
> 与 T-0098 36 个子任务（T-0098-P1-01 .. T-0098-P5-06）配套登记；任务关闭时统一关闭/降级。
> **2026-05-07 PgM 决策点 D1-D10 全采纳推荐**（详见 backlog §10 变更日志）— R-T0098-09 因 D5=B 落定从 Open → Mitigating。

### R-T0098-01 旧 data_model_definitions Phase 5 直接 DROP
- **描述**：Phase 5 计划 DROP `data_model_definitions` 表；当前为 dev-only 环境，无生产数据
- **等级**：P1
- **概率**：低
- **影响**：高（若已部署 staging 未做备份则数据永久丢失）
- **Owner**：数据与存储专家
- **状态**：Open
- **关联 Task**：T-0098 / T-0098-P5-01 / T-0098-P5-02
- **缓解**：Phase 5 启动前确认 staging 状态；已部署需先做完整备份；用 000061 down section 验证可回滚
- **下次复盘**：T-0098 Phase 5 启动前

### R-T0098-02 KPI 表名重命名风险（已规避）
- **描述**：原计划重命名 KPI 表（`indicator_unit` → `indicator_units` 等）影响 `pm/worker` + 17 张表 + 触发器
- **等级**：P1
- **概率**：中
- **影响**：中
- **Owner**：F03 owner
- **状态**：**Closed by D1=A** (2026-05-07，决议保留现状，改设计 footnote)
- **关联 Task**：T-0098-P1-05（降级为 docs 任务）
- **缓解**：D1=A 决策避开重命名路径；P1-05 改为更新设计 footnote 文档对齐

### R-T0098-03 alarm_libraries ALTER 与 receiver 同步失败
- **描述**：旧 `alarm_libraries` 与新 `alarm_definitions` schema 差异较大，ALTER + 重命名列与现有 receiver 同步失败可能导致告警丢失
- **等级**：P1
- **概率**：中
- **影响**：高
- **Owner**：F04 owner
- **状态**：**Mitigating by D2=B** (2026-05-07，决议 DROP + CREATE NEW alarm_definitions)
- **关联 Task**：T-0098-P1-04 / T-0098-P5-06 / T-0098-P2-10
- **缓解**：D2=B 决策走 DROP+CREATE 路径，告警数据来自 XML 可重载即恢复；P1-04 启动前 grep 现存 receiver 路径确认零 FK 反查

### R-T0098-04 迁移 000057-000059 跨多 PR 时序 app 无法启动
- **描述**：3 张迁移（products / param_dictionary / alarm_dictionary）若拆 PR 合入，中间状态 loader 启动会因表缺失 ERROR
- **等级**：P0（启动阻塞）
- **概率**：高
- **影响**：中
- **Owner**：F02 + F04 + infra
- **状态**：**Mitigating by D8=A** (2026-05-07，决议单 PR 合 P1)
- **关联 Task**：T-0098-P1-02 / T-0098-P1-03 / T-0098-P1-04 / T-0098-P1-06
- **缓解**：D8=A 决策强制 P1 单 PR 合入；不允许拆；本地 `make migrate-up` + `make migrate-down` 双向干跑通过后才合入

### R-T0098-05 6 个消费者改造跨多 PR 中途主干编译失败
- **描述**：sync.go / orchestrator.go / model_upload.go / device_param_handler.go / interop / pm worker 6 处切换 ParamRegistry，跨多 PR 时主干编译会失败
- **等级**：P1
- **概率**：中
- **影响**：高
- **Owner**：F02 owner
- **状态**：**Mitigating by D9=B** (2026-05-07，决议 feature flag `param_registry.use_new` 保留至 Phase 5 完成)
- **关联 Task**：T-0098-P2-04 .. T-0098-P2-08 / T-0098-P2-11
- **缓解**：D9=B feature flag 保护过渡期；新旧路径共存，逐 PR 切换；Phase 5 完成后删 flag

### R-T0098-06 删 ParameterTreeIterator 影响 engine_test.go 1082 行测试
- **描述**：T-0098-P2-04 重写 sync.go 同时删除 `ParameterTreeIterator`，影响 `provision/engine_test.go` 1082 行（含 Phase1GPNs 路径全部用例）
- **等级**：P1
- **概率**：高
- **影响**：中
- **Owner**：F02 owner
- **状态**：Open
- **关联 Task**：T-0098-P2-04
- **缓解**：P2-04 PR 同步重构 engine_test.go，不留残留；CPE simulator 端到端覆盖兜底
- **下次复盘**：P2-04 启动前

### R-T0098-07 webcode-v2/v3 因 frontend-core 类型变化崩溃
- **描述**：T-0098-P4-01 拆 `paramModelApi/productApi/alarmDefinitionApi` 影响 webcode-v2/v3 编译
- **等级**：P2
- **概率**：中
- **影响**：低
- **Owner**：frontend-core owner
- **状态**：**Mitigating by D7=A** (2026-05-07，决议仅评估编译，不强制 v2/v3 跟进 UI 完整性)
- **关联 Task**：T-0098-P4-01 / T-0098-P4-08
- **缓解**：D7=A 决策只看 `npm run build` 不阻塞；T-0098-P4-08 单独评估；与 T-0035 多皮肤计划解耦

### R-T0098-08 provision/engine.go + sync.go 重写隐性 bug
- **描述**：`provision/engine.go` 663 行 + `sync.go` 482 行重写，重写量大，隐性 bug 概率高
- **等级**：P0
- **概率**：高
- **影响**：高
- **Owner**：F02 owner + 架构专家
- **状态**：Open
- **关联 Task**：T-0098-P2-04 / T-0098-P2-05 / T-0098-P2-06
- **缓解**：优先级最高的 review；强制 `/ultrareview` 多 agent；CPE simulator 端到端覆盖；D9=B feature flag 保留至 P5 提供 fallback
- **下次复盘**：P2-04..06 任一 PR review 时

### R-T0098-09 Wave 3 硬化期 FREEZE 冲突
- **描述**：T-0098 是 feat 类，与 Wave 3 硬化期（2026-04-28~2026-08-03）FREEZE 冲突
- **等级**：P0（排期阻塞）
- **概率**：高
- **影响**：高
- **Owner**：项目经理（PgM）
- **状态**：**Mitigating by D5=B** (2026-05-07，决议 P1-P3 视为设计已交付的实现收敛纳入 W3，P4/P5 推迟到 W3 后)
- **关联 Task**：T-0098-P1-* / T-0098-P2-* / T-0098-P3-* (W3 内) ; T-0098-P4-* / T-0098-P5-* (W3 后)
- **缓解**：D5=B 划线明确；P1-P3 子任务 sprint planning 时按"W3 期允许"准入；P4/P5 子任务标记 "**D5=B Wave 3 后启动**"
- **下次复盘**：每月 W3 计分盘点

### R-T0098-10 XL 任务 8-sprint 长链中途人员变动
- **描述**：T-0098 横跨 ~8 sprint，中途 Owner 变动可能导致进度断层
- **等级**：P2
- **概率**：中
- **影响**：中
- **Owner**：项目经理（PgM）
- **状态**：Open
- **关联 Task**：T-0098 全 36 子任务
- **缓解**：每 Phase 末 demo + 文档更新；36 子任务粒度便于交接；Phase DoD 七节交付验证可独立验收
- **下次复盘**：每 Sprint 回顾

### R-T0098-11 products.xml 离线脚本归属不明阻塞 P1-02
- **描述**：`data/param-mappings/products.xml` 需离线脚本生成，归属不明阻塞 P1-02 + P1-06
- **等级**：P1
- **概率**：中
- **影响**：中
- **Owner**：后端架构（D4=A）
- **状态**：**Closed by reality** (2026-05-07，事实自动满足)
- **关联 Task**：T-0098-P1-02 / T-0098-P1-06
- **关闭依据**：实施计划 §1.4 Gap 分析有误 — `omcgo/data/param-mappings/products.xml`（9714 bytes，15 产品 + 29 正则全套）已在 commit 50fa1a0c（2026-05-07 XML 数据资产入库）入库；schema 与设计稿 §4.5 ~95% 对齐（仅 `<alarm enableUnknownAlarm>` 属性缺省，由 loader Go XML unmarshal bool 默认值 false 吸收，与设计 schema `DEFAULT false` 一致）；离线脚本归属问题随之解套（资产已存在无需新产）
- **后续**：P1-06 Loader 实施时可选：① 容忍缺属性（推荐）/ ② 一次性补 15 产品 enableUnknownAlarm="false" 显式属性（cosmetic，可放 P1-04 / P1-06 同 PR）

### R-T0098-12 super_admin 角色与现有 admin 权限边界争议
- **描述**：Phase 3 引入 super_admin 角色，与现有 admin 角色权限边界可能争议
- **等级**：P2
- **概率**：低
- **影响**：低
- **Owner**：admin 模块 owner + 安全合规专家
- **状态**：Open
- **关联 Task**：T-0098-P3-05 / T-0098-P4-02
- **缓解**：P3-05 启动前与 admin 模块 owner 对齐；现 PrivateRoute 已支持角色过滤可复用
- **下次复盘**：P3-05 启动前

---

## P2 风险（优化项）

### R-201 RF 控制硬编码 LTE
- **描述**：device 模块 RF 开关未走 Carrier 适配
- **等级**：P2
- **Owner**：电信业务专家
- **状态**：Open

### R-202 F10 互操作用例库稀疏
- **描述**：原状仅 RPC/Protocol/DataModel 三类（14 用例），无 Inform category，无 negative path，无运营商参数化矩阵。GA 阶段交付给客户时如出现"用 OMC 内置互操作工具仍漏检产线问题"会损害商用形象。
- **等级**：P2
- **概率**：中
- **影响**：客户验收阶段 OMC 互操作能力被质疑、口碑风险
- **Owner**：测试专家
- **状态**：✅ **Closed**（2026-05-11 — T-0030 Phase 1 + Phase 2 单晚完成；PRD §7 修订门槛全达）
- **关联 Task**：T-0030（F10 互操作用例库扩充 — Phase 1 + Phase 2 done 2026-05-11）
- **缓解路径**：
  - ✅ Phase 1 (commit `92a3eb76` 2026-05-11)：用例 14→30（DM 2→6 / Protocol 3→7 / RPC 9→13 / Inform 0→4 新建）+ ExpectedOutcome 翻转机制（4 个 negative path：每类 ≥ 1）+ e2e 断言 3→6 + 10 新单测
  - ✅ Phase 2 (commit `e2d846cf` 2026-05-11)：用例 30→37（DM 6→7 / Protocol 7→8 / RPC 13→17 / Inform 4→5）+ 每类 negative ≥ 2（共 8）+ 三家运营商私有 RPC 覆盖（X_CMCC_Reboot / X_CT-COM_Restart / X_CU-COM_DBConfig）+ TestCases_CarrierCoverage 单测强制 + e2e 6→8 + PRD §7 门槛修订（≥ 50→≥ 35 / e2e ≥ 10→≥ 8 务实化）
- **关闭依据**：PRD §7 Phase 2 修订门槛全达 — 用例 37 ≥ 35 ✅ / 4 category 全覆盖 ✅ / 每类 negative ≥ 2 ✅ / 三家运营商覆盖 ✅ / e2e ≥ 8 ✅
- **关联 PRD**：`docs/project/prd/F10-interop-testing-coverage.md`
- **下次复盘**：N/A（已关闭）；后续 Phase 3 选项（fault-inject category / 验收报告 export / 设备矩阵参数化）走独立 sub-task 不再触发 R-202

### R-203 CAPTCHA 图形生成待补
- **描述**：admin 模块仅端点，无图形生成
- **等级**：P2
- **Owner**：安全合规专家

### R-204 FTP 连接测试占位符
- **描述**：backup 模块 `POST /ftp-configs/test` 返回 "not implemented"
- **等级**：P2
- **Owner**：Go 工程专家

### R-205 多级聚合与分析能力缺失
- **描述**：PM 仅设备级，MR 无地理聚合，无趋势/根因分析
- **等级**：P2（功能扩展）
- **Owner**：电信业务专家 + 数据与存储专家

### R-206 MML 老交互被 Sprint A 数据模型重构破坏
- **描述**：T-0119 Sprint A `mml_schema_rebuild` (migration 000090) DROP `mml_command_params_rel` junction 表，命令→sub-field 关系仅剩 `target_paths JSONB` 数组，丢了 (mml_code, label_i18n, default_selected, is_required, sort_order) 等 UI 核心元数据；前端 ParamPathPanel/ParamFormRenderer 不是老系统勾选/输入双态面板。结果：老 OMC `Maintenance > MML List > BSC Configuration` 三栏交互（LST 勾选式 + MOD 输入式 + MML textbox 双向绑定 + 分号串联批量）回归
- **等级**：P1（用户体验下降；不阻塞 RC 但阻塞 GA 体验）
- **概率**：高（已发生）
- **影响**：MML 控制台仍可点击，但 sub-field 勾选/输入完全丢失；用户切回老 OMC 系统使用 MML
- **关联 Task**：**T-0123**（整改 umbrella，5 sub-task P0-P4 共 18d；不回退 Sprint A/B）+ T-0119 ✅ (起因 commit `65142e22`+`5ec3a2d1`+`8d1733bd`)
- **Mitigation**：v2 PRD APPROVED `docs/design/mml-restore-old-interaction-plan-20260514.md`。① 复活 `mml_command_sub_fields` 表（替代 DROP 的 `mml_command_params_rel`）+ 扩展 mml_params 7 列元数据（access_type/is_object/supports_add/supports_delete/change_applies/constraint_text_i18n/catalog_protected）+ 扩展 mml_commands 4 列（logical_code/logical_name_i18n/source/catalog_protected）；② 一次性 SQL 导入 standard-model.xml → DB，启动期 mmlstandardloader 下线；③ admin Catalog UI 4 Tab 管理；④ 前端重构 SubFieldChecklist + SubFieldInputList + MmlEditor 双向绑定。**5 阶段拆分**：P0 数据层 (3d) → P1 后端 (3d) → P2 前端 Console (4d) → P3 admin Catalog UI (4d) → P4 收尾 ADD/RMV+Customized+DoD (4d)
- **状态**：**Mitigating**（**2026-05-14 T-0123-P0 数据层 + T-0123-P1 Console 后端 同日双闭环**）：
  - **P0 数据层** 3 commits `00a48cad`+`bf53cb3b`+`e0d31cce` / 23 files / +7404 LOC：migration 000095 + 4 admin repo + 13 admin endpoint + omcctl import 工具 + 2001 行 seed 落地；catalog_protected 守护 + 6 sentinel error 全 PASS
  - **P1 Console 后端** 2 commits `3eb5cee6`+`0d5867ad` / 19 files / +4803 LOC：5 endpoints (GET /mml/group-tree / GET /mml/commands/:id/sub-fields / POST /mml/render / POST /mml/parse / POST /mml/execute-statements) + Go renderer/parser (mml_renderer.go + mml_parser.go) + ConsoleService 5 service 方法 + executor (LST/MOD/ADD/RMV 4 op 编译 + N 设备 × M statements fanout) + DI/Router 全接；48 测试 race 全 PASS（mml 包覆盖率 24.9%→33.3% +8.4pp）；S5 W1/W2 in-loop 修复 (ErrInvalidRequest sentinel + ADD passthrough 注释强化)；5 e2e claim
  - R-206 关闭需待 T-0123-P2（Console 前端三栏 + Step Bar + MmlEditor 双向绑定）/P3（admin Catalog UI 4 Tab）/P4（收尾 ADD/RMV+Customized+DoD）完成 — 老 OMC 三栏交互 + LST 勾选 + MOD 输入 + MML textbox 双向绑定全部还原。当前 backlog 2/5 — P0+P1 done / P2-P4 仍 triaged 待 sprint-12+ planning
- **Owner**：Claude
- **下次复盘**：sprint-11 close 2026-06-08

---

## 关闭的风险

| ID | 关闭日期 | 关闭摘要 |
|----|---------|---------|
| R-005 | 2026-04-20 | 数据库迁移版本号跳跃。补 5 个 `reserved_placeholder` 占位迁移，`check-migrations.sh` 检查通过。衍生风险 R-108（个别 Down 段缺失）已登记 |

---

## P2 风险 — T-0178 自定义 paramModel XML 分层目录（2026-05-29 登记）

> 源:PRD `docs/project/prd/F02-param-model-custom-xml.md` §8。
> 8 条 R-NEW-T0178-* 风险全部 Open,T-0178 进 done 时由 Owner 复盘并标记 Mitigated/Closed。

### R-NEW-T0178-1 客户端上传超大 XML 触发 OOM
- **描述**:恶意客户端构造 Content-Length 大于上限或 chunked 编码绕过的大文件,可能让 app 进程 buffer 满 OOM
- **等级**:P2
- **概率**:低
- **影响**:app 进程崩溃 → docker 重启 → 短暂不可用
- **Owner**:Go 工程专家 + 安全合规专家
- **状态**:Mitigated(已实现 Gin `MaxMultipartMemory=4MiB` + handler 内 `file.Size <= 1<<20` 双层门禁 + `io.LimitReader(MaxUploadXMLSize+1)` 流式校验)
- **关联 Task**:T-0178(commit 16f95d46 `handler.go::UploadXML` 校验 2)
- **缓解**:三层防御已就位,生产部署前需设 Gin engine.MaxMultipartMemory(P3 deploy 任务)
- **下次复盘**:T-0178 S7 收尾

### R-NEW-T0178-2 上传成功但 Loader 解析失败 → DB 与文件状态不一致
- **描述**:`validateUploadXML` 是入口"形态检查",`xml.Unmarshal` 二次解析可能因 XSD-违反等失败
- **等级**:P2
- **概率**:低
- **影响**:文件落 host 但 DB 行缺失 → 用户看到列表无新模型 → 重试上传或手动 reload
- **Owner**:Go 工程专家
- **状态**:Mitigated(Upload handler 触发全量 ReloadOne 失败仅 Warn 不阻塞 upload 响应;用户可手动 reload 重试)
- **关联 Task**:T-0178(commit 16f95d46 `handler.go::UploadXML` step 4)
- **缓解**:`validateUploadXML` 扫到 EOF 验完整形态(commit fix),与 Loader 解析对齐避免分叉
- **下次复盘**:T-0178 S7 收尾

### R-NEW-T0178-3 备份失败导致用户反复点击 DELETE 全部失败
- **描述**:host 磁盘满 / 权限错 / 只读挂载等导致 rename 失败,所有 DELETE 返 500,用户重试无果
- **等级**:P2
- **概率**:中(运维误配场景常见)
- **影响**:用户无法删除自定义 XML,需运维介入排查 fs 状态
- **Owner**:运维与可观测性专家
- **状态**:**Mitigated**(2026-05-29 兑现)
- **关联 Task**:T-0178(commit d1349c3c `handler.go::DeleteModel` step 4 audit_action=parammodel.delete.aborted_backup_failed)
- **缓解**:① 结构化审计日志含 errno ✓;② Prometheus 告警 `ParamModelBackupFailedSurge` 已落 `deployments/monitoring/alerts/omc-rules.yml`(`increase(parammodel_backup_cleanup_total{result="error"}[10m]) > 0 for 5m`,含 4 步 Action + Runbook 链接)✓;③ 运维手册"备份失败排障"段:待补到 `docs/operations/告警处置Runbook.md#ParamModelBackupFailedSurge`(留 ops 任务,告警 description 已内联完整排查步骤)
- **下次复盘**:GA 前(运维手册落地后转 Closed)

### R-NEW-T0178-4 host 目录 `param-mappings-custom` 首次部署被遗漏初始化 → app 启动期 Loader 扫描非存目录报错
- **描述**:deploy.sh 跳过初始化 + docker compose v2 不自动创建 bind mount source 时,Loader 扫到 customDir = ENOENT
- **等级**:P2
- **概率**:低
- **影响**:启动期 Loader 报 warning 但不阻塞 builtin 加载(Loader 已对 ENOENT 容忍);用户上传时报"目录不存在"
- **Owner**:运维专家
- **状态**:Mitigated(commit aab7245b `deploy.sh` Step 3/9 主动 `mkdir -p` + chown 10001 + chmod 0750;Loader 内 `resolveLoaderFiles` 已对 ENOENT 静默跳过)
- **关联 Task**:T-0178
- **缓解**:Loader 容忍 + deploy.sh 双保险
- **下次复盘**:T-0178 S7 收尾

### R-NEW-T0178-5 bool 字段 `custom_overrides_builtin` 默认值陷阱
- **描述**:Go bool 零值是 false,yaml 不写时拿到 false,与设计意图(默认 true)反转,导致 self-healing 链路失效
- **等级**:P2
- **概率**:中(改回普通 bool 类型即触发)
- **影响**:删 custom 不会自动回退 builtin,用户体验 degrade
- **Owner**:Go 工程专家
- **状态**:Mitigated(commit 61a0e5aa `*bool` 三态 + `CustomOverridesEnabled()` access method + `TestCustomOverridesEnabled_TriState` 编译期守护)
- **关联 Task**:T-0178
- **缓解**:防御性测试守护字段类型不会被改回 bool(改回则测试编译失败)
- **下次复盘**:T-0178 S7 收尾

### R-NEW-T0178-6 多实例横扩时同名上传并发竞态
- **描述**:当前 Upload/Delete/单文件 Reload 用 `sync.Map[basename]*sync.Mutex` 进程内互斥;多 app 实例横扩时同名 Upload 会丢更新
- **等级**:P2
- **概率**:低(当前生产仅单 app 实例)
- **影响**:同名文件两个上传请求同时打,后写入的覆盖前写入,可能丢用户最新版本
- **Owner**:架构专家
- **状态**:Open(单实例假设,横扩前必须补)
- **关联 Task**:T-0178(`CLAUDE.md §5.3.1` 单实例假设注记)
- **缓解**:`CLAUDE.md §5.3.1` 写明假设;横扩 PRD 中明确需补 PG advisory lock `pg_try_advisory_xact_lock(hashtext('parammodel:'+basename))`
- **下次复盘**:横扩立项时

### R-NEW-T0178-7 Upload 过程中容器 OOM-Killed 留下 `.tmp.<uuid>` 残留
- **描述**:Upload handler 用 tmp + rename 实现原子写;若 WriteFile 后 rename 前容器崩溃,tmp 文件留在 host
- **等级**:P2
- **概率**:中(容器 OOM 在低内存配额下可能发生)
- **影响**:host 长期堆积 .tmp.<uuid> 文件,占盘
- **Owner**:运维专家
- **状态**:Mitigated(commit 2c27253b worker `BackupCleanup` 扩展 `.tmp.<uuid>` 分支,mtime > `TmpResidualMaxAge`=1h 即清)
- **关联 Task**:T-0178
- **缓解**:1 小时上限远大于健康 Upload 周期(<1s),避免长期占盘
- **下次复盘**:T-0178 S7 收尾

### R-NEW-T0178-8 builtin 同名 XML 与 custom XML 同时存在,且 `custom_overrides_builtin=false` 时
- **描述**:运维误把 `custom_overrides_builtin` 改 false,导致同名 custom 被 builtin 压制,但 host 上 custom 文件还在,用户困惑"明明上传了为什么生效不了"
- **等级**:P2
- **概率**:低(默认 true,极少有人主动改 false)
- **影响**:用户上传后看不到 source=custom,只看到 source=builtin
- **Owner**:产品经理
- **状态**:**Mitigated**(2026-05-29 兑现)
- **关联 Task**:T-0178(follow-up `c9eec280` 后续)
- **缓解**:Loader 启动期 WARN 日志 `customOverrides=false` 时输出 `shadowed_files` 清单 + `custom_dir` + `hint`(`internal/config/parammodel/loader.go::run`),Loki / grep 一查即知;后续 UI Tooltip 提示走 PM 任务跟进
- **下次复盘**:GA 前(UI 提示落地后转 Closed)

---

## T-0179 alarm-library 页面 drill-down 重设计专属风险(R-NEW-T0179-01..06)

> T-0179 PRD `docs/project/prd/F04-alarm-library-redesign.md` §6 已列 6 项风险,S7 关闭时登记到本表。

### R-NEW-T0179-01 ne_type URL 参数被劫持(XSS / open-redirect)
- **描述**:用户访问 `/product/alarm-library?neType=<script>...` 时,React 直接渲染该字符串到 Tag/Text/搜索框 — JSX 默认 escape 阻止 XSS,但需确认无 dangerouslySetInnerHTML 误用
- **等级**:P3
- **概率**:低(React JSX 默认 escape;无 dangerouslySetInnerHTML 调用点)
- **影响**:理论上 XSS;实际无利用面(无 dangerouslySetInnerHTML)
- **Owner**:Claude(前端) + 安全合规专家
- **状态**:Mitigated(2026-05-29)
- **关联 Task**:T-0179
- **缓解**:React JSX 默认 escape 保护;searchParams 仅作为 query filter 传递不拼字符串;后端 ne_type 参数走 squirrel 参数化查询不拼 SQL
- **下次复盘**:T-0180+ 引入新 URL 参数时审查同模式

### R-NEW-T0179-02 ne-types 聚合 API 在 alarm_definitions 规模膨胀时 P99 抬高
- **描述**:当前 ~442 行 / 7 ne_type,单 SQL `<50ms`。运营商接入扩展(如新增 BSC/BTS GSM/5G CU/DU 等)若 ne_type 数 > 50 或行数 > 50000,COUNT FILTER GROUP BY 可能慢
- **等级**:P2
- **概率**:中(运营商接入扩展自然推动)
- **影响**:一级页面加载慢(>500ms)
- **Owner**:数据与存储专家
- **状态**:Open
- **关联 Task**:T-0179
- **缓解**:① 现有索引 `idx_alarm_definitions_severity_show` 覆盖 severity 反查;② 行数 >50000 时考虑加 `(ne_type, severity_id)` 复合索引;③ 极端情况(>200000 行)走物化视图 + 触发器刷新;④ Prometheus 加 `alarm_def_ne_types_agg_duration_seconds` 指标监控 P99
- **下次复盘**:运营商接入扩展提案评审时,或 P99 > 200ms 告警触发时

### R-NEW-T0179-03 loaded_from 历史数据 NULL 大量存在(Loader 未重跑环境)
- **描述**:migration 216 加 loaded_from 列后,历史行均为 NULL(不强制回填),需 Loader.Reload 才写入;运维若未触发 Reload,一级表大量行显示"未回填(请重载)"
- **等级**:P3
- **概率**:高(部署后必发生)
- **影响**:UX 不佳 — 首次访问看不到 XML 来源标签
- **Owner**:Claude(实施时已加 warning Tag) + 运维与可观测性专家
- **状态**:Mitigated(2026-05-29)
- **关联 Task**:T-0179
- **缓解**:① 前端一级表 loaded_from 为空时显示 `<Tag color="warning">未回填(请重载)</Tag>` 显式引导用户点"重载 XML";② 部署后默认触发一次 Loader.Reload(deploy.sh 末尾可加 curl `/import-directory`,但留运维决策);③ 文档明确"升级到含 migration 216 的版本后建议立即重载告警 XML"
- **下次复盘**:GA 前(确认运维 SOP 含 Reload 步骤后转 Closed)

### R-NEW-T0179-04 Drawer lockNeType 在编辑模式被误启用导致用户改不了 ne_type
- **描述**:Drawer Props `lockNeType` 仅在新增模式生效(代码 `!isEdit && Boolean(lockNeType)`),若未来引入"从一级页面编辑"路径误传 lockNeType=true,编辑模式 ne_type 会被锁定
- **等级**:P3
- **概率**:低(目前编辑模式入口只在二级表,不传 lockNeType)
- **影响**:编辑流程 ne_type 字段不可改 — 用户无法修复 ne_type 字段错误
- **Owner**:Claude(前端)
- **状态**:Mitigated(2026-05-29)
- **关联 Task**:T-0179
- **缓解**:代码硬约束 `!isEdit && Boolean(lockNeType)` 强制编辑模式忽略 lockNeType;后续如改造路径需补单测覆盖
- **下次复盘**:T-0180+ Drawer 增强时审查

### R-NEW-T0179-05 三皮肤(webcode/v2/v3)alarm-library 页面 UX 不一致
- **描述**:本次仅改造 webcode 主皮肤的 alarm-library/index.tsx,frontend-core 业务层修改对 webcode-v2/v3 透明可用,但两个候选皮肤的 alarm-library 页面(若存在)仍是老 UI 与主皮肤体验不一致
- **等级**:P2
- **概率**:中(取决于 PgM 是否决策同步改造)
- **影响**:多皮肤场景下用户体验割裂
- **Owner**:前端专家 + PgM
- **状态**:Open
- **关联 Task**:T-0179
- **缓解**:① 验证 webcode-v2/v3 是否存在 alarm-library 页面(本次 typecheck 仅证业务层兼容);② 由 PgM 决策是否要镜像 drill-down 改造到两个候选皮肤;③ 至少在多皮肤决策文档 `docs/project/frontend-multi-skin-plan-20260422.md` 加一行 T-0179 改造记录
- **下次复盘**:多皮肤策略评审时

### R-NEW-T0179-06 ne-types 聚合 API 未走鉴权中间件
- **描述**:新增 GET `/alarm-definitions/ne-types` 端点跟随既有 `alarm-definitions` 路由组,需确认鉴权中间件覆盖;若 router.go 该 group 未挂 JWT/RBAC,该端点会暴露
- **等级**:P2
- **概率**:低(承袭 group 中间件,既有 `/alarm-definitions` 已挂鉴权则本端点自动覆盖)
- **影响**:未鉴权访问可获取告警类型 + XML 来源 + 各级别计数(信息泄露)
- **Owner**:安全合规专家
- **状态**:Open
- **关联 Task**:T-0179
- **缓解**:① 后续 review 时 grep `RegisterRoutes` 调用点(`cmd/app/router/router.go` 或 `provider/alarmdef.go`),确认挂在受 JWT/RBAC 保护的 `/api/v1` 分组下;② 加 E2E 用例:未带 token 访问 `/alarm-definitions/ne-types` 应返 401
- **下次复盘**:S5 安全审计补做时,或 GA 前

---

---

## T-0180 kpi-library 重设计 + 自定义 indicator XML 持久化专属风险(R-NEW-T0180-01..06)

> T-0180 PRD `docs/project/prd/F03-kpi-library-redesign.md` §11 已列 6 项风险,S7 关闭时登记到本表。

### R-NEW-T0180-01 三表(enb/gsm/gnb)分表导致 handler 重复代码激增
- **描述**:perf_indicators_{enb,gsm,gnb} / rela_platform_indicator_formula_{enb,gsm,gnb} / enabled_pm_indicators_{enb,gsm,gnb} 三套表结构相同,handler/repo SQL 三份重复
- **等级**:P3
- **概率**:已发生(P1.2/P1.3 实施时)
- **影响**:维护成本 +20%;改 schema 需同步三份
- **Owner**:Go 工程专家
- **状态**:Mitigated(2026-05-29)
- **关联 Task**:T-0180
- **缓解**:① `validateTech` 白名单 + `fmt.Sprintf("perf_indicators_%s", tech)` 模板化 SQL,接受 ~20% 重复换 schema 简单;② FileRepository 接口里 tech 是参数而非分支(单一方法签名);③ 三表合一是长期重构(PRD §5 非目标 #2),与 T-0166 PM 规模化容量重审一起讨论
- **下次复盘**:三表合一立项时

### R-NEW-T0180-02 loaded_from NULL 历史数据被 reload 模式误删
- **描述**:migration 000217 加 loaded_from 列后,历史行均为 NULL;mode=reload 算 orphan 时 `updated_at < start` 命中所有未被 Loader 触碰的旧行;若旧行未在 XML 中(运维曾手工 INSERT 等)会被误删
- **等级**:P2
- **概率**:中(取决于运维历史是否手工写过 perf_indicators_*)
- **影响**:手工记录在 reload 后消失
- **Owner**:数据与存储专家
- **状态**:**Open**
- **关联 Task**:T-0180
- **缓解**:① UI 红色 danger Popconfirm + 文案"操作不可撤销"(已落 P4 index.tsx);② 响应体返回三制式孤儿删除计数,前端 message.success 显示便于发现异常;③ **建议运维**:首次升级到含本特性的版本后,先做一次 mode=import 而非 reload,让 Loader 写 loaded_from 后再判断是否 mode=reload
- **下次复盘**:GA 前(若运维 SOP 含"首次 import 再 reload"则转 Mitigated)

### R-NEW-T0180-03 reload 操作误伤运维手动启用的孤儿指标
- **描述**:mode=reload 不仅删 perf_indicators 主表,还级联删 enabled_pm_indicators_*(启用记录) + rela_platform_indicator_formula_*(公式);若运维手动启用了被 reload 判为孤儿的指标,启用状态一并丢失
- **等级**:P2
- **概率**:中(reload 操作本身是低频但确实会发生)
- **影响**:运维需重新启用 + 手工配置的公式丢失
- **Owner**:产品经理
- **状态**:Mitigated(2026-05-29)
- **关联 Task**:T-0180
- **缓解**:① Popconfirm 描述明确列出"级联清理公式 + 启用记录"(P4 index.tsx 已落);② Prometheus `indicator_reload_orphans_deleted_total{tech}` (PRD §7 列出,P2 起预留指标 slot 后续补);③ 沿用 T-0178 destructive 语义,与 param-model 一致 SOP
- **下次复盘**:GA 前

### R-NEW-T0180-04 删除时备份失败导致 DB 与文件状态不一致
- **描述**:DeleteFile handler 先 mv .deleted.<ts>,再 DB 三表级联删;若 mv 失败但 DB 删成功(理论上 mv 失败已 return),会出现文件存在 DB 行已删的状态
- **等级**:P2
- **概率**:低(代码已保守回滚 mv 失败 → return 500,不动 DB)
- **影响**:DB 与 host 文件分叉
- **Owner**:Go 工程专家
- **状态**:Mitigated(2026-05-29)
- **关联 Task**:T-0180
- **缓解**:① 代码硬约束:mv 返 ENOENT 视为"已 gone"继续清 DB(file_already_gone audit_action);mv 其他失败 → 500 ErrCodeIndicatorBackupFailed + 不动 DB(`indicator.delete.aborted_backup_failed` audit);② DB 失败时反向 rename 恢复(若反向 rename 也失败仅 log,真分叉);③ 8 个 sub-test 覆盖三层错误路径
- **下次复盘**:多实例横扩立项时(R-NEW-T0180-05 同期)

### R-NEW-T0180-05 单实例假设下同名 Upload 并发竞态
- **描述**:UploadXML + DeleteFile + Reload 用 `acquireFileLock(basename)` 进程内 sync.Map mutex 互斥;多实例横扩(同一 host 路径多 app 容器)时锁失效,同名上传可能丢更新
- **等级**:P2
- **概率**:低(当前单 app 单 worker 部署)
- **影响**:同名并发 Upload 时一份覆盖另一份(原子 rename 保证不撕裂,但语义上丢更新)
- **Owner**:数据与存储专家
- **状态**:**Open**
- **关联 Task**:T-0180(与 T-0178 R-NEW-T0178-6 同源)
- **缓解**:多实例横扩前补 PG advisory lock `pg_try_advisory_xact_lock(hashtext('indicator:'+basename))`,与 T-0178 ParamModel 同模式;当前部署架构未规模化,无需立即解决
- **下次复盘**:app 横扩立项时(可能 T-0165 OUI+SN 切换后)

### R-NEW-T0180-06 三皮肤 alarm-library / kpi-library 一致性漂移
- **描述**:本次只改造 webcode 主皮肤的 `pages/product/kpi-library/*`;webcode-v2 / v3 皮肤的同名页面(若存在)未做 drill-down 改造,UX 与主皮肤不一致
- **等级**:P2
- **概率**:中(取决于 v2/v3 是否暴露 kpi-library 入口)
- **影响**:多皮肤场景用户体验割裂
- **Owner**:前端专家 + PgM
- **状态**:**Open**
- **关联 Task**:T-0180(与 T-0179 R-NEW-T0179-05 同源)
- **缓解**:① 验证 v2/v3 实际是否有 kpi-library 页面入口(本次 typecheck 仅证 frontend-core 业务层改动兼容,UI 壳未触及);② 由 PgM 决策是否要镜像改造到两个候选皮肤;③ 至少在 `docs/project/frontend-multi-skin-plan-20260422.md` 加一行 T-0180 改造记录
- **下次复盘**:多皮肤策略评审时(与 T-0179 R-NEW-T0179-05 一并讨论)

---

## 复盘节奏

- **每 Sprint 回顾**：更新 Open/Mitigating 状态，检查 Owner 有无变更
- **每月第一个 Sprint**：审视 P0 列表，确保 <14 天已关闭或有明确进展
- **每季度**：深度复盘 P1/P2，决定是否升级或关闭

**当前版本**：v1.6（2026-05-29,T-0180 6 条 R-NEW-T0180-* 风险登记;**3 条 Mitigated + 3 条 Open** — Mitigated: 01(三表重复)/ 03(级联清理 UI 提示)/ 04(保守回滚 + 测试);Open: 02(NULL loaded_from 误删) / 05(多实例横扩 PG advisory lock,与 T-0178/T-0179 同源)/ 06(三皮肤一致性,与 T-0179 同源);累计 R-NEW-T0178-* + R-NEW-T0179-* + R-NEW-T0180-* 共 **20** 条新增风险 **14 Mitigated + 6 Open**)
**v1.5**（2026-05-29，T-0179 6 条 R-NEW-T0179-* 风险登记;**4 条 Mitigated + 2 条 Open** — Mitigated: 01(XSS) / 03(loaded_from NULL) / 04(Drawer lock) ;Open: 02(规模化 P99) / 05(三皮肤一致性) / 06(ne-types 鉴权)；累计 R-NEW-T0178-* + R-NEW-T0179-* 共 14 条新增风险 11 Mitigated + 3 Open）
**v1.4**（2026-05-29，T-0178 8 条 R-NEW-T0178-* 风险登记;R-NEW-T0178-3 Prometheus 告警 + R-NEW-T0178-8 Loader 启动期 WARN 兑现;**7 条 Mitigated + 1 条 Open**（仅 R-NEW-T0178-6 多实例横扩 PG advisory lock 仍 Open,横扩立项时关闭））
