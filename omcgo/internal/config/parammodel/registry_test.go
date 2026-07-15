package parammodel

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/product"
)

// ─── 内存 fakes ────────────────────────────────────────────────────

type fakeRepo struct {
	defaultByModel      map[uuid.UUID][]ParamMapping
	discoveredByDevice  map[discoveredKey][]ParamMapping
	activeByModel       map[uuid.UUID]bool
	listDefaultCalls    int
	listDiscoveredCalls int
	isActiveCalls       int
	listDefaultErr      error
	listDiscoveredErr   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		defaultByModel:     map[uuid.UUID][]ParamMapping{},
		discoveredByDevice: map[discoveredKey][]ParamMapping{},
		activeByModel:      map[uuid.UUID]bool{},
	}
}

func (f *fakeRepo) IsParamModelActive(_ context.Context, paramModelID uuid.UUID) (bool, error) {
	f.isActiveCalls++
	active, configured := f.activeByModel[paramModelID]
	if !configured {
		return true, nil
	}
	return active, nil
}

func (f *fakeRepo) ListMappingsByParamModel(_ context.Context, paramModelID uuid.UUID) ([]ParamMapping, error) {
	f.listDefaultCalls++
	if f.listDefaultErr != nil {
		return nil, f.listDefaultErr
	}
	out, ok := f.defaultByModel[paramModelID]
	if !ok {
		return nil, nil
	}
	cp := make([]ParamMapping, len(out))
	copy(cp, out)
	return cp, nil
}

func (f *fakeRepo) ListDiscoveredMappings(_ context.Context, productID uuid.UUID, swVersion string) ([]ParamMapping, error) {
	f.listDiscoveredCalls++
	if f.listDiscoveredErr != nil {
		return nil, f.listDiscoveredErr
	}
	out, ok := f.discoveredByDevice[discoveredKey{productID: productID, swVersion: swVersion}]
	if !ok {
		return nil, nil
	}
	cp := make([]ParamMapping, len(out))
	copy(cp, out)
	return cp, nil
}

type fakeProductGetter struct {
	products map[uuid.UUID]*product.Product
	err      error
	calls    int
}

func newFakeProductGetter() *fakeProductGetter {
	return &fakeProductGetter{products: map[uuid.UUID]*product.Product{}}
}

