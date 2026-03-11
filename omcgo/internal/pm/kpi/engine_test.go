package kpi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/carrier"
	"github.com/omcgo/omcgo/internal/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/model"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockCounterRepo is a test mock for counter.CounterRepository.
type mockCounterRepo struct {
	queryForKPIFunc func(ctx context.Context, deviceID uuid.UUID, cellID string, counterNames []string, startTime, endTime time.Time) (map[string]float64, error)
}

func (m *mockCounterRepo) BatchInsert(_ context.Context, _ []model.PMCounter) error {
	return nil
}

func (m *mockCounterRepo) Query(_ context.Context, _ counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
	return model.NewListResponse([]model.PMCounter{}, 0, 1, 20), nil
}

func (m *mockCounterRepo) QueryAggregated(_ context.Context, _ counter.CounterFilter) ([]counter.AggregatedCounter, error) {
	return nil, nil
}

func (m *mockCounterRepo) QueryForKPI(ctx context.Context, deviceID uuid.UUID, cellID string, counterNames []string, startTime, endTime time.Time) (map[string]float64, error) {
	if m.queryForKPIFunc != nil {
		return m.queryForKPIFunc(ctx, deviceID, cellID, counterNames, startTime, endTime)
	}
	return nil, nil
}

// mockKPIRepo is a test mock for KPIRepository.
type mockKPIRepo struct {
	insertedValues []model.KPIValue
}

func (m *mockKPIRepo) BatchInsert(_ context.Context, values []model.KPIValue) error {
	m.insertedValues = append(m.insertedValues, values...)
	return nil
}

func (m *mockKPIRepo) Query(_ context.Context, _ KPIFilter) (*model.ListResponse[model.KPIValue], error) {
	return model.NewListResponse([]model.KPIValue{}, 0, 1, 20), nil
}

func (m *mockKPIRepo) ListDefinitions(_ context.Context, _ *model.CarrierCode, _ *model.Technology) ([]model.KPIDefinition, error) {
	return nil, nil
}

func (m *mockKPIRepo) SyncDefinitions(_ context.Context, _ []model.KPIDefinition) error {
	return nil
}

func newTestEngine(counterRepo *mockCounterRepo, kpiRepo *mockKPIRepo) *KPIEngine {
	registry := carrier.NewRegistry()
	registry.Register(cmcc.New())

	logger := zap.NewNop()
	return NewKPIEngine(counterRepo, kpiRepo, registry, logger)
}

func TestKPIEngine_LoadFormulas(t *testing.T) {
	counterRepo := &mockCounterRepo{}
	kpiRepo := &mockKPIRepo{}
	engine := newTestEngine(counterRepo, kpiRepo)

	formulas := engine.Formulas()

	// CMCC has 10 LTE KPIs + 6 NR KPIs = 16 total
	assert.Equal(t, 16, len(formulas), "expected 16 KPI formulas from CMCC carrier")

	// Check that LTE and NR formulas are present
	lteCount := 0
	nrCount := 0
	for _, f := range formulas {
		if f.Technology == model.TechLTE {
			lteCount++
		}
		if f.Technology == model.TechNR {
			nrCount++
		}
		assert.Equal(t, model.CarrierCMCC, f.Carrier)
	}
	assert.Equal(t, 10, lteCount, "expected 10 LTE KPIs")
	assert.Equal(t, 6, nrCount, "expected 6 NR KPIs")
}

