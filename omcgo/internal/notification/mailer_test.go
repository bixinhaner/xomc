package notification

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeTransport 是 emailTransport 的测试替身。
type fakeTransport struct {
	calls       int
	err         error
	lastTo      []string
	lastSubject string
	lastBody    string
}

func (f *fakeTransport) Send(_ context.Context, to []string, subject, body string) error {
	f.calls++
	f.lastTo = to
	f.lastSubject = subject
	f.lastBody = body
	return f.err
}

// newTestMailer 用内存 repo + 给定 transport 装配 Mailer，返回 Mailer 与 history repo（供断言）。
func newTestMailer(t *testing.T, transport emailTransport) (*Mailer, *memHistoryRepo, *memTemplateRepo) {
	t.Helper()
	histRepo := newMemHistoryRepo()
	tplRepo := newMemTemplateRepo()
	m := NewMailer(
		NewTemplateService(tplRepo, nil),
		NewHistoryService(histRepo, nil),
		transport,
		nil,
	)
	return m, histRepo, tplRepo
}

func historyAll(t *testing.T, repo *memHistoryRepo) []NotificationHistory {
	t.Helper()
	resp, err := repo.List(context.Background(), NotificationHistoryFilter{})
	require.NoError(t, err)
	return resp.Items
}

func TestMailer_SendRaw_Success(t *testing.T) {
	t.Parallel()
	tr := &fakeTransport{}
	m, histRepo, _ := newTestMailer(t, tr)

	err := m.SendRaw(context.Background(), []string{"ops@x.com"}, "主题", "正文", nil)
	require.NoError(t, err)
	require.Equal(t, 1, tr.calls)
	require.Equal(t, []string{"ops@x.com"}, tr.lastTo)

	hist := historyAll(t, histRepo)
	require.Len(t, hist, 1)
	require.Equal(t, HistoryStatusSent, hist[0].Status)
	require.NotNil(t, hist[0].SentAt)
	require.Nil(t, hist[0].TemplateID)
}

func TestMailer_SendRaw_TransportFailure(t *testing.T) {
	t.Parallel()
	tr := &fakeTransport{err: errors.New("smtp down")}
	m, histRepo, _ := newTestMailer(t, tr)

	err := m.SendRaw(context.Background(), []string{"ops@x.com"}, "主题", "正文", nil)
	require.ErrorContains(t, err, "smtp down")

	hist := historyAll(t, histRepo)
	require.Len(t, hist, 1)
	require.Equal(t, HistoryStatusFailed, hist[0].Status)
	require.NotNil(t, hist[0].ErrorMessage)
	require.Contains(t, *hist[0].ErrorMessage, "smtp down")
}

func TestMailer_SendRaw_NoRecipients(t *testing.T) {
	t.Parallel()
	tr := &fakeTransport{}
	m, _, _ := newTestMailer(t, tr)

	err := m.SendRaw(context.Background(), nil, "主题", "正文", nil)
	require.ErrorContains(t, err, "no recipients")
	require.Equal(t, 0, tr.calls)
}

func TestMailer_SendRawRecorded_StopsWhenHistoryInsertFails(t *testing.T) {
	t.Parallel()
	tr := &fakeTransport{}
	m, histRepo, _ := newTestMailer(t, tr)
	histRepo.insertErr = errors.New("database unavailable")

	err := m.SendRawRecorded(context.Background(), []string{"ops@x.com"}, "主题", "正文", nil)
	require.ErrorContains(t, err, "record notification history")
	require.Equal(t, 0, tr.calls)
}

func TestMaskEmailAddresses(t *testing.T) {
	t.Parallel()
	require.Equal(t,
		[]string{"o***@example.com", "n***@example.com", "***"},
		maskEmailAddresses([]string{"operator@example.com", "Name <noc@example.com>", "invalid"}),
	)
}

func TestMailer_SendByTemplate_Success(t *testing.T) {
	t.Parallel()
	tr := &fakeTransport{}
	m, histRepo, tplRepo := newTestMailer(t, tr)

	require.NoError(t, tplRepo.Create(context.Background(), &NotificationTemplate{
		Name:    "alert-email",
		Channel: TemplateChannelEmail,
		Subject: "告警 {{.name}}",
		Body:    "设备 {{.sn}} 异常",
		Enabled: true,
	}))

	err := m.SendByTemplate(context.Background(), "alert-email",
		[]string{"ops@x.com"}, map[string]string{"name": "Down", "sn": "SN9"}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, tr.calls)
	require.Equal(t, "告警 Down", tr.lastSubject)
	require.Equal(t, "设备 SN9 异常", tr.lastBody)

	hist := historyAll(t, histRepo)
	require.Len(t, hist, 1)
	require.Equal(t, HistoryStatusSent, hist[0].Status)
	require.NotNil(t, hist[0].TemplateID)
}

func TestMailer_SendByTemplate_NotFound(t *testing.T) {
	t.Parallel()
	tr := &fakeTransport{}
	m, _, _ := newTestMailer(t, tr)

	err := m.SendByTemplate(context.Background(), "missing", []string{"ops@x.com"}, nil, nil)
	require.Error(t, err)
	require.Equal(t, 0, tr.calls)
}

func TestMailer_SendByTemplate_Disabled(t *testing.T) {
	t.Parallel()
	tr := &fakeTransport{}
	m, _, tplRepo := newTestMailer(t, tr)

	require.NoError(t, tplRepo.Create(context.Background(), &NotificationTemplate{
		Name: "off", Channel: TemplateChannelEmail, Subject: "s", Body: "b", Enabled: false,
	}))
	err := m.SendByTemplate(context.Background(), "off", []string{"ops@x.com"}, nil, nil)
	require.ErrorContains(t, err, "disabled")
	require.Equal(t, 0, tr.calls)
}

func TestMailer_SendByTemplate_WrongChannel(t *testing.T) {
	t.Parallel()
	tr := &fakeTransport{}
	m, _, tplRepo := newTestMailer(t, tr)

	require.NoError(t, tplRepo.Create(context.Background(), &NotificationTemplate{
		Name: "sms-tpl", Channel: TemplateChannelSMS, Subject: "s", Body: "b", Enabled: true,
	}))
	err := m.SendByTemplate(context.Background(), "sms-tpl", []string{"ops@x.com"}, nil, nil)
	require.ErrorContains(t, err, "email")
	require.Equal(t, 0, tr.calls)
}
