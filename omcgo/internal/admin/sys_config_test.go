package admin

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// stubSysConfigRepo 是 SysConfigRepository 的 in-memory mock，只实现本测试需要的方法。
type stubSysConfigRepo struct {
	mu sync.Mutex

	createFn      func(ctx context.Context, cfg *SysConfig) error
	getByIDFn     func(ctx context.Context, id uuid.UUID) (*SysConfig, error)
	updateFn      func(ctx context.Context, cfg *SysConfig) error
	deleteFn      func(ctx context.Context, id uuid.UUID) error
	batchUpsertFn func(ctx context.Context, category string, items []BatchItem) (int, error)
	listFn        func(ctx context.Context, category string, publicOnly bool) ([]SysConfig, error)
}

type atomicValidationSysConfigRepo struct {
	stubSysConfigRepo
	called bool
}

func (r *atomicValidationSysConfigRepo) BatchUpsertWithApplyValidated(
	_ context.Context,
	category string,
	items []BatchItem,
	targets []ConfigApplyTarget,
	validator SysConfigCategoryValidator,
) (BatchUpsertResult, error) {
	r.called = true
	values := make(map[string]string, len(items))
	for _, item := range items {
		values[item.Key] = item.Value
	}
	if err := validator(values); err != nil {
		return BatchUpsertResult{}, err
	}
	return BatchUpsertResult{
		Updated: len(items),
		Batch: ConfigApplyBatch{
			ID:       uuid.New(),
			Category: category,
			Status:   summarizeConfigApplyStatus(targets),
			Targets:  targets,
		},
	}, nil
}

func (s *stubSysConfigRepo) Create(ctx context.Context, cfg *SysConfig) error {
	if s.createFn != nil {
		return s.createFn(ctx, cfg)
	}
	return nil
}
func (s *stubSysConfigRepo) GetByID(ctx context.Context, id uuid.UUID) (*SysConfig, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (s *stubSysConfigRepo) GetByKey(_ context.Context, _, _ string) (*SysConfig, error) {
	return nil, nil
}

func (s *stubSysConfigRepo) List(ctx context.Context, category string, publicOnly bool) ([]SysConfig, error) {
	if s.listFn != nil {
		return s.listFn(ctx, category, publicOnly)
	}
	return nil, nil
}
func (s *stubSysConfigRepo) Update(ctx context.Context, cfg *SysConfig) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, cfg)
	}
	return nil
}
func (s *stubSysConfigRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

func (s *stubSysConfigRepo) BatchUpsert(ctx context.Context, category string, items []BatchItem) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.batchUpsertFn != nil {
		return s.batchUpsertFn(ctx, category, items)
	}
	return len(items), nil
}

func TestSysConfigService_BatchUpsert_FiresHooks(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})

	var calls atomic.Int32
	var seenCategory atomic.Value
	svc.RegisterSavedHook(func(_ context.Context, category string) {
		calls.Add(1)
		seenCategory.Store(category)
	})

	n, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "device",
		Items:    []BatchItem{{Key: "periodicSyncEnabled", Value: "true"}},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, int32(1), calls.Load(), "hook 应被触发 1 次")
	assert.Equal(t, "device", seenCategory.Load())
}

func TestSysConfigService_BatchUpsert_MultipleHooksFireInOrder(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})

	var order []string
	var orderMu sync.Mutex
	for _, name := range []string{"a", "b", "c"} {
		n := name
		svc.RegisterSavedHook(func(_ context.Context, _ string) {
			orderMu.Lock()
			defer orderMu.Unlock()
			order = append(order, n)
		})
	}

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "security",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, order, "hook 按注册顺序串行执行")
}

func TestSysConfigService_BatchUpsert_HookPanicIsIsolated(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})

	var hook2Called atomic.Bool
	svc.RegisterSavedHook(func(_ context.Context, _ string) {
		panic("hook 1 boom")
	})
	svc.RegisterSavedHook(func(_ context.Context, _ string) {
		hook2Called.Store(true)
	})

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "device",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.NoError(t, err, "hook panic 不应外溢到 caller")
	assert.True(t, hook2Called.Load(), "前面 hook panic 不应阻塞后续 hook")
}

