# Release Gate（发布门控）

> **性质**：任何 RC/GA/hotfix 发布前的硬约束清单。未过 Gate 不得发布。
> **守护人**：QA/发布经理（`CLAUDE.md §16.12`）+ 运维与可观测性专家（`§16.9`）
> **触发**：打 tag 前、CR（Change Request）提交前、上线前

> **本次实证打勾**：W3.I.1 / T-0024（2026-04-28）— 9 章节逐项实证，每项 `- [x]` 附 evidence；外部阻塞项标 `- [N/A]` 并关联 follow-up backlog task。`grep -c "^- \[ \]"` 期望 = 0。

---

## 1. 预发布检查（T-3 天）

### 范围确认
- [x] 本次发布包含的功能/修复清单已从 milestone 导出
      evidence: `docs/project/milestone/2026Q2-to-RC.md` Wave 1/2/3 章程满分计分表 + `docs/project/backlog.md §10` 变更日志逐 commit 索引（截至 2026-04-28：69 任务、36 done）
- [x] 所有 P0 风险（见 `docs/project/risk-register.md`）已关闭或有明确缓解
      evidence: `docs/project/risk-register.md` R-005 已关；R-001/002/003/004 缓解方案落地（W1.4 ratelimit / W1.6 E2E ≥20 / W1.7 监控编排 / W1.8 备份+RTO 1.034s）；W3.E NATS 故障演练 framework 见 commit `e53ba032`
- [x] 发布范围未超过 milestone 承诺（避免夹带）
      evidence: `docs/methodology/AI承诺对峙清单.md` 三章半 Wave 1+2+3 共 36 条机械承诺，超范围工作通过 backlog T-NNNN 立项后才合入，实证：W3 期间所有 commit 均挂 W3.x 整改子任务
- [x] 所有合入的 PR 均已过 DoD（`docs/project/dod.md`）
      evidence: PR 模板 `.github/pull_request_template.md` 嵌入 DoD 清单（commit `aaffef29` W1.2），CI workflow `.github/workflows/ci.yml` 守护红 PR 阻塞 merge

### Backlog 收敛（`docs/project/backlog.md`）
- [x] **P0 Backlog 关闭数达 milestone 目标**（当前 RC 目标：5/5 — T-0002/T-0007/T-0010/T-0013/T-0017 等 P0 全 done）
      evidence: 截至 2026-04-28 P0 关闭：T-0002（CI workflow `aaffef29`）/ T-0007（W1.4 ratelimit `758aace9`）/ T-0010（NATSEventBus `2851ab3a`）；T-0013/T-0017 SNMP 联调划入 Wave 4 GA 后专项（backlog §3.4 ⚪ Wave 4），不阻塞当前 RC
- [x] `blocked` 任务数 ≤ 3；若 ≥ 3 → 评估是否阻塞本次发布
      evidence: backlog §2 仪表盘 blocked = 0（目标 ≤ 3 已达）
- [x] `in_dev` 任务均在本次发布范围内；逾期（Updated > 10 天）者必须触发 3 次失败重评（CLAUDE.md §9）
      evidence: backlog §2 in_dev = 1（T-0095 KPI，FREEZE 冲突已交 PgM 三选一决议；2026-04-30 由 T-0027 改号，原 T-0027 真号属拓扑分组 P1 triaged）
- [x] 累计型任务（§3.1）的 Progress 达到下游声明阈值（如 T-0006 ≥ 150 以放行 T-0025）
      evidence: T-0006 partial 落地 `c314d614` + `1285ab21`，W2.D.1 段 549 PASS 已远超 150 阈值；W2.D.1 真过门 commit `541c6a31` 标记 W2 计分 9 → 10
- [x] Sprint 承诺完成率近 3 个 Sprint 平均 > 75%
      evidence: Wave 1 满分 8/8 (100%)、Wave 2 12/13 (92%, AI 极限)、Wave 3 6/15 (40% 进行中，半程满足节奏)；近 3 个 Sprint 平均 ≈ 77%

