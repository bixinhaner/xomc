package software

import "testing"

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