func TestSysConfigService_BatchUpsert_RepoErrorSkipsHooks(t *testing.T) {
	repo := &stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, _ []BatchItem) (int, error) {
			return 0, errors.New("pg connection lost")
		},
	}
	svc := NewSysConfigService(repo)

	var hookCalled atomic.Bool
	svc.RegisterSavedHook(func(_ context.Context, _ string) {
		hookCalled.Store(true)
	})

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "device",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.Error(t, err)
	assert.False(t, hookCalled.Load(), "BatchUpsert 失败时 hook 不应触发")
}

func TestSysConfigService_RegisterSavedHook_NilSafe(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})
	svc.RegisterSavedHook(nil) // 不应 panic
	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "device",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.NoError(t, err)
}

func TestSysConfigService_ListPublic_SecurityUsesStrictAllowlist(t *testing.T) {
	repo := &stubSysConfigRepo{
		listFn: func(_ context.Context, category string, publicOnly bool) ([]SysConfig, error) {
			assert.Equal(t, "security", category)
			assert.False(t, publicOnly, "security 分类需先读取后按安全白名单过滤")
			return []SysConfig{
				{Category: "security", Key: "isBrowserAutoRecordPass", Value: "true", IsPublic: false},
				{Category: "security", Key: "defaultPasswd", Value: "secret", IsPublic: false},
				{Category: "security", Key: "legacyPublicKey", Value: "legacy", IsPublic: true},
			}, nil
		},
	}

	items, err := NewSysConfigService(repo).ListPublic(context.Background(), "security")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "isBrowserAutoRecordPass", items[0].Key)
}

func TestSysConfigService_Create_RejectsClientControlledPublicFlag(t *testing.T) {
	called := false
	requestedPublic := true
	svc := NewSysConfigService(&stubSysConfigRepo{
		createFn: func(_ context.Context, _ *SysConfig) error {
			called = true
			return nil
		},
	})

	_, err := svc.Create(context.Background(), CreateSysConfigRequest{
		Category: "system",
		Key:      "system_name",
		Value:    "OMC",
		IsPublic: &requestedPublic,
	})

	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, called)
}

func TestSysConfigService_Create_RejectsSecretThroughDirectCRUD(t *testing.T) {
	called := false
	svc := NewSysConfigService(&stubSysConfigRepo{
		createFn: func(_ context.Context, _ *SysConfig) error {
			called = true
			return nil
		},
	})

	_, err := svc.Create(context.Background(), CreateSysConfigRequest{
		Category: "security",
		Key:      "defaultPasswd",
		Value:    "secret",
	})

	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, called)
}

func TestSysConfigService_Create_RejectsDirectMutation(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})

	_, err := svc.Create(context.Background(), CreateSysConfigRequest{
		Category: "system",
		Key:      "system_name",
		Value:    "OMC",
	})

	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestSysConfigService_Update_RejectsClientControlledPublicFlag(t *testing.T) {
	requestedPublic := true
	getCalled := false
	svc := NewSysConfigService(&stubSysConfigRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*SysConfig, error) {
			getCalled = true
			return &SysConfig{}, nil
		},
	})

	_, err := svc.Update(context.Background(), uuid.New(), UpdateSysConfigRequest{IsPublic: &requestedPublic})

	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, getCalled)
}

func TestSysConfigService_Update_RejectsSecretThroughDirectCRUD(t *testing.T) {
	updated := false
	value := "new secret"
	svc := NewSysConfigService(&stubSysConfigRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*SysConfig, error) {
			return &SysConfig{Category: "agent", Key: "agent_studio_service_token", Value: "old"}, nil
		},
		updateFn: func(_ context.Context, _ *SysConfig) error {
			updated = true
			return nil
		},
	})

	_, err := svc.Update(context.Background(), uuid.New(), UpdateSysConfigRequest{Value: &value})

	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, updated)
}

