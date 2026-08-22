package topology

import (
	"context"
	"fmt"
	"sync"

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
// 规则只处理显式源组中的设备，写入时再次原子校验当前归属；源组中的手工归属
// 允许移动，其他组不会被覆盖。多个目标规则同时命中时，最近编辑者优先。
type GroupMatchEngine struct {
	matcher   *DeviceMatcher
	lister    DeviceLister
	groupRepo DeviceGroupRepository
	eventBus  event.EventBus
	logger    *zap.Logger

	cron     *cron.Cron
	sub      event.Subscription // device.registered
	subAttrs event.Subscription // device.attributes.changed

	// runMu 串行化 MatchGroup / ReEvaluateAll 两类全表/子集扫描查询。
	//
	// 背景（生产事故 20260717）：分组批量编辑时 fireGroupMatch 对每个分组各开一个
	// goroutine 并发调 MatchGroup，一旦短时间内编辑的分组较多，就会有几十个几乎
	// 相同的 devices LEFT JOIN device_info LEFT JOIN device_group_members 查询同时
	// 打到 Postgres，互相卡在 LWLock 上排队，把 CPU 打到 4 核封顶。这些调用本身都
	// 是后台异步触发、不阻塞 HTTP 响应，串行化不影响功能正确性，只是把「N 个并发
	// 查询」变成「排队执行」，避免瞬时打满数据库。
	runMu sync.Mutex
}

// NewGroupMatchEngine 构造分组匹配引擎。
func NewGroupMatchEngine(matcher *DeviceMatcher, lister DeviceLister, groupRepo DeviceGroupRepository, logger *zap.Logger) *GroupMatchEngine {
	return &GroupMatchEngine{matcher: matcher, lister: lister, groupRepo: groupRepo, logger: logger}
}

// SetEventBus 注入 EventBus，使 Start 装配 device.registered 订阅。
func (e *GroupMatchEngine) SetEventBus(bus event.EventBus) { e.eventBus = bus }

// MatchGroup 对单个 L2 分组做源组内匹配。未配置源组的存量规则暂停执行，
// 避免继续跨全部设备组搬迁设备。
// 供 DeviceGroupService 在分组新增/编辑后异步调用。
func (e *GroupMatchEngine) MatchGroup(ctx context.Context, groupID uuid.UUID) error {
	e.runMu.Lock()
	defer e.runMu.Unlock()

	group, err := e.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("get group for match: %w", err)
	}
	if group == nil || group.Level != 2 || group.MatchingMode == "" || group.SourceGroupID == nil {
		return nil // 非 L2 或未配匹配规则 —— 无需匹配
	}
	sourceGroup, err := e.groupRepo.GetByID(ctx, *group.SourceGroupID)
	if err != nil {
		return fmt.Errorf("get source group for match: %w", err)
	}
	if sourceGroup == nil || sourceGroup.Level != 2 {
		return nil
	}

	devices, err := e.lister.ListForRuleEval(ctx, *group.SourceGroupID)
	if err != nil {
		return fmt.Errorf("list devices for group match: %w", err)
	}

	matched, moved, skipped := 0, 0, 0
	for _, d := range devices {
		req := MatchRequest{DeviceID: d.ID, DeviceName: d.Name, SerialNumber: d.SerialNumber, LAC: d.LAC, TAC: d.TAC, CurrentGroupID: d.CurrentGroupID}
		ok, mErr := e.matcher.matchGroup(ctx, *group, req)
		if mErr != nil || !ok {
			continue
		}
		matched++
		affected, aErr := e.groupRepo.MoveDeviceAutoMatched(ctx, *group.SourceGroupID, groupID, d.ID)
		if aErr != nil {
			e.logger.Warn("group match: add device failed",
				zap.String("group_id", groupID.String()),
				zap.String("device_id", d.ID.String()),
				zap.Error(aErr))
			continue
		}
		if affected == 0 {
			skipped++
			continue
		}
		moved++
	}
	if moved > 0 {
		if invalidator, ok := e.groupRepo.(deviceGroupTreeCacheInvalidator); ok {
			invalidator.InvalidateDeviceGroupCounts()
		}
	}
	e.logger.Info("group match done",
		zap.String("group_id", groupID.String()),
		zap.String("group_name", group.Name),
		zap.Int("matched", matched),
		zap.Int("moved", moved),
		zap.Int("skipped", skipped),
		zap.Int("scanned", len(devices)))
	return nil
}

