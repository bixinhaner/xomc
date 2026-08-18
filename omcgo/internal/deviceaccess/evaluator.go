package deviceaccess

import (
	"math"
	"net"
	"slices"
	"strings"
)

type Evaluator struct{}

func (Evaluator) Evaluate(input EvaluationInput) Decision {
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

	if entry, ok := matchingListEntry(input, ListEntryTypeAllow); ok {
		return Decision{
			State:           AccessStateAccepted,
			EffectiveAction: EffectiveActionAccept,
			ReasonCode:      ReasonAllowlistMatched,
			MatchedEntryID:  entry.ID,
		}
	}
	applicable := make([]CompiledRule, 0, len(input.Policy.Rules))
	for _, rule := range input.Policy.Rules {
		if rule.Enabled && serialScopeMatches(rule.SerialScope, input.SerialNumber) && len(rule.Conditions) > 0 {
			applicable = append(applicable, rule)
		}
	}
	if len(applicable) == 0 {
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
		return review(AccessStateReviewRequired, ReasonEvidenceSystemError, checks)
	case hasCollectionFailure:
		return review(AccessStateCollecting, ReasonEvidenceCollectionFailed, checks)
	case hasStale:
		return review(AccessStateCollecting, ReasonEvidenceStale, checks)
	case hasMissing:
		return review(AccessStateCollecting, ReasonEvidenceMissing, checks)
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
		if condition.Type == ConditionTypeGPS && condition.GeoFence != nil && condition.GeoFence.AllowMissing {
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
	case ConditionOperatorWithinRadius:
		if condition.GeoFence == nil || condition.GeoFence.RadiusMeters < 0 || evidence.Point == nil {
			return false, false
		}
		return distanceMeters(condition.GeoFence.Center, *evidence.Point) <= condition.GeoFence.RadiusMeters, true
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
	default:
		return condition.Expected
	}
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
