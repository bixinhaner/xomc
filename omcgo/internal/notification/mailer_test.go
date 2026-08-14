package notification

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/asyncjob"
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

type selectiveTransport struct {
	calls    map[string]int
	failures map[string]error
}

func (s *selectiveTransport) Send(_ context.Context, message EmailMessage) error {
	recipient := message.To[0]
	s.calls[recipient]++
	return s.failures[recipient]
}

type failingHistoryRepo struct {
	*memHistoryRepo
	markAttemptErr error
	markSentErr    error
}

// concurrentRetryHistoryRepo 让两个重试调用先同时读到 failed 状态，
// 以稳定复现“读状态后并发抢占”的竞态。
type concurrentRetryHistoryRepo struct {
	*memHistoryRepo
	mu      sync.Mutex
	waiters int
	ready   chan struct{}
}

func (r *concurrentRetryHistoryRepo) InsertIfAbsent(ctx context.Context, h *NotificationHistory) (*NotificationHistory, bool, error) {
	record, created, err := r.memHistoryRepo.InsertIfAbsent(ctx, h)
	if err != nil || created {
		return record, created, err
	}
	r.mu.Lock()
	r.waiters++
	if r.waiters == 2 {
		close(r.ready)
	}
	r.mu.Unlock()
	<-r.ready
	return record, false, nil
}

