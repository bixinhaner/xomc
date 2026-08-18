package deviceaccess

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PolicyVersionStatus string

const (
	PolicyVersionDraft     PolicyVersionStatus = "draft"
	PolicyVersionPublished PolicyVersionStatus = "published"
	PolicyVersionRetired   PolicyVersionStatus = "retired"
)

type PolicyVersion struct {
	ID          string              `json:"id"`
	Carrier     string              `json:"carrier"`
	Name        string              `json:"name"`
	Version     int64               `json:"version"`
	Status      PolicyVersionStatus `json:"status"`
	Policy      CompiledPolicy      `json:"policy"`
	ContentHash string              `json:"content_hash,omitempty"`
	CreatedBy   string              `json:"created_by,omitempty"`
	PublishedBy string              `json:"published_by,omitempty"`
	PublishedAt *time.Time          `json:"published_at,omitempty"`
}

type PolicyActor struct {
	Carrier       string
	SubjectID     string
	VisibleGroups []uuid.UUID
}

type PolicyStore interface {
	CreateDraft(ctx context.Context, version PolicyVersion) (PolicyVersion, error)
	GetVersion(ctx context.Context, versionID string) (PolicyVersion, error)
	Publish(ctx context.Context, versionID, subjectID string) (PolicyVersion, error)
	DeleteDraft(ctx context.Context, versionID string) error
}

type ReevaluationRequest struct {
	Carrier                 string `json:"carrier"`
	SerialNumber            string `json:"serial_number"`
	TriggerType             string `json:"trigger_type"`
	TriggerEventID          string `json:"trigger_event_id,omitempty"`
	ObservedProductClass    string `json:"observed_product_class,omitempty"`
	ObservedSoftwareVersion string `json:"observed_software_version,omitempty"`
}

type ReevaluationQueue interface {
	Enqueue(ctx context.Context, request ReevaluationRequest) error
}

type PolicyService struct {
	store      PolicyStore
	queue      ReevaluationQueue
	identities IdentityVisibilityChecker
}

func NewPolicyService(store PolicyStore, queue ReevaluationQueue) *PolicyService {
	return &PolicyService{store: store, queue: queue}
}

func (s *PolicyService) SetIdentityVisibilityChecker(checker IdentityVisibilityChecker) {
	s.identities = checker
}

func (s *PolicyService) CreateDraft(ctx context.Context, actor PolicyActor, name string, policy CompiledPolicy) (PolicyVersion, error) {
	if err := validatePolicyActor(actor); err != nil {
		return PolicyVersion{}, err
	}
	if strings.TrimSpace(name) == "" {
		return PolicyVersion{}, fmt.Errorf("%w: policy name is required", ErrInvalidAccessInput)
	}
	policy = normalizeCompiledPolicy(policy)
	if err := validateCompiledPolicy(policy); err != nil {
		return PolicyVersion{}, fmt.Errorf("%w: %v", ErrInvalidAccessInput, err)
	}
	policy.VersionID = ""
	payload, err := json.Marshal(policy)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("encode policy draft: %w", err)
	}
	version := PolicyVersion{
		Carrier:     actor.Carrier,
		Name:        strings.TrimSpace(name),
		Status:      PolicyVersionDraft,
		Policy:      policy,
		ContentHash: contentHash(payload),
		CreatedBy:   actor.SubjectID,
	}
	created, err := s.store.CreateDraft(ctx, version)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("create policy draft: %w", err)
	}
	return created, nil
}

func (s *PolicyService) GetVersion(ctx context.Context, actor PolicyActor, versionID string) (PolicyVersion, error) {
	if err := validatePolicyActor(actor); err != nil {
		return PolicyVersion{}, err
	}
	if strings.TrimSpace(versionID) == "" {
		return PolicyVersion{}, fmt.Errorf("%w: policy version id is required", ErrInvalidAccessInput)
	}
	version, err := s.store.GetVersion(ctx, versionID)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("load policy version: %w", err)
	}
	if version.Carrier != actor.Carrier {
		return PolicyVersion{}, fmt.Errorf("get policy version: %w", ErrPolicyCarrierScope)
	}
	return version, nil
}

func (s *PolicyService) Publish(ctx context.Context, actor PolicyActor, versionID string) (PolicyVersion, error) {
	if err := validatePolicyActor(actor); err != nil {
		return PolicyVersion{}, err
	}
	if strings.TrimSpace(versionID) == "" {
		return PolicyVersion{}, fmt.Errorf("%w: policy version id is required", ErrInvalidAccessInput)
	}
	version, err := s.store.GetVersion(ctx, versionID)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("load policy version: %w", err)
	}
	if version.Carrier != actor.Carrier {
		return PolicyVersion{}, fmt.Errorf("publish policy: %w", ErrPolicyCarrierScope)
	}
	if version.Status != PolicyVersionDraft {
		return PolicyVersion{}, fmt.Errorf("publish policy version %s: %w", versionID, ErrPolicyVersionImmutable)
	}
	published, err := s.store.Publish(ctx, versionID, actor.SubjectID)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("publish policy version: %w", err)
	}
	return published, nil
}

