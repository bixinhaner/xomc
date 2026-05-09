package license

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// --- Mock Repository ---

type mockLicenseRepo struct {
	createFn       func(ctx context.Context, lic *License) error
	getByIDFn      func(ctx context.Context, id uuid.UUID) (*License, error)
	getByCodeFn    func(ctx context.Context, code string) (*License, error)
	updateFn       func(ctx context.Context, lic *License) error
	listFn         func(ctx context.Context, filter LicenseFilter) (*model.ListResponse[License], error)
	summaryFn      func(ctx context.Context) (*LicenseSummary, error)
	getActiveFn    func(ctx context.Context) (*License, error)
	listActiveFn   func(ctx context.Context) ([]*License, error)
	listByDimFn    func(ctx context.Context, deviceType *string, region *string) ([]*License, error)
	countDevFn     func(ctx context.Context) (int, error)
	markExpFn      func(ctx context.Context, id uuid.UUID) error
	updCapAlertFn  func(ctx context.Context, id uuid.UUID, threshold int, at time.Time) error
}

func (m *mockLicenseRepo) Create(ctx context.Context, lic *License) error {
	if m.createFn != nil {
		return m.createFn(ctx, lic)
	}
	return nil
}

func (m *mockLicenseRepo) GetByID(ctx context.Context, id uuid.UUID) (*License, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockLicenseRepo) GetByCode(ctx context.Context, code string) (*License, error) {
	if m.getByCodeFn != nil {
		return m.getByCodeFn(ctx, code)
	}
	return nil, nil
}

func (m *mockLicenseRepo) Update(ctx context.Context, lic *License) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, lic)
	}
	return nil
}

func (m *mockLicenseRepo) List(ctx context.Context, filter LicenseFilter) (*model.ListResponse[License], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockLicenseRepo) Summary(ctx context.Context) (*LicenseSummary, error) {
	if m.summaryFn != nil {
		return m.summaryFn(ctx)
	}
	return nil, nil
}

func (m *mockLicenseRepo) GetActiveLicenseWithMaxDevices(ctx context.Context) (*License, error) {
	if m.getActiveFn != nil {
		return m.getActiveFn(ctx)
	}
	return nil, nil
}

func (m *mockLicenseRepo) ListActiveLicenses(ctx context.Context) ([]*License, error) {
	if m.listActiveFn != nil {
		return m.listActiveFn(ctx)
	}
	return nil, nil
}

func (m *mockLicenseRepo) ListActiveByDimension(
	ctx context.Context, deviceType *string, region *string,
) ([]*License, error) {
	if m.listByDimFn != nil {
		return m.listByDimFn(ctx, deviceType, region)
	}
	return nil, nil
}

func (m *mockLicenseRepo) CountDevices(ctx context.Context) (int, error) {
	if m.countDevFn != nil {
		return m.countDevFn(ctx)
	}
	return 0, nil
}

func (m *mockLicenseRepo) MarkExpired(ctx context.Context, id uuid.UUID) error {
	if m.markExpFn != nil {
		return m.markExpFn(ctx, id)
	}
	return nil
}

func (m *mockLicenseRepo) UpdateCapacityAlert(ctx context.Context, id uuid.UUID, threshold int, at time.Time) error {
	if m.updCapAlertFn != nil {
		return m.updCapAlertFn(ctx, id, threshold, at)
	}
	return nil
}

// --- Helper ---

func newTestService(repo *mockLicenseRepo) *Service {
	return NewService(repo, zap.NewNop())
}

// --- Tests: List ---

func TestService_List(t *testing.T) {
	expected := &model.ListResponse[License]{
		Items:      []License{{ID: uuid.New(), LicenseCode: "LIC-001", Status: StatusActive}},
		Total:      1,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	filter := LicenseFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 20},
	}

	repo := &mockLicenseRepo{
		listFn: func(ctx context.Context, f LicenseFilter) (*model.ListResponse[License], error) {
			assert.Equal(t, filter, f)
			return expected, nil
		},
	}

	svc := newTestService(repo)
	result, err := svc.List(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// --- Tests: GetByID ---

func TestService_GetByID(t *testing.T) {
	licID := uuid.New()
	expected := &License{
		ID:          licID,
		LicenseCode: "LIC-001",
		Status:      StatusActive,
	}

	repo := &mockLicenseRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*License, error) {
			assert.Equal(t, licID, id)
			return expected, nil
		},
	}

	svc := newTestService(repo)
	result, err := svc.GetByID(context.Background(), licID)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// --- Tests: Summary ---

