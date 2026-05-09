package indicator

import (
	"context"
	"encoding/csv"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

type IndicatorManagementService struct {
	groupRepo     GroupRepository
	indicatorRepo IndicatorRepository
	platformRepo  PlatformFormulaRepository
	enabledRepo   EnabledIndicatorRepository
	templateRel   TemplateRelRepository
	custNameRepo  CustNameRepository
	thresholdRepo IndicatorThresholdRepository
	pool          *pgxpool.Pool
	redis         redis.UniversalClient
	logger        *zap.Logger
}

func NewIndicatorManagementService(
	groupRepo GroupRepository,
	indicatorRepo IndicatorRepository,
	platformRepo PlatformFormulaRepository,
	enabledRepo EnabledIndicatorRepository,
	templateRel TemplateRelRepository,
	custNameRepo CustNameRepository,
	thresholdRepo IndicatorThresholdRepository,
	pool *pgxpool.Pool,
	rdb redis.UniversalClient,
	logger *zap.Logger,
) *IndicatorManagementService {
	return &IndicatorManagementService{
		groupRepo:     groupRepo,
		indicatorRepo: indicatorRepo,
		platformRepo:  platformRepo,
		enabledRepo:   enabledRepo,
		templateRel:   templateRel,
		custNameRepo:  custNameRepo,
		thresholdRepo: thresholdRepo,
		pool:          pool,
		redis:         rdb,
		logger:        logger,
	}
}

// ── Group Operations ──────────────────────────────────────────────────────────

func (s *IndicatorManagementService) GetGroupTree(ctx context.Context, req IndicatorGroupTreeRequest) ([]*IndicatorGroup, error) {
	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		return nil, fmt.Errorf("parse device type: %w", err)
	}

	groups, err := s.groupRepo.List(ctx, dt)
	if err != nil {
		return nil, err
	}

	counts, err := s.groupRepo.CountIndicatorsByGroup(ctx, dt)
	if err != nil {
		s.logger.Warn("failed to count indicators by group", zap.Error(err))
	}

	for _, g := range groups {
		g.IndicatorCount = int(counts[g.ID])
	}

	return buildTree(groups), nil
}

func (s *IndicatorManagementService) GetGroupByID(ctx context.Context, dt DeviceType, id string) (*IndicatorGroup, error) {
	return s.groupRepo.GetByID(ctx, dt, id)
}

func (s *IndicatorManagementService) GetGroupList(ctx context.Context, dt DeviceType) ([]*IndicatorGroup, error) {
	return s.groupRepo.List(ctx, dt)
}

func (s *IndicatorManagementService) CreateGroup(ctx context.Context, req *CreateGroupRequest) (*IndicatorGroup, error) {
	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		return nil, fmt.Errorf("parse device type: %w", err)
	}

	id := generateGroupID()
	group := &IndicatorGroup{
		ID:           id,
		EnName:       req.EnName,
		OperatorCode: strPtr(req.OperatorCode),
		IsBuildIn:    "0",
		Description:  nil,
		ParentID:     req.ParentID,
		CnName:       strPtr(req.CnName),
	}

	if err := s.groupRepo.Create(ctx, dt, group); err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return group, nil
}

func (s *IndicatorManagementService) UpdateGroup(ctx context.Context, dt DeviceType, id string, req *UpdateGroupRequest) error {
	return s.groupRepo.Update(ctx, dt, id, req)
}

