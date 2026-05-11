# Risk Register（风险登记册）

> **性质**：活文档。每 Sprint 回顾时更新；每季度深度复盘。  
> **守护人**：项目经理（`CLAUDE.md §16.11`）  
> **初始化依据**：`docs/implementation-completeness-report-20260420.md` 五大 P0 + 次级短板

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
- **状态**：**Mitigating**（2026-05-11 — T-0030 Phase 1 完成 commit `92a3eb76`，Phase 2 留 sprint-11+ 后可降到 Closed）
- **关联 Task**：T-0030（F10 互操作用例库扩充 — Phase 1 done 2026-05-11，Phase 2 待启）
- **缓解路径**：
  - ✅ Phase 1（T-0030 done 2026-05-11）：用例 14→30（DM 2→6 / Protocol 3→7 / RPC 9→13 / Inform 0→4 新建）+ ExpectedOutcome 翻转机制（4 个 negative path：每类 ≥ 1）+ e2e 断言 3→6 + 10 新单测
  - Phase 2（sprint-11+ 可选）：运营商私有用例（X_CMCC_* / X_CT-COM_* / X_CU-COM_*）+ 设备型号矩阵参数化 + 验收报告 export
  - 关闭门槛（→ Closed）：用例 ≥ 50 + 4 category 全覆盖（已 ✅）+ 各 category ≥ 2 个 negative path（当前每类仅 1，差 1）+ e2e ≥ 10 claim（当前 6，差 4）
- **关联 PRD**：`docs/project/prd/F10-interop-testing-coverage.md`
- **下次复盘**：sprint-11 plan 时（2026-05-26）评估是否启动 Phase 2

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

---

## 关闭的风险

| ID | 关闭日期 | 关闭摘要 |
|----|---------|---------|
| R-005 | 2026-04-20 | 数据库迁移版本号跳跃。补 5 个 `reserved_placeholder` 占位迁移，`check-migrations.sh` 检查通过。衍生风险 R-108（个别 Down 段缺失）已登记 |

---

## 复盘节奏

- **每 Sprint 回顾**：更新 Open/Mitigating 状态，检查 Owner 有无变更
- **每月第一个 Sprint**：审视 P0 列表，确保 <14 天已关闭或有明确进展
- **每季度**：深度复盘 P1/P2，决定是否升级或关闭

**当前版本**：v1.1（2026-04-20，全部 Open 风险补"关联 Task"字段，双向链通 Backlog）