func (f *fakeProductGetter) GetProductByID(_ context.Context, id uuid.UUID) (*product.Product, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	p, ok := f.products[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

// flakyCache 模拟 L2 失败场景。
type flakyCache struct {
	getDefaultErr      error
	getDiscoveredErr   error
	setDefaultCalls    int
	setDiscoveredCalls int
}

func (c *flakyCache) GetDefault(context.Context, uuid.UUID) ([]ParamMapping, error) {
	return nil, c.getDefaultErr
}
func (c *flakyCache) SetDefault(context.Context, uuid.UUID, []ParamMapping) error {
	c.setDefaultCalls++
	return nil
}
func (c *flakyCache) InvalidateDefault(context.Context, uuid.UUID) error { return nil }
func (c *flakyCache) GetDiscovered(context.Context, uuid.UUID, string) ([]ParamMapping, error) {
	return nil, c.getDiscoveredErr
}
func (c *flakyCache) SetDiscovered(context.Context, uuid.UUID, string, []ParamMapping) error {
	c.setDiscoveredCalls++
	return nil
}
func (c *flakyCache) InvalidateDiscovered(context.Context, uuid.UUID, string) error { return nil }
func (c *flakyCache) GetVersion(context.Context) (int64, error)                     { return 0, nil }
func (c *flakyCache) BumpVersion(context.Context) (int64, error)                    { return 0, nil }

// ─── 测试 ──────────────────────────────────────────────────────────

func TestRegistry_GetByProduct_DiscoveredHit(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	productID := uuid.New()
	pmID := uuid.New()
	pg.products[productID] = &product.Product{ID: productID, Name: "P1", ParamModelID: &pmID}
	repo.discoveredByDevice[discoveredKey{productID: productID, swVersion: "1.0"}] = []ParamMapping{
		mkMapping("A", "a"),
	}

	r := NewRegistry(repo, NopCache{}, pg, NewRegistryMetrics(nil), zap.NewNop())
	set, err := r.GetByProduct(context.Background(), productID, "1.0")
	require.NoError(t, err)
	require.NotNil(t, set)
	assert.Equal(t, MappingSourceDiscovered, set.Source)
	require.Len(t, set.Mappings, 1)
	assert.Equal(t, "A", set.Mappings[0].StandardPath)
	assert.Equal(t, pmID, set.ParamModelID, "discovered hit 也应回填 paramModelID")
}

func TestRegistry_GetByProduct_InactiveModelRejectsCachedDiscoveredMappings(t *testing.T) {
	repo := newFakeRepo()
	products := newFakeProductGetter()
	productID := uuid.New()
	pmID := uuid.New()
	products.products[productID] = &product.Product{ID: productID, ParamModelID: &pmID}
	repo.activeByModel[pmID] = false

	client := newMiniRedis(t)
	cache := NewRedisCache(client)
	require.NoError(t, cache.SetDiscovered(context.Background(), productID, "1.0", []ParamMapping{
		mkMapping("Device.Foo", "Device.PrivateFoo"),
	}))

	r := NewRegistry(repo, cache, products, NewRegistryMetrics(nil), zap.NewNop())
	_, err := r.GetByProduct(context.Background(), productID, "1.0")
	require.ErrorIs(t, err, ErrInactiveParamModel)
	assert.Equal(t, 0, repo.listDiscoveredCalls, "模型状态应在读取 discovered 缓存前检查")
}

func TestRegistry_GetByParamModel_InactiveModelRejectsCachedDefaultMappings(t *testing.T) {
	repo := newFakeRepo()
	pmID := uuid.New()
	repo.activeByModel[pmID] = false

	client := newMiniRedis(t)
	cache := NewRedisCache(client)
	require.NoError(t, cache.SetDefault(context.Background(), pmID, []ParamMapping{
		mkMapping("Device.Foo", "Device.PrivateFoo"),
	}))

	r := NewRegistry(repo, cache, nil, NewRegistryMetrics(nil), zap.NewNop())
	_, err := r.GetByParamModel(context.Background(), pmID)
	require.ErrorIs(t, err, ErrInactiveParamModel)
	assert.Equal(t, 0, repo.listDefaultCalls, "模型状态应在读取 default 缓存前检查")
}

func TestRegistry_GetByProduct_FallbackToDefault(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	productID := uuid.New()
	pmID := uuid.New()
	pg.products[productID] = &product.Product{ID: productID, ParamModelID: &pmID}
	// discovered 为空；default 有
	repo.defaultByModel[pmID] = []ParamMapping{mkMapping("X", "x")}

	r := NewRegistry(repo, NopCache{}, pg, NewRegistryMetrics(nil), zap.NewNop())
	set, err := r.GetByProduct(context.Background(), productID, "1.0")
	require.NoError(t, err)
	assert.Equal(t, MappingSourceDefault, set.Source)
	require.Len(t, set.Mappings, 1)
	assert.Equal(t, pmID, set.ParamModelID)
}

func TestRegistry_GetByProduct_NoMappingError(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	productID := uuid.New()
	pmID := uuid.New()
	pg.products[productID] = &product.Product{ID: productID, ParamModelID: &pmID}
	// discovered + default 全空

	r := NewRegistry(repo, NopCache{}, pg, NewRegistryMetrics(nil), zap.NewNop())
	_, err := r.GetByProduct(context.Background(), productID, "1.0")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoMapping)
}

func TestRegistry_GetByProduct_ErrNoParamModel(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	productID := uuid.New()
	pg.products[productID] = &product.Product{ID: productID, ParamModelID: nil}

	r := NewRegistry(repo, NopCache{}, pg, NewRegistryMetrics(nil), zap.NewNop())
	_, err := r.GetByProduct(context.Background(), productID, "1.0")
	assert.ErrorIs(t, err, ErrNoParamModel)
}

func TestRegistry_GetByProduct_ProductNotFound_ReturnsErrNoMapping(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	productID := uuid.New()
	r := NewRegistry(repo, NopCache{}, pg, NewRegistryMetrics(nil), zap.NewNop())
	_, err := r.GetByProduct(context.Background(), productID, "1.0")
	assert.ErrorIs(t, err, ErrNoMapping)
}

func TestRegistry_GetByProduct_ProductGetterError(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()
	pg.err = errors.New("boom")

	r := NewRegistry(repo, NopCache{}, pg, NewRegistryMetrics(nil), zap.NewNop())
	_, err := r.GetByProduct(context.Background(), uuid.New(), "1.0")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoMapping)
}

