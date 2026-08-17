package alarm

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------- Mock SMTP server (in-memory) ----------

// mockSMTPBehavior 控制 mock server 在不同阶段的响应。
type mockSMTPBehavior struct {
	// stallOnConnect=true 时连接后不发任何字节，用于触发 dispatcher 超时
	stallOnConnect bool
	// authFail=true 时 AUTH 阶段返 535 认证失败
	authFail bool
	// announceAuth 是否在 EHLO 响应里通告 AUTH PLAIN 扩展
	announceAuth bool
}

// mockSMTPServer 简易 RFC 5321 状态机；记录收到的命令行供断言。
type mockSMTPServer struct {
	listener net.Listener
	addr     string
	host     string
	port     int

	mu       sync.Mutex
	commands []string
	dataBody []byte

	behavior mockSMTPBehavior

	wg   sync.WaitGroup
	done chan struct{}
}

func newMockSMTPServer(t *testing.T, behavior mockSMTPBehavior) *mockSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "listen mock smtp")

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	srv := &mockSMTPServer{
		listener: ln,
		addr:     ln.Addr().String(),
		host:     host,
		port:     port,
		behavior: behavior,
		done:     make(chan struct{}),
	}

	srv.wg.Add(1)
	go srv.acceptLoop()
	return srv
}

func (s *mockSMTPServer) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				return
			}
		}
		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			s.handleConn(c)
		}(conn)
	}
}

func (s *mockSMTPServer) handleConn(conn net.Conn) {
	defer conn.Close()

	if s.behavior.stallOnConnect {
		// 不发 220，等连接被 dispatcher 关闭或测试结束
		<-s.done
		return
	}

	r := bufio.NewReader(conn)

	// 220 greeting
	if _, err := io.WriteString(conn, "220 mock.smtp ESMTP\r\n"); err != nil {
		return
	}

	inData := false
	var dataBuf strings.Builder

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				s.mu.Lock()
				s.dataBody = []byte(dataBuf.String())
				s.mu.Unlock()
				if _, err := io.WriteString(conn, "250 OK queued\r\n"); err != nil {
					return
				}
				continue
			}
			dataBuf.WriteString(line)
			dataBuf.WriteString("\r\n")
			continue
		}

		s.recordCommand(line)
		upper := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			// 多行 250 响应；通告 8BITMIME，按需通告 AUTH。
			if _, err := io.WriteString(conn, "250-mock.smtp Hello\r\n"); err != nil {
				return
			}
			if _, err := io.WriteString(conn, "250-8BITMIME\r\n"); err != nil {
				return
			}
			if s.behavior.announceAuth {
				if _, err := io.WriteString(conn, "250-AUTH PLAIN\r\n"); err != nil {
					return
				}
			}
			if _, err := io.WriteString(conn, "250 OK\r\n"); err != nil {
				return
			}
		case strings.HasPrefix(upper, "AUTH"):
			if s.behavior.authFail {
				if _, err := io.WriteString(conn, "535 5.7.8 authentication failed\r\n"); err != nil {
					return
				}
				continue
			}
			if _, err := io.WriteString(conn, "235 2.7.0 auth ok\r\n"); err != nil {
				return
			}
		case strings.HasPrefix(upper, "MAIL FROM"):
			if _, err := io.WriteString(conn, "250 OK sender\r\n"); err != nil {
				return
			}
		case strings.HasPrefix(upper, "RCPT TO"):
			if _, err := io.WriteString(conn, "250 OK recipient\r\n"); err != nil {
				return
			}
		case strings.HasPrefix(upper, "DATA"):
			inData = true
			if _, err := io.WriteString(conn, "354 end with .\r\n"); err != nil {
				return
			}
		case strings.HasPrefix(upper, "QUIT"):
			_, _ = io.WriteString(conn, "221 bye\r\n")
			return
		case strings.HasPrefix(upper, "NOOP"):
			_, _ = io.WriteString(conn, "250 OK\r\n")
		case strings.HasPrefix(upper, "RSET"):
			_, _ = io.WriteString(conn, "250 OK\r\n")
		default:
			_, _ = io.WriteString(conn, "502 5.5.2 command not implemented\r\n")
		}
	}
}

func (s *mockSMTPServer) recordCommand(line string) {
	s.mu.Lock()
	s.commands = append(s.commands, line)
	s.mu.Unlock()
}

func (s *mockSMTPServer) Commands() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.commands))
	copy(out, s.commands)
	return out
}

func (s *mockSMTPServer) DataBody() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return string(s.dataBody)
}

func (s *mockSMTPServer) Close() {
	close(s.done)
	_ = s.listener.Close()
	s.wg.Wait()
}

// ---------- Tests ----------

