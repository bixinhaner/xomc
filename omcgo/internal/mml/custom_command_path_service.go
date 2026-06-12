package mml

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// custom_command_path_service.go — issue #115 调整3（A1）
//
// 自定义命令 PATH 关联的增删改查业务逻辑。写操作鉴权与 UpdateCustomCommand /
// DeleteCustomCommand 一致（docs/design/mml-user-private-template-crud-20260520.md §4.3）：
// 要求调用者是命令 owner 或 super_admin；读（List）经端点级 RBAC 收敛即可（path 非敏感）。

// SetCustomCommandPathRepo 注入自定义命令 path 关联表仓库（issue #115 调整3）。
func (s *Service) SetCustomCommandPathRepo(repo CustomCommandPathRepository) {
	s.customCommandPathRepo = repo
}

// ensurePathRepo 守卫：path 仓库未注入时报错（理论上 DI 保证注入，纵深防御）。
// wrap ErrUnavailable → 503，与 service struct 上"nil 时 path 端点返 503"的注释一致。
func (s *Service) ensurePathRepo() error {
	if s.customCommandPathRepo == nil {
		return fmt.Errorf("custom command path repository not configured: %w", commonerrors.ErrUnavailable)
	}
	return nil
}

// ListCustomCommandPaths 列出命令的 path 关联（JOIN 字典富化）。
// 仅校验命令存在；不做 owner 校验（path 列表非敏感，端点级 RBAC 已收敛 viewer+）。
func (s *Service) ListCustomCommandPaths(ctx context.Context, commandID uuid.UUID) ([]MMLCustomCommandPathView, error) {
	if err := s.ensurePathRepo(); err != nil {
		return nil, err
	}
	if _, err := s.customCommandRepo.GetByID(ctx, commandID); err != nil {
		return nil, fmt.Errorf("get mml custom command: %w", err)
	}
	return s.customCommandPathRepo.ListByCommand(ctx, commandID)
}

// BatchAddCustomCommandPaths 批量追加 path（owner/super 才能写）。
func (s *Service) BatchAddCustomCommandPaths(
	ctx context.Context,
	commandID uuid.UUID,
	standardPathIDs []uuid.UUID,
	currentUserID uuid.UUID,
	currentUsername string,
	isSuperAdmin bool,
) ([]MMLCustomCommandPath, error) {
	if err := s.ensurePathRepo(); err != nil {
		return nil, err
	}
	if len(standardPathIDs) == 0 {
		return nil, fmt.Errorf("standard_path_ids must be non-empty: %w", commonerrors.ErrInvalidInput)
	}
	cmd, err := s.customCommandRepo.GetByID(ctx, commandID)
	if err != nil {
		return nil, fmt.Errorf("get mml custom command: %w", err)
	}
	if !isOwnerOrSuper(cmd, currentUserID, currentUsername, isSuperAdmin) {
		return nil, fmt.Errorf("only creator or super_admin can modify command paths: %w", commonerrors.ErrForbidden)
	}
	created, err := s.customCommandPathRepo.BatchCreate(ctx, commandID, standardPathIDs)
	if err != nil {
		return nil, err
	}
	s.logger.Info("mml custom command paths batch-added",
		zap.String("command_id", commandID.String()),
		zap.Int("added", len(created)),
		zap.String("actor", currentUserID.String()),
	)
	return created, nil
}

// UpdateCustomCommandPath 改单条关联的 default_selected / sort_order（owner/super 才能写）。
func (s *Service) UpdateCustomCommandPath(
	ctx context.Context,
	commandID, pathID uuid.UUID,
	defaultSelected *bool,
	sortOrder *int,
	currentUserID uuid.UUID,
	currentUsername string,
	isSuperAdmin bool,
) (*MMLCustomCommandPath, error) {
	if err := s.ensurePathRepo(); err != nil {
		return nil, err
	}
	cmd, err := s.customCommandRepo.GetByID(ctx, commandID)
	if err != nil {
		return nil, fmt.Errorf("get mml custom command: %w", err)
	}
	if !isOwnerOrSuper(cmd, currentUserID, currentUsername, isSuperAdmin) {
		return nil, fmt.Errorf("only creator or super_admin can modify command paths: %w", commonerrors.ErrForbidden)
	}
	return s.customCommandPathRepo.Update(ctx, commandID, pathID, defaultSelected, sortOrder)
}

// DeleteCustomCommandPath 删单条关联（owner/super 才能写）。
func (s *Service) DeleteCustomCommandPath(
	ctx context.Context,
	commandID, pathID uuid.UUID,
	currentUserID uuid.UUID,
	currentUsername string,
	isSuperAdmin bool,
) error {
	if err := s.ensurePathRepo(); err != nil {
		return err
	}
	cmd, err := s.customCommandRepo.GetByID(ctx, commandID)
	if err != nil {
		return fmt.Errorf("get mml custom command: %w", err)
	}
	if !isOwnerOrSuper(cmd, currentUserID, currentUsername, isSuperAdmin) {
		return fmt.Errorf("only creator or super_admin can modify command paths: %w", commonerrors.ErrForbidden)
	}
	return s.customCommandPathRepo.Delete(ctx, commandID, pathID)
}
