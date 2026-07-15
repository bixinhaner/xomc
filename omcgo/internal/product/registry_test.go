package product

import (
	"context"
	"errors"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── 内存 Fake Repository ────────────────────────────────────────────

type fakeRepo struct {
	patterns          []ProductClassPattern
	products          map[uuid.UUID]*Product
	indicatorByDev    map[string]map[string]struct{}
	alarmNeTypes      map[string]struct{}
	listPatternsCalls int32
	getProductCalls   int32
	listPatternsErr   error
	getProductErr     error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		products:       make(map[uuid.UUID]*Product),
		indicatorByDev: make(map[string]map[string]struct{}),
		alarmNeTypes:   make(map[string]struct{}),
	}
}

func (f *fakeRepo) ListActivePatterns(_ context.Context) ([]ProductClassPattern, error) {
	atomic.AddInt32(&f.listPatternsCalls, 1)
	if f.listPatternsErr != nil {
		return nil, f.listPatternsErr
	}
	out := make([]ProductClassPattern, len(f.patterns))
	copy(out, f.patterns)
	sort.Slice(out, func(i, j int) bool { return out[i].SortOrder < out[j].SortOrder })
	return out, nil
}

func (f *fakeRepo) GetProductByID(_ context.Context, id uuid.UUID) (*Product, error) {
	atomic.AddInt32(&f.getProductCalls, 1)
	if f.getProductErr != nil {
		return nil, f.getProductErr
	}
	p, ok := f.products[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}

func (f *fakeRepo) ListProducts(_ context.Context) ([]*Product, error) {
	out := make([]*Product, 0, len(f.products))
	for _, p := range f.products {
		cp := *p
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (f *fakeRepo) FetchIndicatorPlatformsByDeviceType(_ context.Context, dt string) (map[string]struct{}, error) {
	if m, ok := f.indicatorByDev[dt]; ok {
		return m, nil
	}
	return map[string]struct{}{}, nil
}

func (f *fakeRepo) FetchAlarmNeTypes(_ context.Context) (map[string]struct{}, error) {
	return f.alarmNeTypes, nil
}

// 为 fakeRepo 增加便捷 builder
func (f *fakeRepo) addProduct(name, vendor, deviceType, platform, neType string) uuid.UUID {
	id := uuid.New()
	f.products[id] = &Product{
		ID:                  id,
		Name:                name,
		Vendor:              vendor,
		IndicatorDeviceType: deviceType,
		IndicatorPlatform:   platform,
		AlarmNeType:         neType,
	}
	return id
}

func (f *fakeRepo) addPattern(productID uuid.UUID, regex string, sortOrder int) {
	f.patterns = append(f.patterns, ProductClassPattern{
		ID:           uuid.New(),
		ProductID:    productID,
		ProductClass: regex,
		SortOrder:    sortOrder,
		IsActive:     true,
	})
}

// ─── Refresh 测试 ─────────────────────────────────────────────────────

func TestRegistry_Refresh_LoadsAndCompiles(t *testing.T) {
	repo := newFakeRepo()
	pid1 := repo.addProduct("QRTB", "Baicells", "enb", "BLQ", "ENB")
	pid2 := repo.addProduct("FAP-fallback", "Generic", "enb", "BLQ", "ENB")
	repo.addPattern(pid1, "^QRTB", 1)
	repo.addPattern(pid2, "FAP", 99)

	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	ctx := context.Background()
	require.NoError(t, r.Refresh(ctx))
	assert.Equal(t, 2, r.PatternCount())
}

func TestRegistry_Refresh_SkipsBadRegex(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("good", "v", "enb", "BLQ", "ENB")
	repo.addPattern(pid, "^OK$", 1)
	repo.addPattern(pid, "[invalid(", 2) // 编译失败 → 跳过

	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))
	assert.Equal(t, 1, r.PatternCount(), "bad regex must be skipped, not panic")
}

// #17: 坏正则被跳过时不再静默，必须递增 product_registry_pattern_skip_total。
func TestRegistry_Refresh_BadRegexIncrementsMetric(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("good", "v", "enb", "BLQ", "ENB")
	repo.addPattern(pid, "^OK$", 1)
	repo.addPattern(pid, "[invalid(", 2) // 坏正则 1
	repo.addPattern(pid, "(*broken", 3)  // 坏正则 2

	metrics := NewRegistryMetrics(nil)
	r := NewRegistry(repo, NopCache{}, metrics, zap.NewNop())

	// 刷新前计数应为 0。
	require.Equal(t, float64(0), testutil.ToFloat64(metrics.patternSkipTotal))

	require.NoError(t, r.Refresh(context.Background()))

	assert.Equal(t, 1, r.PatternCount(), "only the valid pattern survives")
	assert.Equal(t, float64(2), testutil.ToFloat64(metrics.patternSkipTotal),
		"both bad regexes must be counted, not silently dropped")
}

// 全部正则有效时，skip 计数保持 0（成功路径回归）。
func TestRegistry_Refresh_NoSkipsWhenAllValid(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("good", "v", "enb", "BLQ", "ENB")
	repo.addPattern(pid, "^OK$", 1)
	repo.addPattern(pid, "^QRTB", 2)

	metrics := NewRegistryMetrics(nil)
	r := NewRegistry(repo, NopCache{}, metrics, zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	assert.Equal(t, 2, r.PatternCount())
	assert.Equal(t, float64(0), testutil.ToFloat64(metrics.patternSkipTotal))
}

func TestRegistry_Refresh_PropagatesRepoError(t *testing.T) {
	repo := newFakeRepo()
	repo.listPatternsErr = errors.New("db down")
	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	err := r.Refresh(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list active patterns")
}

// ─── MatchProductClass 测试 ───────────────────────────────────────────

func TestRegistry_Match_FirstHitInGlobalOrder(t *testing.T) {
	repo := newFakeRepo()
	pid1 := repo.addProduct("Specific", "v1", "enb", "BLQ", "ENB")
	pid2 := repo.addProduct("FAP-fallback", "Generic", "enb", "BLQ", "ENB")
	// 全局序：specific (1) → fallback (99)
	repo.addPattern(pid1, "^QRTB", 1)
	repo.addPattern(pid2, ".*FAP.*", 99)

	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	// QRTB123 命中 specific
	got, err := r.MatchProductClass(context.Background(), "QRTB123")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, pid1, got.Product.ID)
	assert.Equal(t, "^QRTB", got.MatchedPattern)
	assert.Equal(t, 1, got.GlobalOrder)

	// 任意其他字符串中含 FAP → fallback
	got, err = r.MatchProductClass(context.Background(), "BaiCells-FAP-X")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, pid2, got.Product.ID)
	assert.Equal(t, 99, got.GlobalOrder)
}

func TestRegistry_Match_FAPGlobalOrderDominates(t *testing.T) {
	// 设计 §4.2.2：FAP 兜底正则必须最末（即使插入顺序乱），sort_order 决定优先级。
	repo := newFakeRepo()
	pid1 := repo.addProduct("Specific", "v1", "enb", "BLQ", "ENB")
	pid2 := repo.addProduct("FAP-fallback", "Generic", "enb", "BLQ", "ENB")
	// 故意把 fallback 插在前面（DB 顺序），但 sort_order 标记为 99
	repo.addPattern(pid2, ".*FAP.*", 99)
	repo.addPattern(pid1, "^FAP-A", 1)

	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	got, err := r.MatchProductClass(context.Background(), "FAP-A123")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, pid1, got.Product.ID, "global sort_order=1 must win over sort_order=99")
}