func TestSysConfigService_Update_RejectsDirectMutation(t *testing.T) {
	value := "new value"
	svc := NewSysConfigService(&stubSysConfigRepo{})

	_, err := svc.Update(context.Background(), uuid.New(), UpdateSysConfigRequest{Value: &value})

	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
}

func TestSysConfigService_Delete_RejectsSecretThroughDirectCRUD(t *testing.T) {
	id := uuid.New()
	deleted := false
	svc := NewSysConfigService(&stubSysConfigRepo{
		getByIDFn: func(_ context.Context, gotID uuid.UUID) (*SysConfig, error) {
			assert.Equal(t, id, gotID)
			return &SysConfig{ID: id, Category: "agent", Key: "agent_studio_service_token"}, nil
		},
		deleteFn: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	})

	err := svc.Delete(context.Background(), id)

	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, deleted)
}

// ── Validator hook（issue #548 切片 2 D 后端 sys_configs validator）─────────────────

// TestSysConfigService_RegisterValidator_RejectsInvalidValue：
// validator 返错时 BatchUpsert 应整批不落库，且错误链路含 ErrInvalidInput（handler 翻 400）。
func TestSysConfigService_RegisterValidator_RejectsInvalidValue(t *testing.T) {
	var repoCalled atomic.Bool
	repo := &stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, items []BatchItem) (int, error) {
			repoCalled.Store(true)
			return len(items), nil
		},
	}
	svc := NewSysConfigService(repo)

	var hookCalled atomic.Bool
	svc.RegisterSavedHook(func(_ context.Context, _ string) {
		hookCalled.Store(true)
	})

	svc.RegisterValidator("storage", "minio_public_endpoint", func(v string) error {
		if v == "" {
			return nil
		}
		// 模拟"禁带 scheme"语义
		if len(v) >= 7 && v[:7] == "http://" {
			return errors.New("scheme not allowed")
		}
		return nil
	})

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "storage",
		Items: []BatchItem{
			{Key: "minio_public_endpoint", Value: "http://bad"},
		},
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, commonerrors.ErrInvalidInput),
		"validator 错应包 ErrInvalidInput 让 handler 翻 400，实际: %v", err)
	assert.False(t, repoCalled.Load(), "validator 失败时 repo.BatchUpsert 不应被调用（整批不落库）")
	assert.False(t, hookCalled.Load(), "validator 失败时 SavedHook 不应触发")
}

// TestSysConfigService_RegisterValidator_PassesValidValue：合法值放行 + repo + hook 都跑。
func TestSysConfigService_RegisterValidator_PassesValidValue(t *testing.T) {
	var repoCalled atomic.Bool
	repo := &stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, items []BatchItem) (int, error) {
			repoCalled.Store(true)
			return len(items), nil
		},
	}
	svc := NewSysConfigService(repo)
	var hookCalled atomic.Bool
	svc.RegisterSavedHook(func(_ context.Context, _ string) { hookCalled.Store(true) })

	svc.RegisterValidator("storage", "minio_public_endpoint", func(_ string) error { return nil })

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "storage",
		Items:    []BatchItem{{Key: "minio_public_endpoint", Value: "ok-host:9000"}},
	})
	require.NoError(t, err)
	assert.True(t, repoCalled.Load(), "validator 通过后 repo.BatchUpsert 应被调用")
	assert.True(t, hookCalled.Load(), "validator 通过 + repo 成功 → SavedHook 应触发")
}

// TestSysConfigService_RegisterValidator_OnlyMatchesRegisteredKey：未注册 key 通用 KV 直通。
func TestSysConfigService_RegisterValidator_OnlyMatchesRegisteredKey(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})
	called := atomic.Bool{}
	svc.RegisterValidator("storage", "minio_public_endpoint", func(_ string) error {
		called.Store(true)
		return errors.New("should not be invoked")
	})

	// 同 category 但不同 key → 不应触发该 validator
	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "storage",
		Items:    []BatchItem{{Key: "other_key", Value: "anything"}},
	})
	require.NoError(t, err)
	assert.False(t, called.Load())

	// 不同 category 同 key → 也不应触发
	_, err = svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "other_category",
		Items:    []BatchItem{{Key: "minio_public_endpoint", Value: "anything"}},
	})
	require.NoError(t, err)
	assert.False(t, called.Load())
}

