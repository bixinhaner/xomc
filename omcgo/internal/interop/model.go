package interop

import "time"

// TestCategory classifies interop test cases by their focus area.
type TestCategory string

const (
	CategoryProtocol    TestCategory = "protocol"
	CategoryDataModel   TestCategory = "datamodel"
	CategoryRPC         TestCategory = "rpc"
	CategoryInform      TestCategory = "inform"
	CategoryFaultInject TestCategory = "fault_inject" // T-0114: 故障注入用例
)

// ValidTestCategories returns all supported test categories.
func ValidTestCategories() []TestCategory {
	return []TestCategory{CategoryProtocol, CategoryDataModel, CategoryRPC, CategoryInform, CategoryFaultInject}
}

// IsValid checks whether the category is a recognized test category.
func (c TestCategory) IsValid() bool {
	switch c {
	case CategoryProtocol, CategoryDataModel, CategoryRPC, CategoryInform, CategoryFaultInject:
		return true
	}
	return false
}

// TestCase defines a single conformance test with ordered steps.
//
// ExpectedOutcome controls how the runner aggregates step results:
//   - "" or "pass": default — any step failure marks the case failed.
//   - "fail": negative path — all steps passing marks the case failed
//     ("negative case unexpectedly passed"); at least one step failing
//     marks the case passed ("expected failure observed").
//
// TargetDeviceModels (T-0116) filters which device.ModelName values the
// case applies to. Nil or empty slice = run against any device (default
// behaviour). Non-empty = runner skips the case unless device.ModelName
// exactly matches one of the listed values. Used to scope vendor-specific
// fault / RPC cases (e.g. X_CMCC_Reboot probe only for cmcc-class models).
type TestCase struct {
	ID                 string       `json:"id"`
	Name               string       `json:"name"`
	Description        string       `json:"description"`
	Category           TestCategory `json:"category"`
	Steps              []TestStep   `json:"steps"`
	ExpectedOutcome    string       `json:"expected_outcome,omitempty"`
	TargetDeviceModels []string     `json:"target_device_models,omitempty"`
}

// TestStep describes one action within a test case.
type TestStep struct {
	Order       int                    `json:"order"`
	Description string                 `json:"description"`
	Action      string                 `json:"action"` // "send_rpc", "check_param", "verify_response"
	Params      map[string]interface{} `json:"params"`
}

// TestResult captures the outcome of executing a single test case against a device.
//
// ExpectedOutcome is copied from TestCase for visibility — UI / report can
// distinguish "passed because all steps succeeded" from "passed because the
// expected failure was observed".
type TestResult struct {
	TestCaseID      string        `json:"test_case_id"`
	TestName        string        `json:"test_name"`
	Category        TestCategory  `json:"category"`
	Passed          bool          `json:"passed"`
	Duration        time.Duration `json:"duration_ms"`
	Details         string        `json:"details"`
	Error           string        `json:"error,omitempty"`
	ExpectedOutcome string        `json:"expected_outcome,omitempty"`
}

// RunTestsRequest is the JSON body for the POST /interop/run endpoint.
type RunTestsRequest struct {
	DeviceSN   string         `json:"device_sn" binding:"required"`
	Categories []TestCategory `json:"categories"`
}
