package notification

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

var (
	ErrDeliveryLeaseInvalid       = errors.New("notification delivery does not hold a valid send lease")
	ErrDeliveryOccurrenceFenced   = errors.New("notification delivery is fenced by occurrence state")
	ErrDeliveryChannelUnavailable = errors.New("notification delivery channel is unavailable")
	ErrDeliveryNotFound           = errors.New("notification delivery not found")
	ErrDeliveryRetryNotAllowed    = errors.New("notification delivery cannot be retried")
)

type AuthorizedDelivery struct {
	DomainDelivery
}

type DeliveryAuthorizationRepository interface {
	AuthorizeSend(context.Context, uuid.UUID, time.Time) (AuthorizedDelivery, error)
}

type DeliveryService struct {
	repository  DeliveryAuthorizationRepository
	queries     DeliveryQueryRepository
	permissions RecipientPermissionResolver
	now         func() time.Time
}

func NewDeliveryService(repository DeliveryAuthorizationRepository) *DeliveryService {
	return &DeliveryService{repository: repository, now: func() time.Time { return time.Now().UTC() }}
}

func (s *DeliveryService) SetQueryRepository(repository DeliveryQueryRepository) {
	s.queries = repository
}

func (s *DeliveryService) SetPermissionResolver(permissions RecipientPermissionResolver) {
	s.permissions = permissions
}

type DeliveryFilter struct {
	OccurrenceID *uuid.UUID
	Channel      string
	FlowState    string
	Limit        int
	Offset       int
	Grants       []model.DeviceVisibilityGrant
}

type DeliveryRecord struct {
	DomainDelivery
	DeviceID    uuid.UUID
	DeviceSN    string
	Technology  string
	Severity    int16
	AlarmStatus string
}

