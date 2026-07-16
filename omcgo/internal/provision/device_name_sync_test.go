package provision

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// stubConfigLookup 用固定的 (category,key)->value 映射模拟 sys_configs 读取。
func stubConfigLookup(values map[string]string) NameSyncConfigLookup {
	return func(_ context.Context, category, key string) (string, bool) {
		v, ok := values[category+"."+key]
		return v, ok
	}
}

func newConfigOnlyHook(lookup NameSyncConfigLookup) *DeviceNameSyncHook {
	return &DeviceNameSyncHook{configLookup: lookup, logger: zap.NewNop()}
}

// #758 收尾 / rename P0：名称同步策略收敛为三值枚举（删除 off），
// loadConfig 应正确解析三值、缺失时默认 prompt、未知值退回 prompt。
func TestDeviceNameSync_LoadConfig_Mode(t *testing.T) {
	cases := []struct {
		name     string
		stored   string // 存储的 nameSyncMode 值（空串=不写该键）
		hasKey   bool
		wantMode string
	}{
		{"缺失默认prompt", "", false, NameSyncModePrompt},
		{"自动LMT覆盖网管", NameSyncModeAutoLMTToOMC, true, NameSyncModeAutoLMTToOMC},
		{"自动网管下发LMT", NameSyncModeAutoOMCToLMT, true, NameSyncModeAutoOMCToLMT},
		{"仅提示", NameSyncModePrompt, true, NameSyncModePrompt},
		{"未知值退回prompt", "garbage", true, NameSyncModePrompt},
		{"脏值off退回prompt", "off", true, NameSyncModePrompt},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]string{}
			if tc.hasKey {
				values["device.nameSyncMode"] = tc.stored
			}
			h := newConfigOnlyHook(stubConfigLookup(values))
			cfg := h.loadConfig(context.Background())
			assert.Equal(t, tc.wantMode, cfg.Mode)
		})
	}
}

// configLookup 为 nil（未注入配置源）时应安全退回 prompt，不 panic。
func TestDeviceNameSync_LoadConfig_NilLookup(t *testing.T) {
	h := newConfigOnlyHook(nil)
	cfg := h.loadConfig(context.Background())
	assert.Equal(t, NameSyncModePrompt, cfg.Mode)
}

func TestDeviceNameSync_AutoOMCToLMT_QueuesSPVWithStandardPathWhenTranslatorMissing(t *testing.T) {
	deviceID := uuid.New()
	infoRepo := &stubNameSyncUpdater{}
	sender := &stubNameSyncSender{}
	h := &DeviceNameSyncHook{
		infoRepo:  infoRepo,
		spvSender: sender,
		logger:    zap.NewNop(),
	}
	dev := &model.Device{ID: deviceID, SerialNumber: "SN-NAME-001"}

	err := h.handleOMCToLMT(context.Background(), dev, "LMT旧名", "网管新名", false)

	require.NoError(t, err)
	require.Len(t, sender.calls, 1)
	assert.Equal(t, "SN-NAME-001", sender.calls[0].sn)
	assert.Equal(t, hnbNameStandardPath, sender.calls[0].path)
	assert.Equal(t, "网管新名", sender.calls[0].name)
	assert.Equal(t, []stubNameSyncUpdate{{deviceID: deviceID, pending: false, lmtName: "LMT旧名"}}, infoRepo.updates)
}

func TestDeviceNameSync_AutoOMCToLMT_QueuesNRGNBNamePath(t *testing.T) {
	deviceID := uuid.New()
	infoRepo := &stubNameSyncUpdater{}
	sender := &stubNameSyncSender{}
	h := &DeviceNameSyncHook{
		infoRepo:  infoRepo,
		spvSender: sender,
		logger:    zap.NewNop(),
	}
	dev := &model.Device{ID: deviceID, SerialNumber: "SN-NR-001", Technology: model.TechNR}

	err := h.handleOMCToLMT(context.Background(), dev, "gNB旧名", "网管NR新名", false)

	require.NoError(t, err)
	require.Len(t, sender.calls, 1)
	assert.Equal(t, gnbNameStandardPath, sender.calls[0].path)
	assert.Equal(t, "网管NR新名", sender.calls[0].name)
	assert.Equal(t, []stubNameSyncUpdate{{deviceID: deviceID, pending: false, lmtName: "gNB旧名"}}, infoRepo.updates)
}

