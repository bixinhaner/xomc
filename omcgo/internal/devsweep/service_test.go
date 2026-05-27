package devsweep

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
)

// ── stubs ────────────────────────────────────────────────────────────

type stubDeviceLookup struct {
	dev *model.Device
	err error
}

func (s *stubDeviceLookup) GetBySerialNumber(_ context.Context, _ string) (*model.Device, error) {
	return s.dev, s.err
}

type stubProductMatcher struct {
	match *product.MatchResult
	err   error
}

func (s *stubProductMatcher) MatchProductClass(_ context.Context, _ string) (*product.MatchResult, error) {
	return s.match, s.err
}

type stubParamLookup struct {
	set *parammodel.MappingSet
	err error
}

func (s *stubParamLookup) GetByParamModel(_ context.Context, _ uuid.UUID) (*parammodel.MappingSet, error) {
	return s.set, s.err
}

type stubRepository struct {
	deviceCount int64
	marked      []string
	markErr     error
	pmID        uuid.UUID
}

func (s *stubRepository) MarkUnsupportedBatch(_ context.Context, pmID uuid.UUID, paths []string) (int64, error) {
	if s.markErr != nil {
		return 0, s.markErr
	}
	s.pmID = pmID
	s.marked = append(s.marked, paths...)
	return int64(len(paths)), nil
}

func (s *stubRepository) CountDevicesByParamModel(_ context.Context, _ uuid.UUID) (int64, error) {
	return s.deviceCount, nil
}

type scriptedProber struct {
	outcomes map[string]ProbeOutcome // by standardPath (NOT probePath — service maps back)
}

func (p *scriptedProber) Probe(_ context.Context, _ string, paths []string, batchIdx int) []ProbeRecord {
	out := make([]ProbeRecord, 0, len(paths))
	for _, probePath := range paths {
		// scripted by probePath；service 之后再把 StandardPath 覆盖回去
		oc, ok := p.outcomes[probePath]
		if !ok {
			oc = OutcomeUnknown
		}
		out = append(out, ProbeRecord{
			ProbePath: probePath,
			Outcome:   oc,
			Batch:     batchIdx,
		})
	}
	return out
}

// ── fixtures ─────────────────────────────────────────────────────────

func okDevice(sn, productClass string) *model.Device {
	return &model.Device{
		ID:              uuid.New(),
		SerialNumber:    sn,
		ProductClass:    productClass,
		FirmwareVersion: "BaiBLQ_5.0.16.1_1229",
		IsOnline:        true,
	}
}

func mappingSet(paths ...string) *parammodel.MappingSet {
	pmID := uuid.New()
	ms := make([]parammodel.ParamMapping, 0, len(paths))
	for _, p := range paths {
		ms = append(ms, parammodel.ParamMapping{
			ID:           uuid.New(),
			ParamModelID: pmID,
			StandardPath: p,
			PrivatePath:  p,
			EntryType:    "parameter",
			IsActive:     true,
			IsSupported:  true,
		})
	}
	return &parammodel.MappingSet{
		ParamModelID: pmID,
		Source:       parammodel.MappingSourceDefault,
		Mappings:     ms,
	}
}

func productMatch(pmID uuid.UUID) *product.MatchResult {
	prodID := uuid.New()
	return &product.MatchResult{
		Product: &product.Product{
			ID:           prodID,
			Name:         "BLQ Series",
			ParamModelID: &pmID,
		},
		MatchedPattern: "FAP/mBS31001/.*",
	}
}

// ── tests ─────────────────────────────────────────────────────────────

// Test_Service_Run_Happy_DryRun：3 path 全 supported，dry-run，无 DB 写。
func Test_Service_Run_Happy_DryRun(t *testing.T) {
	set := mappingSet("Device.A", "Device.B", "Device.C")
	devices := &stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")}
	products := &stubProductMatcher{match: productMatch(set.ParamModelID)}
	params := &stubParamLookup{set: set}
	prober := &scriptedProber{outcomes: map[string]ProbeOutcome{
		"Device.A": OutcomeSupported,
		"Device.B": OutcomeSupported,
		"Device.C": OutcomeSupported,
	}}
	repo := &stubRepository{deviceCount: 1}

	svc := NewService(devices, products, params, prober, repo, zap.NewNop())
	res, err := svc.Run(context.Background(), Options{
		DeviceSN: "SN1",
		Apply:    false,
		RPCRate:  1000,
	})
	require.NoError(t, err)
	require.False(t, res.Aborted)
	require.Equal(t, 3, res.CandidateCount)
	require.Equal(t, 3, res.SupportedCount)
	require.Equal(t, 0, res.UnsupportedCount)
	require.Equal(t, 0, res.MarkedCount)
	require.True(t, res.DryRun)
	require.Empty(t, repo.marked, "dry-run must not write DB")
}

