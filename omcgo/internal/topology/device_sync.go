package topology

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// DeviceSyncServiceConfig 配置设备同步服务行为。
type DeviceSyncServiceConfig struct {
	Enabled          bool          // 是否启用自动同步
	InitialSync      bool          // 启动时是否执行全量同步
	FallbackInterval time.Duration // 兜底定时同步间隔，0 表示不启用
	InitialSyncDelay time.Duration // 启动同步延迟，避免启动高峰
	BatchSize        int           // 批量同步大小
}

// DefaultDeviceSyncServiceConfig 返回默认配置。
func DefaultDeviceSyncServiceConfig() DeviceSyncServiceConfig {
	return DeviceSyncServiceConfig{
		Enabled:          true,
		InitialSync:      true,
		FallbackInterval: 1 * time.Hour,
		InitialSyncDelay: 10 * time.Second,
		BatchSize:        100,
	}
}

// DBPool 是 DeviceSyncService 所需的最小数据库访问接口。
// *pgxpool.Pool 天然满足该接口；抽成接口便于单元测试注入 fake。
type DBPool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// DeviceSyncService synchronizes devices to topology nodes.
// 支持事件驱动同步、启动时同步和定时兜底同步三种模式。
type DeviceSyncService struct {
	pool         DBPool
	nodeRepo     TopoNodeRepository
	edgeRepo     TopoEdgeRepository
	eventBus     event.EventBus // 事件总线，用于订阅设备注册事件
	logger       *zap.Logger
	config       DeviceSyncServiceConfig
	subscription event.Subscription // 事件订阅句柄
	cronTicker   *time.Ticker       // 定时同步 ticker
	stopCh       chan struct{}      // 停止信号
}

// NewDeviceSyncService creates a new DeviceSyncService.
func NewDeviceSyncService(pool DBPool, nodeRepo TopoNodeRepository, edgeRepo TopoEdgeRepository, eventBus event.EventBus, logger *zap.Logger) *DeviceSyncService {
	return &DeviceSyncService{
		pool:     pool,
		nodeRepo: nodeRepo,
		edgeRepo: edgeRepo,
		eventBus: eventBus,
		logger:   logger,
		config:   DefaultDeviceSyncServiceConfig(),
		stopCh:   make(chan struct{}),
	}
}

// SetConfig 设置同步服务配置。
func (s *DeviceSyncService) SetConfig(config DeviceSyncServiceConfig) {
	s.config = config
}

// Start 启动设备同步服务。
// 订阅设备注册事件，启动兜底定时同步，执行初始全量同步。
func (s *DeviceSyncService) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("device sync service disabled")
		return nil
	}

	s.logger.Info("starting device sync service",
		zap.Bool("initial_sync", s.config.InitialSync),
		zap.Duration("fallback_interval", s.config.FallbackInterval),
		zap.Int("batch_size", s.config.BatchSize))

	// 1. 订阅设备注册事件
	if s.eventBus != nil {
		if err := s.subscribeDeviceEvents(ctx); err != nil {
			return fmt.Errorf("subscribe device events: %w", err)
		}
	} else {
		s.logger.Warn("event bus not available, event-driven sync disabled")
	}

	// 2. 启动兜底定时同步
	if s.config.FallbackInterval > 0 {
		s.startFallbackSync(ctx)
	}

	// 3. 执行初始全量同步（异步延迟执行）
	if s.config.InitialSync {
		go func() {
			// 使用 Timer 而非 time.After，确保可以及时释放资源
			timer := time.NewTimer(s.config.InitialSyncDelay)
			defer timer.Stop()

			select {
			case <-timer.C:
				s.logger.Info("executing initial topology sync")
				result, err := s.SyncFromDevices(ctx, nil, 0)
				if err != nil {
					s.logger.Error("initial sync failed", zap.Error(err))
				} else {
					s.logger.Info("initial sync completed",
						zap.Int("created", result.Created),
						zap.Int("updated", result.Updated),
						zap.Int("failed", result.Failed),
						zap.Int("total_processed", result.TotalProcessed))
				}
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			}
		}()
	}

	return nil
}

