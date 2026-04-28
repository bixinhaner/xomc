package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// FilterEngine 告警过滤引擎，根据告警过滤规则处理入站告警。
type FilterEngine struct {
	filterRepo     AlarmFilterRuleRepository
	store          AlarmStore
	dispatcher     WebhookDispatcher
	deadLetterRepo DeadLetterRepository
	metrics        *WebhookMetrics
	logger         *zap.Logger
}

// NewFilterEngine 创建告警过滤引擎。
//
// dispatcher / deadLetterRepo / metrics 可为 nil：
//   - dispatcher == nil      → 回退到 noopWebhookDispatcher，notify_webhook 动作记 skipped 计数；
//   - deadLetterRepo == nil  → ErrDeadLetter 时仅记日志，不持久化（适合内存测试）；
//   - metrics == nil         → 不记 Prometheus 指标。
func NewFilterEngine(
	filterRepo AlarmFilterRuleRepository,
	store AlarmStore,
	dispatcher WebhookDispatcher,
	deadLetterRepo DeadLetterRepository,
	metrics *WebhookMetrics,
	logger *zap.Logger,
) *FilterEngine {
	if dispatcher == nil {
		dispatcher = noopWebhookDispatcher{}
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FilterEngine{
		filterRepo:     filterRepo,
		store:          store,
		dispatcher:     dispatcher,
		deadLetterRepo: deadLetterRepo,
		metrics:        metrics,
		logger:         logger,
	}
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

	case FilterTypeAlarmIdentifier:
		if len(rule.AlarmIdentifiers) == 0 {
			return true
		}
		for _, code := range rule.AlarmIdentifiers {
			if code == alarm.AlarmIdentifier {
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
			zap.String("alarm_identifier", alarm.AlarmIdentifier),
			zap.String("rule_name", rule.Name))
		return &ProcessResult{Handled: true, Action: FilterActionIgnore}, nil

	case FilterActionAutoAcknowledge:
		alarm.Status = model.AlarmAcknowledged
		now := time.Now()
		alarm.AcknowledgedAt = &now
		alarm.AcknowledgedBy = strPtr("system:auto_filter:" + rule.Name)
		e.logger.Info("alarm auto-acknowledged by filter",
			zap.String("alarm_identifier", alarm.AlarmIdentifier),
			zap.String("rule_name", rule.Name),
			zap.String("desc", rule.AcknowledgeDesc))
		return &ProcessResult{Handled: true, Action: FilterActionAutoAcknowledge}, nil

	case FilterActionAutoClear:
		e.logger.Info("alarm auto-cleared by filter",
			zap.String("alarm_identifier", alarm.AlarmIdentifier),
			zap.String("rule_name", rule.Name))
		return &ProcessResult{Handled: true, Action: FilterActionAutoClear}, nil

	case FilterActionNotifyWebhook:
		e.dispatchWebhook(ctx, alarm, rule)
		// 不阻塞：dispatch 错误已在 dispatchWebhook 内部 log + 计数 + dead-letter 落库，调用方仍视为已处理
		return &ProcessResult{Handled: true, Action: FilterActionNotifyWebhook}, nil

	default:
		return &ProcessResult{Handled: false, Action: FilterActionDefault}, nil
	}
}

// dispatchWebhook 构建 alarm payload 并交给 dispatcher 发送。
// dispatcher 内部已实现 retry / HMAC，这里负责：
//   - skipped 路径（缺 URL）记 metric 并返回；
//   - ErrDeadLetter 时写一条 dead-letter 记录（如配置了 repo），便于后续排查 / 重投递。
func (e *FilterEngine) dispatchWebhook(ctx context.Context, alarm *model.Alarm, rule *AlarmFilterRule) {
	if rule.WebhookURL == nil || *rule.WebhookURL == "" {
		// DB CHECK 约束理论上拒绝此情况，留作运行时兜底。
		e.logger.Warn("notify_webhook rule missing webhook_url",
			zap.String("rule_name", rule.Name),
			zap.String("alarm_identifier", alarm.AlarmIdentifier))
		if e.metrics != nil {
			e.metrics.DispatchTotal.WithLabelValues("skipped").Inc()
		}
		return
	}

	payload := buildWebhookPayload(alarm)
	body, err := json.Marshal(payload)
	if err != nil {
		e.logger.Warn("marshal webhook payload failed",
			zap.String("rule_name", rule.Name),
			zap.Error(err))
		if e.metrics != nil {
			e.metrics.DispatchTotal.WithLabelValues("dead_letter").Inc()
		}
		return
	}

	secret := ""
	if rule.WebhookSecret != nil {
		secret = *rule.WebhookSecret
	}

	if err := e.dispatcher.Dispatch(ctx, *rule.WebhookURL, secret, body); err != nil {
		// dispatcher 内部已记 metric（success/retry/dead_letter）+ warn；
		// 这里追加 rule 上下文，并在 dead-letter 时持久化死信记录。
		e.logger.Warn("webhook dispatch returned error",
			zap.String("rule_name", rule.Name),
			zap.String("alarm_identifier", alarm.AlarmIdentifier),
			zap.Error(err))
		if errors.Is(err, ErrDeadLetter) && e.deadLetterRepo != nil {
			rec := &DeadLetterRecord{
				FilterID:   rule.ID,
				AlarmID:    alarm.ID,
				Payload:    body,
				LastError:  err.Error(),
				RetryCount: webhookRetryMaxRetry,
				FailedAt:   time.Now(),
			}
			if dlErr := e.deadLetterRepo.Insert(ctx, rec); dlErr != nil {
				e.logger.Error("insert dead-letter record failed",
					zap.String("rule_name", rule.Name),
					zap.String("alarm_identifier", alarm.AlarmIdentifier),
					zap.Error(dlErr))
			}
		}
		return
	}

	e.logger.Info("alarm notify_webhook dispatched",
		zap.String("rule_name", rule.Name),
		zap.String("alarm_identifier", alarm.AlarmIdentifier),
		zap.String("webhook_url", *rule.WebhookURL))
}

// webhookPayload 是当前的 alarm webhook 负载结构。
// 后续任务（用户可配模板）将此结构作为默认模板的字段集。
type webhookPayload struct {
	AlarmIdentifier string    `json:"alarm_identifier"`
	Severity        int       `json:"severity"` // 1=Critical/2=Major/3=Minor/4=Warning，与 global.AlarmSeverity 对齐
	AlarmSource     string    `json:"alarm_source,omitempty"`
	DeviceID        string    `json:"device_id"`
	DeviceSN        string    `json:"device_sn,omitempty"`
	RaisedAt        time.Time `json:"raised_at"`
	Status          string    `json:"status"`
	ProbableCause   string    `json:"probable_cause,omitempty"`
}

func buildWebhookPayload(alarm *model.Alarm) webhookPayload {
	p := webhookPayload{
		AlarmIdentifier: alarm.AlarmIdentifier,
		Severity:        int(alarm.Severity),
		DeviceID:        alarm.DeviceID.String(),
		DeviceSN:        alarm.DeviceSN,
		RaisedAt:        alarm.RaisedAt,
		Status:          string(alarm.Status),
	}
	if alarm.AlarmSource != nil {
		p.AlarmSource = *alarm.AlarmSource
	}
	if alarm.ProbableCause != nil {
		p.ProbableCause = *alarm.ProbableCause
	}
	return p
}

// enrichFromLibrary 从告警库补充告警信息。
func (e *FilterEngine) enrichFromLibrary(ctx context.Context, alarm *model.Alarm) {
	if (alarm.ProbableCause != nil && *alarm.ProbableCause != "") && (alarm.AlarmSource != nil && *alarm.AlarmSource != "") {
		return
	}
	// 后续集成时通过注入 AlarmLibraryRepository 实现
}