// TestSysConfigService_RegisterValidator_NilFnDeletes：传 nil 等价删除该 key 的 validator。
func TestSysConfigService_RegisterValidator_NilFnDeletes(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})
	svc.RegisterValidator("storage", "k", func(_ string) error { return errors.New("reject") })

	// 先确认会拒绝
	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "storage",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.Error(t, err)

	// 注册 nil → 删除
	svc.RegisterValidator("storage", "k", nil)
	_, err = svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "storage",
		Items:    []BatchItem{{Key: "k", Value: "v"}},
	})
	require.NoError(t, err)
}

func TestBatchUpsertRejectsInvalidACSTransferProtocolPolicy(t *testing.T) {
	var repoCalled atomic.Bool
	repo := &stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, items []BatchItem) (int, error) {
			repoCalled.Store(true)
			return len(items), nil
		},
	}
	svc := NewSysConfigService(repo)
	svc.RegisterCategoryValidator(transfercfg.Category, transfercfg.ValidateConfig)

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: transfercfg.Category,
		Items: []BatchItem{
			{Key: transfercfg.KeyProtocolPolicy, Value: "automatic"},
		},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, repoCalled.Load(), "非法协议策略不得写入批次中的任何配置")
}

func TestBatchUpsertDelegatesCategoryValidationToAtomicRepository(t *testing.T) {
	repo := &atomicValidationSysConfigRepo{}
	repo.listFn = func(context.Context, string, bool) ([]SysConfig, error) {
		return nil, errors.New("category validation escaped the repository transaction")
	}
	svc := NewSysConfigService(repo)
	validatorCalled := false
	svc.RegisterCategoryValidator("atomic", func(values map[string]string) error {
		validatorCalled = true
		assert.Equal(t, "value", values["key"])
		return nil
	})

	result, err := svc.BatchUpsertWithResult(context.Background(), BatchUpdateSysConfigRequest{
		Category: "atomic",
		Items:    []BatchItem{{Key: "key", Value: "value"}},
	})

	require.NoError(t, err)
	assert.True(t, repo.called)
	assert.True(t, validatorCalled)
	assert.Equal(t, 1, result.Updated)
}

func TestBatchUpsertRejectsPreferHTTPSWithoutBothHTTPSAddresses(t *testing.T) {
	var repoCalled atomic.Bool
	repo := &stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, items []BatchItem) (int, error) {
			repoCalled.Store(true)
			return len(items), nil
		},
	}
	svc := NewSysConfigService(repo)
	svc.RegisterCategoryValidator(transfercfg.Category, transfercfg.ValidateConfig)

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: transfercfg.Category,
		Items: []BatchItem{
			{Key: transfercfg.KeyProtocolPolicy, Value: transfercfg.ProtocolPolicyPreferHTTPS},
		},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, repoCalled.Load(), "HTTPS 地址不完整时不得写入策略")
}

func TestBatchUpsertRejectsHTTPAddressForPreferHTTPS(t *testing.T) {
	var repoCalled atomic.Bool
	repo := &stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, items []BatchItem) (int, error) {
			repoCalled.Store(true)
			return len(items), nil
		},
	}
	svc := NewSysConfigService(repo)
	svc.RegisterCategoryValidator(transfercfg.Category, transfercfg.ValidateConfig)

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: transfercfg.Category,
		Items: []BatchItem{
			{Key: transfercfg.KeyProtocolPolicy, Value: transfercfg.ProtocolPolicyPreferHTTPS},
			{Key: transfercfg.KeyHTTPSUploadBaseURL, Value: "http://acs.example.com:8080"},
			{Key: transfercfg.KeyHTTPSDownloadBaseURL, Value: "https://acs.example.com:8443"},
		},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, repoCalled.Load(), "HTTPS 字段使用 HTTP scheme 时不得写入任何配置")
}

