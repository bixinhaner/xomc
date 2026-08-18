package pageconfig

import "strings"

func runObjectTechnologyCode(domain Domain, object ScenarioObject) string {
	if domain == DomainPM {
		return technologyCodeOrDefault(object.Tech)
	}
	if strings.TrimSpace(object.Tech) == "" {
		return ""
	}
	return technologyCodeOrDefault(object.Tech)
}

func technologyCodeOrDefault(tech string) string {
	normalized := normalizeTech(tech)
	if normalized == "" {
		return "LTE"
	}
	return normalized
}

func technologyDirectoryName(tech string) string {
	switch technologyCodeOrDefault(tech) {
	case "LTE":
		return "ENB"
	case "GNB":
		return "GNB"
	case "GSM":
		return "GSM"
	default:
		return technologyCodeOrDefault(tech)
	}
}

func technologyDisplayLabel(tech string) string {
	switch technologyCodeOrDefault(tech) {
	case "LTE":
		return "eNB"
	case "GNB":
		return "gNB"
	case "GSM":
		return "GSM"
	default:
		return technologyCodeOrDefault(tech)
	}
}

func technologySummaryFields(domain Domain, object ScenarioObject) map[string]any {
	tech := runObjectTechnologyCode(domain, object)
	if tech == "" {
		return nil
	}
	return map[string]any{
		"technology":           tech,
		"technology_label":     technologyDisplayLabel(tech),
		"technology_directory": technologyDirectoryName(tech),
	}
}

func addTechnologySummaryFields(summary map[string]any, domain Domain, object ScenarioObject) {
	if summary == nil {
		return
	}
	for key, value := range technologySummaryFields(domain, object) {
		summary[key] = value
	}
	if profile := strings.TrimSpace(object.Profile); profile != "" {
		summary["object_profile"] = profile
	}
}

func fileRunBaseSummary(profileName string, group FileGroup, object ScenarioObject, req RunProfileRequest) map[string]any {
	summary := map[string]any{
		"format":         group.Format,
		"period":         group.Period,
		"trigger_reason": runTriggerReason(req),
	}
	if strings.TrimSpace(profileName) != "" {
		summary["profile_name"] = profileName
	}
	addTechnologySummaryFields(summary, group.Domain, object)
	return summary
}

func mergeSummary(base map[string]any, patch map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	for key, value := range patch {
		base[key] = value
	}
	return base
}

func runTechnologyDirectory(run FileRun) string {
	if run.Domain != DomainPM {
		return ""
	}
	if run.Summary != nil {
		if dir := strings.TrimSpace(stringFromSummary(run.Summary["technology_directory"])); dir != "" {
			return dir
		}
		if tech := strings.TrimSpace(stringFromSummary(run.Summary["technology"])); tech != "" {
			return technologyDirectoryName(tech)
		}
	}
	return technologyDirectoryName("")
}

func runObjectIdentityKey(run FileRun) string {
	parts := []string{
		strings.TrimSpace(string(run.Domain)),
		strings.TrimSpace(run.ObjectCode),
	}
	if tech := runTechnologyDirectory(run); tech != "" {
		parts = append(parts, tech)
	}
	if run.Summary != nil {
		if profile := strings.TrimSpace(stringFromSummary(run.Summary["object_profile"])); profile != "" {
			parts = append(parts, profile)
		}
	}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, strings.ToUpper(part))
		}
	}
	if len(out) == 0 {
		return run.ID
	}
	return strings.Join(out, "\x1f")
}
