# Issue #270 MML MOD 命令显示与重试修复审查报告

- 审查日期：2026-08-06
- 审查范围：MML 后端结果映射、Console MOD 回读/重试、TaskRecord Command 展示
- 结论：PASS_WITH_WARNINGS

## 变更摘要

1. 结构化 MOD 的 `param_code` 与 TR-069 path 通过 `paramRefs` 建立对应关系，历史任务可以恢复原始下发值。
2. Console 历史失败 MOD 使用保存的 path/value 快照，按选中的设备重新发送 `SetParameterValues`，不再强制重新配置。
3. Console MOD 复合回读结果行保留命令名称/命令码，结果表在行级字段缺失时从当前命令元信息兜底，避免计划命令显示为 `-`。
4. TaskRecord 命令格式化兼容 `values`、`parameter_values`、`parameters` 的数组和对象结构。
5. 后端结果映射在命令快照缺失时依据实际 RPC 方法补齐最小操作语义。

## 审查项

### 后端

- SQL、认证、运营商分支、资源释放：本次未新增相关风险。
- 结果映射兜底仅使用已有 RPC 方法和命令字段，不改变数据库结构及接口入参。
- 新增测试覆盖命令快照缺失时 `SetParameterValues` → `MOD` 的映射。

### 前端

- `paramRefs` 仅扩展已有 MML command detail 类型，保留旧 `param_paths` 兼容逻辑。
- MOD 历史重试使用当前记录保存的值，不读取顶部可能残留的其他命令配置。
- 重试仍限定为点击的单台设备，并沿用已有 raw execute 通道。
- Console 实时收口、历史懒加载和 TaskRecord 展示均覆盖到对应兜底路径。
- 未发现 CRITICAL 问题。

## 验证

- `npm run build`：通过。
- Console/TaskRecord 聚焦前端测试：3 个文件、13 个测试通过。
- `go build ./...`：通过。
- `go test ./internal/mml`：通过。
- 本地 Docker web 镜像已重建并热重启，`http://localhost:8081/` 返回 HTTP 200。

## 警告与限制

1. `npm run typecheck` 仍有 2 个既有 SystemLicense `BlobPart` 类型错误，位于 `src/pages/SystemLicense/History.tsx` 和 `src/pages/SystemLicense/index.tsx`，与本次修改无关。
2. `go test ./...` 有 1 个环境相关失败：`internal/task/TestService_PG_GetTask_TerminalTombstoneReturnsDurableDetails`，同时伴随测试 Redis 端口连接拒绝；`internal/mml` 测试通过。
3. 本次热重启部署的是本地 `goomc-local`，未部署远程 `172.24.224.251` 测试环境。
4. 本地 MinIO 服务实际运行，但健康检查因容器内缺少 `curl` 标记为 unhealthy；本次通过 `--no-deps` 恢复核心服务，未修改 MinIO 配置。
