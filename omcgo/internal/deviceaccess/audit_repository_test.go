package deviceaccess

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/require"
)

func TestArchiveDecisionRejectsCurrentEffectiveDecisionWithoutWrite(t *testing.T) {
	tx := decisionArchiveTestTx(t, 7, 7)
	store := &PgManagementStore{db: &repositoryTestDB{tx: tx}}

	err := store.ArchiveDecision(
		context.Background(), "cmcc", uuid.New(), uuid.New(), nil, "hide duplicate history",
	)

	require.ErrorIs(t, err, ErrCurrentDecisionArchive)
	require.Empty(t, tx.execSQL)
	require.False(t, tx.committed)
	require.True(t, tx.rolledBack)
}

func TestArchiveDecisionPersistsOnlyHistoricalDecision(t *testing.T) {
	tx := decisionArchiveTestTx(t, 6, 7)
	store := &PgManagementStore{db: &repositoryTestDB{tx: tx}}

	err := store.ArchiveDecision(
		context.Background(), "cmcc", uuid.New(), uuid.New(), nil, "obsolete result",
	)

	require.NoError(t, err)
	require.True(t, tx.committed)
	require.Len(t, tx.execSQL, 1)
	require.Contains(t, tx.execSQL[0], "INSERT INTO device_access_decision_archives")
}

func TestRestoreDecisionUpdatesOnlyActiveArchive(t *testing.T) {
	tx := decisionArchiveTestTx(t, 6, 7)
	store := &PgManagementStore{db: &repositoryTestDB{tx: tx}}

	err := store.RestoreDecision(
		context.Background(), "cmcc", uuid.New(), uuid.New(), nil,
	)

	require.NoError(t, err)
	require.True(t, tx.committed)
	require.Len(t, tx.execSQL, 1)
	require.Contains(t, tx.execSQL[0], "UPDATE device_access_decision_archives")
	require.Contains(t, tx.execSQL[0], "restored_at IS NULL")
}

func TestDecisionFiltersDefaultHidesArchivedAndCanSelectArchived(t *testing.T) {
	defaultQuery, _, err := storage.Psql.Select("COUNT(*)").From("device_access_decisions decision").
		Where(decisionFilters(ManagementFilter{Carrier: "cmcc"})).ToSql()
	require.NoError(t, err)
	require.Contains(t, defaultQuery, "NOT EXISTS")

	archivedQuery, _, err := storage.Psql.Select("COUNT(*)").From("device_access_decisions decision").
		Where(decisionFilters(ManagementFilter{Carrier: "cmcc", ArchiveStatus: "archived"})).ToSql()
	require.NoError(t, err)
	require.Contains(t, archivedQuery, "EXISTS")
	require.NotContains(t, archivedQuery, "NOT EXISTS")
}

func decisionArchiveTestTx(t *testing.T, decisionVersion, currentVersion int64) *repositoryTestTx {
	t.Helper()
	return &repositoryTestTx{queryRow: func(query string, _ ...any) pgx.Row {
		require.Contains(t, query, "FOR UPDATE OF decision")
		return repositoryTestRow{scan: func(dest ...any) error {
			*dest[0].(*int64) = decisionVersion
			*dest[1].(*int64) = currentVersion
			return nil
		}}
	}}
}
