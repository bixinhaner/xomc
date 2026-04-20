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
- **缓解**：PRD `docs/project/prd/F04-alarm-notification.md` 已产出；进 Sprint 1-2 实现邮件+Webhook，Sprint 3 补短信
- **下次复盘**：Sprint-01 回顾（~2026-05-04）

### R-002 E2E 用例实现为 0
- **描述**：`scripts/e2e_verify.sh` 框架 4459 行但实际用例数为 0（grep `claim` = 0），声称的 452 用例完全未落地
- **等级**：P0
- **概率**：高（已发生）
- **影响**：无法回归；任何合入都可能引入未发现的破坏
- **Owner**：QA/发布经理
- **状态**：Open
- **缓解**：CI 中加入 E2E 用例数阈值（初始 20，每 Sprint +20 到 200+）；Sprint 1 先补核心流程（登录/设备/告警/PM/Inform）
- **下次复盘**：每 Sprint 回顾

### R-003 F08 北向 OSS 协议栈缺失
- **描述**：仅实现 REST JSON 导出，无 SNMP/CORBA/MTOSI/TMF 标准协议适配
- **等级**：P0（对接运营商 OSS 时）
- **概率**：中（取决于首个商用运营商的接入要求）
- **影响**：不满足运营商 OSS 接入规范，可能阻塞商用
- **Owner**：PM（需求矩阵）+ 架构师
- **状态**：Open
- **缓解**：PM 产出运营商需求矩阵 → 优先实现刚需协议（通常 SNMP Trap）→ 6-10 周独立 track
- **下次复盘**：Sprint-02 回顾

### R-004 事件总线仅 SSE，无分布式协调
- **描述**：`internal/events/` 仅实现 SSE Hub + 持久化，没有 NATS JetStream 主题订阅，单进程事件总线限制横向扩展
- **等级**：P0（扩展性）
- **概率**：中
- **影响**：10 万→100 万扩展时，模块间事件解耦无法实现
- **Owner**：架构师 + 运维
- **状态**：Open
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

## P1 风险（质量影响）

### R-101 Software 模块缺灰度升级策略
- **描述**：`internal/software/` 批量升级仅支持全量并发控制，无分组/延时/百分比策略
- **等级**：P1
- **概率**：中
- **影响**：大规模升级无法分批验证，失败面影响全量
- **Owner**：电信业务专家
- **状态**：Open
- **缓解**：Sprint 4-5 补灰度策略
- **下次复盘**：Sprint-03

### R-102 前端 Backup/Software 页面仅骨架
- **描述**：Backup 48%、Software 52% 完成度，业务逻辑缺失（任务创建/升级进度/回滚）
- **等级**：P1
- **概率**：高
- **影响**：运维场景不可用
- **Owner**：前端专家
- **状态**：Open
- **缓解**：Sprint 3-4 补业务逻辑
- **下次复盘**：Sprint-02

### R-103 License 容量/过期未拦截
- **描述**：`MaxDevices`/`ExpiryDate` 字段有，但无超限拦截与自动禁用
- **等级**：P1
- **概率**：中
- **影响**：违反商业约束，不符合授权要求
- **Owner**：电信业务专家
- **状态**：Open
- **缓解**：Sprint 3 补拦截逻辑
- **下次复盘**：Sprint-03

### R-104 拓扑自动分组规则引擎未激活
- **描述**：`rule_service` 注释为 TODO，rule_matcher 存在但未接入
- **等级**：P1
- **概率**：低
- **影响**：分组依赖手工维护，规模上不去
- **Owner**：架构师
- **状态**：Open
- **缓解**：Sprint 5 激活
- **下次复盘**：Sprint-04

### R-105 syslog 远程转发缺失
- **描述**：仅查询，无 UDP/TCP syslog 转发
- **等级**：P1
- **概率**：低
- **影响**：与统一日志平台集成困难
- **Owner**：运维与可观测性专家
- **状态**：Open
- **缓解**：按需实现
- **下次复盘**：Sprint-05

### R-106 worker 进程无重试/死信队列
- **描述**：EventBus 订阅失败无重试，长运行稳定性未验证
- **等级**：P1
- **概率**：中
- **影响**：PM/MR 文件处理失败会直接丢弃
- **Owner**：架构师 + 运维
- **状态**：Open
- **缓解**：Sprint 2 与 NATS 改造（R-004）同步
- **下次复盘**：Sprint-02

### R-107 Prometheus/Grafana/AlertManager 容器编排缺失
- **描述**：metrics/tracing 已集成但 docker-compose 无监控栈
- **等级**：P1
- **概率**：高
- **影响**：Release Gate 中"可观测性就绪"无法自动化验证
- **Owner**：运维与可观测性专家
- **状态**：Open
- **缓解**：Sprint 1 补 compose 文件 + 基础 dashboard
- **下次复盘**：Sprint-01

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

## P2 风险（优化项）

### R-201 RF 控制硬编码 LTE
- **描述**：device 模块 RF 开关未走 Carrier 适配
- **等级**：P2
- **Owner**：电信业务专家
- **状态**：Open

### R-202 F10 互操作用例库稀疏
- **描述**：仅 RPC/Protocol/DataModel 三类
- **等级**：P2
- **Owner**：测试专家

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

**当前版本**：v1.0（2026-04-20，初始化）
