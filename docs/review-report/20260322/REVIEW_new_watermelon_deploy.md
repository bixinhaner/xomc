# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-03-22 |
| 作者 | watermelon |
| Scope | deploy |
| 文件数 | 7 |
| 新增行 | 484 |

## 审查范围

新增 `run/` 开发环境管理脚本套件：
- `start-all.sh` — 一键启动全部服务
- `start-deps.sh` — 启动基础依赖（PostgreSQL/Redis/NATS/MinIO）
- `start-backend.sh` — 启动后端三进程（app/acs/worker）
- `start-frontend.sh` — 启动前端 Vite dev server
- `stop-all.sh` — 一键停止全部服务
- `status.sh` — 服务状态检查
- `.gitignore` — 忽略 PID 文件和日志目录

## 审查发现

### INFO

1. **三级进程检测** (`start-backend.sh`): PID 文件 → 进程名(`pgrep -x`) → 端口 LISTEN 状态，确保不会因 PID 文件丢失而重复启动或误判。
2. **进程名 fallback** (`stop-all.sh`, `status.sh`): 即使 PID 文件不存在，也能通过进程名发现并管理正在运行的进程。
3. **端口检测精确** (`start-backend.sh`): 使用 `lsof -sTCP:LISTEN` 仅匹配监听状态，避免误报 ESTABLISHED 连接。
4. **优雅停止** (`stop-all.sh`): 先 SIGTERM 等待 3 秒，再 SIGKILL，给进程清理资源的机会。
5. **PostgreSQL 残留 PID 处理** (`start-deps.sh`): 检测 postmaster.pid 对应进程是否为 postgres，非 postgres 则清理。

### WARNING

1. **`pg_isready` 路径硬编码** (`start-deps.sh:19`, `status.sh:16`): 路径 `/usr/local/Cellar/postgresql@16/16.13/bin/pg_isready` 与特定 Homebrew 安装版本绑定，升级 PostgreSQL 后需手动更新。建议后续用 `$(brew --prefix postgresql@16)/bin/pg_isready` 动态获取。

## 结论

**PASS_WITH_WARNINGS** — 功能完整，检测逻辑健壮，无安全问题。WARNING 项为已知的开发环境路径绑定，不影响使用。