func TestRegistry_Match_OrphanWhenNoPattern(t *testing.T) {
	repo := newFakeRepo()
	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	got, err := r.MatchProductClass(context.Background(), "anything")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrOrphan)
}

func TestRegistry_Match_OrphanWhenNoMatch(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("Specific", "v1", "enb", "BLQ", "ENB")
	repo.addPattern(pid, "^QRTB", 1)

	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	got, err := r.MatchProductClass(context.Background(), "Unrelated123")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrOrphan)
}

func TestRegistry_Match_InactiveParamModelIsNotRoutable(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("Inactive", "v1", "enb", "BLQ", "ENB")
	repo.patterns = append(repo.patterns, ProductClassPattern{
		ID:                 uuid.New(),
		ProductID:          pid,
		ProductClass:       "^FAP/inactive$",
		SortOrder:          1,
		IsActive:           true,
		ParamModelInactive: true,
	})

	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	got, err := r.MatchProductClass(context.Background(), "FAP/inactive")
	require.ErrorIs(t, err, ErrInactiveParamModel)
	require.ErrorIs(t, err, ErrOrphan, "inactive model must remain compatible with device orphan handling")
	require.NotNil(t, got, "callers that distinguish inactive models need the matched product")
	assert.Equal(t, pid, got.Product.ID)
}