// Test_Service_Run_Happy_Apply：1 unsupported，apply=true，DB 写 1 行。
func Test_Service_Run_Happy_Apply(t *testing.T) {
	set := mappingSet("Device.A", "Device.B")
	devices := &stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")}
	products := &stubProductMatcher{match: productMatch(set.ParamModelID)}
	params := &stubParamLookup{set: set}
	prober := &scriptedProber{outcomes: map[string]ProbeOutcome{
		"Device.A": OutcomeSupported,
		"Device.B": OutcomeUnsupported,
	}}
	repo := &stubRepository{deviceCount: 5}

	svc := NewService(devices, products, params, prober, repo, zap.NewNop())
	res, err := svc.Run(context.Background(), Options{
		DeviceSN: "SN1",
		Apply:    true,
		RPCRate:  1000,
	})
	require.NoError(t, err)
	require.False(t, res.Aborted)
	require.Equal(t, 1, res.UnsupportedCount)
	require.Equal(t, 1, res.MarkedCount)
	require.Len(t, repo.marked, 1)
	require.Equal(t, "Device.B", repo.marked[0])
	require.Equal(t, set.ParamModelID, repo.pmID)
}

// Test_Service_Run_DeviceNotFound：device repo 返 nil → abort + device_not_found。
func Test_Service_Run_DeviceNotFound(t *testing.T) {
	svc := NewService(
		&stubDeviceLookup{dev: nil},
		&stubProductMatcher{}, &stubParamLookup{},
		&scriptedProber{}, &stubRepository{},
		zap.NewNop(),
	)
	res, err := svc.Run(context.Background(), Options{DeviceSN: "GHOST", RPCRate: 1000})
	require.Error(t, err)
	require.True(t, res.Aborted)
	require.Equal(t, ErrCodeDeviceNotFound, res.ErrorCode)
}

// Test_Service_Run_DeviceOffline：在线=false → abort + device_offline。
func Test_Service_Run_DeviceOffline(t *testing.T) {
	dev := okDevice("SN1", "FAP/mBS31001/SC")
	dev.IsOnline = false
	svc := NewService(
		&stubDeviceLookup{dev: dev},
		&stubProductMatcher{}, &stubParamLookup{},
		&scriptedProber{}, &stubRepository{},
		zap.NewNop(),
	)
	res, err := svc.Run(context.Background(), Options{DeviceSN: "SN1", RPCRate: 1000})
	require.Error(t, err)
	require.True(t, res.Aborted)
	require.Equal(t, ErrCodeDeviceOffline, res.ErrorCode)
}

// Test_Service_Run_Orphan：productClass 未匹配 → abort + orphan_product_class。
func Test_Service_Run_Orphan(t *testing.T) {
	svc := NewService(
		&stubDeviceLookup{dev: okDevice("SN1", "Unknown/Product/Class")},
		&stubProductMatcher{err: product.ErrOrphan},
		&stubParamLookup{},
		&scriptedProber{}, &stubRepository{},
		zap.NewNop(),
	)
	res, err := svc.Run(context.Background(), Options{DeviceSN: "SN1", RPCRate: 1000})
	require.Error(t, err)
	require.True(t, res.Aborted)
	require.Equal(t, ErrCodeOrphanProduct, res.ErrorCode)
}

// Test_Service_Run_ParamModelNil：product.ParamModelID=nil → abort + param_model_unset。
func Test_Service_Run_ParamModelNil(t *testing.T) {
	match := &product.MatchResult{
		Product: &product.Product{
			ID:           uuid.New(),
			Name:         "Unbound",
			ParamModelID: nil,
		},
	}
	svc := NewService(
		&stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")},
		&stubProductMatcher{match: match},
		&stubParamLookup{},
		&scriptedProber{}, &stubRepository{},
		zap.NewNop(),
	)
	res, err := svc.Run(context.Background(), Options{DeviceSN: "SN1", RPCRate: 1000})
	require.Error(t, err)
	require.True(t, res.Aborted)
	require.Equal(t, ErrCodeParamModelUnset, res.ErrorCode)
}

