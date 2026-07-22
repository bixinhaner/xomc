package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/components"
	"github.com/omcgo/omcgo/internal/core/realtime"
	"github.com/omcgo/omcgo/internal/pm/adhoc"
)

type pmProgressWiringSubscription struct {
	unsubscribed bool
	err          error
}

func (s *pmProgressWiringSubscription) Unsubscribe() error {
	s.unsubscribed = true
	return s.err
}

type pmProgressWiringRealtime struct {
	subjects []string
	subs     []*pmProgressWiringSubscription
	subErrs  []error
}

func (s *pmProgressWiringRealtime) Subscribe(subject string, _ func([]byte)) (realtime.Subscription, error) {
	s.subjects = append(s.subjects, subject)
	var err error
	if len(s.subErrs) > len(s.subs) {
		err = s.subErrs[len(s.subs)]
	}
	sub := &pmProgressWiringSubscription{err: err}
	s.subs = append(s.subs, sub)
	return sub, nil
}

func TestStartPMAdhocProgress_NilShutdownIncludesRollbackFailure(t *testing.T) {
	progressErr := errors.New("progress unsubscribe failed")
	completedErr := errors.New("completed unsubscribe failed")
	realtimeBus := &pmProgressWiringRealtime{subErrs: []error{progressErr, completedErr}}

	hub, err := startPMAdhocProgress(realtimeBus, nil, zap.NewNop())
	require.Error(t, err)
	assert.Nil(t, hub)
	assert.ErrorIs(t, err, progressErr)
	assert.ErrorIs(t, err, completedErr)
	assert.Contains(t, err.Error(), "PM adhoc realtime progress shutdown not wired")
	assert.Contains(t, err.Error(), "rollback PM adhoc realtime progress bridge")
	assert.Contains(t, err.Error(), adhoc.SubjectRealtimeProgress)
	assert.Contains(t, err.Error(), adhoc.SubjectRealtimeCompleted)
	require.Len(t, realtimeBus.subs, 2)
	assert.True(t, realtimeBus.subs[0].unsubscribed)
	assert.True(t, realtimeBus.subs[1].unsubscribed)
}

// TestAppAdhocRepo_HasWatermarkReaderWired 钉死 #528 P3 的装配缺口（检查方回合1阻塞项）。
//
// 建持续任务的唯一入口 = app 进程的 POST /pm/adhoc，其 repo 必须注入「上游完成水位」读取器，
// 否则新建持续任务初始游标退化为 NULL（落回 created_at），sweep 会逐 tick 补出 created_at..水位
// 之间的史前空格批量行——正是 scope 要消除的现象。检查方曾发现水位读取器被误注入到 worker
// 进程（从不建任务），app 端 repo 漏注入。
//
// 本测试直接调用生产构造函数 buildPMAdhocRepo（initPMModule 用的同一函数），pool 传 nil 不发起
// 任何 DB 调用，仅验证注入链路。若有人把 SetWatermarkReader 从 buildPMAdhocRepo 删掉/挪走，
// 此测试立即红——真正钉死生产装配路径，而非镜像复制。
func TestAppAdhocRepo_HasWatermarkReaderWired(t *testing.T) {
	repo := buildPMAdhocRepo(nil, nil, zap.NewNop())

	assert.True(t, repo.HasWatermarkReader(),
		"app 端建持续任务的 adhoc repo 必须注入水位读取器，否则初始游标退化、结果表冒史前空格（#528 P3 回归）")
}

func TestStartPMAdhocProgress_SubscribesOnceAndStopsBeforeHTTP(t *testing.T) {
	logger := zap.NewNop()
	shutdown := components.NewGracefulShutdown(time.Second, logger)
	realtimeBus := &pmProgressWiringRealtime{}

	hub, err := startPMAdhocProgress(realtimeBus, shutdown, logger)
	require.NoError(t, err)
	assert.Equal(t, []string{adhoc.SubjectRealtimeProgress, adhoc.SubjectRealtimeCompleted}, realtimeBus.subjects)

	checkedBeforeHTTP := false
	shutdown.Register("test-http", 1, func(context.Context) error {
		checkedBeforeHTTP = realtimeBus.subs[0].unsubscribed && realtimeBus.subs[1].unsubscribed
		return nil
	})
	require.NoError(t, shutdown.Shutdown(context.Background()))
	assert.True(t, checkedBeforeHTTP)

	events, unsubscribe := hub.Subscribe("task-after-shutdown")
	defer unsubscribe()
	_, open := <-events
	assert.False(t, open)
}
