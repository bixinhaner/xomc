package topology

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// DeviceMatcher 设备匹配引擎，根据分组规则自动匹配设备
type DeviceMatcher struct {
	repo   DeviceGroupRepository
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewDeviceMatcher 创建设备匹配引擎
func NewDeviceMatcher(repo DeviceGroupRepository, pool *pgxpool.Pool, logger *zap.Logger) *DeviceMatcher {
	return &DeviceMatcher{
		repo:   repo,
		pool:   pool,
		logger: logger,
	}
}

// MatchRequest 匹配请求参数
type MatchRequest struct {
	DeviceID     uuid.UUID // 设备 ID
	DeviceName   string    // 设备名称 = devices.site_name（"名称匹配"模式的匹配字段）
	SerialNumber string    // 设备序列号 — serialNumber 模式精确匹配（migration 000124）
	LAC          *int      // LAC 位置区码（可选）
	TAC          *int      // TAC 跟踪区码（可选）
}

// MatchResult 匹配结果
type MatchResult struct {
	GroupID   uuid.UUID    // 匹配到的分组 ID
	GroupName string       // 分组名称
	MatchedBy MatchingMode // 匹配方式
}

// MatchDevice 为设备查找匹配的 L2 分组
// 返回第一个匹配的分组，如果没有匹配则返回 nil
func (m *DeviceMatcher) MatchDevice(ctx context.Context, req MatchRequest) (*MatchResult, error) {
	// 获取所有有匹配规则的 L2 分组
	groups, err := m.repo.GetTreeWithCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get groups: %w", err)
	}

	// 收集所有配了匹配规则的 L2 分组，按 updated_at 降序排列 —— 最近新增/编辑的
	// 分组优先匹配，首个命中即胜，实现"分组最后新增或编辑为优先"。
	var l2Groups []DeviceGroup
	for _, l1Group := range groups {
		for _, g := range l1Group.Children {
			if g.MatchingMode != "" {
				l2Groups = append(l2Groups, g)
			}
		}
	}
	sort.SliceStable(l2Groups, func(i, j int) bool {
		return l2Groups[i].UpdatedAt.After(l2Groups[j].UpdatedAt)
	})

	for _, l2Group := range l2Groups {
		matched, err := m.matchGroup(ctx, l2Group, req)
		if err != nil {
			m.logger.Warn("match group failed",
				zap.String("group_id", l2Group.ID.String()),
				zap.String("group_name", l2Group.Name),
				zap.Error(err),
			)
			continue
		}

		if matched {
			return &MatchResult{
				GroupID:   l2Group.ID,
				GroupName: l2Group.Name,
				MatchedBy: l2Group.MatchingMode,
			}, nil
		}
	}

	return nil, nil
}

// matchGroup 检查设备是否匹配指定分组
func (m *DeviceMatcher) matchGroup(ctx context.Context, group DeviceGroup, req MatchRequest) (bool, error) {
	switch group.MatchingMode {
	case MatchingModeDeviceName:
		return m.matchByDeviceName(group.NameRuleList, req.DeviceName), nil
	case MatchingModeLAC:
		if req.LAC == nil {
			return false, nil
		}
		return m.matchByCode(group.LACList, *req.LAC), nil
	case MatchingModeTAC:
		if req.TAC == nil {
			return false, nil
		}
		return m.matchByCode(group.TACList, *req.TAC), nil
	case MatchingModeSerialNumber:
		if req.SerialNumber == "" {
			return false, nil
		}
		return m.matchBySerialNumber(group.SerialNumberList, req.SerialNumber), nil
	default:
		return false, nil
	}
}

// matchBySerialNumber 序列号精确成员匹配（migration 000124）。
// 设计为简单线性扫描——SN 列表预期百级以内，list lookup 比建 map 还快。
func (m *DeviceMatcher) matchBySerialNumber(list []string, sn string) bool {
	for _, item := range list {
		if item == sn {
			return true
		}
	}
	return false
}

// matchByDeviceName 根据设备名称规则匹配
// 支持多条件 AND/OR 逻辑
func (m *DeviceMatcher) matchByDeviceName(rules []NameRule, deviceName string) bool {
	if len(rules) == 0 || deviceName == "" {
		return false
	}

	// 将规则按 OR 分组
	orGroups := [][]NameRule{{}}
	for _, rule := range rules {
		if rule.AndOr == "or" {
			orGroups = append(orGroups, []NameRule{rule})
		} else {
			orGroups[len(orGroups)-1] = append(orGroups[len(orGroups)-1], rule)
		}
	}

	// 任一 OR 组匹配即成功
	for _, orGroup := range orGroups {
		if m.matchAndGroup(orGroup, deviceName) {
			return true
		}
	}

	return false
}