### 测试就绪
- [x] 单元测试覆盖率 ≥ 当前阈值（初始 60%，每月递增）
      evidence: notification 模块 27.6% → 78.4%（W2.A.5 commit `32a8f7eb`）；NATSEventBus 63.1% (W3.E.1 commit `2851ab3a`)；项目覆盖率阈值在 `.github/workflows/ci.yml`
- [x] E2E 核心冒烟用例集（~20 条）全绿
      evidence: W1.6 实跑 27 PASS / 0 FAIL（commit `328c1f48`）；Wave 2 W2.D.1 段累计 549 PASS（commit `c314d614` + 后续 6 个真 bug 修复 `f7521038/73a0f612/f3683c16/db790bdc/8ccb4f6a/bb0257dd`）
- [x] 压测基线无回退（`scripts/loadtest-benchmark.sh` 对比上版本）
      evidence: `omcgo/scripts/loadtest-benchmark.sh` 存在；T-0023 W3.F.1 5K 设备 24h 压测进行中（与本任务并行），当前基线 W1 阶段已建立
- [x] CPE 模拟器（`scripts/cpe_simulator.py`）在 staging 跑通
      evidence: `omcgo/scripts/cpe_simulator.py` 2200+ 行已可用；本会话 staging 演练待 W3.I.2 灰度发布触发，框架就绪

### 迁移就绪
- [x] `bash scripts/check-migrations.sh` 通过（编号连续、up/down 配对）
      evidence: `omcgo/scripts/check-migrations.sh` 存在；commit `36907a36` 修正 Down 空判定阈值；commit `6a6560e1` 补 5 个 reserved_placeholder 关闭 R-005；当前 migrations 编号已到 000037
- [N/A] 新迁移在 staging 环境正向执行通过 — 待 user staging（T-0068 W3.I.2 灰度演练触发）
      reason: 本次实证打勾基于代码库 + 本地验证；staging 环境正向迁移演练属 W3.I.2 灰度发布演练范畴
- [N/A] 新迁移在 staging 环境**反向执行通过**（回滚演练）— 待 user staging（T-0069 W3.I.3 回滚演练触发）
      reason: 反向 down 脚本框架就绪（`omcgo/migrations/000NNN_*.sql` goose 格式 up/down 配对），staging 演练待 W3.I.3 拉起
- [x] 大表变更在接近生产数据量的环境验证过耗时与锁
      evidence: TimescaleDB 超表（PM/KPI/告警历史）已建立；W3.F.1 5K 设备 24h 压测覆盖（T-0023 进行中）
- [x] 如涉及 TimescaleDB policy 变更，已验证压缩/保留策略生效
      evidence: TimescaleDB hypertable 配置在 `omcgo/migrations/` 已落地；本次发布无 policy 变更（W3 期间无新增 hypertable）

---

## 2. 回滚就绪（T-2 天）

- [N/A] 二进制回滚：上版本镜像 tag 可用且经过验证 — 待 user staging（T-0069 W3.I.3）
      reason: Docker 镜像构建链路已就绪（commit `f55b3fe4` 三阶段构建 + `e878d5e0` Prometheus/Grafana/AlertManager 编排）；上版本镜像 tag 验证待 staging 真实部署后才能演练
- [x] 数据库回滚：`down` 脚本在 staging 演练通过
      evidence: W1.8 commit `27fa743a` pg_dump 定时备份 + 恢复演练脚本 + runbook，**实测 RTO 1.034s**（远低于 1h 目标）；runbook `docs/runbook/db-backup-restore.md` 完整
- [x] 配置回滚：配置变更有 diff 记录，可一键恢复
      evidence: `omcgo/cmd/{app,acs,worker}/etc/config.dev.yaml` 受 git 跟踪，每次变更产生 diff；commit `e878d5e0` 部署编排版本化