// Test_Service_Run_ParamModelNoMapping：Registry 返 ErrNoMapping → abort + no_mappings。
func Test_Service_Run_ParamModelNoMapping(t *testing.T) {
	pmID := uuid.New()
	svc := NewService(
		&stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")},
		&stubProductMatcher{match: productMatch(pmID)},
		&stubParamLookup{err: parammodel.ErrNoMapping},
		&scriptedProber{}, &stubRepository{},
		zap.NewNop(),
	)
	res, err := svc.Run(context.Background(), Options{DeviceSN: "SN1", RPCRate: 1000})
	require.Error(t, err)
	require.True(t, res.Aborted)
	require.Equal(t, ErrCodeNoMappings, res.ErrorCode)
}

// Test_Service_Run_ThresholdAbort：unsupported>50% 且未 --force → abort + unsupported_rate_high。
func Test_Service_Run_ThresholdAbort(t *testing.T) {
	set := mappingSet("Device.A", "Device.B", "Device.C", "Device.D")
	devices := &stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")}
	products := &stubProductMatcher{match: productMatch(set.ParamModelID)}
	params := &stubParamLookup{set: set}
	prober := &scriptedProber{outcomes: map[string]ProbeOutcome{
		"Device.A": OutcomeUnsupported,
		"Device.B": OutcomeUnsupported,
		"Device.C": OutcomeUnsupported, // 3/4 = 75% > 50%
		"Device.D": OutcomeSupported,
	}}
	repo := &stubRepository{deviceCount: 1}

	svc := NewService(devices, products, params, prober, repo, zap.NewNop())
	res, err := svc.Run(context.Background(), Options{
		DeviceSN: "SN1",
		Apply:    true,
		RPCRate:  1000,
	})
	require.Error(t, err)
	require.True(t, res.Aborted)
	require.Equal(t, ErrCodeUnsupportedRateHigh, res.ErrorCode)
	require.Empty(t, repo.marked, "abort must skip DB write")
}

// Test_Service_Run_ThresholdAbort_Force：同上但 --force → 写库通过。
func Test_Service_Run_ThresholdAbort_Force(t *testing.T) {
	set := mappingSet("Device.A", "Device.B", "Device.C", "Device.D")
	devices := &stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")}
	products := &stubProductMatcher{match: productMatch(set.ParamModelID)}
	params := &stubParamLookup{set: set}
	prober := &scriptedProber{outcomes: map[string]ProbeOutcome{
		"Device.A": OutcomeUnsupported,
		"Device.B": OutcomeUnsupported,
		"Device.C": OutcomeUnsupported,
		"Device.D": OutcomeSupported,
	}}
	repo := &stubRepository{deviceCount: 1}

	svc := NewService(devices, products, params, prober, repo, zap.NewNop())
	res, err := svc.Run(context.Background(), Options{
		DeviceSN: "SN1",
		Apply:    true,
		Force:    true,
		RPCRate:  1000,
	})
	require.NoError(t, err)
	require.False(t, res.Aborted)
	require.Equal(t, 3, res.MarkedCount)
}

// Test_Service_Run_ParamModelWide_Unconfirmed：>10 设备且未确认 → abort。
// 用 10 path 中 1 unsupported 保持 unsupported_rate=10% 不触发 fraction gate，
// 单独验证 wide-impact gate。
func Test_Service_Run_ParamModelWide_Unconfirmed(t *testing.T) {
	set := mappingSet("Device.A", "Device.B", "Device.C", "Device.D", "Device.E",
		"Device.F", "Device.G", "Device.H", "Device.I", "Device.J")
	devices := &stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")}
	products := &stubProductMatcher{match: productMatch(set.ParamModelID)}
	params := &stubParamLookup{set: set}
	outcomes := map[string]ProbeOutcome{"Device.A": OutcomeUnsupported}
	for _, p := range []string{"Device.B", "Device.C", "Device.D", "Device.E",
		"Device.F", "Device.G", "Device.H", "Device.I", "Device.J"} {
		outcomes[p] = OutcomeSupported
	}
	prober := &scriptedProber{outcomes: outcomes}
	repo := &stubRepository{deviceCount: 50} // 远超 10

	svc := NewService(devices, products, params, prober, repo, zap.NewNop())
	res, err := svc.Run(context.Background(), Options{
		DeviceSN: "SN1",
		Apply:    true,
		RPCRate:  1000,
	})
	require.Error(t, err)
	require.True(t, res.Aborted)
	require.Equal(t, ErrCodeParamModelWideUnconf, res.ErrorCode)
	require.Empty(t, repo.marked)
}

