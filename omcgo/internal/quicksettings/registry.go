package quicksettings

import "sync"

// Registry 进程内存保存按制式索引的分组元数据。
//
// 启动期由 Loader 调 Replace 写入，REST handler 调 GetByTech 读取。
// 数据量小（55 项 + 4 个分组结构），不需要 Redis L2 缓存。
type Registry struct {
	mu     sync.RWMutex
	byTech map[TechCode][]Group
}

// NewRegistry 构造空 Registry。
func NewRegistry() *Registry {
	return &Registry{byTech: make(map[TechCode][]Group)}
}

// Replace 原子替换某制式的分组列表。Loader 启动期与 Reload 时调用。
func (r *Registry) Replace(tech TechCode, groups []Group) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]Group, len(groups))
	copy(cp, groups)
	r.byTech[tech] = cp
}

// GetByTech 返回指定制式的分组列表副本（按 Registry 内部顺序）。
// 未注册的制式返回空切片。
func (r *Registry) GetByTech(tech TechCode) []Group {
	r.mu.RLock()
	defer r.mu.RUnlock()
	src := r.byTech[tech]
	out := make([]Group, len(src))
	copy(out, src)
	return out
}
