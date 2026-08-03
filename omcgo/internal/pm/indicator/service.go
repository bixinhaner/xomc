package indicator

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

type IndicatorManagementService struct {
	groupRepo        GroupRepository
	indicatorRepo    IndicatorRepository
	platformRepo     PlatformFormulaRepository
	enabledRepo      EnabledIndicatorRepository
	templateRel      TemplateRelRepository
	custNameRepo     CustNameRepository
	thresholdRepo    IndicatorThresholdRepository
	dashboardLayout  DashboardLayoutReferenceChecker
	pool             transactionBeginner
	redis            redis.UniversalClient
	routeInvalidator RouteInvalidator
	logger           *zap.Logger
}

type transactionBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type enabledDependencyReconcileRepository interface {
	EnabledIndicatorRepository
	ListOperatorCodes(context.Context, DeviceType) ([]string, error)
	ListTx(context.Context, DeviceType, string, pgx.Tx) ([]string, error)
}

// DashboardLayoutReferenceChecker checks whether dashboard KPI layouts reference PM indicators.
type DashboardLayoutReferenceChecker interface {
	ReferencedIndicators(ctx context.Context, dt DeviceType, indicatorIDs []string) ([]string, error)
}

// RouteInvalidationTrigger 是 indicator 包暴露给 provider 的低基数失效来源。
// 这里不直接依赖 pm/kpi/router，避免 router -> indicator 的反向 import 环。
type RouteInvalidationTrigger string

const (
	RouteInvalidationTriggerIndicatorWrite RouteInvalidationTrigger = "indicator_write"
	RouteInvalidationTriggerFormulaWrite   RouteInvalidationTrigger = "platform_formula_write"
	RouteInvalidationTriggerGroupDelete    RouteInvalidationTrigger = "indicator_group_delete"
)

// RouteInvalidator 在业务事务提交后失效 KPI Route。
// 调用方必须把失败视为 best-effort：业务写入已提交，不因 route bump 失败回滚。
type RouteInvalidator func(context.Context, RouteInvalidationTrigger) error

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

func (s *IndicatorManagementService) WithDashboardLayoutReferenceChecker(checker DashboardLayoutReferenceChecker) *IndicatorManagementService {
	s.dashboardLayout = checker
	return s
}

func (s *IndicatorManagementService) WithRouteInvalidator(invalidator RouteInvalidator) *IndicatorManagementService {
	s.routeInvalidator = invalidator
	return s
}

// ── Group Operations ──────────────────────────────────────────────────────────

func (s *IndicatorManagementService) GetGroupTree(ctx context.Context, req IndicatorGroupTreeRequest) ([]*IndicatorGroup, error) {
	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		return nil, fmt.Errorf("parse device type: %w", err)
	}

	// platform 非空时,只返有该 platform 公式关联的指标所属分组(前端下拉过滤用)
	groups, err := s.groupRepo.ListByPlatform(ctx, dt, req.Platform)
	if err != nil {
		return nil, err
	}

	counts, err := s.groupRepo.CountIndicatorsByGroup(ctx, dt, req.Platform)
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
		// 空值/非法 device_type → 400（而非裸 error 落 500）。
		// REST/GNB 入口由 query/路由注入；indicatormg 入口靠 body 传入，此处统一兜底。
		return nil, fmt.Errorf("parse device type: %w: %v", commonerrors.ErrInvalidInput, err)
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
	if len(indicatorIDs) > 0 {
		s.refreshRedisCache(ctx, dt)
		s.invalidateRouteCache(ctx, RouteInvalidationTriggerGroupDelete)
	}
	return nil
}

// ── Indicator Operations ─────────────────────────────────────────────────────

func (s *IndicatorManagementService) ListIndicators(ctx context.Context, filter IndicatorListFilter) (*model.ListResponse[IndicatorListItem], error) {
	return s.indicatorRepo.List(ctx, filter)
}

// ListPlatformNames 返回指定 deviceType 在公式表里出现过的全部平台名（去重并按字典序排序）。
func (s *IndicatorManagementService) ListPlatformNames(ctx context.Context, dt DeviceType) ([]string, error) {
	return s.platformRepo.ListPlatformNames(ctx, dt)
}

