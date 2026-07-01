package provision

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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

// #758 收尾：名称同步策略改为单字段 nameSyncMode（四值枚举），
// loadConfig 应正确解析四值、缺失时默认 off、未知值退回 off。
func TestDeviceNameSync_LoadConfig_Mode(t *testing.T) {
	cases := []struct {
		name     string
		stored   string // 存储的 nameSyncMode 值（空串=不写该键）
		hasKey   bool
		wantMode string
	}{
		{"缺失默认off", "", false, NameSyncModeOff},
		{"不处理", NameSyncModeOff, true, NameSyncModeOff},
		{"自动LMT覆盖网管", NameSyncModeAutoLMTToOMC, true, NameSyncModeAutoLMTToOMC},
		{"自动网管下发LMT", NameSyncModeAutoOMCToLMT, true, NameSyncModeAutoOMCToLMT},
		{"仅提示", NameSyncModePrompt, true, NameSyncModePrompt},
		{"未知值退回off", "garbage", true, NameSyncModeOff},
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

// configLookup 为 nil（未注入配置源）时应安全退回 off，不 panic。
func TestDeviceNameSync_LoadConfig_NilLookup(t *testing.T) {
	h := newConfigOnlyHook(nil)
	cfg := h.loadConfig(context.Background())
	assert.Equal(t, NameSyncModeOff, cfg.Mode)
}
