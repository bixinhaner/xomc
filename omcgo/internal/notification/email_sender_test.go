package notification

import (
	"bufio"
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildMessage(t *testing.T) {
	t.Parallel()
	msg := buildMessage("from@x.com", []string{"a@x.com", "b@x.com"}, "测试主题", "line1\nline2")

	require.Contains(t, msg, "From: from@x.com\r\n")
	require.Contains(t, msg, "To: a@x.com, b@x.com\r\n")
	require.Contains(t, msg, "MIME-Version: 1.0\r\n")
	require.Contains(t, msg, "Content-Type: text/plain; charset=\"utf-8\"\r\n")
	// 中文主题须经 RFC 2047 编码
	require.Contains(t, msg, "Subject: =?")
	require.NotContains(t, msg, "Subject: 测试主题")
	// 正文换行统一为 CRLF
	require.Contains(t, msg, "line1\r\nline2")
	// header 与 body 之间有空行
	require.Contains(t, msg, "\r\n\r\n")
}

func TestEmailSender_Send_Disabled(t *testing.T) {
	t.Parallel()
	s := NewEmailSender(SMTPOptions{Enabled: false}, nil)
	err := s.Send(context.Background(), []string{"a@x.com"}, "s", "b")
	require.ErrorContains(t, err, "disabled")
}

func TestEmailSender_Send_NoHost(t *testing.T) {
	t.Parallel()
	s := NewEmailSender(SMTPOptions{Enabled: true, Host: ""}, nil)
	err := s.Send(context.Background(), []string{"a@x.com"}, "s", "b")
	require.ErrorContains(t, err, "host not configured")
}

func TestEmailSender_Send_NoRecipients(t *testing.T) {
	t.Parallel()
	s := NewEmailSender(SMTPOptions{Enabled: true, Host: "127.0.0.1", Port: 25}, nil)
	err := s.Send(context.Background(), nil, "s", "b")
	require.ErrorContains(t, err, "no recipients")
}

func TestEmailSender_Send_Success(t *testing.T) {
	t.Parallel()
	addr, getData := startFakeSMTP(t)
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, _ := strconv.Atoi(portStr)

	s := NewEmailSender(SMTPOptions{
		Enabled: true, Host: host, Port: port, From: "omc-alert@x.com",
	}, nil)
	err = s.Send(context.Background(), []string{"ops@x.com"}, "告警通知", "hello\nworld")
	require.NoError(t, err)

	data := getData()
	require.Contains(t, data, "To: ops@x.com\r\n")
	require.Contains(t, data, "From: omc-alert@x.com\r\n")
	require.Contains(t, data, "hello\r\nworld")
	require.Contains(t, data, "Subject: =?") // 中文主题已编码
}

// startFakeSMTP 起一个最小 SMTP 服务器（单连接），返回监听地址与「取已收到 DATA 正文」的闭包。
func startFakeSMTP(t *testing.T) (addr string, getData func() string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	var mu sync.Mutex
	var captured string
	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		write := func(s string) { _, _ = conn.Write([]byte(s)) }

		write("220 fake ESMTP ready\r\n")
		inData := false
		var body strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if line == ".\r\n" || line == ".\n" {
					inData = false
					mu.Lock()
					captured = body.String()
					mu.Unlock()
					write("250 OK\r\n")
					continue
				}
				body.WriteString(line)
				continue
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				write("250-fake\r\n250 HELP\r\n")
			case strings.HasPrefix(cmd, "DATA"):
				write("354 end with <CRLF>.<CRLF>\r\n")
				inData = true
			case strings.HasPrefix(cmd, "QUIT"):
				write("221 Bye\r\n")
				return
			default: // MAIL / RCPT / RSET / NOOP ...
				write("250 OK\r\n")
			}
		}
	}()

	t.Cleanup(func() {
		_ = ln.Close()
		<-done
	})
	return ln.Addr().String(), func() string {
		mu.Lock()
		defer mu.Unlock()
		return captured
	}
}
