# Code Review: rpctool 完善所有 TR-069 文件类型支持

**Date**: 2026-03-31
**Scope**: `scripts/rpctool/` (common.go, cmd_download.go, cmd_upload.go, README.md)
**Verdict**: PASS_WITH_WARNINGS

## Stats

- 4 files changed, +360 -105 lines

## Findings

### HIGH: Sentinel Error String 作为控制流信号

**文件**: `common.go:110,137`, `cmd_download.go:79`, `cmd_upload.go:89`

使用 `fmt.Errorf("__OUI_CONFIG__")` + `strings.Contains` 判断 OUI 配置类型。应改用 `errors.Is` + sentinel `var`。作为 CLI 调试工具可接受，建议后续迭代优化。

### MEDIUM: uploadFileTypeCodeMap 缺少 code "10" 注释

`resolveUploadFileType` 在 code map 之前拦截 "10"，逻辑正确但代码可读性不足。

### MEDIUM: 代码 4 歧义需注释说明

`log-ext` → `"4 Vendor Log File"` 和 `pm` → `"4 Vendor PM File"` 都使用代码 4，数字参数 `-t 4` 默认映射为 PM，符合 LMT 约定但需注释。

### INFO: var 声明后 := 覆盖

`var resolvedFT string` 后紧跟 `:=` 赋值，前者无意义。随 sentinel error 一并修复。

## Conclusion

无阻塞问题。所有发现均为 CLI 工具内部代码质量改进建议，不影响功能正确性。
