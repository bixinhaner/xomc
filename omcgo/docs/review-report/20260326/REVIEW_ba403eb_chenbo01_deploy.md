# Code Review Report

| 项目 | 值 |
|------|-----|
| Commit (base) | ba403eb |
| Author | chenbo01 |
| Date | 2026-03-26 |
| Scope | deploy (nginx) |
| Type | perf |
| Verdict | ✅ PASS |

## 变更概述

Nginx 日志按端口拆分为独立文件，并挂载到宿主机 `run/logs/nginx/` 目录，便于运维排查。

## 变更文件

| 文件 | 变更 | 说明 |
|------|------|------|
| `deployments/docker/nginx.conf` | +5 -2 | 全局 access_log 关闭，日志下放到各 server 块 |
| `deployments/docker/default.conf` | +8 -2 | 8081 端口写 app_access/error.log，8080 端口写 acs_access/error.log |
| `deployments/docker/docker-compose.yml` | +1 | 挂载 `run/logs/nginx:/var/log/nginx` |

## 审查结果

### CRITICAL: 无

### WARNING: 无

### INFO

- **I1**: ACS 端口 access_log 已开启（从 off 改为 acs_access.log）。10K+ 设备场景下注意磁盘 IO。注释中已标注可按需关闭。

## 日志文件清单

| 文件 | 端口 | 内容 |
|------|------|------|
| `app_access.log` | :8081 | 前端 + REST API 访问日志 |
| `app_error.log` | :8081 | 管理面错误日志 |
| `acs_access.log` | :8080 | TR069 设备访问日志 |
| `acs_error.log` | :8080 | ACS 错误日志 |
| `error.log` | 全局 | Worker 启动/配置级错误 |
