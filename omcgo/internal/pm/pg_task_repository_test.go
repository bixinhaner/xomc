package pm

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// Verify PgTaskRepository implements TaskRepository interface at compile time.
var _ TaskRepository = (*PgTaskRepository)(nil)

// TestPMTaskColumns_CreatorCoalesced 回归 #118：pm_tasks.creator 列可空，
// SELECT 必须 COALESCE 兜底空串，否则扫描 NULL 进不可空 string 导致 List 恒 500。
func TestPMTaskColumns_CreatorCoalesced(t *testing.T) {
	sql, _, err := storage.Psql.Select(pmTaskColumns...).From("pm_tasks").ToSql()
	require.NoError(t, err)

	assert.Contains(t, sql, "COALESCE(creator, '') AS creator",
		"creator 可空列必须 COALESCE 兜底，避免 NULL 扫进 *string 失败（#118）")
}

// fakePMTaskRows 是 pgx.Rows 的最小桩实现，只为驱动 scanPMTaskRow。
type fakePMTaskRows struct {
	scanFn func(dest ...any) error
}

func (f *fakePMTaskRows) Close()                                       {}
func (f *fakePMTaskRows) Err() error                                   { return nil }
func (f *fakePMTaskRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (f *fakePMTaskRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (f *fakePMTaskRows) Next() bool                                   { return false }
func (f *fakePMTaskRows) Scan(dest ...any) error                       { return f.scanFn(dest...) }
func (f *fakePMTaskRows) Values() ([]any, error)                       { return nil, nil }
func (f *fakePMTaskRows) RawValues() [][]byte                          { return nil }
func (f *fakePMTaskRows) Conn() *pgx.Conn                              { return nil }

func TestScanPMTaskRow(t *testing.T) {
	taskID := uuid.New()
	now := time.Now()
	scanErr := errors.New("can't scan into dest[9]")

	// 按 scanPMTaskRow 的 12 个 dest 顺序填值。
	fillRow := func(creator string, deviceSNs, kpiCodes json.RawMessage) func(dest ...any) error {
		return func(dest ...any) error {
			*(dest[0].(*uuid.UUID)) = taskID
			*(dest[1].(*string)) = "smoke-task"
			*(dest[2].(*PMTaskType)) = PMTaskExtraction
			*(dest[3].(*json.RawMessage)) = deviceSNs
			*(dest[4].(*json.RawMessage)) = kpiCodes
			*(dest[5].(*string)) = "15min"
			*(dest[6].(*json.RawMessage)) = nil
			*(dest[7].(*TaskStatus)) = PMTaskPending
			*(dest[8].(*int)) = 0
			*(dest[9].(*string)) = creator
			*(dest[10].(*time.Time)) = now
			*(dest[11].(*time.Time)) = now
			return nil
		}
	}

	tests := []struct {
		name        string
		scanFn      func(dest ...any) error
		wantErr     error
		wantCreator string
		wantSNs     string
	}{
		{
			name:        "creator 经 COALESCE 兜底为空串可正常扫描",
			scanFn:      fillRow("", nil, nil),
			wantCreator: "",
			wantSNs:     "[]",
		},
		{
			name:        "正常行带 creator 与 device_sns",
			scanFn:      fillRow("admin", json.RawMessage(`["sn1"]`), json.RawMessage(`["K1001"]`)),
			wantCreator: "admin",
			wantSNs:     `["sn1"]`,
		},
		{
			name:    "scan 失败被包装返回",
			scanFn:  func(dest ...any) error { return scanErr },
			wantErr: scanErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scanPMTaskRow(&fakePMTaskRows{scanFn: tt.scanFn})
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Contains(t, err.Error(), "scan pm_tasks row")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, taskID, got.ID)
			assert.Equal(t, tt.wantCreator, got.Creator)
			assert.JSONEq(t, tt.wantSNs, string(got.DeviceSNs))
			assert.NotNil(t, got.KPICodes, "KPICodes NULL 应兜底为 []")
		})
	}
}
