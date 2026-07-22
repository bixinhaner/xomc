package mml

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type blockingUpdateCustomCommandRepo struct {
	CustomCommandRepository
	entered chan struct{}
	proceed chan struct{}
}

func (r *blockingUpdateCustomCommandRepo) Update(ctx context.Context, cmd *MMLCustomCommand) error {
	close(r.entered)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.proceed:
		return r.CustomCommandRepository.Update(ctx, cmd)
	}
}

const (
	customPathString = "Device.DeviceInfo.SAS.UserId"
	customPathInt    = "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.Common.Qhyst"
)

func newCustomPathIntegrationEnv(t *testing.T) (*Service, *pgxpool.Pool, uuid.UUID, string) {
	t.Helper()
	pool := newMMLTestPool(t)
	ctx := context.Background()

	var ownerID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, "SELECT id FROM users ORDER BY created_at LIMIT 1").Scan(&ownerID))
	for _, path := range []string{customPathString, customPathInt} {
		var exists bool
		require.NoError(t, pool.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM standard_params WHERE standard_path=$1)", path,
		).Scan(&exists))
		require.True(t, exists, "standard path fixture missing: %s", path)
	}

	name := "itest_custom_paths_" + uuid.NewString()[:8]
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM mml_custom_command WHERE command_name=$1", name)
	})

	svc := NewService(
		&mockCommandRepo{},
		&mockScriptRepo{},
		&mockTaskRepo{},
		NewPgCustomCommandRepository(pool),
		nil,
		zap.NewNop(),
	)
	svc.SetCustomCommandPathRepo(NewPgCustomCommandPathRepository(pool))
	return svc, pool, ownerID, name
}

func TestIntegration_CustomCommandCreateAndUpdate_SynchronizesEnrichedPaths(t *testing.T) {
	svc, _, ownerID, name := newCustomPathIntegrationEnv(t)
	ctx := context.Background()

	created, err := svc.CreateCustomCommand(ctx, &MMLCustomCommand{
		CommandName:   name,
		CommandCode:   "MOD ITEST PATHS",
		OperationType: "MOD",
		CommandScope:  "private",
		Creator:       "itest",
		OwnerUserID:   &ownerID,
		ParamPaths:    []string{customPathString, customPathInt},
	})
	require.NoError(t, err)

	paths, err := svc.ListCustomCommandPaths(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, paths, 2)
	assert.Equal(t, customPathString, paths[0].StandardPath)
	assert.Equal(t, "STRING", paths[0].DataType)
	assert.Equal(t, customPathInt, paths[1].StandardPath)
	assert.Equal(t, "INT", paths[1].DataType)

	updated, err := svc.UpdateCustomCommand(ctx, created.ID, &MMLCustomCommand{
		CommandName:   created.CommandName,
		CommandCode:   created.CommandCode,
		OperationType: created.OperationType,
		CommandScope:  created.CommandScope,
		ParamPaths:    []string{customPathString},
	}, ownerID, "itest", false)
	require.NoError(t, err)
	require.Equal(t, []string{customPathString}, updated.ParamPaths)

	paths, err = svc.ListCustomCommandPaths(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, paths, 1)
	assert.Equal(t, customPathString, paths[0].StandardPath)
}

func TestIntegration_CustomCommandPaths_FallsBackToLegacyParamPaths(t *testing.T) {
	svc, pool, ownerID, name := newCustomPathIntegrationEnv(t)
	ctx := context.Background()

	created, err := svc.CreateCustomCommand(ctx, &MMLCustomCommand{
		CommandName:   name,
		CommandCode:   "MOD ITEST LEGACY",
		OperationType: "MOD",
		CommandScope:  "private",
		Creator:       "itest",
		OwnerUserID:   &ownerID,
		ParamPaths:    []string{customPathString},
	})
	require.NoError(t, err)

	_, err = pool.Exec(
		ctx,
		`UPDATE mml_custom_command
		 SET param_paths = jsonb_build_array(
		   E'\t' || $2::text || E'\n',
		   E'\r' || $2::text || E'\t',
		   E'\t\r\n',
		   E'\r\n' || $3::text || E'\t',
		   E'\t' || $3::text || E'\n'
		 )
		 WHERE id = $1`,
		created.ID,
		customPathString,
		customPathInt,
	)
	require.NoError(t, err)
	_, err = pool.Exec(
		ctx,
		"DELETE FROM mml_custom_command_paths WHERE command_id=$1",
		created.ID,
	)
	require.NoError(t, err)

	paths, err := svc.ListCustomCommandPaths(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, paths, 2)
	assert.Equal(t, customPathString, paths[0].StandardPath)
	assert.Equal(t, "STRING", paths[0].DataType)
	assert.False(t, paths[0].Mutable)
	assert.Equal(t, customPathInt, paths[1].StandardPath)
	assert.False(t, paths[1].Mutable)
}

