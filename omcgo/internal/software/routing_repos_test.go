package software

import (
	"context"
	"testing"
	"time"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	model "github.com/omcgo/omcgo/internal/core/model"

	"github.com/google/uuid"
)

// pagedSubTaskFallback 实现 SubTaskRepository 接口，仅 ListAll 返回真实分页数据，
// 其余方法返回 ErrNotFound 兜底，专供 #653 翻页回归测试。
//
// 数据语义：构造 total 行 UpgradeSubTaskWithTaskName，每次 ListAll(page, pageSize)
// 按 PgSubTaskRepository 的 cap 行为（pageSize > 100 截到 100）切片返回，模拟
// 真实底表"单页上限 100"的语义；total 字段固定为全行数。
type pagedSubTaskFallback struct {
	SubTaskRepository
	items     []UpgradeSubTaskWithTaskName
	callCount int
}

func (s *pagedSubTaskFallback) ListAll(_ context.Context, filter AllSubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	s.callCount++
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	total := int64(len(s.items))
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(s.items) {
		start = len(s.items)
	}
	if end > len(s.items) {
		end = len(s.items)
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return &model.ListResponse[UpgradeSubTaskWithTaskName]{
		Items:      append([]UpgradeSubTaskWithTaskName(nil), s.items[start:end]...),
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// 其余 SubTaskRepository 方法走嵌入式 nil 接口 → 调用即 panic；测试只触达 ListAll。
// 用 ErrNotFound 兜底几个 routing 链路必经的 fan-out 路径，避免误调时 panic。
func (s *pagedSubTaskFallback) GetByID(_ context.Context, _ uuid.UUID) (*UpgradeSubTask, error) {
	return nil, commonerrors.ErrNotFound
}
func (s *pagedSubTaskFallback) GetByCommandKey(_ context.Context, _ string) (*UpgradeSubTask, error) {
	return nil, commonerrors.ErrNotFound
}
func (s *pagedSubTaskFallback) GetActiveByDeviceID(_ context.Context, _ uuid.UUID) (*UpgradeSubTask, error) {
	return nil, commonerrors.ErrNotFound
}

func makeSubTasks(n int) []UpgradeSubTaskWithTaskName {
	out := make([]UpgradeSubTaskWithTaskName, n)
	base := time.Now()
	for i := 0; i < n; i++ {
		out[i] = UpgradeSubTaskWithTaskName{
			UpgradeSubTask: UpgradeSubTask{
				ID:        uuid.New(),
				CreatedAt: model.Time(base.Add(time.Duration(i) * time.Second)),
			},
		}
	}
	return out
}

// TestRoutingSubTaskRepository_ListAll_PagesBeyond100 锁定 #653 回归：单任务
// sub_task > 100 时 RoutingSubTaskRepository.ListAll 必须循环翻页拉全量，而不是
// 只从每张表拿 page=1 的 100 行（旧实现把第 101+ 条静默丢掉，导致 UFTE 设备列表
// 看不到所有子任务）。
//
// 构造 200 行 fallback 数据 + 空 router（无其它子表），外层请求 PageSize=200
// 拿满 200 行。
func TestRoutingSubTaskRepository_ListAll_PagesBeyond100(t *testing.T) {
	fallback := &pagedSubTaskFallback{items: makeSubTasks(200)}
	router := &TransferRepoRouter{}
	r := NewRoutingSubTaskRepository(router, fallback)

	res, err := r.ListAll(context.Background(), AllSubTaskFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 200},
	})
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	// 应用层 pageSize > 100 会被 cap 到 100；total 仍是 200，第 1 页应该返回 100 行。
	if res.Total != 200 {
		t.Errorf("Total = %d, want 200", res.Total)
	}
	if len(res.Items) != 100 {
		t.Errorf("len(Items) = %d, want 100 (first page after cap)", len(res.Items))
	}
	if res.TotalPages != 2 {
		t.Errorf("TotalPages = %d, want 2", res.TotalPages)
	}
	// 关键回归：fallback 内部应被翻到 2 页（page=1 + page=2），证明跨页拉全量。
	if fallback.callCount < 2 {
		t.Errorf("fallback ListAll calls = %d, want >= 2 (paged through cap=100)", fallback.callCount)
	}

	// 第 2 页应该返回剩下的 100 行（不再是空），确保旧实现的 "merged[100:]=空" bug 不再发生。
	fallback.callCount = 0
	res2, err := r.ListAll(context.Background(), AllSubTaskFilter{
		ListRequest: model.ListRequest{Page: 2, PageSize: 100},
	})
	if err != nil {
		t.Fatalf("ListAll page=2: %v", err)
	}
	if len(res2.Items) != 100 {
		t.Errorf("page=2 len(Items) = %d, want 100 (旧 bug 这里返回 0)", len(res2.Items))
	}
	if res2.Total != 200 {
		t.Errorf("page=2 Total = %d, want 200", res2.Total)
	}
}

func TestRoutingSubTaskRepository_DirectDispatchCommandKeyUsesFallback(t *testing.T) {
	fallback := &pagedSubTaskFallback{}
	router := &TransferRepoRouter{
		ConfigBackup:      TransferRepoSet{SubTask: fallback},
		RuntimeLogCollect: TransferRepoSet{SubTask: fallback},
		FaultLogCollect:   TransferRepoSet{SubTask: fallback},
		ImsParamCollect:   TransferRepoSet{SubTask: fallback},
	}
	repo := NewRoutingSubTaskRepository(router, fallback)

	candidates := repo.candidatesByCommandKey("IMS_PARAM_DISTRIBUTE_3d594e25_SN001")
	if len(candidates) != 1 || candidates[0] != fallback {
		t.Fatalf("direct-dispatch candidates = %d, want only fallback repository", len(candidates))
	}
}

// TestRoutingSubTaskRepository_ListAll_EmptyShortCircuit fallback 全空时不应该
// 死循环；翻页提前 break。
func TestRoutingSubTaskRepository_ListAll_EmptyShortCircuit(t *testing.T) {
	fallback := &pagedSubTaskFallback{items: nil}
	r := NewRoutingSubTaskRepository(&TransferRepoRouter{}, fallback)
	res, err := r.ListAll(context.Background(), AllSubTaskFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 100},
	})
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if res.Total != 0 || len(res.Items) != 0 {
		t.Errorf("empty store should return Total=0/Items=0; got Total=%d Items=%d", res.Total, len(res.Items))
	}
	if fallback.callCount != 1 {
		t.Errorf("empty store should call fallback once and break; got %d calls", fallback.callCount)
	}
}

