package northbound

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/northbound/push"
)

// ServerService 北向 OSS 主备服务器业务逻辑。
//
// 与 NorthboundService 分离：那个聚合数据导出 / 推送 / 同步 等数据面能力；
// 本服务专注于"配置面"——主备服务器的 List 与 SetActive。
//
// T-0099：SetActive / Update 成功后通过 EventBus 发 SubjectNorthboundServerChanged
// 事件触发 push engine RefreshActiveTarget，关闭"切换/编辑即生效"的环。
// 启动期 push engine 也会调 GetActiveForPush 兜底一次。
type ServerService struct {
	repo     ServerRepository
	eventBus event.EventBus // 可 nil（测试 / 无事件总线场景），nil 时跳过 publish
	logger   *zap.Logger
}

// NewServerService 创建服务实例。eventBus 可传 nil（测试场景）。
func NewServerService(repo ServerRepository, eventBus event.EventBus, logger *zap.Logger) *ServerService {
	return &ServerService{repo: repo, eventBus: eventBus, logger: logger}
}

// serverChangedPayload 是 SubjectNorthboundServerChanged 事件载荷。
type serverChangedPayload struct {
	Role   string `json:"role"`
	Action string `json:"action"` // "active_switch" | "update"
	Host   string `json:"host"`
	Port   int    `json:"port"`
}

// publishServerChanged 异步发布事件；失败仅 warn，不阻塞主流程。
func (s *ServerService) publishServerChanged(ctx context.Context, role ServerRole, action, host string, port int) {
	if s.eventBus == nil {
		return
	}
	evt, err := event.NewEvent(event.SubjectNorthboundServerChanged, serverChangedPayload{
		Role: string(role), Action: action, Host: host, Port: port,
	})
	if err != nil {
		s.logger.Warn("build server.changed event failed", zap.Error(err))
		return
	}
	if err := s.eventBus.Publish(ctx, event.SubjectNorthboundServerChanged, evt); err != nil {
		s.logger.Warn("publish server.changed failed",
			zap.String("role", string(role)),
			zap.String("action", action),
			zap.Error(err))
	}
}

// GetActiveForPush 实现 push.ActiveServerProvider 接口（T-0099 关闭循环）。
//
// 返 *push.ActiveServerInfo（push 包的简化结构，避免反向 import），无激活
// 返 (nil, nil)；调用者（push engine）按 nil 处理为"移除占位 target"。
func (s *ServerService) GetActiveForPush(ctx context.Context) (*push.ActiveServerInfo, error) {
	srv, err := s.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	if srv == nil {
		return nil, nil
	}
	return &push.ActiveServerInfo{
		Role: string(srv.Role),
		Host: srv.Host,
		Port: srv.Port,
	}, nil
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
	s.publishServerChanged(ctx, role, "active_switch", current.Host, current.Port)
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
	// Update 只对当前 active server 影响 push 目标；非 active 的更新也发事件
	// （Engine 内会重新查 GetActive 决定要不要 refresh，行为统一）。
	s.publishServerChanged(ctx, role, "update", req.Host, req.Port)
	return nil
}
