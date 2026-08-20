package deviceaccess

import (
	"bytes"
	"math"
	"net"
	"slices"
	"strings"
)

type Evaluator struct{}

func (Evaluator) Evaluate(input EvaluationInput) Decision {
	if profile, ok := matchingBypassProfile(input); ok {
		return Decision{
			State:           AccessStateAccepted,
			EffectiveAction: EffectiveActionBypass,
			ReasonCode:      ReasonBypassProfileMatched,
			Checks: []DecisionCheck{{
				CheckID: profile.ID, CheckType: ConditionTypeIdentity, Result: CheckPassed,
				ExpectedSummary: "bypass_profile:" + profile.Name,
			}},
		}
	}
	if input.AuthenticationRequired && !input.Authenticated {
		return reject(AccessStateRejected, EffectiveActionReject, ReasonAuthenticationFailed)
	}

	if input.AssetRetired {
		return reject(AccessStateRevoked, EffectiveActionRevoke, ReasonAssetRetired)
	}

	if entry, ok := matchingListEntry(input, ListEntryTypeRevoked); ok {
		decision := reject(AccessStateRevoked, EffectiveActionRevoke, ReasonRevokedListMatched)
		decision.MatchedEntryID = entry.ID
		return decision
	}
	if entry, ok := matchingListEntry(input, ListEntryTypeDeny); ok {
		decision := reject(AccessStateRejected, EffectiveActionReject, ReasonDenylistMatched)
		decision.MatchedEntryID = entry.ID
		return decision
	}
	// Allowlisted devices bypass ordinary access-control evidence and planning
	// rules. Protocol authentication, retired assets, revoked entries and the
	// denylist remain higher-priority safety gates above this branch.
	if entry, ok := matchingListEntry(input, ListEntryTypeAllow); ok {
		return Decision{
			State:           AccessStateAccepted,
			EffectiveAction: EffectiveActionAccept,
			ReasonCode:      ReasonAllowlistMatched,
			MatchedEntryID:  entry.ID,
		}
	}

	if input.Evidence.Identity == CheckFailed {
		return reject(AccessStateRejected, EffectiveActionReject, ReasonIdentityMismatch)
	}
	if input.Evidence.Identity != CheckPassed {
		return review(AccessStateReviewRequired, ReasonIdentityUnverified, nil)
	}
	if input.Evidence.Ownership == CheckFailed {
		return reject(AccessStateRejected, EffectiveActionReject, ReasonOwnershipMismatch)
	}
	if input.Evidence.Ownership != CheckPassed {
		return review(AccessStateReviewRequired, ReasonOwnershipUnverified, nil)
	}

	applicable := make([]CompiledRule, 0, len(input.Policy.Rules))
	for _, rule := range input.Policy.Rules {
		if rule.Enabled && serialScopeMatches(rule.SerialScope, input.SerialNumber) && len(rule.Conditions) > 0 {
			applicable = append(applicable, rule)
		}
	}
	slices.SortStableFunc(applicable, func(left, right CompiledRule) int {
		return left.Priority - right.Priority
	})
	if len(applicable) == 0 {
		if input.Policy.DefaultAction == PolicyDefaultActionReview {
			return review(AccessStateReviewRequired, ReasonNoApplicableRule, nil)
		}
		return reject(AccessStateRejected, EffectiveActionReject, ReasonNoApplicableRule)
	}

	checks := make([]DecisionCheck, 0)
	var hasMissing, hasStale, hasCollectionFailure, hasSystemError bool
	for _, rule := range applicable {
		ruleChecks, outcome := evaluateConditions(input, rule.Conditions)
		checks = append(checks, ruleChecks...)
		switch outcome {
		case conditionOutcomePassed:
			return Decision{
				State:           AccessStateAccepted,
				EffectiveAction: EffectiveActionAccept,
				ReasonCode:      ReasonRuleMatched,
				MatchedRuleID:   rule.ID,
				Checks:          checks,
			}
		case conditionOutcomeMissing:
			hasMissing = true
		case conditionOutcomeStale:
			hasStale = true
		case conditionOutcomeCollectionFailed:
			hasCollectionFailure = true
		case conditionOutcomeSystemError:
			hasSystemError = true
		}
	}

	switch {
	case hasSystemError:
		return evidenceFailureDecision(input, ReasonEvidenceSystemError, checks)
	case hasCollectionFailure:
		return evidenceFailureDecision(input, ReasonEvidenceCollectionFailed, checks)
	case hasStale:
		return evidenceFailureDecision(input, ReasonEvidenceStale, checks)
	case hasMissing:
		return evidenceFailureDecision(input, ReasonEvidenceMissing, checks)
	case (input.ExistingState == AccessStateAccepted || input.ExistingState == AccessStateRevalidating) && !input.ConfirmedMismatch:
		return review(AccessStateRevalidating, ReasonRuleMismatchPendingConfirmation, checks)
	case input.ConfirmedMismatch:
		decision := reject(AccessStateRejected, EffectiveActionReject, ReasonRuleMismatchConfirmed)
		decision.Checks = checks
		return decision
	default:
		decision := reject(AccessStateRejected, EffectiveActionReject, ReasonRuleMismatch)
		decision.Checks = checks
		return decision
	}
}