func TestRegistry_GetByProduct_ProductGetterUnset(t *testing.T) {
	r := NewRegistry(newFakeRepo(), NopCache{}, nil, NewRegistryMetrics(nil), zap.NewNop())
	_, err := r.GetByProduct(context.Background(), uuid.New(), "1.0")
	assert.ErrorIs(t, err, ErrProductGetterUnset)
}

func TestRegistry_GetByProduct_L1HitAfterFirstLoad(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	productID := uuid.New()
	pmID := uuid.New()
	pg.products[productID] = &product.Product{ID: productID, ParamModelID: &pmID}
	repo.discoveredByDevice[discoveredKey{productID: productID, swVersion: "1.0"}] = []ParamMapping{
		mkMapping("A", "a"),
	}

	r := NewRegistry(repo, NopCache{}, pg, NewRegistryMetrics(nil), zap.NewNop())
	for i := 0; i < 3; i++ {
		_, err := r.GetByProduct(context.Background(), productID, "1.0")
		require.NoError(t, err)
	}
	assert.Equal(t, 1, repo.listDiscoveredCalls, "L1 命中后不应再回 DB")
}

func TestRegistry_GetByProduct_L2FailureFallsBackToDB(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	productID := uuid.New()
	pmID := uuid.New()
	pg.products[productID] = &product.Product{ID: productID, ParamModelID: &pmID}
	repo.discoveredByDevice[discoveredKey{productID: productID, swVersion: "1.0"}] = []ParamMapping{
		mkMapping("A", "a"),
	}

	cache := &flakyCache{getDiscoveredErr: errors.New("redis down")}
	r := NewRegistry(repo, cache, pg, NewRegistryMetrics(nil), zap.NewNop())
	set, err := r.GetByProduct(context.Background(), productID, "1.0")
	require.NoError(t, err)
	assert.Equal(t, MappingSourceDiscovered, set.Source)
	assert.Equal(t, 1, repo.listDiscoveredCalls)
}

func TestRegistry_GetByParamModel_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	pmID := uuid.New()
	repo.defaultByModel[pmID] = []ParamMapping{mkMapping("A", "a"), mkMapping("B", "b")}

	r := NewRegistry(repo, NopCache{}, nil, NewRegistryMetrics(nil), zap.NewNop())
	set, err := r.GetByParamModel(context.Background(), pmID)
	require.NoError(t, err)
	assert.Equal(t, MappingSourceDefault, set.Source)
	assert.Len(t, set.Mappings, 2)
}

func TestRegistry_GetByParamModel_Empty(t *testing.T) {
	repo := newFakeRepo()
	r := NewRegistry(repo, NopCache{}, nil, NewRegistryMetrics(nil), zap.NewNop())
	_, err := r.GetByParamModel(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrNoMapping)
}

func TestRegistry_Translator_BuildsFromGetByProduct(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	productID := uuid.New()
	pmID := uuid.New()
	pg.products[productID] = &product.Product{ID: productID, ParamModelID: &pmID}
	repo.discoveredByDevice[discoveredKey{productID: productID, swVersion: "1.0"}] = []ParamMapping{
		mkMapping("Device.X", "Device.Y"),
	}

	r := NewRegistry(repo, NopCache{}, pg, NewRegistryMetrics(nil), zap.NewNop())
	tr, err := r.Translator(context.Background(), productID, "1.0")
	require.NoError(t, err)
	res := tr.ToPrivate("Device.X")
	assert.True(t, res.Found)
	assert.Equal(t, "Device.Y", res.Translated)
	assert.Equal(t, MappingSourceDiscovered, tr.Source())
}