// pagedTaskFallback 是 TaskRepository 的 fake，仅 List 返回真实分页，其余方法
// 走嵌入式 nil 接口 → 触达即 panic（测试只该走 List）。模拟 PgTaskRepository 的
// pageSize cap 100 行为，让 RoutingTaskRepository.List 的回归测试能真实复现
// "底层硬 cap 100 + 旧实现不翻页 → 第 101 条丢失" 的 bug。
type pagedTaskFallback struct {
	TaskRepository
	items     []UpgradeTask
	callCount int
}

func (s *pagedTaskFallback) List(_ context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
	s.callCount++
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	total := int64(len(s.items))
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(s.items) {
		start = len(s.items)
	}
	if end > len(s.items) {
		end = len(s.items)
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return &model.ListResponse[UpgradeTask]{
		Items:      append([]UpgradeTask(nil), s.items[start:end]...),
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func makeUpgradeTasks(n int) []UpgradeTask {
	out := make([]UpgradeTask, n)
	base := time.Now()
	for i := 0; i < n; i++ {
		out[i] = UpgradeTask{
			ID:        uuid.New(),
			CreatedAt: model.Time(base.Add(time.Duration(i) * time.Second)),
		}
	}
	return out
}

// TestRoutingTaskRepository_List_PagesBeyond100 锁定 #653 同类回归：单业务 task
// 表行数 > 100 时 RoutingTaskRepository.List 必须循环翻页拉全量，而不是只取
// page=1 的 100 行。旧实现把外层 filter 原样下发 → PgTaskRepository 内部 cap
// pageSize=100 → 只返回前 100 → union total=150 → 外层 totalPages=1 → break →
// 第 101+ 条任务在 UFTE 任务管理 Tab「任务列表」永远看不到。
func TestRoutingTaskRepository_List_PagesBeyond100(t *testing.T) {
	fallback := &pagedTaskFallback{items: makeUpgradeTasks(150)}
	router := &TransferRepoRouter{}
	r := NewRoutingTaskRepository(router, fallback)

	res, err := r.List(context.Background(), UpgradeTaskFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 200},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// 应用层 pageSize > 100 会被 cap 到 100；total 仍是 150。
	if res.Total != 150 {
		t.Errorf("Total = %d, want 150", res.Total)
	}
	if len(res.Items) != 100 {
		t.Errorf("len(Items) = %d, want 100 (first page after cap)", len(res.Items))
	}
	if res.TotalPages != 2 {
		t.Errorf("TotalPages = %d, want 2", res.TotalPages)
	}
	// 关键回归：fallback 内部应被翻到 2 页（page=1 + page=2），证明跨页拉全量。
	if fallback.callCount < 2 {
		t.Errorf("fallback List calls = %d, want >= 2 (paged through cap=100)", fallback.callCount)
	}

	// 第 2 页应该返回剩下的 50 行，确保旧实现的 "merged[100:200]=空" bug 不再发生。
	fallback.callCount = 0
	res2, err := r.List(context.Background(), UpgradeTaskFilter{
		ListRequest: model.ListRequest{Page: 2, PageSize: 100},
	})
	if err != nil {
		t.Fatalf("List page=2: %v", err)
	}
	if len(res2.Items) != 50 {
		t.Errorf("page=2 len(Items) = %d, want 50 (旧 bug 这里返回 0)", len(res2.Items))
	}
	if res2.Total != 150 {
		t.Errorf("page=2 Total = %d, want 150", res2.Total)
	}
}

// TestRoutingTaskRepository_List_EmptyShortCircuit fallback 全空时翻页提前 break，
// 不死循环；total=0。
func TestRoutingTaskRepository_List_EmptyShortCircuit(t *testing.T) {
	fallback := &pagedTaskFallback{items: nil}
	r := NewRoutingTaskRepository(&TransferRepoRouter{}, fallback)
	res, err := r.List(context.Background(), UpgradeTaskFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 100},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if res.Total != 0 || len(res.Items) != 0 {
		t.Errorf("empty store should return Total=0/Items=0; got Total=%d Items=%d", res.Total, len(res.Items))
	}
	if fallback.callCount != 1 {
		t.Errorf("empty store should call fallback once and break; got %d calls", fallback.callCount)
	}
}