func (s *IndicatorManagementService) GetIndicatorInfo(ctx context.Context, dt DeviceType, id string) (*PerfIndicator, error) {
	return s.indicatorRepo.GetByID(ctx, dt, id)
}

func (s *IndicatorManagementService) CreateIndicator(ctx context.Context, req *CreateIndicatorRequest) (*PerfIndicator, error) {
	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		return nil, fmt.Errorf("parse device type: %w", err)
	}
	req.Arithmetic = normalizeDurationArithmetic(dt, req.Arithmetic)

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

	// 公式批量分支（issue #640 C 方案）：Formulas 非空时同事务批量写入，校验失败整体回滚。
	// 与单条 Platform 占位分支二选一 —— Formulas 优先（与 model.go FormulaInput 注释一致）。
	if len(req.Formulas) > 0 {
		// 同 platform_name 重复校验（前端阻断后兜底；DB 无唯一约束，必须在 service 层挡住）。
		seen := make(map[string]struct{}, len(req.Formulas))
		for _, f := range req.Formulas {
			if _, dup := seen[f.PlatformName]; dup {
				return nil, fmt.Errorf("duplicate platform in formulas: %s", f.PlatformName)
			}
			seen[f.PlatformName] = struct{}{}
		}
		// 单条语法校验：每条都跑 FormulaValidator，任一失败整体回滚。
		// validator 复用 buildIDMap 的结果（前面 Arithmetic 校验已建好，但当时可能为空跳过），
		// 这里无条件重建以兼容 Arithmetic 为空但 Formulas 非空的场景。
		idMap, err := s.buildIDMap(ctx, dt)
		if err != nil {
			return nil, fmt.Errorf("build ID map for formulas validation: %w", err)
		}
		validator := NewFormulaValidator(idMap)
		batch := make([]*PlatformFormula, 0, len(req.Formulas))
		for _, f := range req.Formulas {
			result := validator.Validate(f.Formula)
			if !result.IsValid {
				return nil, fmt.Errorf("formula validation failed (platform=%s): %s", f.PlatformName, result.ErrorMsg)
			}
			batch = append(batch, &PlatformFormula{
				PlatformName: f.PlatformName,
				IndicatorID:  id,
				Formula:      f.Formula,
				ReportKey:    indicator.ReportKey,
			})
		}
		if err := s.platformRepo.BatchCreate(ctx, dt, batch, tx); err != nil {
			return nil, fmt.Errorf("batch create platform formulas: %w", err)
		}
	} else if req.Platform != "" {
		// 旧前端兼容路径：Platform 非空时同事务内向 perf_formulas_<dt> 写一行占位
		// (Formula=Arithmetic,允许空),否则详情态(?platform=)的 EXISTS 过滤会把新建的指标过滤掉,
		// 看起来"保存成功但查不到"。列表态(主页全局新建,未锁 platform)两个字段都空 → 不写公式。
		formula := &PlatformFormula{
			PlatformName: req.Platform,
			IndicatorID:  id,
			Formula:      req.Arithmetic,
			ReportKey:    indicator.ReportKey,
		}
		if err := s.platformRepo.BatchCreate(ctx, dt, []*PlatformFormula{formula}, tx); err != nil {
			return nil, fmt.Errorf("create platform formula placeholder: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create indicator: %w", err)
	}

	s.refreshRedisCache(ctx, dt)
	s.invalidateRouteCache(ctx, RouteInvalidationTriggerIndicatorWrite)
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
	if req.Arithmetic != nil {
		normalized := normalizeDurationArithmetic(dt, *req.Arithmetic)
		req.Arithmetic = &normalized
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
	if updateAffectsRoute(req) {
		s.invalidateRouteCache(ctx, RouteInvalidationTriggerIndicatorWrite)
	}
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
	s.invalidateRouteCache(ctx, RouteInvalidationTriggerIndicatorWrite)
	return nil
}

func (s *IndicatorManagementService) UpsertPlatformFormula(ctx context.Context, dt DeviceType, indicatorID, platform, formula string) (*PlatformFormula, error) {
	platform, formula, err := normalizePlatformFormulaInput(platform, formula)
	if err != nil {
		return nil, err
	}
	indicatorRecord, err := s.indicatorRepo.GetByID(ctx, dt, indicatorID)
	if err != nil {
		return nil, err
	}
	idMap, err := s.buildIDMap(ctx, dt)
	if err != nil {
		return nil, fmt.Errorf("build ID map for formula validation: %w", err)
	}
	validator := NewFormulaValidator(idMap)
	result := validator.Validate(formula)
	if !result.IsValid {
		return nil, fmt.Errorf("%w: formula validation failed (platform=%s): %s", commonerrors.ErrInvalidInput, platform, result.ErrorMsg)
	}
	existing, err := s.platformRepo.ListByIndicatorID(ctx, dt, indicatorID)
	if err != nil {
		return nil, fmt.Errorf("list existing platform formulas: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := s.platformRepo.DeleteByIndicatorAndPlatform(ctx, dt, indicatorID, platform, tx); err != nil {
		return nil, fmt.Errorf("delete existing platform formula: %w", err)
	}
	out := &PlatformFormula{
		PlatformName: platform,
		IndicatorID:  indicatorID,
		Formula:      formula,
	}
	if indicatorRecord != nil {
		out.ReportKey = indicatorRecord.ReportKey
	}
	for _, item := range existing {
		if item.PlatformName == platform && item.ReportKey != nil {
			out.ReportKey = item.ReportKey
			break
		}
	}
	if err := s.platformRepo.BatchCreate(ctx, dt, []*PlatformFormula{out}, tx); err != nil {
		return nil, fmt.Errorf("create platform formula: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit upsert platform formula: %w", err)
	}
	s.refreshRedisCache(ctx, dt)
	s.invalidateRouteCache(ctx, RouteInvalidationTriggerFormulaWrite)
	return out, nil
}

func (s *IndicatorManagementService) DeletePlatformFormula(ctx context.Context, dt DeviceType, indicatorID, platform string) error {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return fmt.Errorf("%w: platform is required", commonerrors.ErrInvalidInput)
	}
	if _, err := s.indicatorRepo.GetByID(ctx, dt, indicatorID); err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	deleted, err := s.platformRepo.DeleteByIndicatorAndPlatform(ctx, dt, indicatorID, platform, tx)
	if err != nil {
		return fmt.Errorf("delete platform formula: %w", err)
	}
	if deleted == 0 {
		return commonerrors.ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete platform formula: %w", err)
	}
	s.refreshRedisCache(ctx, dt)
	s.invalidateRouteCache(ctx, RouteInvalidationTriggerFormulaWrite)
	return nil
}

// ── Enable/Disable ────────────────────────────────────────────────────────────

var (
	ErrIndicatorUsedByDashboardLayout = fmt.Errorf("%w: indicator is used by dashboard KPI layout", commonerrors.ErrInvalidInput)
	ErrEnabledKPIDependency           = fmt.Errorf("%w: indicator is required by an enabled KPI", commonerrors.ErrInvalidInput)
)

func (s *IndicatorManagementService) EnableIndicators(ctx context.Context, req *EnableIndicatorsRequest) error {
	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		return fmt.Errorf("parse device type: %w", err)
	}
	arithmetic, counters, err := s.indicatorDependencyMetadata(ctx, dt)
	if err != nil {
		return fmt.Errorf("load indicator dependencies: %w", err)
	}
	closure, err := ResolveDependencyClosure(req.IndicatorIDs, arithmetic, counters)
	if err != nil {
		return fmt.Errorf("resolve indicator dependency closure: %w", err)
	}
	indicatorIDs := sortedUnique(append(append([]string(nil), closure.KPIs...), closure.Counters...))

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin enable indicators: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := lockEnabledIndicatorSet(ctx, tx, dt, req.OperatorCode); err != nil {
		return err
	}
	if err := s.enabledRepo.BatchCreate(ctx, dt, req.OperatorCode, indicatorIDs, tx); err != nil {
		return fmt.Errorf("enable indicator dependency closure: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit enable indicators: %w", err)
	}
	s.refreshRedisCache(ctx, dt)
	s.invalidateRouteCache(ctx, RouteInvalidationTriggerIndicatorWrite)
	return nil
}

