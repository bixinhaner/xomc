package alarm

import (
	"testing"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestApplyDataPermission_SuperAdmin(t *testing.T) {
	checker := NewDataPermissionChecker(zap.NewNop())
	permCtx := &PermissionContext{IsSuperAdmin: true, Carrier: "cmcc"}
	filter := &AlarmFilter{}
	checker.ApplyDataPermission(nil, permCtx, filter)
	assert.Nil(t, filter.Carrier)
}

func TestApplyDataPermission_NormalUser(t *testing.T) {
	checker := NewDataPermissionChecker(zap.NewNop())
	permCtx := &PermissionContext{Role: "operator", Carrier: "cmcc"}
	filter := &AlarmFilter{}
	checker.ApplyDataPermission(nil, permCtx, filter)
	assert.NotNil(t, filter.Carrier)
	assert.Equal(t, model.CarrierCode("cmcc"), *filter.Carrier)
}

func TestApplyDataPermission_NilContext(t *testing.T) {
	checker := NewDataPermissionChecker(zap.NewNop())
	filter := &AlarmFilter{}
	checker.ApplyDataPermission(nil, nil, filter)
	assert.Nil(t, filter.Carrier)
}

func TestCanAcknowledge(t *testing.T) {
	checker := NewDataPermissionChecker(zap.NewNop())
	assert.True(t, checker.CanAcknowledge(&PermissionContext{IsSuperAdmin: true}))
	assert.True(t, checker.CanAcknowledge(&PermissionContext{Role: "admin"}))
	assert.True(t, checker.CanAcknowledge(&PermissionContext{Role: "operator"}))
	assert.False(t, checker.CanAcknowledge(&PermissionContext{Role: "viewer"}))
	assert.False(t, checker.CanAcknowledge(nil))
}

func TestCanClear(t *testing.T) {
	checker := NewDataPermissionChecker(zap.NewNop())
	assert.True(t, checker.CanClear(&PermissionContext{IsSuperAdmin: true}))
	assert.True(t, checker.CanClear(&PermissionContext{Role: "admin"}))
	assert.False(t, checker.CanClear(&PermissionContext{Role: "operator"}))
	assert.False(t, checker.CanClear(nil))
}

func TestBuildPermissionContext(t *testing.T) {
	checker := NewDataPermissionChecker(zap.NewNop())
	claims := map[string]interface{}{
		"user_id":          "user-123",
		"role":             "operator",
		"is_super_admin":   false,
		"carrier":          "ctcc",
		"device_group_ids": []interface{}{"group-1", "group-2"},
	}
	ctx := checker.BuildPermissionContext(claims)
	assert.Equal(t, "user-123", ctx.UserID)
	assert.Equal(t, "operator", ctx.Role)
	assert.Equal(t, model.CarrierCode("ctcc"), ctx.Carrier)
	assert.Equal(t, []string{"group-1", "group-2"}, ctx.DeviceGroupIDs)
}

func TestValidatePermission(t *testing.T) {
	checker := NewDataPermissionChecker(zap.NewNop())
	assert.NoError(t, checker.ValidatePermission(&PermissionContext{UserID: "u1"}))
	assert.Error(t, checker.ValidatePermission(nil))
	assert.Error(t, checker.ValidatePermission(&PermissionContext{}))
}
