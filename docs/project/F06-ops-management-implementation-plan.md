# F06 运维管理 — 完整推进计划

> **来源 PRD**：`docs/project/prd/F06-ops-management.md`（847 行）
> **MVP 起点**：commit `6e42a664`（菜单 + 权限 + 5 页打通 50%）
> **作者**：Claude（代 Owner = PM + 架构 + Go + 前端）
> **创建**：2026-05-10
> **状态**：草案（待 PM/PgM 拍板 §1 决策项后启动）

---

## 0. 概览

| 维度 | 说明 |
|------|------|
| **未完成工作量** | ~38 净工作日（PRD §12 估 45 天 − 已投入 ~1 天 MVP）|
| **拆分粒度** | 12 个 T-xxxx → 52 个 sub-task（每个 ≤ 3 天 / Est=M 内）|
| **Sprint 编排** | 8 个 2 周 Sprint × 8 工作日（80% 容量）= **~16 周** |
| **并行可行性** | 单后端 + 单前端 + 0.5 DevOps + 0.5 QA = 主流路径；纯单人需 ~22 周 |
| **关键里程碑** | W1 MVP 完整可用（Sprint 3） / W3 诊断+下载（Sprint 5） / W4 治理（Sprint 6） / GA Ready（Sprint 8） |
| **阻塞前置** | 6 项 Q 决策（§1）必须先拍板 |

---

## 1. 决策前置（Q1-Q6 不解决就不能动 T-xxxx）

| Q# | 议题 | 影响阻塞 | 建议默认 | Owner | 期望决议日 |
|----|------|--------|---------|-------|-----------|
| Q1 | MML 与运维命令融合 | T-0102 SSE 通道架构 | B 双入口共存 | PM + 架构 | Sprint 1 D1 |
| Q2 | 任务调度引擎放哪 | T-0101 全部 sub-task | A 复用 internal/task + 上层 orchestrator | 架构 | Sprint 1 D1 |
| Q3 | 4 眼审批首版必须 | T-0106 是否首版 / ops_tasks schema | A 必须（合规阻塞） | PM + 合规 | Sprint 1 D2 |
| Q4 | 诊断结果存哪 | T-0104 表设计 | A 独立 ops_diagnostics 表 | 数据 | Sprint 1 D3 |
| Q5 | 模板 steps 是否引入 JSON Schema 严格校验 | T-0103-c 实现 | B 文档约定（首版）/ V2 严格 | 架构 | Sprint 1 D3 |
| Q6 | 跨 OMC 实例共享模板 | T-0103-b 导入导出语义 | B 不支持（首版） | PM | Sprint 1 D3 |

**Q 决议会议**：Sprint 1 第 1 周内开 1 次（90 分钟）一次性拍板，会议纪要回写 PRD §13 + backlog Risk register。

---

## 2. WBS 任务分解（52 sub-tasks）

每行格式：`<编号> <标题> <估时-day> <依赖>`；后端 = BE，前端 = FE，数据库 = DB，配置 = OPS。

### 2.1 Foundation 层（Sprint 1，~5 天）

| ID | 标题 | 估 | 类 | 依赖 |
|----|------|---|----|------|
| **F-001** | 6 张新表 migrations（拆 2 笔，避免单文件超大）<br>　 — migrations/000080_ops_extensions.sql（ops_templates+ops_tasks 扩展列）<br>　 — migrations/000081_ops_new_tables.sql（ops_task_executions / ops_diagnostics / ops_downloads / ops_audit_logs / ops_maintenance_windows / ops_playbooks）| 2 | DB | Q3,Q4 |
| **F-002** | 8 个新权限点 seed（ops:command:safe/cautious/dangerous + task:approve + diagnostic:run + download:trigger + maintenance:plan/approve + audit:view + break_glass）+ role_menus 重排（admin/operator/viewer 默认绑定调整）| 1 | DB | Q3 |
| **F-003** | `Carrier.OpsAdapter` 接口骨架 + 3 个空实现（cmcc/ctcc/cucc，仅 default 行为）| 1 | BE | — |
| **F-004** | 6 个内置 OEM 模板 JSON seed（GPS 失锁恢复 / 时间同步 / 计划重启 / 设备隔离 / 现场协助包 / 软重启），含变量化占位符 `${device_sn}` | 1 | DB | F-001 |

