package notification

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// mockRepository implements Repository in memory for unit tests.
type mockRepository struct {
	mu    sync.Mutex
	items map[uuid.UUID]*Notification

	// Failure injection — when non-nil, the matching method returns this error
	// instead of performing the in-memory operation.
	listErr           error
	getByIDErr        error
	createErr         error
	upsertErr         error // T-0157 C3: UpsertByDedup 失败注入
	markReadErr       error
	markAllReadErr    error
	getUnreadCountErr error
	deleteErr         error
	deleteAllErr      error // T-0157 C4

	// Optional intercept hook called on Create before storing.
	onCreate func(*Notification)
}

func newMockRepository() *mockRepository {
	return &mockRepository{items: make(map[uuid.UUID]*Notification)}
}

func (m *mockRepository) seed(notif *Notification) *Notification {
	m.mu.Lock()
	defer m.mu.Unlock()
	if notif.ID == uuid.Nil {
		notif.ID = uuid.New()
	}
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now()
	}
	cp := *notif
	m.items[notif.ID] = &cp
	return &cp
}

func (m *mockRepository) List(_ context.Context, filter NotificationFilter) (*model.ListResponse[Notification], error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	items := []Notification{}
	for _, n := range m.items {
		if n.UserID != filter.UserID {
			continue
		}
		if filter.Type != nil && n.Type != *filter.Type {
			continue
		}
		if filter.IsRead != nil && n.IsRead != *filter.IsRead {
			continue
		}
		items = append(items, *n)
	}
	total := int64(len(items))

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	return model.NewListResponse(items, total, page, pageSize), nil
}

func (m *mockRepository) GetByID(_ context.Context, id uuid.UUID) (*Notification, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if n, ok := m.items[id]; ok {
		cp := *n
		return &cp, nil
	}
	return nil, commonerrors.ErrNotFound
}

func (m *mockRepository) Create(_ context.Context, notif *Notification) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.onCreate != nil {
		m.onCreate(notif)
	}
	if notif.ID == uuid.Nil {
		notif.ID = uuid.New()
	}
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now()
	}
	cp := *notif
	m.items[notif.ID] = &cp
	return nil
}

// UpsertByDedup T-0157 C3: 按 (user_id, dedup_key) upsert；DedupKey 为空降级走 Create。
func (m *mockRepository) UpsertByDedup(ctx context.Context, notif *Notification) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	if notif.DedupKey == nil || *notif.DedupKey == "" {
		return m.Create(ctx, notif)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// 查找已存在的 (user_id, dedup_key)
	for _, n := range m.items {
		if n.UserID != notif.UserID || n.DedupKey == nil {
			continue
		}
		if *n.DedupKey == *notif.DedupKey {
			// 升级 status / title / content / link / priority；保留 id / created_at / is_read
			n.Status = notif.Status
			n.Type = notif.Type
			n.Priority = notif.Priority
			n.Title = notif.Title
			n.Content = notif.Content
			n.Link = notif.Link
			n.Sender = notif.Sender
			cp := *n
			*notif = cp
			return nil
		}
	}
	// 不存在则插入
	if notif.ID == uuid.Nil {
		notif.ID = uuid.New()
	}
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now()
	}
	cp := *notif
	m.items[notif.ID] = &cp
	return nil
}

func (m *mockRepository) MarkRead(_ context.Context, id uuid.UUID, userID string) error {
	if m.markReadErr != nil {
		return m.markReadErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	n, ok := m.items[id]
	if !ok || n.UserID != userID {
		return commonerrors.ErrNotFound
	}
	now := time.Now()
	n.IsRead = true
	n.ReadAt = &now
	return nil
}

func (m *mockRepository) MarkAllRead(_ context.Context, userID string) error {
	if m.markAllReadErr != nil {
		return m.markAllReadErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, n := range m.items {
		if n.UserID == userID && !n.IsRead {
			n.IsRead = true
			n.ReadAt = &now
		}
	}
	return nil
}

func (m *mockRepository) GetUnreadCount(_ context.Context, userID string) (int64, error) {
	if m.getUnreadCountErr != nil {
		return 0, m.getUnreadCountErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var c int64
	for _, n := range m.items {
		if n.UserID == userID && !n.IsRead {
			c++
		}
	}
	return c, nil
}

func (m *mockRepository) Delete(_ context.Context, id uuid.UUID, userID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	n, ok := m.items[id]
	if !ok || n.UserID != userID {
		return commonerrors.ErrNotFound
	}
	delete(m.items, id)
	return nil
}

// DeleteAllByUser T-0157 C4: 删除指定用户的全部消息。
func (m *mockRepository) DeleteAllByUser(_ context.Context, userID string) (int64, error) {
	if m.deleteAllErr != nil {
		return 0, m.deleteAllErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var deleted int64
	for id, n := range m.items {
		if n.UserID == userID {
			delete(m.items, id)
			deleted++
		}
	}
	return deleted, nil
}

// ListStaleByUser T-0157 stale sync mock。
func (m *mockRepository) ListStaleByUser(_ context.Context, userID string) ([]Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Notification
	for _, n := range m.items {
		if n.UserID != userID {
			continue
		}
		if n.Status != StatusQueued && n.Status != StatusSent {
			continue
		}
		if n.DedupKey == nil || *n.DedupKey == "" {
			continue
		}
		out = append(out, *n)
	}
	return out, nil
}

// UpdateStatusByID T-0157 stale sync mock。
func (m *mockRepository) UpdateStatusByID(_ context.Context, id uuid.UUID,
	status NotificationStatus, priority NotificationPriority, title, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n, ok := m.items[id]; ok {
		n.Status = status
		n.Priority = priority
		n.Title = title
		n.Content = content
	}
	return nil
}

// errBoom is a shared sentinel for failure-path testing.
var errBoom = errors.New("boom")