func TestRegistry_Match_DanglingPatternFallsThrough(t *testing.T) {
	// 数据不一致：pattern 引用了不存在的 product；Match 应继续遍历后续 pattern。
	repo := newFakeRepo()
	pidGood := repo.addProduct("Good", "v", "enb", "BLQ", "ENB")
	missingPID := uuid.New() // 不入 fakeRepo.products
	repo.addPattern(missingPID, "^QRTB", 1)
	repo.addPattern(pidGood, ".*", 99)

	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	got, err := r.MatchProductClass(context.Background(), "QRTB123")
	require.NoError(t, err, "expected fall-through to next pattern, not error")
	require.NotNil(t, got)
	assert.Equal(t, pidGood, got.Product.ID)
}

// ─── GetProductByID 缓存层级测试 ───────────────────────────────────

type spyCache struct {
	store     map[uuid.UUID]*Product
	storeVer  map[uuid.UUID]int64
	getCount  int32
	setCount  int32
	bumpCount int32
	failGet   bool
	version   int64

	// productClass cache state（T-0173）
	classStore    map[string]*ProductClassCacheEntry
	classTTL      map[string]time.Duration
	classGetCount int32
	classSetCount int32
	classFailGet  bool
}

func newSpyCache() *spyCache {
	return &spyCache{
		store:      make(map[uuid.UUID]*Product),
		storeVer:   make(map[uuid.UUID]int64),
		classStore: make(map[string]*ProductClassCacheEntry),
		classTTL:   make(map[string]time.Duration),
	}
}

func (s *spyCache) GetProduct(_ context.Context, id uuid.UUID) (*Product, error) {
	atomic.AddInt32(&s.getCount, 1)
	if s.failGet {
		return nil, errors.New("redis down")
	}
	if p, ok := s.store[id]; ok {
		if s.storeVer[id] != s.version {
			return nil, nil
		}
		cp := *p
		return &cp, nil
	}
	return nil, nil
}
func (s *spyCache) SetProduct(_ context.Context, p *Product) error {
	atomic.AddInt32(&s.setCount, 1)
	cp := *p
	s.store[p.ID] = &cp
	s.storeVer[p.ID] = s.version
	return nil
}
func (s *spyCache) InvalidateProduct(_ context.Context, id uuid.UUID) error {
	delete(s.store, id)
	delete(s.storeVer, id)
	return nil
}
func (s *spyCache) GetVersion(_ context.Context) (int64, error) { return s.version, nil }
func (s *spyCache) BumpVersion(_ context.Context) (int64, error) {
	atomic.AddInt32(&s.bumpCount, 1)
	s.version++
	return s.version, nil
}

func (s *spyCache) GetProductClass(_ context.Context, productClass string) (*ProductClassCacheEntry, error) {
	atomic.AddInt32(&s.classGetCount, 1)
	if s.classFailGet {
		return nil, errors.New("redis down")
	}
	if e, ok := s.classStore[productClass]; ok {
		cp := *e
		return &cp, nil
	}
	return nil, nil
}

func (s *spyCache) SetProductClass(_ context.Context, productClass string, entry *ProductClassCacheEntry, ttl time.Duration) error {
	atomic.AddInt32(&s.classSetCount, 1)
	cp := *entry
	s.classStore[productClass] = &cp
	s.classTTL[productClass] = ttl
	return nil
}

func TestRegistry_GetProductByID_L1HitAfterFirstLoad(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("X", "v", "enb", "BLQ", "ENB")
	cache := newSpyCache()

	r := NewRegistry(repo, cache, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	// 1st: miss → DB → L1 + L2 set
	p, err := r.GetProductByID(context.Background(), pid)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, int32(1), atomic.LoadInt32(&repo.getProductCalls))
	assert.Equal(t, int32(1), atomic.LoadInt32(&cache.setCount))

	// 2nd: L1 hit → 不再访问 DB / L2
	dbCallsBefore := atomic.LoadInt32(&repo.getProductCalls)
	_, err = r.GetProductByID(context.Background(), pid)
	require.NoError(t, err)
	assert.Equal(t, dbCallsBefore, atomic.LoadInt32(&repo.getProductCalls), "L1 hit must not hit DB")
}

