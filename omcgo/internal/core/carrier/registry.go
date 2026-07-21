// Package carrier 已在 carrier.go 中声明包注释。

package carrier

import (
	"fmt"
	"sync"

	"github.com/omcgo/omcgo/internal/core/model"
)

// CarrierRegistry 管理运营商适配器的注册和查找。
// 各微服务在启动时初始化一个全局 Registry，并依次注册 cmcc/ctcc/cucc 适配器。
// 运行时不允许修改，所有读操作并发安全。
type CarrierRegistry struct {
	carriers map[model.CarrierCode]Carrier
	order    []model.CarrierCode
	mu       sync.RWMutex
}

// NewRegistry creates a new empty CarrierRegistry.
func NewRegistry() *CarrierRegistry {
	return &CarrierRegistry{
		carriers: make(map[model.CarrierCode]Carrier),
	}
}

// Register adds a carrier adapter to the registry.
func (r *CarrierRegistry) Register(carrier Carrier) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.carriers[carrier.Code()]; !exists {
		r.order = append(r.order, carrier.Code())
	}
	r.carriers[carrier.Code()] = carrier
}

// Get returns the carrier adapter for the given code.
func (r *CarrierRegistry) Get(code model.CarrierCode) (Carrier, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.carriers[code]
	if !ok {
		return nil, fmt.Errorf("carrier not found: %s", code)
	}
	return c, nil
}

// MustGet returns the carrier adapter or logs an error and returns nil if not found.
// Deprecated: prefer Get which returns an explicit error.
func (r *CarrierRegistry) MustGet(code model.CarrierCode) (Carrier, error) {
	return r.Get(code)
}

// All returns all registered carrier adapters.
func (r *CarrierRegistry) All() []Carrier {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Carrier, 0, len(r.carriers))
	for _, c := range r.carriers {
		result = append(result, c)
	}
	return result
}

// DefaultCarrier returns the fallback carrier code used when a device's
// carrier cannot be resolved from its OUI (see ResolveByOUI).
//
// #17: previously the default was a hardcoded model.CarrierCMCC constant at the
// InformHandler wiring site, coupling device registration to a single carrier.
// The default is now derived from the registry: CMCC is preferred when
// registered (matches historical behaviour and is the dominant deployment),
// otherwise the lexicographically smallest registered carrier code is returned
// (deterministic across restarts). Empty string means no carrier is registered
// (callers must treat that as a configuration error).
func (r *CarrierRegistry) DefaultCarrier() model.CarrierCode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.carriers[model.CarrierCMCC]; ok {
		return model.CarrierCMCC
	}
	var chosen model.CarrierCode
	for code := range r.carriers {
		if chosen == "" || code < chosen {
			chosen = code
		}
	}
	return chosen
}

// SupportsMRType reports whether the carrier identified by code collects the
// given measurement-report type. Unknown carriers return false so callers fail
// closed. This lets the registry satisfy the mr/parser MRTypeSupportChecker
// seam without that package depending on the concrete carrier adapters (#17).
func (r *CarrierRegistry) SupportsMRType(code model.CarrierCode, mrType model.MRType) bool {
	r.mu.RLock()
	c, ok := r.carriers[code]
	r.mu.RUnlock()
	if !ok {
		return false
	}
	return c.SupportsMRType(mrType)
}

// ResolveByOUI attempts to identify the carrier by matching a device OUI
// against known OUI-ProductClass combinations across all registered carriers.
// Returns the first matching carrier code in registration order, or empty
// string if no match. Registration order is significant because vendor OUIs
// can legitimately appear in more than one carrier profile.
func (r *CarrierRegistry) ResolveByOUI(oui string) model.CarrierCode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, code := range r.order {
		c := r.carriers[code]
		for _, tech := range c.SupportedTechnologies() {
			for _, info := range c.KnownOUIProductClasses(tech) {
				if info.OUI == oui {
					return c.Code()
				}
			}
		}
	}
	return ""
}
