package deviceaccess

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// RuntimeSettings is the operator-scoped business switch from the legacy OMC.
// It is the only switch controlling whether an operator's devices enter the
// access-control flow; changing it takes effect without a deployment restart.
type RuntimeSettings struct {
	Carrier   string    `json:"carrier"`
	Enabled   bool      `json:"enabled"`
	UpdatedBy string    `json:"updated_by,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type RuntimeSettingsReader interface {
	GetRuntimeSettings(ctx context.Context, carrier string) (RuntimeSettings, error)
}

type RuntimeSettingsStore interface {
	RuntimeSettingsReader
	UpdateRuntimeSettings(ctx context.Context, carrier string, enabled bool, updatedBy string) (RuntimeSettings, error)
}

type PgRuntimeSettingsStore struct {
	db storage.DB
}

func NewPgRuntimeSettingsStore(pool *pgxpool.Pool) *PgRuntimeSettingsStore {
	return newPgRuntimeSettingsStoreWithDB(storage.NewPoolDB(pool))
}

func newPgRuntimeSettingsStoreWithDB(db storage.DB) *PgRuntimeSettingsStore {
	return &PgRuntimeSettingsStore{db: db}
}

func (s *PgRuntimeSettingsStore) GetRuntimeSettings(ctx context.Context, carrier string) (RuntimeSettings, error) {
	carrier = strings.ToLower(strings.TrimSpace(carrier))
	if carrier == "" {
		return RuntimeSettings{}, ErrCarrierRequired
	}
	if s == nil || s.db == nil {
		return RuntimeSettings{}, fmt.Errorf("get device access runtime settings: %w", ErrAccessGateDependencyMissing)
	}
	query, args, err := storage.Psql.
		Select("carrier", "enabled", "COALESCE(updated_by, '')", "updated_at").
		From("device_access_runtime_settings").
		Where(sq.Eq{"carrier": carrier}).
		ToSql()
	if err != nil {
		return RuntimeSettings{}, fmt.Errorf("build device access runtime settings query: %w", err)
	}
	var settings RuntimeSettings
	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&settings.Carrier, &settings.Enabled, &settings.UpdatedBy, &settings.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return RuntimeSettings{Carrier: carrier, Enabled: false}, nil
		}
		return RuntimeSettings{}, fmt.Errorf("get device access runtime settings: %w", err)
	}
	return settings, nil
}

func (s *PgRuntimeSettingsStore) UpdateRuntimeSettings(
	ctx context.Context,
	carrier string,
	enabled bool,
	updatedBy string,
) (RuntimeSettings, error) {
	carrier = strings.ToLower(strings.TrimSpace(carrier))
	updatedBy = strings.TrimSpace(updatedBy)
	if carrier == "" {
		return RuntimeSettings{}, ErrCarrierRequired
	}
	if updatedBy == "" {
		return RuntimeSettings{}, fmt.Errorf("update device access runtime settings: actor is required")
	}
	if s == nil || s.db == nil {
		return RuntimeSettings{}, fmt.Errorf("update device access runtime settings: %w", ErrAccessGateDependencyMissing)
	}
	query, args, err := storage.Psql.
		Insert("device_access_runtime_settings").
		Columns("carrier", "enabled", "updated_by", "updated_at").
		Values(carrier, enabled, updatedBy, sq.Expr("now()")).
		Suffix(`ON CONFLICT (carrier) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			updated_by = EXCLUDED.updated_by,
			updated_at = EXCLUDED.updated_at
			RETURNING carrier, enabled, updated_by, updated_at`).
		ToSql()
	if err != nil {
		return RuntimeSettings{}, fmt.Errorf("build device access runtime settings update: %w", err)
	}
	var settings RuntimeSettings
	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&settings.Carrier, &settings.Enabled, &settings.UpdatedBy, &settings.UpdatedAt,
	); err != nil {
		return RuntimeSettings{}, fmt.Errorf("update device access runtime settings: %w", err)
	}
	return settings, nil
}

func runtimeAccessEnabled(ctx context.Context, reader RuntimeSettingsReader, carrier string) (bool, error) {
	if reader == nil {
		return true, nil
	}
	settings, err := reader.GetRuntimeSettings(ctx, carrier)
	if err != nil {
		return false, err
	}
	return settings.Enabled, nil
}

var _ RuntimeSettingsStore = (*PgRuntimeSettingsStore)(nil)