// ReconcileEnabledDependencies repairs persisted enabled KPI sets created by
// old releases or baseline data. API writes already maintain this invariant;
// startup reconciliation makes the same invariant true before PM collection
// begins, preventing required Counters from being filtered out permanently.
func (s *IndicatorManagementService) ReconcileEnabledDependencies(ctx context.Context) error {
	repo, ok := s.enabledRepo.(enabledDependencyReconcileRepository)
	if !ok {
		return fmt.Errorf("enabled indicator repository does not support dependency reconciliation")
	}
	for _, dt := range []DeviceType{DeviceTypeENB, DeviceTypeGNB, DeviceTypeGSM} {
		operators, err := repo.ListOperatorCodes(ctx, dt)
		if err != nil {
			return fmt.Errorf("list %s enabled indicator scopes: %w", dt, err)
		}
		if len(operators) == 0 {
			continue
		}
		arithmetic, counters, err := s.indicatorDependencyMetadata(ctx, dt)
		if err != nil {
			return fmt.Errorf("load %s indicator dependencies: %w", dt, err)
		}
		for _, operatorCode := range operators {
			tx, err := s.pool.Begin(ctx)
			if err != nil {
				return fmt.Errorf("begin %s dependency reconciliation for %s: %w", dt, operatorCode, err)
			}
			if err := lockEnabledIndicatorSet(ctx, tx, dt, operatorCode); err != nil {
				tx.Rollback(ctx)
				return err
			}
			enabledIDs, err := repo.ListTx(ctx, dt, operatorCode, tx)
			if err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("list %s enabled indicators for %s: %w", dt, operatorCode, err)
			}
			closure, err := ResolveDependencyClosure(enabledIDs, arithmetic, counters)
			if err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("resolve %s enabled dependency closure for %s: %w", dt, operatorCode, err)
			}
			indicatorIDs := sortedUnique(append(append([]string(nil), closure.KPIs...), closure.Counters...))
			if err := repo.BatchCreate(ctx, dt, operatorCode, indicatorIDs, tx); err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("repair %s enabled dependency closure for %s: %w", dt, operatorCode, err)
			}
			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("commit %s enabled dependency closure for %s: %w", dt, operatorCode, err)
			}
		}
		s.refreshRedisCache(ctx, dt)
	}
	s.invalidateRouteCache(ctx, RouteInvalidationTriggerIndicatorWrite)
	return nil
}

