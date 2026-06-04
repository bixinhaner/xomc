package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/asyncjob"
)

// jobEnqueuer 是 Service 入队所需的最小契约（asyncjob.Repository 的子集），便于单测 stub。
type jobEnqueuer interface {
	Insert(ctx context.Context, req asyncjob.InsertRequest) (uuid.UUID, error)
}

// JobPayload 是 async_jobs.payload（job_type=pm_kpi_export）的字段集。
// worker 处理器据此载导出任务（T2 真生成时按 source_type 取数）。
type JobPayload struct {
	ExportTaskID string `json:"export_task_id"`
}

// BuildJobPayload 序列化 worker job payload，保证字段名与 JobPayload 对齐。
func BuildJobPayload(exportTaskID uuid.UUID) (json.RawMessage, error) {
	return json.Marshal(JobPayload{ExportTaskID: exportTaskID.String()})
}

// Service 编排导出任务创建：落表（pending）+ 入队 async_jobs。
type Service struct {
	repo    Repository
	jobRepo jobEnqueuer
}

// NewService 构造 Service。jobRepo 可为 nil（无队列环境下退化为只落表，Create 返错避免静默丢任务）。
func NewService(repo Repository, jobRepo jobEnqueuer) *Service {
	return &Service{repo: repo, jobRepo: jobRepo}
}

// ErrJobRepoNotWired Create 时发现 async job 仓库未注入。
var ErrJobRepoNotWired = errors.New("export: async job repo not wired")

// ErrInvalidSourceType 来源类型非法。
var ErrInvalidSourceType = errors.New("export: invalid source_type")

// Create 建导出任务：落表 status=pending → 入队 pm_kpi_export job → 返回任务记录。
//
// 入队失败时回滚（删除已落表的任务行），避免留下永远不会被处理的孤儿 pending 任务。
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Task, error) {
	if !req.SourceType.Valid() {
		return nil, ErrInvalidSourceType
	}
	if s.jobRepo == nil {
		return nil, ErrJobRepoNotWired
	}

	id, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create export task: %w", err)
	}

	payload, err := BuildJobPayload(id)
	if err != nil {
		_ = s.repo.Delete(ctx, id)
		return nil, fmt.Errorf("build job payload: %w", err)
	}
	if _, err := s.jobRepo.Insert(ctx, asyncjob.InsertRequest{
		JobType:     JobType,
		ScheduledAt: time.Now(),
		Payload:     payload,
	}); err != nil {
		// 入队失败回滚落表，保持任务与队列一致。
		_ = s.repo.Delete(ctx, id)
		return nil, fmt.Errorf("enqueue export job: %w", err)
	}

	return s.repo.Get(ctx, id)
}

// List 透传 Repository.List。
func (s *Service) List(ctx context.Context, filter ListFilter) ([]Task, error) {
	return s.repo.List(ctx, filter)
}

// Get 透传 Repository.Get。
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Task, error) {
	return s.repo.Get(ctx, id)
}

// Delete 透传 Repository.Delete（T1 只删任务记录，不连带删对象存储文件，留 T2 收尾）。
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
