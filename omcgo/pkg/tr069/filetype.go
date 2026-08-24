package tr069

// FileType defines TR069 file transfer types used in Download/Upload RPC.
// Based on TR-069 Amendment 6 standard + project-specific extensions.
type FileType string

const (
	// Standard TR-069 FileType codes
	FileTypeFirmware    FileType = "1"   // 固件主镜像 (IMG) — Download
	FileTypePatch       FileType = "2"   // 补丁包 (PATCH) — Download (vendor extension)
	FileTypeConfig      FileType = "3"   // 厂商配置文件 — 双向
	FileTypePM          FileType = "4"   // PM 性能数据 — Upload (vendor log subtype)
	FileTypeMR          FileType = "5"   // MR 测量报告 — Upload (vendor log subtype)
	FileTypeRunningLog  FileType = "6"   // 运行日志 — Upload
	FileTypeSecurityLog FileType = "7"   // 安全日志 — Upload
	FileTypeFaultLog    FileType = "8"   // 异常重启/故障日志 — Upload
	FileTypePCAP        FileType = "9"   // 抓包文件 — Upload (vendor extension)
	FileTypeWeb         FileType = "10"  // Web 内容 — Download
	FileTypeDataModel   FileType = "11"  // 数据模型 XML — Upload (vendor extension)
	FileTypeSSLCert     FileType = "SSL" // TR069 SSL 证书 — 双向 (vendor extension)
	// IMS 核心网参数文件（FT1~FT12 参数配置类）— 双向 (vendor extension)。
	// CWMP Download/Upload RPC 的 FileType 统一字面值是 "ImsCore Parameters File"
	// （见 internal/imsparam），此枚举仅作为 ACS 上传 URL query 别名 fileType=IMS_PARAM
	// 归一化后的内部路由键（桶/分类、事件发布）。
	FileTypeImsCoreParam FileType = "IMS_PARAM"
	// IMS 核心网日志（FT13/18/19，字面值 "Ims Log File"，URL 别名 IMS_LOG）— 上传。
	FileTypeImsCoreLog FileType = "IMS_LOG"
	// IMS 核心网 License（FT15，字面值 "Ims License File"，URL 别名 IMS_LICENSE）— 上传。
	FileTypeImsCoreLicense FileType = "IMS_LICENSE"
	// IMS 核心网恢复数据（FT17，字面值 "Ims Recovery File"，URL 别名 IMS_RECOVERY）— 上传。
	FileTypeImsCoreRecovery FileType = "IMS_RECOVERY"
)

// String returns the FileType code.
func (ft FileType) String() string {
	return string(ft)
}

// Label returns the Chinese description of a FileType.
func (ft FileType) Label() string {
	if l, ok := fileTypeLabels[ft]; ok {
		return l
	}
	return "未知类型"
}

var fileTypeLabels = map[FileType]string{
	FileTypeFirmware:        "固件主镜像",
	FileTypePatch:           "补丁包",
	FileTypeConfig:          "配置文件",
	FileTypePM:              "PM性能数据",
	FileTypeMR:              "MR测量报告",
	FileTypeRunningLog:      "运行日志",
	FileTypeSecurityLog:     "安全日志",
	FileTypeFaultLog:        "故障日志",
	FileTypePCAP:            "抓包文件",
	FileTypeWeb:             "Web内容",
	FileTypeDataModel:       "数据模型",
	FileTypeSSLCert:         "SSL证书",
	FileTypeImsCoreParam:    "核心网参数文件",
	FileTypeImsCoreLog:      "核心网日志",
	FileTypeImsCoreLicense:  "核心网License文件",
	FileTypeImsCoreRecovery: "核心网恢复数据",
}

// IsUpload returns true if the file type is typically uploaded from CPE to ACS.
func (ft FileType) IsUpload() bool {
	switch ft {
	case FileTypePM, FileTypeMR, FileTypeRunningLog, FileTypeSecurityLog,
		FileTypeFaultLog, FileTypePCAP, FileTypeDataModel:
		return true
	case FileTypeConfig, FileTypeSSLCert, FileTypeImsCoreParam: // config/ssl-cert/ims-param can be both directions
		return true
	case FileTypeImsCoreLog, FileTypeImsCoreLicense, FileTypeImsCoreRecovery: // IMS 日志/License/恢复 — 仅上传
		return true
	default:
		return false
	}
}

// IsDownload returns true if the file type is typically downloaded from ACS to CPE.
func (ft FileType) IsDownload() bool {
	switch ft {
	case FileTypeFirmware, FileTypePatch, FileTypeWeb:
		return true
	case FileTypeConfig, FileTypeSSLCert, FileTypeImsCoreParam: // config/ssl-cert/ims-param can be both directions
		return true
	default:
		return false
	}
}