// Test_Service_Run_ParamModelWide_Confirmed：>10 设备 + 显式确认 → 通过。
func Test_Service_Run_ParamModelWide_Confirmed(t *testing.T) {
	set := mappingSet("Device.A", "Device.B", "Device.C", "Device.D", "Device.E",
		"Device.F", "Device.G", "Device.H", "Device.I", "Device.J")
	devices := &stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")}
	products := &stubProductMatcher{match: productMatch(set.ParamModelID)}
	params := &stubParamLookup{set: set}
	outcomes := map[string]ProbeOutcome{"Device.A": OutcomeUnsupported}
	for _, p := range []string{"Device.B", "Device.C", "Device.D", "Device.E",
		"Device.F", "Device.G", "Device.H", "Device.I", "Device.J"} {
		outcomes[p] = OutcomeSupported
	}
	prober := &scriptedProber{outcomes: outcomes}
	repo := &stubRepository{deviceCount: 50}

	svc := NewService(devices, products, params, prober, repo, zap.NewNop())
	res, err := svc.Run(context.Background(), Options{
		DeviceSN:              "SN1",
		Apply:                 true,
		ConfirmParamModelWide: true,
		RPCRate:               1000,
	})
	require.NoError(t, err)
	require.False(t, res.Aborted)
	require.Equal(t, 1, res.MarkedCount)
}

// Test_Service_Run_BatchPartialFailure：mix supported/unsupported/unknown 计数正确。
func Test_Service_Run_BatchPartialFailure(t *testing.T) {
	set := mappingSet("Device.A", "Device.B", "Device.C", "Device.D", "Device.E")
	devices := &stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")}
	products := &stubProductMatcher{match: productMatch(set.ParamModelID)}
	params := &stubParamLookup{set: set}
	prober := &scriptedProber{outcomes: map[string]ProbeOutcome{
		"Device.A": OutcomeSupported,
		"Device.B": OutcomeSupported,
		"Device.C": OutcomeUnknown, // timeout / 非 9005 fault
		"Device.D": OutcomeUnsupported,
		"Device.E": OutcomeSupported,
	}}
	repo := &stubRepository{deviceCount: 1}

	svc := NewService(devices, products, params, prober, repo, zap.NewNop())
	res, err := svc.Run(context.Background(), Options{
		DeviceSN: "SN1",
		Apply:    false,
		RPCRate:  1000,
	})
	require.NoError(t, err)
	require.Equal(t, 5, res.CandidateCount)
	require.Equal(t, 3, res.SupportedCount)
	require.Equal(t, 1, res.UnsupportedCount)
	require.Equal(t, 1, res.UnknownCount)
}

// Test_Service_Run_PrefixFilter：仅 prefix 匹配的 path 进候选集。
func Test_Service_Run_PrefixFilter(t *testing.T) {
	set := mappingSet("Device.FaultMgmt.X", "Device.FaultMgmt.Y", "Device.WiFi.Z")
	devices := &stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")}
	products := &stubProductMatcher{match: productMatch(set.ParamModelID)}
	params := &stubParamLookup{set: set}
	prober := &scriptedProber{outcomes: map[string]ProbeOutcome{
		"Device.FaultMgmt.X": OutcomeSupported,
		"Device.FaultMgmt.Y": OutcomeSupported,
	}}
	repo := &stubRepository{deviceCount: 1}

	svc := NewService(devices, products, params, prober, repo, zap.NewNop())
	res, err := svc.Run(context.Background(), Options{
		DeviceSN: "SN1",
		Prefix:   "Device.FaultMgmt.",
		RPCRate:  1000,
	})
	require.NoError(t, err)
	require.Equal(t, 2, res.CandidateCount, "prefix filter must drop non-matching")
}

