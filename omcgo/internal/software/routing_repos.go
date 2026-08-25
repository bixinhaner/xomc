package software

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// 本文件提供两个把 software 模块"无感知"接入 transfer/repo 多表分发的 wrapper：
//
//   RoutingTaskRepository    — 实现 TaskRepository 接口；按 sub_task_id / task_id /
//                              commandKey hint 反查目标业务表后转发，不识别 →
//                              fallback 旧 upgrade_tasks。
//   RoutingSubTaskRepository — 实现 SubTaskRepository 接口；同上反查 4 张新 sub_task 表。
//
// 设计原因：UpgradeExecutor / RollbackExecutor 等内部组件持有 e.taskRepo / e.subTaskRepo
// 引用，业务调用路径深、改造点多。通过 wrapper 把"按业务派发"封装在仓储层，executor /
// service 完全不感知，零改动获得多表支持。
//
// 性能权衡：GetByID / GetByCommandKey / GetActiveByDeviceID 等"无 hint"读路径在最坏
// 情况会 fan-out 查 5 张新表 + 1 张旧表 = 6 次 SELECT。但每次操作都是 PK / 唯一索引
// 命中，单次 < 1ms；活跃任务峰值 ~500，完全可接受。
//
// 关键：commandKey 前缀已经能识别业务：
//   "Collect NV|" / "Collect XML|" → ConfigBackup
//   "Collect LOG,"                  → RuntimeLog | FaultLog（两个候选，依次 try）
//   其他（升级 / 回退 / Reboot 等）   → 旧表
// 通过 commandKey hint 把 5 次 fan-out 收敛到 1-2 次。

// =============================================================================
// RoutingTaskRepository — main task wrapper
// =============================================================================

var _ TaskRepository = (*RoutingTaskRepository)(nil)

// RoutingTaskRepository 把 main task 写 / 读路径按业务 dispatch 到对应 BasicTaskRepo。
// 升级 / 回退路径（Canary 等 strategy 方法）仍走 fallback（旧表 *PgTaskRepository），
// 因为 BasicTaskRepo 不含 canary 三方法。
type RoutingTaskRepository struct {
	router   *TransferRepoRouter
	fallback TaskRepository // 必须是真正的 *PgTaskRepository（含 canary）
	// Create 时由 SoftwareService 通过 context value 注入业务 hint，
	// wrapper 据此选目标表。无 hint → fallback。
}

// Context key 类型，避免外部碰撞。
type ctxRouteKey struct{}

// WithRouteHint 把"按 fileType 路由"的 hint 写入 context，让 RoutingTaskRepository
// 在 Create 时选目标表。SoftwareService.BatchCollect / BatchUpgrade 决定调用前注入。
// fileType 同 UFTE typeDef.FileType 字符串，router 内部取前导数字识别。
func WithRouteHint(ctx context.Context, fileType string) context.Context {
	return context.WithValue(ctx, ctxRouteKey{}, fileType)
}

// routeFromContext 提取 context hint，若无返回空串。
func routeFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxRouteKey{}).(string)
	return v
}

// NewRoutingTaskRepository 创建 wrapper。fallback 必须是含 canary 的真实实现，
// router 必须装载好 4 张新表 repo。
func NewRoutingTaskRepository(router *TransferRepoRouter, fallback TaskRepository) *RoutingTaskRepository {
	return &RoutingTaskRepository{router: router, fallback: fallback}
}

func (r *RoutingTaskRepository) pickByHint(ctx context.Context) BasicTaskRepo {
	if hint := routeFromContext(ctx); hint != "" {
		set := r.router.ForUploadFileType(hint)
		if set.Task != nil {
			return set.Task
		}
	}
	return r.fallback
}

// pickByID fan-out 查 5 张 main 表，命中即返回。
func (r *RoutingTaskRepository) pickByID(ctx context.Context, id uuid.UUID) (BasicTaskRepo, *UpgradeTask, error) {
	// 优先 fallback（旧表，仍承载升级 / 回退主流量）
	if task, err := r.fallback.GetByID(ctx, id); err == nil {
		return r.fallback, task, nil
	} else if !errors.Is(err, commonerrors.ErrNotFound) {
		return nil, nil, err
	}
	for _, set := range []TransferRepoSet{r.router.ConfigBackup, r.router.RuntimeLogCollect, r.router.FaultLogCollect, r.router.ConfigRestore, r.router.ImsParamCollect} {
		if set.Task == nil {
			continue
		}
		if task, err := set.Task.GetByID(ctx, id); err == nil {
			return set.Task, task, nil
		} else if !errors.Is(err, commonerrors.ErrNotFound) {
			return nil, nil, err
		}
	}
	return nil, nil, commonerrors.ErrNotFound
}

