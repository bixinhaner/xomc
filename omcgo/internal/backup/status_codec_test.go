package backup

import "testing"

func TestBackupStatusToCode(t *testing.T) {
	cases := []struct {
		in   TaskStatus
		want TaskStatusCode
	}{
		{TaskPending, TaskStatusCreated},
		{TaskRunning, TaskStatusRunning},
		{TaskCompleted, TaskStatusCompleted},
		{TaskFailed, TaskStatusCompleted}, // failed 与 completed 都落 4；用 task_result 区分
		{TaskCancelled, TaskStatusAborting},
		{TaskStatus("garbage"), TaskStatusCreated}, // 防御性默认值
	}
	for _, tc := range cases {
		if got := BackupStatusToCode(tc.in); got != tc.want {
			t.Errorf("BackupStatusToCode(%q) = %d; want %d", tc.in, got, tc.want)
		}
	}
}

func TestBackupResultFromStatus(t *testing.T) {
	cases := []struct {
		in       TaskStatus
		wantCode TaskResultCode
		wantOK   bool
	}{
		{TaskPending, 0, false},
		{TaskRunning, 0, false},
		{TaskCompleted, TaskResultSuccess, true},
		{TaskFailed, TaskResultFailed, true},
		{TaskCancelled, TaskResultFailed, true},
	}
	for _, tc := range cases {
		got, ok := BackupResultFromStatus(tc.in)
		if got != tc.wantCode || ok != tc.wantOK {
			t.Errorf("BackupResultFromStatus(%q) = (%d, %v); want (%d, %v)", tc.in, got, ok, tc.wantCode, tc.wantOK)
		}
	}
}

func TestRestoreStatusToCode(t *testing.T) {
	cases := []struct {
		in   RestoreStatus
		want TaskStatusCode
	}{
		{RestorePending, TaskStatusCreated},
		{RestoreRunning, TaskStatusRunning},
		{RestoreCompleted, TaskStatusCompleted},
		{RestoreFailed, TaskStatusCompleted},
		{RestoreCancelled, TaskStatusAborting},
		{RestoreStatus(""), TaskStatusCreated},
	}
	for _, tc := range cases {
		if got := RestoreStatusToCode(tc.in); got != tc.want {
			t.Errorf("RestoreStatusToCode(%q) = %d; want %d", tc.in, got, tc.want)
		}
	}
}

func TestRestoreResultFromStatus(t *testing.T) {
	cases := []struct {
		in       RestoreStatus
		wantCode TaskResultCode
		wantOK   bool
	}{
		{RestorePending, 0, false},
		{RestoreRunning, 0, false},
		{RestoreCompleted, TaskResultSuccess, true},
		{RestoreFailed, TaskResultFailed, true},
		{RestoreCancelled, TaskResultFailed, true},
	}
	for _, tc := range cases {
		got, ok := RestoreResultFromStatus(tc.in)
		if got != tc.wantCode || ok != tc.wantOK {
			t.Errorf("RestoreResultFromStatus(%q) = (%d, %v); want (%d, %v)", tc.in, got, ok, tc.wantCode, tc.wantOK)
		}
	}
}

func TestStatusFromCode_Roundtrip(t *testing.T) {
	// 对内部能映射的状态，From → To 应稳定。
	backupRoundtrip := []TaskStatus{TaskPending, TaskRunning, TaskCompleted, TaskCancelled}
	for _, s := range backupRoundtrip {
		code := BackupStatusToCode(s)
		back := BackupStatusFromCode(code)
		// failed/completed 都映射回 completed，是预期的有损映射
		if s == TaskCompleted && back != TaskCompleted {
			t.Errorf("backup roundtrip %q lost: code=%d back=%q", s, code, back)
		}
	}

	restoreRoundtrip := []RestoreStatus{RestorePending, RestoreRunning, RestoreCompleted, RestoreCancelled}
	for _, s := range restoreRoundtrip {
		code := RestoreStatusToCode(s)
		back := RestoreStatusFromCode(code)
		if s == RestoreCompleted && back != RestoreCompleted {
			t.Errorf("restore roundtrip %q lost: code=%d back=%q", s, code, back)
		}
	}

	// 规范预留态 (1/3/6) 在内部无映射 → 返回空串。
	if got := BackupStatusFromCode(TaskStatusWaiting); got != "" {
		t.Errorf("BackupStatusFromCode(1 等待中) = %q; want \"\"", got)
	}
	if got := RestoreStatusFromCode(TaskStatusPausing); got != "" {
		t.Errorf("RestoreStatusFromCode(6 暂停中) = %q; want \"\"", got)
	}
}
