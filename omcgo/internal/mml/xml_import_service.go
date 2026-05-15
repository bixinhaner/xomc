// T-0132 MML admin Tab 4 XML import service.
//
// Path：admin uploads standard-model.xml → preview (dry-run diff) → apply (UPSERT).
//
// Sync constraint：解析 + 行构造 helpers 与 omcctl/mml.go::runMMLImport 同源
// （xml_import_helpers.go 直接复制）；SQL 写入 helpers 与 admin_repository.go::
// BatchUpsertStandardParams 配合（ON CONFLICT WHERE catalog_protected=true）。

package mml

import (
	"context"
	"fmt"
	"io"

	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel/mmlstandardloader"
)

// MaxImportRows 单次导入硬上限（防止 DoS / 内存爆炸 / accidental huge XML）。
// standard-model.xml ~2000 行；预留 5x buffer = 10000 行硬顶。
const MaxImportRows = 10000

// XMLImportService admin XML 导入业务层。
type XMLImportService struct {
	paramRepo AdminParamRepository
	audit     AdminAuditWriter
	logger    *zap.Logger
}

// NewXMLImportService 构造 XMLImportService。
func NewXMLImportService(
	paramRepo AdminParamRepository,
	audit AdminAuditWriter,
	logger *zap.Logger,
) *XMLImportService {
	if audit == nil {
		audit = noopAuditWriter{}
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &XMLImportService{
		paramRepo: paramRepo,
		audit:     audit,
		logger:    logger.Named("mml-xml-import"),
	}
}

// ImportBucket 表示一行 XML 数据 vs DB 现状的关系类型。
type ImportBucket string

const (
	BucketAdd     ImportBucket = "add"     // XML 中有 + DB 中无（INSERT）
	BucketModify  ImportBucket = "modify"  // XML 中有 + DB 中 catalog_protected=true（UPDATE）
	BucketSkipped ImportBucket = "skipped" // XML 中有 + DB 中 catalog_protected=false（admin 改过 → 守护跳过）
)

// ImportDiff 单条 diff 详情（preview 返前 N 行）。
type ImportDiff struct {
	Bucket     ImportBucket `json:"bucket"`
	ParamCode  string       `json:"param_code"`
	Tr069Path  string       `json:"tr069_path"`
	ValueType  string       `json:"value_type"`
	AccessType string       `json:"access_type"`
	IsObject   bool         `json:"is_object"`
}

// ImportSummary 三桶统计 + 总数。
type ImportSummary struct {
	Add     int `json:"add"`
	Modify  int `json:"modify"`
	Skipped int `json:"skipped"`
	Total   int `json:"total"`
}

// ImportPreviewResp POST /admin/import/preview 响应。
type ImportPreviewResp struct {
	VersionCode string        `json:"version_code"`
	Summary     ImportSummary `json:"summary"`
	Diffs       []ImportDiff  `json:"diffs"` // 前 MaxPreviewDiffs 行预览
	Truncated   bool          `json:"truncated"`
	Total       int           `json:"total"`
}

// MaxPreviewDiffs preview 返回 diff 数量上限（避免响应 payload 过大；前端表分页式翻 future 加）。
const MaxPreviewDiffs = 200

// ImportApplyResp POST /admin/import/apply 响应。
type ImportApplyResp struct {
	VersionCode  string `json:"version_code"`
	RowsAffected int64  `json:"rows_affected"`
	Total        int    `json:"total"`
}

// Preview 解析 XML + 与现有 DB 状态计算三桶 diff（dry-run，不写表）。
func (s *XMLImportService) Preview(ctx context.Context, xmlReader io.Reader, versionCode string) (*ImportPreviewResp, error) {
	if versionCode == "" {
		return nil, fmt.Errorf("version_code required")
	}
	params, objects, err := mmlstandardloader.ParseStandardXML(xmlReader)
	if err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}
	rows := buildImportRows(params, objects)
	if len(rows) == 0 {
		return nil, fmt.Errorf("xml contained 0 rows")
	}
	if len(rows) > MaxImportRows {
		return nil, fmt.Errorf("xml has %d rows, exceeds max %d", len(rows), MaxImportRows)
	}

	pathState, err := s.paramRepo.ListPathStateByVersion(ctx, versionCode)
	if err != nil {
		return nil, fmt.Errorf("list current path state: %w", err)
	}

	diffs := make([]ImportDiff, 0, len(rows))
	summary := ImportSummary{Total: len(rows)}
	for _, r := range rows {
		var bucket ImportBucket
		protected, exists := pathState[r.Tr069Path]
		switch {
		case !exists:
			bucket = BucketAdd
			summary.Add++
		case protected:
			bucket = BucketModify
			summary.Modify++
		default:
			bucket = BucketSkipped
			summary.Skipped++
		}
		diffs = append(diffs, ImportDiff{
			Bucket:     bucket,
			ParamCode:  r.ParamCode,
			Tr069Path:  r.Tr069Path,
			ValueType:  r.ValueType,
			AccessType: r.AccessType,
			IsObject:   r.IsObject,
		})
	}

	truncated := false
	if len(diffs) > MaxPreviewDiffs {
		diffs = diffs[:MaxPreviewDiffs]
		truncated = true
	}

	s.audit.Write(ctx, "mml.catalog.import.preview", "version:"+versionCode, map[string]any{
		"add":     summary.Add,
		"modify":  summary.Modify,
		"skipped": summary.Skipped,
		"total":   summary.Total,
	})

	return &ImportPreviewResp{
		VersionCode: versionCode,
		Summary:     summary,
		Diffs:       diffs,
		Truncated:   truncated,
		Total:       len(rows),
	}, nil
}

// Apply 解析 XML + 执行批量 UPSERT（守护 catalog_protected=true 的 standard 行）。
func (s *XMLImportService) Apply(ctx context.Context, xmlReader io.Reader, versionCode string) (*ImportApplyResp, error) {
	if versionCode == "" {
		return nil, fmt.Errorf("version_code required")
	}
	params, objects, err := mmlstandardloader.ParseStandardXML(xmlReader)
	if err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}
	rows := buildImportRows(params, objects)
	if len(rows) == 0 {
		return nil, fmt.Errorf("xml contained 0 rows")
	}
	if len(rows) > MaxImportRows {
		return nil, fmt.Errorf("xml has %d rows, exceeds max %d", len(rows), MaxImportRows)
	}

	importRows := make([]ImportRow, len(rows))
	for i, r := range rows {
		importRows[i] = ImportRow(r)
	}

	affected, err := s.paramRepo.BatchUpsertStandardParams(ctx, importRows, versionCode)
	if err != nil {
		return nil, fmt.Errorf("batch upsert: %w", err)
	}

	s.audit.Write(ctx, "mml.catalog.import.apply", "version:"+versionCode, map[string]any{
		"total":         len(rows),
		"rows_affected": affected,
	})
	s.logger.Info("xml import applied",
		zap.String("version_code", versionCode),
		zap.Int("total", len(rows)),
		zap.Int64("rows_affected", affected),
	)

	return &ImportApplyResp{
		VersionCode:  versionCode,
		RowsAffected: affected,
		Total:        len(rows),
	}, nil
}
