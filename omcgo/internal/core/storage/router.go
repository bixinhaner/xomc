package storage

import (
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// BucketAndCategory 根据 TR-069 文件类型返回目标 MinIO Bucket 名称和分类子目录。
// category 为空时，文件直接以 {carrier}/{date}/{sn}/{filename} 格式存储到 Bucket 根目录。
// 使用场景： ACS Upload Handler 收到文件后，由此函数确定文件应存到哪个 Bucket，
// 再结合 ObjectPath 或 FirmwarePath 生成完整 MinIO 对象路径。
//
// 路由映射表：
//
//	Firmware  → firmware bucket,  img/patch 子目录
//	Config    → config_backup bucket, backup/
//	PM/MR     → pm_files/mr_files bucket（直接入根）
//	Logs      → logs bucket, running/security/fault/pcap/
//	DataModel → exchange bucket, datamodel/
//	SSLCert   → config_backup bucket, ssl-cert/
//	ImsCoreParam → config_backup bucket, ims-param/（核心网参数文件，docs/design/imscore-file-transfer.md）
func BucketAndCategory(ft tr069.FileType, buckets appconfig.BucketConfig) (bucket, category string) {
	switch ft {
	case tr069.FileTypeFirmware:
		return buckets.Firmware, "img"
	case tr069.FileTypePatch:
		return buckets.Firmware, "patch"
	case tr069.FileTypeConfig:
		return appconfig.NormalizeConfigBackupBucket(buckets.ConfigBackup), "backup"
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
		return appconfig.NormalizeConfigBackupBucket(buckets.ConfigBackup), "ssl-cert"
	case tr069.FileTypeImsCoreParam:
		return appconfig.NormalizeConfigBackupBucket(buckets.ConfigBackup), "ims-param"
	case tr069.FileTypeImsCoreLog:
		return appconfig.NormalizeConfigBackupBucket(buckets.ConfigBackup), "ims-log"
	case tr069.FileTypeImsCoreLicense:
		return appconfig.NormalizeConfigBackupBucket(buckets.ConfigBackup), "ims-license"
	case tr069.FileTypeImsCoreRecovery:
		return appconfig.NormalizeConfigBackupBucket(buckets.ConfigBackup), "ims-recovery"
	default:
		return buckets.Logs, "unknown"
	}
}
