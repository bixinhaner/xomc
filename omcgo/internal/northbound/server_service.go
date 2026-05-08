package northbound

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// ServerService 北向 OSS 主备服务器业务逻辑。
//
// 与 NorthboundService 分离：那个聚合数据导出 / 推送 / 同步 等数据面能力；
// 本服务专注于"配置面"——主备服务器的 List 与 SetActive。后续 wave 把 push
// engine 改为读 ListActive 的结果作为推送目标即可关闭"切换功能未实际生效"的环。
type ServerService struct {
	repo   ServerRepository
	logger *zap.Logger
}

// NewServerService 创建服务实例。
func NewServerService(repo ServerRepository, logger *zap.Logger) *ServerService {
	return &ServerService{repo: repo, logger: logger}
}

// List 返回主备两行（按 role ASC：primary, standby）。
func (s *ServerService) List(ctx context.Context) ([]Server, error) {
	return s.repo.List(ctx)
}

// GetActive 返回当前激活的服务器；无激活返回 (nil, nil)。
func (s *ServerService) GetActive(ctx context.Context) (*Server, error) {
	servers, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range servers {
		if servers[i].IsActive {
			return &servers[i], nil
		}
	}
	return nil, nil
}

// SetActive 把指定 role 设为激活组（其它 role 自动变 inactive）。
// 若 role 已激活则 noop（不报错，前端可显示"已是激活状态"）。
func (s *ServerService) SetActive(ctx context.Context, role ServerRole) error {
	if !role.IsValid() {
		return fmt.Errorf("invalid role %q", role)
	}

	current, err := s.repo.GetByRole(ctx, role)
	if err != nil {
		return fmt.Errorf("get %s server: %w", role, err)
	}
	if current.IsActive {
		s.logger.Debug("northbound server already active",
			zap.String("role", string(role)))
		return nil
	}

	if err := s.repo.SetActive(ctx, role); err != nil {
		return fmt.Errorf("set %s active: %w", role, err)
	}
	s.logger.Info("northbound active server switched",
		zap.String("role", string(role)),
		zap.String("host", current.Host),
		zap.Int("port", current.Port))
	return nil
}

// Update 修改指定 role 的连接配置。host 必填，port ∈ [1, 65535]。
// role 不合法返回错误；不存在返回 ErrNotFound。
func (s *ServerService) Update(ctx context.Context, role ServerRole, req UpdateServerRequest) error {
	if !role.IsValid() {
		return fmt.Errorf("invalid role %q", role)
	}
	if req.Host == "" {
		return fmt.Errorf("host is required")
	}
	if req.Port < 1 || req.Port > 65535 {
		return fmt.Errorf("port must be in [1, 65535], got %d", req.Port)
	}

	if err := s.repo.Update(ctx, role, req.Host, req.Port, req.Description); err != nil {
		return fmt.Errorf("update %s server: %w", role, err)
	}
	s.logger.Info("northbound server config updated",
		zap.String("role", string(role)),
		zap.String("host", req.Host),
		zap.Int("port", req.Port))
	return nil
}
