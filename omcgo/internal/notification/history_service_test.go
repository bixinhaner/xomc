package notification

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// Compile-time interface check.
var _ HistoryRepository = (*memHistoryRepo)(nil)

// memHistoryRepo is a thread-safe in-memory HistoryRepository for unit tests.
type memHistoryRepo struct {
	mu        sync.Mutex
	items     map[uuid.UUID]*NotificationHistory
	insertErr error
}

func newMemHistoryRepo() *memHistoryRepo {
	return &memHistoryRepo{items: make(map[uuid.UUID]*NotificationHistory)}
}

func (m *memHistoryRepo) List(_ context.Context, filter NotificationHistoryFilter) (*model.ListResponse[NotificationHistory], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]NotificationHistory, 0, len(m.items))
	for _, h := range m.items {
		if filter.Channel != nil && *filter.Channel != h.Channel {
			continue
		}
		if filter.Status != nil && *filter.Status != h.Status {
			continue
		}
		if filter.TemplateID != nil {
			if h.TemplateID == nil || *h.TemplateID != *filter.TemplateID {
				continue
			}
		}
		if filter.AlarmID != nil {
			if h.AlarmID == nil || *h.AlarmID != *filter.AlarmID {
				continue
			}
		}
		out = append(out, *h)
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	return model.NewListResponse(out, int64(len(out)), page, pageSize), nil
}

func (m *memHistoryRepo) GetByID(_ context.Context, id uuid.UUID) (*NotificationHistory, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.items[id]
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	cp := *h
	return &cp, nil
}

func (m *memHistoryRepo) Insert(_ context.Context, h *NotificationHistory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.insertErr != nil {
		return m.insertErr
	}
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now()
	}
	cp := *h
	m.items[h.ID] = &cp
	return nil
}

func (m *memHistoryRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string, errorMessage *string, sentAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.items[id]
	if !ok {
		return commonerrors.ErrNotFound
	}
	h.Status = status
	h.ErrorMessage = errorMessage
	h.SentAt = sentAt
	return nil
}

// ---- tests ----

func TestHistoryService_Insert_Success(t *testing.T) {
	t.Parallel()
	svc := NewHistoryService(newMemHistoryRepo(), nil)

	h := &NotificationHistory{
		Channel:    TemplateChannelEmail,
		Recipients: []string{"a@example.com"},
		Subject:    "s",
		Body:       "b",
		Status:     HistoryStatusPending,
	}
	require.NoError(t, svc.Insert(context.Background(), h))
	assert.NotEqual(t, uuid.Nil, h.ID)
}

func TestHistoryService_Insert_RejectsBadChannel(t *testing.T) {
	t.Parallel()
	svc := NewHistoryService(newMemHistoryRepo(), nil)

	err := svc.Insert(context.Background(), &NotificationHistory{
		Channel:    "ftp",
		Recipients: []string{"a"},
		Status:     HistoryStatusPending,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestHistoryService_Insert_RejectsBadStatus(t *testing.T) {
	t.Parallel()
	svc := NewHistoryService(newMemHistoryRepo(), nil)

	err := svc.Insert(context.Background(), &NotificationHistory{
		Channel:    TemplateChannelSMS,
		Recipients: []string{"a"},
		Status:     "weird",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestHistoryService_Insert_RejectsEmptyRecipients(t *testing.T) {
	t.Parallel()
	svc := NewHistoryService(newMemHistoryRepo(), nil)

	err := svc.Insert(context.Background(), &NotificationHistory{
		Channel:    TemplateChannelEmail,
		Recipients: nil,
		Status:     HistoryStatusPending,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestHistoryService_Insert_RejectsNilEntry(t *testing.T) {
	t.Parallel()
	svc := NewHistoryService(newMemHistoryRepo(), nil)
	err := svc.Insert(context.Background(), nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestHistoryService_StatusTransitions(t *testing.T) {
	t.Parallel()
	repo := newMemHistoryRepo()
	svc := NewHistoryService(repo, nil)
	h := &NotificationHistory{
		Channel:    TemplateChannelWebhook,
		Recipients: []string{"http://example/hook"},
		Status:     HistoryStatusPending,
	}
	require.NoError(t, svc.Insert(context.Background(), h))

	now := time.Now()
	require.NoError(t, svc.MarkSent(context.Background(), h.ID, now))
	got, err := svc.GetByID(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, HistoryStatusSent, got.Status)
	require.NotNil(t, got.SentAt)

	require.NoError(t, svc.MarkFailed(context.Background(), h.ID, "boom"))
	got, err = svc.GetByID(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, HistoryStatusFailed, got.Status)
	require.NotNil(t, got.ErrorMessage)
	assert.Equal(t, "boom", *got.ErrorMessage)

	require.NoError(t, svc.MarkDeadLetter(context.Background(), h.ID, "max retries"))
	got, err = svc.GetByID(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Equal(t, HistoryStatusDeadLetter, got.Status)
}

func TestHistoryService_MarkFailed_BlankMessageBecomesNil(t *testing.T) {
	t.Parallel()
	repo := newMemHistoryRepo()
	svc := NewHistoryService(repo, nil)
	h := &NotificationHistory{
		Channel:    TemplateChannelEmail,
		Recipients: []string{"a@example"},
		Status:     HistoryStatusPending,
	}
	require.NoError(t, svc.Insert(context.Background(), h))

	require.NoError(t, svc.MarkFailed(context.Background(), h.ID, "   "))
	got, err := svc.GetByID(context.Background(), h.ID)
	require.NoError(t, err)
	assert.Nil(t, got.ErrorMessage)
}

func TestHistoryService_List_Filters(t *testing.T) {
	t.Parallel()
	repo := newMemHistoryRepo()
	svc := NewHistoryService(repo, nil)

	tplID := uuid.New()
	alarmID := uuid.New()

	entries := []NotificationHistory{
		{Channel: TemplateChannelEmail, Status: HistoryStatusSent, Recipients: []string{"a"}, TemplateID: &tplID, AlarmID: &alarmID},
		{Channel: TemplateChannelEmail, Status: HistoryStatusFailed, Recipients: []string{"b"}},
		{Channel: TemplateChannelSMS, Status: HistoryStatusSent, Recipients: []string{"c"}},
	}
	for i := range entries {
		require.NoError(t, svc.Insert(context.Background(), &entries[i]))
	}

	emailCh := TemplateChannelEmail
	resp, err := svc.List(context.Background(), NotificationHistoryFilter{
		Channel:     &emailCh,
		ListRequest: model.DefaultListRequest(),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)

	resp2, err := svc.List(context.Background(), NotificationHistoryFilter{
		AlarmID:     &alarmID,
		ListRequest: model.DefaultListRequest(),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp2.Total)

	resp3, err := svc.List(context.Background(), NotificationHistoryFilter{
		TemplateID:  &tplID,
		ListRequest: model.DefaultListRequest(),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp3.Total)
}

func TestHistoryService_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	svc := NewHistoryService(newMemHistoryRepo(), nil)
	_, err := svc.GetByID(context.Background(), uuid.New())
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
}
