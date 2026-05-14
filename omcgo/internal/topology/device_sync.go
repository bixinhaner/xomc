package topology

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// DeviceSyncService synchronizes devices to topology nodes.
type DeviceSyncService struct {
	pool        *pgxpool.Pool
	nodeRepo    TopoNodeRepository
	edgeRepo    TopoEdgeRepository
	logger      *zap.Logger
}

// NewDeviceSyncService creates a new DeviceSyncService.
func NewDeviceSyncService(pool *pgxpool.Pool, nodeRepo TopoNodeRepository, edgeRepo TopoEdgeRepository, logger *zap.Logger) *DeviceSyncService {
	return &DeviceSyncService{
		pool:     pool,
		nodeRepo: nodeRepo,
		edgeRepo: edgeRepo,
		logger:   logger,
	}
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
		existingNode.Label = device.SiteName
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
		Label:     device.SiteName,
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
		       d.ip_address, d.site_name, d.site_id, d.latitude, d.longitude,
		       d.created_at, d.updated_at
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
		var lat, lng *float64

		err := rows.Scan(
			&d.ID, &d.SerialNumber, &d.OUI, &d.ProductClass, &d.Manufacturer,
			&d.ModelName, &d.Carrier, &d.Technology, &d.Status, &d.FirmwareVersion,
			&d.IPAddress, &d.SiteName, &d.SiteID, &lat, &lng,
			&d.CreatedAt, &d.UpdatedAt,
			&lastInformAt, &lastBootAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}

		d.LastInformAt = lastInformAt
		d.LastBootAt = lastBootAt
		if lat != nil {
			d.Latitude = *lat
		}
		if lng != nil {
			d.Longitude = *lng
		}

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
