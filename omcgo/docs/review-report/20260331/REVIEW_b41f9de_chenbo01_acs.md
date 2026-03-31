# Code Review Report

| 项目 | 值 |
|------|---|
| 日期 | 2026-03-31 |
| 基准提交 | b41f9de |
| 作者 | chenbo01 |
| Scope | acs (rpctool) |
| 审查结论 | **PASS** |

## 变更概述

rpctool upload 命令简化：`--url` 参数改为可选，省略时自动从 ACS 配置文件构建完整上传 URL
（包含 path、fileType query param、filename、认证凭据）。

## 变更文件 (5)

| 文件 | 变更 |
|------|------|
| `scripts/rpctool/common.go` | 新增 `uploadQueryParam` 映射表 + `buildUploadURL()` 函数 |
| `scripts/rpctool/cmd_upload.go` | `--url` 改为可选，三级 URL 构建逻辑（无 URL / 仅 host / 完整 URL） |
| `scripts/rpctool/main.go` | 分离 `loadConfig()`，dry-run 模式也加载配置 |
| `scripts/rpctool/README.md` | 更新示例为简化形式，更新参数说明 |
| `cmd/acs/etc/config.dev.yaml` | upload/download base_url 改为网关地址 `http://localhost:8080` |

## 审查发现

### INFO

1. **fallback 逻辑** (`common.go:104-106`): 当 alias 不在 `uploadQueryParam` 映射中时，直接用 alias 作为 query param code。对 raw FileType 字符串输入（含空格）会产生非预期 URL，但这���场景下用户通常会指定完整 `--url`。
2. **URL 完整性检测** (`cmd_upload.go:108`): 用 `strings.Contains(url, "/smallcell/")` 判断是否已是完整 URL，对非标准路径不适用，但当前项目上传路径固定为 `/smallcell/FileUploadService`，可接受。

## 结论

变更范围清晰，逻辑完整（三级 URL 构建 + dry-run 配置加载 + 凭据自动注入）。审查通过。
