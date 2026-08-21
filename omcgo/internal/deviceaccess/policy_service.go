package deviceaccess

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
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
	Username      string
	VisibleGroups []uuid.UUID
}

type PolicyStore interface {
	CreateDraft(ctx context.Context, version PolicyVersion) (PolicyVersion, error)
	UpdateDraft(ctx context.Context, version PolicyVersion) (PolicyVersion, error)
	GetVersion(ctx context.Context, versionID string) (PolicyVersion, error)
	Publish(ctx context.Context, versionID, subjectID string) (PolicyVersion, error)
	DeleteDraft(ctx context.Context, versionID string) error
}

type PolicyHistoryStore interface {
	PreviousVersion(ctx context.Context, versionID string) (PolicyVersion, error)
}

func (s *PolicyService) UpdateDraft(ctx context.Context, actor PolicyActor, versionID string, policy CompiledPolicy) (PolicyVersion, error) {
	if err := validatePolicyActor(actor); err != nil {
		return PolicyVersion{}, err
	}
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return PolicyVersion{}, fmt.Errorf("%w: policy version id is required", ErrInvalidAccessInput)
	}
	current, err := s.store.GetVersion(ctx, versionID)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("load policy draft: %w", err)
	}
	if current.Carrier != actor.Carrier {
		return PolicyVersion{}, ErrPolicyCarrierScope
	}
	if current.Status != PolicyVersionDraft {
		return PolicyVersion{}, ErrPolicyVersionImmutable
	}
	policy = normalizeCompiledPolicy(policy)
	if err := validateCompiledPolicy(policy); err != nil {
		return PolicyVersion{}, fmt.Errorf("%w: %v", ErrInvalidAccessInput, err)
	}
	policy = normalizeUpdatedDraftEntityIDs(policy, current.Policy)
	policy.VersionID = ""
	payload, err := json.Marshal(policy)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("encode policy draft update: %w", err)
	}
	current.Policy = policy
	current.ContentHash = contentHash(payload)
	updated, err := s.store.UpdateDraft(ctx, current)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("update policy draft: %w", err)
	}
	return updated, nil
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
	policy = regenerateDraftEntityIDs(policy)
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
	policy := normalizeCompiledPolicy(version.Policy)
	if err := validateCompiledPolicy(policy); err != nil {
		return PolicyVersion{}, fmt.Errorf("%w: %v", ErrInvalidAccessInput, err)
	}
	published, err := s.store.Publish(ctx, versionID, actor.SubjectID)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("publish policy version: %w", err)
	}
	return published, nil
}

func (s *PolicyService) Difference(ctx context.Context, actor PolicyActor, targetVersionID, baseVersionID string) (PolicyDifference, error) {
	target, err := s.GetVersion(ctx, actor, targetVersionID)
	if err != nil {
		return PolicyDifference{}, err
	}
	var base PolicyVersion
	if strings.TrimSpace(baseVersionID) != "" {
		base, err = s.GetVersion(ctx, actor, baseVersionID)
	} else {
		history, ok := s.store.(PolicyHistoryStore)
		if !ok {
			return PolicyDifference{}, fmt.Errorf("load previous policy version: %w", ErrAccessGateDependencyMissing)
		}
		base, err = history.PreviousVersion(ctx, target.ID)
		if err == nil && base.Carrier != actor.Carrier {
			err = ErrPolicyCarrierScope
		}
	}
	if err != nil {
		return PolicyDifference{}, fmt.Errorf("load policy difference baseline: %w", err)
	}
	return comparePolicyVersions(base, target), nil
}