// Stop 停止设备同步服务。
// 可安全地多次调用。
func (s *DeviceSyncService) Stop() {
	// 使用 select 避免重复关闭 channel 导致 panic
	select {
	case <-s.stopCh:
		// 已经停止，直接返回
		return
	default:
		close(s.stopCh)
	}

	if s.subscription != nil {
		if err := s.subscription.Unsubscribe(); err != nil {
			s.logger.Error("unsubscribe device events failed", zap.Error(err))
		}
		s.subscription = nil
	}

	if s.cronTicker != nil {
		s.cronTicker.Stop()
		s.cronTicker = nil
	}

	s.logger.Info("device sync service stopped")
}

// subscribeDeviceEvents 订阅设备相关事件。
func (s *DeviceSyncService) subscribeDeviceEvents(ctx context.Context) error {
	// 订阅设备注册事件
	sub, err := s.eventBus.QueueSubscribe(
		event.SubjectDeviceRegistered,
		"topology-device-sync", // NATS Queue group，多实例负载均衡
		func(evtCtx context.Context, evt event.Event) error {
			return s.handleDeviceRegistered(evtCtx, evt)
		},
	)
	if err != nil {
		return fmt.Errorf("subscribe device.registered: %w", err)
	}
	s.subscription = sub

	s.logger.Info("subscribed to device events",
		zap.String("subject", event.SubjectDeviceRegistered),
		zap.String("queue", "topology-device-sync"))

	return nil
}

// handleDeviceRegistered 处理设备注册事件。
// 当新设备注册时，自动创建对应的拓扑节点。
func (s *DeviceSyncService) handleDeviceRegistered(ctx context.Context, evt event.Event) error {
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

	s.logger.Debug("device registered, syncing to topology",
		zap.String("device_id", payload.DeviceID.String()),
		zap.String("serial_number", payload.SerialNumber))

	// 获取设备详情
	device, err := s.getDeviceBySN(ctx, payload.SerialNumber)
	if err != nil {
		s.logger.Error("failed to get device for topology sync",
			zap.String("serial_number", payload.SerialNumber),
			zap.Error(err))
		return nil // 不返回错误，避免重试风暴
	}

	// 检查节点是否已存在
	existingNodes, err := s.nodeRepo.ListAll(ctx, nil, nil, nil)
	if err != nil {
		s.logger.Error("failed to list existing nodes", zap.Error(err))
		return nil
	}

	nodeByDeviceSN := make(map[string]*TopoNode)
	for i := range existingNodes {
		if existingNodes[i].DeviceSN != "" {
			nodeByDeviceSN[existingNodes[i].DeviceSN] = &existingNodes[i]
		}
	}

	// 同步设备到拓扑节点
	result := &SyncResult{}
	if err := s.syncDevice(ctx, device, nodeByDeviceSN, result); err != nil {
		s.logger.Error("failed to sync device to topology",
			zap.String("serial_number", payload.SerialNumber),
			zap.Error(err))
		return nil
	}

	s.logger.Info("device synced to topology",
		zap.String("serial_number", payload.SerialNumber),
		zap.String("action", map[bool]string{true: "created", false: "updated"}[result.Created > 0]))

	return nil
}

// startFallbackSync 启动兜底定时同步。
func (s *DeviceSyncService) startFallbackSync(ctx context.Context) {
	s.cronTicker = time.NewTicker(s.config.FallbackInterval)

	go func() {
		for {
			select {
			case <-s.cronTicker.C:
				s.logger.Info("executing fallback topology sync")
				result, err := s.SyncFromDevices(ctx, nil, 0)
				if err != nil {
					s.logger.Error("fallback sync failed", zap.Error(err))
				} else {
					s.logger.Info("fallback sync completed",
						zap.Int("created", result.Created),
						zap.Int("updated", result.Updated),
						zap.Int("failed", result.Failed))
				}
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			}
		}
	}()

	s.logger.Info("fallback sync started",
		zap.Duration("interval", s.config.FallbackInterval))
}

// getDeviceBySN 根据序列号获取设备。
func (s *DeviceSyncService) getDeviceBySN(ctx context.Context, serialNumber string) (model.Device, error) {
	query := `
		SELECT d.id, d.serial_number, d.oui, d.product_class, d.manufacturer,
		       d.model_name, d.carrier, d.technology, d.status, d.firmware_version,
		       d.ip_address::text, d.site_name, COALESCE(d.site_id, '')::text,
		       d.created_at, d.updated_at, d.last_inform_at, d.last_boot_at
		FROM devices d
		WHERE d.serial_number = $1 AND d.deleted_at IS NULL
	`

	var d model.Device
	var lastInformAt, lastBootAt *time.Time

	err := s.pool.QueryRow(ctx, query, serialNumber).Scan(
		&d.ID, &d.SerialNumber, &d.OUI, &d.ProductClass, &d.Manufacturer,
		&d.ModelName, &d.Carrier, &d.Technology, &d.Status, &d.FirmwareVersion,
		&d.IPAddress, &d.DeviceName, &d.SiteID,
		&d.CreatedAt, &d.UpdatedAt,
		&lastInformAt, &lastBootAt,
	)

	if err != nil {
		return model.Device{}, fmt.Errorf("query device: %w", err)
	}

	d.LastInformAt = lastInformAt
	d.LastBootAt = lastBootAt

	return d, nil
}

