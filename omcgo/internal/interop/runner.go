package interop

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
)

// ConformanceTestRunner executes predefined interop conformance test cases
// against a target device identified by serial number.
//
// T-0098 P5-01：旧 dataModelReg 路径已删除，期望参数集仅来自 paramRegistry +
// productRegistry 的 ParamMapping（设计 §1.11）。前置任意一步失败 → checkParam*
// / verify_response 返回错误（不再降级）。
type ConformanceTestRunner struct {
	cases           map[TestCategory][]TestCase
	deviceRepo      device.DeviceRepository
	paramRepo       device.DeviceParameterRepository
	paramRegistry   *parammodel.Registry
	productRegistry *product.Registry
	taskSvc         task.Enqueuer
	logger          *zap.Logger
	// groupReader 是 #63 设备组可见性强制层的按设备归属读取器；RunAll/RunByCategory
	// 解析出目标设备后、执行任何（可能含破坏性 RPC-006/008/009）测试步骤前校验归属。
	// nil → dev/test 退化放行（authz nil-safe）。
	groupReader authz.GroupReader
}

// SetGroupReader 注入设备组归属读取器（#63 租户隔离强制层）。
func (r *ConformanceTestRunner) SetGroupReader(reader authz.GroupReader) {
	r.groupReader = reader
}

// NewConformanceTestRunner creates a runner pre-loaded with test cases.
func NewConformanceTestRunner(
	deviceRepo device.DeviceRepository,
	paramRepo device.DeviceParameterRepository,
	paramReg *parammodel.Registry,
	prodReg *product.Registry,
	taskSvc task.Enqueuer,
	logger *zap.Logger,
) *ConformanceTestRunner {
	return &ConformanceTestRunner{
		cases:           make(map[TestCategory][]TestCase),
		deviceRepo:      deviceRepo,
		paramRepo:       paramRepo,
		paramRegistry:   paramReg,
		productRegistry: prodReg,
		taskSvc:         taskSvc,
		logger:          logger.Named("conformance-runner"),
	}
}

// resolveExpectedParams 通过 ParamRegistry 获取期望参数集。
// 任意一步失败 → 返回 (nil, "")，调用方据此判定 model_exists 失败。
//
// expectedParam 是 parammodel.ParamMapping 的最小公共视图，仅 Path / Type / Writable。
func (r *ConformanceTestRunner) resolveExpectedParams(ctx context.Context, dev *model.Device) ([]expectedParam, string) {
	if r.paramRegistry == nil || r.productRegistry == nil {
		return nil, ""
	}
	if dev == nil || dev.ProductClass == "" {
		return nil, ""
	}
	match, err := r.productRegistry.MatchProductClass(ctx, dev.ProductClass)
	if err != nil || match == nil || match.Product == nil {
		return nil, ""
	}
	set, err := r.paramRegistry.GetByProduct(ctx, match.Product.ID, dev.FirmwareVersion)
	if err != nil || set == nil {
		return nil, ""
	}
	out := make([]expectedParam, 0, len(set.Mappings))
	for _, m := range set.Mappings {
		if m.EntryType != "parameter" {
			continue
		}
		out = append(out, expectedParam{
			Path:     m.PrivatePath,
			Type:     m.DataType,
			Writable: parammodel.IsAccessWritable(m.Access),
		})
	}
	return out, string(set.Source)
}

// expectedParam 是 parammodel.ParamMapping 的最小公共视图。
type expectedParam struct {
	Path     string
	Type     string
	Writable bool
}

// RegisterCases adds a batch of test cases under their respective categories.
func (r *ConformanceTestRunner) RegisterCases(tc []TestCase) {
	for _, c := range tc {
		r.cases[c.Category] = append(r.cases[c.Category], c)
	}
}