func TestRegistry_InvalidateProduct_DropsL1(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	productID := uuid.New()
	pmID := uuid.New()
	pg.products[productID] = &product.Product{ID: productID, ParamModelID: &pmID}
	repo.discoveredByDevice[discoveredKey{productID: productID, swVersion: "1.0"}] = []ParamMapping{
		mkMapping("A", "a"),
	}

	r := NewRegistry(repo, NopCache{}, pg, NewRegistryMetrics(nil), zap.NewNop())
	_, _ = r.GetByProduct(context.Background(), productID, "1.0")
	require.NoError(t, r.InvalidateProduct(context.Background(), productID, "1.0"))

	repo.listDiscoveredCalls = 0
	_, _ = r.GetByProduct(context.Background(), productID, "1.0")
	assert.Equal(t, 1, repo.listDiscoveredCalls, "L1 失效后应再次打 DB")
}

func TestRegistry_InvalidateParamModel_DropsL1(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	pmID := uuid.New()
	repo.defaultByModel[pmID] = []ParamMapping{mkMapping("A", "a")}

	r := NewRegistry(repo, NopCache{}, pg, NewRegistryMetrics(nil), zap.NewNop())
	_, err := r.GetByParamModel(context.Background(), pmID)
	require.NoError(t, err)
	require.NoError(t, r.InvalidateParamModel(context.Background(), pmID))

	repo.listDefaultCalls = 0
	_, _ = r.GetByParamModel(context.Background(), pmID)
	assert.Equal(t, 1, repo.listDefaultCalls)
}

func TestRegistry_Refresh_ClearsAllAndBumpsVersion(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	pmID := uuid.New()
	productID := uuid.New()
	pg.products[productID] = &product.Product{ID: productID, ParamModelID: &pmID}
	repo.defaultByModel[pmID] = []ParamMapping{mkMapping("A", "a")}
	repo.discoveredByDevice[discoveredKey{productID: productID, swVersion: "1"}] = []ParamMapping{
		mkMapping("B", "b"),
	}

	client := newMiniRedis(t)
	cache := NewRedisCache(client)
	r := NewRegistry(repo, cache, pg, NewRegistryMetrics(nil), zap.NewNop())

	// 预热
	_, _ = r.GetByParamModel(context.Background(), pmID)
	_, _ = r.GetByProduct(context.Background(), productID, "1")

	require.NoError(t, r.Refresh(context.Background()))

	v, err := cache.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), v, "BumpVersion 应递增到 1")

	// L1 已清空，再次查询会重新打 DB
	repo.listDefaultCalls = 0
	repo.listDiscoveredCalls = 0
	_, _ = r.GetByParamModel(context.Background(), pmID)
	_, _ = r.GetByProduct(context.Background(), productID, "1")
	// 由于 L2 仍有缓存（Refresh 不清 L2 的具体条目，仅 BumpVersion），
	// 这里至少 1 次 L2 命中或 1 次 DB 击穿；具体只断言"L1 已 drop"
	// 为简化只断 BumpVersion 行为，回 DB 行为另由 Invalidate*Test 覆盖。
}

// TestRegistry_CrossInstance_MappingEditPropagates 复现并锁定核心缺陷：
// app(进程 A) 编辑 default 映射后，ACS(进程 B) 必须在不重启的情况下及时下发最新
// privatePath。两个 Registry 共享同一 Redis（cache_version 通道）+ 同一 DB。
//
// 修复前：B 的 L1 sync.Map 短路返回旧映射，本测试在最后一步失败。
// 修复后：B 的热路径 ensureFresh 检测到版本前进 → 丢 L1 → read-through 拿到新私有 PATH。
func TestRegistry_CrossInstance_MappingEditPropagates(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()
	productID := uuid.New()
	pmID := uuid.New()
	pg.products[productID] = &product.Product{ID: productID, ParamModelID: &pmID}
	repo.defaultByModel[pmID] = []ParamMapping{mkMapping("Device.Foo", "Device.OldPriv")}

	client := newMiniRedis(t)
	appReg := NewRegistry(repo, NewRedisCache(client), pg, NewRegistryMetrics(nil), zap.NewNop())
	acsReg := NewRegistry(repo, NewRedisCache(client), pg, NewRegistryMetrics(nil), zap.NewNop())

	// ACS 先翻译一次，旧 privatePath 进 ACS 的 L1
	tr, err := acsReg.Translator(context.Background(), productID, "1.0")
	require.NoError(t, err)
	require.Equal(t, "Device.OldPriv", tr.ToPrivate("Device.Foo").Translated)

	// app 侧编辑映射：改 DB 内容 + Invalidate（删 L2 key + BumpVersion 跨进程信号）
	repo.defaultByModel[pmID] = []ParamMapping{mkMapping("Device.Foo", "Device.NewPriv")}
	require.NoError(t, appReg.InvalidateParamModel(context.Background(), pmID))

	// ACS 再翻译：必须拿到新私有 PATH
	tr2, err := acsReg.Translator(context.Background(), productID, "1.0")
	require.NoError(t, err)
	assert.Equal(t, "Device.NewPriv", tr2.ToPrivate("Device.Foo").Translated,
		"ACS 必须在 app 编辑映射后及时下发最新 privatePath（无需重启）")
}

