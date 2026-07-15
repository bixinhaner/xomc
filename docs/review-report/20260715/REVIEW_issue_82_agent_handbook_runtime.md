# Issue #82 Agent API 手册运行时生成审查

结论: PASS_WITH_WARNINGS

## 范围

- 关联 Issue：#82「修复 Agent API 手册与运行时路由不一致」。
- OMC 在全部可选模块完成路由注册后，按当前实例的实际 Gin 路由生成不可变 Agent API 手册包。
- 构建期内嵌手册改为详细文档模板；运行时保留匹配路由的详细契约，并为新增路由生成基础契约。
- manifest、搜索索引、分类索引、文档、摘要及分块下载均来自同一份实际路由快照。

## 发现

- CRITICAL：无。
- WARNING：本机未安装 `golangci-lint`，无法执行该项检查；编译、全量测试、定向竞态测试和格式检查均已通过。
- INFO：运行时手册通过 `sync.Once` 只生成一次，请求并发不会重复构建或观察到不同版本。
- INFO：生成失败只记录结构化警告，不阻断 OMC 主业务启动；手册端点会返回明确错误。
- INFO：无数据库迁移、无新 REST 路由、无 WebUI 调用链改动，也不扩大 Agent 的权限边界。

## 验证

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./... -count=1`：通过，共 136 个包，包含 E2E 与 integration。
- `cd omcgo && go test ./internal/agentruntime/... -count=1`：通过。
- `cd omcgo && go test -race ./internal/agentruntime/... -count=1`：通过。
- `cd omcgo && bash scripts/check-migrations.sh`：通过，仅报告项目既有迁移告警，本次无迁移。
- `gofmt -l <本次 Go 文件>`：无输出。
- `git diff --check`：通过。

## DoD

- [x] 后端构建和全量测试通过。
- [x] 新增测试覆盖运行时新增路由、详细模板复用、确定性产物和不安全 operationId 拒绝路径。
- [x] 无新端点、无迁移、无 SQL、无运营商差异、无敏感信息输出。
- [x] 错误不会中断主业务启动，并通过结构化日志给出原因。
- [x] MR 说明需包含 Why、用户影响、验证结果并关联 Issue #82。
- [ ] `golangci-lint`：本机未安装，交由 GitLab CI 或具备工具的评审环境补充验证。

## 风险与用户影响

- 新版本 OMC 启动后会发布与自身实际能力一致的完整手册，Agent Studio 正常使用版本化手册，不再因构建期和运行时路由数不一致进入兼容目录。
- 手册只在启动阶段首次准备，占用一次 CPU 和内存；之后复用内存中的不可变压缩包。
- 若模板损坏或生成失败，OMC 业务功能仍可用，Agent 手册能力明确降级。
