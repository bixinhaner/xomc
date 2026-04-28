package snmp

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
)

// TargetRegistry is the read-write store of SNMP Trap targets.
//
// In the skeleton stage the only concrete implementation is InMemoryRegistry.
// T-0017 will add a PostgreSQL-backed implementation that persists to
// snmp_trap_targets and emits a config-changed event so all replicas refresh.
type TargetRegistry interface {
	// Add inserts a copy of t into the registry. If t.ID is empty, a UUID is
	// generated. Returns ErrDuplicateID when an existing entry collides on ID,
	// and ErrDuplicateOSSName when oss_name collides.
	Add(t *TrapTarget) (*TrapTarget, error)

	// Update replaces the target with t.ID. Returns ErrTargetNotFound if no
	// existing entry has that id.
	Update(t *TrapTarget) error

	// Remove deletes the target with id. Returns true if a target was removed.
	Remove(id string) bool

	// Get returns a defensive copy of the target with id, or false if absent.
	Get(id string) (*TrapTarget, bool)

	// List returns defensive copies of every target sorted by OSSName.
	List() []*TrapTarget

	// ListEnabled returns defensive copies of only the targets where Enabled = true.
	ListEnabled() []*TrapTarget
}

// Registry-level errors are exported so callers (Engine, future REST handlers)
// can branch on them.
var (
	ErrDuplicateID      = errors.New("snmp: duplicate target id")
	ErrDuplicateOSSName = errors.New("snmp: duplicate oss_name")
	ErrTargetNotFound   = errors.New("snmp: target not found")
)

// InMemoryRegistry is a sync.RWMutex-protected map of TrapTarget. It is the
// only registry implementation shipped in the skeleton — the in-memory map
// is replaced by a PG-backed repository in T-0017. The interface is shaped
// so that swap requires zero changes in Engine.
type InMemoryRegistry struct {
	mu      sync.RWMutex
	targets map[string]*TrapTarget // id → target
	byName  map[string]string      // oss_name → id (uniqueness index)
}

// NewInMemoryRegistry constructs an empty in-memory registry.
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		targets: make(map[string]*TrapTarget),
		byName:  make(map[string]string),
	}
}

// Add inserts a copy of t into the registry. See TargetRegistry.Add for the
// detailed error contract.
func (r *InMemoryRegistry) Add(t *TrapTarget) (*TrapTarget, error) {
	if t == nil {
		return nil, errors.New("snmp: nil target")
	}
	if t.OSSName == "" {
		return nil, errors.New("snmp: target missing OSSName")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if _, exists := r.targets[t.ID]; exists {
		return nil, fmt.Errorf("%w: %s", ErrDuplicateID, t.ID)
	}
	if _, exists := r.byName[t.OSSName]; exists {
		return nil, fmt.Errorf("%w: %s", ErrDuplicateOSSName, t.OSSName)
	}

	cp := *t
	r.targets[cp.ID] = &cp
	r.byName[cp.OSSName] = cp.ID
	out := cp
	return &out, nil
}

// Update replaces an existing target. The existing oss_name index is kept in
// sync; renaming a target updates byName atomically under the same lock.
func (r *InMemoryRegistry) Update(t *TrapTarget) error {
	if t == nil {
		return errors.New("snmp: nil target")
	}
	if t.ID == "" {
		return errors.New("snmp: update requires ID")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.targets[t.ID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrTargetNotFound, t.ID)
	}

	// detect oss_name collision against a *different* target
	if otherID, taken := r.byName[t.OSSName]; taken && otherID != t.ID {
		return fmt.Errorf("%w: %s", ErrDuplicateOSSName, t.OSSName)
	}

	if existing.OSSName != t.OSSName {
		delete(r.byName, existing.OSSName)
		r.byName[t.OSSName] = t.ID
	}
	cp := *t
	r.targets[cp.ID] = &cp
	return nil
}

// Remove deletes the target with id. Returns true if present.
func (r *InMemoryRegistry) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.targets[id]
	if !ok {
		return false
	}
	delete(r.targets, id)
	delete(r.byName, t.OSSName)
	return true
}

// Get returns a defensive copy of the target with id.
func (r *InMemoryRegistry) Get(id string) (*TrapTarget, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.targets[id]
	if !ok {
		return nil, false
	}
	cp := *t
	return &cp, true
}

// List returns defensive copies sorted by OSSName for stable ordering.
func (r *InMemoryRegistry) List() []*TrapTarget {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*TrapTarget, 0, len(r.targets))
	for _, t := range r.targets {
		cp := *t
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OSSName < out[j].OSSName })
	return out
}

// ListEnabled returns only targets where Enabled = true.
func (r *InMemoryRegistry) ListEnabled() []*TrapTarget {
	all := r.List()
	out := make([]*TrapTarget, 0, len(all))
	for _, t := range all {
		if t.Enabled {
			out = append(out, t)
		}
	}
	return out
}
