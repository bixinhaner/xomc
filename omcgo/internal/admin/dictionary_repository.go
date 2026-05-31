package admin

import (
	"context"
)

// DictionaryRepository defines the persistence interface for dictionaries.
type DictionaryRepository interface {
	// Create inserts a new dictionary.
	Create(ctx context.Context, dict *Dictionary) error
	// GetByID returns a dictionary by its ID.
	GetByID(ctx context.Context, id int64) (*Dictionary, error)
	// GetByType returns a dictionary with its active details by type.
	GetByType(ctx context.Context, dictType string) (*Dictionary, error)
	// List returns all dictionaries (without details).
	List(ctx context.Context) ([]Dictionary, error)
	// Update updates a dictionary.
	Update(ctx context.Context, dict *Dictionary) error
	// Delete soft-deletes a dictionary and cascades to its details.
	Delete(ctx context.Context, id int64) error

	// ListSourceBound returns dictionaries with non-NULL source_table AND status=true.
	// Used by the daily cron job to enumerate dictionaries to refresh (T-0182).
	ListSourceBound(ctx context.Context) ([]Dictionary, error)

	// UpdateRefreshMetadata writes last_refresh_* columns after a sync run.
	// errMsg may be empty for success; count is the total auto rows after sync.
	UpdateRefreshMetadata(ctx context.Context, dictID int64, status, errMsg string, count int) error
}

// DictionaryDetailRepository defines the persistence interface for dictionary details.
type DictionaryDetailRepository interface {
	// Create inserts a new dictionary detail.
	Create(ctx context.Context, detail *DictionaryDetail) error
	// GetByID returns a detail by its ID.
	GetByID(ctx context.Context, id int64) (*DictionaryDetail, error)
	// List returns filtered, paginated dictionary details.
	List(ctx context.Context, req DictionaryDetailListRequest) ([]DictionaryDetail, int64, error)
	// Update updates a dictionary detail.
	Update(ctx context.Context, detail *DictionaryDetail) error
	// Delete soft-deletes a dictionary detail.
	Delete(ctx context.Context, id int64) error
	// DeleteByDictionaryID soft-deletes all details for a dictionary.
	DeleteByDictionaryID(ctx context.Context, dictID int64) error
	// ListSubtreeIDs 返回 rootID + 它所有未删除的后代 ID（含自身）。
	// 用于「换父」时防环 / 深度校验 / level 级联。
	ListSubtreeIDs(ctx context.Context, rootID int64) ([]int64, error)
	// ShiftLevelDelta 给指定 id 集合的 level 各加 delta。
	// 用于换父时把整个子树 level 平移。
	ShiftLevelDelta(ctx context.Context, ids []int64, delta int) error

	// === T-0182 数据源同步专用方法 ===

	// UpsertAutoBatch 用 (sys_dictionary_id, value, origin='auto') 自然键 upsert
	// 一批 auto 行;返回 (inserted, updated) 计数(按 RETURNING xmax=0 区分)。
	// status=true, sort=0, parent_id=NULL, level=0 (托管字典强制扁平)。
	UpsertAutoBatch(ctx context.Context, dictID int64, rows []AutoDetailRow) (inserted, updated int, err error)
	// DeleteAutoNotIn 删除 (sys_dictionary_id, origin='auto') 中 value 不在
	// keepValues 集合的行(软删 deleted_at=now)。
	DeleteAutoNotIn(ctx context.Context, dictID int64, keepValues []string) (deleted int, err error)
	// CountAutoActive 回读当前 origin='auto' 且未软删的行数,用于写 last_refresh_count。
	CountAutoActive(ctx context.Context, dictID int64) (int, error)
}

// AutoDetailRow 是 UpsertAutoBatch 的批量输入。
type AutoDetailRow struct {
	Label string
	Value string
}
