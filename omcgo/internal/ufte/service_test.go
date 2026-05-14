package ufte

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/software"
)

type ensureBuiltInTaskTypeRepo struct {
	items    []TaskType
	upserted []TaskType
}

func (r *ensureBuiltInTaskTypeRepo) List(_ context.Context) ([]TaskType, error) {
	result := make([]TaskType, len(r.items))
	copy(result, r.items)
	return result, nil
}

func (r *ensureBuiltInTaskTypeRepo) GetByCode(_ context.Context, typeCode string) (*TaskType, error) {
	for i := range r.items {
		if r.items[i].TypeCode == typeCode {
			item := r.items[i]
			return &item, nil
		}
	}
	return nil, nil
}

func (r *ensureBuiltInTaskTypeRepo) Upsert(_ context.Context, item *TaskType) error {
	r.upserted = append(r.upserted, *item)
	r.items = append(r.items, *item)
	return nil
}

func TestService_EnsureBuiltInTaskTypes_InsertsOnlyMissingDefaults(t *testing.T) {
	defaults := builtInTaskTypes()
	repo := &ensureBuiltInTaskTypeRepo{
		items: []TaskType{
			{
				TypeCode:      defaults[0].TypeCode,
				Category:      defaults[0].Category,
				CategoryLabel: defaults[0].CategoryLabel,
				DisplayName:   "自定义后的 4G 升级名称",
				BuiltIn:       true,
			},
		},
	}
	svc := NewService(nil, repo, nil, nil, nil, zap.NewNop())

	inserted, err := svc.EnsureBuiltInTaskTypes(context.Background())
	require.NoError(t, err)
	assert.Equal(t, len(defaults)-1, inserted)
	require.Len(t, repo.upserted, len(defaults)-1)
	for _, item := range repo.upserted {
		assert.NotEqual(t, defaults[0].TypeCode, item.TypeCode)
		assert.True(t, item.BuiltIn)
	}
	assert.Equal(t, "自定义后的 4G 升级名称", repo.items[0].DisplayName)
	assert.Equal(t, len(defaults), len(repo.items))
}

func TestBuiltInTaskTypes_CoversRequiredTemplates(t *testing.T) {
	defaults := builtInTaskTypes()
	required := map[string]struct{}{
		"ENB_IMG_UPGRADE":      {},
		"GNB_IMG_UPGRADE":      {},
		"VERSION_ROLLBACK":     {},
		"RUNTIME_LOG_COLLECT":  {},
		"FAULT_LOG_COLLECT":    {},
		"CONFIG_BACKUP":        {},
		"CONFIG_RESTORE":       {},
	}

	seen := make(map[string]TaskType, len(defaults))
	for _, item := range defaults {
		seen[item.TypeCode] = item
	}

	for code := range required {
		item, ok := seen[code]
		require.Truef(t, ok, "required built-in task type %s should exist", code)
		assert.Truef(t, item.BuiltIn, "required built-in task type %s should be marked built-in", code)
		assert.NotEmptyf(t, item.Category, "required built-in task type %s should have category", code)
		assert.NotEmptyf(t, item.DisplayName, "required built-in task type %s should have display name", code)
	}

	assert.Equal(t, "station_log", seen["RUNTIME_LOG_COLLECT"].Category)
	assert.Equal(t, "config_backup", seen["CONFIG_BACKUP"].Category)
	assert.Equal(t, "config_restore", seen["CONFIG_RESTORE"].Category)
}

func TestService_LoadTaskTypeCatalog_UsesStoredRowsAsSourceOfTruth(t *testing.T) {
	repo := &ensureBuiltInTaskTypeRepo{
		items: []TaskType{
			{
				TypeCode:         "ENB_IMG_UPGRADE",
				Category:         "enb_upgrade",
				CategoryLabel:    "4G升级",
				DisplayName:      "库里的 4G 升级模板",
				Description:      "from db",
				RPCType:          "DOWNLOAD",
				BuiltIn:          true,
				Enabled:          true,
				StepChain:        []string{"CHECK_PERMISSION", "SEND_RPC"},
				PermissionCode:   "CODE_ENB_UPGRADE_IMAGE",
				PlatformScope:    []string{"4G eNB"},
				FileType:         "1",
				FileTypeLabel:    "Firmware Upgrade Image",
				LastEditor:       "system",
			},
			{
				TypeCode:         "CUSTOM_UPLOAD_SAMPLE",
				Category:         "custom_upload",
				CategoryLabel:    "自定义上传",
				DisplayName:      "自定义上传模板",
				Description:      "custom",
				RPCType:          "UPLOAD",
				BuiltIn:          false,
				Enabled:          true,
				StepChain:        []string{"SEND_RPC", "WAIT_RPC_RESPONSE"},
				PermissionCode:   "CODE_CUSTOM_UPLOAD_SAMPLE",
				PlatformScope:    []string{"5G gNB"},
				FileType:         "9",
				FileTypeLabel:    "Custom Upload",
				LastEditor:       "tester",
			},
		},
	}
	svc := NewService(nil, repo, nil, nil, nil, zap.NewNop())

	catalog, err := svc.loadTaskTypeCatalog(context.Background())
	require.NoError(t, err)
	require.Len(t, catalog, 2)

	first, ok := findTaskTypeByCode(catalog, "ENB_IMG_UPGRADE")
	require.True(t, ok)
	assert.Equal(t, "库里的 4G 升级模板", first.DisplayName)
	assert.Equal(t, software.TaskTypeUpgrade, first.softwareTaskType)
	require.NotNil(t, first.techHint)

	custom, ok := findTaskTypeByCode(catalog, "CUSTOM_UPLOAD_SAMPLE")
	require.True(t, ok)
	assert.Equal(t, "自定义上传模板", custom.DisplayName)
	assert.Zero(t, custom.softwareTaskType)
	assert.Nil(t, custom.techHint)
}