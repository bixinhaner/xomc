# Code Review: MinIO 路径自动翻译为 HTTP 下载地址

**Date**: 2026-03-31
**Scope**: `internal/acs/rpc/dispatcher.go`, `internal/software/service.go`, `internal/filemanager/service.go`
**Verdict**: PASS

## Stats

- 3 files changed, +51 -6 lines

## 设计

- `software/service.go` 和 `filemanager/service.go` 存储纯 MinIO 路径（如 `firmware/v2.0.bin`）
- `DownloadHandler.BuildRequest()` 检测 URL 中无 `://` → 拼接为 `{BaseURL}{Path}/{minioPath}`
- 带 `://` 的完整 URL 直接传递给 CPE，不做翻译
- 自动注入下载认证凭据（当 params 中未设置时）

## Findings

### INFO: DispatcherConfig 可选参数模式
`NewDispatcher(cfgs ...DispatcherConfig)` 使用变长参数实现可选配置，保持向后兼容。

### INFO: URL 检测逻辑
`!strings.Contains(params.URL, "://")` 简洁有效，覆盖所有 URL scheme（http/https/ftp/file）。

## Conclusion

逻辑清晰，无安全或正确性问题。