func TestBatchUpsertRejectsNonHTTPSOptionalAddressForForceHTTP(t *testing.T) {
	var repoCalled atomic.Bool
	repo := &stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, items []BatchItem) (int, error) {
			repoCalled.Store(true)
			return len(items), nil
		},
	}
	svc := NewSysConfigService(repo)
	svc.RegisterCategoryValidator(transfercfg.Category, transfercfg.ValidateConfig)

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: transfercfg.Category,
		Items: []BatchItem{
			{Key: transfercfg.KeyProtocolPolicy, Value: transfercfg.ProtocolPolicyForceHTTP},
			{Key: transfercfg.KeyHTTPSUploadBaseURL, Value: "http://acs.example.com:8080"},
		},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, repoCalled.Load(), "force_http 下填写的 HTTPS 地址非法时仍不得写入")
}

func TestBatchUpsertRejectsHTTPSAddressInHTTPTransferField(t *testing.T) {
	var repoCalled atomic.Bool
	repo := &stubSysConfigRepo{
		batchUpsertFn: func(_ context.Context, _ string, items []BatchItem) (int, error) {
			repoCalled.Store(true)
			return len(items), nil
		},
	}
	svc := NewSysConfigService(repo)
	svc.RegisterValidator(transfercfg.Category, transfercfg.KeyUploadBaseURL, transfercfg.ValidateHTTPBaseURL)
	svc.RegisterCategoryValidator(transfercfg.Category, transfercfg.ValidateConfig)

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: transfercfg.Category,
		Items: []BatchItem{
			{Key: transfercfg.KeyUploadBaseURL, Value: "https://acs.example.com:8443"},
		},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, repoCalled.Load(), "HTTP 地址字段使用 HTTPS scheme 时不得写入")
}

func TestBatchUpsertRejectsPartialUpdateWhenPersistedHTTPFieldUsesHTTPS(t *testing.T) {
	var repoCalled atomic.Bool
	repo := &stubSysConfigRepo{
		listFn: func(_ context.Context, category string, _ bool) ([]SysConfig, error) {
			return []SysConfig{
				{Category: category, Key: transfercfg.KeyUploadBaseURL, Value: "https://legacy.example.com:8443/upload"},
			}, nil
		},
		batchUpsertFn: func(_ context.Context, _ string, items []BatchItem) (int, error) {
			repoCalled.Store(true)
			return len(items), nil
		},
	}
	svc := NewSysConfigService(repo)
	svc.RegisterCategoryValidator(transfercfg.Category, transfercfg.ValidateConfig)

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: transfercfg.Category,
		Items: []BatchItem{
			{Key: transfercfg.KeyProtocolPolicy, Value: transfercfg.ProtocolPolicyForceHTTP},
		},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, repoCalled.Load(), "部分更新不得保留违反 HTTP 字段语义的存量值")
}

func TestBatchUpsertAllowsPreferHTTPSUsingPersistedHTTPSAddresses(t *testing.T) {
	repo := &stubSysConfigRepo{
		listFn: func(_ context.Context, category string, _ bool) ([]SysConfig, error) {
			return []SysConfig{
				{Category: category, Key: transfercfg.KeyHTTPSUploadBaseURL, Value: "https://acs.example.com:8443/upload"},
				{Category: category, Key: transfercfg.KeyHTTPSDownloadBaseURL, Value: "https://acs.example.com:8443/download"},
			}, nil
		},
	}
	svc := NewSysConfigService(repo)
	svc.RegisterCategoryValidator(transfercfg.Category, transfercfg.ValidateConfig)

	updated, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: transfercfg.Category,
		Items: []BatchItem{
			{Key: transfercfg.KeyProtocolPolicy, Value: transfercfg.ProtocolPolicyPreferHTTPS},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, updated)
}

