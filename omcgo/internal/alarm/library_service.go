package alarm

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// LibraryService 告警库业务逻辑。
type LibraryService struct {
	repo   AlarmLibraryRepository
	logger *zap.Logger
}

// NewLibraryService 创建告警库服务。
func NewLibraryService(repo AlarmLibraryRepository, logger *zap.Logger) *LibraryService {
	return &LibraryService{repo: repo, logger: logger}
}

func (s *LibraryService) Create(ctx context.Context, req *CreateAlarmLibraryRequest) (*AlarmLibrary, error) {
	lib := &AlarmLibrary{
		ID:            uuid.New(),
		AlarmIdentifier:     req.AlarmIdentifier,
		AlarmSource:   req.AlarmSource,
		EventType:     req.EventType,
		Severity:      req.Severity,
		ProbableCause: req.ProbableCause,
		Explanation:   req.Explanation,
		AdditionalInfo: req.AdditionalInfo,
		Carrier:       strPtr(req.Carrier),
		Technology:    strPtr(req.Technology),
		Enabled:       true,
	}
	if req.Enabled != nil {
		lib.Enabled = *req.Enabled
	}
	if lib.AdditionalInfo == nil {
		lib.AdditionalInfo = make(map[string]any)
	}
	if err := s.repo.Create(ctx, lib); err != nil {
		return nil, fmt.Errorf("create alarm library: %w", err)
	}
	return lib, nil
}

func (s *LibraryService) GetByID(ctx context.Context, id uuid.UUID) (*AlarmLibrary, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *LibraryService) GetByCode(ctx context.Context, code string) (*AlarmLibrary, error) {
	return s.repo.GetByCode(ctx, code)
}

func (s *LibraryService) List(ctx context.Context, filter AlarmLibraryFilter) (*model.ListResponse[AlarmLibrary], error) {
	return s.repo.List(ctx, filter)
}

func (s *LibraryService) Update(ctx context.Context, id uuid.UUID, req *UpdateAlarmLibraryRequest) (*AlarmLibrary, error) {
	lib, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Severity != nil {
		lib.Severity = *req.Severity
	}
	if req.Enabled != nil {
		lib.Enabled = *req.Enabled
	}
	if req.ProbableCause != nil {
		lib.ProbableCause = *req.ProbableCause
	}
	if req.Explanation != nil {
		lib.Explanation = *req.Explanation
	}
	if req.AdditionalInfo != nil {
		lib.AdditionalInfo = req.AdditionalInfo
	}
	if req.Carrier != nil {
		lib.Carrier = strPtr(*req.Carrier)
	}
	if req.Technology != nil {
		lib.Technology = strPtr(*req.Technology)
	}
	if err := s.repo.Update(ctx, lib); err != nil {
		return nil, fmt.Errorf("update alarm library: %w", err)
	}
	return lib, nil
}

func (s *LibraryService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *LibraryService) CreateI18n(ctx context.Context, libraryID uuid.UUID, req *CreateAlarmLibraryI18nRequest) (*AlarmLibraryI18n, error) {
	i18n := &AlarmLibraryI18n{
		ID:            uuid.New(),
		LibraryID:     libraryID,
		Locale:        req.Locale,
		ProbableCause: req.ProbableCause,
		Explanation:   req.Explanation,
	}
	if err := s.repo.CreateI18n(ctx, i18n); err != nil {
		return nil, fmt.Errorf("create i18n: %w", err)
	}
	return i18n, nil
}

func (s *LibraryService) ListI18n(ctx context.Context, libraryID uuid.UUID) ([]AlarmLibraryI18n, error) {
	return s.repo.ListI18n(ctx, libraryID)
}

func (s *LibraryService) DeleteI18n(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteI18n(ctx, id)
}
