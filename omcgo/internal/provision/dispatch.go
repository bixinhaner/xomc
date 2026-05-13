package provision

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/config/template"
	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

// ErrDispatchTemplateInactive 模板未启用时显式 dispatch 拒绝。
// 与 HandleBootstrap 走 template.Match 的"未匹配回退 Path B/C"不同：
// 显式 dispatch 是用户操作，模板停用时应给明确错误。
var ErrDispatchTemplateInactive = errors.New("provision dispatch: template is inactive")

// DispatchTemplate explicitly runs Path A (template-driven SetParameterValues)
// for one device, bypassing the A→B→C selector in HandleBootstrap.
//
// 调用场景：UI 选模板 + 选设备 → POST /api/v1/templates/:id/dispatch。
// 与 HandleBootstrap 的差异：
//   - 不走 template.Match（模板由调用方显式指定）。
//   - 不受 config.AutoConfigure 守门（显式操作语义优先）。
//   - 不查 PathBEnabled / ModelUpload.Enabled（绕开 selector）。
//
// 保留 HandleBootstrap 的关键前置：
//   - cancel stale non-terminal task；
//   - 创建新 ProvisioningTask 并落库；
//   - bindDeviceProduct 回写 product_id / param_model_id（B1，syncService 依赖）；
//   - 复用既有私有 handleTemplateProvisioning：内部走 ResolveTranslator →
//     BuildProvisioningStepsTranslated → EnqueueSteps（standardPath ↔ privatePath
//     翻译已就位）。
func (e *ProvisioningEngine) DispatchTemplate(ctx context.Context,
	tmpl *template.ConfigTemplate, deviceID uuid.UUID) (uuid.UUID, error) {

	if tmpl == nil {
		return uuid.Nil, fmt.Errorf("provision dispatch: template is nil")
	}
	if !tmpl.Active {
		return uuid.Nil, ErrDispatchTemplateInactive
	}

	dev, err := e.deviceService.GetDevice(ctx, deviceID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("provision dispatch: get device %s: %w", deviceID, err)
	}
	if dev == nil {
		return uuid.Nil, fmt.Errorf("provision dispatch: device %s not found", deviceID)
	}

	if existing, _ := e.taskRepo.GetByDeviceID(ctx, deviceID); existing != nil && !IsTerminal(existing.Status) {
		e.logger.Warn("cancelling stale provisioning task for explicit dispatch",
			zap.String("device_sn", dev.SerialNumber),
			zap.String("existing_task_id", existing.ID.String()),
			zap.String("existing_status", string(existing.Status)),
			zap.Duration("task_age", time.Since(existing.CreatedAt)),
		)
		_ = e.failTask(ctx, existing,
			fmt.Errorf("device received explicit template dispatch, cancelling stale task in state %s", existing.Status))
	}

	task := NewProvisioningTask(deviceID)
	now := time.Now()
	task.StartedAt = &now
	if err := e.taskRepo.Create(ctx, task); err != nil {
		return uuid.Nil, fmt.Errorf("provision dispatch: create task: %w", err)
	}
	e.publishEvent(ctx, event.SubjectProvisionStarted, task)

	if err := e.transitionTask(ctx, task, StateIdentifying); err != nil {
		return task.ID, e.failTask(ctx, task, fmt.Errorf("provision dispatch: transition to identifying: %w", err))
	}
	e.bindDeviceProduct(ctx, dev)

	if err := e.transitionTask(ctx, task, StateMatching); err != nil {
		return task.ID, e.failTask(ctx, task, fmt.Errorf("provision dispatch: transition to matching: %w", err))
	}

	if err := e.handleTemplateProvisioning(ctx, task, dev, tmpl, dev.SerialNumber); err != nil {
		return task.ID, err
	}

	e.logger.Info("template dispatched explicitly",
		zap.String("device_sn", dev.SerialNumber),
		zap.String("template_id", tmpl.ID.String()),
		zap.String("task_id", task.ID.String()),
	)
	return task.ID, nil
}