func (s *IndicatorManagementService) DisableIndicators(ctx context.Context, req *EnableIndicatorsRequest) error {
	dt, err := ParseDeviceType(req.DeviceType)
	if err != nil {
		return fmt.Errorf("parse device type: %w", err)
	}
	if err := s.ensureNotReferencedByDashboardLayout(ctx, dt, req.IndicatorIDs); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin disable indicators: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := lockEnabledIndicatorSet(ctx, tx, dt, req.OperatorCode); err != nil {
		return err
	}
	if err := s.ensureNotRequiredByEnabledKPI(ctx, dt, req.OperatorCode, req.IndicatorIDs, tx); err != nil {
		return err
	}
	if err := s.enabledRepo.BatchDelete(ctx, dt, req.OperatorCode, req.IndicatorIDs, tx); err != nil {
		return fmt.Errorf("disable indicators: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit disable indicators: %w", err)
	}
	s.refreshRedisCache(ctx, dt)
	s.invalidateRouteCache(ctx, RouteInvalidationTriggerIndicatorWrite)
	return nil
}

func (s *IndicatorManagementService) GetEnabledIndicatorIDs(ctx context.Context, dt DeviceType, operatorCode string) ([]string, error) {
	return s.enabledRepo.List(ctx, dt, operatorCode)
}

// ── Template Check ────────────────────────────────────────────────────────────

