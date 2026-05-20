// Package backup — CommandKey 编码/解码 (M2 of backup-restore-alignment-plan).
//
// 规范要求 Upload/Download 的 CommandKey 形如 `{cellCode}_BACKUP` /
// `{cellCode}_RESTORE`，但 TransferComplete 回写时我们还需要把 CommandKey 反
// 解到具体 backup_task / restore_task 行。因此实际编码格式为：
//
//	{cellCode}_BACKUP_{taskID8}     // 备份上传
//	{cellCode}_RESTORE_{taskID8}    // 恢复下载
//
// 其中 `taskID8` 是任务 UUID 去掉横杠后的前 8 位 (16^8 ≈ 4.3B 空间，碰撞概率
// 与现有 file_path_recorder 一致)。同时为过渡兼容，路由层也接受历史前缀：
//
//	backup-{taskID8}                // 旧上传 (executor T-0079 前)
//	restore-{taskID8}               // 旧下载 (restore_service)
//
// 解析层 (ParseCommandKey) 同时识别新旧两种格式，便于一个 sprint 的灰度过渡。
package backup

import (
	"regexp"
	"strings"
)

// CommandKeyKind 表示 CommandKey 携带的任务类型。
type CommandKeyKind string

const (
	CommandKeyKindUnknown CommandKeyKind = ""
	CommandKeyKindBackup  CommandKeyKind = "backup"
	CommandKeyKindRestore CommandKeyKind = "restore"
)

// TaskIDPrefixLen 是 CommandKey / 文件名内嵌的任务 UUID 短前缀长度。
const TaskIDPrefixLen = 8

// hex8Re 仅匹配 8 位十六进制串 (大小写均可)；用于识别尾段 taskID8。
var hex8Re = regexp.MustCompile(`^[0-9a-fA-F]{8}$`)

// BuildBackupCommandKey 构造规范化的 Upload CommandKey：
//
//	{cellCode}_BACKUP_{taskID8}
//
// cellCode 为空时回落到 deviceSN（保持非空，方便运维侧检索）。
func BuildBackupCommandKey(cellCode, deviceSN, taskID string) string {
	cc := pickCellCode(cellCode, deviceSN)
	return cc + "_BACKUP_" + shortTaskID(taskID)
}

// BuildRestoreCommandKey 构造规范化的 Download CommandKey：
//
//	{cellCode}_RESTORE_{taskID8}
func BuildRestoreCommandKey(cellCode, deviceSN, taskID string) string {
	cc := pickCellCode(cellCode, deviceSN)
	return cc + "_RESTORE_" + shortTaskID(taskID)
}

// ParseCommandKey 识别 CommandKey 类型并返回任务 UUID 短前缀。
// 同时支持新格式 (`*_BACKUP_xxxxxxxx` / `*_RESTORE_xxxxxxxx`) 和历史前缀格式
// (`backup-xxxxxxxx` / `restore-xxxxxxxx`)；不匹配时返回 (Unknown, "")。
//
// 设计原则：解析对不规整输入 (空串、奇怪前缀、长尾) 一律宽容，返回 Unknown
// 而非 error —— TransferComplete 来自 CPE 完全不可控，router 必须能优雅跳过。
func ParseCommandKey(ck string) (CommandKeyKind, string) {
	if ck == "" {
		return CommandKeyKindUnknown, ""
	}

	// 新格式：以 `_BACKUP_` 或 `_RESTORE_` 作为分隔，末段为 8-hex。
	if idx := strings.LastIndex(ck, "_BACKUP_"); idx >= 0 {
		tail := ck[idx+len("_BACKUP_"):]
		if hex8Re.MatchString(tail) {
			return CommandKeyKindBackup, strings.ToLower(tail)
		}
	}
	if idx := strings.LastIndex(ck, "_RESTORE_"); idx >= 0 {
		tail := ck[idx+len("_RESTORE_"):]
		if hex8Re.MatchString(tail) {
			return CommandKeyKindRestore, strings.ToLower(tail)
		}
	}

	// 旧格式：`backup-xxxxxxxx` / `restore-xxxxxxxx`。
	switch {
	case strings.HasPrefix(ck, "backup-"):
		tail := strings.TrimPrefix(ck, "backup-")
		if hex8Re.MatchString(tail) {
			return CommandKeyKindBackup, strings.ToLower(tail)
		}
	case strings.HasPrefix(ck, "restore-"):
		tail := strings.TrimPrefix(ck, "restore-")
		if hex8Re.MatchString(tail) {
			return CommandKeyKindRestore, strings.ToLower(tail)
		}
	}

	return CommandKeyKindUnknown, ""
}

// pickCellCode 返回首选 cellCode，空时退化为 deviceSN，仍为空则给 "UNKNOWN"。
func pickCellCode(cellCode, deviceSN string) string {
	if cellCode != "" {
		return cellCode
	}
	if deviceSN != "" {
		return deviceSN
	}
	return "UNKNOWN"
}

// parseUUIDLike 去掉横杠以便取前 8 字符（兼容传入裸 32 位或 36 位 UUID）。
func parseUUIDLike(id string) string {
	return strings.ReplaceAll(id, "-", "")
}

// shortTaskID 取任务 UUID 字符串去横杠后的前 8 位；不足 8 位时整段返回。
func shortTaskID(id string) string {
	stripped := parseUUIDLike(id)
	if len(stripped) >= TaskIDPrefixLen {
		return stripped[:TaskIDPrefixLen]
	}
	return stripped
}
