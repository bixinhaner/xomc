package backup

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// PolicyRequest is the JSON body for PUT /backup/policy. Mirrors BackupPolicy
// minus server-managed fields (id / timestamps).
//
// `binding:"required"` is intentionally absent on int fields: gin's required
// rule rejects zero, which would generate a confusing 400 message that doesn't
// match the service-layer validatePolicy() vocabulary. validatePolicy is the
// single source of truth for value-range errors. (Review fix MEDIUM-5.)
type PolicyRequest struct {
	RetentionDays         int        `json:"retention_days"`
	MaxBackupCount        int        `json:"max_backup_count"`
	MinBackupCount        int        `json:"min_backup_count"`
	AutoCleanup           bool       `json:"auto_cleanup"`
	CleanupTime           string     `json:"cleanup_time"`
	CleanupDayOfWeek      int        `json:"cleanup_day_of_week"`
	KeepLastN             int        `json:"keep_last_n"`
	EnableCompression     bool       `json:"enable_compression"`
	CompressionLevel      int        `json:"compression_level"`
	CompressionFormat     string     `json:"compression_format"`
	StorageBackend        string     `json:"storage_backend"`
	FTPConfigID           *uuid.UUID `json:"ftp_config_id,omitempty"`
	LocalPath             string     `json:"local_path"`
	MaxStorageGB          int        `json:"max_storage_gb"`
	EnableEncryption      bool       `json:"enable_encryption"`
	EncryptionAlgorithm   string     `json:"encryption_algorithm"`
	AlertOnFailure        bool       `json:"alert_on_failure"`
	AlertEmail            string     `json:"alert_email"`
	AlertThresholdPercent int        `json:"alert_threshold_percent"`
	AlertSeverity         string     `json:"alert_severity"` // T-0084: warning|major|critical
}

func (req PolicyRequest) toModel() *BackupPolicy {
	return &BackupPolicy{
		RetentionDays:         req.RetentionDays,
		MaxBackupCount:        req.MaxBackupCount,
		MinBackupCount:        req.MinBackupCount,
		AutoCleanup:           req.AutoCleanup,
		CleanupTime:           req.CleanupTime,
		CleanupDayOfWeek:      req.CleanupDayOfWeek,
		KeepLastN:             req.KeepLastN,
		EnableCompression:     req.EnableCompression,
		CompressionLevel:      req.CompressionLevel,
		CompressionFormat:     req.CompressionFormat,
		StorageBackend:        req.StorageBackend,
		FTPConfigID:           req.FTPConfigID,
		LocalPath:             req.LocalPath,
		MaxStorageGB:          req.MaxStorageGB,
		EnableEncryption:      req.EnableEncryption,
		EncryptionAlgorithm:   req.EncryptionAlgorithm,
		AlertOnFailure:        req.AlertOnFailure,
		AlertEmail:            req.AlertEmail,
		AlertThresholdPercent: req.AlertThresholdPercent,
		AlertSeverity:         req.AlertSeverity,
	}
}

// GetPolicy handles GET /api/v1/backup/policy.
// Returns the singleton policy, or DefaultPolicy() when the table is empty
// so first-time callers see canonical defaults.
func (h *Handler) GetPolicy(c *gin.Context) {
	if h.policyService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, commonerrors.ErrInternal)
		return
	}
	policy, err := h.policyService.Get(c.Request.Context())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, policy)
}

// UpdatePolicy handles PUT /api/v1/backup/policy.
// Validates and upserts the singleton policy.
func (h *Handler) UpdatePolicy(c *gin.Context) {
	if h.policyService == nil {
		commonerrors.AbortWithError(c, http.StatusServiceUnavailable, commonerrors.ErrInternal)
		return
	}
	var req PolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	policy, err := h.policyService.Update(c.Request.Context(), req.toModel())
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	response.OK(c, policy)
}