func matchingBypassProfile(input EvaluationInput) (BypassProfile, bool) {
	profiles := append([]BypassProfile(nil), input.Policy.BypassProfiles...)
	slices.SortStableFunc(profiles, func(left, right BypassProfile) int { return left.Priority - right.Priority })
	for _, profile := range profiles {
		if !profile.Enabled || (profile.ValidFrom != nil && input.EvaluatedAt.Before(*profile.ValidFrom)) ||
			(profile.ValidUntil != nil && !input.EvaluatedAt.Before(*profile.ValidUntil)) {
			continue
		}
		if profile.SerialScope != nil && !serialScopeMatches(*profile.SerialScope, input.SerialNumber) {
			continue
		}
		if len(profile.OUIs) > 0 && !containsFold(profile.OUIs, input.OUI) {
			continue
		}
		if len(profile.ProductClasses) > 0 && !containsFold(profile.ProductClasses, input.ProductClass) {
			continue
		}
		return profile, true
	}
	return BypassProfile{}, false
}

func containsFold(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func evidenceFailureDecision(input EvaluationInput, reason ReasonCode, checks []DecisionCheck) Decision {
	if input.CollectionDeadline == nil || input.EvaluatedAt.Before(*input.CollectionDeadline) {
		return review(AccessStateCollecting, reason, checks)
	}
	if input.Policy.FailureMode == FailureModeReviewHold {
		return review(AccessStateReviewRequired, reason, checks)
	}
	decision := reject(AccessStateRejected, EffectiveActionReject, reason)
	decision.Checks = checks
	return decision
}

func reject(state AccessState, action EffectiveAction, reason ReasonCode) Decision {
	return Decision{
		State:           state,
		EffectiveAction: action,
		ReasonCode:      reason,
		FreezeNormal:    true,
	}
}

func review(state AccessState, reason ReasonCode, checks []DecisionCheck) Decision {
	return Decision{
		State:           state,
		EffectiveAction: EffectiveActionReview,
		ReasonCode:      reason,
		Checks:          checks,
		FreezeNormal:    true,
	}
}

func matchingListEntry(input EvaluationInput, entryType ListEntryType) (CompiledListEntry, bool) {
	for _, entry := range input.Policy.ListEntries {
		if entry.Type != entryType || entry.Status != ListEntryStatusActive {
			continue
		}
		if entry.ValidFrom != nil && input.EvaluatedAt.Before(*entry.ValidFrom) {
			continue
		}
		if entry.ValidUntil != nil && !input.EvaluatedAt.Before(*entry.ValidUntil) {
			continue
		}
		if entry.IdentityType == IdentityTypeSerialNumber && entry.IdentityValue == input.SerialNumber {
			return entry, true
		}
	}
	return CompiledListEntry{}, false
}

func serialScopeMatches(scope SerialScope, serialNumber string) bool {
	switch scope.Type {
	case SerialScopeAll:
		return true
	case SerialScopeList:
		return slices.Contains(scope.Values, serialNumber)
	case SerialScopePrefix:
		return scope.Prefix != "" && strings.HasPrefix(serialNumber, scope.Prefix)
	case SerialScopeRange:
		return scope.Start != "" && scope.End != "" && serialNumber >= scope.Start && serialNumber <= scope.End
	default:
		return false
	}
}

type conditionOutcome uint8

const (
	conditionOutcomePassed conditionOutcome = iota
	conditionOutcomeMismatch
	conditionOutcomeMissing
	conditionOutcomeStale
	conditionOutcomeCollectionFailed
	conditionOutcomeSystemError
)

func evaluateConditions(input EvaluationInput, conditions []CompiledCondition) ([]DecisionCheck, conditionOutcome) {
	if len(conditions) == 0 {
		return nil, conditionOutcomePassed
	}

	checks := make([]DecisionCheck, 0, len(conditions))
	outcome := conditionOutcomePassed
	for _, condition := range conditions {
		check, current := evaluateCondition(input, condition)
		checks = append(checks, check)
		outcome = mergeConditionOutcome(outcome, current)
	}
	return checks, outcome
}

func evaluateCondition(input EvaluationInput, condition CompiledCondition) (DecisionCheck, conditionOutcome) {
	evidence, exists := input.Evidence.Values[condition.Type]
	check := DecisionCheck{
		CheckID:         condition.ID,
		CheckType:       condition.Type,
		ExpectedSummary: expectedSummary(condition),
	}
	if exists {
		check.ObservedSummary = observedSummary(evidence)
		check.EvidenceSource = evidence.Source
		check.ObservedAt = evidence.ObservedAt
	}

	if !exists || evidence.Status == EvidenceStatusMissing {
		if condition.Type == ConditionTypeGPS && conditionAllowsMissingGPS(condition) {
			check.Result = CheckSkipped
			return check, conditionOutcomePassed
		}
		if !condition.Required {
			check.Result = CheckSkipped
			return check, conditionOutcomePassed
		}
		check.Result = CheckMissing
		check.ReasonCode = ReasonEvidenceMissing
		return check, conditionOutcomeMissing
	}
	switch evidence.Status {
	case EvidenceStatusCollectionFailed:
		check.Result = CheckError
		check.ReasonCode = ReasonEvidenceCollectionFailed
		return check, conditionOutcomeCollectionFailed
	case EvidenceStatusSystemError:
		check.Result = CheckError
		check.ReasonCode = ReasonEvidenceSystemError
		return check, conditionOutcomeSystemError
	case EvidenceStatusAvailable:
	default:
		check.Result = CheckError
		check.ReasonCode = ReasonEvidenceSystemError
		return check, conditionOutcomeSystemError
	}

	if condition.EvidenceTTL > 0 &&
		(evidence.ObservedAt.IsZero() || input.EvaluatedAt.Sub(evidence.ObservedAt) > condition.EvidenceTTL) {
		check.Result = CheckStale
		check.ReasonCode = ReasonEvidenceStale
		return check, conditionOutcomeStale
	}

	matched, valid := conditionMatches(condition, evidence)
	if !valid {
		check.Result = CheckError
		check.ReasonCode = ReasonEvidenceSystemError
		return check, conditionOutcomeSystemError
	}
	if !matched {
		check.Result = CheckFailed
		check.ReasonCode = ReasonEvidenceMismatch
		return check, conditionOutcomeMismatch
	}

	check.Result = CheckPassed
	return check, conditionOutcomePassed
}

func mergeConditionOutcome(current, next conditionOutcome) conditionOutcome {
	if outcomeSeverity(next) > outcomeSeverity(current) {
		return next
	}
	return current
}

func outcomeSeverity(outcome conditionOutcome) int {
	switch outcome {
	case conditionOutcomeSystemError:
		return 5
	case conditionOutcomeCollectionFailed:
		return 4
	case conditionOutcomeStale:
		return 3
	case conditionOutcomeMissing:
		return 2
	case conditionOutcomeMismatch:
		return 1
	default:
		return 0
	}
}

func conditionMatches(condition CompiledCondition, evidence EvidenceValue) (bool, bool) {
	switch condition.Operator {
	case ConditionOperatorEqual:
		values := evidenceTextValues(evidence)
		return len(values) > 0 && allValuesMatch(values, func(value string) bool {
			return value == condition.Expected
		}), true
	case ConditionOperatorIn:
		values := evidenceTextValues(evidence)
		return len(values) > 0 && allValuesMatch(values, func(value string) bool {
			return slices.Contains(condition.ExpectedAny, value)
		}), true
	case ConditionOperatorCIDR:
		_, network, err := net.ParseCIDR(condition.Expected)
		if err != nil {
			return false, false
		}
		ip := net.ParseIP(evidence.Text)
		if ip == nil {
			return false, true
		}
		return network.Contains(ip), true
	case ConditionOperatorIPRange:
		ranges := condition.IPRanges
		if condition.IPRange != nil {
			ranges = append(ranges, *condition.IPRange)
		}
		if len(ranges) == 0 {
			return false, false
		}
		observed, observedBits := comparableIP(evidence.Text)
		if observed == nil {
			return false, false
		}
		for _, value := range ranges {
			start, startBits := comparableIP(value.Start)
			end, endBits := comparableIP(value.End)
			if start == nil || end == nil || startBits != endBits || startBits != observedBits || bytes.Compare(start, end) > 0 {
				return false, false
			}
			if bytes.Compare(observed, start) >= 0 && bytes.Compare(observed, end) <= 0 {
				return true, true
			}
		}
		return false, true
	case ConditionOperatorWithinRadius:
		if condition.GeoFence == nil || condition.GeoFence.RadiusMeters < 0 || evidence.Point == nil {
			return false, false
		}
		return distanceMeters(condition.GeoFence.Center, *evidence.Point) <= condition.GeoFence.RadiusMeters, true
	case ConditionOperatorWithinBounds:
		bounds := condition.GeoBoundsAny
		if condition.GeoBounds != nil {
			bounds = append(bounds, *condition.GeoBounds)
		}
		if len(bounds) == 0 || evidence.Point == nil {
			return false, false
		}
		for _, value := range bounds {
			if !validGeoBounds(value) {
				return false, false
			}
			if evidence.Point.Latitude >= value.MinLatitude && evidence.Point.Latitude <= value.MaxLatitude &&
				evidence.Point.Longitude >= value.MinLongitude && evidence.Point.Longitude <= value.MaxLongitude {
				return true, true
			}
		}
		return false, true
	default:
		return false, false
	}
}

func expectedSummary(condition CompiledCondition) string {
	switch condition.Operator {
	case ConditionOperatorIn:
		return strings.Join(condition.ExpectedAny, ",")
	case ConditionOperatorWithinRadius:
		if condition.GeoFence == nil {
			return ""
		}
		if condition.GeoFence.AllowMissing {
			return "geofence;allow_missing=true"
		}
		return "geofence"
	case ConditionOperatorIPRange:
		if len(condition.IPRanges) > 0 {
			parts := make([]string, 0, len(condition.IPRanges))
			for _, value := range condition.IPRanges {
				parts = append(parts, value.Start+"-"+value.End)
			}
			return strings.Join(parts, ",")
		}
		if condition.IPRange == nil {
			return ""
		}
		return condition.IPRange.Start + "-" + condition.IPRange.End
	case ConditionOperatorWithinBounds:
		for _, bounds := range condition.GeoBoundsAny {
			if bounds.AllowMissing {
				return "geo_bounds_any;allow_missing=true"
			}
		}
		if condition.GeoBounds != nil && condition.GeoBounds.AllowMissing {
			return "geo_bounds;allow_missing=true"
		}
		return "geo_bounds"
	default:
		return condition.Expected
	}
}

func conditionAllowsMissingGPS(condition CompiledCondition) bool {
	if condition.GeoFence != nil && condition.GeoFence.AllowMissing ||
		condition.GeoBounds != nil && condition.GeoBounds.AllowMissing {
		return true
	}
	for _, bounds := range condition.GeoBoundsAny {
		if bounds.AllowMissing {
			return true
		}
	}
	return false
}

func comparableIP(value string) (net.IP, int) {
	ip := net.ParseIP(value)
	if ip == nil {
		return nil, 0
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4, net.IPv4len
	}
	return ip.To16(), net.IPv6len
}

func validGeoBounds(bounds GeoBounds) bool {
	return bounds.MinLatitude >= -90 && bounds.MaxLatitude <= 90 &&
		bounds.MinLongitude >= -180 && bounds.MaxLongitude <= 180 &&
		bounds.MinLatitude <= bounds.MaxLatitude && bounds.MinLongitude <= bounds.MaxLongitude
}

func observedSummary(evidence EvidenceValue) string {
	if evidence.Point != nil {
		return "geo_point"
	}
	if len(evidence.Texts) > 0 {
		return strings.Join(evidence.Texts, ",")
	}
	return evidence.Text
}

func evidenceTextValues(evidence EvidenceValue) []string {
	if len(evidence.Texts) > 0 {
		return evidence.Texts
	}
	if evidence.Text != "" {
		return []string{evidence.Text}
	}
	return nil
}

func allValuesMatch(values []string, matches func(string) bool) bool {
	for _, value := range values {
		if !matches(value) {
			return false
		}
	}
	return true
}

func distanceMeters(a, b GeoPoint) float64 {
	const earthRadiusMeters = 6371000.0
	lat1 := a.Latitude * math.Pi / 180
	lat2 := b.Latitude * math.Pi / 180
	dLat := (b.Latitude - a.Latitude) * math.Pi / 180
	dLon := (b.Longitude - a.Longitude) * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}
