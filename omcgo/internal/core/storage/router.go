package storage

import (
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// BucketAndCategory returns the target MinIO bucket name and category
// subdirectory for a given TR069 FileType.
// The category is used as the first path segment within the bucket.
// An empty category means files are stored directly under carrier/date.
func BucketAndCategory(ft tr069.FileType, buckets appconfig.BucketConfig) (bucket, category string) {
	switch ft {
	case tr069.FileTypeFirmware:
		return buckets.Firmware, "img"
	case tr069.FileTypePatch:
		return buckets.Firmware, "patch"
	case tr069.FileTypeConfig:
		return buckets.ConfigBackup, "backup"
	case tr069.FileTypePM:
		return buckets.PMFiles, ""
	case tr069.FileTypeMR:
		return buckets.MRFiles, ""
	case tr069.FileTypeRunningLog:
		return buckets.Logs, "running"
	case tr069.FileTypeSecurityLog:
		return buckets.Logs, "security"
	case tr069.FileTypeFaultLog:
		return buckets.Logs, "fault"
	case tr069.FileTypePCAP:
		return buckets.Logs, "pcap"
	case tr069.FileTypeWeb:
		return buckets.Firmware, "web"
	case tr069.FileTypeDataModel:
		return buckets.Exchange, "datamodel"
	case tr069.FileTypeSSLCert:
		return buckets.ConfigBackup, "ssl-cert"
	default:
		return buckets.Logs, "unknown"
	}
}
