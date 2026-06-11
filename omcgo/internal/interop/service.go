package interop

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// CaseRunner abstracts the conformance test execution surface that the
// facade depends on. ConformanceTestRunner satisfies this interface.
//
// Defining this small interface at the consumer side (the facade) lets us
// inject mocks in unit tests without coupling to the concrete runner type.
type CaseRunner interface {
	ListTestCases() map[TestCategory][]TestCase
	RunAll(ctx context.Context, deviceSN string, visibleGroups []uuid.UUID) ([]TestResult, error)
	RunByCategory(ctx context.Context, deviceSN string, category TestCategory, visibleGroups []uuid.UUID) ([]TestResult, error)
}

// ModelValidator abstracts the data-model validation surface. DataModelValidator
// satisfies this interface.
type ModelValidator interface {
	ValidateDevice(ctx context.Context, deviceID uuid.UUID, carrier model.CarrierCode, tech model.Technology, visibleGroups []uuid.UUID) (*ValidationReport, error)
}

// InteropService is the high-level facade that the rest of the application
// (handler, gRPC, scripts) should depend on. It hides the split between the
// conformance runner, the data-model validator and the (future) report layer.
//
// Method semantics:
//   - ListCases returns all registered test cases grouped by category.
//   - RunCases executes test cases against a device. If categories is empty,
//     all categories are executed; otherwise only the listed categories.
//   - RunByCategory executes a single category (validates the category first).
//   - ValidateDevice produces a data-model conformance report.
//   - Summarize collapses a list of TestResult into a RunSummary.
// 所有按设备执行的方法都透传 visibleGroups（#63 设备组可见性强制层三态契约见 authz 包），
// 由底层 runner / validator 在解析设备后做归属校验。
type InteropService interface {
	ListCases(ctx context.Context) map[TestCategory][]TestCase
	RunCases(ctx context.Context, deviceSN string, categories []TestCategory, visibleGroups []uuid.UUID) (RunSummary, error)
	RunByCategory(ctx context.Context, deviceSN string, category TestCategory, visibleGroups []uuid.UUID) (RunSummary, error)
	ValidateDevice(ctx context.Context, deviceID uuid.UUID, carrier model.CarrierCode, tech model.Technology, visibleGroups []uuid.UUID) (*ValidationReport, error)
}

// RunSummary aggregates the outcome of a multi-case run.
type RunSummary struct {
	DeviceSN string       `json:"device_sn"`
	Total    int          `json:"total"`
	Passed   int          `json:"passed"`
	Failed   int          `json:"failed"`
	Results  []TestResult `json:"results"`
}

// service is the default InteropService implementation. It composes a
// CaseRunner with a ModelValidator and adds:
//   - input validation (categories must be valid before being run)
//   - aggregation (RunSummary)
//   - structured logging
//
// All actual work delegates to the underlying components, so this is a
// thin facade rather than a re-implementation.
type service struct {
	runner    CaseRunner
	validator ModelValidator
	logger    *zap.Logger
}

// NewService wires the facade with its dependencies.
func NewService(runner CaseRunner, validator ModelValidator, logger *zap.Logger) InteropService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		runner:    runner,
		validator: validator,
		logger:    logger.Named("interop-service"),
	}
}

// ListCases returns a copy of all registered test cases grouped by category.
func (s *service) ListCases(_ context.Context) map[TestCategory][]TestCase {
	if s.runner == nil {
		return map[TestCategory][]TestCase{}
	}
	return s.runner.ListTestCases()
}

// RunCases executes the requested categories (or all when empty) and returns
// an aggregated summary.
func (s *service) RunCases(ctx context.Context, deviceSN string, categories []TestCategory, visibleGroups []uuid.UUID) (RunSummary, error) {
	if deviceSN == "" {
		return RunSummary{}, fmt.Errorf("interop: device_sn is required")
	}
	if s.runner == nil {
		return RunSummary{}, fmt.Errorf("interop: case runner not configured")
	}

	var results []TestResult
	if len(categories) == 0 {
		all, err := s.runner.RunAll(ctx, deviceSN, visibleGroups)
		if err != nil {
			return RunSummary{}, fmt.Errorf("run all interop cases: %w", err)
		}
		results = all
	} else {
		// Validate up front so a bad category short-circuits before any device call.
		for _, cat := range categories {
			if !cat.IsValid() {
				return RunSummary{}, fmt.Errorf("interop: invalid test category %q", cat)
			}
		}
		for _, cat := range categories {
			catResults, err := s.runner.RunByCategory(ctx, deviceSN, cat, visibleGroups)
			if err != nil {
				return RunSummary{}, fmt.Errorf("run interop category %s: %w", cat, err)
			}
			results = append(results, catResults...)
		}
	}

	summary := summarize(deviceSN, results)
	s.logger.Info("interop run complete",
		zap.String("device_sn", deviceSN),
		zap.Int("total", summary.Total),
		zap.Int("passed", summary.Passed),
		zap.Int("failed", summary.Failed),
	)
	return summary, nil
}

// RunByCategory executes a single category against a device.
func (s *service) RunByCategory(ctx context.Context, deviceSN string, category TestCategory, visibleGroups []uuid.UUID) (RunSummary, error) {
	if deviceSN == "" {
		return RunSummary{}, fmt.Errorf("interop: device_sn is required")
	}
	if !category.IsValid() {
		return RunSummary{}, fmt.Errorf("interop: invalid test category %q", category)
	}
	if s.runner == nil {
		return RunSummary{}, fmt.Errorf("interop: case runner not configured")
	}

	results, err := s.runner.RunByCategory(ctx, deviceSN, category, visibleGroups)
	if err != nil {
		return RunSummary{}, fmt.Errorf("run interop category %s: %w", category, err)
	}
	return summarize(deviceSN, results), nil
}

// ValidateDevice forwards to the underlying ModelValidator.
func (s *service) ValidateDevice(ctx context.Context, deviceID uuid.UUID, carrier model.CarrierCode, tech model.Technology, visibleGroups []uuid.UUID) (*ValidationReport, error) {
	if s.validator == nil {
		return nil, fmt.Errorf("interop: model validator not configured")
	}
	if !carrier.IsValid() {
		return nil, fmt.Errorf("interop: invalid carrier %q", carrier)
	}
	if !tech.IsValid() {
		return nil, fmt.Errorf("interop: invalid technology %q", tech)
	}
	report, err := s.validator.ValidateDevice(ctx, deviceID, carrier, tech, visibleGroups)
	if err != nil {
		return nil, fmt.Errorf("validate device %s: %w", deviceID, err)
	}
	return report, nil
}

// summarize collapses a result slice into a RunSummary. Pure function so it
// can be reused (and tested) without a service instance.
func summarize(deviceSN string, results []TestResult) RunSummary {
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}
	return RunSummary{
		DeviceSN: deviceSN,
		Total:    len(results),
		Passed:   passed,
		Failed:   len(results) - passed,
		Results:  results,
	}
}