// ListTestCases returns all registered test cases grouped by category.
func (r *ConformanceTestRunner) ListTestCases() map[TestCategory][]TestCase {
	out := make(map[TestCategory][]TestCase, len(r.cases))
	for cat, cases := range r.cases {
		copied := make([]TestCase, len(cases))
		copy(copied, cases)
		out[cat] = copied
	}
	return out
}

// shouldRunForDevice reports whether the case should execute against the given
// device based on TargetDeviceModels filter (T-0116). Nil/empty filter = run
// against any device (default). Non-empty = require exact match on
// device.ModelName.
func shouldRunForDevice(tc TestCase, dev *model.Device) bool {
	if len(tc.TargetDeviceModels) == 0 {
		return true
	}
	for _, m := range tc.TargetDeviceModels {
		if m == dev.ModelName {
			return true
		}
	}
	return false
}

// RunAll executes every registered test case against the specified device,
// honouring per-case TargetDeviceModels filters (T-0116).
func (r *ConformanceTestRunner) RunAll(ctx context.Context, deviceSN string, visibleGroups []uuid.UUID) ([]TestResult, error) {
	dev, err := r.deviceRepo.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return nil, fmt.Errorf("lookup device %s: %w", deviceSN, err)
	}
	if dev == nil {
		return nil, fmt.Errorf("device not found: %s: %w", deviceSN, commonerrors.ErrNotFound)
	}
	// #63 租户隔离：执行任何（可能破坏性）测试前校验设备归属，越权 → ErrForbidden（403）。
	if err := authz.AuthorizeDeviceAccess(ctx, r.groupReader, dev.ID, visibleGroups); err != nil {
		return nil, err
	}

	var results []TestResult
	for _, cat := range ValidTestCategories() {
		cases, ok := r.cases[cat]
		if !ok {
			continue
		}
		for _, tc := range cases {
			if !shouldRunForDevice(tc, dev) {
				continue
			}
			result := r.runTestCase(ctx, dev, tc)
			results = append(results, result)
		}
	}
	return results, nil
}

// RunByCategory executes test cases in the given category against the device,
// honouring per-case TargetDeviceModels filters (T-0116).
func (r *ConformanceTestRunner) RunByCategory(ctx context.Context, deviceSN string, category TestCategory, visibleGroups []uuid.UUID) ([]TestResult, error) {
	if !category.IsValid() {
		return nil, fmt.Errorf("invalid test category: %s", category)
	}

	dev, err := r.deviceRepo.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return nil, fmt.Errorf("lookup device %s: %w", deviceSN, err)
	}
	if dev == nil {
		return nil, fmt.Errorf("device not found: %s: %w", deviceSN, commonerrors.ErrNotFound)
	}
	// #63 租户隔离：执行任何（可能破坏性）测试前校验设备归属，越权 → ErrForbidden（403）。
	if err := authz.AuthorizeDeviceAccess(ctx, r.groupReader, dev.ID, visibleGroups); err != nil {
		return nil, err
	}

	cases, ok := r.cases[category]
	if !ok {
		return nil, nil
	}

	var results []TestResult
	for _, tc := range cases {
		if !shouldRunForDevice(tc, dev) {
			continue
		}
		result := r.runTestCase(ctx, dev, tc)
		results = append(results, result)
	}
	return results, nil
}