### 2.2 W1 P0 — 命令通道 + 调度引擎（Sprint 2-3，~10 天）

#### T-0101 TaskExecutor 调度引擎（5 天）

| ID | 标题 | 估 | 依赖 |
|----|------|---|------|
| 0101-a | TaskExecutor 接口设计（small interface：Run / Pause / Resume / Cancel / Rollback）+ DI 注入 | 0.5 | F-001, Q2 |
| 0101-b | 步骤路由器：step.type → rpc/mml/wait/loop/branch dispatcher | 0.5 | F-003 |
| 0101-c | 并发控制（max=20）+ 节流（与 internal/task 队列对接）| 0.5 | — |
| 0101-d | 状态机集成（pending → approved → running → success/failed/cancelled/paused，含 approval 等待）| 0.5 | Q3 |
| 0101-e | 结果聚合 + ops_task_executions 写入（每 device × step 一行）| 0.5 | F-001 |
| 0101-f | 失败策略实现（abort / continue / retry-N / rollback）| 1 | 0101-e |
| 0101-g | 暂停/恢复/取消的 dispatcher 集成（不打断已发出的 RPC，只阻止后续）| 0.5 | 0101-d |
| 0101-h | 回滚引擎（执行 rollback_steps 反向序列，rollback 失败发告警）| 0.5 | 0101-f |
| 0101-i | 断点续传（重启 / 节点切换后从 ops_task_executions 状态恢复进度）| 0.5 | 0101-e |

#### T-0102 即时命令 SSE 通道（3 天）

| ID | 标题 | 估 | 依赖 |
|----|------|---|------|
| 0102-a | SSE Hub + 路由（每用户连接，按 task_id 订阅）| 0.5 | — |
| 0102-b | command-execute endpoint（POST /api/v1/ops/commands/rpc → 入队 → 返 task_id）| 0.5 | 0101-* |
| 0102-c | RPC 命令支持：reboot / factory_reset / get_param / set_param / get_rpc_methods | 1 | 0102-b |
| 0102-d | MML 流式输出（多帧）通过 SSE 推送 | 0.5 | 0102-c |
| 0102-e | 超时 + 重试 + per-device 限流 | 0.5 | — |

#### T-0103 前端补齐（2 天，后端落地后续）

| ID | 标题 | 估 | 依赖 |
|----|------|---|------|
| 0103-a | Templates Edit modal + version + change_log UI | 0.5 | F-001 |
| 0103-b | 模板复制（POST /duplicate） + JSON 导入导出 | 0.5 | F-001 |
| 0103-c | 风险等级 UI（safe/cautious/dangerous Tag + 编辑入口）| 0.25 | F-001 |
| 0103-d | Tasks 设备级流水视图（ops_task_executions 数据可视化）| 0.5 | 0101-e |
| 0103-e | Tasks SSE 实时进度页（进度条 + 设备列表实时切状态）| 0.25 | 0102-a |
| 0103-f | CommandManagement 新建即时命令 UI（按钮 + Modal + SSE 实时输出）| 1 | T-0102 全部 |

### 2.3 W3 — 诊断 + 下载（Sprint 4-5，~10 天）

#### T-0104 诊断子系统（5 天）

| ID | 标题 | 估 | 依赖 |
|----|------|---|------|
| 0104-a | ops_diagnostics 表（在 F-001）+ DiagnosticsService 骨架 | — | F-001 |
| 0104-b | TR-181 IPPingDiagnostics 接入（SPV → 等事件码 8 → GPV 拉结果）| 1 | F-003 |
| 0104-c | TraceRouteDiagnostics 接入 | 0.5 | 0104-b |
| 0104-d | Download/UploadDiagnostics（测带宽）| 1 | 0104-b |
| 0104-e | UDPEchoDiagnostics | 0.5 | 0104-b |
| 0104-f | OMC 侧诊断：reachability / 同站连通性 / GPS 同步聚合 / 邻区一致性 / 告警风暴 | 1.5 | — |
| 0104-g | 诊断结果时序图可视化 + 包导出（PDF/JSON）| 1 | 0104-b..f |
| 0104-h | 厂商扩展 DiagnosticsAdapter 接口（Carrier.MapDiagnosticType）| 0.5 | F-003 |
| 0104-i | 前端 NetworkDiagnosis 接 API（替换占位卡）| 1 | 0104-b..g |

