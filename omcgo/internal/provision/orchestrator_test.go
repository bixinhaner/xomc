package provision

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/internal/model"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Tests: BuildProvisioningSteps
// ---------------------------------------------------------------------------

func TestBuildProvisioningSteps_WithParameters(t *testing.T) {
	tmpl := &template.ConfigTemplate{
		ID:           uuid.New(),
		Name:         "test-template",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
		TemplateType: template.TemplateProvisioning,
		Parameters:   json.RawMessage(`{"Device.WiFi.SSID": "OMC-Test", "Device.WiFi.Channel": 6}`),
		Active:       true,
	}

	steps := BuildProvisioningSteps(tmpl)

	// Expect 3 steps: GetParameterValues, SetParameterValues, Reboot.
	require.Len(t, steps, 3)

	// Step 1: GetParameterValues.
	assert.Equal(t, 1, steps[0].Order)
	assert.Equal(t, MethodGetParameterValues, steps[0].Method)
	assert.True(t, steps[0].Required)
	assert.Equal(t, defaultStepTimeout, steps[0].Timeout)

	// Verify GPV params contain the parameter names.
	var gpvParams map[string]interface{}
	err := json.Unmarshal(steps[0].Params, &gpvParams)
	require.NoError(t, err)
	paramNames, ok := gpvParams["parameter_names"].([]interface{})
	require.True(t, ok)
	assert.Len(t, paramNames, 2)

	// Collect names into a set for order-independent comparison.
	nameSet := make(map[string]bool)
	for _, n := range paramNames {
		nameSet[n.(string)] = true
	}
	assert.True(t, nameSet["Device.WiFi.SSID"])
	assert.True(t, nameSet["Device.WiFi.Channel"])

	// Step 2: SetParameterValues.
	assert.Equal(t, 2, steps[1].Order)
	assert.Equal(t, MethodSetParameterValues, steps[1].Method)
	assert.True(t, steps[1].Required)
	assert.Equal(t, defaultStepTimeout, steps[1].Timeout)
	assert.JSONEq(t, `{"Device.WiFi.SSID": "OMC-Test", "Device.WiFi.Channel": 6}`, string(steps[1].Params))

	// Step 3: Reboot.
	assert.Equal(t, 3, steps[2].Order)
	assert.Equal(t, MethodReboot, steps[2].Method)
	assert.False(t, steps[2].Required)

	var rebootParams map[string]string
	err = json.Unmarshal(steps[2].Params, &rebootParams)
	require.NoError(t, err)
	assert.Contains(t, rebootParams["command_key"], "provision-")
}

func TestBuildProvisioningSteps_EmptyParameters(t *testing.T) {
	tmpl := &template.ConfigTemplate{
		ID:           uuid.New(),
		Name:         "empty-params-template",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
		TemplateType: template.TemplateProvisioning,
		Parameters:   json.RawMessage(`{}`),
		Active:       true,
	}

	steps := BuildProvisioningSteps(tmpl)

	// With empty parameters: no GPV (extractParameterNames returns empty),
	// no SPV (len(tmpl.Parameters) > 0 but the map is empty, however
	// json.RawMessage(`{}`) has len > 0, so SPV step IS included).
	// Actually, `len(tmpl.Parameters) > 0` checks the byte length of the
	// raw JSON, which is 2 for `{}`. So SPV is included.
	// But extractParameterNames(`{}`) returns an empty slice, so GPV is skipped.
	// Result: SetParameterValues + Reboot = 2 steps.
	require.Len(t, steps, 2)

	assert.Equal(t, MethodSetParameterValues, steps[0].Method)
	assert.Equal(t, 1, steps[0].Order)
	assert.Equal(t, MethodReboot, steps[1].Method)
	assert.Equal(t, 2, steps[1].Order)
}

func TestBuildProvisioningSteps_NilParameters(t *testing.T) {
	tmpl := &template.ConfigTemplate{
		ID:           uuid.New(),
		Name:         "nil-params-template",
		Carrier:      model.CarrierCMCC,
		Technology:   model.TechLTE,
		TemplateType: template.TemplateProvisioning,
		Parameters:   nil,
		Active:       true,
	}

	steps := BuildProvisioningSteps(tmpl)

	// With nil parameters: extractParameterNames returns nil (empty),
	// len(nil) == 0 for json.RawMessage, so no GPV and no SPV.
	// Only Reboot step.
	require.Len(t, steps, 1)
	assert.Equal(t, MethodReboot, steps[0].Method)
	assert.Equal(t, 1, steps[0].Order)
}

func TestBuildProvisioningSteps_OrderIsSequential(t *testing.T) {
	tmpl := &template.ConfigTemplate{
		ID:         uuid.New(),
		Parameters: json.RawMessage(`{"Device.X": "y"}`),
		Active:     true,
	}

	steps := BuildProvisioningSteps(tmpl)
	for i, s := range steps {
		assert.Equal(t, i+1, s.Order, "step %d should have Order=%d", i, i+1)
	}
}

