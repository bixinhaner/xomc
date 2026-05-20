package software

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUpgradeTransition(t *testing.T) {
	tests := []struct {
		name    string
		current UpgradeState
		target  UpgradeState
		wantErr bool
	}{
		{"pending to downloading", UpgradePending, UpgradeDownloading, false},
		{"pending to failed", UpgradePending, UpgradeFailed, false},
		{"downloading to rebooting", UpgradeDownloading, UpgradeRebooting, false},
		{"downloading to failed", UpgradeDownloading, UpgradeFailed, false},
		{"rebooting to verifying", UpgradeRebooting, UpgradeVerifying, false},
		{"rebooting to failed", UpgradeRebooting, UpgradeFailed, false},
		{"verifying to completed", UpgradeVerifying, UpgradeCompleted, false},
		{"verifying to failed", UpgradeVerifying, UpgradeFailed, false},

		// Upload RPC（LogCollect 类）专用转换：Pending → Uploading → Completed，
		// 中间可被挂起 / 失败 / 终止。无 Rebooting / Verifying（备份/日志没有重启 + 校验环节）。
		{"pending to uploading", UpgradePending, UpgradeUploading, false},
		{"uploading to completed", UpgradeUploading, UpgradeCompleted, false},
		{"uploading to failed", UpgradeUploading, UpgradeFailed, false},
		{"uploading to suspended", UpgradeUploading, UpgradeSuspended, false},
		{"uploading to terminated", UpgradeUploading, UpgradeTerminated, false},
		{"suspended to uploading", UpgradeSuspended, UpgradeUploading, false},

		// Invalid transitions
		{"pending to completed", UpgradePending, UpgradeCompleted, true},
		{"pending to verifying", UpgradePending, UpgradeVerifying, true},
		{"downloading to completed", UpgradeDownloading, UpgradeCompleted, false},
		{"completed to pending", UpgradeCompleted, UpgradePending, true},
		{"failed to pending", UpgradeFailed, UpgradePending, true},
		{"completed to downloading", UpgradeCompleted, UpgradeDownloading, true},
		// Upload 不应该走 Rebooting / Verifying——这两个状态是 5G 升级专用的安装后阶段。
		{"uploading to rebooting", UpgradeUploading, UpgradeRebooting, true},
		{"uploading to verifying", UpgradeUploading, UpgradeVerifying, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpgradeTransition(tt.current, tt.target)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsUpgradeTerminal(t *testing.T) {
	assert.True(t, IsUpgradeTerminal(UpgradeCompleted))
	assert.True(t, IsUpgradeTerminal(UpgradeFailed))
	assert.False(t, IsUpgradeTerminal(UpgradePending))
	assert.False(t, IsUpgradeTerminal(UpgradeDownloading))
	assert.False(t, IsUpgradeTerminal(UpgradeUploading))
	assert.False(t, IsUpgradeTerminal(UpgradeRebooting))
	assert.False(t, IsUpgradeTerminal(UpgradeVerifying))
}

func TestNextUpgradeState(t *testing.T) {
	tests := []struct {
		current  UpgradeState
		expected UpgradeState
	}{
		{UpgradePending, UpgradeDownloading},
		{UpgradeDownloading, UpgradeRebooting},
		{UpgradeRebooting, UpgradeVerifying},
		{UpgradeVerifying, UpgradeCompleted},
		{UpgradeCompleted, UpgradeCompleted},
		{UpgradeFailed, UpgradeFailed},
	}

	for _, tt := range tests {
		t.Run(string(tt.current), func(t *testing.T) {
			assert.Equal(t, tt.expected, NextUpgradeState(tt.current))
		})
	}
}

func TestValidateUpgradeTransition_UnknownState(t *testing.T) {
	err := ValidateUpgradeTransition(UpgradeState("unknown"), UpgradePending)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown upgrade state")
}

func TestNextStateAfterTC(t *testing.T) {
	assert.Equal(t, UpgradeCompleted, NextStateAfterTC(false), "4G should complete on TC")
	assert.Equal(t, UpgradeRebooting, NextStateAfterTC(true), "5G should enter rebooting on TC")
}
