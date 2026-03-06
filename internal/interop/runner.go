package interop

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/omcgo/omcgo/internal/config/datamodel"
	"github.com/omcgo/omcgo/internal/omcr/device"
)

// ConformanceTestRunner executes predefined interop conformance test cases
// against a target device identified by serial number.
type ConformanceTestRunner struct {
	cases        map[TestCategory][]TestCase
	deviceRepo   device.DeviceRepository
	paramRepo    device.DeviceParameterRepository
	dataModelReg *datamodel.DataModelRegistry
	cmdQueue     cmdqueue.CommandQueue
	logger       *zap.Logger
}

// NewConformanceTestRunner creates a runner pre-loaded with test cases.
func NewConformanceTestRunner(
	deviceRepo device.DeviceRepository,
	paramRepo device.DeviceParameterRepository,
	dataModelReg *datamodel.DataModelRegistry,
	cmdQueue cmdqueue.CommandQueue,
	logger *zap.Logger,
) *ConformanceTestRunner {
	return &ConformanceTestRunner{
		cases:        make(map[TestCategory][]TestCase),
		deviceRepo:   deviceRepo,
		paramRepo:    paramRepo,
		dataModelReg: dataModelReg,
		cmdQueue:     cmdQueue,
		logger:       logger.Named("conformance-runner"),
	}
}

// RegisterCases adds a batch of test cases under their respective categories.
func (r *ConformanceTestRunner) RegisterCases(tc []TestCase) {
	for _, c := range tc {
		r.cases[c.Category] = append(r.cases[c.Category], c)
	}
}

// ListTestCases returns all registered test cases grouped by category.
func (r *ConformanceTestRunner) ListTestCases() map[TestCategory][]TestCase {
	// Return a copy to prevent external mutation.
	out := make(map[TestCategory][]TestCase, len(r.cases))
	for cat, cases := range r.cases {
		copied := make([]TestCase, len(cases))
		copy(copied, cases)
		out[cat] = copied
	}
	return out
}

// RunAll executes every registered test case against the specified device.
func (r *ConformanceTestRunner) RunAll(ctx context.Context, deviceSN string) ([]TestResult, error) {
	dev, err := r.deviceRepo.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return nil, fmt.Errorf("lookup device %s: %w", deviceSN, err)
	}

	var results []TestResult
	for _, cat := range ValidTestCategories() {
		cases, ok := r.cases[cat]
		if !ok {
			continue
		}
		for _, tc := range cases {
			result := r.runTestCase(ctx, dev, tc)
			results = append(results, result)
		}
	}
	return results, nil
}

// RunByCategory executes test cases in the given category against the device.
func (r *ConformanceTestRunner) RunByCategory(ctx context.Context, deviceSN string, category TestCategory) ([]TestResult, error) {
	if !category.IsValid() {
		return nil, fmt.Errorf("invalid test category: %s", category)
	}

	dev, err := r.deviceRepo.GetBySerialNumber(ctx, deviceSN)
	if err != nil {
		return nil, fmt.Errorf("lookup device %s: %w", deviceSN, err)
	}

	cases, ok := r.cases[category]
	if !ok {
		return nil, nil
	}

	var results []TestResult
	for _, tc := range cases {
		result := r.runTestCase(ctx, dev, tc)
		results = append(results, result)
	}
	return results, nil
}

// runTestCase executes a single test case and returns the result.
func (r *ConformanceTestRunner) runTestCase(ctx context.Context, dev *model.Device, tc TestCase) TestResult {
	start := time.Now()
	result := TestResult{
		TestCaseID: tc.ID,
		TestName:   tc.Name,
		Category:   tc.Category,
		Passed:     true,
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

// checkNotEmpty verifies that a device field is not empty. Supports both single
// "field" and batch "fields" parameters.
func (r *ConformanceTestRunner) checkNotEmpty(dev *model.Device, field string, step TestStep) error {
	// Handle batch "fields" parameter.
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

// checkParamPathsPresent verifies that device parameters include paths from the data model.
func (r *ConformanceTestRunner) checkParamPathsPresent(ctx context.Context, dev *model.Device) error {
	dm, err := r.dataModelReg.ResolveForDevice(ctx, dev)
	if err != nil {
		return fmt.Errorf("resolve data model: %w", err)
	}
	if dm == nil {
		return fmt.Errorf("no data model found for device %s", dev.SerialNumber)
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

// checkParamTypesMatch validates parameter types against the data model.
func (r *ConformanceTestRunner) checkParamTypesMatch(ctx context.Context, dev *model.Device) error {
	dm, err := r.dataModelReg.ResolveForDevice(ctx, dev)
	if err != nil {
		return fmt.Errorf("resolve data model: %w", err)
	}
	if dm == nil {
		return fmt.Errorf("no data model found for device %s", dev.SerialNumber)
	}

	params, err := r.paramRepo.GetByDevice(ctx, dev.ID)
	if err != nil {
		return fmt.Errorf("get device parameters: %w", err)
	}

	if len(params) == 0 {
		return fmt.Errorf("device has no reported parameters")
	}

	// Parse the parameter tree from the data model.
	var dmParams []datamodel.Parameter
	if err := json.Unmarshal(dm.ParameterTree, &dmParams); err != nil {
		return fmt.Errorf("parse data model parameter tree: %w", err)
	}

	// Build a lookup of expected types.
	expectedTypes := make(map[string]string, len(dmParams))
	for _, p := range dmParams {
		expectedTypes[p.Path] = p.Type
	}

	// Check each device parameter against expected types.
	mismatches := 0
	for _, p := range params {
		expected, ok := expectedTypes[p.ParameterPath]
		if !ok {
			continue // extra param, not a type mismatch
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

	cmd := &cmdqueue.Command{
		ID:         uuid.New().String(),
		Method:     method,
		Params:     paramsJSON,
		Priority:   5,
		CreatedAt:  time.Now(),
		CommandKey: fmt.Sprintf("interop-%s-%s", dev.SerialNumber, method),
	}

	if err := r.cmdQueue.Push(ctx, dev.SerialNumber, cmd); err != nil {
		return fmt.Errorf("enqueue %s command: %w", method, err)
	}

	r.logger.Info("interop RPC command enqueued",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("method", method),
		zap.String("command_id", cmd.ID),
	)
	return nil
}

// executeVerifyResponse performs response verification checks.
func (r *ConformanceTestRunner) executeVerifyResponse(ctx context.Context, dev *model.Device, step TestStep) error {
	check, _ := step.Params["check"].(string)

	switch check {
	case "command_queued":
		qLen, err := r.cmdQueue.Len(ctx, dev.SerialNumber)
		if err != nil {
			return fmt.Errorf("check command queue length: %w", err)
		}
		if qLen == 0 {
			return fmt.Errorf("command queue is empty after enqueue")
		}
		return nil

	case "model_exists":
		dm, err := r.dataModelReg.ResolveForDevice(ctx, dev)
		if err != nil {
			return fmt.Errorf("resolve data model: %w", err)
		}
		if dm == nil {
			return fmt.Errorf("no data model found for device %s", dev.SerialNumber)
		}
		return nil

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
