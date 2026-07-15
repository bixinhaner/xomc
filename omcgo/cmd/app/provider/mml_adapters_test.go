package provider

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
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
