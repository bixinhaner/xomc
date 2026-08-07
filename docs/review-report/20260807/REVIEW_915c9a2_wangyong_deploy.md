# MinIO 健康检查修复审查报告

- 日期：2026-08-07
- 基线：`915c9a284`
- 分支：`fix/minio-healthcheck`
- 范围：Docker Compose / MinIO
- 结论：`PASS`

## 变更

将 MinIO 健康检查从镜像内不存在的 `curl` 调整为镜像自带的 `mc ready local`，避免服务正常但因检查命令缺失被误判为 unhealthy。

## 审查结果

- 未发现 CRITICAL、WARNING 或安全问题。
- `mc ready local` 检查 MinIO 是否已能提供服务，比仅依赖缺失的 HTTP 客户端更符合当前固定镜像。
- 未修改镜像版本、端口、账号、数据卷或资源限制。

## 验证

- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh config --quiet`：通过。
- 容器内 `command -v mc`：返回 `/usr/bin/mc`。
- 容器内 `mc ready local`：返回 `The cluster is ready`。
- 当前 `goomc-local-minio-1` 健康状态：`healthy`。

## 结论

变更范围单一、可回滚，验证通过，可以提交。