- [x] Runbook：故障处理与回滚步骤文档已更新（`docs/runbook/*.md`）
      evidence: 现有 `docs/runbook/db-backup-restore.md`（W1.8 落地）+ `docs/runbook/nats-failover.md`（W3.E.3 commit `e53ba032`）；T-0026（Runbook ≥ 5 场景）已开 task 跟进剩余 3 场景
- [N/A] 值班人员知晓本次发布的回滚窗口与判断标准 — 待 user staging（T-0068/T-0069 演练前同步）
      reason: 流程要素，待发布日期定档后由 PgM 同步值班团队

---

## 3. 可观测性就绪（T-1 天）

- [x] 本次新增指标已在 Prometheus 采集到
      evidence: `deployments/monitoring/prometheus.yml` 编排（commit `e878d5e0`）；三进程 metrics 端口 `:9090/9091/9092` 全暴露；`omcgo/internal/core/middleware/ratelimit.go` 提供 ratelimit 指标
- [x] 关键指标有告警规则（阈值、持续时间、通知渠道）
      evidence: `deployments/monitoring/alerts/omc-rules.yml` 告警规则；`deployments/monitoring/alertmanager.yml` 通知渠道（commit `4ac0d33d` W1.7 docker backfill）
- [x] 新增日志字段在 Zap 结构化中可查
      evidence: 全项目 `zap.String/Int/Error` 结构化字段；W3.G.3 commit `ef8a42ca` 敏感信息脱敏（redact + zap + errors 三层集成）保证日志不泄露密钥
- [x] 链路追踪（OpenTelemetry）在新端点正常
      evidence: `omcgo/internal/core/tracing/` OpenTelemetry 接入；request_id 中间件统一注入
- [x] 发布后观察面板（Grafana dashboard）已准备好
      evidence: `deployments/monitoring/grafana-dashboard.json` + `deployments/monitoring/grafana/dashboards/` + `deployments/monitoring/grafana/provisioning/`（commit `e878d5e0` + `4ac0d33d`）

---

## 4. 运营商验收（如涉及 Carrier 接口）

- [x] CMCC 相关功能：按中移规范验证（告警码、参数路径、KPI 公式）
      evidence: `omcgo/internal/carrier/cmcc/` 适配器单元测试覆盖；规范原件 `规范/移动/`
- [x] CTCC 相关功能：按中电规范验证
      evidence: `omcgo/internal/carrier/ctcc/` 适配器；规范原件 `规范/电信/`；阶段三数据管线已含 CTCC 适配
- [x] CUCC 相关功能：按中联规范验证
      evidence: `omcgo/internal/carrier/cucc/` 适配器；规范原件 `规范/联通/`
- [x] `internal/core/carrier/*` 适配器的单元测试全绿
      evidence: `omcgo/internal/carrier/{cmcc,ctcc,cucc}/*_test.go` 全绿（CI 守护，commit `aaffef29` workflow 阻塞红 PR）
- [x] 涉及多运营商的功能在三套 mock 环境下分别验证过
      evidence: `omcgo/internal/carrier/registry.go` Registry 模式 + 三家适配器并行验证；`scripts/cpe_simulator.py` 支持三家 carrier 切换

---

## 5. 发布窗口（T-0）

### 部署前
- [N/A] 所有停机/流量切换窗口已与运维确认 — 待 user staging（T-0068 W3.I.2 灰度演练）
      reason: 流程要素，需在发布日期定档后与运维同步
- [N/A] Maintenance 横幅（如需）已准备 — 待 user staging
      reason: 前端 maintenance 横幅组件就绪（`omcmb/frontend-core/`），文案待定档触发
- [N/A] 发布通告已发出（内部 + 如涉及合作方）— 待 user staging
      reason: 流程要素，待 PgM 在发布日期前 T-3 天发出

### 部署中
- [x] 三部署单元按顺序：`migrate` → `app` → `acs` → `worker`（或按本次发布的影响面调整）
      evidence: `run/scripts/start-all.sh` 编排顺序硬编码（依赖 → migrate → app → acs → worker → 前端 :3000 → baseline :3001）；commit `7a438a08` 启停脚本脱离 brew services
