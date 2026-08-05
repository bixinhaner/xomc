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

func TestEmailSender_SendOne_RejectsHeaderInjection(t *testing.T) {
	t.Parallel()
	sender := NewEmailSender(SMTPOptions{
		Enabled: true, Host: "127.0.0.1", Port: 25, From: "omc@example.com",
	}, nil)

	err := sender.SendOne(context.Background(), "ops@example.com", "alarm\r\nBcc: attacker@example.com", "body")

	require.ErrorContains(t, err, "line break")
}

func TestEmailSender_Send_Success(t *testing.T) {
	t.Parallel()
	addr, getData := startFakeSMTP(t, fakeSMTPBehavior{connections: 1, finalDataReply: true, quitReply: true})
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, _ := strconv.Atoi(portStr)

	s := NewEmailSender(SMTPOptions{
		Enabled: true, Host: host, Port: port, From: "omc-alert@x.com",
	}, nil)
	err = s.Send(context.Background(), []string{"ops@x.com"}, "告警通知", "hello\nworld")
	require.NoError(t, err)

	data := getData()[0]
	require.Contains(t, data, "To: ops@x.com\r\n")
	require.Contains(t, data, "From: omc-alert@x.com\r\n")
	require.Contains(t, data, "hello\r\nworld")
	require.Contains(t, data, "Subject: =?") // 中文主题已编码
}

func TestEmailSender_Send_UsesOneEnvelopePerRecipient(t *testing.T) {
	t.Parallel()
	addr, getData := startFakeSMTP(t, fakeSMTPBehavior{connections: 2, finalDataReply: true, quitReply: true})
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, _ := strconv.Atoi(portStr)
	sender := NewEmailSender(SMTPOptions{Enabled: true, Host: host, Port: port, From: "omc@x.com"}, nil)

	require.NoError(t, sender.Send(context.Background(), []string{"one@x.com", "two@x.com"}, "subject", "body"))

	messages := getData()
	require.Len(t, messages, 2)
	require.Contains(t, messages[0], "To: one@x.com\r\n")
	require.NotContains(t, messages[0], "two@x.com")
	require.Contains(t, messages[1], "To: two@x.com\r\n")
	require.NotContains(t, messages[1], "one@x.com")
}

func TestEmailSender_SendOne_QuitFailureDoesNotChangeAcceptedResult(t *testing.T) {
	t.Parallel()
	addr, _ := startFakeSMTP(t, fakeSMTPBehavior{connections: 1, finalDataReply: true, quitReply: false})
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, _ := strconv.Atoi(portStr)
	sender := NewEmailSender(SMTPOptions{Enabled: true, Host: host, Port: port, From: "omc@x.com"}, nil)

	require.NoError(t, sender.SendOne(context.Background(), "ops@x.com", "subject", "body"))
}

func TestEmailSender_VerifyPerformsNoEnvelopeOrData(t *testing.T) {
	t.Parallel()
	addr, getData := startFakeSMTP(t, fakeSMTPBehavior{connections: 1, finalDataReply: true, quitReply: true})
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, _ := strconv.Atoi(portStr)
	sender := NewEmailSender(SMTPOptions{Enabled: true, Host: host, Port: port, From: "omc@x.com"}, nil)

	require.NoError(t, sender.Verify(context.Background()))
	require.Empty(t, getData(), "verification must not issue DATA")
}

func TestEmailSender_SendOne_FinalDataResponseLossIsUnknown(t *testing.T) {
	t.Parallel()
	addr, _ := startFakeSMTP(t, fakeSMTPBehavior{connections: 1, finalDataReply: false, quitReply: false})
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, _ := strconv.Atoi(portStr)
	sender := NewEmailSender(SMTPOptions{Enabled: true, Host: host, Port: port, From: "omc@x.com"}, nil)

	err = sender.SendOne(context.Background(), "ops@x.com", "subject", "body")
	var sendErr *EmailSendError
	require.ErrorAs(t, err, &sendErr)
	require.Equal(t, EmailErrorUnknown, sendErr.Category)
	require.True(t, sendErr.OutcomeUnknown)
	require.False(t, sendErr.Retryable)
}

type fakeSMTPBehavior struct {
	connections    int
	finalDataReply bool
	quitReply      bool
}

// startFakeSMTP starts a minimal SMTP server and returns one captured DATA body
// per connection. Each connection therefore represents one SMTP envelope.
func startFakeSMTP(t *testing.T, behavior fakeSMTPBehavior) (addr string, getData func() []string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	var mu sync.Mutex
	captured := make([]string, 0, behavior.connections)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for range behavior.connections {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			handleFakeSMTPConnection(conn, behavior, &mu, &captured)
		}
	}()

	t.Cleanup(func() {
		_ = ln.Close()
		<-done
	})
	return ln.Addr().String(), func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), captured...)
	}
}

func handleFakeSMTPConnection(conn net.Conn, behavior fakeSMTPBehavior, mu *sync.Mutex, captured *[]string) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	write := func(value string) { _, _ = conn.Write([]byte(value)) }
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
				mu.Lock()
				*captured = append(*captured, body.String())
				mu.Unlock()
				if behavior.finalDataReply {
					write("250 OK\r\n")
				}
				if !behavior.finalDataReply {
					return
				}
				inData = false
				continue
			}
			body.WriteString(line)
			continue
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			write("250-fake\r\n250 HELP\r\n")
		case strings.HasPrefix(command, "DATA"):
			write("354 end with <CRLF>.<CRLF>\r\n")
			inData = true
		case strings.HasPrefix(command, "QUIT"):
			if behavior.quitReply {
				write("221 Bye\r\n")
			}
			return
		default:
			write("250 OK\r\n")
		}
	}
}
