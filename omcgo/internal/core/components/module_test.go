package components

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleGraph_Resolve_Linear(t *testing.T) {
	g := NewModuleGraph()
	g.Add(ModuleInitializer{Name: "a", Depends: nil})
	g.Add(ModuleInitializer{Name: "b", Depends: []string{"a"}})
	g.Add(ModuleInitializer{Name: "c", Depends: []string{"b"}})

	order, err := g.Resolve()
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, order)
}

func TestModuleGraph_Resolve_Diamond(t *testing.T) {
	g := NewModuleGraph()
	g.Add(ModuleInitializer{Name: "a", Depends: nil})
	g.Add(ModuleInitializer{Name: "b", Depends: []string{"a"}})
	g.Add(ModuleInitializer{Name: "c", Depends: []string{"a"}})
	g.Add(ModuleInitializer{Name: "d", Depends: []string{"b", "c"}})

	order, err := g.Resolve()
	require.NoError(t, err)
	// a must come first, d must come last
	assert.Equal(t, "a", order[0])
	assert.Equal(t, "d", order[3])
}

func TestModuleGraph_ParallelGroups(t *testing.T) {
	g := NewModuleGraph()
	g.Add(ModuleInitializer{Name: "config", Depends: nil})
	g.Add(ModuleInitializer{Name: "topology", Depends: nil})
	g.Add(ModuleInitializer{Name: "admin", Depends: []string{"topology"}})
	g.Add(ModuleInitializer{Name: "device", Depends: []string{"topology"}})
	g.Add(ModuleInitializer{Name: "dashboard", Depends: []string{"device", "admin"}})

	groups, err := g.ParallelGroups()
	require.NoError(t, err)
	require.Len(t, groups, 3)

	// Group 0: config, topology (no deps)
	assert.ElementsMatch(t, []string{"config", "topology"}, groups[0])
	// Group 1: admin, device (depend on topology)
	assert.ElementsMatch(t, []string{"admin", "device"}, groups[1])
	// Group 2: dashboard (depends on device + admin)
	assert.Equal(t, []string{"dashboard"}, groups[2])
}

func TestModuleGraph_CircularDependency(t *testing.T) {
	g := NewModuleGraph()
	g.Add(ModuleInitializer{Name: "a", Depends: []string{"b"}})
	g.Add(ModuleInitializer{Name: "b", Depends: []string{"c"}})
	g.Add(ModuleInitializer{Name: "c", Depends: []string{"a"}})

	_, err := g.Resolve()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestModuleGraph_MissingDependency(t *testing.T) {
	g := NewModuleGraph()
	g.Add(ModuleInitializer{Name: "a", Depends: []string{"nonexistent"}})

	_, err := g.Resolve()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown module")
}

func TestModuleGraph_InitAll_Order(t *testing.T) {
	var order []string
	g := NewModuleGraph()
	g.Add(ModuleInitializer{Name: "a", Init: func() error { order = append(order, "a"); return nil }})
	g.Add(ModuleInitializer{Name: "b", Depends: []string{"a"}, Init: func() error { order = append(order, "b"); return nil }})
	g.Add(ModuleInitializer{Name: "c", Depends: []string{"a"}, Init: func() error { order = append(order, "c"); return nil }})
	g.Add(ModuleInitializer{Name: "d", Depends: []string{"b", "c"}, Init: func() error { order = append(order, "d"); return nil }})

	_, err := g.InitAll()
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c", "d"}, order)
}

func TestModuleGraph_InitAll_Error(t *testing.T) {
	g := NewModuleGraph()
	g.Add(ModuleInitializer{Name: "a", Init: func() error { return errors.New("boom") }})

	_, err := g.InitAll()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `init module "a"`)
}

func TestModuleGraph_ModuleNames(t *testing.T) {
	g := NewModuleGraph()
	g.Add(ModuleInitializer{Name: "c", Depends: nil})
	g.Add(ModuleInitializer{Name: "a", Depends: nil})
	g.Add(ModuleInitializer{Name: "b", Depends: nil})

	names := g.ModuleNames()
	assert.Equal(t, []string{"a", "b", "c"}, names)
}