func (s *IndicatorManagementService) IsIndicatorInTemplate(ctx context.Context, indicatorID string) (bool, error) {
	return s.templateRel.ExistsByIndicatorID(ctx, indicatorID)
}

func (s *IndicatorManagementService) ensureNotReferencedByDashboardLayout(ctx context.Context, dt DeviceType, indicatorIDs []string) error {
	if s.dashboardLayout == nil || len(indicatorIDs) == 0 {
		return nil
	}
	referenced, err := s.dashboardLayout.ReferencedIndicators(ctx, dt, indicatorIDs)
	if err != nil {
		return fmt.Errorf("check dashboard KPI layout references: %w", err)
	}
	if len(referenced) > 0 {
		return fmt.Errorf("%w: %v", ErrIndicatorUsedByDashboardLayout, referenced)
	}
	return nil
}

func (s *IndicatorManagementService) indicatorDependencyMetadata(
	ctx context.Context,
	dt DeviceType,
) (map[string]string, map[string]bool, error) {
	items, err := s.indicatorRepo.ListAll(ctx, IndicatorListFilter{DeviceType: string(dt)})
	if err != nil {
		return nil, nil, err
	}
	arithmetic := make(map[string]string, len(items))
	counters := make(map[string]bool, len(items))
	for _, item := range items {
		if item.Arithmetic != nil {
			arithmetic[item.ID] = *item.Arithmetic
		}
		counters[item.ID] = item.IsCounter == "1"
	}
	return arithmetic, counters, nil
}

func (s *IndicatorManagementService) ensureNotRequiredByEnabledKPI(
	ctx context.Context,
	dt DeviceType,
	operatorCode string,
	indicatorIDs []string,
	tx pgx.Tx,
) error {
	var enabledIDs []string
	var err error
	if repo, ok := s.enabledRepo.(interface {
		ListTx(context.Context, DeviceType, string, pgx.Tx) ([]string, error)
	}); ok {
		enabledIDs, err = repo.ListTx(ctx, dt, operatorCode, tx)
	} else {
		enabledIDs, err = s.enabledRepo.List(ctx, dt, operatorCode)
	}
	if err != nil {
		return fmt.Errorf("list enabled indicators: %w", err)
	}
	arithmetic, counters, err := s.indicatorDependencyMetadata(ctx, dt)
	if err != nil {
		return fmt.Errorf("load indicator dependencies: %w", err)
	}
	disabling := make(map[string]struct{}, len(indicatorIDs))
	for _, id := range indicatorIDs {
		disabling[id] = struct{}{}
	}
	for _, enabledID := range enabledIDs {
		if counters[enabledID] {
			continue
		}
		if _, beingDisabled := disabling[enabledID]; beingDisabled {
			continue
		}
		closure, err := ResolveDependencyClosure([]string{enabledID}, arithmetic, counters)
		if err != nil {
			return fmt.Errorf("resolve enabled KPI %s dependencies: %w", enabledID, err)
		}
		dependencies := append(append([]string(nil), closure.Counters...), closure.KPIs...)
		for _, dependencyID := range dependencies {
			if dependencyID == enabledID {
				continue
			}
			if _, conflict := disabling[dependencyID]; conflict {
				return fmt.Errorf("%w: %s depends on %s", ErrEnabledKPIDependency, enabledID, dependencyID)
			}
		}
	}
	return nil
}

func lockEnabledIndicatorSet(
	ctx context.Context,
	tx pgx.Tx,
	dt DeviceType,
	operatorCode string,
) error {
	lockKey := dt.EnabledTable() + "\x1f" + operatorCode
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", lockKey); err != nil {
		return fmt.Errorf("lock enabled indicator set: %w", err)
	}
	return nil
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