// matchAndGroup 匹配 AND 组内的所有条件
func (m *DeviceMatcher) matchAndGroup(rules []NameRule, deviceName string) bool {
	for _, rule := range rules {
		if !m.matchCondition(rule.Condition, rule.Value, deviceName) {
			return false
		}
	}
	return len(rules) > 0
}

// matchCondition 匹配单个条件
func (m *DeviceMatcher) matchCondition(condition, pattern, value string) bool {
	if pattern == "" || value == "" {
		return false
	}

	switch condition {
	case "contain", "contains":
		return strings.Contains(value, pattern)
	case "notContain", "notContains":
		return !strings.Contains(value, pattern)
	case "startWith", "startsWith":
		return strings.HasPrefix(value, pattern)
	case "endWith", "endsWith":
		return strings.HasSuffix(value, pattern)
	default:
		return false
	}
}

// matchByCode 根据 LAC/TAC 列表匹配
func (m *DeviceMatcher) matchByCode(codes []int, target int) bool {
	for _, code := range codes {
		if code == target {
			return true
		}
	}
	return false
}

// AssignDeviceToGroup 将设备分配到匹配的分组
// 如果没有匹配的分组，设备将保持原分组不变
func (m *DeviceMatcher) AssignDeviceToGroup(ctx context.Context, req MatchRequest) (*MatchResult, error) {
	result, err := m.MatchDevice(ctx, req)
	if err != nil {
		return nil, err
	}

	if result == nil {
		// 没有匹配的分组
		return nil, nil
	}

	// 将设备添加到匹配的分组（自动匹配：无条件覆盖，含手工分配）
	if err := m.repo.AddDeviceAutoMatched(ctx, result.GroupID, req.DeviceID); err != nil {
		return nil, fmt.Errorf("add device to group: %w", err)
	}

	m.logger.Info("device assigned to group",
		zap.String("device_id", req.DeviceID.String()),
		zap.String("group_id", result.GroupID.String()),
		zap.String("group_name", result.GroupName),
		zap.String("matched_by", string(result.MatchedBy)),
	)

	return result, nil
}

// HeartbeatAssigner 是 DeviceMatcher 的轻量 wrapper，用于 device 包的
// "心跳后自动分组"钩子（device.GroupAssigner 接口）。
//
// 仅暴露一个方法 AssignDeviceToGroup(ctx, deviceID/sn/name/lac/tac)，结构
// 字段刻意对齐 device.GroupAssignRequest，让 wiring 一行 caller 即可。
type HeartbeatAssigner struct {
	matcher *DeviceMatcher
}

// NewHeartbeatAssigner 构造心跳路径分组适配器。
func NewHeartbeatAssigner(m *DeviceMatcher) *HeartbeatAssigner {
	return &HeartbeatAssigner{matcher: m}
}

// HeartbeatRequest — 与 device.GroupAssignRequest 字段对齐的小 DTO。
type HeartbeatRequest struct {
	DeviceID     uuid.UUID
	DeviceName   string
	SerialNumber string
	LAC          *int
	TAC          *int
}

// AssignByHeartbeat — 与 device.GroupAssigner 接口几乎同名同参；caller 在
// modules.go 里用一个 closure 把 device.GroupAssignRequest 转换为 MatchRequest。
//
// 此方法保留供其他 caller 直接使用；device 包用 closure 适配避免循环依赖。
func (a *HeartbeatAssigner) AssignByHeartbeat(ctx context.Context, req HeartbeatRequest) error {
	if a.matcher == nil {
		return nil
	}
	_, err := a.matcher.AssignDeviceToGroup(ctx, MatchRequest{
		DeviceID:     req.DeviceID,
		DeviceName:   req.DeviceName,
		SerialNumber: req.SerialNumber,
		LAC:          req.LAC,
		TAC:          req.TAC,
	})
	return err
}

// BatchMatchDevices 批量匹配设备
// 用于分组规则变更后重新匹配所有设备
func (m *DeviceMatcher) BatchMatchDevices(ctx context.Context, devices []MatchRequest) (int, error) {
	var matched int
	for _, device := range devices {
		result, err := m.AssignDeviceToGroup(ctx, device)
		if err != nil {
			m.logger.Warn("batch match device failed",
				zap.String("device_id", device.DeviceID.String()),
				zap.Error(err),
			)
			continue
		}
		if result != nil {
			matched++
		}
	}
	return matched, nil
}
