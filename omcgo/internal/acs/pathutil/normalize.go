// Package pathutil provides utilities for normalizing TR069 parameter paths.
package pathutil

import "strings"

// RootDataModelDevice is the root node for Device data model (TR-181).
const RootDataModelDevice = "Device."

// RootDataModelIGD is the root node for InternetGatewayDevice data model (TR-098).
const RootDataModelIGD = "InternetGatewayDevice."

// NormalizeParameterPath normalizes a parameter path to include the appropriate root prefix.
// This handles cases where CPE devices send relative paths without the root prefix.
//
// Parameters:
//   - path: The parameter path from the CPE (may be relative or absolute)
//   - rootVersion: The RootDataModelVersion from the device (e.g., "2.8" for Device model)
//
// Returns the normalized absolute path.
func NormalizeParameterPath(path, rootVersion string) string {
	// Already has Device. prefix
	if strings.HasPrefix(path, RootDataModelDevice) {
		return path
	}

	// Already has InternetGatewayDevice. prefix
	if strings.HasPrefix(path, RootDataModelIGD) {
		return path
	}

	// Determine root prefix based on data model version
	// Device model version 2.x uses "Device." prefix
	// InternetGatewayDevice model uses "InternetGatewayDevice." prefix
	if strings.HasPrefix(rootVersion, "2.") {
		return RootDataModelDevice + path
	}

	// Default to Device. prefix for modern devices
	return RootDataModelDevice + path
}

// IsRelativePath checks if a parameter path is relative (missing root prefix).
func IsRelativePath(path string) bool {
	return !strings.HasPrefix(path, RootDataModelDevice) &&
		!strings.HasPrefix(path, RootDataModelIGD)
}

// GetRootPrefix extracts the root prefix from a normalized path.
// Returns empty string if the path is relative.
func GetRootPrefix(path string) string {
	if strings.HasPrefix(path, RootDataModelDevice) {
		return RootDataModelDevice
	}
	if strings.HasPrefix(path, RootDataModelIGD) {
		return RootDataModelIGD
	}
	return ""
}

// StripRootPrefix removes the root prefix from a path if present.
func StripRootPrefix(path string) string {
	if strings.HasPrefix(path, RootDataModelDevice) {
		return strings.TrimPrefix(path, RootDataModelDevice)
	}
	if strings.HasPrefix(path, RootDataModelIGD) {
		return strings.TrimPrefix(path, RootDataModelIGD)
	}
	return path
}
