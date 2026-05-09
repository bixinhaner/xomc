package provision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/config/parammodel"
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
// sourceID 透传 ProvisioningTask.ID，让 CompletionRouter 在 task.failed/completed
// 时按 source_id 反查并联动该 ProvisioningTask（D2 修复）。
func EnqueueSteps(ctx context.Context, deviceSN string, steps []ProvisioningStep, taskSvc task.Enqueuer, sourceID string) error {
	for _, step := range steps {
		if _, err := taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
			DeviceSN:   deviceSN,
			Method:     step.Method,
			Params:     step.Params,
			Priority:   step.Order,
			CommandKey: fmt.Sprintf("provision-%s-%d", step.Method, step.Order),
			Source:     task.TaskSourceSystem,
			SourceID:   sourceID,
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

// BuildProvisioningStepsTranslated 是 T-0098 P2-05 Path A 双栈版本。
//
// 设计契约（§1.11 Path A）：模板 Parameters JSON 的 key 视为 standardPath；构建步骤
// 前通过 Translator.ToPrivate 翻译为设备私有路径，使 SPV / GPV 下发时携带正确的
// 私有路径。translator 为 nil → 等价于 BuildProvisioningSteps（不翻译，沿用旧栈语义）。
//
// 翻译策略：
//   - Translator.ToPrivate(k).Found=true   → 使用 Translated（privatePath）
//   - Found=false                          → 保留原 key（容错；可能是模板存了 privatePath 老数据）
//   - 模板 Parameters 为空                 → 直接 fallthrough
func BuildProvisioningStepsTranslated(tmpl *template.ConfigTemplate, translator *parammodel.Translator) ([]ProvisioningStep, error) {
	if translator == nil {
		return BuildProvisioningSteps(tmpl)
	}
	translatedParams, err := translateTemplateParameters(tmpl.Parameters, translator)
	if err != nil {
		return nil, fmt.Errorf("translate template parameters: %w", err)
	}
	clone := *tmpl
	clone.Parameters = translatedParams
	return BuildProvisioningSteps(&clone)
}

// translateTemplateParameters 把 Parameters JSON 的 key（standardPath）翻译为 privatePath。
//
// 输入空 / nil → 原样返回。Unmarshal 失败 → 返回原 RawMessage（容错：模板可能存的是
// 字符串数组 / 嵌套对象等其它形态，留给后续 SPV 路径自己消费）。
func translateTemplateParameters(params json.RawMessage, translator *parammodel.Translator) (json.RawMessage, error) {
	if len(params) == 0 || translator == nil {
		return params, nil
	}
	var src map[string]any
	if err := json.Unmarshal(params, &src); err != nil {
		// 容错：保留原 JSON，由旧栈消费
		return params, nil
	}
	if len(src) == 0 {
		return params, nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		result := translator.ToPrivate(k)
		dst[result.Translated] = v
	}
	out, err := json.Marshal(dst)
	if err != nil {
		return nil, fmt.Errorf("marshal translated parameters: %w", err)
	}
	return out, nil
}
