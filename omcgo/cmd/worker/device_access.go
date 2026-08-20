package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/deviceaccess"
	"go.uber.org/zap"
)

func startDeviceAccessWorkers(w *workerInfra) error {
	if w == nil || w.PgPool == nil || w.EventBus == nil || w.GS == nil {
		return fmt.Errorf("start device access workers: dependencies are required")
	}
	logger := w.Logger.Named("device-access")
	repository := deviceaccess.NewPgRepository(w.PgPool)
	policies := deviceaccess.NewPgPolicyProvider(w.PgPool)
	coordinator := deviceaccess.NewReevaluationCoordinator(
		repository,
		deviceaccess.NewAssetEvidenceResolver(repository),
		policies,
	)
	coordinator.SetRuntimeSettingsReader(deviceaccess.NewPgRuntimeSettingsStore(w.PgPool))
	if w.ProductRegistry == nil || w.ParamRegistry == nil {
		return fmt.Errorf("start device access workers: product and parameter registries are required")
	}
	probes := deviceaccess.NewGPSProbeService(
		deviceaccess.NewProductGPSPathResolver(w.ProductRegistry, w.ParamRegistry),
		w.TaskService,
		repository,
		coordinator,
		logger,
	)
	coordinator.SetGPSProbePlanner(deviceaccess.NewPgGPSProbeQueue(w.PgPool))
	reevaluationQueue := deviceaccess.NewPgReevaluationQueue(w.PgPool)
	collectionDeadlines := deviceaccess.NewCollectionDeadlineScheduler(w.PgPool, reevaluationQueue)
	consumer := deviceaccess.NewReevaluationConsumer(w.EventBus, coordinator)
	if err := consumer.Start(); err != nil {
		return err
	}
	policyConsumer := deviceaccess.NewPolicyPublishedConsumer(
		w.EventBus,
		deviceaccess.NewPolicyPublishedExpander(w.PgPool),
	)
	if err := policyConsumer.Start(); err != nil {
		_ = consumer.Stop()
		return err
	}
	probeConsumer := deviceaccess.NewGPSProbeConsumer(w.EventBus, probes)
	if err := probeConsumer.Start(); err != nil {
		_ = policyConsumer.Stop()
		_ = consumer.Stop()
		return err
	}
	dispatcher := deviceaccess.NewOutboxDispatcher(w.PgPool, w.EventBus)
	workerCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		dispatchTicker := time.NewTicker(time.Second)
		recoveryTicker := time.NewTicker(time.Minute)
		collectionDeadlineTicker := time.NewTicker(30 * time.Second)
		defer dispatchTicker.Stop()
		defer recoveryTicker.Stop()
		defer collectionDeadlineTicker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-dispatchTicker.C:
				ctx, dispatchCancel := context.WithTimeout(workerCtx, 15*time.Second)
				_, err := dispatcher.DispatchPending(ctx, 100)
				dispatchCancel()
				if err != nil && !errors.Is(err, context.Canceled) {
					logger.Warn("dispatch device access outbox failed", zap.Error(err))
				}
			case <-recoveryTicker.C:
				ctx, recoveryCancel := context.WithTimeout(workerCtx, 10*time.Second)
				_, err := dispatcher.RequeueStaleDeliveries(ctx, time.Now().UTC().Add(-time.Minute))
				recoveryCancel()
				if err != nil && !errors.Is(err, context.Canceled) {
					logger.Warn("recover stale device access outbox failed", zap.Error(err))
				}
			case <-collectionDeadlineTicker.C:
				ctx, deadlineCancel := context.WithTimeout(workerCtx, 10*time.Second)
				_, err := collectionDeadlines.RunOnce(ctx, 100)
				deadlineCancel()
				if err != nil && !errors.Is(err, context.Canceled) {
					logger.Warn("schedule expired device access collections failed", zap.Error(err))
				}
			}
		}
	}()
	w.GS.Register("device-access-outbox", 2, func(context.Context) error {
		cancel()
		<-done
		probeErr := probeConsumer.Stop()
		policyErr := policyConsumer.Stop()
		consumerErr := consumer.Stop()
		if probeErr != nil {
			return probeErr
		}
		if policyErr != nil {
			return policyErr
		}
		return consumerErr
	})
	logger.Info("device access outbox relay and reevaluation consumer initialized")
	return nil
}