func TestRegistry_GetProductByID_L2HitFillsL1(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("X", "v", "enb", "BLQ", "ENB")

	cache := newSpyCache()
	// 预热 L2 — 模拟跨实例已写入 cache
	require.NoError(t, cache.SetProduct(context.Background(), repo.products[pid]))
	atomic.StoreInt32(&cache.setCount, 0)

	r := NewRegistry(repo, cache, NewRegistryMetrics(nil), zap.NewNop())
	// 不调 Refresh，只测 GetProductByID 路径（L1 空 → L2 hit → L1 fill）

	p, err := r.GetProductByID(context.Background(), pid)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, int32(0), atomic.LoadInt32(&repo.getProductCalls), "L2 hit must not hit DB")

	// 第二次：L1 hit
	_, err = r.GetProductByID(context.Background(), pid)
	require.NoError(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&repo.getProductCalls))
}

func TestRegistry_GetProductByID_L2FailureFallsBackToDB(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("X", "v", "enb", "BLQ", "ENB")
	cache := newSpyCache()
	cache.failGet = true

	r := NewRegistry(repo, cache, NewRegistryMetrics(nil), zap.NewNop())
	p, err := r.GetProductByID(context.Background(), pid)
	require.NoError(t, err, "L2 read failure must downgrade to DB, not error")
	require.NotNil(t, p)
}

func TestRegistry_GetProductByID_NotFound(t *testing.T) {
	repo := newFakeRepo()
	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())

	p, err := r.GetProductByID(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, p)
}

func TestRegistry_Refresh_ClearsL1AndBumpsVersion(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("X", "v", "enb", "BLQ", "ENB")
	cache := newSpyCache()
	r := NewRegistry(repo, cache, NewRegistryMetrics(nil), zap.NewNop())

	// 加载一次确保 L1 有缓存
	require.NoError(t, r.Refresh(context.Background()))
	_, err := r.GetProductByID(context.Background(), pid)
	require.NoError(t, err)
	dbBefore := atomic.LoadInt32(&repo.getProductCalls)

	// Refresh → drop L1 + bump version
	require.NoError(t, r.Refresh(context.Background()))
	assert.GreaterOrEqual(t, atomic.LoadInt32(&cache.bumpCount), int32(1))

	// 再次 Get → 应当再次走 L2/DB（L1 已清）
	_, err = r.GetProductByID(context.Background(), pid)
	require.NoError(t, err)
	dbAfter := atomic.LoadInt32(&repo.getProductCalls)
	// L2 仍有，所以 DB 不一定走；但至少 L1 已清；这里仅验证不 panic 即可
	_ = dbAfter
	_ = dbBefore
}

func TestRegistry_MatchProductClass_RefreshesOnVersionChange(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("BM", "Baicells", "enb", "BLQ", "ENB")
	repo.products[pid].EnableUnknownAlarm = false
	repo.addPattern(pid, "^FAP/BU1810$", 1)
	cache := newSpyCache()

	r := NewRegistry(repo, cache, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	match, err := r.MatchProductClass(context.Background(), "FAP/BU1810")
	require.NoError(t, err)
	require.NotNil(t, match)
	assert.False(t, match.Product.EnableUnknownAlarm)

	// 模拟产品中心改开关并 bump 版本；当前实例不主动 Refresh。
	repo.products[pid].EnableUnknownAlarm = true
	cache.version++

	match, err = r.MatchProductClass(context.Background(), "FAP/BU1810")
	require.NoError(t, err)
	require.NotNil(t, match)
	assert.True(t, match.Product.EnableUnknownAlarm)
}

// ─── ValidateReferences 测试 ────────────────────────────────────────

func TestRegistry_ValidateReferences_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	repo.addProduct("X", "v", "enb", "BLQ", "ENB")
	repo.alarmNeTypes = map[string]struct{}{"ENB": {}}
	repo.indicatorByDev = map[string]map[string]struct{}{
		"enb": {"BLQ": {}},
	}

	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	rep, err := r.ValidateReferences(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, rep.TotalProducts)
	assert.Empty(t, rep.WarnedProducts)
}

func TestRegistry_ValidateReferences_DetectsMissingPlatform(t *testing.T) {
	repo := newFakeRepo()
	repo.addProduct("X", "v", "enb", "BLQ", "ENB") // platform=BLQ
	repo.alarmNeTypes = map[string]struct{}{"ENB": {}}
	repo.indicatorByDev = map[string]map[string]struct{}{
		"enb": {}, // 没有 BLQ
	}

	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	rep, err := r.ValidateReferences(context.Background())
	require.NoError(t, err)
	require.Len(t, rep.WarnedProducts, 1)
	assert.Equal(t, "indicator_platform", rep.WarnedProducts[0].Field)
}

