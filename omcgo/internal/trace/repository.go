package trace

import (
	"context"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// Repository 抽象 trace_tasks + trace_messages 的持久化接口。
type Repository interface {
	// CreateTask 创建任务；若已存在 running 任务（同 SN），由 service 在调用前显式 stop 旧任务。
	CreateTask(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, id uuid.UUID) (*Task, error)
	GetRunningTaskBySN(ctx context.Context, sn string) (*Task, error)
	ListTasks(ctx context.Context, filter TaskFilter) (*model.ListResponse[Task], error)
	UpdateTaskStatus(ctx context.Context, id uuid.UUID, status TaskStatus) error
	IncrementMessageCount(ctx context.Context, id uuid.UUID, delta int) error
	// ListRunningSNs 返回当前所有 running 任务的 (sn, task_id) 映射，
	// ACS 进程启动期加载白名单 + 兜底对账使用。
	ListRunningSNs(ctx context.Context) (map[string]uuid.UUID, error)
	// ListExpired 返回 expires_at < now 且 status=running 的任务，巡检 worker 使用（M2 用）。
	ListExpired(ctx context.Context, limit int) ([]Task, error)
	// PurgeTaskMessages 物理删除任务下所有报文。
	PurgeTaskMessages(ctx context.Context, taskID uuid.UUID) error

	// DeleteTask 物理删除任务行（trace_tasks）。调用方需保证已
	// PurgeTaskMessages（外置对象清理由调用方/sweeper 负责）。未找到返回 ErrNotFound。
	DeleteTask(ctx context.Context, id uuid.UUID) error
	// BatchDeleteTasks 批量物理删除任务行；返回成功删除的行数。
	BatchDeleteTasks(ctx context.Context, ids []uuid.UUID) (int64, error)

	InsertMessage(ctx context.Context, msg *Message) error
	InsertMessages(ctx context.Context, msgs []*Message) error
	ListMessages(ctx context.Context, filter MessageFilter) (*model.ListResponse[Message], error)
	GetMessage(ctx context.Context, taskID, msgID uuid.UUID) (*Message, error)

	// ExportJob 异步下载相关（M2-08）
	CreateExportJob(ctx context.Context, job *ExportJob) error
	GetExportJob(ctx context.Context, id uuid.UUID) (*ExportJob, error)
	UpdateExportJob(ctx context.Context, job *ExportJob) error

	// StorageStats 返回 trace_messages 的 payload 字节总量分布：
	// inline = payload_object_key 为空的行（PG TOAST 存储）；
	// minio  = payload_object_key 非空的行（trace-bulk bucket 外置）。
	// worker StorageSampler 周期采样喂给 omc_trace_storage_bytes gauge。
	StorageStats(ctx context.Context) (inlineBytes, minioBytes int64, err error)
}
