package sdnotify

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestWatchdogInterval(t *testing.T) {
	tests := []struct {
		name     string
		usec     string // "" = 视为未设置
		pid      string // "" = 视为未设置
		wantOK   bool
		wantHalf time.Duration
	}{
		{"unset", "", "", false, 0},
		{"30s without pid", "30000000", "", true, 15 * time.Second},
		{"30s matching pid", "30000000", strconv.Itoa(os.Getpid()), true, 15 * time.Second},
		{"30s wrong pid", "30000000", "999999", false, 0},
		{"invalid usec", "abc", "", false, 0},
		{"zero usec", "0", "", false, 0},
		{"negative usec", "-1", "", false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv 到空串等价于"未设置"（WatchdogInterval 把空串当未设），
			// 且测试结束自动还原，不污染其他用例。
			t.Setenv("WATCHDOG_USEC", tt.usec)
			t.Setenv("WATCHDOG_PID", tt.pid)

			got, ok := WatchdogInterval()
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got != tt.wantHalf {
				t.Fatalf("interval = %v, want %v", got, tt.wantHalf)
			}
		})
	}
}

func TestNotify_noopWithoutSocket(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "") // 非 systemd 环境
	if Ready() {
		t.Fatal("Ready 在无 NOTIFY_SOCKET 时必须 no-op（返回 false）")
	}
	if Stopping() {
		t.Fatal("Stopping 在无 NOTIFY_SOCKET 时必须 no-op（返回 false）")
	}
}

func TestReadyAndStopping_underSystemd(t *testing.T) {
	l := newListener(t)

	if !Ready() {
		t.Fatal("Ready 在有 NOTIFY_SOCKET 时应返回 true")
	}
	assertRecv(t, l, "READY=1")

	if !Stopping() {
		t.Fatal("Stopping 在有 NOTIFY_SOCKET 时应返回 true")
	}
	assertRecv(t, l, "STOPPING=1")
}

func TestStartWatchdog_petsThenStops(t *testing.T) {
	l := newListener(t)
	t.Setenv("WATCHDOG_USEC", "40000") // 40ms 超时 → 20ms 喂狗间隔
	t.Setenv("WATCHDOG_PID", strconv.Itoa(os.Getpid()))

	ctx, cancel := context.WithCancel(context.Background())
	StartWatchdog(ctx)

	assertRecv(t, l, "WATCHDOG=1") // 立即喂一次
	assertRecv(t, l, "WATCHDOG=1") // 至少一次 ticker 喂狗

	cancel()
	time.Sleep(60 * time.Millisecond) // 让 goroutine 观察到 ctx.Done
	drain(t, l)

	// ctx 取消后不应再有喂狗
	l.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	buf := make([]byte, 64)
	if _, _, err := l.ReadFromUnix(buf); err == nil {
		t.Fatal("ctx 取消后看门狗仍在喂狗")
	}
}

func TestStartWatchdog_noopWithoutWatchdog(t *testing.T) {
	t.Setenv("WATCHDOG_USEC", "") // 未启用看门狗
	StartWatchdog(context.Background())
	// 不 panic、不起 goroutine 即通过
}

// --- helpers ---

// newListener 建一个 unixgram 监听并把 NOTIFY_SOCKET 指向它，模拟 systemd。
func newListener(t *testing.T) *net.UnixConn {
	t.Helper()
	dir, err := os.MkdirTemp("", "sdn")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	sock := filepath.Join(dir, "n")
	l, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: sock, Net: "unixgram"})
	if err != nil {
		t.Fatalf("listen unixgram: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	t.Setenv("NOTIFY_SOCKET", sock)
	return l
}

func assertRecv(t *testing.T, l *net.UnixConn, want string) {
	t.Helper()
	_ = l.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 64)
	n, _, err := l.ReadFromUnix(buf)
	if err != nil {
		t.Fatalf("读取 sd_notify 数据报失败（期待 %q）：%v", want, err)
	}
	if got := string(buf[:n]); got != want {
		t.Fatalf("收到 %q，期待 %q", got, want)
	}
}

func drain(t *testing.T, l *net.UnixConn) {
	t.Helper()
	_ = l.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	buf := make([]byte, 64)
	for {
		if _, _, err := l.ReadFromUnix(buf); err != nil {
			return
		}
	}
}