func TestService_Summary(t *testing.T) {
	expected := &LicenseSummary{
		Total:        100,
		Active:       60,
		Expired:      20,
		Pending:      15,
		ExpiringSoon: 5,
	}

	repo := &mockLicenseRepo{
		summaryFn: func(ctx context.Context) (*LicenseSummary, error) {
			return expected, nil
		},
	}

	svc := newTestService(repo)
	result, err := svc.Summary(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expected, result)
	// 未注入 logRepo 时 EnforcementHits7d=0（默认值）
	assert.Equal(t, int64(0), result.EnforcementHits7d)
}

// T-0100-P2: Summary 卡 enforcement_hits_7d 字段。
func TestService_Summary_WithEnforcementHits(t *testing.T) {
	repo := &mockLicenseRepo{
		summaryFn: func(ctx context.Context) (*LicenseSummary, error) {
			return &LicenseSummary{Total: 10, Active: 8}, nil
		},
	}
	logRepo := newMemLogRepo()
	// 喂 3 条 denied + 1 条 success（不计入）
	licID := uuid.New()
	w := NewLogWriter(logRepo, zap.NewNop())
	for i := 0; i < 3; i++ {
		w.Write(context.Background(), LicenseLogEntry{
			LicenseID: &licID,
			LogType:   LogTypeEnforcementCapacity,
			Result:    LogResultDenied,
		})
	}
	w.Write(context.Background(), LicenseLogEntry{
		LicenseID: &licID,
		LogType:   LogTypeImport,
		Result:    LogResultSuccess,
	})

	svc := newTestService(repo)
	svc.SetLogRepo(logRepo)

	result, err := svc.Summary(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.EnforcementHits7d, "应只计入 denied 记录")
	assert.Equal(t, int64(10), result.Total, "其它字段不变")
}

// LogRepo 失败时退化为 0，不阻断 Summary
func TestService_Summary_LogRepoErrorDegradesToZero(t *testing.T) {
	repo := &mockLicenseRepo{
		summaryFn: func(ctx context.Context) (*LicenseSummary, error) {
			return &LicenseSummary{Total: 5}, nil
		},
	}
	logRepo := newMemLogRepo()
	logRepo.failOn[LogTypeImport] = nil // 占位，failOn 不直接控制 CountDenialsSince
	// memLogRepo.CountDenialsSince 不会失败；用一个手动 stub 验证 degraded 路径
	svc := newTestService(repo)
	svc.SetLogRepo(&failingLogRepo{})

	result, err := svc.Summary(context.Background())
	require.NoError(t, err, "Summary 主体不应受 logRepo 错误影响")
	assert.Equal(t, int64(0), result.EnforcementHits7d, "降级为 0")
	assert.Equal(t, int64(5), result.Total)
}

// failingLogRepo 用于测试 logRepo 错误降级路径。
type failingLogRepo struct{}

func (failingLogRepo) Create(_ context.Context, _ *LicenseLog) error {
	return nil
}
func (failingLogRepo) List(_ context.Context, _ LicenseLogFilter) (*model.ListResponse[LicenseLog], error) {
	return nil, nil
}
func (failingLogRepo) ListByLicense(_ context.Context, _ uuid.UUID, _ int) ([]LicenseLog, error) {
	return nil, nil
}
func (failingLogRepo) CountDenialsSince(_ context.Context, _ time.Time) (int64, error) {
	return 0, errors.New("simulated db down")
}
func (failingLogRepo) ListBefore(_ context.Context, _ time.Time, _ int) ([]LicenseLog, error) {
	return nil, nil
}
func (failingLogRepo) DeleteBefore(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

// --- Tests: Activate ---

func TestService_Activate_Success(t *testing.T) {
	licCode := "LIC-PENDING-001"
	lic := &License{
		ID:          uuid.New(),
		LicenseCode: licCode,
		Status:      StatusPending,
	}

	var updatedLic *License
	repo := &mockLicenseRepo{
		getByCodeFn: func(ctx context.Context, code string) (*License, error) {
			assert.Equal(t, licCode, code)
			return lic, nil
		},
		updateFn: func(ctx context.Context, l *License) error {
			updatedLic = l
			return nil
		},
	}

	svc := newTestService(repo)
	result, err := svc.Activate(context.Background(), licCode, false)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Activated)
	assert.Equal(t, StatusActive, result.Activated.Status)
	assert.Empty(t, result.AutoRevoked, "no same-dimension conflict expected")
	require.NotNil(t, updatedLic)
	assert.Equal(t, StatusActive, updatedLic.Status)
}

