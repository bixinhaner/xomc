package trace

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

// captureMockRepo 记录批量插入的调用，用于 flusher 行为测试。
type captureMockRepo struct {
	mockRepo
	mu       sync.Mutex
	inserted []*Message
	batches  int32
	counts   map[uuid.UUID]int
}

func (c *captureMockRepo) InsertMessages(_ context.Context, msgs []*Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inserted = append(c.inserted, msgs...)
	atomic.AddInt32(&c.batches, 1)
	return nil
}

func (c *captureMockRepo) IncrementMessageCount(_ context.Context, id uuid.UUID, d int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.counts == nil {
		c.counts = map[uuid.UUID]int{}
	}
	c.counts[id] += d
	return nil
}

// 仍要满足整个 Repository 接口
func (c *captureMockRepo) CreateTask(_ context.Context, _ *Task) error             { return errUnused }
func (c *captureMockRepo) GetTask(_ context.Context, _ uuid.UUID) (*Task, error)   { return nil, errUnused }
func (c *captureMockRepo) GetRunningTaskBySN(_ context.Context, _ string) (*Task, error) {
	return nil, errUnused
}
func (c *captureMockRepo) ListTasks(_ context.Context, _ TaskFilter) (*model.ListResponse[Task], error) {
	return nil, errUnused
}
func (c *captureMockRepo) UpdateTaskStatus(_ context.Context, _ uuid.UUID, _ TaskStatus) error {
	return errUnused
}
func (c *captureMockRepo) ListExpired(_ context.Context, _ int) ([]Task, error) { return nil, errUnused }
func (c *captureMockRepo) PurgeTaskMessages(_ context.Context, _ uuid.UUID) error {
	return errUnused
}
func (c *captureMockRepo) InsertMessage(_ context.Context, _ *Message) error { return errUnused }
func (c *captureMockRepo) ListMessages(_ context.Context, _ MessageFilter) (*model.ListResponse[Message], error) {
	return nil, errUnused
}
func (c *captureMockRepo) GetMessage(_ context.Context, _, _ uuid.UUID) (*Message, error) {
	return nil, errUnused
}
func (c *captureMockRepo) CreateExportJob(_ context.Context, _ *ExportJob) error { return errUnused }
func (c *captureMockRepo) GetExportJob(_ context.Context, _ uuid.UUID) (*ExportJob, error) {
	return nil, errUnused
}
func (c *captureMockRepo) UpdateExportJob(_ context.Context, _ *ExportJob) error { return errUnused }
func (c *captureMockRepo) ListRunningSNs(_ context.Context) (map[string]uuid.UUID, error) {
	return map[string]uuid.UUID{}, nil
}

func TestService_EnqueueAndFlush(t *testing.T) {
	repo := &captureMockRepo{}
	svc := NewService(repo, Config{QueueSize: 100, FlushSize: 3, FlushDelay: 50 * time.Millisecond}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Start(ctx)

	taskID := uuid.New()
	// 投递 3 条 → 触发 FlushSize 立即落库
	for i := 0; i < 3; i++ {
		svc.EnqueueCapture(&Message{
			TaskID:   taskID,
			DeviceSN: "SN-001",
			Direction: DirectionIn,
		})
	}

	// 等 flusher tick + 写库
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		repo.mu.Lock()
		n := len(repo.inserted)
		repo.mu.Unlock()
		if n >= 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	repo.mu.Lock()
	assert.Equal(t, 3, len(repo.inserted))
	assert.Equal(t, 3, repo.counts[taskID])
	repo.mu.Unlock()

	svc.Stop()
}

func TestService_FlushOnShutdown(t *testing.T) {
	repo := &captureMockRepo{}
	svc := NewService(repo, Config{QueueSize: 100, FlushSize: 1000, FlushDelay: time.Hour}, nil)
	ctx := context.Background()
	svc.Start(ctx)

	// 投递 2 条但远未到 FlushSize 也未到 FlushDelay
	svc.EnqueueCapture(&Message{TaskID: uuid.New(), DeviceSN: "X", Direction: DirectionIn})
	svc.EnqueueCapture(&Message{TaskID: uuid.New(), DeviceSN: "Y", Direction: DirectionOut})

	// Stop 必须 flush 残留
	svc.Stop()

	repo.mu.Lock()
	assert.Equal(t, 2, len(repo.inserted))
	repo.mu.Unlock()
}

func TestService_DropOnQueueFull(t *testing.T) {
	repo := &captureMockRepo{}
	// 不调 Start：没有 flusher 消费，queue 立刻满
	svc := NewService(repo, Config{QueueSize: 2, FlushSize: 1, FlushDelay: time.Hour}, nil)
	taskID := uuid.New()
	for i := 0; i < 10; i++ {
		svc.EnqueueCapture(&Message{TaskID: taskID, DeviceSN: "S"})
	}
	assert.GreaterOrEqual(t, svc.DroppedCount(), uint64(8))
}

func TestService_EnqueueNilSafe(t *testing.T) {
	svc := NewService(&captureMockRepo{}, DefaultConfig(), nil)
	require.NotPanics(t, func() { svc.EnqueueCapture(nil) })
}