// ---------------------------------------------------------------------------
// Tests: EnqueueSteps
// ---------------------------------------------------------------------------

func TestEnqueueSteps_Success(t *testing.T) {
	steps := []ProvisioningStep{
		{Order: 1, Method: MethodGetParameterValues, Params: json.RawMessage(`{}`)},
		{Order: 2, Method: MethodSetParameterValues, Params: json.RawMessage(`{"Device.WiFi.SSID": "test"}`)},
		{Order: 3, Method: MethodReboot, Params: json.RawMessage(`{"command_key": "provision-12345"}`)},
	}

	var pushed []*cmdqueue.Command
	queue := &mockCommandQueue{
		PushFn: func(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error {
			assert.Equal(t, "TEST-DEVICE-SN", deviceSN)
			pushed = append(pushed, cmd)
			return nil
		},
	}

	err := EnqueueSteps(context.Background(), "TEST-DEVICE-SN", steps, queue)
	require.NoError(t, err)
	require.Len(t, pushed, 3)

	// Verify commands reflect step data.
	assert.Equal(t, MethodGetParameterValues, pushed[0].Method)
	assert.Equal(t, 1, pushed[0].Priority)
	assert.Equal(t, "provision-GetParameterValues-1", pushed[0].CommandKey)

	assert.Equal(t, MethodSetParameterValues, pushed[1].Method)
	assert.Equal(t, 2, pushed[1].Priority)
	assert.Equal(t, "provision-SetParameterValues-2", pushed[1].CommandKey)

	assert.Equal(t, MethodReboot, pushed[2].Method)
	assert.Equal(t, 3, pushed[2].Priority)
	assert.Equal(t, "provision-Reboot-3", pushed[2].CommandKey)
}

func TestEnqueueSteps_ErrorOnSecondStep(t *testing.T) {
	steps := []ProvisioningStep{
		{Order: 1, Method: MethodGetParameterValues, Params: json.RawMessage(`{}`)},
		{Order: 2, Method: MethodSetParameterValues, Params: json.RawMessage(`{}`)},
		{Order: 3, Method: MethodReboot, Params: json.RawMessage(`{}`)},
	}

	callCount := 0
	queue := &mockCommandQueue{
		PushFn: func(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error {
			callCount++
			if callCount == 2 {
				return errors.New("redis CLUSTERDOWN")
			}
			return nil
		},
	}

	err := EnqueueSteps(context.Background(), "SN-ERR", steps, queue)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enqueue step 2")
	assert.Contains(t, err.Error(), "SetParameterValues")
	assert.Contains(t, err.Error(), "redis CLUSTERDOWN")
	assert.Equal(t, 2, callCount, "should stop after second step fails")
}

func TestEnqueueSteps_EmptySteps(t *testing.T) {
	queue := &mockCommandQueue{
		PushFn: func(ctx context.Context, deviceSN string, cmd *cmdqueue.Command) error {
			t.Fatal("Push should not be called for empty steps")
			return nil
		},
	}

	err := EnqueueSteps(context.Background(), "SN-EMPTY", nil, queue)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Tests: extractParameterNames
// ---------------------------------------------------------------------------

func TestExtractParameterNames(t *testing.T) {
	t.Run("valid JSON object", func(t *testing.T) {
		params := json.RawMessage(`{"Device.WiFi.SSID": "OMC", "Device.WiFi.Channel": 6}`)
		names := extractParameterNames(params)
		assert.Len(t, names, 2)

		nameSet := make(map[string]bool)
		for _, n := range names {
			nameSet[n] = true
		}
		assert.True(t, nameSet["Device.WiFi.SSID"])
		assert.True(t, nameSet["Device.WiFi.Channel"])
	})

	t.Run("empty JSON object", func(t *testing.T) {
		params := json.RawMessage(`{}`)
		names := extractParameterNames(params)
		assert.Empty(t, names)
	})

	t.Run("nil input", func(t *testing.T) {
		names := extractParameterNames(nil)
		assert.Nil(t, names)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		params := json.RawMessage(`not json`)
		names := extractParameterNames(params)
		assert.Nil(t, names)
	})

	t.Run("JSON array is not an object", func(t *testing.T) {
		params := json.RawMessage(`["a", "b"]`)
		names := extractParameterNames(params)
		assert.Nil(t, names)
	})
}

// ---------------------------------------------------------------------------
// Tests: Constants
// ---------------------------------------------------------------------------

func TestMethodConstants(t *testing.T) {
	assert.Equal(t, "GetParameterValues", MethodGetParameterValues)
	assert.Equal(t, "SetParameterValues", MethodSetParameterValues)
	assert.Equal(t, "Reboot", MethodReboot)
}
