package report

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/model"
)

// ---------------------------------------------------------------------------
// Mock repositories (function-field pattern)
// ---------------------------------------------------------------------------

type mockDefRepo struct {
	createFn  func(ctx context.Context, def *ReportDefinition) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*ReportDefinition, error)
	updateFn  func(ctx context.Context, def *ReportDefinition) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
	listFn    func(ctx context.Context, filter DefinitionFilter) (*model.ListResponse[ReportDefinition], error)
}

func (m *mockDefRepo) Create(ctx context.Context, def *ReportDefinition) error {
	if m.createFn != nil {
		return m.createFn(ctx, def)
	}
	return nil
}

func (m *mockDefRepo) GetByID(ctx context.Context, id uuid.UUID) (*ReportDefinition, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockDefRepo) Update(ctx context.Context, def *ReportDefinition) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, def)
	}
	return nil
}

func (m *mockDefRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockDefRepo) List(ctx context.Context, filter DefinitionFilter) (*model.ListResponse[ReportDefinition], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

type mockRecordRepo struct {
	createFn  func(ctx context.Context, record *ReportRecord) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*ReportRecord, error)
	updateFn  func(ctx context.Context, record *ReportRecord) error
	listFn    func(ctx context.Context, filter RecordFilter) (*model.ListResponse[ReportRecord], error)
}

func (m *mockRecordRepo) Create(ctx context.Context, record *ReportRecord) error {
	if m.createFn != nil {
		return m.createFn(ctx, record)
	}
	return nil
}

func (m *mockRecordRepo) GetByID(ctx context.Context, id uuid.UUID) (*ReportRecord, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockRecordRepo) Update(ctx context.Context, record *ReportRecord) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, record)
	}
	return nil
}

