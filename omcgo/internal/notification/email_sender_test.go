package notification

import (
	"bufio"
	"context"
	"encoding/base64"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildMessage(t *testing.T) {
	t.Parallel()
	msg := buildMessage("from@x.com", []string{"a@x.com", "b@x.com"}, "测试主题", "line1\nline2", "text/plain")

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

func TestBuildMessageHTMLContentType(t *testing.T) {
	t.Parallel()
	msg := buildMessage("from@x.com", []string{"a@x.com"}, "alarm", "<table></table>", "text/html", "<alarm-1@omc.local>")
	require.Contains(t, msg, "Content-Type: text/html; charset=\"utf-8\"\r\n")
	require.Contains(t, msg, "Message-ID: <alarm-1@omc.local>\r\n")
	require.Contains(t, msg, "<table></table>")
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
	s := NewEmailSender(SMTPOptions{Enabled: true, Host: "127.0.0.1", Port: 25, From: "omc@x.com"}, nil)
	err := s.Send(context.Background(), nil, "s", "b")
	require.ErrorContains(t, err, "no recipients")
}

func TestSMTPOptionsValidate_SecurityAndAuthentication(t *testing.T) {
	t.Parallel()

	base := SMTPOptions{
		Enabled: true,
		Host:    "smtp.example.com",
		Port:    25,
		From:    "omc@example.com",
	}
	require.NoError(t, base.Validate(), "内网无认证 SMTP 应合法")

	startTLS := base
	startTLS.SecurityMode = SMTPSecuritySTARTTLS
	require.NoError(t, startTLS.Validate())

	implicitTLS := base
	implicitTLS.SecurityMode = SMTPSecurityImplicitTLS
	require.NoError(t, implicitTLS.Validate())

	badMode := base
	badMode.SecurityMode = "vendor_tls"
	require.ErrorContains(t, badMode.Validate(), "unsupported smtp security mode")

	missingUsername := base
	missingUsername.AuthEnabled = true
	missingUsername.Password = "secret"
	require.ErrorContains(t, missingUsername.Validate(), "username is required")

	missingPassword := base
	missingPassword.AuthEnabled = true
	missingPassword.Username = "omc@example.com"
	require.ErrorContains(t, missingPassword.Validate(), "password is required")

	plaintextAuth := base
	plaintextAuth.AuthEnabled = true
	plaintextAuth.Username = "user"
	plaintextAuth.Password = "secret"
	require.ErrorContains(t, plaintextAuth.Validate(), "requires starttls or implicit_tls")
}

func TestNormalizeRecipients_TrimsAndDeduplicates(t *testing.T) {
	t.Parallel()

	got, err := normalizeRecipients([]string{
		" Ops@example.com ",
		"ops@EXAMPLE.com",
		"NOC <noc@example.com>",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"Ops@example.com", "noc@example.com"}, got)

	_, err = normalizeRecipients([]string{"not-an-email"})
	require.ErrorContains(t, err, "invalid recipient address")
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

func TestEmailSender_SendWithAttachments(t *testing.T) {
	t.Parallel()
	addr, getData := startFakeSMTP(t)
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	s := NewEmailSender(SMTPOptions{
		Enabled: true, Host: host, Port: port, From: "omc-report@x.com",
	}, nil)
	err = s.SendWithAttachments(
		context.Background(),
		[]string{"ops@x.com"},
		"KPI 日报",
		"report ready",
		[]EmailAttachment{{Filename: "测试报表.csv", ContentType: "text/csv; charset=utf-8", Data: []byte("a,b\n1,2\n")}},
	)
	require.NoError(t, err)

	data := getData()
	require.Contains(t, data, "Content-Type: multipart/mixed")
	require.Contains(t, data, "Content-Disposition: attachment;")
	require.Contains(t, data, "filename*=")
	require.Contains(t, data, base64.StdEncoding.EncodeToString([]byte("a,b\n1,2\n")))
}

func TestEmailSender_Send_STARTTLSRequiredButUnavailable(t *testing.T) {
	t.Parallel()
	addr, _ := startFakeSMTP(t)
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	s := NewEmailSender(SMTPOptions{
		Enabled:      true,
		Host:         host,
		Port:         port,
		From:         "omc-alert@x.com",
		SecurityMode: SMTPSecuritySTARTTLS,
	}, nil)
	err = s.Send(context.Background(), []string{"ops@x.com"}, "告警通知", "body")
	require.ErrorContains(t, err, "does not advertise STARTTLS")
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
