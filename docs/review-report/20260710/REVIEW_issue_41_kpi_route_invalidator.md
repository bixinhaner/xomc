# Review：Issue #41 KPI 路由统一失效、人工恢复与告警

- 日期：2026-07-10
- 固定点：`0eb8ae3d`（Issue #40 提交）
- 审查范围：`fix/41-kpi-route-invalidator-recovery-alerts` 相对固定点的工作区 diff
- 规格源：GitLab Issue #41；父 PRD #39
- 结论：PASS；Standards findings = 0；Spec findings = 0；CRITICAL = 0

## Standards

双轴审查的 Standards 初审发现两项硬缺口，均已修复并经二次复审关闭：

1. E2E 最初只用一个多状态 claim，无法分别证明权限行为。现改为 3 个严格 claim：`super_admin=200` 并校验结果结构、未认证 `401`、普通用户 `403`，E/R = 3/1。
2. 新端点最初未进入 OpenAPI。现已补 `POST /api/v1/admin/kpi-routes/refresh` 的 200/401/403/503 契约，并以 `EnvelopeErrorResponse` 对齐实际 `ret/msg/data` 失败信封。

最终复审未发现并发/context、结构化日志、Prometheus label 基数、JWT + `RequireSuperAdmin`、API 文档或 DoD 违规。未发现运营商硬编码、字符串 SQL、裸 `panic`、公共接口扩宽为 `any/interface{}`、删除测试或安全性降低。

## Spec

已满足 Issue #41 全部验收要求：

- 统一失效器先清本进程 L1，再以有界后台 context 最多三次推进 Redis 全局版本。
- 首次成功、重试后成功、三次最终失败、调用方取消后继续有界后台工作、超时停止均有行为测试。
- 最终失败不回滚已提交的 indicator reload；产生 ERROR、请求 ID、低基数失败指标和可操作告警。
- 人工入口仅 `super_admin` 可调用；成功返回版本/尝试次数/同步范围，失败返回脱敏且明确的 503。
- 版本读取失败、stale eviction、失效成功/失败、人工刷新结果均有 Prometheus 信号。
- 告警覆盖 bump 最终失败与版本读取持续失败，规则语法校验通过。
- labels 仅包含受控的 `trigger/result/scope`；未知 trigger 归一为 `other`。
- 无 Redis 测试证明本进程 L1 清理，响应和 Runbook 明确 `scope=local`、多进程不能即时同步。

未发现缺失需求、错误实现或越过 #42/#43 写后热更新范围的 scope creep。

## 安全审查

- 路由挂在已认证 `v1` 的 `superAdminGroup`，再经过 `RequireSuperAdmin`；普通角色不可通过显式 RBAC 绕过。
- 端点无用户输入和 SQL；失败响应不泄露 Redis 地址/底层连接错误，详细错误仅进入带 request ID 的结构化日志。
- 指标 label 不含 product ID、设备 SN、platform 或其他用户输入。

## DoD 与验证

- `go build ./...`：PASS
- `go test ./...`：PASS
- `go test -race ./internal/pm/kpi/router ./internal/pm/indicator ./cmd/app/provider -count=1`：PASS
- `go vet ./...`：PASS
- Router 覆盖率：86.1%（#40 基线 82.3%，未下降）
- `bash -n scripts/e2e_verify.sh`：PASS；新增 3 个严格 claim，E/R = 3/1
- `promtool check rules /rules/omc-rules.yml`：PASS（14 rules found）
- OpenAPI YAML 解析：PASS
- `golangci-lint`：N/A，本机未安装；以 `go vet ./...` 补充静态检查，未伪报执行
- 真实后端 E2E 套件：未执行；本地 compose 无运行中的 app。端点 Gin 行为测试已通过，严格 E2E claim 已进入套件
- 前端、浏览器、迁移：N/A（本次无相关改动）

汇总：Standards 0 findings；Spec 0 findings；最高严重级别为无；CRITICAL = 0。