// Test_Service_Run_iPlaceholder_NormalizedForProbe：catalog 含 {i} 的 path
// 在传给 prober 时应替换为 .0.；StandardPath 字段保留原 {i} 版本。
func Test_Service_Run_iPlaceholder_NormalizedForProbe(t *testing.T) {
	pmID := uuid.New()
	set := &parammodel.MappingSet{
		ParamModelID: pmID,
		Source:       parammodel.MappingSourceDefault,
		Mappings: []parammodel.ParamMapping{{
			ID:           uuid.New(),
			ParamModelID: pmID,
			StandardPath: "Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation",
			PrivatePath:  "Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation",
			EntryType:    "parameter",
			IsActive:     true,
			IsSupported:  true,
		}},
	}
	devices := &stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")}
	products := &stubProductMatcher{match: productMatch(set.ParamModelID)}
	params := &stubParamLookup{set: set}
	// 测试 prober 收到的是替换后的 .0. 形态
	var capturedProbePath string
	prober := proberFunc(func(_ context.Context, _ string, paths []string, batchIdx int) []ProbeRecord {
		if len(paths) > 0 {
			capturedProbePath = paths[0]
		}
		return []ProbeRecord{{ProbePath: paths[0], Outcome: OutcomeSupported, Batch: batchIdx}}
	})
	repo := &stubRepository{deviceCount: 1}

	svc := NewService(devices, products, params, prober, repo, zap.NewNop())
	res, err := svc.Run(context.Background(), Options{DeviceSN: "SN1", RPCRate: 1000})
	require.NoError(t, err)
	require.Equal(t, "Device.FaultMgmt.CurrentAlarm.0.AdditionalInformation", capturedProbePath)
	require.Len(t, res.PerPath, 1)
	require.Equal(t, "Device.FaultMgmt.CurrentAlarm.{i}.AdditionalInformation", res.PerPath[0].StandardPath,
		"StandardPath in result keeps the {i} form for accurate DB UPDATE matching")
}

// Test_Service_Run_DBError：MarkUnsupportedBatch 失败 → 错误透传。
// 单 path 100% unsupported 需要 --force 跳过 fraction gate；本测试目标是 DB 错。
func Test_Service_Run_DBError(t *testing.T) {
	set := mappingSet("Device.A")
	devices := &stubDeviceLookup{dev: okDevice("SN1", "FAP/mBS31001/SC")}
	products := &stubProductMatcher{match: productMatch(set.ParamModelID)}
	params := &stubParamLookup{set: set}
	prober := &scriptedProber{outcomes: map[string]ProbeOutcome{"Device.A": OutcomeUnsupported}}
	repo := &stubRepository{deviceCount: 1, markErr: errors.New("pg conn lost")}

	svc := NewService(devices, products, params, prober, repo, zap.NewNop())
	_, err := svc.Run(context.Background(), Options{
		DeviceSN: "SN1",
		Apply:    true,
		Force:    true,
		RPCRate:  1000,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mark unsupported batch")
}

// Test_Service_Run_DefaultsFilled：normalizeOptions 在零值时填默认。
func Test_Service_Run_DefaultsFilled(t *testing.T) {
	opts := normalizeOptions(Options{DeviceSN: "x"})
	require.Equal(t, 1, opts.BatchSize)
	require.Equal(t, 30*time.Second, opts.RPCTimeout)
	require.Equal(t, 5.0, opts.RPCRate)
	require.InEpsilon(t, SafetyDefaultMaxUnsupportedFraction, opts.Safety.MaxUnsupportedFraction, 1e-9)
	require.Equal(t, SafetyDefaultMaxParamModelDevices, opts.Safety.MaxParamModelDevices)
}

// proberFunc：闭包 Prober，仅本测试用。
type proberFunc func(ctx context.Context, sn string, paths []string, batchIdx int) []ProbeRecord

func (f proberFunc) Probe(ctx context.Context, sn string, paths []string, batchIdx int) []ProbeRecord {
	return f(ctx, sn, paths, batchIdx)
}
