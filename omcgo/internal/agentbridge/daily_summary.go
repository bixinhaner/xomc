package agentbridge

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

type DailySummaryScheduler struct {
	outbox   *OutboxRepository
	targets  RuntimeTargetProvider
	handbook HandbookMetadataProvider
	logger   *zap.Logger
	cancel   context.CancelFunc
	done     chan struct{}
	once     sync.Once
}

func NewDailySummaryScheduler(outbox *OutboxRepository, targets RuntimeTargetProvider, handbook HandbookMetadataProvider, logger *zap.Logger) *DailySummaryScheduler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DailySummaryScheduler{outbox: outbox, targets: targets, handbook: handbook, logger: logger.Named("agent-daily-summary"), done: make(chan struct{})}
}

func (s *DailySummaryScheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.run(ctx)
}
func (s *DailySummaryScheduler) Close() error {
	s.once.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		<-s.done
	})
	return nil
}

func (s *DailySummaryScheduler) run(ctx context.Context) {
	defer close(s.done)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := s.enqueueToday(ctx, time.Now()); err != nil && ctx.Err() == nil {
			s.logger.Warn("enqueue daily operations summary", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *DailySummaryScheduler) enqueueToday(ctx context.Context, now time.Time) error {
	local := now.In(time.Local)
	if local.Hour() < 8 {
		return nil
	}
	target, err := s.targets.GetRuntimeTarget(ctx)
	if err != nil {
		return fmt.Errorf("load agent bridge target: %w", err)
	}
	if !target.Enabled {
		return nil
	}
	metadata, err := s.handbook.HandbookMetadata()
	if err != nil {
		return fmt.Errorf("resolve agent handbook metadata: %w", err)
	}
	digest, _ := metadata["handbookDigest"].(string)
	if strings.TrimSpace(digest) == "" {
		return fmt.Errorf("handbook digest is missing")
	}
	date := local.Format("2006-01-02")
	start := time.Date(local.Year(), local.Month(), local.Day()-1, 8, 0, 0, 0, local.Location())
	end := time.Date(local.Year(), local.Month(), local.Day(), 8, 0, 0, 0, local.Location())
	envelope := ConnectorEventEnvelope{ContractVersion: ContractVersion, EventID: "daily-summary:" + date + ":" + target.ConnectorID, EventType: "omc.daily-operations-summary-requested.v1", Source: target.InstanceName, OccurredAt: end.UTC(), TraceID: "daily-summary:" + date, IntegrationPack: PackRef{Key: XOMCPackageKey, Version: XOMCPackageVersion, Digest: XOMCPackageDigest}, HandbookDigest: digest, Resources: []ResourceRef{{Type: "ne_group", ID: "all", Role: "scope", Label: "all managed network elements"}}, Data: map[string]any{"summaryDate": date, "windowStart": start.UTC(), "windowEnd": end.UTC(), "timezone": local.Location().String(), "apiHandbook": metadata}}
	return s.outbox.Enqueue(ctx, envelope)
}