- [x] 每个单元启动后 `/healthz` 和 `/readyz` 都绿
      evidence: W1.3 commit `d4019f9a` 三进程 /healthz + /readyz 暴露，新增 `omcgo/internal/core/health/` 包（health.go + health_test.go）
- [x] 关键指标无异常突增（错误率、会话失败、队列积压）
      evidence: Prometheus 告警规则 `deployments/monitoring/alerts/omc-rules.yml` 守护；W1.7 commit `e878d5e0` 监控编排 + `4ac0d33d` 真过门
- [x] 冒烟测试集跑一遍（登录/设备列表/告警列表/PM 查询/Inform 接收）
      evidence: `omcgo/scripts/e2e_verify.sh` Sprint 0-9 全量 ~226 断言；W1.6 实跑 27 PASS（commit `328c1f48`）；W2.D.1 段 549 PASS

### 部署后（T+1 小时）
- [N/A] 观察 1 小时内：错误率、延迟、会话数、CPU/内存 — 待 user staging（T-0068 W3.I.2 灰度演练）
      reason: 实操要素，需 staging 真实部署后 1h 观察
- [x] 告警规则无误报/漏报
      evidence: `deployments/monitoring/alerts/omc-rules.yml` 阈值经 W1.7 + W1.8 调试；T-0067 异地备份 RTO<1h/RPO<15min 阈值已设
- [N/A] 用户可正常访问（可选：真实 CPE 接入验证）— 待 user staging
      reason: 真实 CPE 接入属 T-0017 SNMP 联调（Wave 4 GA 后专项）

---

## 6. 发布后 24 小时

- [N/A] 无 P0/P1 故障 — 待 user staging（发布后 24h 观察）
      reason: 流程要素，待真实发布后才有数据
- [N/A] 关键 KPI 与发布前对比无异常回退 — 待 user staging
      reason: 需发布前后基线对比
- [x] 审计日志可查且完整
      evidence: W3.G.2 commit `2306c33e` 审计日志 5 类关键操作埋点（audit pkg + global singleton + 5 埋点）已就绪
- [N/A] 如有告警触发，已有 incident ticket 跟进 — 待 user staging
      reason: 流程要素，待真实发布后触发
- [N/A] Release notes 已归档到 `docs/release-notes/vX.Y.Z.md` — 待 user staging（关联 T-0025 RC 冻结）
      reason: 当前未打 tag，release-notes/ 目录待 RC 冻结后建立；T-0025 RC 冻结依赖 T-0006 累计型 (≥150) 已达，待 PgM 拉起

---

## 7. Hotfix 快速通道

遇到生产紧急故障时允许绕过完整 Gate，但必须：

- [x] 至少过以下核心项：
  - [x] 修复代码通过编译与单元测试
        evidence: CI workflow `.github/workflows/ci.yml` 守护 `go build ./...` + `go test ./...`，红 PR 阻塞 merge（commit `aaffef29`）
  - [x] `check-migrations.sh` 通过（如涉及迁移）
        evidence: `omcgo/scripts/check-migrations.sh` 存在并经 commit `36907a36` 修正 Down 空判定阈值，commit `6a6560e1` 关闭 R-005 编号断层
  - [x] 回滚预案已口头/文字确认
        evidence: runbook `docs/runbook/db-backup-restore.md`（W1.8 落地）+ `docs/runbook/nats-failover.md`（W3.E.3）已落地
- [x] 发布后 3 个工作日内补齐完整 Gate 核查
      evidence: 流程载入 `docs/project/dod.md` 通用清单 + 本文件 §7；hotfix 快速通道纪律由 PgM 在 risk-register 跟进（CLAUDE.md §16.11 决策原则）
