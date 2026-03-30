package tr069

// FileType defines TR069 file transfer types used in Download/Upload RPC.
// Based on TR-069 Amendment 6 standard + project-specific extensions.
type FileType string

const (
	// Standard TR-069 FileType codes
	FileTypeFirmware    FileType = "1"  // 固件主镜像 (IMG) — Download
	FileTypePatch       FileType = "2"  // 补丁包 (PATCH) — Download (vendor extension)
	FileTypeConfig      FileType = "3"  // 厂商配置文件 — 双向
	FileTypePM          FileType = "4"  // PM 性能数据 — Upload (vendor log subtype)
	FileTypeMR          FileType = "5"  // MR 测量报告 — Upload (vendor log subtype)
	FileTypeRunningLog  FileType = "6"  // 运行日志 — Upload
	FileTypeSecurityLog FileType = "7"  // 安全日志 — Upload
	FileTypeFaultLog    FileType = "8"  // 异常重启/故障日志 — Upload
	FileTypePCAP        FileType = "9"  // 抓包文件 — Upload (vendor extension)
	FileTypeWeb         FileType = "10" // Web 内容 — Download
	FileTypeDataModel   FileType = "11" // 数据模型 XML — Upload (vendor extension)
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
	FileTypeFirmware:    "固件主镜像",
	FileTypePatch:       "补丁包",
	FileTypeConfig:      "配置文件",
	FileTypePM:          "PM性能数据",
	FileTypeMR:          "MR测量报告",
	FileTypeRunningLog:  "运行日志",
	FileTypeSecurityLog: "安全日志",
	FileTypeFaultLog:    "故障日志",
	FileTypePCAP:        "抓包文件",
	FileTypeWeb:         "Web内容",
	FileTypeDataModel:   "数据模型",
}

// IsUpload returns true if the file type is typically uploaded from CPE to ACS.
func (ft FileType) IsUpload() bool {
	switch ft {
	case FileTypePM, FileTypeMR, FileTypeRunningLog, FileTypeSecurityLog,
		FileTypeFaultLog, FileTypePCAP, FileTypeDataModel:
		return true
	case FileTypeConfig: // config can be both directions
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
	case FileTypeConfig: // config can be both directions
		return true
	default:
		return false
	}
}