// DeleteDraft removes an unpublished version and its rules/conditions. Published
// and retired versions remain immutable audit records.
func (s *PolicyService) DeleteDraft(ctx context.Context, actor PolicyActor, versionID string) error {
	if err := validatePolicyActor(actor); err != nil {
		return err
	}
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return fmt.Errorf("%w: policy version id is required", ErrInvalidAccessInput)
	}
	version, err := s.store.GetVersion(ctx, versionID)
	if err != nil {
		return fmt.Errorf("load policy version: %w", err)
	}
	if version.Carrier != actor.Carrier {
		return fmt.Errorf("delete policy draft: %w", ErrPolicyCarrierScope)
	}
	if version.Status != PolicyVersionDraft {
		return fmt.Errorf("delete policy version %s: %w", versionID, ErrPolicyVersionImmutable)
	}
	if err := s.store.DeleteDraft(ctx, versionID); err != nil {
		return fmt.Errorf("delete policy draft: %w", err)
	}
	return nil
}

func (s *PolicyService) ReevaluateOne(ctx context.Context, actor PolicyActor, serialNumber string) error {
	if err := validatePolicyActor(actor); err != nil {
		return err
	}
	if s.queue == nil {
		return errors.New("reevaluation dependencies are not configured")
	}
	serialNumber = strings.TrimSpace(serialNumber)
	if serialNumber == "" {
		return fmt.Errorf("%w: serial number is required", ErrInvalidAccessInput)
	}
	if err := authorizeIdentityScope(ctx, s.identities, actor, serialNumber); err != nil {
		return err
	}
	if err := s.queue.Enqueue(ctx, ReevaluationRequest{
		Carrier:      actor.Carrier,
		SerialNumber: serialNumber,
		TriggerType:  "manual",
	}); err != nil {
		return fmt.Errorf("enqueue manual device access reevaluation: %w", err)
	}
	return nil
}

var (
	ErrPolicyCarrierScope     = errors.New("policy carrier scope violation")
	ErrPolicyVersionImmutable = errors.New("published policy version is immutable")
	ErrInvalidAccessInput     = errors.New("invalid device access input")
)

func validatePolicyActor(actor PolicyActor) error {
	if strings.TrimSpace(actor.Carrier) == "" {
		return ErrCarrierRequired
	}
	if strings.TrimSpace(actor.SubjectID) == "" {
		return errors.New("policy actor is required")
	}
	return nil
}

