package deviceaccess

// ReasonCode is a stable machine-readable explanation used by APIs, events,
// audits, and localized user interfaces.
type ReasonCode string

const (
	ReasonNone                            ReasonCode = ""
	ReasonAuthenticationFailed            ReasonCode = "authentication_failed"
	ReasonAssetRetired                    ReasonCode = "asset_retired"
	ReasonRevokedListMatched              ReasonCode = "revoked_list_matched"
	ReasonDenylistMatched                 ReasonCode = "denylist_matched"
	ReasonIdentityMismatch                ReasonCode = "identity_mismatch"
	ReasonIdentityUnverified              ReasonCode = "identity_unverified"
	ReasonDeviceCodeMissing               ReasonCode = "device_code_missing"
	ReasonCloudKeyMissing                 ReasonCode = "cloud_key_missing"
	ReasonOwnershipMismatch               ReasonCode = "ownership_mismatch"
	ReasonOwnershipUnverified             ReasonCode = "ownership_unverified"
	ReasonAllowlistMatched                ReasonCode = "allowlist_matched"
	ReasonBypassProfileMatched            ReasonCode = "bypass_profile_matched"
	ReasonRuleMatched                     ReasonCode = "rule_matched"
	ReasonNoApplicableRule                ReasonCode = "no_applicable_rule"
	ReasonEvidenceMissing                 ReasonCode = "evidence_missing"
	ReasonEvidenceStale                   ReasonCode = "evidence_stale"
	ReasonEvidenceMismatch                ReasonCode = "evidence_mismatch"
	ReasonEvidenceCollectionFailed        ReasonCode = "evidence_collection_failed"
	ReasonEvidenceSystemError             ReasonCode = "evidence_system_error"
	ReasonRuleMismatch                    ReasonCode = "rule_mismatch"
	ReasonRuleMismatchPendingConfirmation ReasonCode = "rule_mismatch_pending_confirmation"
	ReasonRuleMismatchConfirmed           ReasonCode = "rule_mismatch_confirmed"
)
