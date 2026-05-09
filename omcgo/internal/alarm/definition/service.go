package definition

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Service 是 P3-04 handler 的业务编排层。
//
// 职责：
//   - 调用 WriteRepository 完成 CRUD
//   - 写后触发 Registry.Refresh 同步本实例 in-memory 缓存
//   - 写后/refresh 时递增 alarm:cache_version（dictloader §5.4 协议），
//     触发其他实例 30s 轮询感知失效
//   - 返回严重级反向查询结果给上层
type Service struct {
	repo     WriteRepository
	registry *Registry
	redis    redis.UniversalClient
	logger   *zap.Logger
}

// NewService 构造 Service；registry 为 nil 时仅有 DB 直读直写（仍可工作，但
// 缓存不刷）；rdb 为 nil 时跳过跨实例失效广播（单实例可接受）；生产路径必须传齐。
func NewService(repo WriteRepository, registry *Registry, rdb redis.UniversalClient, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repo: repo, registry: registry, redis: rdb, logger: logger.Named("alarmdef.service")}
}

// bumpCacheVersion 递增 alarm:cache_version（dictloader §5.4 标准协议），
// 触发其他实例的 30s 轮询感知失效。redis 为 nil 时静默跳过。
func (s *Service) bumpCacheVersion(ctx context.Context) {
	if s.redis == nil {
		return
	}
	s.redis.Incr(ctx, "alarm:cache_version")
}

// List 走过滤分页。
func (s *Service) List(ctx context.Context, f ListFilter) ([]ResolvedDefinition, int, error) {
	return s.repo.ListWithFilter(ctx, f)
}

// Get 单条详情。
func (s *Service) Get(ctx context.Context, identifier string) (*ResolvedDefinition, error) {
	return s.repo.GetByIdentifier(ctx, identifier)
}

// Create 新建并刷缓存。
func (s *Service) Create(ctx context.Context, in CreateInput) (*ResolvedDefinition, error) {
	rd, err := s.repo.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	s.refreshAsync(ctx, "create", in.Identifier)
	return rd, nil
}

// Update 局部更新并刷缓存。
func (s *Service) Update(ctx context.Context, identifier string, in UpdateInput) (*ResolvedDefinition, error) {
	rd, err := s.repo.Update(ctx, identifier, in)
	if err != nil {
		return nil, err
	}
	s.refreshAsync(ctx, "update", identifier)
	return rd, nil
}

// Delete 删除并刷缓存。
func (s *Service) Delete(ctx context.Context, identifier string) (bool, error) {
	ok, err := s.repo.Delete(ctx, identifier)
	if err != nil {
		return false, err
	}
	if ok {
		s.refreshAsync(ctx, "delete", identifier)
	}
	return ok, nil
}

// ListSeverityLevels 4 行只读种子。
func (s *Service) ListSeverityLevels(ctx context.Context) ([]SeverityLevel, error) {
	return s.repo.ListSeverityLevels(ctx)
}

// UnknownStats 治理闭环聚合查询。
func (s *Service) UnknownStats(ctx context.Context, productID *uuid.UUID, days int) ([]UnknownAlarmStat, error) {
	return s.repo.UnknownStats(ctx, productID, days)
}

// RefreshCache 手动刷新（HTTP cache/refresh）。
// 行为：本实例 Registry.Refresh + 跨实例 cache_version INCR。
func (s *Service) RefreshCache(ctx context.Context) error {
	if s.registry == nil {
		return fmt.Errorf("registry not wired; cache refresh disabled")
	}
	if err := s.registry.Refresh(ctx); err != nil {
		return err
	}
	s.bumpCacheVersion(ctx)
	return nil
}

// refreshAsync 写路径完成后异步刷缓存；失败仅 WARN（DB 已落地，缓存最多落后一拍）。
// 同时 INCR cache_version 通知其他实例。
func (s *Service) refreshAsync(ctx context.Context, op, identifier string) {
	if s.registry == nil {
		return
	}
	if err := s.registry.Refresh(ctx); err != nil {
		s.logger.Warn("alarm-def registry refresh after write failed",
			zap.String("op", op),
			zap.String("identifier", identifier),
			zap.Error(err),
		)
	}
	s.bumpCacheVersion(ctx)
}
