// Package carrier 已在 carrier.go 中声明包注释。

package carrier

import (
	"errors"
	"fmt"
	"sync"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ErrAmbiguousCarrier 表示设备身份同时命中多个运营商，调用方必须要求
// 显式 carrier 或提供可唯一识别的 ProductClass，禁止选择注册顺序首项。
var ErrAmbiguousCarrier = errors.New("carrier is ambiguous for device identity")

// ErrUnresolvedCarrier means no registered OUI/ProductClass profile can own
// the observed identity and no deployment carrier is available for candidate
// review.
var ErrUnresolvedCarrier = errors.New("carrier is unresolved for device identity")

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
// carrier cannot be resolved from its device identity (see ResolveByIdentity).
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

// ResolveByOUI identifies a carrier only when the OUI belongs to exactly one
// registered carrier. Shared vendor OUIs return an empty code so callers do not
// silently assign the device to the first registered carrier.
func (r *CarrierRegistry) ResolveByOUI(oui string) model.CarrierCode {
	code, err := r.ResolveByIdentity(oui, "")
	if err != nil {
		return ""
	}
	return code
}

// ResolveByIdentity identifies a carrier from the complete TR-069 device
// identity. ProductClass disambiguates vendor OUIs shared by multiple carrier
// profiles. Unknown OUIs return ("", nil) so deployment-level fallback remains
// possible; ambiguous identities return ErrAmbiguousCarrier and must not fall
// back to a default carrier.
func (r *CarrierRegistry) ResolveByIdentity(oui, productClass string) (model.CarrierCode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ouiMatches := make(map[model.CarrierCode]struct{})
	exactMatches := make(map[model.CarrierCode]struct{})
	for _, code := range r.order {
		c := r.carriers[code]
		for _, tech := range c.SupportedTechnologies() {
			for _, info := range c.KnownOUIProductClasses(tech) {
				if info.OUI != oui {
					continue
				}
				ouiMatches[code] = struct{}{}
				if productClass != "" && info.ProductClass == productClass {
					exactMatches[code] = struct{}{}
				}
			}
		}
	}

	if len(exactMatches) == 1 {
		return firstCarrierInRegistrationOrder(r.order, exactMatches), nil
	}
	if len(exactMatches) > 1 {
		return "", fmt.Errorf("%w: oui=%q product_class=%q", ErrAmbiguousCarrier, oui, productClass)
	}
	if len(ouiMatches) == 1 {
		return firstCarrierInRegistrationOrder(r.order, ouiMatches), nil
	}
	if len(ouiMatches) > 1 {
		return "", fmt.Errorf("%w: oui=%q product_class=%q", ErrAmbiguousCarrier, oui, productClass)
	}
	return "", nil
}

func firstCarrierInRegistrationOrder(order []model.CarrierCode, matches map[model.CarrierCode]struct{}) model.CarrierCode {
	for _, code := range order {
		if _, ok := matches[code]; ok {
			return code
		}
	}
	return ""
}
