# 代码审查报告：Sync GPV 会话恢复与结果汇总

- 日期：2026-07-14
- 基线：`03031793e` (`origin/main`)
- 作者：wangyong
- 范围：ACS 会话、设备任务队列、Path B 参数同步、参数树状态展示、NATS 部署配置
- 结论：`PASS_WITH_WARNINGS`

## 变更概览

本次变更加强新 Inform 对上一 CWMP 会话遗留任务的恢复，以 PostgreSQL 为权威恢复 `sent` 任务，并修复 Redis Cluster 下并发出队的重复派发风险。Path B 同步新增按设备的 PostgreSQL advisory lock 与运行中防重，自动同步改用每轮独立 `source_id`，同时扩展同步结果统计和前端展示。NATS `max_payload` 提升到 5 MiB，使典型 BSC BTS 对象能够整对象同步。

## 审查发现

### WARNING-1：防重跳过可能使 provisioning 永久停留在 syncing

- 位置：`omcgo/internal/provision/sync_pathb.go:80`
- 关联：`omcgo/internal/provision/engine.go:738`

已有同步任务时 `StartPathBSync` 返回 `(true, 0, nil)`。`handleAutoSync` 在调用前已经把 provisioning task 切换到 `StateSyncing`，随后仅判断 `used`，因此会把防重跳过视为成功启动。该 provisioning task 没有对应 `source_id` 的 GPV 子任务，无法通过完成回调退出 `syncing`。

建议为“已启动 / 已存在 / 不可用”提供明确返回状态，或让 provisioning 调用方在进入 `syncing` 前处理 `gpvTaskCount == 0`。

### WARNING-2：批量 GPV Fault 会将未执行路径统计为成功

- 位置：`omcgo/internal/task/pg_repository.go:593`
- 关联：`omcgo/internal/acs/handler.go:1250`

GPV Fault 的 `param_faults` 仅包含 FaultString 暴露的一条 `badPath` 提示。一旦该数组非空，汇总 SQL 不再把任务的全部请求路径计入失败，随后用 `requested - failed` 得到成功数。例如三路径 GPV 遇到不可恢复的 9002 Fault 时，界面可能显示成功 2、失败 1，尽管整条 RPC 没有成功结果。

建议失败的 GPV command 将全部请求路径计为失败，并用 `param_faults` 只补充匹配路径的错误详情。

### WARNING-3：大对象展开阈值未给 NATS envelope 留预算

- 位置：`omcgo/internal/provision/sync_pathb_expand.go:126`

对象响应估算达到完整 5 MiB 才触发展开，而服务端 `max_payload` 同样为 5 MiB。估算值尚未覆盖事件 envelope 和字段值波动，落在 3.75–5 MiB 区间的响应仍可能超过 NATS 上限。

建议复用已有的 `gpvNATSPayloadBudgetBytes`（5 MiB 的 75%）作为展开阈值。

## 安全与质量检查

- SQL 使用 Squirrel 或参数化 SQL；未发现字符串拼接注入风险。
- 未新增运营商硬编码分支。
- Redis 并发出队以 `ZREM` 返回值作为抢占结果，避免同一任务被多个实例返回。
- advisory lock 使用 transaction-scoped lock，并通过带超时的 rollback 释放。
- 前端新增文案均进入中英文 i18n。
- 未发现凭据、环境文件、构建产物或本机 AI 配置进入提交。

## 验证

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./...`：通过。
- `cd omcmb && npm run typecheck`：通过。
- `git diff --check`：通过。
- `docker compose -f deployments/docker/docker-compose.yml config --quiet`：通过。
- `docker compose -f deployments/docker/docker-compose.test.yml config --quiet`：通过。
- Release Compose 独立解析：未验证；本机未提供发行包要求的镜像及数据库环境变量。

## 结论

未发现流程定义中的 `CRITICAL` 问题，允许提交并进入 MR 审查。上述三项 WARNING 建议在合入前由维护者确认处理策略。