func TestKPIEngine_Calculate_ValidCounters(t *testing.T) {
	counterRepo := &mockCounterRepo{
		queryForKPIFunc: func(_ context.Context, _ uuid.UUID, _ string, _ []string, _, _ time.Time) (map[string]float64, error) {
			return map[string]float64{
				"rrc_conn_setup_succ":     950,
				"rrc_conn_setup_att":      1000,
				"erab_setup_succ":         780,
				"erab_setup_att":          800,
				"s1_sig_conn_setup_succ":  490,
				"s1_sig_conn_setup_att":   500,
				"erab_abnormal_release":   5,
				"erab_normal_release":     95,
				"intra_freq_ho_succ":      180,
				"intra_freq_ho_att":       200,
				"inter_freq_ho_succ":      90,
				"inter_freq_ho_att":       100,
				"dl_prb_used_avg":         75,
				"dl_prb_total":            100,
				"ul_prb_used_avg":         40,
				"ul_prb_total":            100,
				"pdcp_sdu_dl_volume":      125000000,
				"pdcp_sdu_ul_volume":      25000000,
				"period_seconds":          900,
			}, nil
		},
	}
	kpiRepo := &mockKPIRepo{}
	engine := newTestEngine(counterRepo, kpiRepo)

	deviceID := uuid.New()
	startTime := time.Now().Add(-15 * time.Minute)
	endTime := time.Now()

	values, err := engine.Calculate(
		context.Background(),
		deviceID, "Cell1",
		startTime, endTime,
		model.CarrierCMCC, model.TechLTE,
	)
	require.NoError(t, err)

	// Should produce values for all 10 LTE KPIs since all counters are available
	assert.Equal(t, 10, len(values))

	// Verify specific KPI values
	kpiMap := make(map[string]float64)
	for _, v := range values {
		kpiMap[v.KPIName] = v.KPIValue
	}

	assert.InDelta(t, 95.0, kpiMap["lte_rrc_setup_success_rate"], 0.01)
	assert.InDelta(t, 97.5, kpiMap["lte_erab_setup_success_rate"], 0.01)
	assert.InDelta(t, 5.0, kpiMap["lte_erab_drop_rate"], 0.01)
	assert.InDelta(t, 75.0, kpiMap["lte_dl_prb_utilization"], 0.01)
}

func TestKPIEngine_Calculate_MissingCounters(t *testing.T) {
	counterRepo := &mockCounterRepo{
		queryForKPIFunc: func(_ context.Context, _ uuid.UUID, _ string, _ []string, _, _ time.Time) (map[string]float64, error) {
			// Only return RRC counters, missing others
			return map[string]float64{
				"rrc_conn_setup_succ": 950,
				"rrc_conn_setup_att":  1000,
				"period_seconds":      900,
			}, nil
		},
	}
	kpiRepo := &mockKPIRepo{}
	engine := newTestEngine(counterRepo, kpiRepo)

	deviceID := uuid.New()
	startTime := time.Now().Add(-15 * time.Minute)
	endTime := time.Now()

	values, err := engine.Calculate(
		context.Background(),
		deviceID, "Cell1",
		startTime, endTime,
		model.CarrierCMCC, model.TechLTE,
	)
	require.NoError(t, err)

	// Only the KPIs whose counters are available should be computed
	// rrc_conn_setup_succ/att -> lte_rrc_setup_success_rate should be there
	found := false
	for _, v := range values {
		if v.KPIName == "lte_rrc_setup_success_rate" {
			found = true
			assert.InDelta(t, 95.0, v.KPIValue, 0.01)
		}
	}
	assert.True(t, found, "expected lte_rrc_setup_success_rate to be calculated")

	// KPIs requiring missing counters should be skipped
	for _, v := range values {
		assert.NotEqual(t, "lte_erab_setup_success_rate", v.KPIName,
			"should not compute erab KPI with missing counters")
	}
}

func TestKPIEngine_CalculateAndStore(t *testing.T) {
	counterRepo := &mockCounterRepo{
		queryForKPIFunc: func(_ context.Context, _ uuid.UUID, _ string, _ []string, _, _ time.Time) (map[string]float64, error) {
			return map[string]float64{
				"rrc_conn_setup_succ": 950,
				"rrc_conn_setup_att":  1000,
				"period_seconds":      900,
			}, nil
		},
	}
	kpiRepo := &mockKPIRepo{}
	engine := newTestEngine(counterRepo, kpiRepo)

	deviceID := uuid.New()
	collectTime := time.Now()

	values, err := engine.CalculateAndStore(
		context.Background(),
		deviceID, "Cell1",
		collectTime,
		model.CarrierCMCC, model.TechLTE,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, values)

	// Verify values were persisted
	assert.Equal(t, len(values), len(kpiRepo.insertedValues))
}

func TestKPIEngine_Calculate_NoApplicableFormulas(t *testing.T) {
	counterRepo := &mockCounterRepo{}
	kpiRepo := &mockKPIRepo{}
	engine := newTestEngine(counterRepo, kpiRepo)

	deviceID := uuid.New()
	startTime := time.Now().Add(-15 * time.Minute)
	endTime := time.Now()

	// Use a carrier code that has no registered formulas
	values, err := engine.Calculate(
		context.Background(),
		deviceID, "Cell1",
		startTime, endTime,
		model.CarrierCTCC, model.TechLTE, // CTCC not registered
	)
	require.NoError(t, err)
	assert.Nil(t, values)
}