#### T-0105 下载子系统（5 天）

| ID | 标题 | 估 | 依赖 |
|----|------|---|------|
| 0105-a | ops_downloads 表（在 F-001）+ DownloadService 骨架 | — | F-001 |
| 0105-b | 7 类内容采集（config/log/pm/mr/diagnostic_bundle/pcap/gps）— transfer.Upload 触发 | 2 | — |
| 0105-c | 打 zip 包（OMC 并发取多文件 → 合包 → MinIO 存）| 1 | 0105-b |
| 0105-d | MinIO presigned URL（1h TTL）+ 浏览器 Range 请求 | 0.5 | 0105-b |
| 0105-e | 过期清理 cron（默认 90 天归档冷存 / 删除可配）| 0.5 | 0105-a |
| 0105-f | 断点续传（transfer 失败重传）| 0.5 | 0105-b |
| 0105-g | 前端 Downloads 接 API（替换占位卡）+ 下载篮 | 1 | 0105-b..d |

### 2.4 W4 — 治理（Sprint 6，~9 天）

#### T-0109 审计归档（2 天，建议先做）

| ID | 标题 | 估 | 依赖 |
|----|------|---|------|
| 0109-a | ops_audit_logs 表（在 F-001）| — | F-001 |
| 0109-b | 全量 hook：所有 op_type（template_run / command_run / task_create / cancel / diagnostic / download / approval）写日志 | 1 | F-001 |
| 0109-c | 12 月保留 cron（超期归档到 MinIO，按月 JSONL.gz）| 0.5 | 0109-b |
| 0109-d | 前端审计查询页（多维过滤 + CSV/JSON 导出）| 0.5 | 0109-b |

#### T-0106 4 眼审批流（4 天）

| ID | 标题 | 估 | 依赖 |
|----|------|---|------|
| 0106-a | ops_tasks 扩展列（approval_state / approver_user_id / approved_at）— 在 F-001 | — | F-001 |
| 0106-b | ApprovalService（4 眼校验：approver ≠ creator） + risk_level 自动判定 | 1 | F-001 |
| 0106-c | 审批通知接入（notification 中心，邮件 + UI 弹窗）| 1 | — |
| 0106-d | 前端审批页（列表 / 详情 / 同意 / 拒绝 + Reason）| 1 | 0106-b |
| 0106-e | 任务创建侧自动 risk 评估 + UI 提示（"此任务需审批"）| 1 | 0106-b |

#### T-0107 维护窗口（3 天）

| ID | 标题 | 估 | 依赖 |
|----|------|---|------|
| 0107-a | ops_maintenance_windows 表（在 F-001）+ Service | — | F-001 |
| 0107-b | 事件机制：start/end 发 maintenance.window.* 事件；alarm/provision/ops 订阅 | 1 | 0107-a |
| 0107-c | 告警抑制集成（alarm 模块订阅 → suppressed=true）| 0.5 | 0107-b |
| 0107-d | 暂停自动开站 + 允许高风险动作旁路 | 0.5 | 0107-b |
| 0107-e | 前端 UI（列表 / 创建 / 审批 / active 列表）| 1 | 0107-a |

### 2.5 W5 — 智能化（Sprint 7，~5 天）

#### T-0108 健康巡检（3 天）

| ID | 标题 | 估 | 依赖 |
|----|------|---|------|
| 0108-a | cron 调度器（默认每周一 02:00）+ scope 选择（group / 全网）| 1 | T-0104 |
| 0108-b | 报告生成（PDF + CSV，存 MinIO）| 1 | 0108-a |
| 0108-c | 邮件 / SMS 通知（notification 接入）| 0.5 | — |
| 0108-d | 前端配置 UI（定时设置 / 历史报告下载）| 0.5 | 0108-b |