// SyncFromDevices synchronizes devices to topology nodes for a given domain.
// It creates new nodes for devices that don't have corresponding topology nodes,
// and updates existing nodes with current device status.
func (s *DeviceSyncService) SyncFromDevices(ctx context.Context, domainID *uuid.UUID, limit int) (*SyncResult, error) {
	s.logger.Info("starting device to topology sync", zap.Any("domain_id", domainID))

	// Query devices
	devices, err := s.queryDevices(ctx, domainID, limit)
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}

	s.logger.Info("found devices for sync", zap.Int("count", len(devices)))

	result := &SyncResult{
		TotalProcessed: len(devices),
		Created:        0,
		Updated:        0,
		Failed:         0,
	}

	// Get existing nodes to avoid duplicates
	existingNodes, err := s.nodeRepo.ListAll(ctx, domainID, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("list existing nodes: %w", err)
	}

	nodeByDeviceSN := make(map[string]*TopoNode)
	for i := range existingNodes {
		if existingNodes[i].DeviceSN != "" {
			nodeByDeviceSN[existingNodes[i].DeviceSN] = &existingNodes[i]
		}
	}

	// Sync each device
	for _, device := range devices {
		if err := s.syncDevice(ctx, device, nodeByDeviceSN, result); err != nil {
			s.logger.Error("failed to sync device",
				zap.String("serial_number", device.SerialNumber),
				zap.Error(err))
			result.Failed++
		}
	}

	s.logger.Info("device to topology sync completed",
		zap.Int("created", result.Created),
		zap.Int("updated", result.Updated),
		zap.Int("failed", result.Failed))

	return result, nil
}

// SyncResult summarizes the sync operation.
type SyncResult struct {
	TotalProcessed int    `json:"total_processed"`
	Created        int    `json:"created"`
	Updated        int    `json:"updated"`
	Failed         int    `json:"failed"`
	Message        string `json:"message,omitempty"`
}

// syncDevice syncs a single device to a topology node.
func (s *DeviceSyncService) syncDevice(ctx context.Context, device model.Device, nodeByDeviceSN map[string]*TopoNode, result *SyncResult) error {
	// Determine node type from device characteristics
	nodeType := inferNodeType(device)

	// Determine node status from device status
	nodeStatus := mapDeviceStatusToNodeStatus(device.Status)

	// Check if node already exists
	if existingNode, ok := nodeByDeviceSN[device.SerialNumber]; ok {
		// Update existing node
		existingNode.Label = device.DeviceName
		if existingNode.Label == "" {
			existingNode.Label = device.SerialNumber
		}
		existingNode.NodeType = nodeType
		existingNode.Status = nodeStatus

		if err := s.nodeRepo.Update(ctx, existingNode); err != nil {
			return fmt.Errorf("update node: %w", err)
		}
		result.Updated++
		return nil
	}

	// Create new node
	var siteID *uuid.UUID
	if device.SiteID != "" {
		if id, err := uuid.Parse(device.SiteID); err == nil {
			siteID = &id
		}
	}

	// Generate initial position (will be overridden by layout algorithm)
	x, y := 500.0, 500.0

	node := &TopoNode{
		ID:        uuid.New(),
		Label:     device.DeviceName,
		NodeType:  nodeType,
		X:         x,
		Y:         y,
		Status:    nodeStatus,
		DeviceSN:  device.SerialNumber,
		SiteID:    siteID,
		DomainID:  nil, // Will be set by filter if needed
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if node.Label == "" {
		node.Label = device.SerialNumber
	}

	if err := s.nodeRepo.Create(ctx, node); err != nil {
		return fmt.Errorf("create node: %w", err)
	}

	result.Created++

	// Update map
	nodeByDeviceSN[device.SerialNumber] = node

	return nil
}

// queryDevices queries devices from the database.
func (s *DeviceSyncService) queryDevices(ctx context.Context, domainID *uuid.UUID, limit int) ([]model.Device, error) {
	query := `
		SELECT d.id, d.serial_number, d.oui, d.product_class, d.manufacturer,
		       d.model_name, d.carrier, d.technology, d.status, d.firmware_version,
		       d.ip_address::text, d.site_name, COALESCE(d.site_id, '')::text,
		       d.created_at, d.updated_at, d.last_inform_at, d.last_boot_at
		FROM devices d
		WHERE d.deleted_at IS NULL
	`

	args := []interface{}{}
	argIdx := 1

	if domainID != nil {
		query += fmt.Sprintf(" AND d.id IN (SELECT device_id FROM device_group_members WHERE group_id = $%d)", argIdx)
		args = append(args, domainID)
		argIdx++
	}

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
	}

	query += " ORDER BY d.created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		var d model.Device
		var lastInformAt, lastBootAt *time.Time

		err := rows.Scan(
			&d.ID, &d.SerialNumber, &d.OUI, &d.ProductClass, &d.Manufacturer,
			&d.ModelName, &d.Carrier, &d.Technology, &d.Status, &d.FirmwareVersion,
			&d.IPAddress, &d.DeviceName, &d.SiteID,
			&d.CreatedAt, &d.UpdatedAt,
			&lastInformAt, &lastBootAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}

		d.LastInformAt = lastInformAt
		d.LastBootAt = lastBootAt
		devices = append(devices, d)
	}

	return devices, rows.Err()
}