// buildIDMap builds the ID→formula map used by FormulaValidator to recognise
// which tokens in a user formula are legal indicator/counter references.
//
// 历史 bug（#134）：原实现先 groupRepo.List 取分组再 GetIDsByGroupID 收集指标。
// 但 indicator Loader 设计上只建 default 占位组、指标的 group_id 直接取 XML
// groupId 从不回写组表 → 全新栈上组表恒空 → idMap 恒空 → 校验器误拒一切引用
// 真实计数器（如 C000200015）的合法公式（400 Expression is invalid）。
//
// 修复：直接 ListAll 拉该 deviceType 表（perf_indicators_<enb/gnb/gsm>）下全部
// 指标构建 idMap，绕开组表。平台维度由 dt.IndicatorTable() 物理分表保证正确：
// ENB 公式只对照 ENB 指标表、GNB 只对照 GNB 表。不按 operator_code 过滤，确保
// 任意运营商写入的自定义 KPI（K90000* 递归展开依赖）都在 map 里可见。
func (s *IndicatorManagementService) buildIDMap(ctx context.Context, dt DeviceType) (map[string]string, error) {
	items, err := s.indicatorRepo.ListAll(ctx, IndicatorListFilter{DeviceType: string(dt)})
	if err != nil {
		return nil, err
	}

	idMap := make(map[string]string, len(items))
	for _, ind := range items {
		arithmetic := ""
		if ind.Arithmetic != nil {
			arithmetic = *ind.Arithmetic
		}
		idMap[ind.ID] = arithmetic
	}
	return idMap, nil
}

// BumpCacheVersion 递增 indicator:cache_version（dictloader 标准协议 key），
// 触发其他实例 30s 轮询感知缓存失效。供 upload-xml 端点(重载后刷新)+ 写路径调用。
func (s *IndicatorManagementService) BumpCacheVersion(ctx context.Context) {
	if s.redis == nil {
		return
	}
	s.redis.Incr(ctx, "indicator:cache_version")
}

func (s *IndicatorManagementService) InvalidateRouteCache(ctx context.Context, trigger RouteInvalidationTrigger) {
	s.invalidateRouteCache(ctx, trigger)
}

func (s *IndicatorManagementService) invalidateRouteCache(ctx context.Context, trigger RouteInvalidationTrigger) {
	if s.routeInvalidator == nil {
		return
	}
	if err := s.routeInvalidator(ctx, trigger); err != nil {
		if s.logger == nil {
			return
		}
		s.logger.Warn("business write committed but KPI route invalidation failed; manual refresh can recover",
			zap.String("trigger", string(trigger)),
			zap.Error(err))
	}
}

func updateAffectsRoute(req *UpdateIndicatorRequest) bool {
	return req.EnName != nil ||
		req.UnitID != nil ||
		req.Arithmetic != nil ||
		req.StatisType != nil
}

// normalizeDurationArithmetic 把页面公式编辑器的友好关键字 Duration 转成当前制式
// 已登记的「统计时长」合成 Counter 编号。KPI 路由/表达式引擎只消费编号公式；若把
// Duration 原样落库，它会成为永远缺失的依赖，最终 KPI 无值（#27）。只替换完整 token，
// 避免误改 DurationValue/MyDuration 等合法标识符。
func normalizeDurationArithmetic(dt DeviceType, arithmetic string) string {
	return CompileRuntimeArithmetic(dt, arithmetic)
}

func normalizePlatformFormulaInput(platform, formula string) (string, string, error) {
	platform = strings.TrimSpace(platform)
	formula = strings.TrimSpace(formula)
	switch {
	case platform == "":
		return "", "", fmt.Errorf("%w: platform is required", commonerrors.ErrInvalidInput)
	case formula == "":
		return "", "", fmt.Errorf("%w: formula is required", commonerrors.ErrInvalidInput)
	default:
		return platform, formula, nil
	}
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
		} else {
			// 2026-05-29:父节点缺失 → 作为根节点回退,避免悄悄吞掉数据。
			// 历史 bug:seed/000062 把 22+8+15 个 ENB/GSM/GNB 分组的 parent_id
			// 都设成 'e1e2466f156f44cfa116985008f2f298' 这一硬编码 root,但该
			// root 自身未 seeded → 原 buildTree 把所有 45 个分组全 strip,
			// /indicator-groups 端点返 [],前端"按分组筛选"下拉始终空。
			// fail-open 后,孤儿分组仍可作根节点露出,UI 至少能看见 HO/EQPT/...
			roots = append(roots, g)
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
