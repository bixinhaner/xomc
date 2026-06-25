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
