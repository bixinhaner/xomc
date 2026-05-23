package dashboard

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service 是 dashboard 业务层，把 Repository 包装为带 ACL 的对外接口。
//
// ACL 规则（与 share.go 一致）：
//   - owner 可读 + 可写 + 可删 + 可分享 + 可派生
//   - shared_with 用户可读 + 可派生（fork 出去成为自己的）
//   - 其它用户全部 403
type Service struct {
	repo   Repository
	logger *zap.Logger
}

// NewService 构造 Service。
func NewService(repo Repository, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repo: repo, logger: logger.Named("pm.dashboard.service")}
}

// Errors

var (
	ErrPermissionDenied = errors.New("dashboard: permission denied")
)

// ── Dashboard ────────────────────────────────────────────────────────────

func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, req CreateDashboardRequest) (*Dashboard, error) {
	return s.repo.CreateDashboard(ctx, ownerID, req)
}

// Get 读取 dashboard。校验 requester 是 owner 或 shared_with 成员。
func (s *Service) Get(ctx context.Context, requesterID, id uuid.UUID) (*Dashboard, error) {
	d, err := s.repo.GetDashboard(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanRead(d, requesterID) {
		return nil, ErrPermissionDenied
	}
	return d, nil
}

// List 列出 requester 拥有 + 被分享的所有 dashboard。
func (s *Service) List(ctx context.Context, requesterID uuid.UUID) ([]Dashboard, error) {
	return s.repo.ListByOwnerOrShared(ctx, requesterID)
}

// Update 仅 owner。
func (s *Service) Update(ctx context.Context, requesterID, id uuid.UUID, req UpdateDashboardRequest) error {
	d, err := s.repo.GetDashboard(ctx, id)
	if err != nil {
		return err
	}
	if !CanWrite(d, requesterID) {
		return ErrPermissionDenied
	}
	return s.repo.UpdateDashboard(ctx, id, req)
}

// Delete 仅 owner。
func (s *Service) Delete(ctx context.Context, requesterID, id uuid.UUID) error {
	d, err := s.repo.GetDashboard(ctx, id)
	if err != nil {
		return err
	}
	if !CanWrite(d, requesterID) {
		return ErrPermissionDenied
	}
	return s.repo.DeleteDashboard(ctx, id)
}

// Fork 派生新 dashboard。要求 requester 能读源 dashboard。新 dashboard owner = requester。
func (s *Service) Fork(ctx context.Context, requesterID, srcID uuid.UUID, newName string) (*Dashboard, error) {
	src, err := s.repo.GetDashboard(ctx, srcID)
	if err != nil {
		return nil, err
	}
	if !CanRead(src, requesterID) {
		return nil, ErrPermissionDenied
	}
	return s.repo.Fork(ctx, srcID, requesterID, newName)
}

// Share 仅 owner。
func (s *Service) Share(ctx context.Context, requesterID uuid.UUID, req ShareRequest) error {
	d, err := s.repo.GetDashboard(ctx, req.DashboardID)
	if err != nil {
		return err
	}
	if !CanWrite(d, requesterID) {
		return ErrPermissionDenied
	}
	// 过滤掉 owner 自己
	filtered := make([]uuid.UUID, 0, len(req.UserIDs))
	for _, uid := range req.UserIDs {
		if uid != d.OwnerID {
			filtered = append(filtered, uid)
		}
	}
	return s.repo.AddShare(ctx, req.DashboardID, filtered)
}

// Unshare 仅 owner。
func (s *Service) Unshare(ctx context.Context, requesterID uuid.UUID, req UnshareRequest) error {
	d, err := s.repo.GetDashboard(ctx, req.DashboardID)
	if err != nil {
		return err
	}
	if !CanWrite(d, requesterID) {
		return ErrPermissionDenied
	}
	return s.repo.RemoveShare(ctx, req.DashboardID, req.UserID)
}

// ── Panel ────────────────────────────────────────────────────────────────

func (s *Service) CreatePanel(ctx context.Context, requesterID uuid.UUID, req CreatePanelRequest) (*Panel, error) {
	d, err := s.repo.GetDashboard(ctx, req.DashboardID)
	if err != nil {
		return nil, err
	}
	if !CanWrite(d, requesterID) {
		return nil, ErrPermissionDenied
	}
	return s.repo.CreatePanel(ctx, req)
}

func (s *Service) UpdatePanel(ctx context.Context, requesterID, panelID uuid.UUID, req CreatePanelRequest) error {
	p, err := s.repo.GetPanel(ctx, panelID)
	if err != nil {
		return err
	}
	d, err := s.repo.GetDashboard(ctx, p.DashboardID)
	if err != nil {
		return err
	}
	if !CanWrite(d, requesterID) {
		return ErrPermissionDenied
	}
	req.DashboardID = p.DashboardID // 防止跨 dashboard 移动
	return s.repo.UpdatePanel(ctx, panelID, req)
}

func (s *Service) DeletePanel(ctx context.Context, requesterID, panelID uuid.UUID) error {
	p, err := s.repo.GetPanel(ctx, panelID)
	if err != nil {
		return err
	}
	d, err := s.repo.GetDashboard(ctx, p.DashboardID)
	if err != nil {
		return err
	}
	if !CanWrite(d, requesterID) {
		return ErrPermissionDenied
	}
	return s.repo.DeletePanel(ctx, panelID)
}

func (s *Service) ListPanels(ctx context.Context, requesterID, dashboardID uuid.UUID) ([]Panel, error) {
	d, err := s.repo.GetDashboard(ctx, dashboardID)
	if err != nil {
		return nil, err
	}
	if !CanRead(d, requesterID) {
		return nil, ErrPermissionDenied
	}
	return s.repo.ListPanels(ctx, dashboardID)
}

// ── UserPreferences ──────────────────────────────────────────────────────

// GetUserPreferences 取用户在指定制式下的偏好；不存在返默认空偏好（按制式）。
//
// T-0164 收尾 G6-Gap-3：技术维度独立持久化，每个用户在每种制式下偏好分离。
func (s *Service) GetUserPreferences(ctx context.Context, userID uuid.UUID, tech Technology) (*UserPreferences, error) {
	p, err := s.repo.GetUserPreferences(ctx, userID, tech)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return &UserPreferences{
				UserID:        userID,
				Technology:    tech,
				KPICardLayout: []byte(`{}`),
				SharedFilters: []byte(`{}`),
			}, nil
		}
		return nil, err
	}
	return p, nil
}

// UpsertUserPreferences 写整行偏好。caller 须构造完整 UserPreferences（含 Technology）。
func (s *Service) UpsertUserPreferences(ctx context.Context, p *UserPreferences) error {
	return s.repo.UpsertUserPreferences(ctx, p)
}
