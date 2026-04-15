package alarm

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// FilterEngine 告警过滤引擎，根据告警过滤规则处理入站告警。
type FilterEngine struct {
	filterRepo AlarmFilterRuleRepository
	store      AlarmStore
	logger     *zap.Logger
}

// NewFilterEngine 创建告警过滤引擎。
func NewFilterEngine(filterRepo AlarmFilterRuleRepository, store AlarmStore, logger *zap.Logger) *FilterEngine {
	return &FilterEngine{filterRepo: filterRepo, store: store, logger: logger}
}

// ProcessResult 过滤处理结果。
type ProcessResult struct {
	Handled bool   // 是否被过滤规则处理
	Action  string // 执行的动作
}

// ProcessAlarm 处理入站告警，返回处理结果。
func (e *FilterEngine) ProcessAlarm(ctx context.Context, alarm *model.Alarm, deviceID uuid.UUID) (*ProcessResult, error) {
	// 1. 从告警库补充信息
	e.enrichFromLibrary(ctx, alarm)

	// 2. 获取所有启用的过滤规则
	rules, err := e.filterRepo.ListEnabled(ctx)
	if err != nil {
		e.logger.Warn("list enabled filter rules failed", zap.Error(err))
		return &ProcessResult{Handled: false, Action: FilterActionDefault}, nil
	}

	// 3. 按优先级匹配规则
	for _, rule := range rules {
		if e.match(alarm, deviceID, &rule) {
			result, err := e.executeAction(ctx, alarm, &rule)
			if err != nil {
				return nil, err
			}
			return result, nil
		}
	}

	// 4. 无匹配规则，走默认流程
	return &ProcessResult{Handled: false, Action: FilterActionDefault}, nil
}

// match 检查告警是否匹配过滤规则。
func (e *FilterEngine) match(alarm *model.Alarm, deviceID uuid.UUID, rule *AlarmFilterRule) bool {
	switch rule.FilterType {
	case FilterTypeAlarmSource:
		if len(rule.AlarmSources) == 0 {
			return true
		}
		for _, src := range rule.AlarmSources {
			if alarm.AlarmSource != nil && src == *alarm.AlarmSource {
				return true
			}
		}
		return false

	case FilterTypeAlarmCode:
		if len(rule.AlarmCodes) == 0 {
			return true
		}
		for _, code := range rule.AlarmCodes {
			if code == alarm.AlarmCode {
				return true
			}
		}
		return false

	case FilterTypeDevice:
		if len(rule.DeviceIDs) == 0 {
			return true
		}
		for _, did := range rule.DeviceIDs {
			if did == deviceID {
				return true
			}
		}
		return false

	case FilterTypeDeviceGroup:
		if len(rule.DeviceGroupIDs) == 0 {
			return true
		}
		return true // 设备组匹配由上层处理

	default:
		return true
	}
}

// executeAction 执行过滤动作。
func (e *FilterEngine) executeAction(ctx context.Context, alarm *model.Alarm, rule *AlarmFilterRule) (*ProcessResult, error) {
	switch rule.Action {
	case FilterActionIgnore:
		e.logger.Debug("alarm ignored by filter rule",
			zap.String("alarm_code", alarm.AlarmCode),
			zap.String("rule_name", rule.Name))
		return &ProcessResult{Handled: true, Action: FilterActionIgnore}, nil

	case FilterActionAutoAcknowledge:
		alarm.Status = model.AlarmAcknowledged
		now := time.Now()
		alarm.AcknowledgedAt = &now
		alarm.AcknowledgedBy = strPtr("system:auto_filter:" + rule.Name)
		e.logger.Info("alarm auto-acknowledged by filter",
			zap.String("alarm_code", alarm.AlarmCode),
			zap.String("rule_name", rule.Name),
			zap.String("desc", rule.AcknowledgeDesc))
		return &ProcessResult{Handled: true, Action: FilterActionAutoAcknowledge}, nil

	case FilterActionAutoClear:
		e.logger.Info("alarm auto-cleared by filter",
			zap.String("alarm_code", alarm.AlarmCode),
			zap.String("rule_name", rule.Name))
		return &ProcessResult{Handled: true, Action: FilterActionAutoClear}, nil

	default:
		return &ProcessResult{Handled: false, Action: FilterActionDefault}, nil
	}
}

// enrichFromLibrary 从告警库补充告警信息。
func (e *FilterEngine) enrichFromLibrary(ctx context.Context, alarm *model.Alarm) {
	if (alarm.ProbableCause != nil && *alarm.ProbableCause != "") && (alarm.AlarmSource != nil && *alarm.AlarmSource != "") {
		return
	}
	// 后续集成时通过注入 AlarmLibraryRepository 实现
}
