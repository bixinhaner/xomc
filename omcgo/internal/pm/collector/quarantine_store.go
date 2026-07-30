package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

const technologyMismatchReason = "technology_mismatch"

type QuarantineRecord struct {
	SourceFileID       uuid.UUID
	DeviceSN           string
	DeclaredTechnology string
	DetectedTechnology string
	Reason             string
	MinIOPath          string
	Evidence           []string
}

type QuarantineStore interface {
	Save(ctx context.Context, record QuarantineRecord) (bool, error)
}

type PgQuarantineStore struct {
	pool *pgxpool.Pool
}

func NewPgQuarantineStore(pool *pgxpool.Pool) *PgQuarantineStore {
	return &PgQuarantineStore{pool: pool}
}

func (s *PgQuarantineStore) Save(ctx context.Context, record QuarantineRecord) (bool, error) {
	evidence, err := json.Marshal(record.Evidence)
	if err != nil {
		return false, fmt.Errorf("marshal PM quarantine evidence: %w", err)
	}
	query, args, err := storage.Psql.Insert("pm_file_quarantines").
		Columns(
			"source_file_id", "device_sn", "declared_technology",
			"detected_technology", "reason", "minio_path", "evidence",
		).
		Values(
			record.SourceFileID, record.DeviceSN, record.DeclaredTechnology,
			record.DetectedTechnology, record.Reason, record.MinIOPath, json.RawMessage(evidence),
		).
		Suffix("ON CONFLICT (source_file_id, reason) DO NOTHING").
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build PM quarantine insert: %w", err)
	}
	tag, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("insert PM quarantine: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func quarantineSourceFileID(payload *FileReceivedPayload) uuid.UUID {
	sourceIdentity := strings.TrimSpace(payload.DeviceSN) + "\x1f" + strings.TrimSpace(payload.MinIOPath)
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(sourceIdentity))
}
