package device

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// newRenameTestService 创建仅含 rename 所需依赖的 DeviceService。
func newRenameTestService(ctrl *gomock.Controller, devRepo DeviceRepository, infoRepo DeviceInfoRepository, mode string) *DeviceService {
	svc := NewDeviceService(devRepo, nil, nil, nil, zap.NewNop())
	svc.SetDeviceInfoRepo(infoRepo)
	svc.SetSysConfigLookup(func(_ context.Context, _, _ string) (string, bool) {
		if mode == "" {
			return "", false
		}
		return mode, true
	})
	return svc
}

func TestRenameDevice_AutoLMTToOMC_Rejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	devRepo := NewMockDeviceRepository(ctrl)
	infoRepo := NewMockDeviceInfoRepository(ctrl)
	svc := newRenameTestService(ctrl, devRepo, infoRepo, "auto_lmt_to_omc")

	err := svc.RenameDevice(context.Background(), uuid.New(), "新名称")

	require.Error(t, err)
	var bizErr *commonerrors.BusinessError
	assert.True(t, errors.As(err, &bizErr))
	assert.Equal(t, global.ErrCodeDeviceRenameNotAllowed, bizErr.Code)
	assert.True(t, errors.Is(err, commonerrors.ErrForbidden))
}

func TestRenameDevice_Prompt_SetsPendingWithOldLMTName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deviceID := uuid.New()
	devRepo := NewMockDeviceRepository(ctrl)
	infoRepo := NewMockDeviceInfoRepository(ctrl)
	svc := newRenameTestService(ctrl, devRepo, infoRepo, "prompt")

	dev := &model.Device{ID: deviceID, SerialNumber: "TEST001", DeviceName: "旧名称"}
	info := &DeviceInfo{DeviceName: "旧名称", LMTDeviceName: "基站侧旧名"}

	devRepo.EXPECT().GetByID(gomock.Any(), deviceID).Return(dev, nil)
	infoRepo.EXPECT().GetByDeviceID(gomock.Any(), deviceID).Return(info, nil)
	infoRepo.EXPECT().UpdateDeviceName(gomock.Any(), deviceID, "新名称").Return(nil)
	devRepo.EXPECT().UpdateSiteName(gomock.Any(), deviceID, "新名称").Return(nil)
	// prompt 模式：置 pending=true，lmtName 必须是旧值 "基站侧旧名"
	infoRepo.EXPECT().UpdateNameSyncFields(gomock.Any(), deviceID, true, "基站侧旧名").Return(nil)

	err := svc.RenameDevice(context.Background(), deviceID, "新名称")
	assert.NoError(t, err)
}

func TestRenameDevice_Prompt_DoesNotDispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deviceID := uuid.New()
	devRepo := NewMockDeviceRepository(ctrl)
	infoRepo := NewMockDeviceInfoRepository(ctrl)
	// taskSvc 未注入 → 验证没有任务被创建
	svc := newRenameTestService(ctrl, devRepo, infoRepo, "prompt")

	dev := &model.Device{ID: deviceID, SerialNumber: "TEST002", DeviceName: "旧名"}
	info := &DeviceInfo{DeviceName: "旧名", LMTDeviceName: "lmt旧名"}

	devRepo.EXPECT().GetByID(gomock.Any(), deviceID).Return(dev, nil)
	infoRepo.EXPECT().GetByDeviceID(gomock.Any(), deviceID).Return(info, nil)
	infoRepo.EXPECT().UpdateDeviceName(gomock.Any(), deviceID, "新名").Return(nil)
	devRepo.EXPECT().UpdateSiteName(gomock.Any(), deviceID, "新名").Return(nil)
	infoRepo.EXPECT().UpdateNameSyncFields(gomock.Any(), deviceID, true, "lmt旧名").Return(nil)

	err := svc.RenameDevice(context.Background(), deviceID, "新名")
	assert.NoError(t, err)
}

