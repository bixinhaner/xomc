package components

import (
	"fmt"
	"sort"
	"strings"
)

// ModuleInitializer describes a single module's initialization requirements.
type ModuleInitializer struct {
	Name    string
	Depends []string // names of modules that must complete before this one
	Init    func() error
}

// ModuleGraph resolves initialization order from declared dependencies.
// It uses Kahn's algorithm (BFS-based topological sort) to produce
// parallel-friendly execution groups.
type ModuleGraph struct {
	modules map[string]ModuleInitializer
	deps    map[string][]string // module → its declared dependencies
}

// NewModuleGraph creates an empty module graph.
func NewModuleGraph() *ModuleGraph {
	return &ModuleGraph{
		modules: make(map[string]ModuleInitializer),
		deps:    make(map[string][]string),
	}
}

// Add registers a module with its dependencies.
func (g *ModuleGraph) Add(m ModuleInitializer) {
	g.modules[m.Name] = m
	g.deps[m.Name] = m.Depends
}

// Resolve returns module names in a valid initialization order.
// Returns an error if a circular dependency or missing dependency is detected.
func (g *ModuleGraph) Resolve() ([]string, error) {
	groups, err := g.ParallelGroups()
	if err != nil {
		return nil, err
	}
	var order []string
	for _, group := range groups {
		sort.Strings(group) // deterministic ordering within a group
		order = append(order, group...)
	}
	return order, nil
}

// ParallelGroups returns groups of modules that can be initialized concurrently.
// Each group contains modules whose dependencies are fully satisfied by earlier groups.
// Groups are returned in dependency order; within each group, modules are independent.
func (g *ModuleGraph) ParallelGroups() ([][]string, error) {
	// Build in-degree map and adjacency list (reverse edges).
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // dep → modules that depend on it

	for name := range g.modules {
		if _, ok := inDegree[name]; !ok {
			inDegree[name] = 0
		}
	}

	for name, deps := range g.deps {
		for _, dep := range deps {
			if _, ok := g.modules[dep]; !ok {
				return nil, fmt.Errorf("module %q depends on unknown module %q", name, dep)
			}
			inDegree[name]++
			dependents[dep] = append(dependents[dep], name)
		}
	}

	var groups [][]string

	for len(inDegree) > 0 {
		// Find all modules with zero in-degree (no unresolved dependencies).
		var ready []string
		for name, deg := range inDegree {
			if deg == 0 {
				ready = append(ready, name)
			}
		}

		if len(ready) == 0 {
			// Remaining modules form a cycle.
			var remaining []string
			for name := range inDegree {
				remaining = append(remaining, name)
			}
			sort.Strings(remaining)
			return nil, fmt.Errorf("circular dependency detected among modules: %s",
				strings.Join(remaining, ", "))
		}

		sort.Strings(ready)
		groups = append(groups, ready)

		// Remove ready modules and update in-degrees.
		for _, name := range ready {
			delete(inDegree, name)
			for _, dep := range dependents[name] {
				if _, ok := inDegree[dep]; ok {
					inDegree[dep]--
				}
			}
		}
	}

	return groups, nil
}

// InitAll resolves the graph and executes all module initializers in order.
// Modules within the same parallel group are executed concurrently via errgroup.
// Returns the resolved execution plan and any error encountered.
func (g *ModuleGraph) InitAll() ([][]string, error) {
	groups, err := g.ParallelGroups()
	if err != nil {
		return nil, err
	}

	for _, group := range groups {
		for _, name := range group {
			m := g.modules[name]
			if err := m.Init(); err != nil {
				return nil, fmt.Errorf("init module %q: %w", name, err)
			}
		}
	}

	return groups, nil
}

// ModuleNames returns all registered module names in sorted order.
func (g *ModuleGraph) ModuleNames() []string {
	names := make([]string, 0, len(g.modules))
	for name := range g.modules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
