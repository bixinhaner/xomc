package main

import (
	"context"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/components"
)

// initACS initializes infrastructure for the ACS engine (Redis + NATS + optional PostgreSQL/MinIO).
func initACS(ctx context.Context, cfg *appconfig.ACSConfig) (*components.Infra, error) {
	inf, err := components.NewInfra(cfg.Log, cfg.Metrics.Port)
	if err != nil {
		return nil, err
	}

	if err := inf.InitTracer(ctx, cfg.Tracer, "omcgo-acs"); err != nil {
		return nil, err
	}

	if err := inf.ConnectRedis(cfg.Redis); err != nil {
		return nil, err
	}
	if err := inf.ConnectNATS(ctx, cfg.NATS); err != nil {
		return nil, err
	}
	// PostgreSQL connection for task persistence
	if cfg.DB.DSN != "" {
		if err := inf.ConnectPostgres(ctx, cfg.DB); err != nil {
			return nil, err
		}
	}
	// TimescaleDB connection for TR069 trace_messages（KPI/时序库物理分离后 trace 写时序库）。
	// 与 DB 同样按 DSN 是否配置决定是否连接；连接后 inf.TsPool 非 nil，trace 旁路才注入 TsPool。
	if cfg.TSDB.DSN != "" {
		if err := inf.ConnectTimescale(ctx, cfg.TSDB); err != nil {
			return nil, err
		}
	}
	// MinIO connection for file upload proxy
	if cfg.MinIO.Endpoint != "" {
		if err := inf.ConnectMinIO(ctx, cfg.MinIO); err != nil {
			return nil, err
		}
	}
	inf.CreateEventBus()

	return inf, nil
}