// ReEvaluateAll 全量重评估：遍历所有设备，各自归入"最近编辑且命中"的 L2 分组。
// 作为 cron @hourly 兜底，纠正心跳/事件路径可能的遗漏。
func (e *GroupMatchEngine) ReEvaluateAll(ctx context.Context) error {
	e.runMu.Lock()
	defer e.runMu.Unlock()

	devices, err := e.lister.ListAllForRuleEval(ctx)
	if err != nil {
		return fmt.Errorf("list devices for re-evaluate: %w", err)
	}
	reqs := make([]MatchRequest, 0, len(devices))
	for _, d := range devices {
		reqs = append(reqs, MatchRequest{
			DeviceID: d.ID, DeviceName: d.Name, SerialNumber: d.SerialNumber, LAC: d.LAC, TAC: d.TAC, CurrentGroupID: d.CurrentGroupID,
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
	d, err := e.lister.GetByID(ctx, payload.DeviceID)
	if err != nil {
		return fmt.Errorf("get registered device for group match: %w", err)
	}
	if d == nil {
		return nil
	}
	_, err = e.matcher.AssignDeviceToGroup(ctx, MatchRequest{
		DeviceID:       d.ID,
		DeviceName:     d.Name,
		SerialNumber:   d.SerialNumber,
		LAC:            d.LAC,
		TAC:            d.TAC,
		CurrentGroupID: d.CurrentGroupID,
	})
	return err
}

// handleAttributesChanged 处理 device.attributes.changed 事件 —— LAC/TAC 实时归组。
//
// device.registered 仅在首次注册触发且不带 LAC/TAC；后续 Inform 写入 device_info
// 后由 device.DeviceService.PublishDeviceAttributesChangedEvent 发本事件。Payload 只
// 带 device_id + serial_number + changed_fields，订阅方自己从 lister 读最新 LAC/TAC
// 当前值（让数据源始终是 DB，避免事件传递中状态不一致）。
//
// 设备已被 ACS 删除（事件到达时 lister 找不到）时安全 no-op。
func (e *GroupMatchEngine) handleAttributesChanged(ctx context.Context, evt event.Event) error {
	var payload struct {
		DeviceID      uuid.UUID `json:"device_id"`
		SerialNumber  string    `json:"serial_number"`
		ChangedFields []string  `json:"changed_fields"`
	}
	if err := evt.DecodePayload(&payload); err != nil {
		return fmt.Errorf("decode device.attributes.changed payload: %w", err)
	}
	if payload.DeviceID == uuid.Nil {
		return fmt.Errorf("device.attributes.changed payload missing device_id")
	}
	d, err := e.lister.GetByID(ctx, payload.DeviceID)
	if err != nil {
		return fmt.Errorf("get device for match: %w", err)
	}
	if d == nil {
		e.logger.Debug("device.attributes.changed: device gone before re-match",
			zap.String("device_id", payload.DeviceID.String()))
		return nil
	}
	_, err = e.matcher.AssignDeviceToGroup(ctx, MatchRequest{
		DeviceID:       d.ID,
		DeviceName:     d.Name,
		SerialNumber:   d.SerialNumber,
		LAC:            d.LAC,
		TAC:            d.TAC,
		CurrentGroupID: d.CurrentGroupID,
	})
	if err != nil {
		return fmt.Errorf("assign device to group after attrs changed: %w", err)
	}
	e.logger.Info("device.attributes.changed re-matched",
		zap.String("device_id", payload.DeviceID.String()),
		zap.Strings("changed_fields", payload.ChangedFields))
	return nil
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
	if e.eventBus != nil && e.subAttrs == nil {
		sub, err := e.eventBus.QueueSubscribe(
			event.SubjectDeviceAttributesChanged,
			"topology-group-match-attrs", // 独立 queue group，与 registered 分流
			func(ctx context.Context, evt event.Event) error {
				return e.handleAttributesChanged(ctx, evt)
			},
		)
		if err != nil {
			return fmt.Errorf("subscribe device.attributes.changed: %w", err)
		}
		e.subAttrs = sub
		e.logger.Info("group match subscribed device.attributes.changed",
			zap.String("queue", "topology-group-match-attrs"))
	}
	return nil
}

// Stop 优雅停止 cron + 取消事件订阅。幂等。
func (e *GroupMatchEngine) Stop() {
	if e.sub != nil {
		_ = e.sub.Unsubscribe()
		e.sub = nil
	}
	if e.subAttrs != nil {
		_ = e.subAttrs.Unsubscribe()
		e.subAttrs = nil
	}
	if e.cron != nil {
		c := e.cron.Stop()
		<-c.Done()
		e.cron = nil
	}
}