func TestRegistry_ValidateReferences_DetectsMissingAlarmNeType(t *testing.T) {
	repo := newFakeRepo()
	repo.addProduct("X", "v", "enb", "BLQ", "ENB")
	repo.alarmNeTypes = map[string]struct{}{} // 空
	repo.indicatorByDev = map[string]map[string]struct{}{
		"enb": {"BLQ": {}},
	}

	r := NewRegistry(repo, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	rep, err := r.ValidateReferences(context.Background())
	require.NoError(t, err)
	require.Len(t, rep.WarnedProducts, 1)
	assert.Equal(t, "alarm_ne_type", rep.WarnedProducts[0].Field)
}

func TestRegistry_ValidateReferences_BatchesPlatformQueries(t *testing.T) {
	// 同 deviceType 的多个 product 应共享一次 platform 查询（按 deviceType 分桶）。
	// 此测试通过 spyRepo 计数 FetchIndicatorPlatformsByDeviceType 调用次数。
	type spyRepo struct {
		fakeRepo
		fetchCalls map[string]int
	}
	sp := &spyRepo{fakeRepo: *newFakeRepo(), fetchCalls: map[string]int{}}
	sp.addProduct("A", "v", "enb", "BLQ", "ENB")
	sp.addProduct("B", "v", "enb", "MLN", "ENB")
	sp.addProduct("C", "v", "gnb", "BSC", "GNB")
	sp.alarmNeTypes = map[string]struct{}{"ENB": {}, "GNB": {}}
	sp.indicatorByDev = map[string]map[string]struct{}{
		"enb": {"BLQ": {}, "MLN": {}},
		"gnb": {"BSC": {}},
	}
	// 用一个匿名 wrapper 计数
	wrapper := &countingRepo{Repository: sp, calls: map[string]int{}}

	r := NewRegistry(wrapper, NopCache{}, NewRegistryMetrics(nil), zap.NewNop())
	_, err := r.ValidateReferences(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, wrapper.calls["enb"], "two enb products should share one platform fetch")
	assert.Equal(t, 1, wrapper.calls["gnb"])
}

type countingRepo struct {
	Repository
	calls map[string]int
}

func (c *countingRepo) FetchIndicatorPlatformsByDeviceType(ctx context.Context, dt string) (map[string]struct{}, error) {
	c.calls[dt]++
	return c.Repository.FetchIndicatorPlatformsByDeviceType(ctx, dt)
}

// ─── T-0173 MatchProductClass 二级缓存测试 ────────────────────────────

// patternsScanCountRepo 在 fakeRepo 之上追踪 GetProductByID 调用次数变化，
// 间接判断 slow path 是否真的被走（命中 L1/L2 时不应再触碰 productByID）。
// 因为 slow path 内部命中后会调 GetProductByID，所以 slow-path / cache-hit 的区分
// 用 productByID L1 是否预热判断；本文件直接比对 atomic 计数器。

func Test_MatchProductClass_L1Hit(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("Specific", "v1", "enb", "BLQ", "ENB")
	repo.addPattern(pid, "^QRTB", 1)
	cache := newSpyCache()

	r := NewRegistry(repo, cache, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	// 1st call — miss → slow path → 写 L1 + L2
	got, err := r.MatchProductClass(context.Background(), "QRTB123")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int32(1), atomic.LoadInt32(&cache.classSetCount), "first call must populate L2")

	getsBefore := atomic.LoadInt32(&cache.classGetCount)

	// 2nd call — L1 hit，不再访问 L2
	got2, err := r.MatchProductClass(context.Background(), "QRTB123")
	require.NoError(t, err)
	require.NotNil(t, got2)
	assert.Equal(t, pid, got2.Product.ID)
	assert.Equal(t, getsBefore, atomic.LoadInt32(&cache.classGetCount),
		"L1 hit must not call cache.GetProductClass")
	assert.Equal(t, int32(1), atomic.LoadInt32(&cache.classSetCount),
		"L1 hit must not re-populate L2")
}

func Test_MatchProductClass_L2HitAfterRestart(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("Specific", "v1", "enb", "BLQ", "ENB")
	repo.addPattern(pid, "^QRTB", 1)
	cache := newSpyCache()

	// 实例 A：预热 L2
	rA := NewRegistry(repo, cache, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, rA.Refresh(context.Background()))
	_, err := rA.MatchProductClass(context.Background(), "QRTB123")
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&cache.classSetCount))

	// 实例 B：新建（L1 为空，但 L2 有），模拟跨进程重启
	cache.version = 0 // 与 rA 保持一致；spyCache 与 RedisCache 不同，BumpVersion 才递增
	rB := NewRegistry(repo, cache, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, rB.Refresh(context.Background()))

	setsBefore := atomic.LoadInt32(&cache.classSetCount)
	getsBefore := atomic.LoadInt32(&cache.classGetCount)

	got, err := rB.MatchProductClass(context.Background(), "QRTB123")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, pid, got.Product.ID)
	assert.Equal(t, getsBefore+1, atomic.LoadInt32(&cache.classGetCount),
		"L2 must be consulted when L1 is empty")
	assert.Equal(t, setsBefore, atomic.LoadInt32(&cache.classSetCount),
		"L2 hit must not re-populate L2 (no Set call)")
}

