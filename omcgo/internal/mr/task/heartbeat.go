package task

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// HeartbeatSubscriber 订阅 SubjectMRFileUploaded（由 acs.upload.Handler 发布），
// 在设备成功上报 MR 文件后：
//  1. 反查该 cell 当前所属的活跃任务，按其 report_period 计算 TTL
//  2. 写 Redis MRFileReport_{cellCode}（TTL = HeartbeatTTLSeconds）
//  3. 调 repo.TouchHeartbeat 把 PG 端 last_heartbeat 更新为 now、missed 清零、
//     health_status 置为 normal
//
// 与 transfer.Bridge 的关系：bridge 处理 AutonomousTransferComplete 事件
// （CPE 主动通知 ACS"我上传完了"），是另一条 MR 流入路径（如部分老固件）；
// HeartbeatSubscriber 主要服务于 OMC 直传场景（fileType=MR HTTP POST）。
//
// 部署：在 worker 进程注册，与 scheduler 一起跑。
type HeartbeatSubscriber struct {
	repo      Repository
	redisCli  redis.UniversalClient
	eventBus  event.EventBus
	logger    *zap.Logger
	deduper   *event.Deduper
	metrics   *Metrics // 可 nil
}

// SetMetrics 注入 Prometheus 指标（可选）。
func (s *HeartbeatSubscriber) SetMetrics(m *Metrics) { s.metrics = m }

// NewHeartbeatSubscriber 创建心跳订阅器。
//   - redisCli: 非 nil；Redis 不可用时心跳无法写，本订阅整体跳过（fail-soft）
//   - deduper: 可选；NATS JetStream 多实例部署时配上避免重复 PG 写
func NewHeartbeatSubscriber(
	repo Repository,
	redisCli redis.UniversalClient,
	eventBus event.EventBus,
	deduper *event.Deduper,
	logger *zap.Logger,
) *HeartbeatSubscriber {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &HeartbeatSubscriber{
		repo:     repo,
		redisCli: redisCli,
		eventBus: eventBus,
		logger:   logger.Named("mr-heartbeat-sub"),
		deduper:  deduper,
	}
}

// Subscribe 注册到 EventBus。幂等：重复调不会重复订阅（按 queue group 互斥）。
func (s *HeartbeatSubscriber) Subscribe() error {
	if s.eventBus == nil {
		return fmt.Errorf("mr heartbeat subscriber: eventBus is nil")
	}
	handler := event.EventHandler(s.handleMRFileUploaded)
	if s.deduper != nil {
		handler = s.deduper.Wrap("mr-heartbeat", handler)
	}
	if _, err := s.eventBus.QueueSubscribe(
		event.SubjectMRFileUploaded,
		"mr-heartbeat",
		handler,
	); err != nil {
		return fmt.Errorf("subscribe %s: %w", event.SubjectMRFileUploaded, err)
	}
	s.logger.Info("mr heartbeat subscribed", zap.String("subject", event.SubjectMRFileUploaded))
	return nil
}

// MRUploadPayload 与 acs.upload.publishMRFileUploadedEvent 的 payload 形状对齐。
type MRUploadPayload struct {
	Bucket     string `json:"bucket"`
	ObjectPath string `json:"object_path"`
	FileName   string `json:"file_name"`
	CellCode   string `json:"cell_code"`
	DeviceSN   string `json:"device_sn"`
	FileSize   int64  `json:"file_size"`
}

func (s *HeartbeatSubscriber) handleMRFileUploaded(ctx context.Context, evt event.Event) error {
	var p MRUploadPayload
	if err := evt.DecodePayload(&p); err != nil {
		return fmt.Errorf("decode mr.file.uploaded payload: %w", err)
	}
	if p.CellCode == "" {
		// 发布方已防御性过滤，但二次校验保留
		return nil
	}
	s.metrics.IncFileUploaded(p.CellCode)

	// 1) PG 心跳更新（无活跃任务时 matched=false，是合法的）
	matched, err := s.repo.TouchHeartbeat(ctx, p.CellCode)
	if err != nil {
		// PG 错误：log + 继续，让 Redis 心跳能写上（Redis 心跳是用户最直观的"在跑"信号）
		s.logger.Warn("touch heartbeat PG failed",
			zap.String("cell", p.CellCode), zap.Error(err))
	} else if !matched {
		s.logger.Debug("mr file uploaded but no active task progress matched",
			zap.String("cell", p.CellCode))
		// 没有活跃任务：不写 Redis 心跳（没人会读）
		return nil
	}

	// 2) 反查 cell 所属活跃任务，按 report_period 计算 TTL
	tasks, err := s.repo.ListActiveTasksByCell(ctx, p.CellCode)
	if err != nil {
		return fmt.Errorf("list active tasks by cell %s: %w", p.CellCode, err)
	}
	if len(tasks) == 0 {
		return nil // 与 matched=false 等价
	}
	// 一般只有 1 个任务；并发场景取最近一个（repo 已按 start_time DESC 排序）
	uploadPeriodSec, ok := UploadPeriodSeconds(tasks[0].ReportPeriod)
	if !ok {
		return fmt.Errorf("invalid report_period in task %s: %s",
			tasks[0].TaskID, tasks[0].ReportPeriod)
	}
	ttlSec := HeartbeatTTLSeconds(uploadPeriodSec)
	if ttlSec <= 0 {
		return nil
	}

	// 3) Redis 写入
	if s.redisCli != nil {
		key := HeartbeatRedisKey(p.CellCode)
		if err := s.redisCli.Set(ctx, key,
			time.Now().Format(time.RFC3339), time.Duration(ttlSec)*time.Second,
		).Err(); err != nil {
			// Redis 故障不向上抛（PG 端已更新），下一周期补救
			s.logger.Warn("write MR heartbeat redis failed",
				zap.String("key", key), zap.Error(err))
		}
	}

	s.logger.Debug("MR heartbeat refreshed",
		zap.String("cell", p.CellCode),
		zap.String("task_id", tasks[0].TaskID.String()),
		zap.Int("ttl_sec", ttlSec),
	)
	return nil
}
