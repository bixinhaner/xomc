package backup

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"

	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// cleanupTimeRe enforces "HH:MM" 24-hour format. Service layer rejects malformed
// input before it reaches the DB CHECK constraint to surface 400 ErrInvalidInput.
var cleanupTimeRe = regexp.MustCompile(`^(?:[01][0-9]|2[0-3]):[0-5][0-9]$`)

// PolicyService is the service-layer wrapper around PolicyRepository. Lives on
// its own struct (not folded into Service) so DI wiring can inject the repo
// without touching the existing task/schedule constructor signature.
type PolicyService struct {
	repo   PolicyRepository
	logger *zap.Logger
	// T-0075: optional KeyProvider — when wired, validatePolicy rejects
	// EnableEncryption=true if the algorithm requires a KEK and the provider
	// reports Available()=false. nil-safe: nil means encryption availability
	// is not enforced at PUT time (UI Alert remains the only signal).
	keyProvider KeyProvider
}

// NewPolicyService constructs a PolicyService.
func NewPolicyService(repo PolicyRepository, logger *zap.Logger) *PolicyService {
	return &PolicyService{repo: repo, logger: logger.Named("backup-policy")}
}

// SetKeyProvider wires a backup-encryption KeyProvider for T-0075 validation.
// Pass nil to disable PUT-time KEK availability checks (UI/operator picks up
// the slack via the persisted-warning Tag). Constructed separately from
// NewPolicyService to keep the signature stable for existing call sites.
func (s *PolicyService) SetKeyProvider(kp KeyProvider) {
	s.keyProvider = kp
}

// Get returns the persisted policy or DefaultPolicy() when none exists.
// Callers always receive a non-nil pointer with valid defaults, simplifying
// frontend bootstrap.
func (s *PolicyService) Get(ctx context.Context) (*BackupPolicy, error) {
	policy, err := s.repo.Get(ctx)
	if errors.Is(err, commonerrors.ErrNotFound) {
		return DefaultPolicy(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("get backup policy: %w", err)
	}
	return policy, nil
}

// Update validates and upserts the policy. Validation mirrors DB CHECK
// constraints so a 400 fires *before* the round-trip to PG, giving clearer
// error messages.
func (s *PolicyService) Update(ctx context.Context, p *BackupPolicy) (*BackupPolicy, error) {
	if p == nil {
		return nil, fmt.Errorf("nil policy: %w", commonerrors.ErrInvalidInput)
	}
	if err := s.validatePolicy(p); err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, p); err != nil {
		return nil, fmt.Errorf("upsert backup policy: %w", err)
	}
	s.logger.Info("backup policy updated",
		zap.String("policy_id", p.ID.String()),
		zap.String("storage_backend", p.StorageBackend),
		zap.Bool("auto_cleanup", p.AutoCleanup),
		zap.Bool("enable_compression", p.EnableCompression),
		zap.Bool("enable_encryption", p.EnableEncryption),
		zap.Bool("alert_on_failure", p.AlertOnFailure),
	)
	return p, nil
}

// validatePolicy is the package-level legacy entry point retained for tests
// that exercise validation without a *PolicyService instance. The
// PolicyService method (s.validatePolicy below) wraps this with KeyProvider
// availability checks (T-0075).
func (s *PolicyService) validatePolicy(p *BackupPolicy) error {
	if err := validatePolicy(p); err != nil {
		return err
	}
	// T-0075: encryption activation requires a usable KEK. AES-256-CBC and
	// ChaCha20-Poly1305 are persisted but not yet implemented (T-0085 stub
	// rejection mirrors T-0074 lz4/bzip2 pattern).
	if p.EnableEncryption {
		switch p.EncryptionAlgorithm {
		case "AES-256-GCM":
			if s.keyProvider != nil && !s.keyProvider.Available() {
				return fmt.Errorf("encryption_algorithm=AES-256-GCM enabled but encryption key not configured (set %s env var to a 64-hex-char string): %w",
					EnvBackupEncryptionKey, commonerrors.ErrInvalidInput)
			}
		case "AES-256-CBC", "ChaCha20-Poly1305":
			return fmt.Errorf("encryption_algorithm=%s not yet implemented (暂未支持); choose AES-256-GCM, or set enable_encryption=false (请选择 AES-256-GCM 或关闭加密): %w",
				p.EncryptionAlgorithm, commonerrors.ErrInvalidInput)
		}
	}
	return nil
}

