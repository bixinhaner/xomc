package topology

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/event"
)

// GroupMatchEngine 设备分组自动匹配引擎。
//
// device_rules 引擎下线后，设备自动归组统一由本引擎驱动 —— 直接消费 L2 分组
// 自带的匹配规则（device_groups.matching_mode / name_rule_list / lac_list /
// tac_list / serial_number_list）。触发时机覆盖：
//   - 分组新增/编辑后  → MatchGroup（DeviceGroupService 异步调用）
//   - 新设备首次注册   → handleDeviceRegistered（device.registered 事件）
//   - 设备心跳 inform  → HeartbeatAssigner（matcher.go，另行接线）
//   - cron @hourly     → ReEvaluateAll（兜底全量重评估）
//
// 命中即无条件归组（经 AddDeviceAutoMatched，覆盖手工分配）；多个 L2 分组同时
// 命中时，"最近新增/编辑的分组优先"（DeviceMatcher.MatchDevice 按 updated_at
// 降序遍历）。
type GroupMatchEngine struct {
	matcher   *DeviceMatcher
	lister    DeviceLister
	groupRepo DeviceGroupRepository
	eventBus  event.EventBus
	logger    *zap.Logger

	cron *cron.Cron
	sub  event.Subscription
}

// NewGroupMatchEngine 构造分组匹配引擎。
func NewGroupMatchEngine(matcher *DeviceMatcher, lister DeviceLister, groupRepo DeviceGroupRepository, logger *zap.Logger) *GroupMatchEngine {
	return &GroupMatchEngine{matcher: matcher, lister: lister, groupRepo: groupRepo, logger: logger}
}

// SetEventBus 注入 EventBus，使 Start 装配 device.registered 订阅。
func (e *GroupMatchEngine) SetEventBus(bus event.EventBus) { e.eventBus = bus }

// MatchGroup 对单个 L2 分组做全量设备匹配：遍历所有设备，命中该组匹配规则的
// 归入此组。分组不存在 / 非 L2 / 未配匹配规则时安全 no-op。
// 供 DeviceGroupService 在分组新增/编辑后异步调用。
func (e *GroupMatchEngine) MatchGroup(ctx context.Context, groupID uuid.UUID) error {
	group, err := e.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("get group for match: %w", err)
	}
	if group == nil || group.Level != 2 || group.MatchingMode == "" {
		return nil // 非 L2 或未配匹配规则 —— 无需匹配
	}

	devices, err := e.lister.ListAllForRuleEval(ctx)
	if err != nil {
		return fmt.Errorf("list devices for group match: %w", err)
	}

	matched := 0
	for _, d := range devices {
		req := MatchRequest{DeviceID: d.ID, DeviceName: d.Name, SerialNumber: d.SerialNumber, LAC: d.LAC, TAC: d.TAC}
		ok, mErr := e.matcher.matchGroup(ctx, *group, req)
		if mErr != nil || !ok {
			continue
		}
		if aErr := e.groupRepo.AddDeviceAutoMatched(ctx, groupID, d.ID); aErr != nil {
			e.logger.Warn("group match: add device failed",
				zap.String("group_id", groupID.String()),
				zap.String("device_id", d.ID.String()),
				zap.Error(aErr))
			continue
		}
		matched++
	}
	e.logger.Info("group match done",
		zap.String("group_id", groupID.String()),
		zap.String("group_name", group.Name),
		zap.Int("matched", matched),
		zap.Int("scanned", len(devices)))
	return nil
}

// ReEvaluateAll 全量重评估：遍历所有设备，各自归入"最近编辑且命中"的 L2 分组。
// 作为 cron @hourly 兜底，纠正心跳/事件路径可能的遗漏。
func (e *GroupMatchEngine) ReEvaluateAll(ctx context.Context) error {
	devices, err := e.lister.ListAllForRuleEval(ctx)
	if err != nil {
		return fmt.Errorf("list devices for re-evaluate: %w", err)
	}
	reqs := make([]MatchRequest, 0, len(devices))
	for _, d := range devices {
		reqs = append(reqs, MatchRequest{
			DeviceID: d.ID, DeviceName: d.Name, SerialNumber: d.SerialNumber, LAC: d.LAC, TAC: d.TAC,
		})
	}
	matched, err := e.matcher.BatchMatchDevices(ctx, reqs)
	if err != nil {
		return fmt.Errorf("batch match devices: %w", err)
	}
	e.logger.Info("group re-evaluate done",
		zap.Int("matched", matched),
		zap.Int("scanned", len(devices)))
	return nil
}

// handleDeviceRegistered 处理 device.registered 事件 —— 单设备匹配归组。
//
// 注册时设备尚无 site_name（站点名称为配置项），名称匹配此刻无意义 —— 交由
// cron / 心跳路径接力；SN 匹配模式不受影响（payload 带 serial_number）。
func (e *GroupMatchEngine) handleDeviceRegistered(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceID     uuid.UUID `json:"device_id"`
		SerialNumber string    `json:"serial_number"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode device.registered payload: %w", err)
	}
	if payload.DeviceID == uuid.Nil {
		return fmt.Errorf("device.registered payload missing device_id")
	}
	_, err := e.matcher.AssignDeviceToGroup(ctx, MatchRequest{
		DeviceID:     payload.DeviceID,
		SerialNumber: payload.SerialNumber,
	})
	return err
}

// Start 装配 cron @hourly 全量重评估 + device.registered 订阅。幂等。
func (e *GroupMatchEngine) Start(ctx context.Context) error {
	if e.cron == nil {
		e.cron = cron.New()
		if _, err := e.cron.AddFunc("@hourly", func() {
			if err := e.ReEvaluateAll(ctx); err != nil {
				e.logger.Error("group re-evaluate (cron) failed", zap.Error(err))
			}
		}); err != nil {
			e.cron = nil
			return fmt.Errorf("register group-match cron: %w", err)
		}
		e.cron.Start()
		e.logger.Info("group match cron started", zap.String("schedule", "@hourly"))
	}
	if e.eventBus != nil && e.sub == nil {
		sub, err := e.eventBus.QueueSubscribe(
			event.SubjectDeviceRegistered,
			"topology-group-match", // 独立 queue group，与已下线的 device_rules 引擎区分
			func(ctx context.Context, evt event.Event) error {
				return e.handleDeviceRegistered(ctx, evt)
			},
		)
		if err != nil {
			return fmt.Errorf("subscribe device.registered: %w", err)
		}
		e.sub = sub
		e.logger.Info("group match subscribed device.registered",
			zap.String("queue", "topology-group-match"))
	}
	return nil
}

// Stop 优雅停止 cron + 取消事件订阅。幂等。
func (e *GroupMatchEngine) Stop() {
	if e.sub != nil {
		_ = e.sub.Unsubscribe()
		e.sub = nil
	}
	if e.cron != nil {
		c := e.cron.Stop()
		<-c.Done()
		e.cron = nil
	}
}
