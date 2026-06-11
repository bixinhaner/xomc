// Package backup — restore task model (T-0072).
//
// One RestoreTask row per `POST /backup/restore` call. Per-device fan-out is
// represented in TargetDeviceSNs (jsonb in DB); per-device progress is queried
// from the `device_tasks` table (method=Download) at read time, not stored
// here, to avoid double-write inconsistency.
package backup

import (
	"time"

	"github.com/google/uuid"
)

// RestoreStatus mirrors the DB CHECK constraint on restore_tasks.status.
type RestoreStatus string

const (
	RestorePending RestoreStatus = "pending"
	RestoreRunning RestoreStatus = "running"
	// RestoreDownloaded 是 #70 引入的中间态：CPE 已回报 TransferComplete
	// (FaultCode==0) "文件下载成功"，但 OMC 尚未完成主动完整性校验 —— 据此区分
	// "下载成功" 与 "校验通过/完成"。状态机：
	//   pending → running → downloaded → (completed | failed)
	RestoreDownloaded RestoreStatus = "downloaded"
	RestoreCompleted  RestoreStatus = "completed"
	RestoreFailed     RestoreStatus = "failed"
	RestoreCancelled  RestoreStatus = "cancelled"
)

// RestoreHashAlgo 标识 expected_hash / verified_hash 列里指纹的算法。
// 基线取自 config_snapshots.md5（#61 后为明文配置指纹），故默认 md5；
// 若快照库后续补 sha256 指纹，可在不改表的前提下切换。
type RestoreHashAlgo string

const (
	RestoreHashMD5    RestoreHashAlgo = "md5"
	RestoreHashSHA256 RestoreHashAlgo = "sha256"
)

// RestoreVerificationMethod 记录主动校验实际走的方式，用于审计与排查。
type RestoreVerificationMethod string

const (
	// RestoreVerifyNone：未执行主动校验（无 verifier 装配，或设备不支持回读）。
	RestoreVerifyNone RestoreVerificationMethod = "none"
	// RestoreVerifyGPVReadback：TR-069 GetParameterValues 回读关键参数比对。
	RestoreVerifyGPVReadback RestoreVerificationMethod = "gpv_readback"
	// RestoreVerifyDeviceChecksum：设备直接回报配置指纹与期望比对。
	RestoreVerifyDeviceChecksum RestoreVerificationMethod = "device_checksum"
)

// RestoreTask is the persisted row backing a restore operation.
type RestoreTask struct {
	ID               uuid.UUID     `json:"id"`
	SourceBucket     string        `json:"source_bucket"`
	SourceObjectPath string        `json:"source_object_path"`
	TargetDeviceSNs  []string      `json:"target_device_sns"`
	Status           RestoreStatus `json:"status"`
	Progress         int           `json:"progress"`
	ErrorMessage     *string       `json:"error_message,omitempty"`
	StartedAt        *time.Time    `json:"started_at,omitempty"`
	CompletedAt      *time.Time    `json:"completed_at,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	CreatedBy        *string       `json:"created_by,omitempty"`

	// M1 of backup-restore-alignment-plan: 运营商规范字段，全部可空。
	TaskSeq      *int64  `json:"task_seq,omitempty"`
	TaskName     *string `json:"task_name,omitempty"`
	TaskResult   *int16  `json:"task_result,omitempty"`
	OperatorCode *string `json:"operator_code,omitempty"`
	CreateUser   *string `json:"create_user,omitempty"`

	// #70 主动完整性校验字段，全部可空（migration 000036）。
	// ExpectedHash：下发 Download 时记录的期望明文指纹（基线取 config_snapshots.md5）。
	// HashAlgo：ExpectedHash 的算法（md5 / sha256）。
	// VerifiedHash / VerificationMethod / VerifiedAt：主动校验结果留痕。
	// DownloadedAt：进入 downloaded 态（CPE 回报下载成功）的时刻。
	// SourceVersion：快照/备份来源的设备软件版本，供跨版本恢复对比与审计。
	ExpectedHash       *string                    `json:"expected_hash,omitempty"`
	HashAlgo           *RestoreHashAlgo           `json:"hash_algo,omitempty"`
	VerifiedHash       *string                    `json:"verified_hash,omitempty"`
	VerificationMethod *RestoreVerificationMethod `json:"verification_method,omitempty"`
	VerifiedAt         *time.Time                 `json:"verified_at,omitempty"`
	DownloadedAt       *time.Time                 `json:"downloaded_at,omitempty"`
	SourceVersion      *string                    `json:"source_version,omitempty"`
}