func (r *RoutingTaskRepository) Create(ctx context.Context, task *UpgradeTask) error {
	return r.pickByHint(ctx).Create(ctx, task)
}

func (r *RoutingTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*UpgradeTask, error) {
	_, task, err := r.pickByID(ctx, id)
	return task, err
}

func (r *RoutingTaskRepository) Update(ctx context.Context, task *UpgradeTask) error {
	repo, _, err := r.pickByID(ctx, task.ID)
	if err != nil {
		return err
	}
	return repo.Update(ctx, task)
}

func (r *RoutingTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus, result TaskResult) error {
	repo, _, err := r.pickByID(ctx, id)
	if err != nil {
		return err
	}
	return repo.UpdateStatus(ctx, id, status, result)
}

func (r *RoutingTaskRepository) IncrementCounts(ctx context.Context, taskID uuid.UUID, successDelta, failDelta int) error {
	repo, _, err := r.pickByID(ctx, taskID)
	if err != nil {
		return err
	}
	return repo.IncrementCounts(ctx, taskID, successDelta, failDelta)
}

func (r *RoutingTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	repo, _, err := r.pickByID(ctx, id)
	if err != nil {
		// Delete 失败 → fallback 旧表也试一下（Create 前的 cleanup 路径，可能根本没存）
		return r.fallback.Delete(ctx, id)
	}
	return repo.Delete(ctx, id)
}

