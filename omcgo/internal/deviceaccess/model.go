package deviceaccess

import "time"

// AccessState is independent from device lifecycle and online state.
type AccessState string

const (
	AccessStateReviewRequired AccessState = "review_required"
	AccessStateCollecting     AccessState = "collecting"
	AccessStateAccepted       AccessState = "accepted"
	AccessStateRejected       AccessState = "rejected"
	AccessStateRevalidating   AccessState = "revalidating"
	AccessStateRevoked        AccessState = "revoked"
)

type EffectiveAction string

const (
	EffectiveActionAccept EffectiveAction = "accept"
	EffectiveActionReview EffectiveAction = "review"
	EffectiveActionReject EffectiveAction = "reject"
	EffectiveActionRevoke EffectiveAction = "revoke"
)

// PolicyDefaultAction defines how to handle a device when no enabled rule applies.
type PolicyDefaultAction string

const (
	PolicyDefaultActionReject PolicyDefaultAction = "reject"
)

type CredentialStatus string

const (
	CredentialStatusActive        CredentialStatus = "active"
	CredentialStatusNotConfigured CredentialStatus = "not_configured"
	CredentialStatusUnknown       CredentialStatus = "unknown"
)

type CheckResult string

const (
	CheckPassed  CheckResult = "passed"
	CheckFailed  CheckResult = "failed"
	CheckMissing CheckResult = "missing"
	CheckStale   CheckResult = "stale"
	CheckError   CheckResult = "error"
	CheckSkipped CheckResult = "skipped"
)

type EvidenceStatus string

const (
	EvidenceStatusAvailable        EvidenceStatus = "available"
	EvidenceStatusMissing          EvidenceStatus = "missing"
	EvidenceStatusCollectionFailed EvidenceStatus = "collection_failed"
	EvidenceStatusSystemError      EvidenceStatus = "system_error"
)

type ConditionType string

const (
	ConditionTypeIdentity   ConditionType = "identity"
	ConditionTypeAsset      ConditionType = "asset"
	ConditionTypeTAC        ConditionType = "tac"
	ConditionTypeECGI       ConditionType = "ecgi"
	ConditionTypeObservedIP ConditionType = "observed_ip"
	ConditionTypeGPS        ConditionType = "gps"
	ConditionTypeSecurity   ConditionType = "security"
)

type ConditionOperator string

const (
	ConditionOperatorEqual        ConditionOperator = "equal"
	ConditionOperatorIn           ConditionOperator = "in"
	ConditionOperatorCIDR         ConditionOperator = "cidr"
	ConditionOperatorWithinRadius ConditionOperator = "within_radius"
)

type SerialScopeType string

const (
	SerialScopeAll    SerialScopeType = "all"
	SerialScopeList   SerialScopeType = "list"
	SerialScopePrefix SerialScopeType = "prefix"
	SerialScopeRange  SerialScopeType = "range"
)

type ListEntryType string

const (
	ListEntryTypeDeny    ListEntryType = "deny"
	ListEntryTypeAllow   ListEntryType = "allow"
	ListEntryTypeRevoked ListEntryType = "revoked"
)

type ListEntryStatus string

const (
	ListEntryStatusActive   ListEntryStatus = "active"
	ListEntryStatusDisabled ListEntryStatus = "disabled"
)

type IdentityType string

const (
	IdentityTypeSerialNumber IdentityType = "serial_number"
)

type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type GeoFence struct {
	Center       GeoPoint `json:"center"`
	RadiusMeters float64  `json:"radius_meters"`
	// AllowMissing is the explicit #21155 policy for devices that do not
	// report usable latitude/longitude values. It is persisted with the GPS
	// expected-value JSON so the baseline schema does not need a new column.
	AllowMissing bool `json:"allow_missing,omitempty"`
}

type EvidenceValue struct {
	Status     EvidenceStatus `json:"status"`
	Text       string         `json:"text,omitempty"`
	Texts      []string       `json:"texts,omitempty"`
	Point      *GeoPoint      `json:"point,omitempty"`
	Source     string         `json:"source,omitempty"`
	ObservedAt time.Time      `json:"observed_at,omitempty"`
}

type EvidenceSet struct {
	Identity  CheckResult                     `json:"identity"`
	Ownership CheckResult                     `json:"ownership"`
	Values    map[ConditionType]EvidenceValue `json:"values"`
}

type SerialScope struct {
	Type   SerialScopeType `json:"type"`
	Values []string        `json:"values,omitempty"`
	Prefix string          `json:"prefix,omitempty"`
	Start  string          `json:"start,omitempty"`
	End    string          `json:"end,omitempty"`
}

type CompiledCondition struct {
	ID          string            `json:"id"`
	Type        ConditionType     `json:"type"`
	Operator    ConditionOperator `json:"operator"`
	Expected    string            `json:"expected,omitempty"`
	ExpectedAny []string          `json:"expected_any,omitempty"`
	GeoFence    *GeoFence         `json:"geo_fence,omitempty"`
	Required    bool              `json:"required"`
	EvidenceTTL time.Duration     `json:"evidence_ttl"`
}

type CompiledRule struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Enabled     bool                `json:"enabled"`
	SerialScope SerialScope         `json:"serial_scope"`
	Conditions  []CompiledCondition `json:"conditions"`
}

type CompiledListEntry struct {
	ID            string          `json:"id"`
	Type          ListEntryType   `json:"type"`
	IdentityType  IdentityType    `json:"identity_type"`
	IdentityValue string          `json:"identity_value"`
	Status        ListEntryStatus `json:"status"`
	Reason        string          `json:"reason,omitempty"`
	ValidFrom     *time.Time      `json:"valid_from,omitempty"`
	ValidUntil    *time.Time      `json:"valid_until,omitempty"`
}

type CompiledPolicy struct {
	VersionID     string              `json:"version_id"`
	DefaultAction PolicyDefaultAction `json:"default_action"`
	ListEntries   []CompiledListEntry `json:"list_entries"`
	Rules         []CompiledRule      `json:"rules"`
}

type EvaluationInput struct {
	Carrier                string         `json:"carrier"`
	SerialNumber           string         `json:"serial_number"`
	AuthenticationRequired bool           `json:"authentication_required"`
	Authenticated          bool           `json:"authenticated"`
	AssetRetired           bool           `json:"asset_retired"`
	ExistingState          AccessState    `json:"existing_state"`
	ConfirmedMismatch      bool           `json:"confirmed_mismatch"`
	EvaluatedAt            time.Time      `json:"evaluated_at"`
	Evidence               EvidenceSet    `json:"evidence"`
	Policy                 CompiledPolicy `json:"policy"`
}

type DecisionCheck struct {
	CheckID         string        `json:"check_id"`
	CheckType       ConditionType `json:"check_type"`
	Result          CheckResult   `json:"result"`
	ExpectedSummary string        `json:"expected_summary,omitempty"`
	ObservedSummary string        `json:"observed_summary,omitempty"`
	EvidenceSource  string        `json:"evidence_source,omitempty"`
	ObservedAt      time.Time     `json:"observed_at,omitempty"`
	ReasonCode      ReasonCode    `json:"reason_code,omitempty"`
}

type Decision struct {
	State           AccessState     `json:"state"`
	EffectiveAction EffectiveAction `json:"effective_action"`
	ReasonCode      ReasonCode      `json:"reason_code"`
	MatchedRuleID   string          `json:"matched_rule_id,omitempty"`
	MatchedEntryID  string          `json:"matched_entry_id,omitempty"`
	Checks          []DecisionCheck `json:"checks"`
	FreezeNormal    bool            `json:"freeze_normal"`
}
