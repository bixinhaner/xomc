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

// ResolveByOUI attempts to identify the carrier by matching a device OUI
// against known OUI-ProductClass combinations across all registered carriers.
// Returns the first matching carrier code, or empty string if no match.
func (r *CarrierRegistry) ResolveByOUI(oui string) model.CarrierCode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.carriers {
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