// Rollback publishes historical content as a new immutable version. The source
// version is never edited. If publication fails, the newly-created draft is
// intentionally retained so the operator can inspect or retry it.
func (s *PolicyService) Rollback(ctx context.Context, actor PolicyActor, sourceVersionID string) (PolicyVersion, error) {
	source, err := s.GetVersion(ctx, actor, sourceVersionID)
	if err != nil {
		return PolicyVersion{}, err
	}
	if source.Status != PolicyVersionRetired {
		return PolicyVersion{}, fmt.Errorf("rollback policy version %s: %w", sourceVersionID, ErrPolicyRollbackSource)
	}
	draft, err := s.CreateDraft(ctx, actor, source.Name, source.Policy)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("create policy rollback draft: %w", err)
	}
	if _, err := s.Publish(ctx, actor, draft.ID); err != nil {
		return PolicyVersion{}, fmt.Errorf("publish policy rollback draft %s: %w", draft.ID, err)
	}
	published, err := s.GetVersion(ctx, actor, draft.ID)
	if err != nil {
		return PolicyVersion{}, fmt.Errorf("reload published rollback version: %w", err)
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
		TriggerType:  TriggerManualReevaluation,
	}); err != nil {
		return fmt.Errorf("enqueue manual device access reevaluation: %w", err)
	}
	return nil
}

var (
	ErrPolicyCarrierScope     = errors.New("policy carrier scope violation")
	ErrPolicyVersionImmutable = errors.New("published policy version is immutable")
	ErrPolicyRollbackSource   = errors.New("policy rollback source must be retired")
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
		return fmt.Errorf("policy default action must be %q", PolicyDefaultActionReject)
	}
	if policy.FailureMode != FailureModeFailClosed {
		return fmt.Errorf("policy failure mode must be %q", FailureModeFailClosed)
	}
	if policy.CollectionTimeout < time.Minute || policy.CollectionTimeout > 24*time.Hour {
		return fmt.Errorf("policy collection timeout must be between 1 minute and 24 hours")
	}
	if len(policy.ListEntries) > 0 {
		return errors.New("policy list entries must be managed through the access-list API")
	}
	if len(policy.BypassProfiles) > 100 {
		return errors.New("policy has more than 100 bypass profiles")
	}
	profileIDs := make(map[string]struct{}, len(policy.BypassProfiles))
	profilePriorities := make(map[int]string, len(policy.BypassProfiles))
	for _, profile := range policy.BypassProfiles {
		if _, err := uuid.Parse(profile.ID); err != nil {
			return fmt.Errorf("bypass profile id %q is invalid", profile.ID)
		}
		if _, exists := profileIDs[profile.ID]; exists {
			return fmt.Errorf("bypass profile id %q is duplicated", profile.ID)
		}
		profileIDs[profile.ID] = struct{}{}
		if strings.TrimSpace(profile.Name) == "" || strings.TrimSpace(profile.Reason) == "" {
			return fmt.Errorf("bypass profile %s requires name and reason", profile.ID)
		}
		if len(profile.Name) > 128 || len(profile.Reason) > 512 {
			return fmt.Errorf("bypass profile %s name or reason is too long", profile.ID)
		}
		if profile.ValidUntil == nil {
			return fmt.Errorf("bypass profile %s requires valid-until", profile.ID)
		}
		if profile.Priority <= 0 {
			return fmt.Errorf("bypass profile %s has invalid priority %d", profile.ID, profile.Priority)
		}
		if existing, exists := profilePriorities[profile.Priority]; exists {
			return fmt.Errorf("bypass profile priority %d is duplicated by %s and %s", profile.Priority, existing, profile.ID)
		}
		profilePriorities[profile.Priority] = profile.ID
		hasBoundedSerialScope := profile.SerialScope != nil && profile.SerialScope.Type != SerialScopeAll
		if !hasBoundedSerialScope && len(profile.OUIs) == 0 && len(profile.ProductClasses) == 0 {
			return fmt.Errorf("bypass profile %s has no match selector", profile.ID)
		}
		if profile.SerialScope != nil {
			if err := validateSerialScope(profile.ID, *profile.SerialScope); err != nil {
				return fmt.Errorf("bypass profile scope: %w", err)
			}
		}
		if err := validateBypassSelectorValues(profile.ID, "OUI", profile.OUIs); err != nil {
			return err
		}
		if err := validateBypassSelectorValues(profile.ID, "ProductClass", profile.ProductClasses); err != nil {
			return err
		}
		if profile.ValidFrom != nil && profile.ValidUntil != nil && !profile.ValidFrom.Before(*profile.ValidUntil) {
			return fmt.Errorf("bypass profile %s valid-until must be after valid-from", profile.ID)
		}
	}
	ruleIDs := make(map[string]struct{}, len(policy.Rules))
	priorities := make(map[int]string, len(policy.Rules))
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
		if rule.Priority <= 0 {
			return fmt.Errorf("policy rule %s has invalid priority %d", ruleID, rule.Priority)
		}
		if existingRuleID, exists := priorities[rule.Priority]; exists {
			return fmt.Errorf("policy rule priority %d is duplicated by %s and %s", rule.Priority, existingRuleID, ruleID)
		}
		priorities[rule.Priority] = ruleID
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

