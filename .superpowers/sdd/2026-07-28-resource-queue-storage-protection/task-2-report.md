# Task 2 实施报告：Grafana / PromQL 静态校验入口

## 范围

仅完成 Task 2：

- 新增 `deployments/monitoring/tests/validate-dashboards.sh`。
- 新增 `deployments/monitoring/tests/promql-probes.txt`。
- 补充 `deployments/monitoring/README.md` 的 Prometheus → Grafana → 浏览器验收顺序和指标语义。

未实现 Task 3/8 的 dashboard、alert、recording rule 或 Go 代码，也未修改现有 dashboard JSON。

## 实现结果

`validate-dashboards.sh` 会：

- 对 `grafana/dashboards/` 下四个基线 dashboard 和后续新增 JSON 使用 `jq` 校验；
- 缺少基线 dashboard 时给出明确错误；尚无后续新增 dashboard 时输出非致命提示；
- 校验顶层 UID 非空且唯一；
- 拒绝 JSON 字符串中的 `namespace="omcgo"` 与 `name=~`。

`promql-probes.txt` 覆盖 CPU、内存、node filesystem、cAdvisor、MinIO、NATS、Redis、PostgreSQL、PM durable queue 与写入保护，并为每项写明 MUST-HAVE、NO-DATA 或 ZERO 的判定语义。

README 已明确：不得仅凭 dashboard JSON 中存在 panel 认定完成；必须先验证 Prometheus `/api/v1/query`，再验证 Grafana panel，最后以浏览器中的实际 dashboard 为准。README 同时明确排除进程内内存队列，并区分真实 `0`、`No data` 与查询/采集 `failure`。

## 验证状态

未执行 `deployments/monitoring/tests/validate-dashboards.sh` 或额外 `jq` 命令：在开始验证前收到“立即停止、不要再运行额外长时间命令”的指令，遵从该指令先保存报告并提交已有改动。

预期 concern：现有 dashboard JSON 仍包含 Task 3 范围内的禁用旧筛选（`namespace="omcgo"`、`name=~`）。本 Task 的脚本会将其报告为失败；本次未改动 dashboard 以避免越过 Task 2 范围。

## 提交范围

提交仅包含本报告和以下三个 Task 2 文件；未包含未跟踪的 `.codex-tools/`。