// validatePolicy enforces the same value ranges as the DB CHECK constraints,
// plus a few cross-field rules (e.g., cleanup_time format, alert_email shape
// when alert_on_failure is true).
func validatePolicy(p *BackupPolicy) error {
	if p.RetentionDays < 1 || p.RetentionDays > 3650 {
		return fmt.Errorf("retention_days out of range [1, 3650]: %w", commonerrors.ErrInvalidInput)
	}
	if p.MaxBackupCount < 1 {
		return fmt.Errorf("max_backup_count must be >= 1: %w", commonerrors.ErrInvalidInput)
	}
	if p.MinBackupCount < 1 {
		return fmt.Errorf("min_backup_count must be >= 1: %w", commonerrors.ErrInvalidInput)
	}
	if p.MinBackupCount > p.MaxBackupCount {
		return fmt.Errorf("min_backup_count cannot exceed max_backup_count: %w", commonerrors.ErrInvalidInput)
	}
	if p.CleanupDayOfWeek < -1 || p.CleanupDayOfWeek > 6 {
		return fmt.Errorf("cleanup_day_of_week out of range [-1, 6]: %w", commonerrors.ErrInvalidInput)
	}
	if !cleanupTimeRe.MatchString(p.CleanupTime) {
		return fmt.Errorf("cleanup_time must be HH:MM 24h: %w", commonerrors.ErrInvalidInput)
	}
	if p.KeepLastN < 1 {
		return fmt.Errorf("keep_last_n must be >= 1: %w", commonerrors.ErrInvalidInput)
	}
	if p.CompressionLevel < 1 || p.CompressionLevel > 9 {
		return fmt.Errorf("compression_level out of range [1, 9]: %w", commonerrors.ErrInvalidInput)
	}
	if _, ok := validBackupPolicyCompressionFormats[p.CompressionFormat]; !ok {
		return fmt.Errorf("invalid compression_format %q: %w", p.CompressionFormat, commonerrors.ErrInvalidInput)
	}
	// T-0077: lz4 + bzip2 are now first-class implementations (alongside gzip
	// and zstd from T-0074). The previous "reject when EnableCompression=true"
	// guard was removed because all four formats produce real compressed
	// streams now.
	if _, ok := validBackupPolicyStorageBackends[p.StorageBackend]; !ok {
		return fmt.Errorf("invalid storage_backend %q: %w", p.StorageBackend, commonerrors.ErrInvalidInput)
	}
	// Cross-field consistency for storage backend (review fix HIGH-4):
	// local → local_path must be non-empty; ftp/sftp → ftp_config_id required.
	switch p.StorageBackend {
	case "local":
		if p.LocalPath == "" {
			return fmt.Errorf("local_path required when storage_backend=local: %w", commonerrors.ErrInvalidInput)
		}
	case "ftp", "sftp":
		if p.FTPConfigID == nil {
			return fmt.Errorf("ftp_config_id required when storage_backend=%s: %w", p.StorageBackend, commonerrors.ErrInvalidInput)
		}
	}
	if _, ok := validBackupPolicyEncryptionAlgorithms[p.EncryptionAlgorithm]; !ok {
		return fmt.Errorf("invalid encryption_algorithm %q: %w", p.EncryptionAlgorithm, commonerrors.ErrInvalidInput)
	}
	if p.MaxStorageGB < 1 {
		return fmt.Errorf("max_storage_gb must be >= 1: %w", commonerrors.ErrInvalidInput)
	}
	if p.AlertThresholdPercent < 50 || p.AlertThresholdPercent > 95 {
		return fmt.Errorf("alert_threshold_percent out of range [50, 95]: %w", commonerrors.ErrInvalidInput)
	}
	if p.AlertOnFailure && p.AlertEmail != "" {
		if _, err := mail.ParseAddress(p.AlertEmail); err != nil {
			return fmt.Errorf("alert_email invalid: %w", commonerrors.ErrInvalidInput)
		}
	}
	return nil
}
