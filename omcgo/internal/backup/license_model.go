// Package backup — DeviceLicense (T-0165).
//
// 设备 license 库：每台设备一行最新 license。结构与 ConfigSnapshot 对齐，
// 但来源只有"手动导入"一种（license 不来自备份链路）。下发链路与配置恢复一致：
// LICENSE_UPGRADE 任务按 SN 取 license → device-licenses bucket → TR-069 Download
// (FileType="License File")。
package backup

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

// LicenseSource 当前仅支持手动导入；预留枚举以便将来扩展。
type LicenseSource string

const (
	LicenseSourceManualUpload LicenseSource = "manual_upload"
)

// LicenseFileExt 允许的 license 文件扩展名（小写）。当前仅 .lic。
type LicenseFileExt string

const (
	LicenseExtLIC LicenseFileExt = "lic"
)

// LicenseBucketDefault 是 license 文件的默认 MinIO bucket。
// modules.go::initBackupModule 启动时 EnsureBucket。
const LicenseBucketDefault = "device-licenses"

// DeviceLicense 是 device_licenses 表的行模型。
type DeviceLicense struct {
	SerialNumber string        `json:"serial_number"`
	EnbName      *string       `json:"enb_name,omitempty"`
	ProductType  *string       `json:"product_type,omitempty"`
	FileName     string        `json:"file_name"`     // 规范化后：<SN>_LIC.<ext>
	FileExt      string        `json:"file_ext"`      // "lic" | "bin" | "dat"
	ObjectBucket string        `json:"object_bucket"` // "device-licenses"
	ObjectPath   string        `json:"object_path"`   // "<SN>_LIC.<ext>"
	MD5          *string       `json:"md5,omitempty"`
	FileSize     int64         `json:"file_size"`
	Source       LicenseSource `json:"source"`
	Description  *string       `json:"description,omitempty"`
	// AutoDispatchPending marks a preinstalled license that still needs to be
	// sent when its device is registered or comes online.
	AutoDispatchPending bool      `json:"auto_dispatch_pending"`
	UpdateBy            *string   `json:"update_by,omitempty"`
	UpdateTime          time.Time `json:"update_time"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// LicenseFilter 列表查询条件。
type LicenseFilter struct {
	SerialNumber  string
	EnbName       string
	ProductType   string   // 单值精确匹配（向后兼容）
	ProductTypes  []string // 多值精确匹配（IN），按产品名称下拉过滤时由 handler 解析 product_id 写入
	UpdatedAfter  *time.Time
	UpdatedBefore *time.Time
	model.ListRequest
}

// licenseFileNamePattern 匹配 "<serial>.lic[.任意尾缀]" 形态（大小写不敏感）。
// 容忍 macOS Safari 下载附加 ".html" / Windows 邮件客户端附加 ".txt" 等场景：
// 只要主体是 <SN>.lic（出现在文件名头部），就视作合法，落库时统一 <SN>.lic 命名。
var licenseFileNamePattern = regexp.MustCompile(`^([A-Za-z0-9_-]+)\.lic(?:\.[A-Za-z0-9]+)*$`)

// ValidateImportLicenseFileName 校验导入文件名并提取 SN。
// 允许的形态：
//   - <SN>.lic
//   - <SN>.LIC / .Lic（大小写不敏感）
//   - <SN>.lic.html / .lic.txt（macOS / 邮件客户端自动附加的尾缀，会被忽略）
//
// 返回的 ext 一律 "lic"；调用方据此 LicenseObjectPath 落 MinIO key = <SN>.lic。
func ValidateImportLicenseFileName(fileName, expectedSN string) (sn, ext string, err error) {
	if fileName == "" {
		return "", "", fmt.Errorf("文件名不能为空")
	}
	// 大小写归一 + trim
	lower := strings.ToLower(strings.TrimSpace(fileName))
	m := licenseFileNamePattern.FindStringSubmatch(lower)
	if m == nil {
		return "", "", fmt.Errorf("license 文件名不符合规范，期望格式 <serialNumber>.lic，实际收到 %q", fileName)
	}
	parsedSN := m[1]
	// 原始大小写归还（SN 可能含大写字符）
	if idx := strings.Index(strings.ToLower(fileName), ".lic"); idx > 0 {
		parsedSN = fileName[:idx]
	}
	if expectedSN != "" && parsedSN != expectedSN {
		return "", "", fmt.Errorf("文件名 SN=%q 与期望 SN=%q 不一致", parsedSN, expectedSN)
	}
	return parsedSN, string(LicenseExtLIC), nil
}

// LicenseObjectPath 构造 MinIO object key：根级 "<SN>.lic"。
// ext 参数保留是为了 API 形态对齐 SnapshotObjectPath，目前必须 "lic"。
func LicenseObjectPath(serialNumber, ext string) string {
	_ = ext // 当前只有 .lic 一种；ext 由 ValidateImportLicenseFileName 保证为 "lic"
	return fmt.Sprintf("%s.lic", serialNumber)
}

// NormalizeLicenseExt 验证并归一化扩展名（保留以防其它调用）。
func NormalizeLicenseExt(rawName string) (string, error) {
	rawExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(rawName)), ".")
	if rawExt != string(LicenseExtLIC) {
		return "", fmt.Errorf("license 扩展名必须是 .lic，实际 %q", rawName)
	}
	return rawExt, nil
}
