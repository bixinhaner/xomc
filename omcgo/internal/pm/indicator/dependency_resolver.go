package indicator

import (
	"errors"
	"fmt"
	"sort"
)

var (
	ErrCircularIndicatorDependency = errors.New("circular indicator dependency")
	ErrUnknownIndicatorDependency  = errors.New("unknown indicator dependency")
)

type DependencyClosure struct {
	Requested []string
	KPIs      []string
	Counters  []string
}

// ResolveDependencyClosure expands requested KPI arithmetic recursively until
// only counter dependencies remain. All output slices are unique and sorted so
// repository writes and audit logs are deterministic.
func ResolveDependencyClosure(
	requested []string,
	arithmeticByID map[string]string,
	isCounterByID map[string]bool,
) (DependencyClosure, error) {
	closure := DependencyClosure{Requested: sortedUnique(requested)}
	kpis := make(map[string]struct{})
	counters := make(map[string]struct{})
	visiting := make(map[string]bool)
	visited := make(map[string]bool)

	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("%w: %s", ErrCircularIndicatorDependency, id)
		}
		if visited[id] {
			return nil
		}
		isCounter, exists := isCounterByID[id]
		if !exists {
			return fmt.Errorf("%w: %s", ErrUnknownIndicatorDependency, id)
		}
		if isCounter {
			counters[id] = struct{}{}
			visited[id] = true
			return nil
		}

		formula := arithmeticByID[id]
		if formula == "" {
			return fmt.Errorf("%w: %s has empty arithmetic", ErrUnknownIndicatorDependency, id)
		}
		kpis[id] = struct{}{}
		visiting[id] = true
		for _, token := range tokenize(formula) {
			if isOperator(token) || isNumber(token) || token == "Duration" {
				continue
			}
			if err := visit(token); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}

	for _, id := range closure.Requested {
		if err := visit(id); err != nil {
			return DependencyClosure{}, err
		}
	}
	closure.KPIs = sortedKeys(kpis)
	closure.Counters = sortedKeys(counters)
	return closure, nil
}

func sortedUnique(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	return sortedKeys(set)
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
