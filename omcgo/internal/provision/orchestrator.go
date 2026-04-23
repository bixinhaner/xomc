package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/task"
)

const (
	// MethodGetParameterValues is the TR069 RPC method for reading parameters.
	MethodGetParameterValues = "GetParameterValues"
	// MethodSetParameterValues is the TR069 RPC method for writing parameters.
	MethodSetParameterValues = "SetParameterValues"
	// MethodReboot is the TR069 RPC method for rebooting the device.
	MethodReboot = "Reboot"

	defaultStepTimeout = 30 * time.Second
)

// BuildProvisioningSteps generates the sequence of RPC steps needed to
// provision a device according to the given template.
func BuildProvisioningSteps(tmpl *template.ConfigTemplate) ([]ProvisioningStep, error) {
	var steps []ProvisioningStep

	// Step 1: GetParameterValues — read current device configuration.
	paramNames := extractParameterNames(tmpl.Parameters)
	if len(paramNames) > 0 {
		gpvParams, err := json.Marshal(map[string]interface{}{
			"parameter_names": paramNames,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal gpv params: %w", err)
		}
		steps = append(steps, ProvisioningStep{
			Order:    len(steps) + 1,
			Method:   MethodGetParameterValues,
			Params:   gpvParams,
			Timeout:  defaultStepTimeout,
			Required: true,
		})
	}

	// Step 2: SetParameterValues — write the template parameters.
	if len(tmpl.Parameters) > 0 {
		steps = append(steps, ProvisioningStep{
			Order:    len(steps) + 1,
			Method:   MethodSetParameterValues,
			Params:   tmpl.Parameters,
			Timeout:  defaultStepTimeout,
			Required: true,
		})
	}

	// Step 3: Reboot (optional — only if template has many parameter changes).
	// For now, always include a reboot step as most provisioning requires it.
	rebootParams, err := json.Marshal(map[string]string{
		"command_key": fmt.Sprintf("provision-%d", time.Now().Unix()),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal reboot params: %w", err)
	}
	steps = append(steps, ProvisioningStep{
		Order:    len(steps) + 1,
		Method:   MethodReboot,
		Params:   rebootParams,
		Timeout:  60 * time.Second,
		Required: false,
	})

	return steps, nil
}

// EnqueueSteps pushes provisioning steps into the unified device task queue.
func EnqueueSteps(ctx context.Context, deviceSN string, steps []ProvisioningStep, taskSvc task.Enqueuer) error {
	for _, step := range steps {
		if _, err := taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
			DeviceSN:   deviceSN,
			Method:     step.Method,
			Params:     step.Params,
			Priority:   step.Order,
			CommandKey: fmt.Sprintf("provision-%s-%d", step.Method, step.Order),
			Source:     task.TaskSourceSystem,
		}); err != nil {
			return fmt.Errorf("enqueue step %d (%s) for %s: %w", step.Order, step.Method, deviceSN, err)
		}
	}
	return nil
}

// extractParameterNames extracts the parameter paths from template parameters JSON.
func extractParameterNames(params json.RawMessage) []string {
	var paramMap map[string]interface{}
	if err := json.Unmarshal(params, &paramMap); err != nil {
		return nil
	}

	names := make([]string, 0, len(paramMap))
	for k := range paramMap {
		names = append(names, k)
	}
	return names
}