func TestRenameDevice_AutoOMCToLMT_ClearsPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deviceID := uuid.New()
	devRepo := NewMockDeviceRepository(ctrl)
	infoRepo := NewMockDeviceInfoRepository(ctrl)
	svc := newRenameTestService(ctrl, devRepo, infoRepo, "auto_omc_to_lmt")
	// taskSvc 未注入，SPV 分支静默跳过（不影响主流程）

	dev := &model.Device{ID: deviceID, SerialNumber: "TEST003", DeviceName: "旧名"}
	info := &DeviceInfo{DeviceName: "旧名", LMTDeviceName: "lmt旧名"}

	devRepo.EXPECT().GetByID(gomock.Any(), deviceID).Return(dev, nil)
	infoRepo.EXPECT().GetByDeviceID(gomock.Any(), deviceID).Return(info, nil)
	infoRepo.EXPECT().UpdateDeviceName(gomock.Any(), deviceID, "新名").Return(nil)
	devRepo.EXPECT().UpdateSiteName(gomock.Any(), deviceID, "新名").Return(nil)
	// auto_omc_to_lmt：清 pending=false，lmtName 传旧值
	infoRepo.EXPECT().UpdateNameSyncFields(gomock.Any(), deviceID, false, "lmt旧名").Return(nil)

	err := svc.RenameDevice(context.Background(), deviceID, "新名")
	assert.NoError(t, err)
}

func TestRenameDevice_FailedSPV_DoesNotRollback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 验证：SPV 下发失败时双写已完成（不回滚），pending 清除也继续执行
	deviceID := uuid.New()
	devRepo := NewMockDeviceRepository(ctrl)
	infoRepo := NewMockDeviceInfoRepository(ctrl)
	svc := newRenameTestService(ctrl, devRepo, infoRepo, "auto_omc_to_lmt")
	// 注入一个必定失败的 mock taskSvc
	mockTask := &stubFailTaskSvc{}
	svc.taskSvc = mockTask

	dev := &model.Device{ID: deviceID, SerialNumber: "TEST004", DeviceName: "旧名"}
	info := &DeviceInfo{DeviceName: "旧名", LMTDeviceName: "lmt旧名"}

	devRepo.EXPECT().GetByID(gomock.Any(), deviceID).Return(dev, nil)
	infoRepo.EXPECT().GetByDeviceID(gomock.Any(), deviceID).Return(info, nil)
	// 双写必须执行（不回滚）
	infoRepo.EXPECT().UpdateDeviceName(gomock.Any(), deviceID, "新名").Return(nil)
	devRepo.EXPECT().UpdateSiteName(gomock.Any(), deviceID, "新名").Return(nil)
	// pending 清除仍继续
	infoRepo.EXPECT().UpdateNameSyncFields(gomock.Any(), deviceID, false, "lmt旧名").Return(nil)

	err := svc.RenameDevice(context.Background(), deviceID, "新名")
	// 下发失败不影响整体返回
	assert.NoError(t, err)
}

func TestRenameDevice_DualWrite_BothTables(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	deviceID := uuid.New()
	devRepo := NewMockDeviceRepository(ctrl)
	infoRepo := NewMockDeviceInfoRepository(ctrl)
	svc := newRenameTestService(ctrl, devRepo, infoRepo, "prompt")

	dev := &model.Device{ID: deviceID, SerialNumber: "TEST005", DeviceName: "老名"}
	info := &DeviceInfo{DeviceName: "老名", LMTDeviceName: "lmt老名"}

	devRepo.EXPECT().GetByID(gomock.Any(), deviceID).Return(dev, nil)
	infoRepo.EXPECT().GetByDeviceID(gomock.Any(), deviceID).Return(info, nil)

	// 验证两张表都写了
	infoRepo.EXPECT().UpdateDeviceName(gomock.Any(), deviceID, "新名").Return(nil).Times(1)
	devRepo.EXPECT().UpdateSiteName(gomock.Any(), deviceID, "新名").Return(nil).Times(1)
	infoRepo.EXPECT().UpdateNameSyncFields(gomock.Any(), deviceID, true, "lmt老名").Return(nil)

	err := svc.RenameDevice(context.Background(), deviceID, "新名")
	assert.NoError(t, err)
}

func TestRenameDevice_EmptyName_Rejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := newRenameTestService(ctrl, NewMockDeviceRepository(ctrl), NewMockDeviceInfoRepository(ctrl), "prompt")
	err := svc.RenameDevice(context.Background(), uuid.New(), "  ")
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

// stubFailTaskSvc 模拟 SPV 下发总是失败的 taskSvc。
type stubFailTaskSvc struct{}

func (s *stubFailTaskSvc) CreateTask(_ context.Context, _ *task.CreateTaskRequest) (*task.Task, error) {
	return nil, errors.New("network unreachable")
}

func (s *stubFailTaskSvc) GetQueueLength(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