func TestService_Activate_AlreadyActive(t *testing.T) {
	licCode := "LIC-ACTIVE-001"
	lic := &License{
		ID:          uuid.New(),
		LicenseCode: licCode,
		Status:      StatusActive,
	}

	repo := &mockLicenseRepo{
		getByCodeFn: func(ctx context.Context, code string) (*License, error) {
			return lic, nil
		},
		// updateFn intentionally not set; should NOT be called
	}

	svc := newTestService(repo)
	result, err := svc.Activate(context.Background(), licCode, false)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Activated)
	assert.Equal(t, StatusActive, result.Activated.Status, "should return same license (idempotent)")
	assert.Equal(t, lic.ID, result.Activated.ID)
}

func TestService_Activate_Revoked(t *testing.T) {
	licCode := "LIC-REVOKED-001"
	lic := &License{
		ID:          uuid.New(),
		LicenseCode: licCode,
		Status:      StatusRevoked,
	}

	repo := &mockLicenseRepo{
		getByCodeFn: func(ctx context.Context, code string) (*License, error) {
			return lic, nil
		},
	}

	svc := newTestService(repo)
	result, err := svc.Activate(context.Background(), licCode, false)

	require.Error(t, err)
	assert.Nil(t, result)
	var bErr *commonerrors.BusinessError
	assert.True(t, errors.As(err, &bErr))
	assert.Equal(t, 9100, bErr.Code)
}

// --- Tests: Activate same-dimension flow (T-0100-P3) ---

func TestService_Activate_SameDimensionConflict_NoForce(t *testing.T) {
	licCode := "LIC-NEW-001"
	deviceType := "pico"
	region := "huabei"
	target := &License{
		ID:          uuid.New(),
		LicenseCode: licCode,
		Status:      StatusPending,
		DeviceType:  &deviceType,
		Region:      &region,
	}
	old := &License{
		ID:          uuid.New(),
		LicenseCode: "LIC-OLD-001",
		LicenseName: "old",
		Status:      StatusActive,
		DeviceType:  &deviceType,
		Region:      &region,
	}

	updateCalls := 0
	repo := &mockLicenseRepo{
		getByCodeFn: func(_ context.Context, _ string) (*License, error) { return target, nil },
		listByDimFn: func(_ context.Context, dt *string, rg *string) ([]*License, error) {
			require.NotNil(t, dt)
			require.NotNil(t, rg)
			assert.Equal(t, deviceType, *dt)
			assert.Equal(t, region, *rg)
			return []*License{old}, nil
		},
		updateFn: func(_ context.Context, _ *License) error {
			updateCalls++
			return nil
		},
	}

	result, err := newTestService(repo).Activate(context.Background(), licCode, false)
	require.Error(t, err)
	assert.Nil(t, result)

	var conflict *SameDimensionConflictError
	require.True(t, errors.As(err, &conflict))
	assert.Len(t, conflict.Conflicting, 1)
	assert.Equal(t, old.ID, conflict.Conflicting[0].ID)
	assert.Equal(t, 0, updateCalls, "no Update should fire when force=false")
	// 校验 Unwrap → ErrAlreadyExists（HTTP 409 映射）
	assert.True(t, errors.Is(err, commonerrors.ErrAlreadyExists))
}

