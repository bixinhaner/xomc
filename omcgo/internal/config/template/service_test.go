package template

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// --- mock repository ---

type mockTemplateRepo struct {
	findBestMatchFn    func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, templateType TemplateType) (*ConfigTemplate, error)
	listFn             func(ctx context.Context, filter ConfigTemplateFilter) (*model.ListResponse[ConfigTemplate], error)
	createFn           func(ctx context.Context, t *ConfigTemplate) error
	getByIDFn          func(ctx context.Context, id uuid.UUID) (*ConfigTemplate, error)
	updateFn           func(ctx context.Context, t *ConfigTemplate) error
	deleteFn           func(ctx context.Context, id uuid.UUID) error
	findByCarrierTechFn func(ctx context.Context, carrier model.CarrierCode, tech model.Technology, templateType TemplateType) ([]ConfigTemplate, error)
}

func (m *mockTemplateRepo) Create(ctx context.Context, t *ConfigTemplate) error {
	if m.createFn != nil {
		return m.createFn(ctx, t)
	}
	return nil
}

func (m *mockTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*ConfigTemplate, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTemplateRepo) Update(ctx context.Context, t *ConfigTemplate) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, t)
	}
	return nil
}

func (m *mockTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockTemplateRepo) List(ctx context.Context, filter ConfigTemplateFilter) (*model.ListResponse[ConfigTemplate], error) {
	if m.listFn != nil {
		return m.listFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockTemplateRepo) FindByCarrierTech(ctx context.Context, carrier model.CarrierCode, tech model.Technology, templateType TemplateType) ([]ConfigTemplate, error) {
	if m.findByCarrierTechFn != nil {
		return m.findByCarrierTechFn(ctx, carrier, tech, templateType)
	}
	return nil, nil
}

func (m *mockTemplateRepo) FindBestMatch(ctx context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, templateType TemplateType) (*ConfigTemplate, error) {
	if m.findBestMatchFn != nil {
		return m.findBestMatchFn(ctx, carrier, tech, productClass, templateType)
	}
	return nil, nil
}

// --- tests ---

func TestConfigTemplateService_Match_NilDevice(t *testing.T) {
	repo := &mockTemplateRepo{}
	svc := NewConfigTemplateService(repo, zap.NewNop())

	result, err := svc.Match(context.Background(), nil)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "device is nil")
}

func TestConfigTemplateService_Match_Found(t *testing.T) {
	expectedTemplate := &ConfigTemplate{
		ID:           uuid.New(),
		Name:         "cmcc-lte-provision",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
		ProductClass: "SmallCell",
		TemplateType: TemplateProvisioning,
		Active:       true,
	}

	repo := &mockTemplateRepo{
		findBestMatchFn: func(_ context.Context, carrier model.CarrierCode, tech model.Technology, productClass string, templateType TemplateType) (*ConfigTemplate, error) {
			assert.Equal(t, model.CarrierCMCC, carrier)
			assert.Equal(t, model.TechLTE, tech)
			assert.Equal(t, "SmallCell", productClass)
			assert.Equal(t, TemplateProvisioning, templateType)
			return expectedTemplate, nil
		},
	}
	svc := NewConfigTemplateService(repo, zap.NewNop())

	device := &model.Device{
		SerialNumber: "DEV001",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
		ProductClass: "SmallCell",
	}

	result, err := svc.Match(context.Background(), device)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expectedTemplate.ID, result.ID)
	assert.Equal(t, expectedTemplate.Name, result.Name)
}

func TestConfigTemplateService_Match_NotFound(t *testing.T) {
	repo := &mockTemplateRepo{
		findBestMatchFn: func(_ context.Context, _ model.CarrierCode, _ model.Technology, _ string, _ TemplateType) (*ConfigTemplate, error) {
			return nil, nil
		},
	}
	svc := NewConfigTemplateService(repo, zap.NewNop())

	device := &model.Device{
		SerialNumber: "DEV002",
		Carrier:      model.CarrierCTCC,
		Technology:   model.TechNR,
	}

	result, err := svc.Match(context.Background(), device)

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestConfigTemplateService_Match_RepoError(t *testing.T) {
	repoErr := errors.New("database connection lost")

	repo := &mockTemplateRepo{
		findBestMatchFn: func(_ context.Context, _ model.CarrierCode, _ model.Technology, _ string, _ TemplateType) (*ConfigTemplate, error) {
			return nil, repoErr
		},
	}
	svc := NewConfigTemplateService(repo, zap.NewNop())

	device := &model.Device{
		SerialNumber: "DEV003",
		Carrier:      model.CarrierCUCC,
		Technology:   model.TechLTE,
	}

	result, err := svc.Match(context.Background(), device)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, repoErr)
	assert.Contains(t, err.Error(), "DEV003")
}

func TestConfigTemplateService_ListTemplates(t *testing.T) {
	expectedResp := &model.ListResponse[ConfigTemplate]{
		Items: []ConfigTemplate{
			{ID: uuid.New(), Name: "template-1"},
			{ID: uuid.New(), Name: "template-2"},
		},
		Total:      2,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	filter := ConfigTemplateFilter{
		Carrier:    model.CarrierCMCC,
		Technology: model.TechLTE,
	}

	repo := &mockTemplateRepo{
		listFn: func(_ context.Context, f ConfigTemplateFilter) (*model.ListResponse[ConfigTemplate], error) {
			assert.Equal(t, filter.Carrier, f.Carrier)
			assert.Equal(t, filter.Technology, f.Technology)
			return expectedResp, nil
		},
	}
	svc := NewConfigTemplateService(repo, zap.NewNop())

	result, err := svc.ListTemplates(context.Background(), filter)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, expectedResp.Items[0].Name, result.Items[0].Name)
}
