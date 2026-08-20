package deviceaccess

import (
	"context"
	"encoding/json"
	"time"
)

type ImportType string

const (
	ImportTypeAccessList    ImportType = "access_list"
	ImportTypeRuleDimension ImportType = "rule_dimension"
)

type ImportDimension string

const (
	ImportDimensionSN   ImportDimension = "sn"
	ImportDimensionTAC  ImportDimension = "tac"
	ImportDimensionECGI ImportDimension = "ecgi"
	ImportDimensionIP   ImportDimension = "ip"
	ImportDimensionGPS  ImportDimension = "gps"
)

type ImportMode string

const (
	ImportModeAppend  ImportMode = "append"
	ImportModeReplace ImportMode = "replace"
)

type ImportFailurePolicy string

const (
	ImportFailureStrict    ImportFailurePolicy = "strict"
	ImportFailureValidOnly ImportFailurePolicy = "valid_only"
)

type ImportBatchStatus string

const (
	ImportBatchUploaded   ImportBatchStatus = "uploaded"
	ImportBatchValidated  ImportBatchStatus = "validated"
	ImportBatchCommitting ImportBatchStatus = "committing"
	ImportBatchCommitted  ImportBatchStatus = "committed"
	ImportBatchFailed     ImportBatchStatus = "failed"
	ImportBatchRolledBack ImportBatchStatus = "rolled_back"
)

type ImportRowStatus string

const (
	ImportRowValid     ImportRowStatus = "valid"
	ImportRowInvalid   ImportRowStatus = "invalid"
	ImportRowDuplicate ImportRowStatus = "duplicate"
	ImportRowNoChange  ImportRowStatus = "no_change"
)

type ImportBatch struct {
	ID                    string              `json:"id"`
	Carrier               string              `json:"carrier"`
	Type                  ImportType          `json:"import_type"`
	EntryType             ListEntryType       `json:"entry_type,omitempty"`
	TargetPolicyVersionID string              `json:"target_policy_version_id,omitempty"`
	TargetRuleID          string              `json:"target_rule_id,omitempty"`
	Dimension             ImportDimension     `json:"dimension,omitempty"`
	Mode                  ImportMode          `json:"mode"`
	FailurePolicy         ImportFailurePolicy `json:"failure_policy"`
	Status                ImportBatchStatus   `json:"status"`
	SourceFilename        string              `json:"source_filename"`
	ContentSHA256         string              `json:"content_sha256"`
	ScopeSHA256           string              `json:"scope_sha256,omitempty"`
	TotalCount            int                 `json:"total_count"`
	ValidCount            int                 `json:"valid_count"`
	InvalidCount          int                 `json:"invalid_count"`
	ChangedCount          int                 `json:"changed_count"`
	IdempotencyKey        string              `json:"idempotency_key,omitempty"`
	CreatedBy             string              `json:"created_by,omitempty"`
	CommittedBy           string              `json:"committed_by,omitempty"`
	RolledBackBy          string              `json:"rolled_back_by,omitempty"`
	ReversalOfBatchID     string              `json:"reversal_of_batch_id,omitempty"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
	CommittedAt           *time.Time          `json:"committed_at,omitempty"`
	RolledBackAt          *time.Time          `json:"rolled_back_at,omitempty"`
	Snapshot              json.RawMessage     `json:"-"`
}

type ImportPreviewRequest struct {
	Type                  ImportType
	EntryType             ListEntryType
	Mode                  ImportMode
	FailurePolicy         ImportFailurePolicy
	SourceFilename        string
	Content               []byte
	IdempotencyKey        string
	TargetPolicyVersionID string
	TargetRuleID          string
	Dimension             ImportDimension
	scopeSHA256           string
}

type ImportPreviewResult struct {
	Batch        ImportBatch         `json:"batch"`
	Rows         []ImportRow         `json:"rows"`
	Entries      []CompiledListEntry `json:"entries"`
	DisableCount int                 `json:"disable_count"`
}

type RuleDimensionPreviewResult struct {
	Batch        ImportBatch          `json:"batch"`
	Rows         []ImportRow          `json:"rows"`
	Values       []RuleDimensionValue `json:"values"`
	DisableCount int                  `json:"disable_count"`
}

type RuleDimensionValue struct {
	Dimension    ImportDimension `json:"dimension"`
	SerialNumber string          `json:"serial_number,omitempty"`
	Value        string          `json:"value,omitempty"`
	IPRange      *IPRange        `json:"ip_range,omitempty"`
	GeoBounds    *GeoBounds      `json:"geo_bounds,omitempty"`
}

type RuleDimensionPreview struct {
	Rows         []ImportRow
	Values       []RuleDimensionValue
	TotalCount   int
	ValidCount   int
	InvalidCount int
}

type ImportBatchDetail struct {
	Batch ImportBatch `json:"batch"`
	Rows  []ImportRow `json:"rows"`
}

type ImportPreviewService interface {
	PreviewAccessList(context.Context, PolicyActor, ImportPreviewRequest) (ImportPreviewResult, error)
	PreviewRuleDimension(context.Context, PolicyActor, ImportPreviewRequest) (RuleDimensionPreviewResult, error)
}

type RuleDimensionGovernanceService interface {
	ExportRuleDimension(context.Context, PolicyActor, string, string, ImportDimension) ([]byte, error)
	ClearRuleDimension(context.Context, PolicyActor, string, string, ImportDimension) (ImportBatch, error)
}

type ImportBatchMutationService interface {
	CommitImport(context.Context, PolicyActor, string) (ImportBatch, error)
	RollbackImport(context.Context, PolicyActor, string) (ImportBatch, error)
}

type ImportCommitOptions struct {
	ConfirmReplaceWithInvalid bool `json:"confirm_replace_with_invalid"`
}

type ConfirmedImportBatchMutationService interface {
	CommitImportConfirmed(context.Context, PolicyActor, string, ImportCommitOptions) (ImportBatch, error)
}

type ImportBatchReadService interface {
	ListImportBatches(context.Context, PolicyActor, ImportBatchListFilter) ([]ImportBatch, int64, error)
	GetImportBatch(context.Context, PolicyActor, string) (ImportBatchDetail, error)
	ListImportErrors(context.Context, PolicyActor, string) ([]ImportRow, error)
}

type ImportBatchListFilter struct {
	Carrier               string
	Type                  ImportType
	TargetPolicyVersionID string
	TargetRuleID          string
	Dimension             ImportDimension
	Page                  int
	PageSize              int
}

type ImportRow struct {
	ID               string          `json:"id"`
	BatchID          string          `json:"batch_id"`
	RowNumber        int             `json:"row_number"`
	RawValue         map[string]any  `json:"raw_value,omitempty"`
	NormalizedValue  map[string]any  `json:"normalized_value,omitempty"`
	ValidationStatus ImportRowStatus `json:"validation_status"`
	ErrorCode        string          `json:"error_code,omitempty"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	TargetID         string          `json:"target_id,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}