func TestService_Activate_SameDimensionConflict_Force(t *testing.T) {
	licCode := "LIC-NEW-002"
	deviceType := "pico"
	target := &License{
		ID:          uuid.New(),
		LicenseCode: licCode,
		Status:      StatusPending,
		DeviceType:  &deviceType,
	}
	old1 := &License{ID: uuid.New(), LicenseCode: "LIC-OLD-A", Status: StatusActive, DeviceType: &deviceType}
	old2 := &License{ID: uuid.New(), LicenseCode: "LIC-OLD-B", Status: StatusActive, DeviceType: &deviceType}

	var updates []*License
	repo := &mockLicenseRepo{
		getByCodeFn: func(_ context.Context, _ string) (*License, error) { return target, nil },
		listByDimFn: func(_ context.Context, _ *string, _ *string) ([]*License, error) {
			return []*License{old1, old2}, nil
		},
		updateFn: func(_ context.Context, l *License) error {
			updates = append(updates, l)
			return nil
		},
	}

	result, err := newTestService(repo).Activate(context.Background(), licCode, true)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Activated)
	assert.Equal(t, StatusActive, result.Activated.Status)
	assert.Len(t, result.AutoRevoked, 2)
	// 期望写入顺序：先 revoke 两条旧 active，再激活新 license（共 3 次 Update）
	require.Len(t, updates, 3)
	assert.Equal(t, StatusRevoked, updates[0].Status)
	assert.Equal(t, StatusRevoked, updates[1].Status)
	assert.Equal(t, StatusActive, updates[2].Status)
	assert.Equal(t, target.ID, updates[2].ID)
}

func TestService_Activate_SameDimension_NullBucket(t *testing.T) {
	// device_type 与 region 均 nil 也算同一桶；与 nil 桶冲突时 force=false 应 409。
	licCode := "LIC-NULL-001"
	target := &License{ID: uuid.New(), LicenseCode: licCode, Status: StatusPending}
	old := &License{ID: uuid.New(), LicenseCode: "LIC-NULL-OLD", Status: StatusActive}

	repo := &mockLicenseRepo{
		getByCodeFn: func(_ context.Context, _ string) (*License, error) { return target, nil },
		listByDimFn: func(_ context.Context, dt *string, rg *string) ([]*License, error) {
			assert.Nil(t, dt)
			assert.Nil(t, rg)
			return []*License{old}, nil
		},
	}

	_, err := newTestService(repo).Activate(context.Background(), licCode, false)
	require.Error(t, err)
	var conflict *SameDimensionConflictError
	require.True(t, errors.As(err, &conflict))
	assert.Nil(t, conflict.DeviceType)
	assert.Nil(t, conflict.Region)
}

// --- Tests: Revoke ---

func TestService_Revoke_Success(t *testing.T) {
	licID := uuid.New()
	lic := &License{
		ID:     licID,
		Status: StatusActive,
	}

	var updatedLic *License
	repo := &mockLicenseRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*License, error) {
			assert.Equal(t, licID, id)
			return lic, nil
		},
		updateFn: func(ctx context.Context, l *License) error {
			updatedLic = l
			return nil
		},
	}

	svc := newTestService(repo)
	err := svc.Revoke(context.Background(), licID)

	require.NoError(t, err)
	require.NotNil(t, updatedLic)
	assert.Equal(t, StatusRevoked, updatedLic.Status)
}

func TestService_Revoke_AlreadyRevoked(t *testing.T) {
	licID := uuid.New()
	lic := &License{
		ID:     licID,
		Status: StatusRevoked,
	}

	repo := &mockLicenseRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*License, error) {
			return lic, nil
		},
		// updateFn intentionally not set; should NOT be called
	}

	svc := newTestService(repo)
	err := svc.Revoke(context.Background(), licID)

	assert.NoError(t, err, "revoking an already-revoked license should be idempotent (return nil)")
}

// --- Tests: Import ---

func TestService_Import(t *testing.T) {
	var captured *License
	repo := &mockLicenseRepo{
		createFn: func(ctx context.Context, lic *License) error {
			captured = lic
			lic.ID = uuid.New()
			return nil
		},
	}

	svc := newTestService(repo)

	lic := &License{
		LicenseName: "Test License",
		LicenseCode: "LIC-NEW-001",
		ProductName: "OMC Pro",
		LicenseType: TypePerpetual,
		MaxDevices:  100,
		// Status and Features intentionally left zero/nil
	}

	result, err := svc.Import(context.Background(), lic)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, StatusPending, captured.Status, "empty status should default to pending")
	assert.NotNil(t, captured.Features, "nil features should default to []")
	assert.Equal(t, json.RawMessage("[]"), captured.Features)
}

