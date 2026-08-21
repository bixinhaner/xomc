package carrier

import "testing"

func TestDefaultAccessIdentityPolicyUsesInformEvidenceNotHTTPCredential(t *testing.T) {
	policy := DefaultAccessIdentityPolicy()
	if len(policy.DeviceCodePaths) == 0 || policy.DeviceCodePaths[0] != "Device.DeviceInfo.SiteId" {
		t.Fatalf("unexpected device code paths: %#v", policy.DeviceCodePaths)
	}
	if len(policy.CloudKeyPaths) != 4 || policy.CloudKeyPaths[0] != "Device.DeviceInfo.CloudKey" || policy.CloudKeyPaths[1] != "Device.DeviceInfo.X_COM_CloudKey" {
		t.Fatalf("unexpected CloudKey paths: %#v", policy.CloudKeyPaths)
	}
}
