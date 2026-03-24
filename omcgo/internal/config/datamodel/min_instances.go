package datamodel

import "strings"

type minInstancesRule struct {
	pattern string
	min     int
}

// minInstancesRules defines the default minimum instances based on TR-196/TR-181 specs.
// Rules are matched in order; first match wins.
var minInstancesRules = []minInstancesRule{
	// More specific rules first (first match wins).
	// Neighbor cells can be fully deleted
	{pattern: ".NeighborList.LTECell.", min: 0},
	{pattern: ".NeighborList.InterRATCell.", min: 0},
	// SCTP associations can be fully deleted
	{pattern: ".Transport.SCTP.Assoc.", min: 0},
	// PLMN: at least 1 (network access required)
	{pattern: ".PLMNList.", min: 1},
	// NR SNSSAI: at least 1
	{pattern: ".SNSSAI.", min: 1},
	// Tunnel: at least 1 (transport required)
	{pattern: ".Tunnel.", min: 1},
	// FAPService: at least 1 (core base station service) — broad, keep last
	{pattern: "Device.Services.FAPService.", min: 1},
}

// GetMinInstances returns the minimum number of instances for an object,
// based on ObjectInfo.MinInstances (if explicitly set) or the built-in rule table.
func GetMinInstances(obj ObjectInfo) int {
	if obj.MinInstances > 0 {
		return obj.MinInstances
	}
	// Match against rule table.
	for _, rule := range minInstancesRules {
		if matchMinInstancePattern(obj.Name, rule.pattern) {
			return rule.min
		}
	}
	// Default: conservatively require at least 1 instance.
	return 1
}

// ApplyMinInstances fills MinInstances for all objects using the rule table.
func ApplyMinInstances(objects []ObjectInfo) []ObjectInfo {
	for i := range objects {
		if !ContainsPlaceholder(objects[i].Name) {
			continue
		}
		objects[i].MinInstances = GetMinInstances(objects[i])
	}
	return objects
}

// matchMinInstancePattern checks if the object name matches a rule pattern.
// Patterns can use leading "." for suffix matching.
func matchMinInstancePattern(name, pattern string) bool {
	if strings.HasPrefix(pattern, ".") {
		return strings.Contains(name, pattern)
	}
	return strings.Contains(name, pattern)
}