func TestBatchUpsertAllowsForceHTTPWithoutHTTPSAddresses(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})
	svc.RegisterCategoryValidator(transfercfg.Category, transfercfg.ValidateConfig)

	updated, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: transfercfg.Category,
		Items: []BatchItem{
			{Key: transfercfg.KeyProtocolPolicy, Value: transfercfg.ProtocolPolicyForceHTTP},
			{Key: transfercfg.KeyHTTPSUploadBaseURL, Value: ""},
			{Key: transfercfg.KeyHTTPSDownloadBaseURL, Value: ""},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 3, updated)
}

func TestBatchItemsToApplyState_RedactsSecrets(t *testing.T) {
	state := batchItemsToApplyState("security", []BatchItem{
		{Key: "defaultPasswd", Value: "plaintext-password"},
		{Key: "isBrowserAutoRecordPass", Value: "true"},
	})

	assert.Equal(t, "[REDACTED]", state["defaultPasswd"])
	assert.Equal(t, "true", state["isBrowserAutoRecordPass"])
	assert.NotContains(t, state, "plaintext-password")

	acsState := batchItemsToApplyState("acs_transfer", []BatchItem{
		{Key: "uploadPassword", Value: "upload-secret"},
		{Key: "downloadPassword", Value: "download-secret"},
	})
	assert.Equal(t, "[REDACTED]", acsState["uploadPassword"])
	assert.Equal(t, "[REDACTED]", acsState["downloadPassword"])
}

func TestBatchUpsertPreservesBlankACSCredentials(t *testing.T) {
	var persisted []BatchItem
	svc := NewSysConfigService(&stubSysConfigRepo{batchUpsertFn: func(_ context.Context, _ string, items []BatchItem) (int, error) {
		persisted = append([]BatchItem(nil), items...)
		return len(items), nil
	}})

	updated, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "acs_transfer",
		Items: []BatchItem{
			{Key: "uploadPassword", Value: ""},
			{Key: "downloadPassword", Value: "rotated"},
			{Key: "uploadBaseURL", Value: "https://acs.example.com"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, 2, updated)
	require.Equal(t, []BatchItem{
		{Key: "downloadPassword", Value: "rotated"},
		{Key: "uploadBaseURL", Value: "https://acs.example.com"},
	}, persisted)
}

func TestBatchUpsertRejectsAllBlankACSCredentialPlaceholders(t *testing.T) {
	repoCalled := false
	svc := NewSysConfigService(&stubSysConfigRepo{batchUpsertFn: func(_ context.Context, _ string, _ []BatchItem) (int, error) {
		repoCalled = true
		return 0, nil
	}})

	_, err := svc.BatchUpsert(context.Background(), BatchUpdateSysConfigRequest{
		Category: "acs_transfer",
		Items: []BatchItem{
			{Key: "uploadPassword", Value: ""},
			{Key: "downloadPassword", Value: ""},
		},
	})

	assert.ErrorIs(t, err, commonerrors.ErrInvalidInput)
	assert.False(t, repoCalled)
}

func TestConfigApplyTargetsIncludeRequiredTargetsEvenWhenModuleDependencyIsMissing(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})
	svc.RegisterApplyHandler("storage", "alarm_history_retention", func(context.Context, ConfigApplyWork) (map[string]any, error) {
		return nil, nil
	})

	storageTargets := svc.configApplyTargetsForCategory("storage")
	require.Equal(t, []string{"alarm_history_retention", "minio_presign_endpoint"}, []string{
		storageTargets[0].Target, storageTargets[1].Target,
	})
	minioTargets := svc.configApplyTargetsForCategory("minio.retention")
	require.Len(t, minioTargets, 1)
	require.Equal(t, "minio_raw_file_lifecycle", minioTargets[0].Target)
}

func TestACSConfigApplyTargetOnlyClaimsEventDelivery(t *testing.T) {
	svc := NewSysConfigService(&stubSysConfigRepo{})

	targets := svc.configApplyTargetsForCategory("acs_transfer")

	require.Len(t, targets, 1)
	require.Equal(t, "acs_transfer_event_delivery", targets[0].Target)
	require.Equal(t, ConfigApplySuccessScopeEventDelivered, targets[0].SuccessScope)
}