func validateCompiledPolicy(policy CompiledPolicy) error {
	if policy.DefaultAction != PolicyDefaultActionReject {
		return fmt.Errorf("unsupported policy default action %q", policy.DefaultAction)
	}
	if len(policy.ListEntries) > 0 {
		return errors.New("policy list entries must be managed through the access-list API")
	}
	ruleIDs := make(map[string]struct{}, len(policy.Rules))
	conditionIDs := make(map[string]struct{})
	for _, rule := range policy.Rules {
		ruleID := strings.TrimSpace(rule.ID)
		if ruleID == "" {
			return errors.New("policy rule id is required")
		}
		if _, exists := ruleIDs[ruleID]; exists {
			return fmt.Errorf("policy rule id %q is duplicated", ruleID)
		}
		ruleIDs[ruleID] = struct{}{}
		if err := validateSerialScope(ruleID, rule.SerialScope); err != nil {
			return err
		}
		if len(rule.Conditions) == 0 {
			return fmt.Errorf("policy rule %s has no conditions", ruleID)
		}
		for _, condition := range rule.Conditions {
			conditionID := strings.TrimSpace(condition.ID)
			if conditionID == "" {
				return fmt.Errorf("policy rule %s has a condition without an id", ruleID)
			}
			if _, exists := conditionIDs[conditionID]; exists {
				return fmt.Errorf("policy condition id %q is duplicated", conditionID)
			}
			conditionIDs[conditionID] = struct{}{}
			if err := validatePolicyCondition(ruleID, condition); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeCompiledPolicy(policy CompiledPolicy) CompiledPolicy {
	if policy.DefaultAction == "" {
		policy.DefaultAction = PolicyDefaultActionReject
	}
	for index := range policy.Rules {
		policy.Rules[index].ID = strings.TrimSpace(policy.Rules[index].ID)
		policy.Rules[index].Name = strings.TrimSpace(policy.Rules[index].Name)
		if policy.Rules[index].Name == "" {
			policy.Rules[index].Name = policy.Rules[index].ID
		}
	}
	return policy
}

func validateSerialScope(ruleID string, scope SerialScope) error {
	switch scope.Type {
	case SerialScopeAll:
		return nil
	case SerialScopeList:
		if len(scope.Values) == 0 {
			return fmt.Errorf("policy rule %s serial list is empty", ruleID)
		}
		seen := make(map[string]struct{}, len(scope.Values))
		for _, value := range scope.Values {
			if value == "" || value != strings.TrimSpace(value) {
				return fmt.Errorf("policy rule %s has an invalid serial number", ruleID)
			}
			if _, exists := seen[value]; exists {
				return fmt.Errorf("policy rule %s serial number %q is duplicated", ruleID, value)
			}
			seen[value] = struct{}{}
		}
		return nil
	case SerialScopePrefix:
		if scope.Prefix == "" || scope.Prefix != strings.TrimSpace(scope.Prefix) {
			return fmt.Errorf("policy rule %s serial prefix is required", ruleID)
		}
		return nil
	case SerialScopeRange:
		if scope.Start == "" || scope.End == "" ||
			scope.Start != strings.TrimSpace(scope.Start) || scope.End != strings.TrimSpace(scope.End) {
			return fmt.Errorf("policy rule %s serial range is incomplete", ruleID)
		}
		if scope.Start > scope.End {
			return fmt.Errorf("policy rule %s serial range start exceeds end", ruleID)
		}
		return nil
	default:
		return fmt.Errorf("policy rule %s has unsupported serial scope %q", ruleID, scope.Type)
	}
}

func validatePolicyCondition(ruleID string, condition CompiledCondition) error {
	conditionID := strings.TrimSpace(condition.ID)
	if condition.EvidenceTTL < 0 {
		return fmt.Errorf("policy condition %s in rule %s has a negative evidence TTL", conditionID, ruleID)
	}
	switch condition.Type {
	case ConditionTypeTAC, ConditionTypeECGI:
		if condition.Operator != ConditionOperatorEqual && condition.Operator != ConditionOperatorIn {
			return unsupportedPolicyOperator(ruleID, condition)
		}
	case ConditionTypeObservedIP:
		if condition.Operator != ConditionOperatorEqual && condition.Operator != ConditionOperatorIn && condition.Operator != ConditionOperatorCIDR {
			return unsupportedPolicyOperator(ruleID, condition)
		}
	case ConditionTypeGPS:
		if condition.Operator != ConditionOperatorWithinRadius {
			return unsupportedPolicyOperator(ruleID, condition)
		}
	default:
		return fmt.Errorf("policy condition %s in rule %s has unsupported type %q", conditionID, ruleID, condition.Type)
	}

	switch condition.Operator {
	case ConditionOperatorEqual:
		if condition.Expected == "" || condition.Expected != strings.TrimSpace(condition.Expected) {
			return fmt.Errorf("policy condition %s in rule %s requires an expected value", conditionID, ruleID)
		}
		if condition.Type == ConditionTypeObservedIP && net.ParseIP(condition.Expected) == nil {
			return fmt.Errorf("policy condition %s in rule %s has an invalid IP address", conditionID, ruleID)
		}
	case ConditionOperatorIn:
		if len(condition.ExpectedAny) == 0 {
			return fmt.Errorf("policy condition %s in rule %s requires expected values", conditionID, ruleID)
		}
		seen := make(map[string]struct{}, len(condition.ExpectedAny))
		for _, value := range condition.ExpectedAny {
			if value == "" || value != strings.TrimSpace(value) {
				return fmt.Errorf("policy condition %s in rule %s has an invalid expected value", conditionID, ruleID)
			}
			if condition.Type == ConditionTypeObservedIP && net.ParseIP(value) == nil {
				return fmt.Errorf("policy condition %s in rule %s has an invalid IP address", conditionID, ruleID)
			}
			if _, exists := seen[value]; exists {
				return fmt.Errorf("policy condition %s in rule %s has duplicate expected value %q", conditionID, ruleID, value)
			}
			seen[value] = struct{}{}
		}
	case ConditionOperatorCIDR:
		if _, _, err := net.ParseCIDR(condition.Expected); err != nil {
			return fmt.Errorf("policy condition %s in rule %s has an invalid CIDR: %w", conditionID, ruleID, err)
		}
	case ConditionOperatorWithinRadius:
		if condition.GeoFence == nil || condition.GeoFence.RadiusMeters <= 0 {
			return fmt.Errorf("policy condition %s in rule %s requires a positive geofence radius", conditionID, ruleID)
		}
		if condition.GeoFence.Center.Latitude < -90 || condition.GeoFence.Center.Latitude > 90 ||
			condition.GeoFence.Center.Longitude < -180 || condition.GeoFence.Center.Longitude > 180 {
			return fmt.Errorf("policy condition %s in rule %s has an invalid geofence center", conditionID, ruleID)
		}
	}
	return nil
}

func unsupportedPolicyOperator(ruleID string, condition CompiledCondition) error {
	return fmt.Errorf(
		"policy condition %s in rule %s does not support operator %q",
		strings.TrimSpace(condition.ID), ruleID, condition.Operator,
	)
}

func contentHash(payload []byte) string {
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