// List 跨业务列表查询：fan-out 各表 list，合并后按 created_at DESC 排序。
// 这是 SoftwareService.ListTasks 与 ufte.Service.loadAllTasks 走的路径
// （UFTE 任务管理 Tab「任务列表」跨业务用 List + filter）。
//
// #653 同类：每张 task 表内部循环翻页拉全量。早期实现把 filter 原样下发到每张表，
// 然后按 union 后的 total 算 totalPages 给外层切片——但底层 PgTaskRepository.List
// 硬 cap pageSize=100，导致单业务任务数 > 100 时只拿到前 100 条，第 101 条起
// 静默丢失（外层 loadAllTasks 看到 page >= totalPages 就 break）。
// 修复策略与 RoutingSubTaskRepository.ListAll 一致：单表内 cap=100 循环翻页拉全，
// 累加 total 一次，再 union + sort + 应用层切片。
func (r *RoutingTaskRepository) List(ctx context.Context, filter UpgradeTaskFilter) (*model.ListResponse[UpgradeTask], error) {
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

	// 每张表内部循环翻页拉全量。单表分页大小跟 PgTaskRepository 的硬 cap 100 对齐。
	const perTablePageSize = 100
	merged := make([]UpgradeTask, 0)
	var total int64
	for _, repo := range r.allTaskRepos() {
		repoTotalSeen := false
		for subPage := 1; ; subPage++ {
			sub := filter
			sub.Page = subPage
			sub.PageSize = perTablePageSize
			res, err := repo.List(ctx, sub)
			if err != nil {
				return nil, err
			}
			if !repoTotalSeen {
				// 每张表只累加一次 total（res.Total 是该表过滤后的全行数，与 page 无关）
				total += res.Total
				repoTotalSeen = true
			}
			merged = append(merged, res.Items...)
			if subPage >= res.TotalPages || len(res.Items) == 0 {
				break
			}
		}
	}

	// 应用层排序（各表已 ORDER BY created_at DESC，merge 再排）
	sortTasksByCreatedDesc(merged)

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(merged) {
		merged = nil
	} else if end > len(merged) {
		merged = merged[start:]
	} else {
		merged = merged[start:end]
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return &model.ListResponse[UpgradeTask]{
		Items: merged, Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages,
	}, nil
}

// allTaskRepos 返回所有装载的 main task repo（fallback 优先）。
func (r *RoutingTaskRepository) allTaskRepos() []BasicTaskRepo {
	out := []BasicTaskRepo{r.fallback}
	for _, set := range []TransferRepoSet{r.router.ConfigBackup, r.router.RuntimeLogCollect, r.router.FaultLogCollect, r.router.ConfigRestore, r.router.ImsParamCollect} {
		if set.Task != nil {
			out = append(out, set.Task)
		}
	}
	return out
}

// Canary 三方法直接转 fallback（4 张新业务表 schema 复制了 canary 列但业务上用不到）。
func (r *RoutingTaskRepository) GetCanaryFields(ctx context.Context, id uuid.UUID) (*CanaryFields, error) {
	return r.fallback.GetCanaryFields(ctx, id)
}
func (r *RoutingTaskRepository) UpdateCanaryFields(ctx context.Context, id uuid.UUID, fields *CanaryFields) error {
	return r.fallback.UpdateCanaryFields(ctx, id, fields)
}
func (r *RoutingTaskRepository) ListActiveCanaryTaskIDs(ctx context.Context) ([]uuid.UUID, error) {
	return r.fallback.ListActiveCanaryTaskIDs(ctx)
}

// sortTasksByCreatedDesc 简单冒泡（小数据集，无需 sort.Slice 开销）
func sortTasksByCreatedDesc(items []UpgradeTask) {
	n := len(items)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			ti := time.Time(items[i].CreatedAt)
			tj := time.Time(items[j].CreatedAt)
			if tj.After(ti) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

// =============================================================================
// RoutingSubTaskRepository — sub_task wrapper
// =============================================================================

var _ SubTaskRepository = (*RoutingSubTaskRepository)(nil)

// RoutingSubTaskRepository 同上理。
type RoutingSubTaskRepository struct {
	router   *TransferRepoRouter
	fallback SubTaskRepository
}

func NewRoutingSubTaskRepository(router *TransferRepoRouter, fallback SubTaskRepository) *RoutingSubTaskRepository {
	return &RoutingSubTaskRepository{router: router, fallback: fallback}
}

// pickByHint 用 context 里 fileType 选 sub_task repo。
func (r *RoutingSubTaskRepository) pickByHint(ctx context.Context) BasicSubTaskRepo {
	if hint := routeFromContext(ctx); hint != "" {
		set := r.router.ForUploadFileType(hint)
		if set.SubTask != nil {
			return set.SubTask
		}
	}
	return r.fallback
}

// candidatesByCommandKey 根据 commandKey 前缀返回候选 repo 列表（最多 2 个）。
// 不匹配 → 全部 5 张表 fan-out。
func (r *RoutingSubTaskRepository) candidatesByCommandKey(commandKey string) []BasicSubTaskRepo {
	switch {
	case strings.HasPrefix(commandKey, "IMS_PARAM_DISTRIBUTE_") ||
		strings.HasPrefix(commandKey, "CONFIG_RESTORE_") ||
		strings.HasPrefix(commandKey, "LICENSE_UPGRADE_"):
		// Direct-dispatch placeholder tasks are stored in the legacy table.
		return []BasicSubTaskRepo{r.fallback}
	case strings.HasPrefix(commandKey, "Collect NV") || strings.HasPrefix(commandKey, "Collect XML"):
		if r.router.ConfigBackup.SubTask != nil {
			return []BasicSubTaskRepo{r.router.ConfigBackup.SubTask}
		}
	case strings.HasPrefix(commandKey, "Collect LOG"):
		// LOG 前缀 RUNTIME 和 FAULT 共享 → 都试
		var cands []BasicSubTaskRepo
		if r.router.RuntimeLogCollect.SubTask != nil {
			cands = append(cands, r.router.RuntimeLogCollect.SubTask)
		}
		if r.router.FaultLogCollect.SubTask != nil {
			cands = append(cands, r.router.FaultLogCollect.SubTask)
		}
		if len(cands) > 0 {
			return cands
		}
	}
	// 其他（升级 Download / SPV 触发的 FAULT_LOG_COLLECT 用纯 UUID 当 CommandKey / 任何
	// 未识别前缀）→ fan-out 全部表，跟 GetByID / pickByID 同行为。否则像
	// ExecuteOneSetParamCollect 用 sub_task UUID 当 CommandKey 的链路会拿不到行，导致
	// HandleSetParamsResponseForCollect 静默 return，sub_task 永远卡 uploading（5 分钟前
	// 实测踩坑：fault_log_collect_sub_tasks 行查不到，handler 走 errors.Is(ErrNotFound)
	// → return nil → SOAP Fault 信息扔了）。
	return r.allSubTaskRepos()
}

// pickByID fan-out 查所有 sub_task 表找到 id。
func (r *RoutingSubTaskRepository) pickByID(ctx context.Context, id uuid.UUID) (BasicSubTaskRepo, *UpgradeSubTask, error) {
	if task, err := r.fallback.GetByID(ctx, id); err == nil {
		return r.fallback, task, nil
	} else if !errors.Is(err, commonerrors.ErrNotFound) {
		return nil, nil, err
	}
	for _, set := range []TransferRepoSet{r.router.ConfigBackup, r.router.RuntimeLogCollect, r.router.FaultLogCollect, r.router.ConfigRestore, r.router.ImsParamCollect} {
		if set.SubTask == nil {
			continue
		}
		if task, err := set.SubTask.GetByID(ctx, id); err == nil {
			return set.SubTask, task, nil
		} else if !errors.Is(err, commonerrors.ErrNotFound) {
			return nil, nil, err
		}
	}
	return nil, nil, commonerrors.ErrNotFound
}

func (r *RoutingSubTaskRepository) allSubTaskRepos() []BasicSubTaskRepo {
	out := []BasicSubTaskRepo{r.fallback}
	for _, set := range []TransferRepoSet{r.router.ConfigBackup, r.router.RuntimeLogCollect, r.router.FaultLogCollect, r.router.ConfigRestore, r.router.ImsParamCollect} {
		if set.SubTask != nil {
			out = append(out, set.SubTask)
		}
	}
	return out
}

func (r *RoutingSubTaskRepository) Create(ctx context.Context, task *UpgradeSubTask) error {
	return r.pickByHint(ctx).Create(ctx, task)
}

func (r *RoutingSubTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*UpgradeSubTask, error) {
	_, task, err := r.pickByID(ctx, id)
	return task, err
}

func (r *RoutingSubTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string) error {
	repo, _, err := r.pickByID(ctx, id)
	if err != nil {
		return err
	}
	return repo.UpdateStatus(ctx, id, status, errorMsg)
}

func (r *RoutingSubTaskRepository) UpdateStatusWithCode(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string, code FailureCode) error {
	repo, _, err := r.pickByID(ctx, id)
	if err != nil {
		return err
	}
	return repo.UpdateStatusWithCode(ctx, id, status, errorMsg, code)
}

func (r *RoutingSubTaskRepository) UpdateStatusByOperator(ctx context.Context, id uuid.UUID, status UpgradeState, errorMsg string) error {
	repo, _, err := r.pickByID(ctx, id)
	if err != nil {
		return err
	}
	return repo.UpdateStatusByOperator(ctx, id, status, errorMsg)
}

func (r *RoutingSubTaskRepository) Update(ctx context.Context, task *UpgradeSubTask) error {
	repo, _, err := r.pickByID(ctx, task.ID)
	if err != nil {
		return err
	}
	return repo.Update(ctx, task)
}

func (r *RoutingSubTaskRepository) List(ctx context.Context, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	return r.ListByTaskID(ctx, filter.TaskID, filter)
}

func (r *RoutingSubTaskRepository) ListByTaskID(ctx context.Context, taskID uuid.UUID, filter SubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
	// task_id 唯一存在于某张 sub_task 表 → fan-out 找命中那张
	for _, repo := range r.allSubTaskRepos() {
		page, err := repo.ListByTaskID(ctx, taskID, filter)
		if err != nil {
			return nil, err
		}
		if page.Total > 0 {
			return page, nil
		}
	}
	// 全部空 → 返回空列表
	return &model.ListResponse[UpgradeSubTaskWithTaskName]{Items: nil, Total: 0, Page: filter.Page, PageSize: filter.PageSize}, nil
}

// ListAll 跨业务全量 sub_task — UFTE 设备列表 / software handler 用。
//
// 跨表 union + 应用层排序分页：把 filter（不带 pagination）下发到每张子表 → 各表
// 内部按 created_at DESC 排好序的页 → merge 全部 items → 按 created_at DESC 重排 →
// 应用 page/pageSize 切片。total 是 sum，分页是新的 total 上的页。
//
// #653：早期实现把每张表的 wideFilter 硬编码 page=1、pageSize=100，**不翻页**，
// 单任务 sub_task > 100 时第 101 条起永远拉不到（UFTE 设备列表/导出丢一半）。
// 现在循环翻页拉每张表全量，再 union + 切片。
//
// 性能：每张子表当前最坏只有数百行（未来到 10 万级时单表也只是  万级 hypertable
// 早做分区即可），全 union 加载到内存可控。如需更严格性能，后续可改成数据库
// VIEW (CREATE VIEW v_all_sub_tasks AS SELECT ... FROM upgrade_sub_tasks UNION ALL
// SELECT ... FROM config_backup_sub_tasks ...) + 单次分页查询。
func (r *RoutingSubTaskRepository) ListAll(ctx context.Context, filter AllSubTaskFilter) (*model.ListResponse[UpgradeSubTaskWithTaskName], error) {
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

	// 每张表内部循环翻页拉全量。单表分页大小跟 PgSubTaskRepository 的硬 cap 100 对齐
	// （子层 pageSize>100 会被截到 100，所以这里直接用 100 一次最多拉一页）。
	const perTablePageSize = 100
	merged := make([]UpgradeSubTaskWithTaskName, 0)
	var total int64
	for _, repo := range r.allSubTaskRepos() {
		repoTotalSeen := false
		for subPage := 1; ; subPage++ {
			sub := filter
			sub.Page = subPage
			sub.PageSize = perTablePageSize
			res, err := repo.ListAll(ctx, sub)
			if err != nil {
				return nil, err
			}
			if !repoTotalSeen {
				// 每张表只累加一次 total（res.Total 是该表过滤后的全行数，与 page 无关）
				total += res.Total
				repoTotalSeen = true
			}
			merged = append(merged, res.Items...)
			if subPage >= res.TotalPages || len(res.Items) == 0 {
				break
			}
		}
	}

	sortSubTasksByCreatedDesc(merged)

	// 应用层分页
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(merged) {
		merged = nil
	} else if end > len(merged) {
		merged = merged[start:]
	} else {
		merged = merged[start:end]
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return &model.ListResponse[UpgradeSubTaskWithTaskName]{
		Items: merged, Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages,
	}, nil
}

// sortSubTasksByCreatedDesc — 同上 sortTasksByCreatedDesc 的 sub_task 版本。
func sortSubTasksByCreatedDesc(items []UpgradeSubTaskWithTaskName) {
	n := len(items)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			ti := time.Time(items[i].CreatedAt)
			tj := time.Time(items[j].CreatedAt)
			if tj.After(ti) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func (r *RoutingSubTaskRepository) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (*UpgradeSubTask, error) {
	// 设备最多在一张表里有 active sub_task（device_active_tasks PK 约束）
	for _, repo := range r.allSubTaskRepos() {
		task, err := repo.GetActiveByDeviceID(ctx, deviceID)
		if err == nil {
			return task, nil
		}
		if !errors.Is(err, commonerrors.ErrNotFound) {
			return nil, err
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (r *RoutingSubTaskRepository) GetByCommandKey(ctx context.Context, commandKey string) (*UpgradeSubTask, error) {
	for _, repo := range r.candidatesByCommandKey(commandKey) {
		task, err := repo.GetByCommandKey(ctx, commandKey)
		if err == nil {
			return task, nil
		}
		if !errors.Is(err, commonerrors.ErrNotFound) {
			return nil, err
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (r *RoutingSubTaskRepository) BatchCreate(ctx context.Context, tasks []*UpgradeSubTask) error {
	return r.pickByHint(ctx).BatchCreate(ctx, tasks)
}

func (r *RoutingSubTaskRepository) DeleteByTaskID(ctx context.Context, taskID uuid.UUID) error {
	// fan-out 删（taskID 只在一张表，删 5 张都安全）
	var firstErr error
	for _, repo := range r.allSubTaskRepos() {
		if err := repo.DeleteByTaskID(ctx, taskID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (r *RoutingSubTaskRepository) FailStale(ctx context.Context, cutoffs StaleTimeouts) (StaleFailures, error) {
	var merged StaleFailures
	// per-business override：fault_log_collect 的 uploading 走 FaultLogUpload（默认 15min），
	// 跟其它表（30min TransferComplete）拉开。其它业务暂时无差异化需求，沿用 cutoffs 原值。
	faultLogCuts := cutoffs
	if cutoffs.FaultLogUpload > 0 {
		faultLogCuts.TransferComplete = cutoffs.FaultLogUpload
	}
	for _, repo := range r.allSubTaskRepos() {
		c := cutoffs
		if r.router.FaultLogCollect.SubTask != nil && repo == r.router.FaultLogCollect.SubTask {
			c = faultLogCuts
		}
		m, err := repo.FailStale(ctx, c)
		if err != nil {
			return StaleFailures{}, err
		}
		merged.Merge(m)
	}
	return merged, nil
}

// FailStaleWithDetails preserves exact reaped device identities where the
// physical repository supports them. Legacy split-table repositories still
// contribute placeholder rows so parent task counts continue to finalize.
func (r *RoutingSubTaskRepository) FailStaleWithDetails(ctx context.Context, cutoffs StaleTimeouts) ([]UpgradeSubTask, error) {
	var failed []UpgradeSubTask
	faultLogCuts := cutoffs
	if cutoffs.FaultLogUpload > 0 {
		faultLogCuts.TransferComplete = cutoffs.FaultLogUpload
	}
	for _, repo := range r.allSubTaskRepos() {
		c := cutoffs
		if r.router.FaultLogCollect.SubTask != nil && repo == r.router.FaultLogCollect.SubTask {
			c = faultLogCuts
		}
		if detailed, ok := repo.(interface {
			FailStaleWithDetails(context.Context, StaleTimeouts) ([]UpgradeSubTask, error)
		}); ok {
			items, err := detailed.FailStaleWithDetails(ctx, c)
			if err != nil {
				return nil, err
			}
			failed = append(failed, items...)
			continue
		}
		failures, err := repo.FailStale(ctx, c)
		if err != nil {
			return nil, err
		}
		for taskID, count := range failures.TaskCounts {
			for range count {
				failed = append(failed, UpgradeSubTask{TaskID: taskID, Status: UpgradeFailed})
			}
		}
	}
	return failed, nil
}

func (r *RoutingSubTaskRepository) UpdateFailureReasonByTask(ctx context.Context, taskID uuid.UUID, code FailureCode) error {
	for _, repo := range r.allSubTaskRepos() {
		if err := repo.UpdateFailureReasonByTask(ctx, taskID, code); err != nil {
			return err
		}
	}
	return nil
}

// UpdateDestVersionByCommandKey 走 commandKey 前缀候选：CONFIG_RESTORE / LICENSE_UPGRADE
// 等"直接派发"任务的 sub_task 落在 fallback 旧表（CreatePlaceholderTrackingTask 路径
// 当前不注 route hint）。candidatesByCommandKey 不识别 CONFIG_RESTORE_/LICENSE_UPGRADE_
// 前缀 → 全表 fan-out，结果一致。每张表 UPDATE WHERE command_key= 仅命中含此 key 的行，
// 不影响其它表。
func (r *RoutingSubTaskRepository) UpdateDestVersionByCommandKey(ctx context.Context, commandKey, destVersion string) error {
	for _, repo := range r.candidatesByCommandKey(commandKey) {
		if err := repo.UpdateDestVersionByCommandKey(ctx, commandKey, destVersion); err != nil {
			return err
		}
	}
	return nil
}

// UpdateDestVersionByID 按子任务 ID 路由到命中那张表后单字段更新 dest_version（qa-614 #371）。
func (r *RoutingSubTaskRepository) UpdateDestVersionByID(ctx context.Context, id uuid.UUID, destVersion string) error {
	repo, _, err := r.pickByID(ctx, id)
	if err != nil {
		return err
	}
	return repo.UpdateDestVersionByID(ctx, id, destVersion)
}