#### T-0110 知识库 Playbook（2 天）

| ID | 标题 | 估 | 依赖 |
|----|------|---|------|
| 0110-a | ops_playbooks 表（在 F-001）+ Service | — | F-001 |
| 0110-b | 匹配引擎（alarm_pattern → recommended_templates，按 success_rate 排序）| 0.5 | T-0101 |
| 0110-c | 告警详情页"推荐处置"区块 + 一键应用模板 | 1 | 0110-b |
| 0110-d | 知识库管理 UI（PM/SysAdmin 维护）| 0.5 | 0110-a |

### 2.6 W6 — 紧急响应（Sprint 7-8 部分，~3 天）

#### T-0111 break-glass（3 天）

| ID | 标题 | 估 | 依赖 |
|----|------|---|------|
| 0111-a | 临时权限激活流（申请 + 工单号 + 理由 + 30 min 自动失效）| 1 | T-0109 |
| 0111-b | 全程录像（操作命令 + 输出按时间戳进 ops_audit_logs.break_glass=true）| 0.5 | T-0109 |
| 0111-c | 强制告警通知给安全负责人 | 0.5 | — |
| 0111-d | 24h 复盘评注 + 未评注阻塞下次激活 | 1 | 0111-b |

### 2.7 收尾（Sprint 8）