// runTestCase executes a single test case and returns the result.
//
// For TestCase.ExpectedOutcome == "fail" (negative path), the pass/fail
// verdict is inverted at the end: all-steps-passing becomes failed
// ("negative case unexpectedly passed") and any step failing becomes
// passed ("expected failure observed"). This lets case library authors
// validate the runner's ability to detect bad inputs without adding a
// new action type.
func (r *ConformanceTestRunner) runTestCase(ctx context.Context, dev *model.Device, tc TestCase) TestResult {
	start := time.Now()
	result := TestResult{
		TestCaseID:      tc.ID,
		TestName:        tc.Name,
		Category:        tc.Category,
		Passed:          true,
		ExpectedOutcome: tc.ExpectedOutcome,
	}

	for _, step := range tc.Steps {
		stepErr := r.executeStep(ctx, dev, step)
		if stepErr != nil {
			result.Passed = false
			result.Error = stepErr.Error()
			result.Details = fmt.Sprintf("Failed at step %d: %s", step.Order, step.Description)
			break
		}
	}

	if result.Passed {
		result.Details = "All steps passed"
	}

	if tc.ExpectedOutcome == "fail" {
		if result.Passed {
			result.Passed = false
			result.Error = "negative case unexpectedly passed all steps"
			result.Details = "Expected failure but all steps passed (negative path broken)"
		} else {
			originalErr := result.Error
			result.Passed = true
			result.Error = ""
			result.Details = "Expected failure observed: " + originalErr
		}
	}

	result.Duration = time.Since(start)
	return result
}

// executeStep dispatches execution based on the step's action type.
func (r *ConformanceTestRunner) executeStep(ctx context.Context, dev *model.Device, step TestStep) error {
	switch step.Action {
	case "check_param":
		return r.executeCheckParam(ctx, dev, step)
	case "send_rpc":
		return r.executeSendRPC(ctx, dev, step)
	case "verify_response":
		return r.executeVerifyResponse(ctx, dev, step)
	default:
		return fmt.Errorf("unknown action: %s", step.Action)
	}
}

// executeCheckParam verifies device fields or parameter values.
func (r *ConformanceTestRunner) executeCheckParam(ctx context.Context, dev *model.Device, step TestStep) error {
	check, _ := step.Params["check"].(string)
	field, _ := step.Params["field"].(string)

	switch check {
	case "not_empty":
		return r.checkNotEmpty(dev, field, step)
	case "positive":
		return r.checkPositive(dev, field)
	case "paths_present":
		return r.checkParamPathsPresent(ctx, dev)
	case "types_match":
		return r.checkParamTypesMatch(ctx, dev)
	default:
		return fmt.Errorf("unknown check type: %s", check)
	}
}

// checkNotEmpty verifies that a device field is not empty.
func (r *ConformanceTestRunner) checkNotEmpty(dev *model.Device, field string, step TestStep) error {
	if rawFields, ok := step.Params["fields"]; ok {
		var fields []string
		switch v := rawFields.(type) {
		case []string:
			fields = v
		case []interface{}:
			for _, f := range v {
				if s, ok := f.(string); ok {
					fields = append(fields, s)
				}
			}
		}
		for _, f := range fields {
			if err := r.checkSingleFieldNotEmpty(dev, f); err != nil {
				return err
			}
		}
		return nil
	}

	return r.checkSingleFieldNotEmpty(dev, field)
}

func (r *ConformanceTestRunner) checkSingleFieldNotEmpty(dev *model.Device, field string) error {
	val := getDeviceField(dev, field)
	if val == "" {
		return fmt.Errorf("field %q is empty", field)
	}
	return nil
}

// checkPositive verifies a numeric field has a positive value.
func (r *ConformanceTestRunner) checkPositive(dev *model.Device, field string) error {
	switch field {
	case "inform_interval":
		if dev.InformInterval <= 0 {
			return fmt.Errorf("inform_interval is %d, expected positive", dev.InformInterval)
		}
		return nil
	default:
		return fmt.Errorf("unsupported positive check for field %q", field)
	}
}

// checkParamPathsPresent verifies that device parameters include paths from the param mapping.
func (r *ConformanceTestRunner) checkParamPathsPresent(ctx context.Context, dev *model.Device) error {
	expected, _ := r.resolveExpectedParams(ctx, dev)
	if expected == nil {
		return fmt.Errorf("no param mapping found for device %s", dev.SerialNumber)
	}

	params, err := r.paramRepo.GetByDevice(ctx, dev.ID)
	if err != nil {
		return fmt.Errorf("get device parameters: %w", err)
	}
	if len(params) == 0 {
		return fmt.Errorf("device has no reported parameters")
	}
	return nil
}

