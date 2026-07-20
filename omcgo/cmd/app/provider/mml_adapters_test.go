package provider

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/mml"
	"github.com/omcgo/omcgo/internal/product"
)

type inactiveMMLParamRepo struct{}

func (inactiveMMLParamRepo) IsParamModelActive(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}
func (inactiveMMLParamRepo) ListMappingsByParamModel(context.Context, uuid.UUID) ([]parammodel.ParamMapping, error) {
	return []parammodel.ParamMapping{{StandardPath: "Device.Test.Param", PrivatePath: "Device.Private.Param"}}, nil
}
func (inactiveMMLParamRepo) ListDiscoveredMappings(context.Context, uuid.UUID, string) ([]parammodel.ParamMapping, error) {
	return []parammodel.ParamMapping{{StandardPath: "Device.Test.Param", PrivatePath: "Device.Private.Param"}}, nil
}

type activeMMLParamRepo struct {
	mappings []parammodel.ParamMapping
}

func (activeMMLParamRepo) IsParamModelActive(context.Context, uuid.UUID) (bool, error) {
	return true, nil
}
func (r activeMMLParamRepo) ListMappingsByParamModel(context.Context, uuid.UUID) ([]parammodel.ParamMapping, error) {
	return r.mappings, nil
}
func (activeMMLParamRepo) ListDiscoveredMappings(context.Context, uuid.UUID, string) ([]parammodel.ParamMapping, error) {
	return nil, nil
}

type mmlProductRepo struct {
	product *product.Product
	pattern product.ProductClassPattern
}

func (r mmlProductRepo) ListActivePatterns(context.Context) ([]product.ProductClassPattern, error) {
	return []product.ProductClassPattern{r.pattern}, nil
}
func (r mmlProductRepo) GetProductByID(_ context.Context, id uuid.UUID) (*product.Product, error) {
	if r.product == nil || r.product.ID != id {
		return nil, nil
	}
	copy := *r.product
	return &copy, nil
}
func (r mmlProductRepo) ListProducts(context.Context) ([]*product.Product, error) {
	return []*product.Product{r.product}, nil
}
func (mmlProductRepo) FetchIndicatorPlatformsByDeviceType(context.Context, string) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}
func (mmlProductRepo) FetchAlarmNeTypes(context.Context) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

func newActiveMMLPathTranslator(t *testing.T, mappings []parammodel.ParamMapping) mml.PathTranslator {
	t.Helper()
	ctx := context.Background()
	productID := uuid.New()
	paramModelID := uuid.New()
	productRow := &product.Product{ID: productID, Name: "active-product", ParamModelID: &paramModelID}
	products := product.NewRegistry(mmlProductRepo{
		product: productRow,
		pattern: product.ProductClassPattern{
			ID:           uuid.New(),
			ProductID:    productID,
			ProductClass: "^ACTIVE-MODEL$",
			SortOrder:    1,
			IsActive:     true,
		},
	}, product.NopCache{}, nil, zap.NewNop())
	require.NoError(t, products.Refresh(ctx))

	params := parammodel.NewRegistry(
		activeMMLParamRepo{mappings: mappings},
		parammodel.NopCache{},
		products,
		nil,
		zap.NewNop(),
	)
	return NewMMLPathTranslator(products, params, nil, zap.NewNop())
}

func multiPrefixMappings() []parammodel.ParamMapping {
	return []parammodel.ParamMapping{
		{
			StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.ParamA",
			PrivatePath:  "Device.VendorA.FAPService.{i}.CellConfig.LTE.ParamA",
		},
		{
			StandardPath: "Device.Services.FAPService.{i}.CellConfig.NR.ParamB",
			PrivatePath:  "InternetGatewayDevice.Services.FAPService.{i}.CellConfig.NR.ParamB",
		},
	}
}

func TestMMLPathTranslator_AcceptsPartialPrefixWithMultiplePrivateCandidates(t *testing.T) {
	translator := newActiveMMLPathTranslator(t, multiPrefixMappings())

	outcome, err := translator.TranslateForDevice(
		context.Background(),
		"ACTIVE-MODEL",
		"1.0",
		[]string{
			"Device.Services.FAPService.",
			"Device.Services.FAPService.1.CellConfig.LTE.ParamA",
		},
	)

	require.NoError(t, err)
	require.Len(t, outcome.Paths, 2, "TranslationOutcome keeps one record per standard input")
	require.Equal(t, "Device.VendorA.FAPService.", outcome.Paths[0].Private)
	require.Equal(t, "Device.VendorA.FAPService.1.CellConfig.LTE.ParamA", outcome.Paths[1].Private)
}

func TestMMLPathTranslator_RejectsUnknownPartialPrefix(t *testing.T) {
	translator := newActiveMMLPathTranslator(t, multiPrefixMappings())

	outcome, err := translator.TranslateForDevice(
		context.Background(),
		"ACTIVE-MODEL",
		"1.0",
		[]string{"Device.Services.Unknown."},
	)

	require.Nil(t, outcome)
	var unsupported *mml.ErrPathUnsupported
	require.ErrorAs(t, err, &unsupported)
	require.Equal(t, []string{"Device.Services.Unknown."}, unsupported.Paths)
}

func TestMMLPathTranslator_InactiveParamModelDoesNotPassthrough(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	paramModelID := uuid.New()
	productRow := &product.Product{ID: productID, Name: "inactive-product", ParamModelID: &paramModelID}
	products := product.NewRegistry(mmlProductRepo{
		product: productRow,
		pattern: product.ProductClassPattern{
			ID:           uuid.New(),
			ProductID:    productID,
			ProductClass: "^INACTIVE-MODEL$",
			SortOrder:    1,
			IsActive:     true,
		},
	}, product.NopCache{}, nil, zap.NewNop())
	require.NoError(t, products.Refresh(ctx))

	params := parammodel.NewRegistry(inactiveMMLParamRepo{}, parammodel.NopCache{}, products, nil, zap.NewNop())
	translator := NewMMLPathTranslator(products, params, nil, zap.NewNop())
	outcome, err := translator.TranslateForDevice(ctx, "INACTIVE-MODEL", "1.0", []string{"Device.Test.Param"})
	require.ErrorIs(t, err, parammodel.ErrInactiveParamModel)
	require.Nil(t, outcome, "inactive model must not silently execute MML with standard-path passthrough")
}

func TestMMLPathTranslator_InactiveProductRouteDoesNotUseOrphanPassthrough(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()
	paramModelID := uuid.New()
	products := product.NewRegistry(mmlProductRepo{
		product: &product.Product{ID: productID, Name: "inactive-route", ParamModelID: &paramModelID},
		pattern: product.ProductClassPattern{
			ID:                 uuid.New(),
			ProductID:          productID,
			ProductClass:       "^INACTIVE-ROUTE$",
			SortOrder:          1,
			IsActive:           true,
			ParamModelInactive: true,
		},
	}, product.NopCache{}, nil, zap.NewNop())
	require.NoError(t, products.Refresh(ctx))

	params := parammodel.NewRegistry(inactiveMMLParamRepo{}, parammodel.NopCache{}, products, nil, zap.NewNop())
	translator := NewMMLPathTranslator(products, params, nil, zap.NewNop())
	outcome, err := translator.TranslateForDevice(ctx, "INACTIVE-ROUTE", "1.0", []string{"Device.Test.Param"})
	require.ErrorIs(t, err, product.ErrInactiveParamModel)
	require.Nil(t, outcome, "inactive product route must not be downgraded to orphan_passthrough")
}
