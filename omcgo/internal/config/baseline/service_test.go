package baseline

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/model"
)

// --- mock repositories ---

type mockBaselineRepo struct {
	createFn  func(ctx context.Context, baseline *BaselineConfig) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*BaselineConfig, error)
	updateFn  func(ctx context.Context, baseline *BaselineConfig) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
	listFn    func(ctx context.Context, filter BaselineFilter) (*model.ListResponse[BaselineConfig], error)
}

func (m *mockBaselineRepo) Create(ctx context.Context, baseline *BaselineConfig) error {
	if m.createFn != nil {
		return m.createFn(ctx, baseline)
	}
	return nil
}

func (m *mockBaselineRepo) GetByID(ctx context.Context, id uuid.UUID) (*BaselineConfig, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockBaselineRepo) Update(ctx context.Context, baseline *BaselineConfig) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, baseline)
	}
	return nil
}

func (m *mockBaselineRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockBaselineRepo) List(ctx context.Context, filter BaselineFilter) (*model.ListResponse[BaselineConfig], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

type mockTaskRepo struct {
	createFn func(ctx context.Context, task *ConfigTask) error
	listFn   func(ctx context.Context, filter ConfigTaskFilter) (*model.ListResponse[ConfigTask], error)
}

func (m *mockTaskRepo) Create(ctx context.Context, task *ConfigTask) error {
	if m.createFn != nil {
		return m.createFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepo) List(ctx context.Context, filter ConfigTaskFilter) (*model.ListResponse[ConfigTask], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

type mockNeighborRepo struct {
	listFn func(ctx context.Context, filter NeighborFilter) (*model.ListResponse[NeighborParam], error)
}

func (m *mockNeighborRepo) List(ctx context.Context, filter NeighborFilter) (*model.ListResponse[NeighborParam], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

// helper to build the service with mocks
func newTestService(br *mockBaselineRepo, tr *mockTaskRepo, nr *mockNeighborRepo) *Service {
	if br == nil {
		br = &mockBaselineRepo{}
	}
	if tr == nil {
		tr = &mockTaskRepo{}
	}
	if nr == nil {
		nr = &mockNeighborRepo{}
	}
	return NewService(br, tr, nr, zap.NewNop())
}

// --- baseline tests ---

func TestService_CreateBaseline(t *testing.T) {
	var captured *BaselineConfig

	br := &mockBaselineRepo{
		createFn: func(_ context.Context, b *BaselineConfig) error {
			captured = b
			b.ID = uuid.New() // simulate DB assigning ID
			return nil
		},
	}
	svc := newTestService(br, nil, nil)

	baseline := &BaselineConfig{
		BaselineName: "test-baseline",
	}

	result, err := svc.CreateBaseline(context.Background(), baseline)

	require.NoError(t, err)
	require.NotNil(t, result)
	// status defaults to "draft"
	assert.Equal(t, BaselineDraft, captured.Status)
	// params defaults to "[]"
	assert.Equal(t, json.RawMessage("[]"), captured.Params)
	assert.Equal(t, "test-baseline", captured.BaselineName)
}

func TestService_CreateBaseline_PreservesExplicitValues(t *testing.T) {
	br := &mockBaselineRepo{
		createFn: func(_ context.Context, b *BaselineConfig) error {
			return nil
		},
	}
	svc := newTestService(br, nil, nil)

	params := json.RawMessage(`[{"name":"param1","value":"val1"}]`)
	baseline := &BaselineConfig{
		BaselineName: "explicit-baseline",
		Status:       BaselineActive,
		Params:       params,
	}

	result, err := svc.CreateBaseline(context.Background(), baseline)

	require.NoError(t, err)
	require.NotNil(t, result)
	// explicit status should be preserved
	assert.Equal(t, BaselineActive, result.Status)
	// explicit params should be preserved
	assert.JSONEq(t, `[{"name":"param1","value":"val1"}]`, string(result.Params))
}

func TestService_GetBaseline(t *testing.T) {
	id := uuid.New()
	expected := &BaselineConfig{
		ID:           id,
		BaselineName: "found-baseline",
		Status:       BaselineDraft,
	}

	br := &mockBaselineRepo{
		getByIDFn: func(_ context.Context, gotID uuid.UUID) (*BaselineConfig, error) {
			assert.Equal(t, id, gotID)
			return expected, nil
		},
	}
	svc := newTestService(br, nil, nil)

	result, err := svc.GetBaseline(context.Background(), id)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expected.ID, result.ID)
	assert.Equal(t, expected.BaselineName, result.BaselineName)
}

func TestService_UpdateBaseline(t *testing.T) {
	id := uuid.New()
	existingParams := json.RawMessage(`[{"name":"existing"}]`)

	existing := &BaselineConfig{
		ID:           id,
		BaselineName: "old-name",
		Status:       BaselineDraft,
		Params:       existingParams,
	}

	var updated *BaselineConfig

	br := &mockBaselineRepo{
		getByIDFn: func(_ context.Context, gotID uuid.UUID) (*BaselineConfig, error) {
			assert.Equal(t, id, gotID)
			return existing, nil
		},
		updateFn: func(_ context.Context, b *BaselineConfig) error {
			updated = b
			return nil
		},
	}
	svc := newTestService(br, nil, nil)

	desc := "new description"
	input := &BaselineConfig{
		BaselineName: "new-name",
		Description:  &desc,
		// Params is nil -> should preserve existing params
	}

	result, err := svc.UpdateBaseline(context.Background(), id, input)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "new-name", updated.BaselineName)
	assert.Equal(t, &desc, updated.Description)
	// Params should be preserved from existing when input.Params is nil
	assert.JSONEq(t, `[{"name":"existing"}]`, string(updated.Params))
}

func TestService_UpdateBaseline_OverridesParams(t *testing.T) {
	id := uuid.New()

	existing := &BaselineConfig{
		ID:           id,
		BaselineName: "old-name",
		Status:       BaselineDraft,
		Params:       json.RawMessage(`[{"name":"old"}]`),
	}

	var updated *BaselineConfig

	br := &mockBaselineRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*BaselineConfig, error) {
			return existing, nil
		},
		updateFn: func(_ context.Context, b *BaselineConfig) error {
			updated = b
			return nil
		},
	}
	svc := newTestService(br, nil, nil)

	newParams := json.RawMessage(`[{"name":"new"}]`)
	input := &BaselineConfig{
		BaselineName: "updated-name",
		Params:       newParams,
	}

	result, err := svc.UpdateBaseline(context.Background(), id, input)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.JSONEq(t, `[{"name":"new"}]`, string(updated.Params))
}

