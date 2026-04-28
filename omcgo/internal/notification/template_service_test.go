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
var _ TemplateRepository = (*memTemplateRepo)(nil)

// memTemplateRepo is a thread-safe in-memory TemplateRepository for unit tests.
type memTemplateRepo struct {
	mu    sync.Mutex
	items map[uuid.UUID]*NotificationTemplate
	names map[string]uuid.UUID
}

func newMemTemplateRepo() *memTemplateRepo {
	return &memTemplateRepo{
		items: make(map[uuid.UUID]*NotificationTemplate),
		names: make(map[string]uuid.UUID),
	}
}

func (m *memTemplateRepo) List(_ context.Context, filter NotificationTemplateFilter) (*model.ListResponse[NotificationTemplate], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]NotificationTemplate, 0, len(m.items))
	for _, t := range m.items {
		if filter.Channel != nil && *filter.Channel != t.Channel {
			continue
		}
		if filter.Language != nil && *filter.Language != t.Language {
			continue
		}
		if filter.Enabled != nil && *filter.Enabled != t.Enabled {
			continue
		}
		out = append(out, *t)
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

func (m *memTemplateRepo) GetByID(_ context.Context, id uuid.UUID) (*NotificationTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.items[id]
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (m *memTemplateRepo) GetByName(_ context.Context, name string) (*NotificationTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.names[name]
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	cp := *m.items[id]
	return &cp, nil
}

func (m *memTemplateRepo) Create(_ context.Context, tpl *NotificationTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.names[tpl.Name]; ok {
		return commonerrors.ErrAlreadyExists
	}
	if tpl.ID == uuid.Nil {
		tpl.ID = uuid.New()
	}
	now := time.Now()
	if tpl.CreatedAt.IsZero() {
		tpl.CreatedAt = now
	}
	tpl.UpdatedAt = now
	cp := *tpl
	m.items[tpl.ID] = &cp
	m.names[tpl.Name] = tpl.ID
	return nil
}

func (m *memTemplateRepo) Update(_ context.Context, tpl *NotificationTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.items[tpl.ID]
	if !ok {
		return commonerrors.ErrNotFound
	}
	if otherID, exists := m.names[tpl.Name]; exists && otherID != tpl.ID {
		return commonerrors.ErrAlreadyExists
	}
	delete(m.names, cur.Name)
	tpl.UpdatedAt = time.Now()
	cp := *tpl
	m.items[tpl.ID] = &cp
	m.names[tpl.Name] = tpl.ID
	return nil
}

func (m *memTemplateRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.items[id]
	if !ok {
		return commonerrors.ErrNotFound
	}
	delete(m.names, t.Name)
	delete(m.items, id)
	return nil
}

// ---- tests ----

func TestTemplateService_Create_Success(t *testing.T) {
	t.Parallel()
	svc := NewTemplateService(newMemTemplateRepo(), nil)

	tpl, err := svc.Create(context.Background(), CreateTemplateRequest{
		Name:      "alarm-critical-zh",
		Channel:   TemplateChannelEmail,
		Subject:   "Alert: {{title}}",
		Body:      "Hello {{user}}",
		Variables: []string{"title", "user", " title "}, // duplicate after trim
	})

	require.NoError(t, err)
	assert.Equal(t, "alarm-critical-zh", tpl.Name)
	assert.Equal(t, TemplateChannelEmail, tpl.Channel)
	assert.Equal(t, TemplateLanguageZhCN, tpl.Language) // default applied
	assert.True(t, tpl.Enabled)
	// Trimmed + deduped variables, order preserved.
	assert.Equal(t, []string{"title", "user"}, tpl.Variables)
}

func TestTemplateService_Create_RejectsBadChannel(t *testing.T) {
	t.Parallel()
	svc := NewTemplateService(newMemTemplateRepo(), nil)

	_, err := svc.Create(context.Background(), CreateTemplateRequest{
		Name:    "bad",
		Channel: "telegram", // unsupported
		Subject: "x",
		Body:    "y",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestTemplateService_Create_RejectsEmptyName(t *testing.T) {
	t.Parallel()
	svc := NewTemplateService(newMemTemplateRepo(), nil)

	_, err := svc.Create(context.Background(), CreateTemplateRequest{
		Name:    "   ", // only whitespace
		Channel: TemplateChannelEmail,
		Subject: "x",
		Body:    "y",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestTemplateService_Create_DuplicateName(t *testing.T) {
	t.Parallel()
	svc := NewTemplateService(newMemTemplateRepo(), nil)
	req := CreateTemplateRequest{
		Name:    "dup",
		Channel: TemplateChannelSMS,
		Subject: "s",
		Body:    "b",
	}
	_, err := svc.Create(context.Background(), req)
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrAlreadyExists))
}

func TestTemplateService_Update_PartialFields(t *testing.T) {
	t.Parallel()
	svc := NewTemplateService(newMemTemplateRepo(), nil)
	created, err := svc.Create(context.Background(), CreateTemplateRequest{
		Name:    "partial",
		Channel: TemplateChannelEmail,
		Subject: "old subject",
		Body:    "old body",
	})
	require.NoError(t, err)

	newSubject := "new subject"
	disabled := false
	updated, err := svc.Update(context.Background(), created.ID, UpdateTemplateRequest{
		Subject: &newSubject,
		Enabled: &disabled,
	})
	require.NoError(t, err)
	assert.Equal(t, "new subject", updated.Subject)
	assert.Equal(t, "old body", updated.Body) // unchanged
	assert.False(t, updated.Enabled)
}

func TestTemplateService_Update_NotFound(t *testing.T) {
	t.Parallel()
	svc := NewTemplateService(newMemTemplateRepo(), nil)

	_, err := svc.Update(context.Background(), uuid.New(), UpdateTemplateRequest{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
}

func TestTemplateService_Update_RejectsBlankName(t *testing.T) {
	t.Parallel()
	svc := NewTemplateService(newMemTemplateRepo(), nil)
	tpl, err := svc.Create(context.Background(), CreateTemplateRequest{
		Name:    "name1",
		Channel: TemplateChannelEmail,
		Subject: "s",
		Body:    "b",
	})
	require.NoError(t, err)

	blank := "   "
	_, err = svc.Update(context.Background(), tpl.ID, UpdateTemplateRequest{Name: &blank})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestTemplateService_Delete(t *testing.T) {
	t.Parallel()
	svc := NewTemplateService(newMemTemplateRepo(), nil)
	tpl, err := svc.Create(context.Background(), CreateTemplateRequest{
		Name:    "del",
		Channel: TemplateChannelWebhook,
		Subject: "s",
		Body:    "b",
	})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(context.Background(), tpl.ID))

	_, err = svc.GetByID(context.Background(), tpl.ID)
	assert.True(t, errors.Is(err, commonerrors.ErrNotFound))
}

func TestTemplateService_GetByName(t *testing.T) {
	t.Parallel()
	svc := NewTemplateService(newMemTemplateRepo(), nil)
	created, err := svc.Create(context.Background(), CreateTemplateRequest{
		Name:    "by-name",
		Channel: TemplateChannelEmail,
		Subject: "s",
		Body:    "b",
	})
	require.NoError(t, err)

	got, err := svc.GetByName(context.Background(), "by-name")
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	_, err = svc.GetByName(context.Background(), "")
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput))
}

func TestTemplateService_List_FiltersByChannel(t *testing.T) {
	t.Parallel()
	svc := NewTemplateService(newMemTemplateRepo(), nil)
	for _, ch := range []string{TemplateChannelEmail, TemplateChannelSMS, TemplateChannelEmail} {
		_, err := svc.Create(context.Background(), CreateTemplateRequest{
			Name:    "t-" + ch + "-" + uuid.NewString(),
			Channel: ch,
			Subject: "s",
			Body:    "b",
		})
		require.NoError(t, err)
	}

	emailCh := TemplateChannelEmail
	resp, err := svc.List(context.Background(), NotificationTemplateFilter{
		Channel:     &emailCh,
		ListRequest: model.DefaultListRequest(),
	})
	require.NoError(t, err)
	for _, tpl := range resp.Items {
		assert.Equal(t, TemplateChannelEmail, tpl.Channel)
	}
	assert.Equal(t, int64(2), resp.Total)
}