type blockingCountingTransport struct {
	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func (t *blockingCountingTransport) Send(_ context.Context, _ EmailMessage) error {
	if t.calls.Add(1) == 1 {
		close(t.entered)
	}
	<-t.release
	return nil
}

func (r *failingHistoryRepo) ClaimAttempt(ctx context.Context, id uuid.UUID, attemptedAt, staleBefore time.Time) (bool, error) {
	if r.markAttemptErr != nil {
		return false, r.markAttemptErr
	}
	return r.memHistoryRepo.ClaimAttempt(ctx, id, attemptedAt, staleBefore)
}

func (r *failingHistoryRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, sentAt *time.Time) error {
	if status == HistoryStatusSent && r.markSentErr != nil {
		return r.markSentErr
	}
	return r.memHistoryRepo.UpdateStatus(ctx, id, status, errorMessage, sentAt)
}

func (f *fakeTransport) Send(_ context.Context, message EmailMessage) error {
	f.calls++
	f.lastTo = message.To
	f.lastSubject = message.Subject
	f.lastBody = message.TextBody
	return f.err
}

// newTestMailer 用内存 repo + 给定 transport 装配 Mailer，返回 Mailer 与 history repo（供断言）。
func newTestMailer(t *testing.T, transport EmailTransport) (*Mailer, *memHistoryRepo, *memTemplateRepo) {
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

func TestMailer_SendMessage_DeduplicatesSentDelivery(t *testing.T) {
	t.Parallel()
	tr := &fakeTransport{}
	m, histRepo, _ := newTestMailer(t, tr)
	message := EmailMessage{To: []string{"ops@x.com"}, Subject: "告警", TextBody: "正文"}
	metadata := DeliveryMetadata{BusinessType: "alarm", BusinessID: "A-1", DedupKey: "alarm:A-1:raised"}

	require.NoError(t, m.SendMessage(context.Background(), message, metadata))
	require.NoError(t, m.SendMessage(context.Background(), message, metadata))
	require.Equal(t, 1, tr.calls)
	hist := historyAll(t, histRepo)
	require.Len(t, hist, 1)
	require.Equal(t, 1, hist[0].RetryCount)
	require.NotNil(t, hist[0].AttemptedAt)
	require.NotNil(t, hist[0].DedupKey)
	require.Equal(t, metadata.DedupKey, *hist[0].DedupKey)
}

func TestMailer_SendMessage_DeadLetterIsTerminalFailure(t *testing.T) {
	t.Parallel()
	transport := &fakeTransport{err: errors.New("mailbox unavailable")}
	mailer, historyRepo, _ := newTestMailer(t, transport)
	message := EmailMessage{To: []string{"ops@x.com"}, Subject: "告警", TextBody: "正文"}
	metadata := DeliveryMetadata{DedupKey: "alarm:A-dead:raised", FinalAttempt: true}

	err := mailer.SendMessage(context.Background(), message, metadata)
	require.ErrorContains(t, err, "mailbox unavailable")
	history := historyAll(t, historyRepo)
	require.Len(t, history, 1)
	require.Equal(t, HistoryStatusDeadLetter, history[0].Status)
	require.Equal(t, 1, transport.calls)

	transport.err = nil
	err = mailer.SendMessage(context.Background(), message, metadata)
	require.ErrorIs(t, err, ErrDeliveryDeadLetter)
	require.Equal(t, 1, transport.calls, "dead-lettered delivery must not be resent or reported successful")
}

func TestMailer_SendMessage_ConcurrentFailedRetryIsClaimedOnce(t *testing.T) {
	t.Parallel()
	baseRepo := newMemHistoryRepo()
	dedupKey := "alarm:A-concurrent:raised"
	record := &NotificationHistory{
		Channel:    TemplateChannelEmail,
		Recipients: []string{"ops@x.com"},
		Subject:    "告警",
		Body:       "正文",
		Status:     HistoryStatusFailed,
		DedupKey:   &dedupKey,
	}
	require.NoError(t, baseRepo.Insert(context.Background(), record))

	repo := &concurrentRetryHistoryRepo{memHistoryRepo: baseRepo, ready: make(chan struct{})}
	transport := &blockingCountingTransport{entered: make(chan struct{}), release: make(chan struct{})}
	mailer := NewMailer(nil, NewHistoryService(repo, nil), transport, nil)
	message := EmailMessage{To: record.Recipients, Subject: record.Subject, TextBody: record.Body}
	metadata := DeliveryMetadata{DedupKey: dedupKey}

	errs := make(chan error, 2)
	for range 2 {
		go func() {
			errs <- mailer.SendMessage(context.Background(), message, metadata)
		}()
	}

	<-transport.entered
	var duplicateErr error
	select {
	case duplicateErr = <-errs:
	case <-time.After(2 * time.Second):
		close(transport.release)
		t.Fatal("duplicate retry was not fenced while the claimed SMTP delivery was in progress")
	}
	require.ErrorIs(t, duplicateErr, ErrDeliveryPending)
	close(transport.release)
	require.NoError(t, <-errs)
	require.Equal(t, int32(1), transport.calls.Load(), "concurrent retries must produce one SMTP delivery")

	history, err := baseRepo.GetByID(context.Background(), record.ID)
	require.NoError(t, err)
	require.Equal(t, HistoryStatusSent, history.Status)
	require.Equal(t, 1, history.RetryCount)
}

func TestMailer_SendMessageBatch_IsolatesRecipientFailuresAndRetriesOnlyFailed(t *testing.T) {
	t.Parallel()
	transport := &selectiveTransport{
		calls: map[string]int{},
		failures: map[string]error{
			"rejected@example.com": errors.New("mailbox rejected"),
		},
	}
	mailer, historyRepo, _ := newTestMailer(t, transport)
	message := EmailMessage{
		To:       []string{"accepted@example.com", "rejected@example.com"},
		Subject:  "告警",
		TextBody: "正文",
	}
	metadata := DeliveryMetadata{BusinessType: "alarm", BusinessID: "A-5", DedupKey: "alarm:A-5:raised"}

	results, err := mailer.SendMessageBatch(context.Background(), message, metadata)
	require.ErrorContains(t, err, "mailbox rejected")
	require.Len(t, results, 1)
	require.Equal(t, "accepted@example.com", results[0].Recipient)
	require.Equal(t, 1, transport.calls["accepted@example.com"])
	require.Equal(t, 1, transport.calls["rejected@example.com"])

	histories := historyAll(t, historyRepo)
	require.Len(t, histories, 2)
	statusByRecipient := make(map[string]string, len(histories))
	dedupKeys := make(map[string]struct{}, len(histories))
	for _, history := range histories {
		require.Len(t, history.Recipients, 1)
		statusByRecipient[history.Recipients[0]] = history.Status
		require.NotNil(t, history.DedupKey)
		dedupKeys[*history.DedupKey] = struct{}{}
	}
	require.Equal(t, HistoryStatusSent, statusByRecipient["accepted@example.com"])
	require.Equal(t, HistoryStatusFailed, statusByRecipient["rejected@example.com"])
	require.Len(t, dedupKeys, 2)

	delete(transport.failures, "rejected@example.com")
	results, err = mailer.SendMessageBatch(context.Background(), message, metadata)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, 1, transport.calls["accepted@example.com"], "sent recipient must be skipped on retry")
	require.Equal(t, 2, transport.calls["rejected@example.com"], "only failed recipient is retried")

	for _, history := range historyAll(t, historyRepo) {
		require.Equal(t, HistoryStatusSent, history.Status)
	}
}

func TestMailer_SendMessage_RetriesStalePendingDelivery(t *testing.T) {
	t.Parallel()
	tr := &fakeTransport{}
	m, histRepo, _ := newTestMailer(t, tr)
	metadata := DeliveryMetadata{BusinessType: "alarm", BusinessID: "A-2", DedupKey: "alarm:A-2:raised"}
	oldAttempt := time.Now().Add(-pendingRetryAfter - time.Minute)
	dedupKey := metadata.DedupKey
	record := &NotificationHistory{
		Channel:      TemplateChannelEmail,
		Recipients:   []string{"ops@x.com"},
		Subject:      "告警",
		Body:         "正文",
		Status:       HistoryStatusPending,
		BusinessType: stringPointer(metadata.BusinessType),
		BusinessID:   stringPointer(metadata.BusinessID),
		DedupKey:     &dedupKey,
	}
	require.NoError(t, histRepo.Insert(context.Background(), record))
	claimed, err := histRepo.ClaimAttempt(context.Background(), record.ID, oldAttempt, oldAttempt.Add(-pendingRetryAfter))
	require.NoError(t, err)
	require.True(t, claimed)

	err = m.SendMessage(context.Background(), EmailMessage{To: record.Recipients, Subject: record.Subject, TextBody: record.Body}, metadata)
	require.NoError(t, err)
	require.Equal(t, 1, tr.calls)
	hist := historyAll(t, histRepo)
	require.Len(t, hist, 1)
	require.Equal(t, HistoryStatusSent, hist[0].Status)
	require.Equal(t, 2, hist[0].RetryCount)
}

func TestMailerPendingFenceExpiresBeforeAsyncJobZombieRecovery(t *testing.T) {
	t.Parallel()
	require.Less(t, pendingRetryAfter, asyncjob.ZombieThreshold)
}

func TestMailer_SendMessage_DoesNotSendWhenIdempotentAttemptCannotBeRecorded(t *testing.T) {
	t.Parallel()
	transport := &fakeTransport{}
	repo := &failingHistoryRepo{memHistoryRepo: newMemHistoryRepo(), markAttemptErr: errors.New("database unavailable")}
	mailer := NewMailer(nil, NewHistoryService(repo, nil), transport, nil)

	err := mailer.SendMessage(context.Background(), EmailMessage{
		To: []string{"ops@x.com"}, Subject: "告警", TextBody: "正文",
	}, DeliveryMetadata{DedupKey: "alarm:A-3:raised"})

	require.ErrorContains(t, err, "record idempotent email attempt")
	require.Zero(t, transport.calls)
}

func TestMailer_SendMessage_DoesNotReportSuccessWhenSentStatusCannotBeRecorded(t *testing.T) {
	t.Parallel()
	transport := &fakeTransport{}
	repo := &failingHistoryRepo{memHistoryRepo: newMemHistoryRepo(), markSentErr: errors.New("database unavailable")}
	mailer := NewMailer(nil, NewHistoryService(repo, nil), transport, nil)
	message := EmailMessage{To: []string{"ops@x.com"}, Subject: "告警", TextBody: "正文"}
	metadata := DeliveryMetadata{DedupKey: "alarm:A-4:raised"}

	err := mailer.SendMessage(context.Background(), message, metadata)
	require.ErrorContains(t, err, "record idempotent email sent status")
	require.Equal(t, 1, transport.calls)

	err = mailer.SendMessage(context.Background(), message, metadata)
	require.ErrorIs(t, err, ErrDeliveryPending)
	require.Equal(t, 1, transport.calls, "fresh pending history must fence a duplicate SMTP send")
}

func stringPointer(value string) *string { return &value }

func TestMailer_SendRaw_NoRecipients(t *testing.T) {
	t.Parallel()
	tr := &fakeTransport{}
	m, _, _ := newTestMailer(t, tr)

	err := m.SendRaw(context.Background(), nil, "主题", "正文", nil)
	require.ErrorContains(t, err, "no recipients")
	require.Equal(t, 0, tr.calls)
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