func (m *mockRecordRepo) List(ctx context.Context, filter RecordFilter) (*model.ListResponse[ReportRecord], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestService(defRepo *mockDefRepo, recordRepo *mockRecordRepo) *Service {
	return NewService(defRepo, recordRepo, nil, zap.NewNop())
}

// ---------------------------------------------------------------------------
// Definition Tests
// ---------------------------------------------------------------------------

func TestService_CreateDefinition(t *testing.T) {
	var saved *ReportDefinition
	defRepo := &mockDefRepo{
		createFn: func(_ context.Context, def *ReportDefinition) error {
			saved = def
			return nil
		},
	}

	svc := newTestService(defRepo, &mockRecordRepo{})

	def := &ReportDefinition{
		ID:         uuid.New(),
		ReportName: "Daily Performance Report",
		ReportType: ReportPerformance,
		Period:     PeriodDaily,
		Creator:    "admin",
	}

	result, err := svc.CreateDefinition(context.Background(), def)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Defaults applied
	assert.Equal(t, ReportDraft, saved.Status, "status should default to draft")
	assert.Equal(t, []string{"pdf"}, saved.Format, "format should default to [pdf]")
	assert.Equal(t, []string{}, saved.KPICodes, "KPICodes should default to empty slice")
	assert.Equal(t, []string{}, saved.DeviceGroups, "DeviceGroups should default to empty slice")
	assert.Equal(t, "Daily Performance Report", saved.ReportName)
	assert.Equal(t, ReportPerformance, saved.ReportType)
}

func TestService_GetDefinition(t *testing.T) {
	id := uuid.New()
	expected := &ReportDefinition{ID: id, ReportName: "Test Report"}

	defRepo := &mockDefRepo{
		getByIDFn: func(_ context.Context, qID uuid.UUID) (*ReportDefinition, error) {
			assert.Equal(t, id, qID)
			return expected, nil
		},
	}

	svc := newTestService(defRepo, &mockRecordRepo{})
	result, err := svc.GetDefinition(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestService_UpdateDefinition(t *testing.T) {
	id := uuid.New()
	existing := &ReportDefinition{
		ID:             id,
		ReportName:     "Old Report",
		ReportType:     ReportAlarm,
		Description:    "Old description",
		Format:         []string{"pdf"},
		Period:         PeriodDaily,
		KPICodes:       []string{"KPI001"},
		DeviceGroups:   []string{"group-a"},
		AutoGenerate:   false,
		CronExpression: "0 0 * * *",
		Status:         ReportDraft,
		Creator:        "old-admin",
	}

	var updated *ReportDefinition
	defRepo := &mockDefRepo{
		getByIDFn: func(_ context.Context, qID uuid.UUID) (*ReportDefinition, error) {
			assert.Equal(t, id, qID)
			return existing, nil
		},
		updateFn: func(_ context.Context, def *ReportDefinition) error {
			updated = def
			return nil
		},
	}

	svc := newTestService(defRepo, &mockRecordRepo{})

	patch := &ReportDefinition{
		ReportName:     "New Report",
		ReportType:     ReportPerformance,
		Description:    "New description",
		Format:         []string{"pdf", "csv"},
		Period:         PeriodWeekly,
		KPICodes:       []string{"KPI001", "KPI002"},
		DeviceGroups:   []string{"group-a", "group-b"},
		AutoGenerate:   true,
		CronExpression: "0 8 * * 1",
		Status:         ReportPublished,
		Creator:        "new-admin",
	}

	result, err := svc.UpdateDefinition(context.Background(), id, patch)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "New Report", updated.ReportName)
	assert.Equal(t, ReportPerformance, updated.ReportType)
	assert.Equal(t, "New description", updated.Description)
	assert.Equal(t, []string{"pdf", "csv"}, updated.Format)
	assert.Equal(t, PeriodWeekly, updated.Period)
	assert.Equal(t, []string{"KPI001", "KPI002"}, updated.KPICodes)
	assert.Equal(t, []string{"group-a", "group-b"}, updated.DeviceGroups)
	assert.True(t, updated.AutoGenerate)
	assert.Equal(t, "0 8 * * 1", updated.CronExpression)
	assert.Equal(t, ReportPublished, updated.Status)
	assert.Equal(t, "new-admin", updated.Creator)
}

func TestService_DeleteDefinition(t *testing.T) {
	id := uuid.New()
	var deletedID uuid.UUID

	defRepo := &mockDefRepo{
		deleteFn: func(_ context.Context, qID uuid.UUID) error {
			deletedID = qID
			return nil
		},
	}

	svc := newTestService(defRepo, &mockRecordRepo{})
	err := svc.DeleteDefinition(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, id, deletedID, "should delegate correct ID to repo")
}

func TestService_ListDefinitions(t *testing.T) {
	expected := &model.ListResponse[ReportDefinition]{
		Items:      []ReportDefinition{{ID: uuid.New(), ReportName: "Report A"}},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	defRepo := &mockDefRepo{
		listFn: func(_ context.Context, filter DefinitionFilter) (*model.ListResponse[ReportDefinition], error) {
			return expected, nil
		},
	}

	svc := newTestService(defRepo, &mockRecordRepo{})
	result, err := svc.ListDefinitions(context.Background(), DefinitionFilter{})

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// ---------------------------------------------------------------------------
// Record Tests
// ---------------------------------------------------------------------------

func TestService_ListRecords(t *testing.T) {
	expected := &model.ListResponse[ReportRecord]{
		Items:      []ReportRecord{{ID: uuid.New(), ReportName: "Report-2025-01"}},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	recordRepo := &mockRecordRepo{
		listFn: func(_ context.Context, filter RecordFilter) (*model.ListResponse[ReportRecord], error) {
			return expected, nil
		},
	}

	svc := newTestService(&mockDefRepo{}, recordRepo)
	result, err := svc.ListRecords(context.Background(), RecordFilter{})

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestService_GetRecord(t *testing.T) {
	id := uuid.New()
	expected := &ReportRecord{ID: id, ReportName: "Report-2025-01"}

	recordRepo := &mockRecordRepo{
		getByIDFn: func(_ context.Context, qID uuid.UUID) (*ReportRecord, error) {
			assert.Equal(t, id, qID)
			return expected, nil
		},
	}

	svc := newTestService(&mockDefRepo{}, recordRepo)
	result, err := svc.GetRecord(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// ---------------------------------------------------------------------------
// Generate Test
// ---------------------------------------------------------------------------

func TestService_Generate(t *testing.T) {
	defID := uuid.New()
	defName := "Performance Report"

	var savedRecord *ReportRecord
	var defUpdated *ReportDefinition

	defRepo := &mockDefRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*ReportDefinition, error) {
			return &ReportDefinition{
				ID:         defID,
				ReportName: defName,
				Format:     []string{"csv", "pdf"},
				Status:     ReportPublished,
			}, nil
		},
		updateFn: func(_ context.Context, def *ReportDefinition) error {
			defUpdated = def
			return nil
		},
	}

	recordRepo := &mockRecordRepo{
		createFn: func(_ context.Context, record *ReportRecord) error {
			savedRecord = record
			return nil
		},
	}

	svc := newTestService(defRepo, recordRepo)
	result, err := svc.Generate(context.Background(), defID, "2025-01")

	require.NoError(t, err)
	require.NotNil(t, result)

	// Record assertions
	assert.Equal(t, defID, savedRecord.ReportDefinitionID)
	assert.Equal(t, "Performance Report-2025-01", savedRecord.ReportName, "name should be built from definition name + period")
	assert.Equal(t, "2025-01", savedRecord.Period)
	assert.Equal(t, RecordGenerating, savedRecord.Status, "status should be generating")
	assert.Equal(t, "csv", savedRecord.Format, "format should be first element of definition.Format")
	assert.False(t, savedRecord.GenerateTime.IsZero(), "GenerateTime should be set")

	// Definition's LastGenTime should be updated
	require.NotNil(t, defUpdated)
	require.NotNil(t, defUpdated.LastGenTime, "LastGenTime should be set on definition")
}
