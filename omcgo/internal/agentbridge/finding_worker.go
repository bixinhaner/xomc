package agentbridge

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

var ErrFindingVisibilityUnresolved = errors.New("finding visibility cannot be resolved from authoritative resources")

type FindingWorker struct {
	client   *Client
	repo     *FindingRepository
	workerID string
	logger   *zap.Logger
	cancel   context.CancelFunc
	done     chan struct{}
	once     sync.Once
}

func NewFindingWorker(client *Client, repo *FindingRepository, workerID string, logger *zap.Logger) *FindingWorker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FindingWorker{client: client, repo: repo, workerID: workerID, logger: logger.Named("agent-finding-worker"), done: make(chan struct{})}
}

func (w *FindingWorker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go w.run(ctx)
}

func (w *FindingWorker) Close() error {
	w.once.Do(func() {
		if w.cancel != nil {
			w.cancel()
		}
		<-w.done
	})
	return nil
}

func (w *FindingWorker) run(ctx context.Context) {
	defer close(w.done)
	for ctx.Err() == nil {
		items, err := w.client.LeaseFindings(ctx, w.workerID, 8)
		if err != nil {
			if ctx.Err() == nil {
				if errors.Is(err, ErrBridgeNotConnected) {
					w.logger.Debug("agent finding worker waiting for connector configuration")
				} else {
					w.logger.Warn("lease agent findings", zap.Error(err))
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
			}
			continue
		}
		if len(items) == 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}
		for _, delivery := range items {
			w.accept(ctx, delivery)
		}
	}
}

func (w *FindingWorker) accept(ctx context.Context, delivery FindingDelivery) {
	ack := FindingDeliveryAck{
		ContractVersion: ContractVersion, LeaseToken: delivery.LeaseToken, WorkerID: w.workerID,
		Status: "rejected", CommittedAt: time.Now().UTC(), TraceID: delivery.TraceID,
	}
	if err := validateFindingDelivery(delivery); err != nil {
		ack.Error = &ToolError{Code: "OUTPUT_SCHEMA_INVALID", Message: err.Error(), Retryable: false}
	} else {
		localID, _, visible, err := w.repo.Accept(ctx, delivery)
		if err == nil && visible {
			ack.Status = "delivered"
			ack.LocalFindingID = localID
		} else if err == nil {
			ack.Error = &ToolError{Code: "VISIBILITY_UNRESOLVED", Message: ErrFindingVisibilityUnresolved.Error(), Retryable: false}
		} else {
			ack.Error = &ToolError{Code: "FINDING_PROJECTION_FAILED", Message: err.Error(), Retryable: !errors.Is(err, ErrFindingConflict)}
		}
	}
	ack.CommittedAt = time.Now().UTC()
	if err := w.client.AckFinding(ctx, delivery.DeliveryID, ack); err != nil {
		w.logger.Error("ack agent finding", zap.String("delivery_id", delivery.DeliveryID), zap.Error(err))
	}
}

func validateFindingDelivery(delivery FindingDelivery) error {
	if delivery.ContractVersion != ContractVersion || delivery.Finding.SchemaVersion != ContractVersion {
		return fmt.Errorf("unsupported finding contract version")
	}
	if delivery.DeliveryID == "" || delivery.FindingID == "" || delivery.RunID == "" || delivery.ScenarioKey == "" {
		return fmt.Errorf("finding delivery identifiers are incomplete")
	}
	if delivery.Finding.Title == "" || delivery.Finding.Summary == "" {
		return fmt.Errorf("finding title and summary are required")
	}
	if delivery.Finding.Confidence < 0 || delivery.Finding.Confidence > 1 {
		return fmt.Errorf("finding confidence is outside 0..1")
	}
	severity := delivery.Finding.Severity
	if severity != "info" && severity != "low" && severity != "medium" && severity != "high" && severity != "critical" {
		return fmt.Errorf("unsupported finding severity %q", severity)
	}
	if len(delivery.Finding.ResourceRefs) == 0 {
		return fmt.Errorf("finding has no authoritative resource references")
	}
	for _, fact := range delivery.Finding.Facts {
		refs, ok := fact["evidenceRefs"].([]any)
		if !ok || len(refs) == 0 {
			return fmt.Errorf("every finding fact must carry evidenceRefs")
		}
	}
	return nil
}
