package storage

import (
	"fmt"
	"time"
)

// ObjectPath builds a MinIO object path for device-originated files.
// Format: {category}/{carrier}/{YYYY}/{MM}/{DD}/{deviceSN}/{filename}
// Example: running/cmcc/2026/03/30/BCI-SN-00001/task123_20260330150000.log
func ObjectPath(category, carrier, deviceSN, filename string) string {
	now := time.Now()
	if category == "" {
		return fmt.Sprintf("%s/%s/%s/%s",
			carrier,
			now.Format("2006/01/02"),
			deviceSN,
			filename,
		)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		category,
		carrier,
		now.Format("2006/01/02"),
		deviceSN,
		filename,
	)
}

// ObjectPathAt builds a MinIO object path with an explicit timestamp.
func ObjectPathAt(category, carrier, deviceSN, filename string, t time.Time) string {
	if category == "" {
		return fmt.Sprintf("%s/%s/%s/%s",
			carrier,
			t.Format("2006/01/02"),
			deviceSN,
			filename,
		)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		category,
		carrier,
		t.Format("2006/01/02"),
		deviceSN,
		filename,
	)
}

// FirmwarePath builds a MinIO object path for firmware-related files.
// Format: {category}/{carrier}/{productClass}/{version}/{filename}
// Example: img/cmcc/Nova436Q/V100R001C00B060/firmware.bin
func FirmwarePath(category, carrier, productClass, version, filename string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		category,
		carrier,
		productClass,
		version,
		filename,
	)
}

// ExchangePath builds a MinIO object path for data exchange files (import/export).
// Format: {category}/{YYYY}/{MM}/{DD}/{userID}/{filename}
// Example: import/2026/03/30/user_001/devices.xlsx
func ExchangePath(category, userID, filename string) string {
	now := time.Now()
	return fmt.Sprintf("%s/%s/%s/%s",
		category,
		now.Format("2006/01/02"),
		userID,
		filename,
	)
}

// DataModelPath builds a MinIO object path for data model XML files.
// Format: datamodel/{carrier}/{oui}/{productClass}/{deviceSN}_{timestamp}.xml
func DataModelPath(carrier, oui, productClass, deviceSN string) string {
	ts := time.Now().Format("20060102150405")
	return fmt.Sprintf("datamodel/%s/%s/%s/%s_%s.xml",
		carrier,
		oui,
		productClass,
		deviceSN,
		ts,
	)
}

// DatePrefix returns the current date as a path prefix: YYYY/MM/DD
func DatePrefix() string {
	return time.Now().Format("2006/01/02")
}
