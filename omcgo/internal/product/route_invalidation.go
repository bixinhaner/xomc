package product

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	componentlogger "github.com/omcgo/omcgo/internal/core/components/logger"
)

// RouteInvalidationTrigger 是 product 包暴露给 provider 的低基数 KPI route 失效来源。
// provider 层负责映射到 pm/kpi/router，避免 product -> router -> product import cycle。
type RouteInvalidationTrigger string

const (
	RouteInvalidationTriggerProductWrite  RouteInvalidationTrigger = "product_write"
	RouteInvalidationTriggerProductReload RouteInvalidationTrigger = "product_reload"
)

// RouteInvalidator 在 product 写入事务提交后失效 KPI Route。
type RouteInvalidator func(context.Context, RouteInvalidationTrigger) error

type ProductRouteFields struct {
	DeviceType string
	Platform   string
}

func normalizeProductRouteFields(deviceType, platform string) ProductRouteFields {
	return ProductRouteFields{
		DeviceType: strings.ToLower(strings.TrimSpace(deviceType)),
		Platform:   strings.TrimSpace(platform),
	}
}

func productRouteFieldsFromProduct(p *Product) ProductRouteFields {
	if p == nil {
		return ProductRouteFields{}
	}
	return normalizeProductRouteFields(p.IndicatorDeviceType, p.IndicatorPlatform)
}

func productRouteFieldsChanged(before, after *Product) bool {
	return productRouteFieldsFromProduct(before) != productRouteFieldsFromProduct(after)
}

func RouteFieldsMapChanged(before, after map[string]ProductRouteFields) bool {
	if len(before) != len(after) {
		return true
	}
	for name, prev := range before {
		next, ok := after[name]
		if !ok || next != prev {
			return true
		}
	}
	return false
}

// ListRouteFieldsByProductName returns the fields that affect KPI route selection.
func (r *PgRepository) ListRouteFieldsByProductName(ctx context.Context) (map[string]ProductRouteFields, error) {
	rows, err := r.pool.Query(ctx, `SELECT product_name, indicator_device_type, indicator_platform FROM products`)
	if err != nil {
		return nil, fmt.Errorf("list product route fields: %w", err)
	}
	defer rows.Close()
	out := make(map[string]ProductRouteFields, 16)
	for rows.Next() {
		var name, deviceType, platform string
		if err := rows.Scan(&name, &deviceType, &platform); err != nil {
			return nil, fmt.Errorf("scan product route fields: %w", err)
		}
		out[name] = normalizeProductRouteFields(deviceType, platform)
	}
	return out, rows.Err()
}

func logRouteInvalidationFailure(ctx context.Context, logger *zap.Logger, trigger RouteInvalidationTrigger, err error) {
	if logger == nil {
		return
	}
	fields := []zap.Field{
		zap.String("trigger", string(trigger)),
		zap.Error(err),
	}
	if requestID := componentlogger.GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	logger.Warn("product write committed but KPI route invalidation failed; manual refresh can recover",
		fields...)
}
