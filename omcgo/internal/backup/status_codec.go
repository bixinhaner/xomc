// Package backup — task status codec (M1 of backup-restore-alignment-plan).
//
// The operator规范要求 `task_status` 用 0-6 七态整数表示，与运行时 task_result (1=成功 / 2=失败)
// 一起对外暴露。当前内部 backup_tasks.status / restore_tasks.status 仍用字符串
// (pending/running/...)，本文件提供双向映射，HTTP / 北向接口在序列化阶段使用。
//
// 状态映射（保持 internal string 不动，外暴露 int）：
//
//	0 已创建    ↔ pending
//	1 等待中    ↔ (内部目前无此态，预留：调度器排队但未派发)
//	2 进行中    ↔ running
//	3 已暂停    ↔ (内部目前无此态，预留：未来 SuspendTask 实现)
//	4 已完成    ↔ completed
//	5 终止中    ↔ (内部目前无此态，预留：终止请求已发起但未确认)
//	6 暂停中    ↔ (内部目前无此态，预留：暂停请求已发起但未确认)
//
// 此外两个内部态在规范中无直接对应：
//
//	failed     → 4 已完成 + task_result=2 失败
//	cancelled  → 5 终止中 (业务上 cancelled = 终止完成；规范没有"已终止"码，
//	             复用 5 表达，端方仍可通过 task_result/error_message 区分)
package backup

// TaskStatusCode 是规范 0-6 七态整数。
type TaskStatusCode int

const (
	TaskStatusCreated   TaskStatusCode = 0 // 已创建
	TaskStatusWaiting   TaskStatusCode = 1 // 等待中（预留）
	TaskStatusRunning   TaskStatusCode = 2 // 进行中
	TaskStatusSuspended TaskStatusCode = 3 // 已暂停（预留）
	TaskStatusCompleted TaskStatusCode = 4 // 已完成（含 failed —— 区分见 task_result）
	TaskStatusAborting  TaskStatusCode = 5 // 终止中（cancelled 映射至此）
	TaskStatusPausing   TaskStatusCode = 6 // 暂停中（预留）
)

// TaskResultCode 是规范 task_result：1=成功 / 2=失败。null = 未结束。
type TaskResultCode int

const (
	TaskResultSuccess TaskResultCode = 1
	TaskResultFailed  TaskResultCode = 2
)

// BackupStatusToCode maps backup TaskStatus to the 0-6 spec integer.
// Returns 0 for unknown values (defensive default — treats as "已创建" rather
// than panicking, since this codec runs on the response path).
func BackupStatusToCode(s TaskStatus) TaskStatusCode {
	switch s {
	case TaskPending:
		return TaskStatusCreated
	case TaskRunning:
		return TaskStatusRunning
	case TaskCompleted, TaskFailed:
		return TaskStatusCompleted
	case TaskCancelled:
		return TaskStatusAborting
	default:
		return TaskStatusCreated
	}
}

// BackupResultFromStatus derives the spec task_result for terminal backup states.
// Non-terminal states return (0, false) so callers can render NULL.
func BackupResultFromStatus(s TaskStatus) (TaskResultCode, bool) {
	switch s {
	case TaskCompleted:
		return TaskResultSuccess, true
	case TaskFailed, TaskCancelled:
		return TaskResultFailed, true
	default:
		return 0, false
	}
}

// RestoreStatusToCode mirrors BackupStatusToCode for restore tasks.
func RestoreStatusToCode(s RestoreStatus) TaskStatusCode {
	switch s {
	case RestorePending:
		return TaskStatusCreated
	case RestoreRunning:
		return TaskStatusRunning
	case RestoreCompleted, RestoreFailed:
		return TaskStatusCompleted
	case RestoreCancelled:
		return TaskStatusAborting
	default:
		return TaskStatusCreated
	}
}

// RestoreResultFromStatus mirrors BackupResultFromStatus for restore tasks.
func RestoreResultFromStatus(s RestoreStatus) (TaskResultCode, bool) {
	switch s {
	case RestoreCompleted:
		return TaskResultSuccess, true
	case RestoreFailed, RestoreCancelled:
		return TaskResultFailed, true
	default:
		return 0, false
	}
}

// BackupStatusFromCode is the inverse direction — accepts a spec 0-6 code from
// external input (e.g. north-bound API filter) and returns the internal
// TaskStatus to query by. Returns "" for codes that don't have a current
// internal counterpart (1 等待中 / 3 已暂停 / 6 暂停中); callers should treat
// "" as "no rows".
func BackupStatusFromCode(c TaskStatusCode) TaskStatus {
	switch c {
	case TaskStatusCreated:
		return TaskPending
	case TaskStatusRunning:
		return TaskRunning
	case TaskStatusCompleted:
		return TaskCompleted // failed 同样落 4；调用方按需结合 task_result 二次过滤
	case TaskStatusAborting:
		return TaskCancelled
	default:
		return ""
	}
}

// RestoreStatusFromCode mirrors BackupStatusFromCode for restore tasks.
func RestoreStatusFromCode(c TaskStatusCode) RestoreStatus {
	switch c {
	case TaskStatusCreated:
		return RestorePending
	case TaskStatusRunning:
		return RestoreRunning
	case TaskStatusCompleted:
		return RestoreCompleted
	case TaskStatusAborting:
		return RestoreCancelled
	default:
		return ""
	}
}
