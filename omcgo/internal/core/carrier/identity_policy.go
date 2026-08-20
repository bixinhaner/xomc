package carrier

import "github.com/omcgo/omcgo/internal/core/model"

// AccessIdentityPolicy centralizes carrier-specific TR-069 identity evidence.
// Required flags remain explicit so a carrier can tighten admission without
// scattering carrier comparisons through the device-access domain.
type AccessIdentityPolicy struct {
	DeviceCodePaths    []string
	CloudKeyPaths      []string
	DeviceCodeRequired bool
	CloudKeyRequired   bool
}

type accessIdentityPolicyProvider interface {
	AccessIdentityPolicy() AccessIdentityPolicy
}

func DefaultAccessIdentityPolicy() AccessIdentityPolicy {
	return AccessIdentityPolicy{
		DeviceCodePaths: []string{
			"Device.DeviceInfo.SiteId",
			"Device.DeviceInfo.UserLabel",
			"InternetGatewayDevice.DeviceInfo.SiteId",
			"InternetGatewayDevice.DeviceInfo.UserLabel",
		},
		CloudKeyPaths: []string{
			"Device.DeviceInfo.CloudKey",
			"Device.DeviceInfo.X_COM_CloudKey",
			"InternetGatewayDevice.DeviceInfo.CloudKey",
			"InternetGatewayDevice.DeviceInfo.X_COM_CloudKey",
		},
	}
}

func (r *CarrierRegistry) AccessIdentityPolicy(code model.CarrierCode) (AccessIdentityPolicy, error) {
	carrierAdapter, err := r.Get(code)
	if err != nil {
		return AccessIdentityPolicy{}, err
	}
	if provider, ok := carrierAdapter.(accessIdentityPolicyProvider); ok {
		return provider.AccessIdentityPolicy(), nil
	}
	return DefaultAccessIdentityPolicy(), nil
}
