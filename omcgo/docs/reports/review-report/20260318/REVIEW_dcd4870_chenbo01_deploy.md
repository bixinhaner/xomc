# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-03-18 |
| 作者 | chenbo01 |
| 基准 | dcd4870 |
| Scope | deploy, acs |
| Type | fix |
| 文件数 | 5 |

## 变更概述

调整 Docker 部署端口规划，通过 nginx 8080 端口统一代理 ACS 服务。App 端口从 8080 改为 8081，ACS 内部端口改为标准 TR-069 端口 7547，nginx 作为 ACS 网关对外提供 8080 端口。

## 审查结果

**结论: PASS**

### 检查项

| # | 检查项 | 结果 | 说明 |
|---|--------|------|------|
| 1 | 端口规划一致性 | ✅ | nginx:8080 → acs:7547/acs，app:8081 用于 API，各端口职责清晰 |
| 2 | nginx 配置 | ✅ | ACS Gateway 支持 TR-069 长连接（300s 超时）、10M body、健康检查代理 |
| 3 | Dockerfile 端口 | ✅ | EXPOSE 8081 与 OMCGO_SERVER_PORT 一致 |
| 4 | 服务依赖 | ✅ | web depends_on app + acs |
| 5 | 测试配置同步 | ✅ | config.test.yaml 端口与 prod 一致 |

### 变更详情

| 文件 | 变更 |
|------|------|
| `docker-compose.yml` | 移除 ACS 8080 端口暴露，App 端口改为 8081，nginx 新增 8080 端口 |
| `nginx.conf` | 新增 ACS Gateway server 块（port 8080），代理 / 到 acs:7547/acs，/healthz 到 acs:7547/healthz |
| `Dockerfile.app` | EXPOSE 端口改为 8081 8444 |
| `config.test.yaml` | ACS 端口改为 7547/7548（与 prod 一致） |
| `server.go` | handler 路径从 "" 改为 "/acs" |

### 零 CRITICAL / 零 WARNING
