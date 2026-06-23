// Package backup — BackupPolicy persistence (T-0071 / R-102 followup).
//
// Singleton policy: one row per deployment. Application-layer enforcement
// (PolicyService.Get/Update) reads `ORDER BY updated_at DESC LIMIT 1` and
// upserts; DB has no UNIQUE constraint to leave per-tenant extension space.
//
// ⚠ Enforcement boundary: this module delivers persistence only. The fields
// below are stored faithfully but most are not yet acted upon by the backup
// executor. See `docs/project/prd/T-0071-backup-policy-persistence.md` §2 for
// the per-category enforcement followup roadmap (T-0073 cleanup+alarm,
// T-0074 compression, T-0075 encryption).
package backup

import (
	"time"

	"github.com/google/uuid"
)

// BackupPolicy is the persisted policy row mirroring the frontend
// `BackupPolicyValues` shape (19 fields × 7 categories).
type BackupPolicy struct {
	ID uuid.UUID `json:"id"`

	// 保留策略
	RetentionDays  int `json:"retention_days"`
	MaxBackupCount int `json:"max_backup_count"`
	MinBackupCount int `json:"min_backup_count"`

	// 自动清理（enforcement = T-0073）
	AutoCleanup      bool   `json:"auto_cleanup"`
	CleanupTime      string `json:"cleanup_time"`        // "HH:MM"
	CleanupDayOfWeek int    `json:"cleanup_day_of_week"` // -1 = everyday; 0..6 = sun..sat
	KeepLastN        int    `json:"keep_last_n"`

	// 压缩（enforcement = T-0074）
	EnableCompression bool   `json:"enable_compression"`
	CompressionLevel  int    `json:"compression_level"`  // 1..9
	CompressionFormat string `json:"compression_format"` // gzip|bzip2|lz4|zstd

	// 存储
	StorageBackend string     `json:"storage_backend"` // local|ftp|sftp|nfs
	FTPConfigID    *uuid.UUID `json:"ftp_config_id,omitempty"`
	LocalPath      string     `json:"local_path"`
	MaxStorageGB   int        `json:"max_storage_gb"`

	// 加密（enforcement = T-0075，安全敏感）
	EnableEncryption    bool   `json:"enable_encryption"`
	EncryptionAlgorithm string `json:"encryption_algorithm"` // AES-256-GCM|AES-256-CBC|ChaCha20-Poly1305

	// 告警（enforcement = T-0073/T-0082；severity policy-driven = T-0084）
	AlertOnFailure        bool   `json:"alert_on_failure"`
	AlertEmail            string `json:"alert_email"`
	AlertThresholdPercent int    `json:"alert_threshold_percent"`
	// AlertSeverity controls the severity field on both backup_task_failed
	// (T-0073) and backup_storage_threshold_exceeded (T-0082) alarm payloads.
	// Single column covers both alarm types — operators rarely want to split
	// failure-vs-storage severity (PRD T-0084 §2.1).
	AlertSeverity string `json:"alert_severity"` // warning|major|critical

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DefaultPolicy returns the canonical defaults used when no row exists yet.
// Values mirror frontend `DEFAULT_VALUES` so first-load UI matches DB defaults.
func DefaultPolicy() *BackupPolicy {
	return &BackupPolicy{
		RetentionDays:         30,
		MaxBackupCount:        100,
		MinBackupCount:        3,
		AutoCleanup:           true,
		CleanupTime:           "03:00",
		CleanupDayOfWeek:      -1,
		KeepLastN:             5,
		// issue #585：配置备份默认不再压缩。只有 PM / MR 才走压缩链路
		// （PM 由 ACS pmSyncGzip / MR 由 worker 压缩），配置备份原始字节直存，
		// 避免任务管理 presigned URL 出现 .xml.gz 命名 / 浏览器侧 Content-Encoding
		// 透明解压行为不一致。CompressionLevel/CompressionFormat 仍保留默认值
		// 仅作 schema 兼容，运维即使在 UI 勾选启用，ACS 也已不再装配压缩链路。
		EnableCompression:     false,
		CompressionLevel:      6,
		CompressionFormat:     "gzip",
		StorageBackend:        "local",
		LocalPath:             "/var/backup/omc",
		MaxStorageGB:          500,
		EnableEncryption:      false,
		EncryptionAlgorithm:   "AES-256-GCM",
		AlertOnFailure:        true,
		AlertEmail:            "",
		AlertThresholdPercent: 80,
		AlertSeverity:         "major",
	}
}

// validBackupPolicyCompressionFormats / ...StorageBackends / ...EncryptionAlgorithms
// pinned at code level to mirror DB CHECK constraints; service uses these in
// validation so a 400 error fires before the DB does.
var (
	validBackupPolicyCompressionFormats = map[string]struct{}{
		"gzip": {}, "bzip2": {}, "lz4": {}, "zstd": {},
	}
	validBackupPolicyStorageBackends = map[string]struct{}{
		"local": {}, "ftp": {}, "sftp": {}, "nfs": {},
	}
	// EncryptionAlgorithm has no DB CHECK (the column is open-ended VARCHAR
	// so SecOps can register new algorithms without a migration), but the
	// service still rejects unknown values to avoid storing typos.
	validBackupPolicyEncryptionAlgorithms = map[string]struct{}{
		"AES-256-GCM":       {},
		"AES-256-CBC":       {},
		"ChaCha20-Poly1305": {},
	}
	// validBackupPolicyAlertSeverities mirrors the DB CHECK constraint on
	// backup_policies.alert_severity (T-0084). 3GPP 32.111 lists more
	// granular levels (minor/indeterminate/etc) but operators rarely need
	// them for backup; broaden the set in a future migration if needed.
	validBackupPolicyAlertSeverities = map[string]struct{}{
		"warning":  {},
		"major":    {},
		"critical": {},
	}
)

// SingletonPolicyID is the deterministic UUID used for the single backup
// policy row. Combined with INSERT ... ON CONFLICT (id) DO UPDATE in the
// repository, two concurrent first-time PUTs converge to one row instead of
// racing two INSERTs (review fix HIGH-1).
var SingletonPolicyID = uuid.MustParse("00000000-0000-0000-0000-0000000b0019")