// inferNodeType determines the topology node type based on device characteristics.
// This follows telecom industry best practices for device classification.
func inferNodeType(device model.Device) string {
	// Priority: explicit model classification > technology > product class patterns > defaults

	// Check for explicit eNodeB/gNodeB patterns in product class
	productClass := device.ProductClass
	switch {
	case containsAny(productClass, []string{"eNB", "enb", "eNodeB", "LTE", "TD-LTE", "FDD-LTE"}):
		return "eNB"
	case containsAny(productClass, []string{"gNB", "gnb", "gNodeB", "NR", "5G", "SA", "NSA"}):
		return "gNB"
	case containsAny(productClass, []string{"CPE", "cpe", "Home", "Residential", "Indoor"}):
		return "CPE"
	case containsAny(productClass, []string{"EGW", "egw", "Gateway", "EPC", "Core"}):
		return "eGW"
	}

	// Fallback to technology
	switch device.Technology {
	case model.TechLTE:
		return "eNB"
	case model.TechNR:
		return "gNB"
	}

	// Default to CPE for small cells
	return "CPE"
}

// mapDeviceStatusToNodeStatus converts device status to topology node status.
func mapDeviceStatusToNodeStatus(status model.DeviceStatus) NodeStatus {
	switch status {
	case model.DeviceActive:
		return NodeOnline
	case model.DeviceOffline:
		return NodeOffline
	case model.DeviceMaintenance:
		return NodeMaintenance
	default:
		return NodeOffline
	}
}

// containsAny checks if the string contains any of the substrings (case-insensitive).
func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if containsIgnoreCase(s, sub) {
			return true
		}
	}
	return false
}

// containsIgnoreCase is a simple case-insensitive contains check.
func containsIgnoreCase(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			subc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if subc >= 'A' && subc <= 'Z' {
				subc += 32
			}
			if sc != subc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// CalculateStatistics computes topology statistics from nodes and edges.
func CalculateStatistics(nodes []TopoNode, edges []TopoEdge) *TopoStatistics {
	stats := &TopoStatistics{
		TotalNodes:     len(nodes),
		NodeTypeCounts: make(map[string]int),
	}

	for _, node := range nodes {
		stats.NodeTypeCounts[node.NodeType]++
		switch node.Status {
		case NodeOnline:
			stats.OnlineNodes++
		case NodeOffline:
			stats.OfflineNodes++
		case NodeAlarm:
			stats.AlarmNodes++
		case NodeMaintenance:
			stats.MaintenanceNodes++
		}
	}

	stats.TotalEdges = len(edges)
	for _, edge := range edges {
		switch edge.Status {
		case EdgeActive:
			stats.ActiveEdges++
		case EdgeInactive:
			stats.InactiveEdges++
		case EdgeDegraded:
			stats.DegradedEdges++
		}
	}

	return stats
}
