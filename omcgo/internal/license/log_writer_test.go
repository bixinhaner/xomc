package license

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// memLogRepo 是 LicenseLogRepository 的内存实现，用于 LogWriter 单测。
type memLogRepo struct {
	mu      sync.Mutex
	logs    []LicenseLog
	failOn  map[LogType]error // 模拟 Create 失败
	created chan LicenseLog
}

func newMemLogRepo() *memLogRepo {
	return &memLogRepo{
		failOn:  make(map[LogType]error),
		created: make(chan LicenseLog, 100),
	}
}

func (m *memLogRepo) Create(_ context.Context, log *LicenseLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err, ok := m.failOn[log.LogType]; ok {
		return err
	}
	m.logs = append(m.logs, *log)
	select {
	case m.created <- *log:
	default:
	}
	return nil
}

func (m *memLogRepo) List(_ context.Context, filter LicenseLogFilter) (*model.ListResponse[LicenseLog], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	filtered := make([]LicenseLog, 0, len(m.logs))
	for _, l := range m.logs {
		if filter.LicenseID != nil {
			if l.LicenseID == nil || *l.LicenseID != *filter.LicenseID {
				continue
			}
		}
		if filter.ActorUserID != nil {
			if l.ActorUserID == nil || *l.ActorUserID != *filter.ActorUserID {
				continue
			}
		}
		if len(filter.LogTypes) > 0 {
			match := false
			for _, t := range filter.LogTypes {
				if l.LogType == t {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if len(filter.Results) > 0 {
			match := false
			for _, r := range filter.Results {
				if l.Result == r {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if filter.StartTime != nil && l.CreatedAt.Before(*filter.StartTime) {
			continue
		}
		if filter.EndTime != nil && l.CreatedAt.After(*filter.EndTime) {
			continue
		}
		filtered = append(filtered, l)
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	return model.NewListResponse(filtered, int64(len(filtered)), page, pageSize), nil
}

func (m *memLogRepo) ListByLicense(_ context.Context, licenseID uuid.UUID, limit int) ([]LicenseLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]LicenseLog, 0, limit)
	for _, l := range m.logs {
		if l.LicenseID != nil && *l.LicenseID == licenseID {
			out = append(out, l)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (m *memLogRepo) snapshot() []LicenseLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]LicenseLog, len(m.logs))
	copy(cp, m.logs)
	return cp
}

// ---------------------------------------------------------------------------
// NoopLogWriter — 不应炸，不应入库，不应 panic
// ---------------------------------------------------------------------------

func TestNoopLogWriter_DoesNothing(t *testing.T) {
	w := NoopLogWriter{}
	require.NotPanics(t, func() {
		w.Write(context.Background(), LicenseLogEntry{
			LogType: LogTypeImport,
			Result:  LogResultSuccess,
		})
	})
}

// ---------------------------------------------------------------------------
// pgLogWriter happy path — 9 种 log_type × 4 种 result 全覆盖
// ---------------------------------------------------------------------------

func TestPgLogWriter_AllLogTypesAndResults(t *testing.T) {
	cases := []struct {
		name    string
		logType LogType
		result  LogResult
	}{
		{"import-success", LogTypeImport, LogResultSuccess},
		{"import-failed", LogTypeImport, LogResultFailed},
		{"activate-success", LogTypeActivate, LogResultSuccess},
		{"revoke-success", LogTypeRevoke, LogResultSuccess},
		{"query-detail", LogTypeQueryDetail, LogResultSuccess},
		{"enforcement-capacity", LogTypeEnforcementCapacity, LogResultDenied},
		{"enforcement-expiry", LogTypeEnforcementExpiry, LogResultDenied},
		{"capacity-alert", LogTypeCapacityAlert, LogResultWarning},
		{"expiry-alert", LogTypeExpiryAlert, LogResultWarning},
		{"auto-expire", LogTypeAutoExpire, LogResultSuccess},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMemLogRepo()
			w := NewLogWriter(repo, zap.NewNop())
			licID := uuid.New()
			actorID := uuid.New()

			w.Write(context.Background(), LicenseLogEntry{
				LicenseID:   &licID,
				LogType:     tc.logType,
				ActorUserID: &actorID,
				Result:      tc.result,
				Details: map[string]any{
					"summary": "test " + tc.name,
				},
				ClientIP:  "127.0.0.1",
				UserAgent: "test-agent",
			})

			got := repo.snapshot()
			require.Len(t, got, 1)
			assert.Equal(t, tc.logType, got[0].LogType)
			assert.Equal(t, tc.result, got[0].Result)
			assert.Equal(t, &licID, got[0].LicenseID)
			assert.Equal(t, &actorID, got[0].ActorUserID)
			require.NotNil(t, got[0].ClientIP)
			assert.Equal(t, "127.0.0.1", *got[0].ClientIP)
			require.NotNil(t, got[0].UserAgent)
			assert.Equal(t, "test-agent", *got[0].UserAgent)

			// details JSON 至少含 summary 字段
			var details map[string]any
			require.NoError(t, json.Unmarshal(got[0].Details, &details))
			assert.Contains(t, details["summary"], tc.name)
		})
	}
}

// ---------------------------------------------------------------------------
// system 操作（actor=nil）+ license 已删（license_id=nil）边界
// ---------------------------------------------------------------------------

func TestPgLogWriter_SystemOperationNilActor(t *testing.T) {
	repo := newMemLogRepo()
	w := NewLogWriter(repo, zap.NewNop())

	w.Write(context.Background(), LicenseLogEntry{
		LogType: LogTypeAutoExpire,
		Result:  LogResultSuccess,
		Details: map[string]any{"summary": "cron auto-expire"},
		// 无 ActorUserID / 无 ClientIP / 无 UserAgent — 模拟 cron 触发
	})

	got := repo.snapshot()
	require.Len(t, got, 1)
	assert.Nil(t, got[0].ActorUserID, "system 操作 actor 必须 nil")
	assert.Nil(t, got[0].ClientIP)
	assert.Nil(t, got[0].UserAgent)
}

func TestPgLogWriter_NilLicenseID(t *testing.T) {
	repo := newMemLogRepo()
	w := NewLogWriter(repo, zap.NewNop())

	w.Write(context.Background(), LicenseLogEntry{
		LogType: LogTypeImport,
		Result:  LogResultFailed,
		Details: map[string]any{
			"summary":      "import failed before license created",
			"license_code": "BAD-CODE",
		},
	})

	got := repo.snapshot()
	require.Len(t, got, 1)
	assert.Nil(t, got[0].LicenseID, "import 失败时 license_id 必须 nil")
}

// ---------------------------------------------------------------------------
// repo 写库失败 — 不阻断，仅 warn（永不 return error）
// ---------------------------------------------------------------------------

func TestPgLogWriter_RepoFailure_NonBlocking(t *testing.T) {
	repo := newMemLogRepo()
	repo.failOn[LogTypeImport] = errors.New("simulated db down")
	w := NewLogWriter(repo, zap.NewNop())

	require.NotPanics(t, func() {
		w.Write(context.Background(), LicenseLogEntry{
			LogType: LogTypeImport,
			Result:  LogResultSuccess,
			Details: map[string]any{"summary": "import"},
		})
	})

	// 即使 repo.Create 返回 error，write 仍然应返回（不 panic、不阻塞）
	assert.Empty(t, repo.snapshot(), "失败时不应有日志入库")
}

// ---------------------------------------------------------------------------
// Details marshal 失败 — 降级 `{}` 不阻断主业务
// ---------------------------------------------------------------------------

func TestPgLogWriter_DetailsMarshalFailure_FallbackEmpty(t *testing.T) {
	repo := newMemLogRepo()
	w := NewLogWriter(repo, zap.NewNop())

	// channel 不能 JSON marshal，会触发 fallback 路径
	unmarshallable := map[string]any{
		"chan": make(chan int),
	}
	w.Write(context.Background(), LicenseLogEntry{
		LogType: LogTypeImport,
		Result:  LogResultSuccess,
		Details: unmarshallable,
	})

	got := repo.snapshot()
	require.Len(t, got, 1, "marshal 失败仍应入库（details 降级 `{}`）")
	assert.JSONEq(t, `{}`, string(got[0].Details))
}

// ---------------------------------------------------------------------------
// 空 Details — 也应降级 `{}` 而非 null
// ---------------------------------------------------------------------------

func TestPgLogWriter_EmptyDetails_NormalizedToEmptyObject(t *testing.T) {
	repo := newMemLogRepo()
	w := NewLogWriter(repo, zap.NewNop())

	w.Write(context.Background(), LicenseLogEntry{
		LogType: LogTypeImport,
		Result:  LogResultSuccess,
		// 没传 Details
	})

	got := repo.snapshot()
	require.Len(t, got, 1)
	assert.JSONEq(t, `{}`, string(got[0].Details))
}

// ---------------------------------------------------------------------------
// 并发写 — 确保 LogWriter 在并发场景下不丢日志
// ---------------------------------------------------------------------------

func TestPgLogWriter_ConcurrentWrites(t *testing.T) {
	repo := newMemLogRepo()
	w := NewLogWriter(repo, zap.NewNop())

	const N = 50
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			w.Write(context.Background(), LicenseLogEntry{
				LogType: LogTypeImport,
				Result:  LogResultSuccess,
				Details: map[string]any{"i": i},
			})
		}(i)
	}
	wg.Wait()

	got := repo.snapshot()
	assert.Len(t, got, N, "并发 N 次写应全部入库")
}