type DeliveryView struct {
	ID                  uuid.UUID  `json:"id"`
	EventID             uuid.UUID  `json:"event_id"`
	OccurrenceID        uuid.UUID  `json:"occurrence_id"`
	DeviceID            uuid.UUID  `json:"device_id"`
	DeviceSN            string     `json:"device_sn"`
	Technology          string     `json:"technology"`
	Severity            int16      `json:"severity"`
	AlarmStatus         string     `json:"alarm_status"`
	RuleVersionID       uuid.UUID  `json:"rule_version_id"`
	TemplateVersionID   uuid.UUID  `json:"template_version_id"`
	Channel             string     `json:"channel"`
	DispatchKind        string     `json:"dispatch_kind"`
	SequenceNo          int        `json:"sequence_no"`
	RecipientType       string     `json:"recipient_type"`
	MaskedAddress       string     `json:"masked_address"`
	FlowState           string     `json:"flow_state"`
	DeliveryResult      string     `json:"delivery_result"`
	SuppressionReason   *string    `json:"suppression_reason,omitempty"`
	MaintenanceWindowID *uuid.UUID `json:"maintenance_window_id,omitempty"`
	FailureReason       *string    `json:"failure_reason,omitempty"`
	OriginDeliveryID    *uuid.UUID `json:"origin_delivery_id,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type DeliveryQueryRepository interface {
	ListDeliveries(context.Context, DeliveryFilter) ([]DeliveryRecord, error)
	GetDelivery(context.Context, uuid.UUID, []model.DeviceVisibilityGrant) (*DeliveryRecord, error)
	ListDeliveryAttempts(context.Context, uuid.UUID) ([]DomainDeliveryAttempt, error)
	RetryDeadLetters(context.Context, []uuid.UUID, []model.DeviceVisibilityGrant, string, string, time.Time) (int, error)
}

func (s *DeliveryService) List(ctx context.Context, userID uuid.UUID, superAdmin bool, filter DeliveryFilter) ([]DeliveryView, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("list notification deliveries: query repository is required")
	}
	grants, err := s.deliveryGrants(ctx, userID, superAdmin)
	if err != nil {
		return nil, err
	}
	filter.Grants = grants
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	records, err := s.queries.ListDeliveries(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list notification deliveries: %w", err)
	}
	views := make([]DeliveryView, 0, len(records))
	for _, record := range records {
		views = append(views, deliveryView(record))
	}
	return views, nil
}

func (s *DeliveryService) Get(ctx context.Context, userID uuid.UUID, superAdmin bool, id uuid.UUID) (DeliveryView, error) {
	grants, err := s.deliveryGrants(ctx, userID, superAdmin)
	if err != nil {
		return DeliveryView{}, err
	}
	record, err := s.queries.GetDelivery(ctx, id, grants)
	if err != nil {
		return DeliveryView{}, fmt.Errorf("get notification delivery: %w", err)
	}
	return deliveryView(*record), nil
}

func (s *DeliveryService) Attempts(ctx context.Context, userID uuid.UUID, superAdmin bool, id uuid.UUID) ([]DomainDeliveryAttempt, error) {
	grants, err := s.deliveryGrants(ctx, userID, superAdmin)
	if err != nil {
		return nil, err
	}
	if _, err := s.queries.GetDelivery(ctx, id, grants); err != nil {
		return nil, fmt.Errorf("authorize notification delivery attempts: %w", err)
	}
	attempts, err := s.queries.ListDeliveryAttempts(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list notification delivery attempts: %w", err)
	}
	return attempts, nil
}

func (s *DeliveryService) Retry(
	ctx context.Context,
	userID uuid.UUID,
	superAdmin bool,
	ids []uuid.UUID,
	reason string,
	actor string,
) (int, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return 0, fmt.Errorf("retry notification deliveries: 1 to 100 delivery IDs are required")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len([]rune(reason)) > 500 {
		return 0, fmt.Errorf("retry notification deliveries: reason is required and must not exceed 500 characters")
	}
	grants, err := s.deliveryGrants(ctx, userID, superAdmin)
	if err != nil {
		return 0, err
	}
	count, err := s.queries.RetryDeadLetters(ctx, ids, grants, reason, actor, s.now())
	if err != nil {
		return 0, fmt.Errorf("retry notification deliveries: %w", err)
	}
	return count, nil
}

func (s *DeliveryService) deliveryGrants(ctx context.Context, userID uuid.UUID, superAdmin bool) ([]model.DeviceVisibilityGrant, error) {
	if s == nil || s.queries == nil || s.permissions == nil || userID == uuid.Nil {
		return nil, fmt.Errorf("authorize notification delivery visibility: dependencies and user ID are required")
	}
	grants, err := s.permissions.GetUserVisibleDeviceGrants(ctx, userID, superAdmin)
	if err != nil {
		return nil, fmt.Errorf("resolve notification delivery visibility grants: %w", err)
	}
	return grants, nil
}

func deliveryView(record DeliveryRecord) DeliveryView {
	return DeliveryView{
		ID: record.ID, EventID: record.EventID, OccurrenceID: record.OccurrenceID,
		DeviceID: record.DeviceID, DeviceSN: record.DeviceSN, Technology: record.Technology,
		Severity: record.Severity, AlarmStatus: record.AlarmStatus, RuleVersionID: record.RuleVersionID,
		TemplateVersionID: record.TemplateVersionID, Channel: record.Channel,
		DispatchKind: record.DispatchKind, SequenceNo: record.SequenceNo, RecipientType: record.RecipientType,
		MaskedAddress: maskedRecipientFingerprint(record.RecipientFingerprint), FlowState: record.FlowState,
		DeliveryResult: record.DeliveryResult, SuppressionReason: record.SuppressionReason,
		MaintenanceWindowID: record.MaintenanceWindowID, FailureReason: record.FailureReason,
		OriginDeliveryID: record.OriginDeliveryID, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

func maskedRecipientFingerprint(fingerprint []byte) string {
	encoded := hex.EncodeToString(fingerprint)
	if len(encoded) > 8 {
		encoded = encoded[len(encoded)-8:]
	}
	if encoded == "" {
		return "recipient-unknown"
	}
	return "recipient-…" + encoded
}

// AuthorizeSend is the last internal gate before any external provider call.
// Claiming and provider I/O remain Task 11 worker responsibilities.
func (s *DeliveryService) AuthorizeSend(ctx context.Context, deliveryID uuid.UUID) (AuthorizedDelivery, error) {
	if s == nil || s.repository == nil {
		return AuthorizedDelivery{}, fmt.Errorf("authorize notification send: repository is required")
	}
	if deliveryID == uuid.Nil {
		return AuthorizedDelivery{}, fmt.Errorf("authorize notification send: delivery ID is required")
	}
	delivery, err := s.repository.AuthorizeSend(ctx, deliveryID, s.now())
	if err != nil {
		return AuthorizedDelivery{}, fmt.Errorf("authorize notification send: %w", err)
	}
	return delivery, nil
}

type deliveryAuthorizationSnapshot struct {
	Delivery             DomainDelivery
	OccurrenceStatus     string
	OccurrenceVersion    int64
	OccurrenceGeneration int64
	ChannelEnabled       bool
	CircuitState         string
}

type deliveryAuthorizationRejection struct {
	flowState string
	reason    string
	err       error
}

func evaluateDeliveryAuthorization(snapshot deliveryAuthorizationSnapshot, now time.Time) *deliveryAuthorizationRejection {
	delivery := snapshot.Delivery
	if delivery.FlowState != "sending" || delivery.LeaseExpiresAt == nil || !delivery.LeaseExpiresAt.After(now) {
		return &deliveryAuthorizationRejection{err: ErrDeliveryLeaseInvalid}
	}
	if !snapshot.ChannelEnabled {
		return &deliveryAuthorizationRejection{flowState: "suppressed", reason: "channel_disabled", err: ErrDeliveryChannelUnavailable}
	}
	if snapshot.CircuitState != "closed" {
		return &deliveryAuthorizationRejection{flowState: "suppressed", reason: "channel_circuit_open", err: ErrDeliveryChannelUnavailable}
	}
	if snapshot.OccurrenceVersion < delivery.OccurrenceVersion ||
		(delivery.DispatchKind != DispatchKindDigest && snapshot.OccurrenceGeneration != delivery.ScheduleGeneration) {
		return &deliveryAuthorizationRejection{flowState: "cancelled", reason: "occurrence_fenced", err: ErrDeliveryOccurrenceFenced}
	}

	validStatus := false
	switch delivery.DispatchKind {
	case DispatchKindInitial, DispatchKindEscalation, DispatchKindRepeat:
		validStatus = snapshot.OccurrenceStatus == "active"
	case DispatchKindRecovery:
		validStatus = snapshot.OccurrenceStatus == "cleared"
	case DispatchKindDigest:
		validStatus = snapshot.OccurrenceStatus == "active" || snapshot.OccurrenceStatus == "cleared"
	}
	if !validStatus {
		return &deliveryAuthorizationRejection{flowState: "cancelled", reason: "occurrence_fenced", err: ErrDeliveryOccurrenceFenced}
	}
	return nil
}
