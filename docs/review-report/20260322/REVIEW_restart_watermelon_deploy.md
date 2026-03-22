# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-03-22 |
| 作者 | watermelon |
| Scope | deploy |
| 文件数 | 4 |
| 新增行 | 39 |

## 审查范围

- 新增 `run/scripts/restart-all.sh` 一键重启脚本（停止→编译→启动）
- 修改三个 `config.dev.yaml` 日志输出路径：`/var/log/omcgo/` → `../run/logs/`

## 审查发现

### INFO

1. **restart-all.sh 流程完整**: stop → make build → start-all，编译失败时 `set -e` + 显式 `exit 1` 阻止继续启动。
2. **日志路径仅改 dev 配置**: prod/test 配置未受影响，生产环境无风险。
3. **相对路径依赖工作目录**: `../run/logs/` 依赖于从 `omcgo/` 目录启动，与 `start-backend.sh` 中 `cd "$OMCGO_DIR"` 一致。

## 结论

**PASS** — 变更清晰安全，无风险。