func TestSMTPEmailDispatcher_Dispatch_Success(t *testing.T) {
	t.Parallel()

	srv := newMockSMTPServer(t, mockSMTPBehavior{})
	t.Cleanup(srv.Close)

	cfg := EmailConfig{
		Enabled: true,
		Host:    srv.host,
		Port:    srv.port,
		From:    "alarm@omc.test",
		Timeout: 2 * time.Second,
	}
	reg := prometheus.NewRegistry()
	metrics := NewEmailMetrics(reg)
	d := NewSMTPEmailDispatcher(cfg, zap.NewNop(), metrics)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := d.Dispatch(ctx, []string{"ops@omc.test", "noc@omc.test"}, "Test Alarm", "Body line\nLine 2")
	require.NoError(t, err)

	cmds := srv.Commands()
	// 期望命令序列至少包含：EHLO / MAIL FROM / RCPT TO * 2 / DATA / QUIT
	requireCommandPrefix(t, cmds, "EHLO")
	requireCommandPrefix(t, cmds, "MAIL FROM")
	rcptCount := 0
	for _, c := range cmds {
		if strings.HasPrefix(strings.ToUpper(c), "RCPT TO") {
			rcptCount++
		}
	}
	assert.Equal(t, 2, rcptCount, "RCPT TO should be sent for each recipient")
	requireCommandPrefix(t, cmds, "DATA")

	body := srv.DataBody()
	assert.Contains(t, body, "Subject: Test Alarm")
	assert.Contains(t, body, "From: alarm@omc.test")
	assert.Contains(t, body, "To: ops@omc.test, noc@omc.test")
	assert.Contains(t, body, "MIME-Version: 1.0")
	assert.Contains(t, body, "Content-Type: text/plain; charset=\"utf-8\"")
	assert.Contains(t, body, "Body line")
}

func TestSMTPEmailDispatcher_Dispatch_Timeout(t *testing.T) {
	t.Parallel()

	srv := newMockSMTPServer(t, mockSMTPBehavior{stallOnConnect: true})
	t.Cleanup(srv.Close)

	cfg := EmailConfig{
		Enabled: true,
		Host:    srv.host,
		Port:    srv.port,
		From:    "alarm@omc.test",
		Timeout: 200 * time.Millisecond,
	}
	d := NewSMTPEmailDispatcher(cfg, zap.NewNop(), NewEmailMetrics(prometheus.NewRegistry()))

	start := time.Now()
	err := d.Dispatch(context.Background(), []string{"ops@omc.test"}, "Subj", "body")
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded), "expected DeadlineExceeded, got %v", err)
	assert.Less(t, elapsed, 600*time.Millisecond, "dispatch should return shortly after timeout")
}

func TestSMTPEmailDispatcher_Dispatch_AuthFail(t *testing.T) {
	t.Parallel()

	srv := newMockSMTPServer(t, mockSMTPBehavior{announceAuth: true, authFail: true})
	t.Cleanup(srv.Close)

	cfg := EmailConfig{
		Enabled:     true,
		AuthEnabled: true,
		Host:        srv.host,
		Port:        srv.port,
		From:        "alarm@omc.test",
		Username:    "user",
		Password:    "wrong",
		Timeout:     2 * time.Second,
	}
	d := NewSMTPEmailDispatcher(cfg, zap.NewNop(), NewEmailMetrics(prometheus.NewRegistry()))

	err := d.Dispatch(context.Background(), []string{"ops@omc.test"}, "Subj", "body")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp auth")
}

func TestSMTPEmailDispatcher_Dispatch_ParamValidation(t *testing.T) {
	t.Parallel()

	d := NewSMTPEmailDispatcher(EmailConfig{
		Enabled: true,
		Host:    "127.0.0.1",
		Port:    1, // never reached
		From:    "alarm@omc.test",
	}, zap.NewNop(), NewEmailMetrics(prometheus.NewRegistry()))

	cases := []struct {
		name    string
		to      []string
		subject string
		want    string
	}{
		{"empty to", nil, "subj", "no recipients"},
		{"empty addr in list", []string{"ok@omc.test", " "}, "subj", "invalid recipient address"},
		{"empty subject", []string{"ok@omc.test"}, " ", "email subject is required"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := d.Dispatch(context.Background(), tc.to, tc.subject, "body")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

// ---------- helpers ----------

func requireCommandPrefix(t *testing.T, cmds []string, prefix string) {
	t.Helper()
	for _, c := range cmds {
		if strings.HasPrefix(strings.ToUpper(c), strings.ToUpper(prefix)) {
			return
		}
	}
	t.Fatalf("expected command with prefix %q in: %v", prefix, cmds)
}
