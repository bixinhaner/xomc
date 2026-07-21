package admin

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type leaseApplyRepo struct {
	mu sync.Mutex

	renewOK        bool
	renewErr       error
	renewCalls     int
	lastRenewUntil time.Time
	completed      bool
	completeErr    error
	batchResult    BatchUpsertResult
	batch          *ConfigApplyBatch
	claimErr       error
}

func (r *leaseApplyRepo) Create(context.Context, *SysConfig) error { return nil }
func (r *leaseApplyRepo) GetByID(context.Context, uuid.UUID) (*SysConfig, error) {
	return nil, nil
}
func (r *leaseApplyRepo) GetByKey(context.Context, string, string) (*SysConfig, error) {
	return nil, nil
}
func (r *leaseApplyRepo) List(context.Context, string, bool) ([]SysConfig, error) { return nil, nil }
func (r *leaseApplyRepo) Update(context.Context, *SysConfig) error                { return nil }
func (r *leaseApplyRepo) Delete(context.Context, uuid.UUID) error                 { return nil }
func (r *leaseApplyRepo) BatchUpsert(context.Context, string, []BatchItem) (int, error) {
	return 0, nil
}
func (r *leaseApplyRepo) BatchUpsertWithApply(context.Context, string, []BatchItem, []ConfigApplyTarget) (BatchUpsertResult, error) {
	return r.batchResult, nil
}
func (r *leaseApplyRepo) GetApplyBatch(context.Context, uuid.UUID) (*ConfigApplyBatch, error) {
	return r.batch, nil
}
func (r *leaseApplyRepo) ClaimApplyTarget(context.Context, uuid.UUID, string) (*ConfigApplyWork, error) {
	return nil, r.claimErr
}
func (r *leaseApplyRepo) ClaimNextApplyTarget(context.Context) (*ConfigApplyWork, error) {
	return nil, nil
}
func (r *leaseApplyRepo) RenewApplyTargetLease(_ context.Context, _ ConfigApplyWork, until time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.renewCalls++
	r.lastRenewUntil = until
	return r.renewOK, r.renewErr
}
func (r *leaseApplyRepo) CompleteApplyTarget(_ context.Context, _ ConfigApplyWork, _ map[string]any, applyErr error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completed = true
	r.completeErr = applyErr
	return nil
}

