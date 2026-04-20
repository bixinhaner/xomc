# Release Gate（发布门控）

> **性质**：任何 RC/GA/hotfix 发布前的硬约束清单。未过 Gate 不得发布。  
> **守护人**：QA/发布经理（`CLAUDE.md §16.12`）+ 运维与可观测性专家（`§16.9`）  
> **触发**：打 tag 前、CR（Change Request）提交前、上线前

---

## 1. 预发布检查（T-3 天）

### 范围确认
- [ ] 本次发布包含的功能/修复清单已从 milestone 导出
- [ ] 所有 P0 风险（见 `docs/project/risk-register.md`）已关闭或有明确缓解
- [ ] 发布范围未超过 milestone 承诺（避免夹带）
- [ ] 所有合入的 PR 均已过 DoD（`docs/project/dod.md`）

### Backlog 收敛（`docs/project/backlog.md`）
- [ ] **P0 Backlog 关闭数达 milestone 目标**（当前 RC 目标：5/5 — T-0002/T-0007/T-0010/T-0013/T-0017 等 P0 全 done）
- [ ] `blocked` 任务数 ≤ 3；若 ≥ 3 → 评估是否阻塞本次发布
- [ ] `in_dev` 任务均在本次发布范围内；逾期（Updated > 10 天）者必须触发 3 次失败重评（CLAUDE.md §9）
- [ ] 累计型任务（§3.1）的 Progress 达到下游声明阈值（如 T-0006 ≥ 150 以放行 T-0025）
- [ ] Sprint 承诺完成率近 3 个 Sprint 平均 > 75%

### 测试就绪
- [ ] 单元测试覆盖率 ≥ 当前阈值（初始 60%，每月递增）
- [ ] E2E 核心冒烟用例集（~20 条）全绿
- [ ] 压测基线无回退（`scripts/loadtest-benchmark.sh` 对比上版本）
- [ ] CPE 模拟器（`scripts/cpe_simulator.py`）在 staging 跑通

### 迁移就绪
- [ ] `bash scripts/check-migrations.sh` 通过（编号连续、up/down 配对）
- [ ] 新迁移在 staging 环境正向执行通过
- [ ] 新迁移在 staging 环境**反向执行通过**（回滚演练）
- [ ] 大表变更在接近生产数据量的环境验证过耗时与锁
- [ ] 如涉及 TimescaleDB policy 变更，已验证压缩/保留策略生效

---

## 2. 回滚就绪（T-2 天）

- [ ] 二进制回滚：上版本镜像 tag 可用且经过验证
- [ ] 数据库回滚：`down` 脚本在 staging 演练通过
- [ ] 配置回滚：配置变更有 diff 记录，可一键恢复
- [ ] Runbook：故障处理与回滚步骤文档已更新（`docs/runbook/*.md`）
- [ ] 值班人员知晓本次发布的回滚窗口与判断标准

---

## 3. 可观测性就绪（T-1 天）

- [ ] 本次新增指标已在 Prometheus 采集到
- [ ] 关键指标有告警规则（阈值、持续时间、通知渠道）
- [ ] 新增日志字段在 Zap 结构化中可查
- [ ] 链路追踪（OpenTelemetry）在新端点正常
- [ ] 发布后观察面板（Grafana dashboard）已准备好

---

## 4. 运营商验收（如涉及 Carrier 接口）

- [ ] CMCC 相关功能：按中移规范验证（告警码、参数路径、KPI 公式）
- [ ] CTCC 相关功能：按中电规范验证
- [ ] CUCC 相关功能：按中联规范验证
- [ ] `internal/core/carrier/*` 适配器的单元测试全绿
- [ ] 涉及多运营商的功能在三套 mock 环境下分别验证过

---

## 5. 发布窗口（T-0）

### 部署前
- [ ] 所有停机/流量切换窗口已与运维确认
- [ ] Maintenance 横幅（如需）已准备
- [ ] 发布通告已发出（内部 + 如涉及合作方）

### 部署中
- [ ] 三部署单元按顺序：`migrate` → `app` → `acs` → `worker`（或按本次发布的影响面调整）
- [ ] 每个单元启动后 `/healthz` 和 `/readyz` 都绿
- [ ] 关键指标无异常突增（错误率、会话失败、队列积压）
- [ ] 冒烟测试集跑一遍（登录/设备列表/告警列表/PM 查询/Inform 接收）

### 部署后（T+1 小时）
- [ ] 观察 1 小时内：错误率、延迟、会话数、CPU/内存
- [ ] 告警规则无误报/漏报
- [ ] 用户可正常访问（可选：真实 CPE 接入验证）

---

## 6. 发布后 24 小时

- [ ] 无 P0/P1 故障
- [ ] 关键 KPI 与发布前对比无异常回退
- [ ] 审计日志可查且完整
- [ ] 如有告警触发，已有 incident ticket 跟进
- [ ] Release notes 已归档到 `docs/release-notes/vX.Y.Z.md`

---

## 7. Hotfix 快速通道

遇到生产紧急故障时允许绕过完整 Gate，但必须：

- [ ] 至少过以下核心项：
  - [ ] 修复代码通过编译与单元测试
  - [ ] `check-migrations.sh` 通过（如涉及迁移）
  - [ ] 回滚预案已口头/文字确认
- [ ] 发布后 3 个工作日内补齐完整 Gate 核查
- [ ] 3 个工作日内产出 postmortem（`docs/postmortem/YYYYMMDD-*.md`）
- [ ] 下个 Sprint 回顾时复盘

---

## 8. Gate 失败处理

若某项 Gate 未过：
1. **立即停止发布**，不得"先上再说"
2. 记录失败项到 `docs/project/milestone/current.md` 的"Gate 失败登记"
3. 评估：修复 → 重新过 Gate ｜ 延期 → 通知相关方 ｜ 降级范围 → 拆分发布
4. 修复后不是打勾就行，要**重跑相应测试**

---

## 9. 本文件演进

- 每次发布回顾时补充遗漏的 Gate 项
- 每季度与 `docs/project/dod.md` 做一次一致性对齐
- 新风险类型（如引入新中间件）要先扩充 Gate 再投产

**当前版本**：v1.0（2026-04-20）
