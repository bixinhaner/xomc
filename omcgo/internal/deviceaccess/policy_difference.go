package deviceaccess

import (
	"bytes"
	"encoding/json"
	"sort"
)

type PolicyDifference struct {
	BaseVersionID            string `json:"base_version_id,omitempty"`
	BaseVersion              int64  `json:"base_version,omitempty"`
	TargetVersionID          string `json:"target_version_id"`
	TargetVersion            int64  `json:"target_version"`
	DefaultActionChanged     bool   `json:"default_action_changed"`
	FailureModeChanged       bool   `json:"failure_mode_changed"`
	CollectionTimeoutChanged bool   `json:"collection_timeout_changed"`
	BypassProfilesAdded      int    `json:"bypass_profiles_added"`
	BypassProfilesRemoved    int    `json:"bypass_profiles_removed"`
	BypassProfilesChanged    int    `json:"bypass_profiles_changed"`
	RulesAdded               int    `json:"rules_added"`
	RulesRemoved             int    `json:"rules_removed"`
	RulesChanged             int    `json:"rules_changed"`
	ConditionsAdded          int    `json:"conditions_added"`
	ConditionsRemoved        int    `json:"conditions_removed"`
	NoChanges                bool   `json:"no_changes"`
}

func comparePolicyVersions(base, target PolicyVersion) PolicyDifference {
	difference := PolicyDifference{
		BaseVersionID: base.ID, BaseVersion: base.Version,
		TargetVersionID: target.ID, TargetVersion: target.Version,
		DefaultActionChanged:     base.Policy.DefaultAction != target.Policy.DefaultAction,
		FailureModeChanged:       base.Policy.FailureMode != target.Policy.FailureMode,
		CollectionTimeoutChanged: base.Policy.CollectionTimeout != target.Policy.CollectionTimeout,
	}
	difference.BypassProfilesAdded, difference.BypassProfilesRemoved, difference.BypassProfilesChanged = comparePrioritizedEntities(
		base.Policy.BypassProfiles, target.Policy.BypassProfiles,
		func(value BypassProfile) int { return value.Priority }, canonicalBypassProfile,
	)
	difference.RulesAdded, difference.RulesRemoved, difference.RulesChanged = comparePrioritizedEntities(
		base.Policy.Rules, target.Policy.Rules,
		func(value CompiledRule) int { return value.Priority }, canonicalRule,
	)
	baseConditions := canonicalConditionSet(base.Policy.Rules)
	targetConditions := canonicalConditionSet(target.Policy.Rules)
	difference.ConditionsAdded, difference.ConditionsRemoved = multisetDifference(baseConditions, targetConditions)
	difference.NoChanges = !difference.DefaultActionChanged && !difference.FailureModeChanged &&
		!difference.CollectionTimeoutChanged && difference.BypassProfilesAdded == 0 &&
		difference.BypassProfilesRemoved == 0 && difference.BypassProfilesChanged == 0 &&
		difference.RulesAdded == 0 && difference.RulesRemoved == 0 && difference.RulesChanged == 0 &&
		difference.ConditionsAdded == 0 && difference.ConditionsRemoved == 0
	return difference
}

func comparePrioritizedEntities[T any](base, target []T, priority func(T) int, canonical func(T) []byte) (added, removed, changed int) {
	baseOrdered := append([]T(nil), base...)
	targetOrdered := append([]T(nil), target...)
	sort.SliceStable(baseOrdered, func(i, j int) bool { return priority(baseOrdered[i]) < priority(baseOrdered[j]) })
	sort.SliceStable(targetOrdered, func(i, j int) bool { return priority(targetOrdered[i]) < priority(targetOrdered[j]) })
	shared := len(baseOrdered)
	if len(targetOrdered) < shared {
		shared = len(targetOrdered)
	}
	for index := 0; index < shared; index++ {
		if !bytes.Equal(canonical(baseOrdered[index]), canonical(targetOrdered[index])) {
			changed++
		}
	}
	added = len(targetOrdered) - shared
	removed = len(baseOrdered) - shared
	return added, removed, changed
}

func canonicalBypassProfile(value BypassProfile) []byte {
	value.ID = ""
	value.Priority = 0
	value.OUIs = append([]string(nil), value.OUIs...)
	value.ProductClasses = append([]string(nil), value.ProductClasses...)
	sort.Strings(value.OUIs)
	sort.Strings(value.ProductClasses)
	if value.SerialScope != nil {
		scope := *value.SerialScope
		scope.Values = append([]string(nil), scope.Values...)
		value.SerialScope = &scope
		sort.Strings(value.SerialScope.Values)
	}
	payload, _ := json.Marshal(value)
	return payload
}

func canonicalRule(value CompiledRule) []byte {
	value.ID = ""
	value.Priority = 0
	value.SerialScope.Values = append([]string(nil), value.SerialScope.Values...)
	sort.Strings(value.SerialScope.Values)
	conditionPayloads := make([]string, 0, len(value.Conditions))
	for _, condition := range value.Conditions {
		conditionPayloads = append(conditionPayloads, string(canonicalCondition(condition)))
	}
	sort.Strings(conditionPayloads)
	value.Conditions = nil
	payload, _ := json.Marshal(struct {
		Rule       CompiledRule `json:"rule"`
		Conditions []string     `json:"conditions"`
	}{Rule: value, Conditions: conditionPayloads})
	return payload
}

func canonicalCondition(value CompiledCondition) []byte {
	value.ID = ""
	value.ExpectedAny = append([]string(nil), value.ExpectedAny...)
	value.IPRanges = append([]IPRange(nil), value.IPRanges...)
	value.GeoBoundsAny = append([]GeoBounds(nil), value.GeoBoundsAny...)
	sort.Strings(value.ExpectedAny)
	sort.Slice(value.IPRanges, func(left, right int) bool {
		if value.IPRanges[left].Start == value.IPRanges[right].Start {
			return value.IPRanges[left].End < value.IPRanges[right].End
		}
		return value.IPRanges[left].Start < value.IPRanges[right].Start
	})
	sort.Slice(value.GeoBoundsAny, func(left, right int) bool {
		leftPayload, _ := json.Marshal(value.GeoBoundsAny[left])
		rightPayload, _ := json.Marshal(value.GeoBoundsAny[right])
		return bytes.Compare(leftPayload, rightPayload) < 0
	})
	payload, _ := json.Marshal(value)
	return payload
}

func canonicalConditionSet(rules []CompiledRule) map[string]int {
	result := make(map[string]int)
	for _, rule := range rules {
		for _, condition := range rule.Conditions {
			key := string(canonicalCondition(condition))
			result[key]++
		}
	}
	return result
}

func multisetDifference(base, target map[string]int) (added, removed int) {
	for key, count := range target {
		if delta := count - base[key]; delta > 0 {
			added += delta
		}
	}
	for key, count := range base {
		if delta := count - target[key]; delta > 0 {
			removed += delta
		}
	}
	return added, removed
}