func TestExecuteApplyWorkRenewsLeaseUntilHandlerCompletes(t *testing.T) {
	repo := &leaseApplyRepo{renewOK: true}
	svc := NewSysConfigService(repo)
	svc.applyLeaseDuration = 100 * time.Millisecond
	svc.applyLeaseRenewInterval = 5 * time.Millisecond

	handlerRelease := make(chan struct{})
	svc.RegisterApplyHandler("storage", "minio", func(ctx context.Context, work ConfigApplyWork) (map[string]any, error) {
		require.NotEqual(t, uuid.Nil, work.LeaseToken)
		select {
		case <-handlerRelease:
			return map[string]any{"ok": true}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	work := ConfigApplyWork{
		Batch:      ConfigApplyBatch{ID: uuid.New(), Category: "storage"},
		Target:     ConfigApplyTarget{Target: "minio", Status: ConfigApplyStatusApplying},
		LeaseToken: uuid.New(),
	}
	done := make(chan error, 1)
	go func() { done <- svc.executeApplyWork(context.Background(), repo, work) }()

	require.Eventually(t, func() bool {
		repo.mu.Lock()
		defer repo.mu.Unlock()
		return repo.renewCalls > 0 && repo.lastRenewUntil.After(time.Now())
	}, time.Second, 5*time.Millisecond)
	close(handlerRelease)
	require.NoError(t, <-done)

	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.True(t, repo.completed)
	require.NoError(t, repo.completeErr)
}

func TestExecuteApplyWorkCancelsHandlerWhenLeaseIsLost(t *testing.T) {
	repo := &leaseApplyRepo{renewOK: false}
	svc := NewSysConfigService(repo)
	svc.applyLeaseDuration = 100 * time.Millisecond
	svc.applyLeaseRenewInterval = 5 * time.Millisecond

	handlerCanceled := make(chan struct{})
	svc.RegisterApplyHandler("storage", "minio", func(ctx context.Context, _ ConfigApplyWork) (map[string]any, error) {
		<-ctx.Done()
		close(handlerCanceled)
		return nil, ctx.Err()
	})

	work := ConfigApplyWork{
		Batch:      ConfigApplyBatch{ID: uuid.New(), Category: "storage"},
		Target:     ConfigApplyTarget{Target: "minio", Status: ConfigApplyStatusApplying},
		LeaseToken: uuid.New(),
	}
	err := svc.executeApplyWork(context.Background(), repo, work)
	require.ErrorContains(t, err, "config apply lease lost")
	require.Eventually(t, func() bool {
		select {
		case <-handlerCanceled:
			return true
		default:
			return false
		}
	}, time.Second, 5*time.Millisecond)

	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.False(t, repo.completed, "a worker that lost its lease must not complete the target")
}

func TestExecuteApplyWorkReturnsRenewalErrorWithoutCompleting(t *testing.T) {
	repo := &leaseApplyRepo{renewOK: true, renewErr: errors.New("database unavailable")}
	svc := NewSysConfigService(repo)
	svc.applyLeaseDuration = 100 * time.Millisecond
	svc.applyLeaseRenewInterval = 5 * time.Millisecond
	svc.RegisterApplyHandler("storage", "minio", func(ctx context.Context, _ ConfigApplyWork) (map[string]any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	err := svc.executeApplyWork(context.Background(), repo, ConfigApplyWork{
		Batch:      ConfigApplyBatch{ID: uuid.New(), Category: "storage"},
		Target:     ConfigApplyTarget{Target: "minio", Status: ConfigApplyStatusApplying},
		LeaseToken: uuid.New(),
	})

	require.ErrorContains(t, err, "renew config apply lease")
	require.ErrorContains(t, err, "database unavailable")
	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.False(t, repo.completed)
}

func TestExecuteApplyWorkConvertsHandlerPanicToFailedApplication(t *testing.T) {
	repo := &leaseApplyRepo{renewOK: true}
	svc := NewSysConfigService(repo)
	svc.RegisterApplyHandler("storage", "minio", func(context.Context, ConfigApplyWork) (map[string]any, error) {
		panic("injected handler panic")
	})

	err := svc.executeApplyWork(context.Background(), repo, ConfigApplyWork{
		Batch:  ConfigApplyBatch{ID: uuid.New(), Category: "storage"},
		Target: ConfigApplyTarget{Target: "minio", Status: ConfigApplyStatusApplying},
	})

	require.NoError(t, err)
	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.True(t, repo.completed)
	require.ErrorContains(t, repo.completeErr, "config apply handler panic")
}

func TestBatchUpsertReturnsDurableBatchWhenImmediateApplyFails(t *testing.T) {
	batch := ConfigApplyBatch{
		ID: uuid.New(), Category: "storage", Status: ConfigApplyStatusPending,
		Targets: []ConfigApplyTarget{{Target: "minio", Status: ConfigApplyStatusPending}},
	}
	repo := &leaseApplyRepo{
		batchResult: BatchUpsertResult{Updated: 1, Batch: batch},
		batch:       &batch,
		claimErr:    errors.New("database temporarily unavailable"),
	}
	svc := NewSysConfigService(repo)
	svc.RegisterApplyHandler("storage", "minio", func(context.Context, ConfigApplyWork) (map[string]any, error) {
		return nil, nil
	})

	result, err := svc.BatchUpsertWithResult(context.Background(), BatchUpdateSysConfigRequest{
		Category: "storage", Items: []BatchItem{{Key: "k", Value: "v"}},
	})

	require.NoError(t, err)
	require.Equal(t, batch.ID, result.Batch.ID)
	require.Equal(t, ConfigApplyStatusPending, result.Batch.Status)
}
