package backup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var policyColumns = []string{
	"id",
	"retention_days", "max_backup_count", "min_backup_count",
	"auto_cleanup", "cleanup_time", "cleanup_day_of_week", "keep_last_n",
	"enable_compression", "compression_level", "compression_format",
	"storage_backend", "ftp_config_id", "local_path", "max_storage_gb",
	"enable_encryption", "encryption_algorithm",
	"alert_on_failure", "alert_email", "alert_threshold_percent", "alert_severity",
	"created_at", "updated_at",
}

var _ PolicyRepository = (*PgPolicyRepository)(nil)

// PgPolicyRepository is a PostgreSQL implementation of PolicyRepository.
type PgPolicyRepository struct {
	pool *pgxpool.Pool
}

// NewPgPolicyRepository creates a new PgPolicyRepository.
func NewPgPolicyRepository(pool *pgxpool.Pool) *PgPolicyRepository {
	return &PgPolicyRepository{pool: pool}
}

func scanPolicy(row pgx.Row) (*BackupPolicy, error) {
	var p BackupPolicy
	var ftpID sql.NullString
	if err := row.Scan(
		&p.ID,
		&p.RetentionDays, &p.MaxBackupCount, &p.MinBackupCount,
		&p.AutoCleanup, &p.CleanupTime, &p.CleanupDayOfWeek, &p.KeepLastN,
		&p.EnableCompression, &p.CompressionLevel, &p.CompressionFormat,
		&p.StorageBackend, &ftpID, &p.LocalPath, &p.MaxStorageGB,
		&p.EnableEncryption, &p.EncryptionAlgorithm,
		&p.AlertOnFailure, &p.AlertEmail, &p.AlertThresholdPercent, &p.AlertSeverity,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if ftpID.Valid {
		if id, err := uuid.Parse(ftpID.String); err == nil {
			p.FTPConfigID = &id
		}
	}
	return &p, nil
}

// Get returns the most recently updated policy row, or commonerrors.ErrNotFound
// when the table is empty. Singleton enforcement happens here: callers always
// get exactly one row even if multiple exist (administrator edge case).
func (r *PgPolicyRepository) Get(ctx context.Context) (*BackupPolicy, error) {
	query, args, err := storage.Psql.Select(policyColumns...).
		From("backup_policies").
		OrderBy("updated_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get policy SQL: %w", err)
	}
	policy, err := scanPolicy(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get backup policy: %w", err)
	}
	return policy, nil
}

// Upsert atomically inserts the singleton row (PK = SingletonPolicyID) or
// updates it on conflict. Replaces the prior Get-then-INSERT/UPDATE pattern
// which had a TOCTOU race: two concurrent first-time PUTs could both observe
// ErrNotFound and both INSERT, breaking the singleton invariant. ON CONFLICT
// (id) DO UPDATE is a single statement and acquires the row lock atomically.
// (Review fix HIGH-1.)
func (r *PgPolicyRepository) Upsert(ctx context.Context, p *BackupPolicy) error {
	if p == nil {
		return fmt.Errorf("upsert nil policy: %w", commonerrors.ErrInvalidInput)
	}

	q := `
		INSERT INTO backup_policies (
			id,
			retention_days, max_backup_count, min_backup_count,
			auto_cleanup, cleanup_time, cleanup_day_of_week, keep_last_n,
			enable_compression, compression_level, compression_format,
			storage_backend, ftp_config_id, local_path, max_storage_gb,
			enable_encryption, encryption_algorithm,
			alert_on_failure, alert_email, alert_threshold_percent, alert_severity
		) VALUES (
			$1,
			$2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15,
			$16, $17,
			$18, $19, $20, $21
		)
		ON CONFLICT (id) DO UPDATE SET
			retention_days          = EXCLUDED.retention_days,
			max_backup_count        = EXCLUDED.max_backup_count,
			min_backup_count        = EXCLUDED.min_backup_count,
			auto_cleanup            = EXCLUDED.auto_cleanup,
			cleanup_time            = EXCLUDED.cleanup_time,
			cleanup_day_of_week     = EXCLUDED.cleanup_day_of_week,
			keep_last_n             = EXCLUDED.keep_last_n,
			enable_compression      = EXCLUDED.enable_compression,
			compression_level       = EXCLUDED.compression_level,
			compression_format      = EXCLUDED.compression_format,
			storage_backend         = EXCLUDED.storage_backend,
			ftp_config_id           = EXCLUDED.ftp_config_id,
			local_path              = EXCLUDED.local_path,
			max_storage_gb          = EXCLUDED.max_storage_gb,
			enable_encryption       = EXCLUDED.enable_encryption,
			encryption_algorithm    = EXCLUDED.encryption_algorithm,
			alert_on_failure        = EXCLUDED.alert_on_failure,
			alert_email             = EXCLUDED.alert_email,
			alert_threshold_percent = EXCLUDED.alert_threshold_percent,
			alert_severity          = EXCLUDED.alert_severity,
			updated_at              = NOW()
		RETURNING ` + strings.Join(policyColumns, ", ")

	row := r.pool.QueryRow(ctx, q,
		SingletonPolicyID,
		p.RetentionDays, p.MaxBackupCount, p.MinBackupCount,
		p.AutoCleanup, p.CleanupTime, p.CleanupDayOfWeek, p.KeepLastN,
		p.EnableCompression, p.CompressionLevel, p.CompressionFormat,
		p.StorageBackend, ftpConfigArg(p.FTPConfigID), p.LocalPath, p.MaxStorageGB,
		p.EnableEncryption, p.EncryptionAlgorithm,
		p.AlertOnFailure, p.AlertEmail, p.AlertThresholdPercent, p.AlertSeverity,
	)
	upserted, err := scanPolicy(row)
	if err != nil {
		return fmt.Errorf("upsert backup policy: %w", err)
	}
	*p = *upserted
	return nil
}

// ftpConfigArg returns nil for SQL NULL when the policy has no FTP config
// reference, otherwise the underlying uuid value.
func ftpConfigArg(id *uuid.UUID) any {
	if id == nil {
		return nil
	}
	return *id
}
