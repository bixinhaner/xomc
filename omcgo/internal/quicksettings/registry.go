package quicksettings

import "sync"

// Registry 进程内存保存按 paramModel name 索引的分组元数据。
//
// 启动期由 Loader 调 Replace 写入(每个 XML 文件 = 一个 paramModel),
// REST handler 调 GetByParamModel 读取。
// 数据量小(每 paramModel 数十项),不需要 Redis L2 缓存。
type Registry struct {
	mu             sync.RWMutex
	byParamModel   map[string][]Group
}

// NewRegistry 构造空 Registry。
func NewRegistry() *Registry {
	return &Registry{byParamModel: make(map[string][]Group)}
}

// Replace 原子替换某 paramModel 的分组列表。Loader 启动期与 Reload 时调用。
func (r *Registry) Replace(paramModel string, groups []Group) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]Group, len(groups))
	copy(cp, groups)
	r.byParamModel[paramModel] = cp
}

// GetByParamModel 返回指定 paramModel 的分组列表副本(按 Registry 内部顺序)。
// 未注册的 paramModel 返回空切片。
func (r *Registry) GetByParamModel(paramModel string) []Group {
	r.mu.RLock()
	defer r.mu.RUnlock()
	src := r.byParamModel[paramModel]
	out := make([]Group, len(src))
	copy(out, src)
	return out
}

// KnownParamModels 返回已加载的所有 paramModel name(按字典序),供诊断/管理 API 使用。
func (r *Registry) KnownParamModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byParamModel))
	for k := range r.byParamModel {
		out = append(out, k)
	}
	return out
}
