# Code Review Report

| 项目 | 值 |
|------|---|
| 日期 | 2026-03-31 |
| 基准提交 | 3d4a0a5 |
| 作者 | chenbo01 |
| Scope | acs (rpctool) |
| 审查结论 | **PASS** |

## 变更概述

rpctool 工具增强：
1. 新增 `config-11` 上传文件类型别名（`11 Configuration File`）
2. 默认配置从 `config.dev.yaml` 改为 `config.local.yaml`（修复本地运行 Redis DNS 解析失败）
3. 新增 `--redis` 和 `--db` 命令行参数支持直接覆盖连接地址
4. README 文档同步更新

## 变更文件 (3)

| 文件 | 变更 |
|------|------|
| `scripts/rpctool/common.go` | +1 uploadFileTypeMap 条目 `config-11` |
| `scripts/rpctool/main.go` | 默认配置改 local.yaml + `--redis`/`--db` CLI flags |
| `scripts/rpctool/README.md` | config-11 示例、对照表、注意事项 |

## 审查发现

### INFO

1. **downloadFileTypeMap 对齐不一致** (`common.go:83-86`)
   - `script` 条目缩短了对齐空格，其他条目保留。纯格式问题，不影响功能。

2. **预存 UTF-8 乱码** (`main.go:52`)
   - `"设备序列号 (必���)"` 应为 `"设备序列号 (必填)"`。此问题在本次变更之前已存在。

## 结论

无安全风险，无逻辑错误，变更范围小且明确。审查通过。
