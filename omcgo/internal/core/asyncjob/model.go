// Package asyncjob 是 T-0164-P8 / G8 通用异步任务框架。
//
// 设计文档：docs/design/pm-kpi-pipeline-improvements.md §4.8
//
// 用途：进程级通用异步任务总线，承载 G5 cron 实例（小时/日/周/月聚合）+ 未来批量计算。
// G7 自定义聚合任务**不进**本框架（走 PM 模块 per-module 表 pm_tasks，设计 §4.7 锁定）。
//
// 状态机：
//
//	pending → running → succeeded
//	                 ↘  failed (attempt < max_attempts → 由调用方重新 Insert pending）
//	                 ↘  zombie (heartbeat_at < now()-5min, by Sweeper → ResetZombie 回到 pending)
//
// 装配模式：
//
//	repo := asyncjob.NewPgRepository(pool)
//	registry := asyncjob.NewRegistry(repo, lockOwner)
//	registry.Register(&MyAggregator{})  // 实现 JobRunner
//	// per-jobType goroutine 循环调 registry.RunNext(ctx, "my_job_type")
//	// sweeper 在另一 goroutine 调 NewSweeper(repo).Run(ctx) 60s tick
package asyncjob

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Status 是 async_jobs.status 的枚举值。
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusZombie    Status = "zombie"
	StatusCanceled  Status = "canceled"
)

// Job 对应 async_jobs 表的一行。
type Job struct {
	ID            uuid.UUID
	JobType       string
	Status        Status
	ScheduleExpr  string // cron expression（cron-triggered 任务才填）
	ScheduledAt   time.Time
	StartedAt     *time.Time
	FinishedAt    *time.Time
	HeartbeatAt   *time.Time
	LockOwner     string
	Attempt       int
	MaxAttempts   int
	Payload       json.RawMessage
	Result        json.RawMessage
	ErrorMessage  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// InsertRequest 创建新任务（status='pending'）。
type InsertRequest struct {
	JobType      string
	ScheduleExpr string
	ScheduledAt  time.Time
	Payload      json.RawMessage
	MaxAttempts  int // 0 表示用 DefaultMaxAttempts
}

// DefaultMaxAttempts 是 InsertRequest.MaxAttempts 未指定时的默认值。
const DefaultMaxAttempts = 3

// HeartbeatInterval 心跳上报间隔。
const HeartbeatInterval = 30 * time.Second

// ZombieThreshold heartbeat_at 距 now() 超过此值则视为僵尸（被 sweeper 重置）。
const ZombieThreshold = 5 * time.Minute

// SweeperInterval sweeper 扫描间隔。
const SweeperInterval = 60 * time.Second

// Errors

// ErrNoPendingJob LockNextPending 找不到可用任务。
var ErrNoPendingJob = errors.New("no pending async job available")

// ErrJobNotRunning UpdateHeartbeat / 完成类操作发现任务不在 running 状态。
var ErrJobNotRunning = errors.New("async job not in running state")

// ErrAttemptsExhausted ResetZombie 时发现已达 max_attempts，无法重置 pending（直接置 failed）。
var ErrAttemptsExhausted = errors.New("async job attempts exhausted")
