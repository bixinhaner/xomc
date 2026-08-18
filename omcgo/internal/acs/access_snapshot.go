package acs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const defaultAccessSnapshotTTL = 2 * time.Minute

var errAccessSnapshotInvalid = errors.New("invalid access snapshot")

// AccessSnapshot is the small authorization summary that ACS may use on the
// session hot path. It deliberately contains no policy conditions.
type AccessSnapshot struct {
	SerialNumber      string    `json:"serial_number"`
	Carrier           string    `json:"carrier"`
	OUI               string    `json:"oui,omitempty"`
	ProductClass      string    `json:"product_class,omitempty"`
	State             string    `json:"state"`
	NormalTasksFrozen bool      `json:"normal_tasks_frozen"`
	DecisionVersion   int64     `json:"decision_version"`
	EvidenceVersion   int64     `json:"evidence_version"`
	PolicyVersionID   string    `json:"policy_version,omitempty"`
	PublishedAt       time.Time `json:"published_at"`
	ExpiresAt         time.Time `json:"expires_at"`
}

// AccessSnapshotStore stores the last effective authorization summary.
type AccessSnapshotStore interface {
	Set(ctx context.Context, snapshot AccessSnapshot) error
	Get(ctx context.Context, serialNumber, oui, productClass string) (AccessSnapshot, bool, error)
}

// AccessSnapshotProjector materializes decision events into the ACS hot-path
// store. Queue subscription keeps multiple ACS instances from duplicating the
// projection work while the result remains shared in Redis.
type AccessSnapshotProjector struct {
	bus   event.EventBus
	store AccessSnapshotStore
	log   *zap.Logger
	subs  []event.Subscription
}

func NewAccessSnapshotProjector(bus event.EventBus, store AccessSnapshotStore, logger *zap.Logger) *AccessSnapshotProjector {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AccessSnapshotProjector{bus: bus, store: store, log: logger.Named("access-snapshot")}
}

func (p *AccessSnapshotProjector) Start() error {
	if p == nil || p.bus == nil || p.store == nil {
		return fmt.Errorf("start access snapshot projector: %w", errAccessSnapshotInvalid)
	}
	for index, subject := range []string{
		event.SubjectDeviceAccessAccepted,
		event.SubjectDeviceAccessReviewRequired,
		event.SubjectDeviceAccessCollectRequested,
		event.SubjectDeviceAccessRevalidating,
		event.SubjectDeviceAccessRejected,
		event.SubjectDeviceAccessRevoked,
	} {
		queue := fmt.Sprintf("acs-access-snapshot-%d", index)
		sub, err := p.bus.QueueSubscribe(subject, queue, p.project)
		if err != nil {
			_ = p.Stop()
			return fmt.Errorf("subscribe %s: %w", subject, err)
		}
		p.subs = append(p.subs, sub)
	}
	return nil
}

func (p *AccessSnapshotProjector) Stop() error {
	var first error
	for _, sub := range p.subs {
		if err := sub.Unsubscribe(); err != nil && first == nil {
			first = err
		}
	}
	p.subs = nil
	return first
}

func (p *AccessSnapshotProjector) project(ctx context.Context, evt event.Event) error {
	var payload AccessSnapshot
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode access snapshot event: %w", err)
	}
	if err := p.store.Set(ctx, payload); err != nil {
		p.log.Warn("project access snapshot failed", zap.Error(err), zap.String("subject", evt.Subject))
		return err
	}
	return nil
}

// RedisAccessSnapshotStore implements AccessSnapshotStore with JSON + TTL.
type RedisAccessSnapshotStore struct {
	rdb redis.UniversalClient
	ttl time.Duration
	now func() time.Time
}

func NewRedisAccessSnapshotStore(rdb redis.UniversalClient, ttl time.Duration) *RedisAccessSnapshotStore {
	if ttl <= 0 {
		ttl = defaultAccessSnapshotTTL
	}
	return &RedisAccessSnapshotStore{rdb: rdb, ttl: ttl, now: time.Now}
}

