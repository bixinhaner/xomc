package admin

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type builtInPermissionExec struct {
	sql  string
	args []any
}

type builtInPermissionTx struct {
	pgx.Tx
	execs       []builtInPermissionExec
	tags        []pgconn.CommandTag
	execErrAt   int
	commitCalls int
	rollback    int
}

func (tx *builtInPermissionTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tx.execs = append(tx.execs, builtInPermissionExec{sql: sql, args: args})
	if tx.execErrAt > 0 && len(tx.execs) == tx.execErrAt {
		return pgconn.CommandTag{}, errors.New("insert failed")
	}
	if len(tx.tags) < len(tx.execs) {
		return pgconn.NewCommandTag("INSERT 0 0"), nil
	}
	return tx.tags[len(tx.execs)-1], nil
}

func (tx *builtInPermissionTx) Commit(context.Context) error {
	tx.commitCalls++
	return nil
}

func (tx *builtInPermissionTx) Rollback(context.Context) error {
	tx.rollback++
	return nil
}

func TestReconcileBuiltInAPIPermissionsAddsBusinessBaselineAndRefreshesPolicy(t *testing.T) {
	tx := &builtInPermissionTx{tags: []pgconn.CommandTag{
		pgconn.NewCommandTag("INSERT 0 2"),
		pgconn.NewCommandTag("INSERT 0 3"),
		pgconn.NewCommandTag("INSERT 0 1"),
	}}
	refresher := &recordingPolicyRefresher{}
	repo := &PgRoleRepository{
		pool:            &policyTestDB{tx: tx},
		policyRefresher: refresher,
	}

	result, err := repo.ReconcileBuiltInAPIPermissions(context.Background())

	require.NoError(t, err)
	assert.Equal(t, BuiltInAPIPermissionGrantResult{Admin: 2, Operator: 3, Viewer: 1}, result)
	require.Len(t, tx.execs, 3)
	assert.Equal(t, 1, tx.commitCalls)
	assert.Equal(t, []string{"reload", "notify"}, refresher.calls)

	assert.Contains(t, tx.execs[0].args, builtInAdminRoleID)
	assert.Contains(t, tx.execs[1].args, builtInOperatorRoleID)
	assert.Contains(t, tx.execs[2].args, builtInViewerRoleID)
	assert.NotContains(t, strings.ToUpper(tx.execs[0].sql), "METHOD")
	assert.NotContains(t, strings.ToUpper(tx.execs[1].sql), "METHOD")
	assert.Contains(t, strings.ToUpper(tx.execs[2].sql), "METHOD")
	assert.Contains(t, tx.execs[2].args, "GET")
}

func TestReconcileBuiltInAPIPermissionsRefreshesPeersWhenSeedAlreadyCompletedBaseline(t *testing.T) {
	tx := &builtInPermissionTx{}
	refresher := &recordingPolicyRefresher{}
	repo := &PgRoleRepository{
		pool:            &policyTestDB{tx: tx},
		policyRefresher: refresher,
	}

	result, err := repo.ReconcileBuiltInAPIPermissions(context.Background())

	require.NoError(t, err)
	assert.Equal(t, BuiltInAPIPermissionGrantResult{}, result)
	assert.Equal(t, 1, tx.commitCalls)
	assert.Equal(t, []string{"reload", "notify"}, refresher.calls)
}

func TestReconcileBuiltInAPIPermissionsDoesNotRefreshAfterPersistenceFailure(t *testing.T) {
	tx := &builtInPermissionTx{execErrAt: 2}
	refresher := &recordingPolicyRefresher{}
	repo := &PgRoleRepository{
		pool:            &policyTestDB{tx: tx},
		policyRefresher: refresher,
	}

	_, err := repo.ReconcileBuiltInAPIPermissions(context.Background())

	require.ErrorContains(t, err, "operator")
	assert.Zero(t, tx.commitCalls)
	assert.Empty(t, refresher.calls)
}
