package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/alarm/definition"
	appcontext "github.com/omcgo/omcgo/internal/core/context"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// alarmDefLookup 抽象 enrichFromLibrary 所需的告警字典只读查询。
// 由 *definition.Registry 满足；抽成接口便于 filter_engine 单测注入 stub，
// 并避免 FilterEngine 依赖 Registry 的全部生命周期方法（Refresh/Count…）。
type alarmDefLookup interface {
	Lookup(ctx context.Context, identifier string) (*definition.ResolvedDefinition, error)
}

// FilterEngine 告警过滤引擎，根据告警过滤规则处理入站告警。
type FilterEngine struct {
	filterRepo      AlarmFilterRuleRepository
	store           AlarmStore
	dispatcher      WebhookDispatcher
	deadLetterRepo  DeadLetterRepository
	deviceGroupResolver DeviceGroupResolver
	alarmDefs       alarmDefLookup
	metrics         *WebhookMetrics
	emailDispatcher EmailDispatcher
	logger          *zap.Logger
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
		filterRepo:      filterRepo,
		store:           store,
		dispatcher:      dispatcher,
		deadLetterRepo:  deadLetterRepo,
		metrics:         metrics,
		emailDispatcher: noopEmailDispatcher{}, // 默认 noop；DI 后通过 SetEmailDispatcher 切换
		logger:          logger,
	}
}

// SetEmailDispatcher 注入邮件派发器（W2.A.1/T-0007 整合）。
//
// 与 WebhookDispatcher 走构造函数参数不同，邮件 dispatcher 用 setter 是因为
// 兼容路径在 provider 层复用通知中心 transport，DI 装配晚于 NewFilterEngine 调用。
// dispatcher == nil 时回退到 noopEmailDispatcher。
func (e *FilterEngine) SetEmailDispatcher(dispatcher EmailDispatcher) {
	if dispatcher == nil {
		dispatcher = noopEmailDispatcher{}
	}
	e.emailDispatcher = dispatcher
}

// SetDeviceGroupResolver injects the resolver used by device_group filter rules.
// resolver == nil means device_group rules fail closed instead of matching all alarms.
func (e *FilterEngine) SetDeviceGroupResolver(resolver DeviceGroupResolver) {
	e.deviceGroupResolver = resolver
}

