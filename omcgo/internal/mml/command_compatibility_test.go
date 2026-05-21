package mml

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestComputeCommandCompatibility 表驱动覆盖纯函数 ComputeCommandCompatibility
// 的全部分支：空 paths / 全 supported / 部分 / 全 unsupported / 边界。
func TestComputeCommandCompatibility(t *testing.T) {
	id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	id3 := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	id4 := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	tests := []struct {
		name            string
		commands        []CommandPathRow
		supported       map[string]struct{}
		wantUnsupported []uuid.UUID
	}{
		{
			name: "all paths supported → no unsupported",
			commands: []CommandPathRow{
				{ID: id1, Paths: []string{"P1", "P2"}},
			},
			supported:       map[string]struct{}{"P1": {}, "P2": {}, "P3": {}},
			wantUnsupported: nil,
		},
		{
			name: "missing one path → command marked unsupported",
			commands: []CommandPathRow{
				{ID: id1, Paths: []string{"P1", "P_MISSING"}},
			},
			supported:       map[string]struct{}{"P1": {}},
			wantUnsupported: []uuid.UUID{id1},
		},
		{
			name: "empty paths → considered supported (no warning)",
			commands: []CommandPathRow{
				{ID: id1, Paths: nil},
				{ID: id2, Paths: []string{}},
			},
			supported:       map[string]struct{}{},
			wantUnsupported: nil,
		},
		{
			name: "empty supported set → all with paths are unsupported",
			commands: []CommandPathRow{
				{ID: id1, Paths: []string{"P1"}},
				{ID: id2, Paths: []string{}},
				{ID: id3, Paths: []string{"P2", "P3"}},
			},
			supported:       map[string]struct{}{},
			wantUnsupported: []uuid.UUID{id1, id3},
		},
		{
			name: "partial match — mix of supported and unsupported",
			commands: []CommandPathRow{
				{ID: id1, Paths: []string{"P1"}},
				{ID: id2, Paths: []string{"P_X"}},
				{ID: id3, Paths: []string{"P1", "P2"}},
				{ID: id4, Paths: []string{"P1", "P_Y"}},
			},
			supported:       map[string]struct{}{"P1": {}, "P2": {}},
			wantUnsupported: []uuid.UUID{id2, id4},
		},
		{
			name: "single command, single path, exact match → supported",
			commands: []CommandPathRow{
				{ID: id1, Paths: []string{"Device.DeviceInfo.UserLabel"}},
			},
			supported:       map[string]struct{}{"Device.DeviceInfo.UserLabel": {}},
			wantUnsupported: nil,
		},
		{
			name: "command with many paths, only last one missing → marked unsupported",
			commands: []CommandPathRow{
				{ID: id1, Paths: []string{"P1", "P2", "P3", "P4", "P_MISSING"}},
			},
			supported:       map[string]struct{}{"P1": {}, "P2": {}, "P3": {}, "P4": {}},
			wantUnsupported: []uuid.UUID{id1},
		},
		{
			name:            "empty commands list → nil result",
			commands:        []CommandPathRow{},
			supported:       map[string]struct{}{"P1": {}},
			wantUnsupported: nil,
		},
		{
			name: "preserve input order in unsupported result",
			commands: []CommandPathRow{
				{ID: id3, Paths: []string{"P_X"}},
				{ID: id1, Paths: []string{"P_Y"}},
				{ID: id2, Paths: []string{"P1"}},
				{ID: id4, Paths: []string{"P_Z"}},
			},
			supported:       map[string]struct{}{"P1": {}},
			wantUnsupported: []uuid.UUID{id3, id1, id4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeCommandCompatibility(tt.commands, tt.supported)
			assert.Equal(t, tt.wantUnsupported, got)
		})
	}
}

// TestCompatibilityService_GetCommandCompatibility_NotInitialized 验证
// service 没有正确 wire 时返回明确错误而非 panic。
func TestCompatibilityService_GetCommandCompatibility_NotInitialized(t *testing.T) {
	svc := &CompatibilityService{}
	_, err := svc.GetCommandCompatibility(context.Background(), "FAP/MLN/SC")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not properly initialized")
}
