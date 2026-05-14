package topology

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/omcgo/omcgo/internal/core/event"
	"go.uber.org/zap"
)

// TopoNotifier publishes topology-related events to NATS.
type TopoNotifier struct {
	bus    event.EventBus
	logger *zap.Logger
}

// NewTopoNotifier creates a new TopoNotifier.
func NewTopoNotifier(bus event.EventBus, logger *zap.Logger) *TopoNotifier {
	return &TopoNotifier{
		bus:    bus,
		logger: logger,
	}
}

// TopologyEventType represents the type of topology event.
type TopologyEventType string

const (
	// EventNodeChanged indicates a topology node was created, updated, or deleted.
	EventNodeChanged TopologyEventType = "topology.node.changed"
	// EventEdgeChanged indicates a topology edge was created, updated, or deleted.
	EventEdgeChanged TopologyEventType = "topology.edge.changed"
	// DeviceStatusChanged indicates a device's status changed (affects corresponding topology node).
	EventDeviceStatusChanged TopologyEventType = "topology.device.status_changed"
	// TopologySynced indicates a topology sync operation completed.
	EventTopologySynced TopologyEventType = "topology.synced"
)

// TopologyEvent represents a topology-related event.
type TopologyEvent struct {
	Type      TopologyEventType `json:"type"`
	Timestamp string            `json:"timestamp"`
	Data      interface{}       `json:"data"`
}

// NodeChangedData contains data for node change events.
type NodeChangedData struct {
	NodeID   string          `json:"node_id"`
	ChangeType string         `json:"change_type"` // created, updated, deleted
	Node      *TopoNode       `json:"node,omitempty"`
}

// DeviceStatusChangedData contains data for device status change events.
type DeviceStatusChangedData struct {
	DeviceSN   string    `json:"device_sn"`
	OldStatus  NodeStatus `json:"old_status,omitempty"`
	NewStatus  NodeStatus `json:"new_status"`
	AlarmCount int       `json:"alarm_count,omitempty"`
}

// PublishNodeChange publishes a node change event.
func (n *TopoNotifier) PublishNodeChange(ctx context.Context, nodeID string, changeType string, node *TopoNode) error {
	payload := TopologyEvent{
		Type:      EventNodeChanged,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: NodeChangedData{
			NodeID:     nodeID,
			ChangeType: changeType,
			Node:       node,
		},
	}

	evt, err := event.NewEvent(string(EventNodeChanged), payload)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	if err := n.bus.Publish(ctx, string(EventNodeChanged), evt); err != nil {
		return fmt.Errorf("publish node change: %w", err)
	}

	n.logger.Debug("published node change event",
		zap.String("node_id", nodeID),
		zap.String("change_type", changeType))

	return nil
}

// PublishEdgeChange publishes an edge change event.
func (n *TopoNotifier) PublishEdgeChange(ctx context.Context, edgeID string, changeType string, edge *TopoEdge) error {
	payload := TopologyEvent{
		Type:      EventEdgeChanged,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"edge_id":     edgeID,
			"change_type": changeType,
			"edge":        edge,
		},
	}

	evt, err := event.NewEvent(string(EventEdgeChanged), payload)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	if err := n.bus.Publish(ctx, string(EventEdgeChanged), evt); err != nil {
		return fmt.Errorf("publish edge change: %w", err)
	}

	n.logger.Debug("published edge change event",
		zap.String("edge_id", edgeID),
		zap.String("change_type", changeType))

	return nil
}

// PublishDeviceStatusChange publishes a device status change event.
// This is typically called by the device module when a device's status changes.
func (n *TopoNotifier) PublishDeviceStatusChange(ctx context.Context, deviceSN string, oldStatus, newStatus NodeStatus, alarmCount int) error {
	payload := TopologyEvent{
		Type:      EventDeviceStatusChanged,
		Timestamp: time.Now().Format(time.RFC3339),
		Data: DeviceStatusChangedData{
			DeviceSN:   deviceSN,
			OldStatus:  oldStatus,
			NewStatus:  newStatus,
			AlarmCount: alarmCount,
		},
	}

	evt, err := event.NewEvent(string(EventDeviceStatusChanged), payload)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	if err := n.bus.Publish(ctx, string(EventDeviceStatusChanged), evt); err != nil {
		return fmt.Errorf("publish device status change: %w", err)
	}

	n.logger.Debug("published device status change event",
		zap.String("device_sn", deviceSN),
		zap.String("new_status", string(newStatus)))

	return nil
}