func (s *IndicatorManagementService) DeleteGroup(ctx context.Context, dt DeviceType, id string) error {
	group, err := s.groupRepo.GetByID(ctx, dt, id)
	if err != nil {
		return err
	}
	if group.IsBuildIn == "1" {
		return fmt.Errorf("cannot delete built-in group")
	}

	indicatorIDs, err := s.indicatorRepo.GetIDsByGroupID(ctx, dt, id)
	if err != nil {
		return fmt.Errorf("get indicators in group: %w", err)
	}

	for _, indID := range indicatorIDs {
		inTpl, err := s.templateRel.ExistsByIndicatorID(ctx, indID)
		if err != nil {
			return fmt.Errorf("check template association for %s: %w", indID, err)
		}
		if inTpl {
			return fmt.Errorf("indicator %s is referenced by a template, cannot delete group", indID)
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.platformRepo.DeleteByIndicatorIDs(ctx, dt, indicatorIDs, tx); err != nil {
		return fmt.Errorf("delete formulas: %w", err)
	}
	if err := s.indicatorRepo.DeleteByGroupID(ctx, dt, id, tx); err != nil {
		return fmt.Errorf("delete indicators in group: %w", err)
	}
	if err := s.groupRepo.Delete(ctx, dt, id, tx); err != nil {
		return fmt.Errorf("delete group: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete group: %w", err)
	}
	return nil
}

// ── Indicator Operations ─────────────────────────────────────────────────────

func (s *IndicatorManagementService) ListIndicators(ctx context.Context, filter IndicatorListFilter) (*model.ListResponse[IndicatorListItem], error) {
	return s.indicatorRepo.List(ctx, filter)
}

func (s *IndicatorManagementService) GetIndicatorInfo(ctx context.Context, dt DeviceType, id string) (*PerfIndicator, error) {
	return s.indicatorRepo.GetByID(ctx, dt, id)
}

func (s *IndicatorManagementService) CreateIndicator(ctx context.Context, req *CreateIndicatorRequest) (*PerfIndicator, error) {
	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		return nil, fmt.Errorf("parse device type: %w", err)
	}

	operatorCode := req.OperatorCode
	if operatorCode == "" {
		operatorCode = "default"
	}

	var id string
	if req.IsCounter == "1" {
		id, err = s.indicatorRepo.GetNextCounterID(ctx)
	} else {
		id, err = s.indicatorRepo.GetNextKPIID(ctx, dt, operatorCode)
	}
	if err != nil {
		return nil, fmt.Errorf("generate indicator ID: %w", err)
	}

	if req.Arithmetic != "" {
		idMap, err := s.buildIDMap(ctx, dt)
		if err != nil {
			return nil, fmt.Errorf("build ID map for validation: %w", err)
		}
		validator := NewFormulaValidator(idMap)
		result := validator.Validate(req.Arithmetic)
		if !result.IsValid {
			return nil, fmt.Errorf("formula validation failed: %s", result.ErrorMsg)
		}
		if result.IsCounter {
			req.IsCounter = "1"
		}
	}

	indicator := &PerfIndicator{
		ID:                id,
		EnName:            req.EnName,
		CnName:            strPtr(req.CnName),
		EnDescription:     strPtr(req.EnDescription),
		CnDescription:     strPtr(req.CnDescription),
		GroupID:           req.GroupID,
		OperatorCode:      strPtr(operatorCode),
		DataType:          strPtr(req.DataType),
		UnitID:            strPtr(req.UnitID),
		Updator:           strPtr(req.Updator),
		IsBuildIn:         "0",
		IsCounter:         req.IsCounter,
		Arithmetic:        strPtr(req.Arithmetic),
		StatisType:        strPtr(req.StatisType),
		CalculatingStatus: strPtr("0"),
	}
	if dt.HasProductTypes() {
		indicator.ProductTypes = strPtr(req.ProductTypes)
		indicator.IndicatorLevel = strPtr(req.IndicatorLevel)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.indicatorRepo.Create(ctx, dt, indicator, tx); err != nil {
		return nil, fmt.Errorf("create indicator: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create indicator: %w", err)
	}

	s.refreshRedisCache(ctx, dt)
	return indicator, nil
}

func (s *IndicatorManagementService) UpdateIndicator(ctx context.Context, dt DeviceType, id string, req *UpdateIndicatorRequest) error {
	existing, err := s.indicatorRepo.GetByID(ctx, dt, id)
	if err != nil {
		return err
	}
	if existing.IsBuildIn == "1" {
		return fmt.Errorf("cannot modify built-in indicator")
	}

	if req.Arithmetic != nil && *req.Arithmetic != "" {
		idMap, err := s.buildIDMap(ctx, dt)
		if err != nil {
			return fmt.Errorf("build ID map for validation: %w", err)
		}
		validator := NewFormulaValidator(idMap)
		result := validator.Validate(*req.Arithmetic)
		if !result.IsValid {
			return fmt.Errorf("formula validation failed: %s", result.ErrorMsg)
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.indicatorRepo.Update(ctx, dt, id, req, tx); err != nil {
		return fmt.Errorf("update indicator: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update indicator: %w", err)
	}

	s.refreshRedisCache(ctx, dt)
	return nil
}

func (s *IndicatorManagementService) DeleteIndicator(ctx context.Context, dt DeviceType, id string) error {
	existing, err := s.indicatorRepo.GetByID(ctx, dt, id)
	if err != nil {
		return err
	}
	if existing.IsBuildIn == "1" {
		return fmt.Errorf("cannot delete built-in indicator")
	}

	inTpl, err := s.templateRel.ExistsByIndicatorID(ctx, id)
	if err != nil {
		return fmt.Errorf("check template association: %w", err)
	}
	if inTpl {
		return fmt.Errorf("indicator is referenced by a template")
	}

	inThreshold, err := s.thresholdRepo.ExistsByIndicatorID(ctx, id)
	if err != nil {
		s.logger.Warn("check threshold association", zap.Error(err))
	}
	if inThreshold {
		return fmt.Errorf("indicator is referenced by a threshold")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.platformRepo.DeleteByIndicatorID(ctx, dt, id, tx); err != nil {
		return fmt.Errorf("delete formulas: %w", err)
	}
	if err := s.indicatorRepo.Delete(ctx, dt, id, tx); err != nil {
		return fmt.Errorf("delete indicator: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete indicator: %w", err)
	}

	s.refreshRedisCache(ctx, dt)
	return nil
}

// ── Enable/Disable ────────────────────────────────────────────────────────────

func (s *IndicatorManagementService) EnableIndicators(ctx context.Context, req *EnableIndicatorsRequest) error {
	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		return fmt.Errorf("parse device type: %w", err)
	}
	return s.enabledRepo.BatchCreate(ctx, dt, req.OperatorCode, req.IndicatorIDs, nil)
}

func (s *IndicatorManagementService) DisableIndicators(ctx context.Context, req *EnableIndicatorsRequest) error {
	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		return fmt.Errorf("parse device type: %w", err)
	}
	return s.enabledRepo.BatchDelete(ctx, dt, req.OperatorCode, req.IndicatorIDs, nil)
}

func (s *IndicatorManagementService) GetEnabledIndicatorIDs(ctx context.Context, dt DeviceType, operatorCode string) ([]string, error) {
	return s.enabledRepo.List(ctx, dt, operatorCode)
}

// ── Template Check ────────────────────────────────────────────────────────────

func (s *IndicatorManagementService) IsIndicatorInTemplate(ctx context.Context, indicatorID string) (bool, error) {
	return s.templateRel.ExistsByIndicatorID(ctx, indicatorID)
}

// ── Export ────────────────────────────────────────────────────────────────────

func (s *IndicatorManagementService) ExportIndicators(ctx context.Context, req ExportRequest) ([]byte, error) {
	_, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		return nil, fmt.Errorf("parse device type: %w", err)
	}

	filter := IndicatorListFilter{
		DeviceType:   req.DeviceType,
		GroupID:      req.GroupID,
		OperatorCode: req.OperatorCode,
	}

	items, err := s.indicatorRepo.ListAll(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list indicators for export: %w", err)
	}

	var buf []byte
	buf = append(buf, []byte{0xEF, 0xBB, 0xBF}...) // UTF-8 BOM

	writer := csv.NewWriter(&byteWriter{buf: &buf})
	headers := []string{"ID", "English Name", "Chinese Name", "Group ID", "Is Counter", "Arithmetic", "Unit ID"}
	writer.Write(headers)

	for _, item := range items {
		cnName := ""
		if item.CnName != nil {
			cnName = *item.CnName
		}
		arithmetic := ""
		if item.Arithmetic != nil {
			arithmetic = *item.Arithmetic
		}
		unitID := ""
		if item.UnitID != nil {
			unitID = *item.UnitID
		}
		writer.Write([]string{item.ID, item.EnName, cnName, item.GroupID, item.IsCounter, arithmetic, unitID})
	}
	writer.Flush()

	return buf, nil
}

// ── Custom Name ───────────────────────────────────────────────────────────────

func (s *IndicatorManagementService) UpdateCustName(ctx context.Context, req *CustNameUpdateRequest) error {
	cn := &CustName{
		OperatorCode: req.OperatorCode,
		PerfID:       req.PerfID,
		CustName:     req.CustName,
	}
	return s.custNameRepo.Upsert(ctx, cn, nil)
}

func (s *IndicatorManagementService) UpdateCounterName(ctx context.Context, dt DeviceType, id string, newName string) error {
	req := &UpdateIndicatorRequest{
		CnName: &newName,
	}
	return s.indicatorRepo.Update(ctx, dt, id, req, nil)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (s *IndicatorManagementService) buildIDMap(ctx context.Context, dt DeviceType) (map[string]string, error) {
	groups, err := s.groupRepo.List(ctx, dt)
	if err != nil {
		return nil, err
	}

	var allIDs []string
	for _, g := range groups {
		ids, err := s.indicatorRepo.GetIDsByGroupID(ctx, dt, g.ID)
		if err != nil {
			return nil, err
		}
		allIDs = append(allIDs, ids...)
	}

	if len(allIDs) == 0 {
		return map[string]string{}, nil
	}

	indicators, err := s.indicatorRepo.ListByIDs(ctx, dt, allIDs)
	if err != nil {
		return nil, err
	}

	idMap := make(map[string]string, len(indicators))
	for _, ind := range indicators {
		arithmetic := ""
		if ind.Arithmetic != nil {
			arithmetic = *ind.Arithmetic
		}
		idMap[ind.ID] = arithmetic
	}
	return idMap, nil
}

// BumpCacheVersion 递增 indicator:cache_version（dictloader 标准协议 key），
// 触发其他实例 30s 轮询感知缓存失效。供 cache/refresh 端点 + 写路径调用。
func (s *IndicatorManagementService) BumpCacheVersion(ctx context.Context) {
	if s.redis == nil {
		return
	}
	s.redis.Incr(ctx, "indicator:cache_version")
}

// refreshRedisCache 写路径调用，等价于 BumpCacheVersion；保留旧签名（dt 参数当前不再用，
// 统一用单 key indicator:cache_version，与设计 §2.8 协议对齐）。
func (s *IndicatorManagementService) refreshRedisCache(ctx context.Context, dt DeviceType) {
	_ = dt
	s.BumpCacheVersion(ctx)
}

func buildTree(groups []*IndicatorGroup) []*IndicatorGroup {
	nodeMap := make(map[string]*IndicatorGroup, len(groups))
	for _, g := range groups {
		nodeMap[g.ID] = g
	}

	var roots []*IndicatorGroup
	for _, g := range groups {
		if g.ParentID == "" || g.ParentID == "0" || g.ParentID == g.ID {
			roots = append(roots, g)
		} else if parent, ok := nodeMap[g.ParentID]; ok {
			parent.Children = append(parent.Children, g)
		}
	}
	return roots
}

func generateGroupID() string {
	return uuid.New().String()
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// byteWriter implements io.Writer using a byte slice pointer.
type byteWriter struct {
	buf *[]byte
}

func (w *byteWriter) Write(p []byte) (int, error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}
