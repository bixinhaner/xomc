package license

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// --- Mock Repository ---

type mockLicenseRepo struct {
	createFn    func(ctx context.Context, lic *License) error
	getByIDFn   func(ctx context.Context, id uuid.UUID) (*License, error)
	getByCodeFn func(ctx context.Context, code string) (*License, error)
	updateFn    func(ctx context.Context, lic *License) error
	listFn      func(ctx context.Context, filter LicenseFilter) (*model.ListResponse[License], error)
	summaryFn   func(ctx context.Context) (*LicenseSummary, error)
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
	result, err := svc.Activate(context.Background(), licCode)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, StatusActive, result.Status)
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
	result, err := svc.Activate(context.Background(), licCode)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, StatusActive, result.Status, "should return same license (idempotent)")
	assert.Equal(t, lic.ID, result.ID)
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
	result, err := svc.Activate(context.Background(), licCode)

	require.Error(t, err)
	assert.Nil(t, result)
	var bErr *commonerrors.BusinessError
	assert.True(t, errors.As(err, &bErr))
	assert.Equal(t, 9100, bErr.Code)
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