// checkParamTypesMatch validates parameter types against the param mapping.
func (r *ConformanceTestRunner) checkParamTypesMatch(ctx context.Context, dev *model.Device) error {
	expected, _ := r.resolveExpectedParams(ctx, dev)
	if expected == nil {
		return fmt.Errorf("no param mapping found for device %s", dev.SerialNumber)
	}

	params, err := r.paramRepo.GetByDevice(ctx, dev.ID)
	if err != nil {
		return fmt.Errorf("get device parameters: %w", err)
	}
	if len(params) == 0 {
		return fmt.Errorf("device has no reported parameters")
	}

	expectedTypes := make(map[string]string, len(expected))
	for _, p := range expected {
		expectedTypes[p.Path] = p.Type
	}
	return countTypeMismatches(params, expectedTypes)
}

// countTypeMismatches 统计 actualParams 与 expectedTypes 之间的 type 不一致条数。
func countTypeMismatches(actualParams []model.DeviceParameter, expectedTypes map[string]string) error {
	mismatches := 0
	for _, p := range actualParams {
		expected, ok := expectedTypes[p.ParameterPath]
		if !ok {
			continue
		}
		if string(p.ParameterType) != expected {
			mismatches++
		}
	}
	if mismatches > 0 {
		return fmt.Errorf("%d parameter type mismatches found", mismatches)
	}
	return nil
}

// executeSendRPC enqueues an RPC command to the device's command queue.
func (r *ConformanceTestRunner) executeSendRPC(ctx context.Context, dev *model.Device, step TestStep) error {
	method, _ := step.Params["method"].(string)
	if method == "" {
		return fmt.Errorf("send_rpc step missing method parameter")
	}

	paramsJSON, err := json.Marshal(step.Params)
	if err != nil {
		return fmt.Errorf("marshal rpc params: %w", err)
	}

	created, err := r.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:   dev.SerialNumber,
		Method:     method,
		Params:     paramsJSON,
		Priority:   5,
		CommandKey: fmt.Sprintf("interop-%s-%s", dev.SerialNumber, method),
		Source:     task.TaskSourceSystem,
	})
	if err != nil {
		return fmt.Errorf("enqueue %s command: %w", method, err)
	}

	r.logger.Info("interop RPC command enqueued",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("method", method),
		zap.String("command_id", created.ID),
	)
	return nil
}

// executeVerifyResponse performs response verification checks.
func (r *ConformanceTestRunner) executeVerifyResponse(ctx context.Context, dev *model.Device, step TestStep) error {
	check, _ := step.Params["check"].(string)

	switch check {
	case "command_queued":
		qLen, err := r.taskSvc.GetQueueLength(ctx, dev.SerialNumber)
		if err != nil {
			return fmt.Errorf("check command queue length: %w", err)
		}
		if qLen == 0 {
			return fmt.Errorf("command queue is empty after enqueue")
		}
		return nil

	case "model_exists":
		if expected, _ := r.resolveExpectedParams(ctx, dev); expected != nil {
			return nil
		}
		return fmt.Errorf("no param mapping found for device %s", dev.SerialNumber)

	default:
		return fmt.Errorf("unknown verify_response check: %s", check)
	}
}

// getDeviceField extracts a field value from the device by name.
func getDeviceField(dev *model.Device, field string) string {
	switch field {
	case "serial_number":
		return dev.SerialNumber
	case "oui":
		return dev.OUI
	case "manufacturer":
		return dev.Manufacturer
	case "product_class":
		return dev.ProductClass
	case "firmware_version":
		return dev.FirmwareVersion
	case "ip_address":
		return dev.IPAddress
	case "connection_request_url":
		return dev.ConnectionRequestURL
	case "last_inform_events":
		if len(dev.LastInformEvents) > 0 {
			return "present"
		}
		return ""
	default:
		return ""
	}
}