func TestDeviceNameSync_GetLMTDeviceName_ReadsNRGNBName(t *testing.T) {
	deviceID := uuid.New()
	h := &DeviceNameSyncHook{
		paramRepo: &stubNameParamRepo{
			params: []model.DeviceParameter{{
				DeviceID:       deviceID,
				ParameterPath:  "Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBName",
				ParameterValue: "gNB-设备侧名称",
			}},
		},
		logger: zap.NewNop(),
	}

	got, err := h.getLMTDeviceName(context.Background(), deviceID)

	require.NoError(t, err)
	assert.Equal(t, "gNB-设备侧名称", got)
}

type stubNameSyncUpdater struct {
	updates []stubNameSyncUpdate
}

type stubNameSyncUpdate struct {
	deviceID uuid.UUID
	pending  bool
	lmtName  string
}

func (s *stubNameSyncUpdater) UpdateNameSyncFields(_ context.Context, deviceID uuid.UUID, pending bool, lmtName string) error {
	s.updates = append(s.updates, stubNameSyncUpdate{deviceID: deviceID, pending: pending, lmtName: lmtName})
	return nil
}

func (s *stubNameSyncUpdater) UpdateDeviceName(context.Context, uuid.UUID, string) error {
	return nil
}

type stubNameSyncSender struct {
	calls []stubNameSyncSenderCall
}

type stubNameSyncSenderCall struct {
	sn   string
	path string
	name string
}

func (s *stubNameSyncSender) SendNameToDevice(_ context.Context, dev *model.Device, path, name string) error {
	s.calls = append(s.calls, stubNameSyncSenderCall{sn: dev.SerialNumber, path: path, name: name})
	return nil
}

type stubNameParamRepo struct {
	params []model.DeviceParameter
}

func (s *stubNameParamRepo) BatchUpsert(context.Context, uuid.UUID, []model.DeviceParameter) error {
	return nil
}

func (s *stubNameParamRepo) GetByDevice(context.Context, uuid.UUID) ([]model.DeviceParameter, error) {
	return nil, nil
}

func (s *stubNameParamRepo) GetByPath(_ context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error) {
	for _, p := range s.params {
		if p.DeviceID == deviceID && p.ParameterPath == path {
			cp := p
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *stubNameParamRepo) DeleteByDevice(context.Context, uuid.UUID) error {
	return nil
}

func (s *stubNameParamRepo) DeleteByPathPrefix(context.Context, uuid.UUID, string) (int64, error) {
	return 0, nil
}

func (s *stubNameParamRepo) GetByPathPrefix(_ context.Context, deviceID uuid.UUID, prefix string) ([]model.DeviceParameter, error) {
	var out []model.DeviceParameter
	for _, p := range s.params {
		if p.DeviceID == deviceID && strings.HasPrefix(p.ParameterPath, prefix) {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *stubNameParamRepo) CountByPathPrefix(context.Context, uuid.UUID, string) (int, error) {
	return 0, nil
}

func (s *stubNameParamRepo) SearchByKeyword(context.Context, uuid.UUID, string, int) ([]model.DeviceParameter, error) {
	return nil, nil
}

func (s *stubNameParamRepo) GetDirectChildLeaves(context.Context, uuid.UUID, string, int, int) ([]model.DeviceParameter, int, error) {
	return nil, 0, nil
}

func (s *stubNameParamRepo) GetByGroup(context.Context, uuid.UUID, string) ([]model.DeviceParameter, error) {
	return nil, nil
}

func (s *stubNameParamRepo) GetByFAPInstance(context.Context, uuid.UUID, int) ([]model.DeviceParameter, error) {
	return nil, nil
}

func (s *stubNameParamRepo) GetByFAPInstanceAndGroup(context.Context, uuid.UUID, int, string) ([]model.DeviceParameter, error) {
	return nil, nil
}