// PublishTopologySynced publishes a topology sync completion event.
func (n *TopoNotifier) PublishTopologySynced(ctx context.Context, result *SyncResult) error {
	payload := TopologyEvent{
		Type:      EventTopologySynced,
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      result,
	}

	evt, err := event.NewEvent(string(EventTopologySynced), payload)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	if err := n.bus.Publish(ctx, string(EventTopologySynced), evt); err != nil {
		return fmt.Errorf("publish topology synced: %w", err)
	}

	n.logger.Info("published topology synced event",
		zap.Int("created", result.Created),
		zap.Int("updated", result.Updated))

	return nil
}

// TopologyEventHandler handles topology events from NATS.
type TopologyEventHandler struct {
	logger   *zap.Logger
	onNodeChange    func(NodeChangedData)
	onEdgeChange    func(map[string]interface{})
	onStatusChange  func(DeviceStatusChangedData)
	onSyncComplete  func(*SyncResult)
}

// NewTopologyEventHandler creates a new topology event handler.
func NewTopologyEventHandler(logger *zap.Logger) *TopologyEventHandler {
	return &TopologyEventHandler{
		logger:  logger,
	}
}

// SetNodeChangeCallback sets the callback for node change events.
func (h *TopologyEventHandler) SetNodeChangeCallback(fn func(NodeChangedData)) {
	h.onNodeChange = fn
}

// SetEdgeChangeCallback sets the callback for edge change events.
func (h *TopologyEventHandler) SetEdgeChangeCallback(fn func(map[string]interface{})) {
	h.onEdgeChange = fn
}

// SetStatusChangeCallback sets the callback for device status change events.
func (h *TopologyEventHandler) SetStatusChangeCallback(fn func(DeviceStatusChangedData)) {
	h.onStatusChange = fn
}

// SetSyncCompleteCallback sets the callback for sync complete events.
func (h *TopologyEventHandler) SetSyncCompleteCallback(fn func(*SyncResult)) {
	h.onSyncComplete = fn
}

// HandleMessage processes an incoming NATS message.
func (h *TopologyEventHandler) HandleMessage(msg *nats.Msg) {
	var evt TopologyEvent
	if err := json.Unmarshal(msg.Data, &evt); err != nil {
		h.logger.Error("failed to unmarshal topology event", zap.Error(err))
		return
	}

	h.logger.Debug("received topology event",
		zap.String("type", string(evt.Type)))

	switch evt.Type {
	case EventNodeChanged:
		if h.onNodeChange != nil {
			data, ok := evt.Data.(map[string]interface{})
			if !ok {
				return
			}
			// Convert to NodeChangedData
			nodeDataJSON, _ := json.Marshal(data)
			var nodeData NodeChangedData
			if err := json.Unmarshal(nodeDataJSON, &nodeData); err == nil {
				h.onNodeChange(nodeData)
			}
		}
	case EventEdgeChanged:
		if h.onEdgeChange != nil {
			if data, ok := evt.Data.(map[string]interface{}); ok {
				h.onEdgeChange(data)
			}
		}
	case EventDeviceStatusChanged:
		if h.onStatusChange != nil {
			data, ok := evt.Data.(map[string]interface{})
			if !ok {
				return
			}
			statusDataJSON, _ := json.Marshal(data)
			var statusData DeviceStatusChangedData
			if err := json.Unmarshal(statusDataJSON, &statusData); err == nil {
				h.onStatusChange(statusData)
			}
		}
	case EventTopologySynced:
		if h.onSyncComplete != nil {
			data, ok := evt.Data.(map[string]interface{})
			if !ok {
				return
			}
			syncDataJSON, _ := json.Marshal(data)
			var syncData SyncResult
			if err := json.Unmarshal(syncDataJSON, &syncData); err == nil {
				h.onSyncComplete(&syncData)
			}
		}
	}
}
