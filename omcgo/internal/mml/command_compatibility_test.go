package mml

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/product"
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

// ── T-0177 stubs ─────────────────────────────────────────────────────

// stubProductMatcher 实现 productClassMatcher 窄接口。
type stubProductMatcher struct {
	result *product.MatchResult
	err    error
	calls  int
}

func (s *stubProductMatcher) MatchProductClass(_ context.Context, _ string) (*product.MatchResult, error) {
	s.calls++
	return s.result, s.err
}

// stubParamMappingLookup 实现 paramMappingLookup 窄接口。
type stubParamMappingLookup struct {
	set   *parammodel.MappingSet
	err   error
	calls int
}

func (s *stubParamMappingLookup) GetByParamModel(_ context.Context, _ uuid.UUID) (*parammodel.MappingSet, error) {
	s.calls++
	return s.set, s.err
}

// stubCmdPathRepo 实现 CommandPathRepository。
type stubCmdPathRepo struct {
	rows []CommandPathRow
	err  error
}

func (s *stubCmdPathRepo) ListAllCommandPaths(_ context.Context) ([]CommandPathRow, error) {
	return s.rows, s.err
}

// uuidPtr 构造 uuid 指针的小工具。
func uuidPtr(u uuid.UUID) *uuid.UUID { return &u }

// Test_GetCommandCompatibility_MatchedProductClass_UsesProductRegistry
// happy path：productMatcher 返 valid MatchResult → paramRegistry 被调
// → 命中 mapping 的命令支持，未命中的标 unsupported。
func Test_GetCommandCompatibility_MatchedProductClass_UsesProductRegistry(t *testing.T) {
	productID := uuid.New()
	paramModelID := uuid.New()
	cmdSupported := uuid.New()
	cmdUnsupported := uuid.New()

	matcher := &stubProductMatcher{
		result: &product.MatchResult{
			Product: &product.Product{
				ID:           productID,
				ParamModelID: uuidPtr(paramModelID),
			},
			MatchedPattern: "FAP/.*",
			GlobalOrder:    10,
		},
	}
	paramRegistry := &stubParamMappingLookup{
		set: &parammodel.MappingSet{
			ParamModelID: paramModelID,
			Mappings: []parammodel.ParamMapping{
				{StandardPath: "Device.WiFi.SSID.{i}.Enable"},
				{StandardPath: "Device.WiFi.Radio.{i}.Channel"},
			},
		},
	}
	cmdRepo := &stubCmdPathRepo{
		rows: []CommandPathRow{
			{ID: cmdSupported, Paths: []string{"Device.WiFi.SSID.{i}.Enable"}},
			{ID: cmdUnsupported, Paths: []string{"Device.WiFi.UNKNOWN"}},
		},
	}

	svc := NewCompatibilityService(matcher, paramRegistry, cmdRepo, zap.NewNop())
	got, err := svc.GetCommandCompatibility(context.Background(), "FAP/MLN/SC")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, 1, matcher.calls, "productMatcher.MatchProductClass should be called once")
	assert.Equal(t, 1, paramRegistry.calls, "paramRegistry.GetByParamModel should be called once for resolved product")
	assert.Equal(t, productID, got.ProductID)
	assert.Equal(t, paramModelID, got.ParamModelID)
	assert.Equal(t, []uuid.UUID{cmdUnsupported}, got.UnsupportedCommandIDs)
}

// Test_GetCommandCompatibility_OrphanProductClass_EmptySupported_200
// orphan path：productMatcher 返 ErrOrphan → 不报错 + supportedPaths 空
// + 所有有 path 的命令在 unsupported 列表 + paramRegistry 不被调用。
func Test_GetCommandCompatibility_OrphanProductClass_EmptySupported_200(t *testing.T) {
	cmdA := uuid.New()
	cmdB := uuid.New()
	cmdNoPath := uuid.New()

	matcher := &stubProductMatcher{err: product.ErrOrphan}
	paramRegistry := &stubParamMappingLookup{} // 不应被调到
	cmdRepo := &stubCmdPathRepo{
		rows: []CommandPathRow{
			{ID: cmdA, Paths: []string{"Device.X"}},
			{ID: cmdB, Paths: []string{"Device.Y"}},
			{ID: cmdNoPath, Paths: nil}, // 无 path → 仍视为 supported（跟其他分支一致）
		},
	}

	svc := NewCompatibilityService(matcher, paramRegistry, cmdRepo, zap.NewNop())
	got, err := svc.GetCommandCompatibility(context.Background(), "UNKNOWN/CLASS")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, 0, paramRegistry.calls, "paramRegistry must not be called for orphan productClass")
	assert.Equal(t, uuid.Nil, got.ProductID, "orphan: productID should remain zero value")
	assert.Equal(t, uuid.Nil, got.ParamModelID, "orphan: paramModelID should remain zero value")
	assert.Equal(t, []uuid.UUID{cmdA, cmdB}, got.UnsupportedCommandIDs)
	assert.Equal(t, "UNKNOWN/CLASS", got.ProductClass)
}

// Test_GetCommandCompatibility_MatchError_ReturnsError
// 非 ErrOrphan 错误（DB / Redis / regex 等）必须返到上层，不被静默吞掉。
func Test_GetCommandCompatibility_MatchError_ReturnsError(t *testing.T) {
	matcherErr := errors.New("boom: cache version fetch failed")
	matcher := &stubProductMatcher{err: matcherErr}
	paramRegistry := &stubParamMappingLookup{}
	cmdRepo := &stubCmdPathRepo{}

	svc := NewCompatibilityService(matcher, paramRegistry, cmdRepo, zap.NewNop())
	got, err := svc.GetCommandCompatibility(context.Background(), "FAP/MLN/SC")
	require.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "match product_class")
	assert.ErrorIs(t, err, matcherErr)
}

// Test_GetCommandCompatibility_ProductHasNoParamModel_EmptySupported
// match 到 product 但 ParamModelID == nil → 走"无 paramModel 链路"分支：
// supportedPaths 空 + paramRegistry 不被调用 + 返 productID（非零）但
// paramModelID 零值。
func Test_GetCommandCompatibility_ProductHasNoParamModel_EmptySupported(t *testing.T) {
	productID := uuid.New()
	cmdA := uuid.New()

	matcher := &stubProductMatcher{
		result: &product.MatchResult{
			Product: &product.Product{
				ID:           productID,
				ParamModelID: nil, // 关键：product 存在但未配 paramModel
			},
			MatchedPattern: "FAP/.*",
		},
	}
	paramRegistry := &stubParamMappingLookup{} // 不应被调
	cmdRepo := &stubCmdPathRepo{
		rows: []CommandPathRow{
			{ID: cmdA, Paths: []string{"Device.WiFi.SSID.{i}.Enable"}},
		},
	}

	svc := NewCompatibilityService(matcher, paramRegistry, cmdRepo, zap.NewNop())
	got, err := svc.GetCommandCompatibility(context.Background(), "FAP/MLN/SC")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, 0, paramRegistry.calls, "paramRegistry must not be called when ParamModelID is nil")
	assert.Equal(t, productID, got.ProductID, "productID should be populated even when paramModel missing")
	assert.Equal(t, uuid.Nil, got.ParamModelID, "paramModelID stays zero when product has no paramModel")
	assert.Equal(t, []uuid.UUID{cmdA}, got.UnsupportedCommandIDs)
}
