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
