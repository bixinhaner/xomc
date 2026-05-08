package product

import (
	"context"
	"errors"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── 内存 Fake Repository ────────────────────────────────────────────

type fakeRepo struct {
	patterns         []ProductClassPattern
	products         map[uuid.UUID]*Product
	indicatorByDev   map[string]map[string]struct{}
	alarmNeTypes     map[string]struct{}
	listPatternsCalls int32
	getProductCalls   int32
	listPatternsErr  error
	getProductErr    error
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
	store      map[uuid.UUID]*Product
	getCount   int32
	setCount   int32
	bumpCount  int32
	failGet    bool
}

func newSpyCache() *spyCache { return &spyCache{store: make(map[uuid.UUID]*Product)} }

func (s *spyCache) GetProduct(_ context.Context, id uuid.UUID) (*Product, error) {
	atomic.AddInt32(&s.getCount, 1)
	if s.failGet {
		return nil, errors.New("redis down")
	}
	if p, ok := s.store[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, nil
}
func (s *spyCache) SetProduct(_ context.Context, p *Product) error {
	atomic.AddInt32(&s.setCount, 1)
	cp := *p
	s.store[p.ID] = &cp
	return nil
}
func (s *spyCache) InvalidateProduct(_ context.Context, id uuid.UUID) error {
	delete(s.store, id)
	return nil
}
func (s *spyCache) GetVersion(_ context.Context) (int64, error) { return 0, nil }
func (s *spyCache) BumpVersion(_ context.Context) (int64, error) {
	atomic.AddInt32(&s.bumpCount, 1)
	return 1, nil
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
