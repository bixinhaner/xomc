package device

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// T-0162 P2 — ValidateLifecycleTransition 单元测试
//
// 覆盖：
//   1. 所有合法转移（validLifecycleTransitions 全集）
//   2. decommissioned 终态：转任何目标都拒
//   3. 非法 current / target 取值
//   4. 同状态转移（current == target）：未在 map 中 → 拒（防止误转）
// ============================================================================

func TestValidateLifecycleTransition_AllValidPaths(t *testing.T) {
	tests := []struct {
		name string
		from model.DeviceLifecycle
		to   model.DeviceLifecycle
	}{
		// discovered →
		{"discovered to registered", model.LifecycleDiscovered, model.LifecycleRegistered},
		{"discovered to commissioned", model.LifecycleDiscovered, model.LifecycleCommissioned},

		// registered →
		{"registered to provisioning", model.LifecycleRegistered, model.LifecycleProvisioning},
		{"registered to commissioned", model.LifecycleRegistered, model.LifecycleCommissioned},
		{"registered to decommissioned", model.LifecycleRegistered, model.LifecycleDecommissioned},

		// provisioning →
		{"provisioning to commissioned", model.LifecycleProvisioning, model.LifecycleCommissioned},
		{"provisioning to registered", model.LifecycleProvisioning, model.LifecycleRegistered},
		{"provisioning to decommissioned", model.LifecycleProvisioning, model.LifecycleDecommissioned},

		// commissioned →
		{"commissioned to maintenance", model.LifecycleCommissioned, model.LifecycleMaintenance},
		{"commissioned to decommissioned", model.LifecycleCommissioned, model.LifecycleDecommissioned},

		// maintenance →
		{"maintenance to commissioned", model.LifecycleMaintenance, model.LifecycleCommissioned},
		{"maintenance to decommissioned", model.LifecycleMaintenance, model.LifecycleDecommissioned},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, ValidateLifecycleTransition(tt.from, tt.to),
				"%s → %s should be allowed", tt.from, tt.to)
		})
	}
}

// TestValidateLifecycleTransition_DecommissionedIsTerminal — D3 决策：
// decommissioned 不可逆，转任何目标都应拒。
func TestValidateLifecycleTransition_DecommissionedIsTerminal(t *testing.T) {
	tests := []model.DeviceLifecycle{
		model.LifecycleDiscovered, model.LifecycleRegistered, model.LifecycleProvisioning,
		model.LifecycleCommissioned, model.LifecycleMaintenance, model.LifecycleDecommissioned,
	}
	for _, target := range tests {
		t.Run(string(target), func(t *testing.T) {
			err := ValidateLifecycleTransition(model.LifecycleDecommissioned, target)
			require.Error(t, err, "decommissioned → %s should be rejected", target)
			assert.Contains(t, err.Error(), "terminal", "error should mention terminal state")
		})
	}
}

// TestValidateLifecycleTransition_InvalidTarget — current 合法、target 非法值
func TestValidateLifecycleTransition_InvalidTarget(t *testing.T) {
	err := ValidateLifecycleTransition(model.LifecycleCommissioned, model.DeviceLifecycle("offline"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target lifecycle")
}

// TestValidateLifecycleTransition_InvalidCurrent — current 非法值
func TestValidateLifecycleTransition_InvalidCurrent(t *testing.T) {
	err := ValidateLifecycleTransition(model.DeviceLifecycle("active"), model.LifecycleCommissioned)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid current lifecycle")
}

// TestValidateLifecycleTransition_SameStateRejected — current == target 不在 map 中 → 拒
// 避免业务代码误转同状态（应直接跳过，而非调状态机）。
func TestValidateLifecycleTransition_SameStateRejected(t *testing.T) {
	err := ValidateLifecycleTransition(model.LifecycleCommissioned, model.LifecycleCommissioned)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid lifecycle transition")
}

// TestValidateLifecycleTransition_BackwardRejected — 倒退路径默认拒（如
// commissioned → registered，业务上几乎从不允许）
func TestValidateLifecycleTransition_BackwardRejected(t *testing.T) {
	tests := []struct {
		name string
		from model.DeviceLifecycle
		to   model.DeviceLifecycle
	}{
		{"commissioned to discovered", model.LifecycleCommissioned, model.LifecycleDiscovered},
		{"commissioned to registered", model.LifecycleCommissioned, model.LifecycleRegistered},
		{"commissioned to provisioning", model.LifecycleCommissioned, model.LifecycleProvisioning},
		{"maintenance to provisioning", model.LifecycleMaintenance, model.LifecycleProvisioning},
		{"discovered to provisioning", model.LifecycleDiscovered, model.LifecycleProvisioning},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, ValidateLifecycleTransition(tt.from, tt.to),
				"%s → %s should be rejected", tt.from, tt.to)
		})
	}
}

// TestDeviceLifecycle_IsValid — 类型方法测试
func TestDeviceLifecycle_IsValid(t *testing.T) {
	for _, v := range []model.DeviceLifecycle{
		model.LifecycleDiscovered, model.LifecycleRegistered, model.LifecycleProvisioning,
		model.LifecycleCommissioned, model.LifecycleMaintenance, model.LifecycleDecommissioned,
	} {
		assert.True(t, v.IsValid(), "%q should be valid", v)
	}
	for _, v := range []model.DeviceLifecycle{"active", "offline", "", "garbage"} {
		assert.False(t, v.IsValid(), "%q should be invalid", v)
	}
}