func TestService_DeleteBaseline(t *testing.T) {
	id := uuid.New()
	deleteCalled := false

	br := &mockBaselineRepo{
		deleteFn: func(_ context.Context, gotID uuid.UUID) error {
			assert.Equal(t, id, gotID)
			deleteCalled = true
			return nil
		},
	}
	svc := newTestService(br, nil, nil)

	err := svc.DeleteBaseline(context.Background(), id)

	require.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestService_ListBaselines(t *testing.T) {
	expectedResp := &model.ListResponse[BaselineConfig]{
		Items: []BaselineConfig{
			{ID: uuid.New(), BaselineName: "b1"},
			{ID: uuid.New(), BaselineName: "b2"},
		},
		Total:      2,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	statusFilter := BaselineDraft
	filter := BaselineFilter{
		Status: &statusFilter,
	}

	br := &mockBaselineRepo{
		listFn: func(_ context.Context, f BaselineFilter) (*model.ListResponse[BaselineConfig], error) {
			require.NotNil(t, f.Status)
			assert.Equal(t, BaselineDraft, *f.Status)
			return expectedResp, nil
		},
	}
	svc := newTestService(br, nil, nil)

	result, err := svc.ListBaselines(context.Background(), filter)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.Items, 2)
}

// --- config task tests ---

func TestService_CreateConfigTask(t *testing.T) {
	var captured *ConfigTask

	tr := &mockTaskRepo{
		createFn: func(_ context.Context, task *ConfigTask) error {
			captured = task
			task.ID = uuid.New()
			return nil
		},
	}
	svc := newTestService(nil, tr, nil)

	task := &ConfigTask{
		TaskName: "batch-push",
		TaskType: ConfigTaskParamSync,
	}

	result, err := svc.CreateConfigTask(context.Background(), task)

	require.NoError(t, err)
	require.NotNil(t, result)
	// status forced to "pending"
	assert.Equal(t, ConfigTaskPending, captured.Status)
	// progress forced to 0
	assert.Equal(t, 0, captured.Progress)
	// nil DeviceSns defaults to "[]"
	assert.Equal(t, json.RawMessage("[]"), captured.DeviceSns)
}

func TestService_CreateConfigTask_PreservesDeviceSns(t *testing.T) {
	var captured *ConfigTask

	tr := &mockTaskRepo{
		createFn: func(_ context.Context, task *ConfigTask) error {
			captured = task
			return nil
		},
	}
	svc := newTestService(nil, tr, nil)

	deviceSns := json.RawMessage(`["SN001","SN002"]`)
	task := &ConfigTask{
		TaskName:  "targeted-push",
		TaskType:  ConfigTaskBatchConfig,
		DeviceSns: deviceSns,
	}

	result, err := svc.CreateConfigTask(context.Background(), task)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ConfigTaskPending, captured.Status)
	assert.JSONEq(t, `["SN001","SN002"]`, string(captured.DeviceSns))
}

func TestService_ListConfigTasks(t *testing.T) {
	expectedResp := &model.ListResponse[ConfigTask]{
		Items: []ConfigTask{
			{ID: uuid.New(), TaskName: "task-1"},
		},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	statusFilter := ConfigTaskPending
	filter := ConfigTaskFilter{
		Status: &statusFilter,
	}

	tr := &mockTaskRepo{
		listFn: func(_ context.Context, f ConfigTaskFilter) (*model.ListResponse[ConfigTask], error) {
			require.NotNil(t, f.Status)
			assert.Equal(t, ConfigTaskPending, *f.Status)
			return expectedResp, nil
		},
	}
	svc := newTestService(nil, tr, nil)

	result, err := svc.ListConfigTasks(context.Background(), filter)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(1), result.Total)
	assert.Len(t, result.Items, 1)
}

// --- neighbor tests ---

func TestService_ListNeighbors(t *testing.T) {
	expectedResp := &model.ListResponse[NeighborParam]{
		Items: []NeighborParam{
			{ID: uuid.New(), SourceCellID: "CELL-A", TargetCellID: "CELL-B", NeighborType: NeighborIntraFreq},
		},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	srcCell := "CELL-A"
	filter := NeighborFilter{
		SourceCellID: &srcCell,
	}

	nr := &mockNeighborRepo{
		listFn: func(_ context.Context, f NeighborFilter) (*model.ListResponse[NeighborParam], error) {
			require.NotNil(t, f.SourceCellID)
			assert.Equal(t, "CELL-A", *f.SourceCellID)
			return expectedResp, nil
		},
	}
	svc := newTestService(nil, nil, nr)

	result, err := svc.ListNeighbors(context.Background(), filter)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(1), result.Total)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, NeighborIntraFreq, result.Items[0].NeighborType)
}
