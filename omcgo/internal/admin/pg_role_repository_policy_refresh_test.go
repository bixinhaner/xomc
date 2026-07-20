package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingPolicyRefresher struct {
	calls     []string
	reloadErr error
	notifyErr error
}

func (r *recordingPolicyRefresher) ReloadPolicy() error {
	r.calls = append(r.calls, "reload")
	return r.reloadErr
}

func (r *recordingPolicyRefresher) NotifyPolicyChange() error {
	r.calls = append(r.calls, "notify")
	return r.notifyErr
}

type policyTestDB struct {
	storage.DB
	execTag  pgconn.CommandTag
	execErr  error
	tx       pgx.Tx
	beginErr error
}

func (d *policyTestDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return d.execTag, d.execErr
}

func (d *policyTestDB) Begin(context.Context) (pgx.Tx, error) {
	return d.tx, d.beginErr
}

type policyTestTx struct {
	pgx.Tx
	calls     []string
	execErr   error
	commitErr error
}

func (tx *policyTestTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.calls = append(tx.calls, "exec")
	return pgconn.NewCommandTag("INSERT 0 1"), tx.execErr
}

func (tx *policyTestTx) Commit(context.Context) error {
	tx.calls = append(tx.calls, "commit")
	return tx.commitErr
}

func (tx *policyTestTx) Rollback(context.Context) error {
	tx.calls = append(tx.calls, "rollback")
	return nil
}

func TestPgRoleRepositoryPolicyWritesRefreshCurrentThenPeers(t *testing.T) {
	userID, roleID, endpointID := uuid.New(), uuid.New(), uuid.New()
	cases := []struct {
		name string
		run  func(*PgRoleRepository) error
		db   *policyTestDB
	}{
		{
			name: "assign role",
			run: func(repo *PgRoleRepository) error {
				return repo.AssignRole(context.Background(), userID, roleID)
			},
			db: &policyTestDB{execTag: pgconn.NewCommandTag("INSERT 0 1")},
		},
		{
			name: "remove role",
			run: func(repo *PgRoleRepository) error {
				return repo.RemoveRole(context.Background(), userID, roleID)
			},
			db: &policyTestDB{execTag: pgconn.NewCommandTag("DELETE 1")},
		},
		{
			name: "set role endpoints",
			run: func(repo *PgRoleRepository) error {
				return repo.SetRoleApiEndpoints(context.Background(), roleID, []uuid.UUID{endpointID})
			},
			db: func() *policyTestDB {
				tx := &policyTestTx{}
				return &policyTestDB{tx: tx}
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refresher := &recordingPolicyRefresher{}
			repo := &PgRoleRepository{pool: tc.db, policyRefresher: refresher}

			require.NoError(t, tc.run(repo))
			assert.Equal(t, []string{"reload", "notify"}, refresher.calls)
		})
	}
}

func TestPgRoleRepositoryPolicyRefreshErrorsAreReturned(t *testing.T) {
	t.Run("reload failure stops notify", func(t *testing.T) {
		refresher := &recordingPolicyRefresher{reloadErr: errors.New("reload failed")}
		repo := &PgRoleRepository{
			pool:            &policyTestDB{execTag: pgconn.NewCommandTag("INSERT 0 1")},
			policyRefresher: refresher,
		}

		err := repo.AssignRole(context.Background(), uuid.New(), uuid.New())

		require.ErrorContains(t, err, "persisted")
		require.ErrorContains(t, err, "reload")
		assert.Equal(t, []string{"reload"}, refresher.calls)
	})

	t.Run("notify failure is returned", func(t *testing.T) {
		refresher := &recordingPolicyRefresher{notifyErr: errors.New("publish failed")}
		repo := &PgRoleRepository{
			pool:            &policyTestDB{execTag: pgconn.NewCommandTag("INSERT 0 1")},
			policyRefresher: refresher,
		}

		err := repo.AssignRole(context.Background(), uuid.New(), uuid.New())

		require.ErrorContains(t, err, "persisted")
		require.ErrorContains(t, err, "notify")
		assert.Equal(t, []string{"reload", "notify"}, refresher.calls)
	})
}

func TestPgRoleRepositoryPolicyDoesNotRefreshBeforePersistence(t *testing.T) {
	userID, roleID, endpointID := uuid.New(), uuid.New(), uuid.New()
	cases := []struct {
		name string
		run  func(*PgRoleRepository) error
		db   *policyTestDB
	}{
		{
			name: "assign role database mutation failure",
			run: func(repo *PgRoleRepository) error {
				return repo.AssignRole(context.Background(), userID, roleID)
			},
			db: &policyTestDB{execErr: errors.New("insert failed")},
		},
		{
			name: "set role endpoints begin failure",
			run: func(repo *PgRoleRepository) error {
				return repo.SetRoleApiEndpoints(context.Background(), roleID, []uuid.UUID{endpointID})
			},
			db: &policyTestDB{beginErr: errors.New("begin failed")},
		},
		{
			name: "set role endpoints transaction exec failure",
			run: func(repo *PgRoleRepository) error {
				return repo.SetRoleApiEndpoints(context.Background(), roleID, []uuid.UUID{endpointID})
			},
			db: &policyTestDB{tx: &policyTestTx{execErr: errors.New("delete failed")}},
		},
		{
			name: "set role endpoints commit failure",
			run: func(repo *PgRoleRepository) error {
				return repo.SetRoleApiEndpoints(context.Background(), roleID, []uuid.UUID{endpointID})
			},
			db: &policyTestDB{tx: &policyTestTx{commitErr: errors.New("commit failed")}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refresher := &recordingPolicyRefresher{}
			repo := &PgRoleRepository{pool: tc.db, policyRefresher: refresher}

			require.Error(t, tc.run(repo))
			assert.Empty(t, refresher.calls)
		})
	}
}

func TestPgRoleRepositoryPolicyRefresherIsIndependentOfPermissionChecking(t *testing.T) {
	refresher := &recordingPolicyRefresher{}
	repo := &PgRoleRepository{policyRefresher: refresher}

	require.NoError(t, repo.refreshPolicyAfterPersist())
	assert.Equal(t, []string{"reload", "notify"}, refresher.calls)
}
