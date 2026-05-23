// Package backup — config_snapshots 表（T-0164）。
//
// 设计文档：docs/project/config-snapshot-table-plan-20260522.md
//
// 每台设备一行最新配置快照。与 backup_tasks（按任务粒度）正交：
//   - backup_tasks.file_path：第 N 次备份留底，保留任务历史
//   - config_snapshots：该设备**当前最新**的可用配置，覆盖式更新
//
// 写入入口：
//   1. 备份链路：FilePathRecorder 写完 backup_restore_file 后调用
//      SnapshotService.PromoteFromBackup → MinIO server-side copy +
//      Upsert（规范化命名到 <SN>_CFG.<ext>）
//   2. 手动导入：SnapshotHandler.Import → multipart 校验命名 +
//      MinIO PutObject + Upsert
//
// 读取入口：
//   1. 配置快照库页面：按 SN / enb_name / product_type 过滤
//   2. Restore 创建：按 SN 列表 BatchGet → 派生 device_tasks
//
// 历史数据不回填（用户确认 2026-05-22）。
package backup

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/core/model"
)

// SnapshotSource 标识快照来源，用于审计与排查。
type SnapshotSource string

const (
	// SnapshotSourceBackup 表示快照由备份任务自动 promote。
	SnapshotSourceBackup SnapshotSource = "backup"
	// SnapshotSourceManualUpload 表示快照由用户手动导入。
	SnapshotSourceManualUpload SnapshotSource = "manual_upload"
)

// SnapshotFileExt 是允许的快照文件扩展名（小写）。
type SnapshotFileExt string

const (
	SnapshotExtXML SnapshotFileExt = "xml"
	SnapshotExtNV  SnapshotFileExt = "nv"
)

// SnapshotBucketDefault 是快照文件的默认 MinIO bucket。
// 在 modules.go::initBackupModule 启动时通过 EnsureBucket 创建。
const SnapshotBucketDefault = "config-snapshots"

// ConfigSnapshot 是 config_snapshots 表的行模型。
type ConfigSnapshot struct {
	SerialNumber string         `json:"serial_number"`
	EnbName      *string        `json:"enb_name,omitempty"`
	ProductType  *string        `json:"product_type,omitempty"`
	FileName     string         `json:"file_name"`     // 规范化后：<SN>_CFG.<ext>
	FileExt      string         `json:"file_ext"`      // "xml" | "nv"
	ObjectBucket string         `json:"object_bucket"` // "config-snapshots"
	ObjectPath   string         `json:"object_path"`   // "<SN>_CFG.<ext>"
	MD5          *string        `json:"md5,omitempty"`
	FileSize     int64          `json:"file_size"`
	Source       SnapshotSource `json:"source"`
	SourceTaskID *uuid.UUID     `json:"source_task_id,omitempty"`
	UpdateBy     *string        `json:"update_by,omitempty"`
	UpdateTime   time.Time      `json:"update_time"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// SnapshotFilter 列表查询条件。
type SnapshotFilter struct {
	SerialNumber   string          // 模糊匹配（ILIKE）
	EnbName        string          // 模糊匹配
	ProductType    string          // 精确匹配
	Source         *SnapshotSource // 可选
	UpdatedAfter   *time.Time
	UpdatedBefore  *time.Time
	model.ListRequest
}

// ─────────────────────────────────────────────────────────────────────────
// 文件名校验与规范化
// ─────────────────────────────────────────────────────────────────────────

// snapshotFileNamePattern 匹配 "<serial>_CFG.<ext>" 形态。
// SN 允许字母数字下划线短横线，扩展名 xml/nv（大小写不限，解析时统一转小写）。
var snapshotFileNamePattern = regexp.MustCompile(`^([A-Za-z0-9_-]+)_CFG\.(xml|nv|XML|NV|Xml|Nv)$`)

// Sentinel 错误集合 —— 同时供后端 error wrap 与前端友好文案使用。
var (
	ErrEmptyFileName       = errors.New("文件名不能为空")
	ErrEmptySerialNumber   = errors.New("serial_number 不能为空")
	ErrInvalidConfigFileExt = errors.New("配置文件扩展名必须是 .xml 或 .nv")
	ErrInvalidConfigFileName = errors.New(
		"配置文件名不符合规范，期望格式 <serialNumber>_CFG.xml 或 <serialNumber>_CFG.nv")
	ErrFileNameSerialMismatch = errors.New("文件名前缀与目标 serial_number 不一致")
)

// NormalizeSnapshotFileName 用于**备份链路**：把任意输入文件名规范化为
// "<serialNumber>_CFG.<ext>"。该路径**允许**输入文件名前缀与 SN 不一致
// （例如 backup-a1b2c3d4-SN001.xml），统一以传入的 serialNumber 重写。
//
// 仅校验扩展名合法。扩展名一律转小写（xml / nv）。
//
// 返回：规范化后的文件名、扩展名（小写）、错误。
func NormalizeSnapshotFileName(rawName, serialNumber string) (normalized, ext string, err error) {
	if serialNumber == "" {
		return "", "", ErrEmptySerialNumber
	}
	if rawName == "" {
		return "", "", ErrEmptyFileName
	}
	rawExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(rawName)), ".")
	switch rawExt {
	case string(SnapshotExtXML), string(SnapshotExtNV):
		// ok
	default:
		return "", "", fmt.Errorf("%w: 实际收到 %q", ErrInvalidConfigFileExt, rawName)
	}
	return fmt.Sprintf("%s_CFG.%s", serialNumber, rawExt), rawExt, nil
}

// ValidateImportFileName 用于**手动导入**：严格校验文件名必须形如
// "<serialNumber>_CFG.xml" 或 "<serialNumber>_CFG.nv"，
// 且 SN 前缀必须与 expectedSN（如非空）一致。
//
// 错误返回时附带期望格式与实际值，便于前端直接展示给用户。
// 调用方需把扩展名统一小写后再去查找对象（本函数返回的 ext 已是小写）。
func ValidateImportFileName(fileName, expectedSN string) (sn, ext string, err error) {
	if fileName == "" {
		return "", "", ErrEmptyFileName
	}
	m := snapshotFileNamePattern.FindStringSubmatch(fileName)
	if m == nil {
		return "", "", fmt.Errorf("%w，实际收到 %q", ErrInvalidConfigFileName, fileName)
	}
	parsedSN := m[1]
	parsedExt := strings.ToLower(m[2])
	if expectedSN != "" && parsedSN != expectedSN {
		return "", "", fmt.Errorf("%w：文件名 SN=%q，期望 SN=%q",
			ErrFileNameSerialMismatch, parsedSN, expectedSN)
	}
	return parsedSN, parsedExt, nil
}

// SnapshotObjectPath 构造 MinIO object key（与 bucket 解耦，便于将来切换）。
// 当前一律使用根级路径 "<SN>_CFG.<ext>"，简单可读。
func SnapshotObjectPath(serialNumber, ext string) string {
	return fmt.Sprintf("%s_CFG.%s", serialNumber, ext)
}
