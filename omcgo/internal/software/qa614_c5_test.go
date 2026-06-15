package software

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// newExecutorForBackfill 构造一个只装配 backfillDestVersion 所需依赖的 UpgradeExecutor。
// backfillDestVersion 只用 deviceRepo / subTaskRepo / logger，其余可留 nil。
func newExecutorForBackfill(dev *svcMockDeviceRepo, sub *svcMockSubTaskRepo) *UpgradeExecutor {
	return &UpgradeExecutor{
		deviceRepo:  dev,
		subTaskRepo: sub,
		logger:      zap.NewNop(),
	}
}

// qa-614 #371：完成后回填目标版本——子任务 DestVersion 为空时，用设备最新上报的固件版本回填。
func TestBackfillDestVersion_FillsFromDeviceWhenEmpty(t *testing.T) {
	const sn = "SN-5G-ROLLBACK-001"
	const newVer = "BNQ_V2.1.0"

	var gotID uuid.UUID
	var gotVer string
	subRepo := &svcMockSubTaskRepo{
		updateDestByIDFn: func(_ context.Context, id uuid.UUID, destVersion string) error {
			gotID = id
			gotVer = destVersion
			return nil
		},
	}
	devRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn, FirmwareVersion: newVer}, nil
		},
	}

	e := newExecutorForBackfill(devRepo, subRepo)
	subTask := &UpgradeSubTask{ID: uuid.New(), DeviceSN: sn, DestVersion: ""}

	e.backfillDestVersion(context.Background(), subTask, sn)

	assert.Equal(t, subTask.ID, gotID, "应按子任务 ID 回填")
	assert.Equal(t, newVer, gotVer, "应回填设备上报的新固件版本")
	assert.Equal(t, newVer, subTask.DestVersion, "内存中的子任务也应同步更新")
}

// qa-614 #371：创建时已显式指定目标固件版本的，完成后不被设备版本覆盖。
func TestBackfillDestVersion_KeepsExplicitTarget(t *testing.T) {
	const sn = "SN-5G-ROLLBACK-002"
	called := false
	subRepo := &svcMockSubTaskRepo{
		updateDestByIDFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			called = true
			return nil
		},
	}
	devRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn, FirmwareVersion: "DEVICE_REPORTED"}, nil
		},
	}

	e := newExecutorForBackfill(devRepo, subRepo)
	subTask := &UpgradeSubTask{ID: uuid.New(), DeviceSN: sn, DestVersion: "EXPLICIT_TARGET"}

	e.backfillDestVersion(context.Background(), subTask, sn)

	assert.False(t, called, "DestVersion 已非空不应再回填")
	assert.Equal(t, "EXPLICIT_TARGET", subTask.DestVersion)
}

// qa-614 #371：设备未上报固件版本（或查不到设备）时不写空值、不报错。
func TestBackfillDestVersion_NoOpWhenDeviceVersionEmpty(t *testing.T) {
	const sn = "SN-5G-ROLLBACK-003"
	called := false
	subRepo := &svcMockSubTaskRepo{
		updateDestByIDFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			called = true
			return nil
		},
	}
	devRepo := &svcMockDeviceRepo{
		getBySerialNumberFn: func(_ context.Context, _ string) (*model.Device, error) {
			return &model.Device{SerialNumber: sn, FirmwareVersion: ""}, nil
		},
	}

	e := newExecutorForBackfill(devRepo, subRepo)
	subTask := &UpgradeSubTask{ID: uuid.New(), DeviceSN: sn}

	e.backfillDestVersion(context.Background(), subTask, sn)

	assert.False(t, called, "设备版本为空时不应写库")
	assert.Empty(t, subTask.DestVersion)
}

// qa-614 #372/#379：固件重复导入命中唯一索引 23505 → 翻成 ErrAlreadyExists；
// 即便上层 service 再 wrap 一层，errors.Is 仍穿透命中，HTTPStatusFromError 映射 409。
func TestFirmwareDuplicate_MapsToAlreadyExists_409(t *testing.T) {
	// repository 层翻译后的哨兵错误（pg_firmware_repository.go Create 中的等价构造）。
	repoErr := fmt.Errorf("%w: 该产品类型+版本+文件类型的固件已存在，请勿重复导入或修改版本号", commonerrors.ErrAlreadyExists)
	// service.UploadFirmware 的二次包装（service.go:235）。
	svcErr := fmt.Errorf("create firmware record: %w", repoErr)

	require.True(t, stderrors.Is(svcErr, commonerrors.ErrAlreadyExists),
		"二次包装后 errors.Is 仍应命中 ErrAlreadyExists")
	assert.Equal(t, http.StatusConflict, commonerrors.HTTPStatusFromError(svcErr),
		"应映射为 HTTP 409 Conflict")
}