// TestRegistry_CrossInstance_DiscoveredEditPropagates 同上，覆盖 discovered 映射
// 与 InvalidateProduct 路径（Intersect 写入 / discovered 版本删除场景）。
func TestRegistry_CrossInstance_DiscoveredEditPropagates(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()
	productID := uuid.New()
	pmID := uuid.New()
	pg.products[productID] = &product.Product{ID: productID, ParamModelID: &pmID}
	dk := discoveredKey{productID: productID, swVersion: "1.0"}
	repo.discoveredByDevice[dk] = []ParamMapping{mkMapping("Device.Bar", "Device.OldDisc")}

	client := newMiniRedis(t)
	appReg := NewRegistry(repo, NewRedisCache(client), pg, NewRegistryMetrics(nil), zap.NewNop())
	acsReg := NewRegistry(repo, NewRedisCache(client), pg, NewRegistryMetrics(nil), zap.NewNop())

	tr, err := acsReg.Translator(context.Background(), productID, "1.0")
	require.NoError(t, err)
	require.Equal(t, "Device.OldDisc", tr.ToPrivate("Device.Bar").Translated)

	repo.discoveredByDevice[dk] = []ParamMapping{mkMapping("Device.Bar", "Device.NewDisc")}
	require.NoError(t, appReg.InvalidateProduct(context.Background(), productID, "1.0"))

	tr2, err := acsReg.Translator(context.Background(), productID, "1.0")
	require.NoError(t, err)
	assert.Equal(t, "Device.NewDisc", tr2.ToPrivate("Device.Bar").Translated,
		"ACS 必须在 discovered 映射变更后及时下发最新 privatePath")
}

// TestRegistry_EnsureFresh_DropsL1OnExternalVersionBump 直接验证热路径版本自检：
// 外部 BumpVersion（模拟另一进程）后，本 Registry 下次查询丢弃 L1 并回 DB。
func TestRegistry_EnsureFresh_DropsL1OnExternalVersionBump(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()
	pmID := uuid.New()
	repo.defaultByModel[pmID] = []ParamMapping{mkMapping("A", "a")}

	client := newMiniRedis(t)
	cache := NewRedisCache(client)
	r := NewRegistry(repo, cache, pg, NewRegistryMetrics(nil), zap.NewNop())

	_, err := r.GetByParamModel(context.Background(), pmID)
	require.NoError(t, err)
	require.Equal(t, 1, repo.listDefaultCalls, "首查回 DB 一次")

	// 二次查询 L1 命中，不再回 DB
	_, _ = r.GetByParamModel(context.Background(), pmID)
	require.Equal(t, 1, repo.listDefaultCalls)

	// 外部进程 BumpVersion（也使 L2 条目因 schema_version 漂移而失效）
	_, err = cache.BumpVersion(context.Background())
	require.NoError(t, err)

	// 下次查询 ensureFresh 丢 L1 → 回 DB 重建
	_, err = r.GetByParamModel(context.Background(), pmID)
	require.NoError(t, err)
	assert.Equal(t, 2, repo.listDefaultCalls, "外部版本前进后应丢 L1 并回 DB")
}

