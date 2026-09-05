package agentbridge

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

type HeartbeatWorker struct {
	client   *Client
	digest   func() (string, error)
	workerID string
	logger   *zap.Logger
	cancel   context.CancelFunc
	done     chan struct{}
	once     sync.Once
}

func NewHeartbeatWorker(client *Client, digest func() (string, error), workerID string, logger *zap.Logger) *HeartbeatWorker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &HeartbeatWorker{client: client, digest: digest, workerID: workerID, logger: logger.Named("agent-heartbeat"), done: make(chan struct{})}
}

func (w *HeartbeatWorker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go w.run(ctx)
}

func (w *HeartbeatWorker) Close() error {
	w.once.Do(func() {
		if w.cancel != nil {
			w.cancel()
		}
		<-w.done
	})
	return nil
}

func (w *HeartbeatWorker) run(ctx context.Context) {
	defer close(w.done)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if digest, err := w.digest(); err == nil {
			err = w.client.SendHeartbeat(ctx, ConnectorHeartbeat{WorkerID: w.workerID, HandbookDigest: digest})
			if err != nil && !errors.Is(err, ErrBridgeNotConnected) && ctx.Err() == nil {
				w.logger.Warn("send connector heartbeat", zap.Error(err))
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
