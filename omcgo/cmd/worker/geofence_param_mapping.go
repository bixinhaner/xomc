package main

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/product"
	"go.uber.org/zap"
)

type workerGeofenceMappingReader struct {
	products *product.Registry
	params   *parammodel.Registry
}

func newWorkerGeofenceMappingReader(
	products *product.Registry,
	params *parammodel.Registry,
) *workerGeofenceMappingReader {
	return &workerGeofenceMappingReader{products: products, params: params}
}

func (r *workerGeofenceMappingReader) GetByProductClass(
	ctx context.Context,
	productClass string,
	softwareVersion string,
) ([]carrier.GeofenceControlMapping, error) {
	if r == nil || r.products == nil || r.params == nil {
		return nil, fmt.Errorf("worker geofence ParamModel dependencies are required")
	}
	matched, err := r.products.MatchProductClass(ctx, productClass)
	if err != nil {
		return nil, fmt.Errorf("match geofence product class %q: %w", productClass, err)
	}
	if matched == nil || matched.Product == nil {
		return nil, fmt.Errorf("product class %q has no product mapping", productClass)
	}
	translator, err := r.params.Translator(
		ctx, matched.Product.ID, softwareVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve geofence ParamModel translator: %w", err)
	}
	mappings := translator.Mappings()
	result := make([]carrier.GeofenceControlMapping, 0, len(mappings))
	for _, mapping := range mappings {
		enumValues := ""
		if mapping.EnumValues != nil {
			enumValues = *mapping.EnumValues
		}
		result = append(result, carrier.GeofenceControlMapping{
			StandardPath:      mapping.StandardPath,
			PrivatePath:       mapping.PrivatePath,
			EntryType:         mapping.EntryType,
			Access:            mapping.Access,
			EnumValues:        enumValues,
			ProductRadioModes: matched.Product.RadioModes,
			IsActive:          mapping.IsActive,
			IsSupported:       mapping.IsSupported,
		})
	}
	return result, nil
}

func newWorkerParamRegistry(
	w *workerInfra,
	products *product.Registry,
	logger *zap.Logger,
) *parammodel.Registry {
	// worker 内所有 ParamModel 消费者共享同一实例。除避免重复缓存外，更重要的是
	// NewRegistryMetrics 使用 MustRegister，同一 Prometheus Registry 上构造第二套会 panic。
	if w.ParamRegistry != nil {
		return w.ParamRegistry
	}
	var cache parammodel.Cache = parammodel.NopCache{}
	if w.Redis != nil {
		cache = parammodel.NewRedisCache(w.Redis)
	}
	registry := parammodel.NewRegistry(
		parammodel.NewPgRepository(w.PgPool),
		cache,
		products,
		parammodel.NewRegistryMetrics(w.MetricsReg),
		logger,
	)
	if err := registry.Refresh(context.Background()); err != nil {
		logger.Warn("geofence ParamRegistry refresh failed; control mapping will fail closed",
			zap.Error(err))
	}
	w.ParamRegistry = registry
	return registry
}
