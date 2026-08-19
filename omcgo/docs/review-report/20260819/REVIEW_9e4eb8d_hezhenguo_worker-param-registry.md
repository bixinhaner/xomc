# Worker ParamModel Registry 复用修复审查报告

- 日期：2026-08-19
- Issue：#342
- 审查基线：`origin/main` (`ea22b421d`)
- 待审提交：`4492c3a79`
- 范围：`omcgo/cmd/worker`
- 结论：**PASS_WITH_WARNINGS**

## 变更摘要

worker 原有围栏控制启动路径已创建 ParamModel Registry，并向进程级 Prometheus Registry 注册参数模型指标。接入控制 worker 再次创建同类 Registry 时，会因同名 collector 重复注册而 panic。本次将 ParamModel Registry 挂载到 `workerInfra`，由围栏控制和接入控制共享，并新增回归测试锁定单进程复用契约。

## 审查结果

### CRITICAL

无。

### WARNING

无代码问题。

### INFO

- Registry 创建与复用均发生在 worker 串行启动阶段，不引入并发写入风险。
- 接入控制继续复用同一个 Product Registry 与 ParamModel Registry，GPS 路径解析行为不变。
- 未新增 API、数据库迁移、SQL、运营商分支或用户可见文案。
- 本机未安装 `golangci-lint`，未执行该工具；已执行 `go vet ./...` 且通过，CI 仍应运行项目配置的 lint 门禁。

## 验证证据

在 `omcgo/` 目录执行：

| 命令 | 结果 |
| --- | --- |
| `go build ./...` | PASS |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./cmd/worker` | PASS |
| `git diff --check origin/main...HEAD` | PASS |

## DoD 结论

- 回归测试覆盖重复创建入口的复用契约。
- 无新增端点，E2E 为 N/A。
- 无迁移，迁移演练为 N/A。
- 无前端改动，浏览器验证与前端类型检查为 N/A。
- 可进入提交、推送与 Merge Request 流程。