func validateBypassSelectorValues(profileID, field string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > 128 {
			return fmt.Errorf("bypass profile %s has an invalid %s selector", profileID, field)
		}
		key := strings.ToUpper(value)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("bypass profile %s has duplicate %s selector %q", profileID, field, value)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func normalizeCompiledPolicy(policy CompiledPolicy) CompiledPolicy {
	if policy.DefaultAction == "" {
		policy.DefaultAction = PolicyDefaultActionReject
	}
	if policy.FailureMode == "" {
		policy.FailureMode = FailureModeFailClosed
	}
	if policy.CollectionTimeout == 0 {
		policy.CollectionTimeout = 15 * time.Minute
	}
	for index := range policy.BypassProfiles {
		profile := &policy.BypassProfiles[index]
		profile.ID = strings.TrimSpace(profile.ID)
		profile.Name = strings.TrimSpace(profile.Name)
		profile.Reason = strings.TrimSpace(profile.Reason)
		for valueIndex := range profile.OUIs {
			profile.OUIs[valueIndex] = strings.ToUpper(strings.TrimSpace(profile.OUIs[valueIndex]))
		}
		for valueIndex := range profile.ProductClasses {
			profile.ProductClasses[valueIndex] = strings.TrimSpace(profile.ProductClasses[valueIndex])
		}
		if profile.ValidFrom != nil {
			value := profile.ValidFrom.UTC()
			profile.ValidFrom = &value
		}
		if profile.ValidUntil != nil {
			value := profile.ValidUntil.UTC()
			profile.ValidUntil = &value
		}
	}
	for index := range policy.Rules {
		rule := &policy.Rules[index]
		rule.ID = strings.TrimSpace(rule.ID)
		rule.Name = strings.TrimSpace(rule.Name)
		if rule.Priority == 0 {
			// Preserve the legacy API contract where rule order was represented by
			// array position. New clients send an explicit positive priority.
			rule.Priority = (index + 1) * 100
		}
		if rule.Name == "" {
			rule.Name = rule.ID
		}
		for conditionIndex := range rule.Conditions {
			condition := &rule.Conditions[conditionIndex]
			if condition.Type != ConditionTypeObservedIP {
				continue
			}
			switch condition.Operator {
			case ConditionOperatorEqual:
				condition.Expected = normalizeObservedHost(condition.Expected)
			case ConditionOperatorIn:
				for valueIndex := range condition.ExpectedAny {
					condition.ExpectedAny[valueIndex] = normalizeObservedHost(condition.ExpectedAny[valueIndex])
				}
			}
		}
	}
	return policy
}

// PostgreSQL inet values are commonly rendered with a host prefix (/32 or
// /128). Equality and IN conditions represent hosts, so accept that lossless
// representation while leaving network prefixes to the CIDR operator.
func normalizeObservedHost(value string) string {
	trimmed := strings.TrimSpace(value)
	if prefix, err := netip.ParsePrefix(trimmed); err == nil && prefix.Bits() == prefix.Addr().BitLen() {
		return prefix.Addr().Unmap().String()
	}
	if address, err := netip.ParseAddr(trimmed); err == nil {
		return address.Unmap().String()
	}
	return trimmed
}

// Rule and condition IDs are global database keys, not stable business IDs
// across immutable policy versions. New drafts therefore always receive fresh
// server-owned IDs so cloning or rolling back an older version cannot collide
// with the historical rows it copies. UpdateDraft deliberately does not call
// this helper and preserves the IDs of the draft being edited.
func regenerateDraftEntityIDs(policy CompiledPolicy) CompiledPolicy {
	for index := range policy.BypassProfiles {
		policy.BypassProfiles[index].ID = uuid.NewString()
	}
	for ruleIndex := range policy.Rules {
		policy.Rules[ruleIndex].ID = uuid.NewString()
		for conditionIndex := range policy.Rules[ruleIndex].Conditions {
			policy.Rules[ruleIndex].Conditions[conditionIndex].ID = uuid.NewString()
		}
	}
	return policy
}

func normalizeUpdatedDraftEntityIDs(incoming, current CompiledPolicy) CompiledPolicy {
	existingProfiles := make(map[string]struct{}, len(current.BypassProfiles))
	for _, profile := range current.BypassProfiles {
		existingProfiles[profile.ID] = struct{}{}
	}
	for index := range incoming.BypassProfiles {
		if _, exists := existingProfiles[incoming.BypassProfiles[index].ID]; !exists {
			incoming.BypassProfiles[index].ID = uuid.NewString()
		}
	}
	existingRules := make(map[string]map[string]struct{}, len(current.Rules))
	for _, rule := range current.Rules {
		conditions := make(map[string]struct{}, len(rule.Conditions))
		for _, condition := range rule.Conditions {
			conditions[condition.ID] = struct{}{}
		}
		existingRules[rule.ID] = conditions
	}
	for ruleIndex := range incoming.Rules {
		rule := &incoming.Rules[ruleIndex]
		existingConditions, ruleExists := existingRules[rule.ID]
		if !ruleExists {
			rule.ID = uuid.NewString()
			existingConditions = nil
		}
		for conditionIndex := range rule.Conditions {
			if _, exists := existingConditions[rule.Conditions[conditionIndex].ID]; !exists {
				rule.Conditions[conditionIndex].ID = uuid.NewString()
			}
		}
	}
	return incoming
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
		if condition.Operator != ConditionOperatorEqual && condition.Operator != ConditionOperatorIn &&
			condition.Operator != ConditionOperatorCIDR && condition.Operator != ConditionOperatorIPRange {
			return unsupportedPolicyOperator(ruleID, condition)
		}
	case ConditionTypeGPS:
		if condition.Operator != ConditionOperatorWithinRadius && condition.Operator != ConditionOperatorWithinBounds {
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
	case ConditionOperatorIPRange:
		ranges := condition.IPRanges
		if condition.IPRange != nil {
			ranges = append(ranges, *condition.IPRange)
		}
		if len(ranges) == 0 {
			return fmt.Errorf("policy condition %s in rule %s requires an IP range", conditionID, ruleID)
		}
		for _, value := range ranges {
			start, startBits := comparableIP(value.Start)
			end, endBits := comparableIP(value.End)
			if start == nil || end == nil || startBits != endBits || bytes.Compare(start, end) > 0 {
				return fmt.Errorf("policy condition %s in rule %s has an invalid IP range", conditionID, ruleID)
			}
		}
	case ConditionOperatorWithinRadius:
		if condition.GeoFence == nil || condition.GeoFence.RadiusMeters <= 0 {
			return fmt.Errorf("policy condition %s in rule %s requires a positive geofence radius", conditionID, ruleID)
		}
		if condition.GeoFence.Center.Latitude < -90 || condition.GeoFence.Center.Latitude > 90 ||
			condition.GeoFence.Center.Longitude < -180 || condition.GeoFence.Center.Longitude > 180 {
			return fmt.Errorf("policy condition %s in rule %s has an invalid geofence center", conditionID, ruleID)
		}
	case ConditionOperatorWithinBounds:
		bounds := condition.GeoBoundsAny
		if condition.GeoBounds != nil {
			bounds = append(bounds, *condition.GeoBounds)
		}
		if len(bounds) == 0 {
			return fmt.Errorf("policy condition %s in rule %s has invalid GPS bounds", conditionID, ruleID)
		}
		for _, value := range bounds {
			if !validGeoBounds(value) {
				return fmt.Errorf("policy condition %s in rule %s has invalid GPS bounds", conditionID, ruleID)
			}
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