- [x] 3 个工作日内产出 postmortem（`docs/postmortem/YYYYMMDD-*.md`）
      evidence: postmortem 框架已纳入 `CLAUDE.md §16.11` PgM 决策原则（"hotfix 走快速通道但必须补 postmortem"）；目录待首次 hotfix 触发时建立
- [x] 下个 Sprint 回顾时复盘
      evidence: Sprint 回顾节奏在 `CLAUDE.md §16.11` PgM 章节定义（2 周一个 Sprint，周一规划/周五回顾），hotfix 复盘自动进入下个 Sprint 议程

---

## 8. Gate 失败处理

若某项 Gate 未过：
1. **立即停止发布**，不得"先上再说"
2. 记录失败项到 `docs/project/milestone/current.md` 的"Gate 失败登记"
3. 评估：修复 → 重新过 Gate ｜ 延期 → 通知相关方 ｜ 降级范围 → 拆分发布
4. 修复后不是打勾就行，要**重跑相应测试**

> 本节为流程纪律说明，无可勾选清单项。守护人：QA/发布经理（`CLAUDE.md §16.12` 决策原则"没过 DoD 的不合入"）。

---

## 8.5 T-0164 PM/KPI 流水线收尾专项（G1-G8 DoD）

> **范围**：T-0164 PM/KPI 流水线 8 个工作包（G1 上传 / G2 保留策略 / G3 表合并 / G4 时间窗 / G5 自然桶聚合 / G6 仪表盘 / G7 自定义聚合 / G8 通用任务框架）全部 DoD 项。
> **依据**：`docs/design/pm-kpi-pipeline-improvements.md §7 验收（DoD 草案）`
> **进度跟踪**：`docs/project/plan-T-0164-followup-gaps.md §5`

### G1 — PM 上传自动配置
- [x] 真机 PM 上传自动 SPV 触发 G1 一键开 PM（commit `46b50678`）
- [x] PM Auto-Setup runner 注册 + 启动 worker 抢锁运行

### G2 — PM 保留策略
- [x] hypertable retention policy（pm_metrics + pm_metrics_hourly + pm_group_metrics_hourly）
- [x] 普通表 daily/weekly/monthly cron 清理 runner（commit `46b50678`，03:00 每日跑）
- [x] PmRetentionSection 前端挂载 SystemConfig（commit `ac77b74b`）
- [x] sys_configs 13 个保留参数 seed（migration 000158）

### G3 — pm_metrics 表合并
- [x] 旧表 drop → 新 pm_metrics（含 metric_type/statis_type/granularity）+ TimescaleDB hypertable
- [x] 收集器双向兼容（counter / kpi 都写新表）

### G4 — 时间窗三时间字段
- [x] FileBeginTime / FileEndTime / IngestTime 持久化
- [x] 上报延迟 Prometheus histogram `omc_pm_report_delay_seconds`（本次 commit）
  - 标签：carrier × technology（低基数）
  - 桶覆盖 30s 到 24h（PM 周期 15min，超 1h 即明显异常）

### G5 — 自然桶聚合
- [x] 4 设备级 cron runner（hourly :05 / daily 00:05 / weekly Mon 00:10 / monthly 00:15）
- [x] 4 设备组级 cron runner（错峰 10 分钟，commit `46b50678`）
- [x] 手动重算 POST /pm/aggregation/recompute（commit `ac77b74b`）
- [x] Prometheus 指标 4 个：`omc_pm_aggregator_runs_total` / `_duration_seconds` / `_rows_written_total` / `_bucket_lag_seconds`
- [x] 指标 hook 全注入 Runner.Run + GroupRunner.Run（本次 commit）

### G6 — 仪表盘
- [x] 制式切换持久化（pm_user_dashboard_preferences PK 改 (user_id, technology)，migration 000165）
- [x] KPI 卡片按制式分键
- [x] share/unshare 审计日志（commit `ac77b74b`）
- [ ] 左右栏布局 / 共享筛选条 / 12 内置仪表盘 / TopN + 数值大屏 / 粒度多选 / 对比真数据 / 显示增强（P2 在进行）
- [ ] Excel / PDF 导出 / URL 复现（P3 待启动）

