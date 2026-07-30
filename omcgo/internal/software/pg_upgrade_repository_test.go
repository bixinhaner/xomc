package software

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// issue #667：覆盖 shouldSetSubTaskStartedAt 全部 9 种 UpgradeState。
// 这个判断决定了 sub_task 在哪些状态跃迁时写 started_at——前端「开始时间」列
// 是否能反映「真正轮到该设备级别开始升级」的精确时刻。误改会让 N 个离线设备
// 的「开始时间」全部用 createdAt（任务下发时刻）兜底显示，与并发调度行为不符。
func TestShouldSetSubTaskStartedAt(t *testing.T) {
	cases := []struct {
		status UpgradeState
		want   bool
		why    string
	}{
		// 写入：调度器已实际处理该 sub_task
		{UpgradeSuspended, true, "executor 把 sub_task 从 pending 挑出来发现设备离线 → 这就是「轮到该设备」时刻"},
		{UpgradeDownloading, true, "执行态：开始下载固件 / RPC 已派发"},
		{UpgradeUploading, true, "执行态：备份 / 日志开始上传"},
		{UpgradeRebooting, true, "执行态：5G 手动回退直跳 rebooting（qa-614 #371）"},
		{UpgradeVerifying, true, "执行态：升级后版本核验"},
		{UpgradeFailed, true, "pre-flight 失败（FIRMWARE_NOT_FOUND / DEVICE_NOT_FOUND）从 pending 直跳 failed，executor 已处理过"},
		// 不写入
		{UpgradePending, false, "排队中，调度器尚未挑出来"},
		{UpgradeCompleted, false, "终态必先经 verifying/uploading（已写过 started_at），不主动补写避免覆盖语义"},
		{UpgradeTerminated, false, "操作员主动叫停；若停在 pending 由 mapDeviceItem 显示层兜底 = endedAt（issue #655）"},
	}
	for _, c := range cases {
		t.Run(string(c.status), func(t *testing.T) {
			got := shouldSetSubTaskStartedAt(c.status)
			if got != c.want {
				t.Errorf("shouldSetSubTaskStartedAt(%q) = %v, want %v — %s",
					c.status, got, c.want, c.why)
			}
		})
	}
}

func TestDefaultUpgradeTaskReaperTimeouts_DownloadingWaitsTenMinutes(t *testing.T) {
	timeouts := defaultUpgradeTaskReaperTimeouts()
	if timeouts.RPCResponse != 10*time.Minute {
		t.Fatalf("downloading RPC response timeout = %s, want 10m", timeouts.RPCResponse)
	}
	if timeouts.TransferComplete != 30*time.Minute {
		t.Fatalf("TransferComplete timeout = %s, want 30m", timeouts.TransferComplete)
	}
}

func TestBuildFailStaleSubTasksSQL_DownloadingUsesDownloadTimeoutReason(t *testing.T) {
	query := buildFailStaleSubTasksSQL()

	if !strings.Contains(query, "WHEN ust.status = 'downloading' THEN 'Download response timed out: no DownloadResponse from device.'") {
		t.Fatalf("downloading stale message must describe DownloadResponse timeout, not TransferComplete.\nSQL: %s", query)
	}
	if !strings.Contains(query, "WHEN ust.status = 'downloading' THEN 'DOWNLOAD_TIMEOUT'") {
		t.Fatalf("downloading stale failure_reason must be DOWNLOAD_TIMEOUT so UI i18n does not show TransferComplete timeout.\nSQL: %s", query)
	}
	if strings.Contains(query, "WHEN ust.status = 'downloading' THEN 'Timed out waiting for TransferComplete") {
		t.Fatalf("downloading stale path must not mention TransferComplete.\nSQL: %s", query)
	}
}

// TestBuildUpdateSubTaskStatusSQL_StartedAtGate 验证 buildUpdateSubTaskStatusSQL 的 SQL
// 输出在 applyStartedAt=true（UpdateStatusWithCode）和 applyStartedAt=false
// （UpdateStatusByOperator）两条路径下对 started_at 列的处理差异。
//
// 业务语义（issue #667 后续）：
//   - executor 自然推进（applyStartedAt=true）：对 shouldSetSubTaskStartedAt 命中的状态
//     写入 started_at = COALESCE(started_at, now())。
//   - operator 主动操作（applyStartedAt=false）：永不动 started_at，等真正轮到设备被
//     executor 挑出来时再写——典型场景是 SuspendUpgrade 把 pending sub_task 翻成
//     suspended，此时 sub_task 还没被调度过。
func TestBuildUpdateSubTaskStatusSQL_StartedAtGate(t *testing.T) {
	id := uuid.New()

	t.Run("scheduler path (applyStartedAt=true) writes started_at for Suspended", func(t *testing.T) {
		sql, _, err := buildUpdateSubTaskStatusSQL(id, UpgradeSuspended, "waiting for device online", "", true)
		if err != nil {
			t.Fatalf("build SQL: %v", err)
		}
		if !strings.Contains(sql, "started_at") {
			t.Errorf("scheduler path should set started_at for Suspended (executor picked up offline device).\nSQL: %s", sql)
		}
	})

	t.Run("operator path (applyStartedAt=false) skips started_at for Suspended", func(t *testing.T) {
		sql, _, err := buildUpdateSubTaskStatusSQL(id, UpgradeSuspended, "task suspended by operator", "", false)
		if err != nil {
			t.Fatalf("build SQL: %v", err)
		}
		if strings.Contains(sql, "started_at") {
			t.Errorf("operator path should NOT touch started_at — operator suspending pending sub_task is not 'device picked up'.\nSQL: %s", sql)
		}
	})

	t.Run("operator path on Terminated still writes completed_at", func(t *testing.T) {
		sql, _, err := buildUpdateSubTaskStatusSQL(id, UpgradeTerminated, "task terminated by operator", "", false)
		if err != nil {
			t.Fatalf("build SQL: %v", err)
		}
		if strings.Contains(sql, "started_at") {
			t.Errorf("operator path should NOT touch started_at on Terminated.\nSQL: %s", sql)
		}
		if !strings.Contains(sql, "completed_at") {
			t.Errorf("Terminated must still set completed_at (terminal state).\nSQL: %s", sql)
		}
	})

	t.Run("scheduler path on Pending does not write started_at", func(t *testing.T) {
		sql, _, err := buildUpdateSubTaskStatusSQL(id, UpgradePending, "", "", true)
		if err != nil {
			t.Fatalf("build SQL: %v", err)
		}
		if strings.Contains(sql, "started_at") {
			t.Errorf("Pending is not a 'picked up' state — started_at must stay untouched.\nSQL: %s", sql)
		}
	})
}
