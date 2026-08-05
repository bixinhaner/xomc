package notification

import (
	"bytes"
	"encoding/hex"
	"sort"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
)

func MatchRules(snapshot event.AlarmLifecycleSnapshot, candidates []RuleCandidate) []RuleMatch {
	matches := make([]RuleMatch, 0, len(candidates))
	for _, candidate := range candidates {
		if !ruleMatches(snapshot, candidate.Conditions) {
			continue
		}
		matches = append(matches, RuleMatch{
			RuleID: candidate.ID, Priority: candidate.Priority,
			Specificity: ruleSpecificity(candidate.Conditions),
		})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Priority != matches[j].Priority {
			return matches[i].Priority < matches[j].Priority
		}
		if matches[i].Specificity != matches[j].Specificity {
			return matches[i].Specificity > matches[j].Specificity
		}
		return bytes.Compare(matches[i].RuleID[:], matches[j].RuleID[:]) < 0
	})
	return matches
}

func ruleMatches(snapshot event.AlarmLifecycleSnapshot, conditions RuleMatchConditions) bool {
	if !containsStringOrEmpty(conditions.AlarmIdentifiers, snapshot.AlarmIdentifier) ||
		!containsSeverityOrEmpty(conditions.Severities, snapshot.Severity) ||
		!containsUUIDOrEmpty(conditions.DeviceIDs, snapshot.DeviceID) ||
		!containsCarrierOrEmpty(conditions.Carriers, snapshot.Carrier) {
		return false
	}
	technology := model.Technology("")
	if snapshot.Technology != nil {
		technology = model.Technology(*snapshot.Technology)
	}
	return containsTechnologyOrEmpty(conditions.Technologies, technology)
}

func ruleSpecificity(conditions RuleMatchConditions) int {
	specificity := 0
	if len(conditions.AlarmIdentifiers) > 0 {
		specificity++
	}
	if len(conditions.Severities) > 0 {
		specificity++
	}
	if len(conditions.DeviceIDs) > 0 {
		specificity++
	}
	if len(conditions.Carriers) > 0 {
		specificity++
	}
	if len(conditions.Technologies) > 0 {
		specificity++
	}
	return specificity
}

func MergeRuleRecipientCandidates(candidates []RuleRecipientCandidate) []RuleRecipientCandidate {
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i].Match, candidates[j].Match
		if left.Priority != right.Priority {
			return left.Priority < right.Priority
		}
		if left.Specificity != right.Specificity {
			return left.Specificity > right.Specificity
		}
		return bytes.Compare(left.RuleID[:], right.RuleID[:]) < 0
	})
	unique := make([]RuleRecipientCandidate, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		key := candidate.Recipient.Channel + ":" + hex.EncodeToString(candidate.Recipient.Fingerprint)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, candidate)
	}
	return unique
}

func containsStringOrEmpty(values []string, want string) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsSeverityOrEmpty(values []model.AlarmSeverity, want model.AlarmSeverity) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsUUIDOrEmpty(values []uuid.UUID, want uuid.UUID) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsCarrierOrEmpty(values []model.CarrierCode, want model.CarrierCode) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsTechnologyOrEmpty(values []model.Technology, want model.Technology) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