func (s *RedisAccessSnapshotStore) Set(ctx context.Context, snapshot AccessSnapshot) error {
	if s == nil || s.rdb == nil {
		return fmt.Errorf("set access snapshot: %w", errAccessSnapshotInvalid)
	}
	snapshot.SerialNumber = strings.TrimSpace(snapshot.SerialNumber)
	snapshot.Carrier = strings.TrimSpace(snapshot.Carrier)
	if snapshot.SerialNumber == "" || snapshot.Carrier == "" || snapshot.State == "" {
		return fmt.Errorf("set access snapshot: %w", errAccessSnapshotInvalid)
	}
	now := s.now().UTC()
	if snapshot.PublishedAt.IsZero() {
		snapshot.PublishedAt = now
	}
	if snapshot.ExpiresAt.IsZero() {
		snapshot.ExpiresAt = now.Add(s.ttl)
	}
	remaining := snapshot.ExpiresAt.Sub(now)
	if !snapshot.ExpiresAt.After(now) || remaining <= 0 {
		return fmt.Errorf("set access snapshot: %w", errAccessSnapshotInvalid)
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("encode access snapshot: %w", err)
	}
	key := redisx.Keys.ACSAccessSnapshot(snapshot.SerialNumber)
	for attempt := 0; attempt < 3; attempt++ {
		err = s.rdb.Watch(ctx, func(tx *redis.Tx) error {
			currentPayload, getErr := tx.Get(ctx, key).Bytes()
			if getErr == nil {
				var current AccessSnapshot
				if decodeErr := json.Unmarshal(currentPayload, &current); decodeErr != nil {
					return fmt.Errorf("decode current access snapshot: %w", decodeErr)
				}
				if current.DecisionVersion >= snapshot.DecisionVersion {
					return nil
				}
			} else if !errors.Is(getErr, redis.Nil) {
				return getErr
			}
			_, pipelineErr := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, key, payload, remaining)
				return nil
			})
			return pipelineErr
		}, key)
		if !errors.Is(err, redis.TxFailedErr) {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("write access snapshot: %w", err)
	}
	return nil
}

func (s *RedisAccessSnapshotStore) Get(ctx context.Context, serialNumber, oui, productClass string) (AccessSnapshot, bool, error) {
	if s == nil || s.rdb == nil {
		return AccessSnapshot{}, false, fmt.Errorf("get access snapshot: %w", errAccessSnapshotInvalid)
	}
	serialNumber = strings.TrimSpace(serialNumber)
	if serialNumber == "" {
		return AccessSnapshot{}, false, fmt.Errorf("get access snapshot: %w", errAccessSnapshotInvalid)
	}
	payload, err := s.rdb.Get(ctx, redisx.Keys.ACSAccessSnapshot(serialNumber)).Result()
	if errors.Is(err, redis.Nil) {
		return AccessSnapshot{}, false, nil
	}
	if err != nil {
		return AccessSnapshot{}, false, fmt.Errorf("read access snapshot: %w", err)
	}
	var snapshot AccessSnapshot
	if err := json.Unmarshal([]byte(payload), &snapshot); err != nil {
		return AccessSnapshot{}, false, fmt.Errorf("decode access snapshot: %w", err)
	}
	now := s.now().UTC()
	if snapshot.SerialNumber != serialNumber || !snapshot.ExpiresAt.After(now) {
		return AccessSnapshot{}, false, nil
	}
	if strings.TrimSpace(oui) != "" && snapshot.OUI != "" && !strings.EqualFold(strings.TrimSpace(oui), snapshot.OUI) {
		return AccessSnapshot{}, false, nil
	}
	if strings.TrimSpace(productClass) != "" && snapshot.ProductClass != "" && strings.TrimSpace(productClass) != snapshot.ProductClass {
		return AccessSnapshot{}, false, nil
	}
	return snapshot, true, nil
}
