# Code Review Report

| 项目 | 值 |
|------|-----|
| 审查时间 | 2026-03-24 |
| 基准提交 | 64b030b |
| 作者 | chenbo01 |
| Scope | deploy |
| 变更文件 | 15 |
| 插入/删除 | +146 / -138 |
| 结论 | **PASS_WITH_WARNINGS** |

## 变更概要

统一 Docker 多环境配置隔离方案：所有环境（dev/test/prod）配置文件内置于镜像，通过 `OMCGO_ENV` 环境变量切换，消除 `docker-compose.yml` 中大量冗余的环境变量覆盖。

## 变更文件清单

| 文件 | 变更类型 |
|------|---------|
| `.gitignore` | fix: `worker` → `/worker` 避免匹配 `cmd/worker/` |
| `cmd/{acs,app,worker}/etc/config.dev.yaml` | `localhost` → Docker 服务名 |
| `cmd/{acs,app,worker}/etc/config.test.yaml` | `localhost` → Docker 服务名 |
| `cmd/{acs,app,worker}/etc/config.prod.yaml` | `${VAR}` 占位符 → 实际 Docker 值 |
| `deployments/docker/Dockerfile.{acs,app,worker}` | 打包三套配置 + entrypoint.sh |
| `deployments/docker/docker-compose.yml` | 精简 ~45 行冗余环境变量 |
| `deployments/docker/entrypoint.sh` | 重写为环境感知配置解析 |

## 审查发现

### WARNING

1. **生产配置包含开发凭据** — `config.prod.yaml` 中硬编码了 `omcgo123`、`minioadmin`、JWT secret 等开发密码。当前阶段可接受（Docker Compose 开发环境），真正生产部署前必须替换为安全凭据。

### INFO

1. `.gitignore` 修复正确 — `/worker` 仅匹配根目录下的 `worker`，不再误匹配 `cmd/worker/`
2. `entrypoint.sh` 包含合理的 fallback 逻辑（配置文件不存在时降级到 dev）
3. `docker-compose.yml` 大幅精简，每个服务只需 `OMCGO_ENV` 一个环境变量
4. 生产日志级别调整为 `warn`、轮转参数扩大（20MB/30天/50份）— 合理
5. Dockerfile 统一了 `TZ=Asia/Shanghai` 和 `OMCGO_SERVICE` 环境变量

## 使用说明

```bash
# 默认 dev 环境
docker compose up -d

# 指定环境
OMCGO_ENV=test docker compose up -d
OMCGO_ENV=prod docker compose up -d
```
