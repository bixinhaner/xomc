package device

import "github.com/google/uuid"

// BatchOperationResult summarises the outcome of a batch operation.
type BatchOperationResult struct {
	Total     int              `json:"total"`
	Succeeded int              `json:"succeeded"`
	Failed    int              `json:"failed"`
	Errors    []BatchItemError `json:"errors,omitempty"`
}

// BatchItemError records the failure reason for a single item in a batch operation.
type BatchItemError struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// BatchIDsRequest is the request body for batch operations that accept a list of device IDs.
type BatchIDsRequest struct {
	IDs []uuid.UUID `json:"ids" binding:"required"`
}

// RestoreResult 描述回收站批量恢复的结果（#378）。
// 恢复改为「可部分成功」：与某活跃设备同 serial_number+carrier 的回收站行会被
// 跳过（不触发部分唯一索引 23505 冲突），其余正常恢复，整批不再因一台冲突回滚 500。
type RestoreResult struct {
	// Restored 实际恢复（deleted_at 置 NULL）的设备数。
	Restored int64 `json:"restored"`
	// Skipped 因 SN+carrier 冲突被跳过未恢复的设备数。
	Skipped int64 `json:"skipped"`
	// Conflicts 被跳过设备的明细（id / serialNumber / reason）。
	Conflicts []RestoreConflict `json:"conflicts,omitempty"`
}

// RestoreConflict 记录单台因 SN 冲突被跳过的设备（#378）。
type RestoreConflict struct {
	ID           string `json:"id"`
	SerialNumber string `json:"serialNumber"`
	Reason       string `json:"reason"`
}
