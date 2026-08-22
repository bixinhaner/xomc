package parammodel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/appconfig"
)

func paramModelIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		t.Skip("TEST_PG_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("create PG pool: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("connect PG: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestLoader_RecountsPersistedStatsInsteadOfTrustingXMLTotalEntries(t *testing.T) {
	pool := paramModelIntegrationPool(t)
	ctx := context.Background()
	modelID := uuid.New()
	modelName := "TEST-LOADER-STATS-" + modelID.String()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM param_models WHERE name = $1`, modelName)
	})

	baseDir := t.TempDir()
	mappingDir := filepath.Join(baseDir, "param-mappings")
	require.NoError(t, os.MkdirAll(mappingDir, 0o755))
	xmlPath := filepath.Join(mappingDir, "test-loader-stats.xml")
	xmlBody := fmt.Sprintf(`<?xml version="1.0"?>
<parameterModel paramModel="%s" totalEntries="999">
  <objects>
    <object name="Device.Test." standardPath="Device.Test." access="READ_ONLY"/>
  </objects>
  <parameters>
    <param name="Device.Private.Param" standardPath="Device.Test.Param" access="READ_WRITE" type="STRING"/>
  </parameters>
</parameterModel>`, modelName)
	require.NoError(t, os.WriteFile(xmlPath, []byte(xmlBody), 0o600))

	loader := NewLoader(pool, appconfig.ParamModelLoaderConfig{}, baseDir, zap.NewNop())
	_, loadedModel, err := loader.loadParamModelFile(ctx, xmlPath, false)
	require.NoError(t, err)
	assert.Equal(t, modelName, loadedModel)

	var entries, objects, params int
	err = pool.QueryRow(ctx, `
SELECT total_entries, total_objects, total_params
  FROM param_models
 WHERE name = $1`, modelName).Scan(&entries, &objects, &params)
	require.NoError(t, err)
	assert.Equal(t, 2, entries)
	assert.Equal(t, 1, objects)
	assert.Equal(t, 1, params)
}

func TestLoaderReloadPreservesInactiveExistingModelAndActivatesNewModel(t *testing.T) {
	pool := paramModelIntegrationPool(t)
	ctx := context.Background()
	suffix := uuid.New().String()
	existingName := "TEST-INACTIVE-" + suffix
	newName := "TEST-NEW-" + suffix
	standardPath := "Device.Test.LoaderReload." + suffix + "."

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM param_models WHERE name = ANY($1::text[])`, []string{existingName, newName})
		_, _ = pool.Exec(context.Background(), `DELETE FROM standard_params WHERE standard_path = $1`, standardPath)
	})

	_, err := pool.Exec(ctx, `
INSERT INTO param_models (id, name, total_entries, total_objects, total_params, is_active, loaded_from)
VALUES ($1, $2, 0, 0, 0, FALSE, $3)`, uuid.New(), existingName, "param-mappings/"+existingName+".xml")
	require.NoError(t, err)

	baseDir := t.TempDir()
	mappingDir := filepath.Join(baseDir, "param-mappings")
	require.NoError(t, os.MkdirAll(mappingDir, 0o755))

	existingXML := fmt.Sprintf(`<?xml version="1.0"?>
<parameterModel paramModel="%s">
  <parameters>
    <param name="%s" standardPath="%s" access="READ_WRITE" type="STRING"/>
  </parameters>
</parameterModel>`, existingName, standardPath, standardPath)
	newXML := fmt.Sprintf(`<?xml version="1.0"?>
<parameterModel paramModel="%s">
  <parameters>
    <param name="%s" standardPath="%s" access="READ_WRITE" type="STRING"/>
  </parameters>
</parameterModel>`, newName, standardPath, standardPath+"New")
	standardXML := fmt.Sprintf(`<?xml version="1.0"?>
<standardModel>
  <parameters>
    <param standardPath="%s" access="READ_WRITE" type="STRING"/>
  </parameters>
</standardModel>`, standardPath)
	require.NoError(t, os.WriteFile(filepath.Join(mappingDir, existingName+".xml"), []byte(existingXML), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(mappingDir, newName+".xml"), []byte(newXML), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(mappingDir, "standard-model.xml"), []byte(standardXML), 0o600))

	loader := NewLoader(pool, appconfig.ParamModelLoaderConfig{}, baseDir, zap.NewNop())
	_, err = loader.Reload(ctx)
	require.NoError(t, err)

	var existingActive, newActive bool
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT is_active FROM param_models WHERE name = $1`, existingName).Scan(&existingActive))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT is_active FROM param_models WHERE name = $1`, newName).Scan(&newActive))
	assert.False(t, existingActive, "reload must preserve an existing model's manual inactive state")
	assert.True(t, newActive, "a newly imported model should be active by default")
}

func TestPgRepository_ModelStatsFollowActiveMappings(t *testing.T) {
	pool := paramModelIntegrationPool(t)
	ctx := context.Background()
	repo := NewPgRepository(pool)
	modelID := uuid.New()
	modelName := "TEST-STATS-" + modelID.String()

	_, err := pool.Exec(ctx, `
INSERT INTO param_models (id, name, total_entries, total_objects, total_params, is_active, loaded_from)
VALUES ($1, $2, 999, 333, 666, TRUE, 'param-mappings-custom/test-stats.xml')`, modelID, modelName)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM param_models WHERE id = $1`, modelID)
	})

	_, err = pool.Exec(ctx, `
INSERT INTO param_mappings
    (id, param_model_id, standard_path, private_path, entry_type, is_storable, is_active, is_supported, source)
VALUES
    ($1, $2, 'Device.Test.', 'Device.Private.', 'object', TRUE, TRUE, TRUE, 'custom'),
    ($3, $2, 'Device.Hidden', 'Device.Private.Hidden', 'parameter', TRUE, FALSE, TRUE, 'custom')`,
		uuid.New(), modelID, uuid.New())
	require.NoError(t, err)

	assertStats := func(entries, objects, params int) {
		t.Helper()
		model, getErr := repo.GetParamModelByName(ctx, modelName)
		require.NoError(t, getErr)
		assert.Equal(t, entries, model.TotalEntries)
		assert.Equal(t, objects, model.TotalObjects)
		assert.Equal(t, params, model.TotalParams)
		assert.Equal(t, model.TotalObjects+model.TotalParams, model.TotalEntries)

		models, listErr := repo.ListParamModels(ctx)
		require.NoError(t, listErr)
		for _, listed := range models {
			if listed.ID == modelID {
				assert.Equal(t, *model, listed)
				return
			}
		}
		t.Fatalf("model %s missing from list", modelName)
	}

	assertStats(1, 1, 0)
	created, err := repo.CreateMapping(ctx, modelID, CreateMappingInput{
		StandardPath: "Device.Test.Param",
		PrivatePath:  "Device.Private.Param",
		EntryType:    "parameter",
		IsStorable:   true,
	})
	require.NoError(t, err)
	assertStats(2, 1, 1)

	deleted, err := repo.DeleteMapping(ctx, created.ID)
	require.NoError(t, err)
	require.True(t, deleted)
	assertStats(1, 1, 0)

	_, err = repo.GetParamModelByName(ctx, fmt.Sprintf("missing-%s", modelID))
	assert.ErrorIs(t, err, ErrNoParamModel)
}