| ID | 标题 | 估 |
|----|------|---|
| **C-001** | 运营商差异适配实现（cmcc/ctcc/cucc OpsAdapter 完整方法）| 1 |
| **C-002** | E2E 全量回归（scripts/e2e_verify.sh ops/* 补 ~30 个 check_status）| 1 |
| **C-003** | 性能回归（5K 设备并发 batch 命令、批量诊断、压测）| 1 |
| **C-004** | 文档与 SOP（PRD 状态更新 / DoD 勾选 / Release Gate 检核 / 用户手册）| 1 |
| **C-005** | 风险 / Risk Register 清算（关闭关联 R-xxx）| 0.5 |

---

## 3. 依赖图（DAG，关键路径）

```
              ┌──────[Q1/Q2/Q3 决议]──────┐
              ↓                             ↓
      F-001 / F-002 / F-003 / F-004 (Foundation)
              │
              ├──→ T-0101 调度引擎 ──┬──→ T-0103 前端补齐
              │                       │
              ├──→ T-0102 SSE 通道 ──┘   ↓
              │                          (W1 MVP 完整 ✅)
              │
              ├──→ T-0104 诊断 ───────────→ T-0108 健康巡检
              │     (W3 P1)                  (W5 P2)
              │
              ├──→ T-0105 下载
              │     (W3 P1)
              │
              ├──→ T-0109 审计 ─────────┬──→ T-0111 break-glass
              │     (W4 P2)             │      (W6 P3)
              │                          │
              ├──→ T-0106 审批流 ───────┘
              │     (W4 P1)
              │
              ├──→ T-0107 维护窗口
              │     (W4 P2)
              │
              └──→ T-0110 知识库
                    (W5 P3)
```

**关键路径**（最长链）：Q 决议 → F-001 → T-0101 → T-0103 → T-0104 → T-0108 = ~14 工作日（约 3 Sprint）。

---

## 4. Sprint 编排（8 个 2 周 Sprint）

| Sprint | 目标 | 核心交付 | 验收里程碑 |
|--------|------|---------|-----------|
| **S1** | 决策 + 地基 | Q1-Q6 拍板会 / F-001..004 全部完成 / 6 内置模板可见 | Q 议项归档 / Foundation 表全部就绪 |
| **S2** | 调度引擎 | T-0101-a..i 全部完成 / 单元测试 ≥ 70% | TaskExecutor 单元 + 集成测试绿；空模板可被真正执行 |
| **S3** | 命令通道 + 前端 W1 完整 | T-0102 全部 + T-0103-a..f 全部 / 6 内置模板可点"执行" | **W1 MVP 完整**：5 页"真能用"，L1 一线可用 |
| **S4** | 诊断（上） | T-0104-a..e（TR-181 五对象接入） | IPPing/TraceRoute/Throughput 可用 |
| **S5** | 诊断（下） + 下载 | T-0104-f..i + T-0105-a..g | **W3 完整**：诊断 / 下载页"真能用" |
| **S6** | 治理 | T-0109 + T-0106 + T-0107 | **W4 完整**：审计 12 月保留 / 4 眼审批 / 维护窗口生效 |
| **S7** | 智能化 + 紧急 | T-0108 + T-0110 + T-0111-a..b | **W5-W6 主体**：巡检报告 + 知识库 + break-glass 基础 |
| **S8** | 收尾 | C-001..005 / T-0111-c..d / E2E + 压测 + 文档 | **GA Ready**：DoD 全勾 + Release Gate 9 章过 |

**Sprint 容量假设**：单 Sprint = 2 周 × 5 工作日 × 80% = 8 工作日 / 人；2-3 人并行 = 16-24 工作日。

---

## 5. 数据迁移计划

| migration | 内容 | Sprint |
|-----------|------|--------|
| `000080_ops_extensions.sql` | ops_templates +6 列 / ops_tasks +6 列（risk_level / version / change_log / owner_user_id / rollback_steps / target_carriers / template_snapshot / approval_state / approver_user_id / approved_at / batch_config / failure_policy）| S1 |
| `000081_ops_new_tables.sql` | 6 张新表：ops_task_executions / ops_diagnostics / ops_downloads / ops_audit_logs / ops_maintenance_windows / ops_playbooks（含索引 + 触发器）| S1 |
| `seed/000082_seed_ops_permission_points.sql` | 8 个新权限点 menus + role_menus 调整 + role_api_permissions 扩展 | S1 |
| `seed/000083_seed_ops_builtin_templates.sql` | 6 个内置 OEM 模板 JSON（GPS 失锁 / 时间同步 / 计划重启 / 设备隔离 / 现场协助包 / 软重启）| S1 |

> 编号顺延 origin 当前最大，会议日实际确认。所有迁移**幂等**（ON CONFLICT DO NOTHING），完整 Up/Down。

---

## 6. 风险登记

| Risk ID | 描述 | 严重性 | 缓解 |
|---------|------|-------|------|
| R-O01 | 任务调度引擎设计争议 | **高** | Q2 必须 S1 第 1 周拍板；架构专家与 Go 主力专题对齐 1.5 天 |
| R-O02 | SSE 在 nginx / k8s ingress 下连接保持问题 | 中 | S3 提前做 SSE 长连接专项测试（k8s ingress timeout / nginx buffer 设置）|
| R-O03 | 6 个内置模板的步骤正确性 | 中 | 邀请运营商运维专家 review，分批落地（先 2 个高频，2 周内灰度）|
| R-O04 | TR-181 Diagnostics 不同厂商兼容性 | 中 | Carrier OpsAdapter 抽象前置；S4 第 1 周做厂商兼容性矩阵 |
| R-O05 | 4 眼审批改 ops_tasks 状态机契约 | 中 | S2 ops_tasks 扩展列时同步引入 approval_state（默认 not_required）保持兼容 |
| R-O06 | 维护窗口跨域订阅 alarm / provision | 低 | EventBus 既有，沿用；不要建专用 channel |
| R-O07 | break-glass 滥用 | 中 | S7 内置告警 + 24h 复盘强制 + 季度审计抽查 |
| R-O08 | 38 天估算偏乐观 | 中 | S5 末次回顾必须重新计 burn-down，超时则砍 W5/W6 进 V2 |

---

## 7. 质量门（DoD 每 sub-task 必过）

- [ ] `go build ./...` 绿 + `go test -race -count=1 ./internal/ops/... -cover` ≥ 70%
- [ ] 前端 `npx tsc --noEmit` 绿 + `npm run lint` 绿
- [ ] 迁移 up/down 双向跑通；`bash scripts/check-migrations.sh` 不引入新冲突
- [ ] 新端点：每个有 ≥ 1 个 E2E `check_status` 断言
- [ ] 新 metric / log：`grep -rn <name>` 命中 ≥ 1
- [ ] 所有写操作进 `ops_audit_logs`（T-0109 落地后强制）
- [ ] 高风险动作（reboot/factory_reset/批量 > 50 台）走 4 眼审批（T-0106 落地后强制）
- [ ] 运营商差异通过 `OpsAdapter`，禁 `if carrier == "..."` 硬编码
- [ ] 文档：PRD 对应章节状态更新（in_dev / done）

---

## 8. 资源估算

| 角色 | 人月 | 主要工作 |
|------|------|---------|
| **Go 后端（主力）** | 2 | T-0101 / 0102 / 0104 / 0105 / 0106 / 0109 主路径 |
| **Go 后端（backup）** | 0.5 | 维护窗口 / 巡检 / break-glass / 测试 |
| **前端 React** | 1.5 | T-0103 / 0104-i / 0105-g / 0106-d / 0107-e / 0108-d / 审计查询 UI |
| **DevOps** | 0.5 | SSE / cron / MinIO 归档 / k8s 配置 / 监控 |
| **QA** | 0.5 | E2E 用例补齐 / 压测 / 回归 |
| **PM / 架构 / 合规** | 0.3 | Q 决议 / risk-register / 验收 |
| **合计** | **~5.3 人月** | 对应 ~16 周 × 2-3 人协同 |

---

## 9. 关键里程碑

| 里程碑 | 时间 | 交付物 | 价值 |
|-------|------|--------|------|
| **M1 决议封顶** | S1 W1 | Q1-Q6 全部拍板，risk-register 更新 | 解锁所有后续 |
| **M2 Foundation Ready** | S1 W2 | 8 张表 + 8 权限点 + Adapter 接口 + 内置模板 | 后端开干 |
| **M3 W1 MVP 完整** | S3 末 | 5 页"真能用"：Templates 全 CRUD / Tasks 真执行 / Commands 即时通道 | **首次可向用户演示** |
| **M4 W3 完整** | S5 末 | 诊断 + 下载真能用 | **故障定位能力补齐** |
| **M5 W4 治理 Ready** | S6 末 | 审计 / 审批 / 维护窗口 | **等保合规阻塞解除** |
| **M6 GA Ready** | S8 末 | 全部 12 个 T-xxxx 完成 + E2E 全绿 + 压测过 + 文档完整 | **可对接外部客户** |

---

## 10. 启动 Checklist（每个 Sprint 开始前）

- [ ] backlog 中所有 sub-task 已登记，状态 `triaged → planned`，回填 Sprint + Owner
- [ ] 前置依赖（Q 决议 / 上游 sub-task done）全部满足
- [ ] Sprint 容量计算清楚（人 × 天 × 0.8）
- [ ] 风险与缓解方案已对齐
- [ ] DoD 清单准备好

---

## 11. 与 dev-pipeline 的对齐

- 每个 sub-task 走 `/dev-pipeline pick T-0xxx-y` 完整 S0–S7
- 简单子任务（如 schema 扩列 / seed）走 `--fast bugfix` 跳 S0/S1/S2
- 跨域改动（如 T-0107 维护窗口同时改 alarm + provision + ops）走"联合变更"PR
- 每周一 PM/PgM Triage 检视 burn-down 与 Sprint 实际进度

---

## 12. 关联文档

- PRD：`docs/project/prd/F06-ops-management.md`
- MVP 起点 commit：`6e42a664`
- 邻接 PRD：F06-license-management / F06-mml（待立项）
- DoD：`docs/project/dod.md`
- Release Gate：`docs/project/release-gate.md`
- 风险登记：`docs/project/risk-register.md`
- backlog：`docs/project/backlog.md`（T-0101..T-0111 待登记）

---

## 变更日志

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1 | 2026-05-10 | Claude | 初稿：8 Sprint × 52 sub-task × ~38 工作日，含决议前置 + Foundation + 6 阶段路线图 |