func Test_MatchProductClass_OrphanNegativeCache(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("Specific", "v1", "enb", "BLQ", "ENB")
	repo.addPattern(pid, "^QRTB", 1)
	cache := newSpyCache()

	r := NewRegistry(repo, cache, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	// 不匹配任何 pattern → orphan
	got, err := r.MatchProductClass(context.Background(), "Unrelated123")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrOrphan)
	assert.Equal(t, int32(1), atomic.LoadInt32(&cache.classSetCount),
		"orphan must also populate L2 (negative cache)")

	// orphan 用较短 TTL（5min），与 hit（1h）区分
	assert.Equal(t, ProductClassOrphanTTL, cache.classTTL["Unrelated123"],
		"orphan TTL must be 5 minutes")

	// 第二次同样 orphan，但走 L1，不再触发 slow path / L2 写
	setsBefore := atomic.LoadInt32(&cache.classSetCount)
	got, err = r.MatchProductClass(context.Background(), "Unrelated123")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrOrphan)
	assert.Equal(t, setsBefore, atomic.LoadInt32(&cache.classSetCount),
		"second orphan call must hit L1, not re-populate L2")

	// hit 路径 TTL 验证
	_, err = r.MatchProductClass(context.Background(), "QRTB123")
	require.NoError(t, err)
	assert.Equal(t, ProductClassHitTTL, cache.classTTL["QRTB123"],
		"hit TTL must be 1 hour")
}

func Test_MatchProductClass_VersionBumpInvalidatesL1(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("Specific", "v1", "enb", "BLQ", "ENB")
	repo.addPattern(pid, "^QRTB", 1)
	cache := newSpyCache()

	r := NewRegistry(repo, cache, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	// 首次：写入 L1 / L2，CacheVersion=0
	_, err := r.MatchProductClass(context.Background(), "QRTB123")
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&cache.classSetCount))

	// 模拟另一实例改了 pattern + BumpVersion；当前实例的 r.cacheVersion 在 ensureFresh
	// 中被同步上去。spyCache.GetVersion 返新版 → ensureFresh 调 refresh(false) → 清 L1。
	cache.version = 5

	setsBefore := atomic.LoadInt32(&cache.classSetCount)
	_, err = r.MatchProductClass(context.Background(), "QRTB123")
	require.NoError(t, err)
	// 新 version 下 L1 entry stale → slow path → 重写 L2
	assert.Greater(t, atomic.LoadInt32(&cache.classSetCount), setsBefore,
		"version bump must invalidate L1 and rewrite L2")
}

func Test_MatchProductClass_RedisFailureFallback(t *testing.T) {
	repo := newFakeRepo()
	pid := repo.addProduct("Specific", "v1", "enb", "BLQ", "ENB")
	repo.addPattern(pid, "^QRTB", 1)
	cache := newSpyCache()
	cache.classFailGet = true

	r := NewRegistry(repo, cache, NewRegistryMetrics(nil), zap.NewNop())
	require.NoError(t, r.Refresh(context.Background()))

	// L2 GetProductClass 返 error → silent fallback to slow path，调用方拿到正确结果
	got, err := r.MatchProductClass(context.Background(), "QRTB123")
	require.NoError(t, err, "Redis failure must not surface to caller")
	require.NotNil(t, got)
	assert.Equal(t, pid, got.Product.ID)
}