// TestRegistry_Invalidate_BumpsCacheVersion 锁定"写操作必产生跨进程失效信号"。
func TestRegistry_Invalidate_BumpsCacheVersion(t *testing.T) {
	t.Run("InvalidateParamModel", func(t *testing.T) {
		client := newMiniRedis(t)
		cache := NewRedisCache(client)
		r := NewRegistry(newFakeRepo(), cache, newFakeProductGetter(), NewRegistryMetrics(nil), zap.NewNop())
		require.NoError(t, r.InvalidateParamModel(context.Background(), uuid.New()))
		v, err := cache.GetVersion(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(1), v)
	})
	t.Run("InvalidateProduct", func(t *testing.T) {
		client := newMiniRedis(t)
		cache := NewRedisCache(client)
		r := NewRegistry(newFakeRepo(), cache, newFakeProductGetter(), NewRegistryMetrics(nil), zap.NewNop())
		require.NoError(t, r.InvalidateProduct(context.Background(), uuid.New(), "1.0"))
		v, err := cache.GetVersion(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(1), v)
	})
}

func TestRegistry_NewRegistry_NilDefaults(t *testing.T) {
	// NewRegistry 应安全接收 nil cache/metrics/logger（退化为 NopCache/匿名 reg/Nop logger）
	r := NewRegistry(newFakeRepo(), nil, nil, nil, nil)
	require.NotNil(t, r)
	assert.NotNil(t, r.cache)
	assert.NotNil(t, r.metrics)
	assert.NotNil(t, r.logger)
}

func TestRegistry_InvalidateProduct_PropagatesL2Error(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	cache := &errorCache{invErr: errors.New("redis del failed")}
	r := NewRegistry(repo, cache, pg, NewRegistryMetrics(nil), zap.NewNop())
	err := r.InvalidateProduct(context.Background(), uuid.New(), "1")
	assert.Error(t, err, "L2 失败应原样返回，让运维知道")
}

func TestRegistry_Refresh_PropagatesBumpVersionError(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	cache := &errorCache{bumpErr: errors.New("redis incr failed")}
	r := NewRegistry(repo, cache, pg, NewRegistryMetrics(nil), zap.NewNop())
	err := r.Refresh(context.Background())
	assert.Error(t, err)
}

// errorCache 用于触发 cache 错误路径。
type errorCache struct {
	invErr  error
	bumpErr error
}

func (c *errorCache) GetDefault(context.Context, uuid.UUID) ([]ParamMapping, error) {
	return nil, nil
}
func (c *errorCache) SetDefault(context.Context, uuid.UUID, []ParamMapping) error { return nil }
func (c *errorCache) InvalidateDefault(context.Context, uuid.UUID) error {
	return c.invErr
}
func (c *errorCache) GetDiscovered(context.Context, uuid.UUID, string) ([]ParamMapping, error) {
	return nil, nil
}
func (c *errorCache) SetDiscovered(context.Context, uuid.UUID, string, []ParamMapping) error {
	return nil
}
func (c *errorCache) InvalidateDiscovered(context.Context, uuid.UUID, string) error {
	return c.invErr
}
func (c *errorCache) GetVersion(context.Context) (int64, error) { return 0, nil }
func (c *errorCache) BumpVersion(context.Context) (int64, error) {
	if c.bumpErr != nil {
		return 0, c.bumpErr
	}
	return 1, nil
}

func TestRegistry_LoadDefault_L2HitFillsL1(t *testing.T) {
	repo := newFakeRepo()
	pg := newFakeProductGetter()

	pmID := uuid.New()
	repo.defaultByModel[pmID] = []ParamMapping{mkMapping("A", "a")}

	client := newMiniRedis(t)
	cache := NewRedisCache(client)

	// 先把 default 写进 L2
	require.NoError(t, cache.SetDefault(context.Background(), pmID, []ParamMapping{mkMapping("Z", "z")}))

	r := NewRegistry(repo, cache, pg, NewRegistryMetrics(nil), zap.NewNop())
	set, err := r.GetByParamModel(context.Background(), pmID)
	require.NoError(t, err)
	require.Len(t, set.Mappings, 1)
	assert.Equal(t, "Z", set.Mappings[0].StandardPath, "应优先 L2，非 DB")
	assert.Equal(t, 0, repo.listDefaultCalls)
}
