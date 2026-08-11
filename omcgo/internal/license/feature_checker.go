package license

import "encoding/json"

// HasFeature reports whether a feature path is authorized.
// A string value of "All" authorizes the current subtree.
func HasFeature(featureList FeatureList, path ...string) bool {
	if len(path) == 0 || len(featureList) == 0 {
		return false
	}
	return hasFeatureRaw(json.RawMessage(featureList), path)
}

func hasFeatureValue(value any, path []string) bool {
	switch current := value.(type) {
	case string:
		return current == "All"
	case map[string]any:
		if len(path) == 0 {
			return false
		}
		child, ok := current[path[0]]
		if !ok {
			return false
		}
		return hasFeatureValue(child, path[1:])
	case []any:
		if len(path) != 1 {
			return false
		}
		for _, item := range current {
			if item == "All" || item == path[0] {
				return true
			}
		}
	}
	return false
}

func hasFeatureRaw(raw json.RawMessage, path []string) bool {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	return hasFeatureValue(value, path)
}

// extractTimeLimitHours pulls the legacy cumulative-usage limit (hours) out of the
// feature_list envelope. Returns 0 when absent (no cumulative limit / perpetual).
func extractTimeLimitHours(featureList FeatureList) int {
	if len(featureList) == 0 {
		return 0
	}
	var env struct {
		TimeLimitHours int `json:"time_limit_hours"`
	}
	if err := json.Unmarshal(featureList, &env); err != nil {
		return 0
	}
	return env.TimeLimitHours
}

// extractAuthorizationTree unwraps the persisted feature_list envelope to the
// nested authorization tree that HasFeature traverses. The envelope shape is
// {legacy_feature_ids, legacy_feature_codes, features, authorization_tree};
// HasFeature must operate on authorization_tree alone, not the whole envelope.
// Returns nil when the envelope or tree is absent (HasFeature then returns false).
func extractAuthorizationTree(featureList FeatureList) json.RawMessage {
	if len(featureList) == 0 {
		return nil
	}
	var envelope struct {
		Tree json.RawMessage `json:"authorization_tree"`
	}
	if err := json.Unmarshal(featureList, &envelope); err != nil {
		return nil
	}
	return envelope.Tree
}