func TestService_Import_WithExplicitStatus(t *testing.T) {
	repo := &mockLicenseRepo{
		createFn: func(ctx context.Context, lic *License) error {
			lic.ID = uuid.New()
			return nil
		},
	}

	svc := newTestService(repo)

	lic := &License{
		LicenseName: "Trial License",
		LicenseCode: "LIC-TRIAL-001",
		ProductName: "OMC Trial",
		LicenseType: TypeTrial,
		Status:      StatusTrial,
		MaxDevices:  10,
		Features:    json.RawMessage(`["feature_a"]`),
	}

	result, err := svc.Import(context.Background(), lic)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, StatusTrial, result.Status, "explicit status should be preserved")
	assert.Equal(t, json.RawMessage(`["feature_a"]`), result.Features, "explicit features should be preserved")
}

// TestService_Import_DuplicateCode_ReturnsAlreadyExists verifies that
// importing a license whose code already exists returns ErrAlreadyExists
// (which the HTTP layer maps to 409), not a raw 500.
//
// This guards against the W2.D.1.b T-0057 license/import bug where a
// unique-violation on license_code surfaced as 500.
func TestService_Import_DuplicateCode_ReturnsAlreadyExists(t *testing.T) {
	existing := &License{
		ID:          uuid.New(),
		LicenseCode: "LIC-DUP-001",
		LicenseName: "Existing",
		ProductName: "P",
		Status:      StatusActive,
	}

	createCalled := false
	repo := &mockLicenseRepo{
		getByCodeFn: func(ctx context.Context, code string) (*License, error) {
			assert.Equal(t, "LIC-DUP-001", code)
			return existing, nil
		},
		createFn: func(ctx context.Context, lic *License) error {
			createCalled = true
			return nil
		},
	}

	svc := newTestService(repo)

	lic := &License{
		LicenseName: "Duplicate Attempt",
		LicenseCode: "LIC-DUP-001",
		ProductName: "P",
		LicenseType: TypeSubscription,
	}

	result, err := svc.Import(context.Background(), lic)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errors.Is(err, commonerrors.ErrAlreadyExists),
		"duplicate license_code must surface ErrAlreadyExists; got: %v", err)
	assert.False(t, createCalled, "Create must not be called when license_code already exists")
}

// TestService_Import_MissingFields_ReturnsInvalidInput verifies field
// validation: missing license_code / license_name / product_name should
// surface ErrInvalidInput (→ 400), not 500 from a downstream NOT NULL
// constraint failure.
func TestService_Import_MissingFields_ReturnsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		lic  *License
	}{
		{
			name: "nil license",
			lic:  nil,
		},
		{
			name: "missing license_code",
			lic: &License{
				LicenseName: "Has Name",
				ProductName: "Has Product",
			},
		},
		{
			name: "missing license_name",
			lic: &License{
				LicenseCode: "LIC-X",
				ProductName: "Has Product",
			},
		},
		{
			name: "missing product_name",
			lic: &License{
				LicenseCode: "LIC-X",
				LicenseName: "Has Name",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockLicenseRepo{
				createFn: func(ctx context.Context, lic *License) error {
					t.Fatalf("Create must not be called for invalid input")
					return nil
				},
				getByCodeFn: func(ctx context.Context, code string) (*License, error) {
					t.Fatalf("GetByCode must not be called for invalid input")
					return nil, nil
				},
			}
			svc := newTestService(repo)

			result, err := svc.Import(context.Background(), tc.lic)

			require.Error(t, err)
			assert.Nil(t, result)
			assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput),
				"missing required field must surface ErrInvalidInput; got: %v", err)
		})
	}
}

// TestService_Import_NotFoundFromRepo_TreatedAsAvailable verifies that
// a repository returning ErrNotFound from GetByCode is interpreted as
// "the code is available" — Import should proceed to Create, not bubble
// the NotFound to the caller.
func TestService_Import_NotFoundFromRepo_TreatedAsAvailable(t *testing.T) {
	createCalled := false
	repo := &mockLicenseRepo{
		getByCodeFn: func(ctx context.Context, code string) (*License, error) {
			return nil, commonerrors.ErrNotFound
		},
		createFn: func(ctx context.Context, lic *License) error {
			createCalled = true
			lic.ID = uuid.New()
			return nil
		},
	}
	svc := newTestService(repo)

	lic := &License{
		LicenseName: "Fresh License",
		LicenseCode: "LIC-FRESH-001",
		ProductName: "P",
	}

	result, err := svc.Import(context.Background(), lic)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, createCalled, "Create should be called when GetByCode returns ErrNotFound")
}
