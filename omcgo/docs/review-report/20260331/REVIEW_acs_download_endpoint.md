# Code Review: ACS 文件下载端点实现

**Date**: 2026-03-31
**Scope**: `internal/acs/download/`, `server.go`, `handler.go`, `main.go`, `appconfig/config.go`, YAML configs
**Verdict**: PASS (CRITICAL 已修复)

## Stats

- 9 files changed, +216 -1 lines (含安全修复)

## 已修复的 CRITICAL 问题

### C1: Basic Auth 时序攻击 → 已修复
- 原: `username != h.username` 字符串直接比较
- 修: `crypto/subtle.ConstantTimeCompare` 常量时间比较
- 参考: upload/handler.go 同样实现

### C2: 路径遍历防护不充分 → 已修复
- 原: 仅检查 objectPath 的 `..`
- 修: `path.Clean` 清洗 bucket 和 objectPath，同时拒绝 `..` 和 `/` 前缀

## 已修复的 MEDIUM 问题

### M1: Content-Disposition 引号格式 → 已修复
- `%q` (Go 语法) → `"%s"` (RFC 6266 格式)

### M4: 多余的 WriteHeader 调用 → 已修复
- HEAD 请求和 GET 请求均移除显式 `WriteHeader(200)` 调用

## 已知但接受的问题

### MEDIUM: Dispatcher 重建模式
main.go 中先创建 dispatcher 再覆盖，可在后续重构中改为直接传递 DispatcherConfig。

### MEDIUM: YAML 中硬编码密码
dev/test/prod YAML 含��认密码。生产环境应通过 `OMCGO_DOWNLOAD_PASSWORD` 环境变量覆盖。

### INFO: 缺少 ETag/Last-Modified 响应头
可在后续迭代中从 MinIO ObjectInfo 添加，支持条件请求减少重复传输。

## Conclusion

CRITICAL 安全问题已全部修复。下载端点实现与 upload handler 安全标准一致。
