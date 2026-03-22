package main

import (
	"context"

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/carrier"
	"github.com/omcgo/omcgo/internal/core/carrier/cmcc"
	"github.com/omcgo/omcgo/internal/core/carrier/ctcc"
	"github.com/omcgo/omcgo/internal/core/carrier/cucc"
	"github.com/omcgo/omcgo/internal/core/components"
	"go.uber.org/zap"
)

// workerInfra extends components.Infra with worker-specific dependencies.
type workerInfra struct {
	*components.Infra
	Carriers *carrier.CarrierRegistry
	CmdQueue *cmdqueue.RedisCommandQueue
}

// initWorker initializes all infrastructure for the background worker.
func initWorker(ctx context.Context, cfg *appconfig.WorkerConfig) (*workerInfra, error) {
	inf, err := components.NewInfra(cfg.Log, cfg.Metrics.Port)
	if err != nil {
		return nil, err
	}

	if err := inf.InitTracer(ctx, cfg.Tracer, "omcgo-worker"); err != nil {
		return nil, err
	}

	if err := inf.ConnectPostgres(ctx, cfg.DB); err != nil {
		return nil, err
	}
	if err := inf.ConnectTimescale(ctx, cfg.TSDB); err != nil {
		return nil, err
	}
	if err := inf.ConnectRedis(cfg.Redis); err != nil {
		return nil, err
	}
	if err := inf.ConnectNATS(ctx, cfg.NATS); err != nil {
		return nil, err
	}
	if err := inf.ConnectMinIO(ctx, cfg.MinIO); err != nil {
		return nil, err
	}
	inf.CreateEventBus()

	w := &workerInfra{Infra: inf}
	w.registerCarriers()
	w.CmdQueue = cmdqueue.NewRedisCommandQueue(inf.Redis)

	return w, nil
}

func (w *workerInfra) registerCarriers() {
	reg := carrier.NewRegistry()
	reg.Register(cmcc.New())
	reg.Register(ctcc.New())
	reg.Register(cucc.New())
	w.Carriers = reg
	w.Logger.Info("carrier registry initialized", zap.Int("carriers", len(reg.All())))
}