func TestIntegration_CustomCommandPathMutations_KeepJSONAndAssociationsConsistent(t *testing.T) {
	svc, pool, ownerID, name := newCustomPathIntegrationEnv(t)
	ctx := context.Background()

	created, err := svc.CreateCustomCommand(ctx, &MMLCustomCommand{
		CommandName:   name,
		CommandCode:   "MOD ITEST PATH CRUD",
		OperationType: "MOD",
		CommandScope:  "private",
		Creator:       "itest",
		OwnerUserID:   &ownerID,
		ParamPaths:    []string{customPathString},
	})
	require.NoError(t, err)

	var intStandardPathID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT id FROM standard_params WHERE standard_path=$1",
		customPathInt,
	).Scan(&intStandardPathID))

	pathRepo := svc.customCommandPathRepo.(*PgCustomCommandPathRepository)
	added, err := pathRepo.BatchCreate(ctx, created.ID, []uuid.UUID{intStandardPathID})
	require.NoError(t, err)
	require.Len(t, added, 1)

	stored, err := svc.customCommandRepo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{customPathString, customPathInt}, stored.ParamPaths)

	paths, err := pathRepo.ListByCommand(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, paths, 2)
	assert.True(t, paths[0].Mutable)
	assert.True(t, paths[1].Mutable)

	intAssociation := paths[1]
	selected := true
	first := -1
	updated, err := pathRepo.Update(
		ctx,
		created.ID,
		intAssociation.ID,
		&selected,
		&first,
	)
	require.NoError(t, err)
	assert.True(t, updated.DefaultSelected)

	stored, err = svc.customCommandRepo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{customPathInt, customPathString}, stored.ParamPaths)

	paths, err = pathRepo.ListByCommand(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, paths, 2)
	assert.Equal(t, customPathInt, paths[0].StandardPath)
	assert.True(t, paths[0].DefaultSelected)

	require.NoError(t, pathRepo.Delete(ctx, created.ID, paths[0].ID))
	stored, err = svc.customCommandRepo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{customPathString}, stored.ParamPaths)

	paths, err = pathRepo.ListByCommand(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, paths, 1)
	assert.Equal(t, customPathString, paths[0].StandardPath)
	assert.True(t, paths[0].Mutable)
}

func TestIntegration_CustomCommandUpdate_InvalidPathRollsBackAndRetainsSelection(t *testing.T) {
	svc, _, ownerID, name := newCustomPathIntegrationEnv(t)
	ctx := context.Background()

	created, err := svc.CreateCustomCommand(ctx, &MMLCustomCommand{
		CommandName:   name,
		CommandCode:   "MOD ITEST PATH ROLLBACK",
		OperationType: "MOD",
		CommandScope:  "private",
		Creator:       "itest",
		OwnerUserID:   &ownerID,
		ParamPaths:    []string{customPathString},
	})
	require.NoError(t, err)

	pathRepo := svc.customCommandPathRepo.(*PgCustomCommandPathRepository)
	paths, err := pathRepo.ListByCommand(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, paths, 1)

	selected := true
	_, err = pathRepo.Update(ctx, created.ID, paths[0].ID, &selected, nil)
	require.NoError(t, err)

	_, err = svc.UpdateCustomCommand(ctx, created.ID, &MMLCustomCommand{
		CommandName:   created.CommandName,
		CommandCode:   created.CommandCode,
		OperationType: created.OperationType,
		CommandScope:  created.CommandScope,
		ParamPaths:    []string{customPathString, "Device.Missing.Standard.Path"},
	}, ownerID, "itest", false)
	require.Error(t, err)

	stored, err := svc.customCommandRepo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{customPathString}, stored.ParamPaths)

	paths, err = pathRepo.ListByCommand(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, paths, 1)
	assert.Equal(t, customPathString, paths[0].StandardPath)
	assert.True(t, paths[0].DefaultSelected)
}

func TestIntegration_CustomCommandUpdate_OmittedPathsDoesNotLoseConcurrentPathMutation(t *testing.T) {
	svc, pool, ownerID, name := newCustomPathIntegrationEnv(t)
	ctx := context.Background()

	created, err := svc.CreateCustomCommand(ctx, &MMLCustomCommand{
		CommandName:   name,
		CommandCode:   "MOD ITEST PATH CONCURRENCY",
		OperationType: "MOD",
		CommandScope:  "private",
		Creator:       "itest",
		OwnerUserID:   &ownerID,
		ParamPaths:    []string{customPathString},
	})
	require.NoError(t, err)

	var intStandardPathID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		"SELECT id FROM standard_params WHERE standard_path=$1",
		customPathInt,
	).Scan(&intStandardPathID))

	baseRepo := svc.customCommandRepo
	blockingRepo := &blockingUpdateCustomCommandRepo{
		CustomCommandRepository: baseRepo,
		entered:                 make(chan struct{}),
		proceed:                 make(chan struct{}),
	}
	svc.customCommandRepo = blockingRepo

	updateDone := make(chan error, 1)
	go func() {
		_, updateErr := svc.UpdateCustomCommand(ctx, created.ID, &MMLCustomCommand{
			CommandName:   created.CommandName,
			CommandCode:   created.CommandCode,
			OperationType: created.OperationType,
			CommandScope:  created.CommandScope,
			Description:   "metadata-only update",
			ParamPaths:    nil,
		}, ownerID, "itest", false)
		updateDone <- updateErr
	}()

	<-blockingRepo.entered
	pathRepo := svc.customCommandPathRepo.(*PgCustomCommandPathRepository)
	added, err := pathRepo.BatchCreate(ctx, created.ID, []uuid.UUID{intStandardPathID})
	require.NoError(t, err)
	require.Len(t, added, 1)
	close(blockingRepo.proceed)
	require.NoError(t, <-updateDone)

	stored, err := baseRepo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{customPathString, customPathInt}, stored.ParamPaths)

	paths, err := pathRepo.ListByCommand(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, paths, 2)
	assert.Equal(t, customPathString, paths[0].StandardPath)
	assert.Equal(t, customPathInt, paths[1].StandardPath)
}