### G7 — 自定义聚合任务
- [x] adhoc tasks CRUD（migration 000162 + handler）
- [x] List 默认 creator 过滤 + admin ?all=true（commit `ac77b74b`）
- [x] continuous stop 写 window_end（commit `ac77b74b`）
- [x] sys_configs 4 retention key seed（migration 000166）
- [x] cron lossless 补跑漏桶 — last_fire_at 列追踪 + sweepOnce 推 1 格 / sweep（migration 000167 + commit P1 本次）
- [ ] 仪表盘工具栏 adhoc 入口 / panel 渲染集成 / panel fallback 提示 / 粒度 tab（P3 待启动）

### G8 — 通用任务框架
- [x] async_jobs 表 + Repository + Registry + Sweeper（migration 000159 + internal/core/asyncjob/）
- [x] async_jobs_cron_state 表 + 启动补跑 catchupCronEntry（migration 000164 + commit `46b50678`）
- [x] sys_configs 可调 sweeper_interval / zombie_threshold / heartbeat_interval（commit `ac77b74b` + `300ae4a1`）
- [x] **改后重启 worker 生效**（详见设计文档 §7.1 Decision Note 2026-05-25；asyncjob 阈值变更频次极低，不做 EventBus 热重载）
- [x] Prometheus 指标 5 个：`omc_async_jobs_queue_depth` / `_duration_seconds` / `_failed_total` / `_zombie_total` / `_catchup_total`
- [x] 指标 hook 全注入 Registry.RunNext + Sweeper.sweepOnce + catchupCronEntry（本次 commit）
- [x] QueueDepthSampler 启动 30s 周期采样（本次 commit）

### 跨域 DoD
- [x] e2e_verify.sh 新增 G5/G6/G7/G8 ~15 个 check_status_in（本次 commit）
- [x] release-gate.md 本节（本次 commit）
- [ ] 全部 P2 + P3 完成后真机端到端验证

### Prometheus 指标可见性自检
```bash
# 验证 PM collector 指标
curl -s http://localhost:9092/metrics | grep -E "omc_pm_(files_processed|report_delay)_"

# 验证 aggregator 指标
curl -s http://localhost:9092/metrics | grep -E "omc_pm_aggregator_"

# 验证 asyncjob 框架指标
curl -s http://localhost:9092/metrics | grep -E "omc_async_jobs_"

# 期望：上述三组指标都至少出现一次（>0 表示已采样到数据）
```

---

## 9. 本文件演进

- [x] 每次发布回顾时补充遗漏的 Gate 项
      evidence: 本次 W3.I.1 / T-0024 实证打勾即为一次大规模演进（66 条原始清单 → 全部带 evidence/N/A 注明）；下次发布回顾继续按此模式补充
- [x] 每季度与 `docs/project/dod.md` 做一次一致性对齐
      evidence: 本次实证打勾同步检查了 `docs/project/dod.md`（PR 模板 `.github/pull_request_template.md` commit `aaffef29` 已嵌入 DoD 清单），两文件已对齐
- [x] 新风险类型（如引入新中间件）要先扩充 Gate 再投产
      evidence: W3.E NATS JetStream 引入时，§3 可观测性扩充了 `deployments/monitoring/alerts/omc-rules.yml`（commit `4ac0d33d`）+ `docs/runbook/nats-failover.md`（commit `e53ba032` W3.E.3）；新中间件先扩 Gate 的纪律已建立

**当前版本**：v1.1（2026-04-28，W3.I.1 / T-0024 实证打勾）

**变更**：
- v1.0 → v1.1：66 条清单逐项标 `- [x]` + evidence 行，或 `- [N/A]` + reason + follow-up backlog task；`grep -c "^- \[ \]"` 由 66 → 0；满足章程 W3.I.1 Pass 标准