// SetAlarmDefLookup 注入告警字典只读查询（issue #67 运行期 i18n）。
//
// 用 setter 与 deviceGroupResolver 同理：AlarmDefRegistry 在 DI 容器里晚于
// NewFilterEngine 装配（依赖 dictloader 启动期 Refresh），且测试可零改保持兼容。
// lookup == nil 时 enrichFromLibrary 静默跳过补全（保留设备上报原文，零回归）。
func (e *FilterEngine) SetAlarmDefLookup(lookup alarmDefLookup) {
	e.alarmDefs = lookup
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
		if e.match(ctx, alarm, deviceID, &rule) {
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
// 规则中凡是填了值的维度都必须同时命中；filter_type 仅保留给存量契约和列表展示。
func (e *FilterEngine) match(ctx context.Context, alarm *model.Alarm, deviceID uuid.UUID, rule *AlarmFilterRule) bool {
	if len(rule.AlarmSources) > 0 {
		if alarm.AlarmSource == nil {
			return false
		}

		matched := false
		for _, src := range rule.AlarmSources {
			if src == *alarm.AlarmSource {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(rule.AlarmIdentifiers) > 0 {
		matched := false
		for _, code := range rule.AlarmIdentifiers {
			if code == alarm.AlarmIdentifier {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(rule.DeviceIDs) > 0 {
		matched := false
		for _, did := range rule.DeviceIDs {
			if did == deviceID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(rule.DeviceGroupIDs) > 0 {
		if e.deviceGroupResolver == nil {
			e.logger.Warn("device group rule skipped: resolver not configured",
				zap.String("rule_name", rule.Name),
				zap.String("device_id", deviceID.String()))
			return false
		}

		groupID, err := e.deviceGroupResolver.GetGroupID(ctx, deviceID)
		if err != nil {
			e.logger.Warn("resolve device group failed",
				zap.String("rule_name", rule.Name),
				zap.String("device_id", deviceID.String()),
				zap.Error(err))
			return false
		}
		if groupID == nil {
			return false
		}

		matched := false
		for _, gid := range rule.DeviceGroupIDs {
			if gid == *groupID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
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
		ackNote := autoAcknowledgeNote(rule)
		alarm.AcknowledgedAt = &now
		alarm.AcknowledgedBy = strPtr("system:auto_filter:" + rule.Name)
		alarm.AckNote = &ackNote
		e.logger.Info("alarm auto-acknowledged by filter",
			zap.String("alarm_identifier", alarm.AlarmIdentifier),
			zap.String("rule_name", rule.Name),
			zap.String("desc", rule.AcknowledgeDesc))
		return &ProcessResult{Handled: true, Action: FilterActionAutoAcknowledge}, nil

	case FilterActionAutoClear:
		clearNote := autoClearNote(rule)
		alarm.ClearedBy = strPtr("system")
		alarm.ClearNote = &clearNote
		e.logger.Info("alarm auto-cleared by filter",
			zap.String("alarm_identifier", alarm.AlarmIdentifier),
			zap.String("rule_name", rule.Name))
		return &ProcessResult{Handled: true, Action: FilterActionAutoClear}, nil

	case FilterActionNotifyWebhook:
		e.dispatchWebhook(ctx, alarm, rule)
		// 不阻塞：dispatch 错误已在 dispatchWebhook 内部 log + 计数 + dead-letter 落库，调用方仍视为已处理
		return &ProcessResult{Handled: true, Action: FilterActionNotifyWebhook}, nil

	case FilterActionNotifyEmail:
		e.dispatchEmail(ctx, alarm, rule)
		// 与 webhook 一致：rule 命中即视为已处理，dispatch 失败仅在 emailDispatcher 内部记 metric
		return &ProcessResult{Handled: true, Action: FilterActionNotifyEmail}, nil

	case FilterActionLegacyNotificationBarrier:
		e.logger.Debug("alarm handled by legacy notification compatibility barrier",
			zap.String("alarm_identifier", alarm.AlarmIdentifier),
			zap.String("rule_name", rule.Name))
		return &ProcessResult{Handled: true, Action: FilterActionLegacyNotificationBarrier}, nil

	default:
		return &ProcessResult{Handled: false, Action: FilterActionDefault}, nil
	}
}

func autoAcknowledgeNote(rule *AlarmFilterRule) string {
	if desc := strings.TrimSpace(rule.AcknowledgeDesc); desc != "" {
		return desc
	}
	if name := strings.TrimSpace(rule.Name); name != "" {
		return fmt.Sprintf("auto-acknowledged by alarm rule: %s", name)
	}
	return "auto-acknowledged by alarm rule"
}

func autoClearNote(rule *AlarmFilterRule) string {
	if name := strings.TrimSpace(rule.Name); name != "" {
		return fmt.Sprintf("auto-cleared by alarm rule: %s", name)
	}
	return "auto-cleared by alarm rule"
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

// dispatchEmail 构建告警邮件并交给 emailDispatcher 发送（W2.A.1/T-0007 整合）。
//
// emailDispatcher 内部已实现 timeout + metrics（success/failure/timeout）。
// 这里负责：
//   - skipped 路径：rule.EmailRecipients 为空 → 仅记日志（DB 列允许 NULL，但 action=notify_email 时应非空，应用层校验由 handler 层负责）；
//   - dispatch 失败：仅记日志，不阻塞；rule 命中视为已处理（与 dispatchWebhook 风格一致）。
func (e *FilterEngine) dispatchEmail(ctx context.Context, alarm *model.Alarm, rule *AlarmFilterRule) {
	if len(rule.EmailRecipients) == 0 {
		e.logger.Warn("notify_email rule missing recipients",
			zap.String("rule_name", rule.Name),
			zap.String("alarm_identifier", alarm.AlarmIdentifier))
		return
	}

	subject := buildEmailSubject(alarm)
	body := buildEmailBody(alarm)

	if err := e.emailDispatcher.Dispatch(ctx, rule.EmailRecipients, subject, body); err != nil {
		e.logger.Warn("email dispatch returned error",
			zap.String("rule_name", rule.Name),
			zap.String("alarm_identifier", alarm.AlarmIdentifier),
			zap.Strings("to", rule.EmailRecipients),
			zap.Error(err))
		return
	}

	e.logger.Info("alarm notify_email dispatched",
		zap.String("rule_name", rule.Name),
		zap.String("alarm_identifier", alarm.AlarmIdentifier),
		zap.Strings("to", rule.EmailRecipients))
}

// buildEmailSubject 拼最小可用的邮件标题。模板化由 T-0043 通知模板/历史 UI 任务接管。
func buildEmailSubject(alarm *model.Alarm) string {
	host := alarm.DeviceSN
	if host == "" {
		host = alarm.DeviceID.String()
	}
	return fmt.Sprintf("[OMC告警-Sev%d] %s on %s", int(alarm.Severity), alarm.AlarmIdentifier, host)
}

// buildEmailBody 拼纯文本邮件正文。字段集合与 webhookPayload 一致。
func buildEmailBody(alarm *model.Alarm) string {
	var sb strings.Builder
	sb.WriteString("OMC 告警通知\n\n")
	sb.WriteString(fmt.Sprintf("告警标识: %s\n", alarm.AlarmIdentifier))
	sb.WriteString(fmt.Sprintf("严重等级: %d\n", int(alarm.Severity)))
	if alarm.AlarmSource != nil && *alarm.AlarmSource != "" {
		sb.WriteString(fmt.Sprintf("告警源: %s\n", *alarm.AlarmSource))
	}
	sb.WriteString(fmt.Sprintf("设备 ID: %s\n", alarm.DeviceID.String()))
	if alarm.DeviceSN != "" {
		sb.WriteString(fmt.Sprintf("设备 SN: %s\n", alarm.DeviceSN))
	}
	sb.WriteString(fmt.Sprintf("发生时间: %s\n", alarm.RaisedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("当前状态: %s\n", alarm.Status))
	if alarm.ProbableCause != nil && *alarm.ProbableCause != "" {
		sb.WriteString(fmt.Sprintf("可能原因: %s\n", *alarm.ProbableCause))
	}
	return sb.String()
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

// enrichFromLibrary 从告警字典（alarm_definitions）按请求 locale 补全告警信息（issue #67）。
//
// 补全规则（以 ctx locale 选 cn/en 列，COALESCE(NULLIF(en,''),cn) 退化）：
//   - Description（告警名）：字典命中则用字典本地化名覆盖；命中失败（unknown identifier）
//     保留设备上报原文，绝不置空。
//   - ProbableCause（可能原因）：设备未上报时用字典本地化 probable_cause 补；设备已上报则尊重原文。
//
// 设计取舍：写入侧（落库时）按当时 locale 冻结 Description，保证 webhook/邮件/历史归档拿到
// 已本地化的名称；活跃/历史**列表**展示则在 pg_store 查询侧再 JOIN alarm_definitions 按
// 请求 locale 取名（见 localizedAlarmNameExpr），使语言切换即时生效、且反映字典后续编辑。
func (e *FilterEngine) enrichFromLibrary(ctx context.Context, alarm *model.Alarm) {
	if e.alarmDefs == nil || alarm == nil || alarm.AlarmIdentifier == "" {
		return
	}

	rd, err := e.alarmDefs.Lookup(ctx, alarm.AlarmIdentifier)
	if err != nil {
		// unknown identifier 或查询错误：保留设备上报原文，不阻断处理。
		if !errors.Is(err, definition.ErrUnknownIdentifier) {
			e.logger.Warn("alarm definition lookup failed during enrich",
				zap.String("alarm_identifier", alarm.AlarmIdentifier),
				zap.Error(err))
		}
		return
	}

	loc := appcontext.GetLocale(ctx)

	if name := localizedName(loc, rd.CnName, rd.EnName); name != "" {
		alarm.Description = name
	}

	if alarm.ProbableCause == nil || *alarm.ProbableCause == "" {
		if cause := localizedName(loc, rd.CnProbableCause, rd.EnProbableCause); cause != "" {
			alarm.ProbableCause = &cause
		}
	}
}

// localizedName 按 locale 在中/英文之间取值，并做 COALESCE(NULLIF(en,''),cn) 式退化：
//   - en-US：优先英文，英文空则回退中文；
//   - 其它（含 zh-CN / 缺省）：优先中文，中文空则回退英文。
// 两者皆空返回空串，调用方据此决定是否保留原文。
func localizedName(loc appcontext.Locale, cn, en string) string {
	if loc == appcontext.LocaleEN {
		if en != "" {
			return en
		}
		return cn
	}
	if cn != "" {
		return cn
	}
	return en
}
