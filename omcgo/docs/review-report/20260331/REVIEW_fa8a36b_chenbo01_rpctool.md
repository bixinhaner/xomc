# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-03-31 |
| 基准提交 | fa8a36b |
| 作者 | chenbo01 |
| Scope | acs (rpctool) |
| 审查结论 | **PASS** |

## 变更概述

新增 `rpctool` CLI 调试工具，用于手动创建 TR-069 RPC 任务。工具通过 `TaskService` 双写 Redis 队列和 PostgreSQL `device_tasks` 表。

### 文件列表

| 文件 | 操作 |
|------|------|
| `scripts/rpctool/main.go` | 新增 — CLI 入口，cobra 根命令，基础设施初始化 |
| `scripts/rpctool/common.go` | 新增 — FileType 映射表，createAndPrint 通用函数 |
| `scripts/rpctool/cmd_upload.go` | 新增 — Upload RPC 命令 |
| `scripts/rpctool/cmd_download.go` | 新增 — Download RPC 命令 |
| `scripts/rpctool/cmd_getpv.go` | 新增 — GetParameterValues/SetParameterValues 命令 |
| `scripts/rpctool/cmd_getpn.go` | 新增 — GetParameterNames 命令 |
| `scripts/rpctool/cmd_reboot.go` | 新增 — Reboot/FactoryReset 命令 |
| `scripts/rpctool/cmd_addobject.go` | 新增 — AddObject/DeleteObject 命令 |
| `scripts/rpctool/cmd_queue.go` | 新增 — 队列管理（list/peek/len/clear/history） |
| `scripts/rpctool/README.md` | 新增 — 工具文档 |
| `Makefile` | 修改 — 新增 `build-rpctool` 目标 |

## 审查检���项

### Go 工程规范

- [x] 错误处理：所有错误正确 wrap 并返回上下文
- [x] 命名规范：函数和变量遵循 Go 惯例
- [x] 接口使用：通过 `TaskService` 抽象层操作，未直接操作 Redis/PG
- [x] 资源管理：`requireSNAndInfra` 确保 dry-run 时不初始化连接

### 安全审查

- [x] 无 SQL 注入风险（通过 TaskService/Repository 层操作）
- [x] 无硬编码密码/凭证
- [x] queue clear 操作需 `--force` 确认

### TR-069 协议

- [x] FileType 映射完整覆盖 TR-069 标准类型和运营商扩展类型
- [x] Upload/Download FileType 正确区分（Upload: 1=���置/2=日志，Download: 1=固件/2=Web/3=配置）
- [x] SetParameterValues 支持类型声明（xsd:string, xsd:unsignedInt 等）

### 发现

无 CRITICAL 或 WARNING 级问题。

| 级别 | 说明 |
|------|------|
| INFO | `queue list` 直接操作 Redis key `acs:taskq:` 而非通过 TaskQueue 接口，属于调试工具可接受 |
| INFO | `main.go` 中 `pgPool` 未在程序退出时显式关闭，CLI 工具场景可接受（进程退出自动释放） |
