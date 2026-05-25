package device

import "sync"

// Sequencer 保证同一 device（按 serial_number）的事件处理在进程内串行。
//
// 设计动机（真机验证 2026-05-25 暴露的事件竞态）：
//   - ACS 收 SOAP 是顺序的（同一 CPE 一个 HTTP session 内一来一回）；
//   - 但 ACS publish 到 EventBus 后，不同 subject 的订阅者并发处理，同一 device 的
//     PERIODIC Inform 与 RebootResponse 等事件会乱序覆盖（PERIODIC 异步写 is_online=true
//     可能晚于 RebootResponse 写 is_online=false，导致 BOOT 来时仍是 Active，
//     device.online 不发，PM SPV 不重发）；
//   - 在 EventBus 之上加 per-device 锁，按 publish 顺序 FIFO 处理，从根因消除竞态，
//     而不是在 UpdateFromInform 里给 BOOT 加例外补丁。
//
// 适用范围：当前 OMC docker-compose 单 worker 实例。多 worker 实例时需要换 Redis
// 分布式锁或 NATS partition assignment（device_sn → fixed worker）—— v2 议题。
//
// 容量预估：100 万设备 ≈ 100 万 *sync.Mutex（每个 ~24 字节）≈ 24 MB；
// 不加 LRU 回收（mutex 不持有 goroutine 时仅占内存，GC 不会回收 sync.Map 已写入项，
// 但成本可接受）。100 万 + 高频 churn 场景再加回收。
//
// 死锁防护：调用方拿到 unlock 闭包必须用 defer 包住，确保 panic 时锁能释放。
// 持锁期间禁止再 acquire 同一 device 的锁（不可重入）。
type Sequencer struct {
	locks sync.Map // map[string]*sync.Mutex
}

// NewSequencer 构造一个空的 Sequencer，所有方法 goroutine-safe。
func NewSequencer() *Sequencer {
	return &Sequencer{}
}

// Lock 获取指定 serial_number 的锁。返回 unlock 闭包，调用方必须用 defer 包住：
//
//	unlock := seq.Lock(sn)
//	defer unlock()
//
// 同一 sn 已被持锁时本调用阻塞直到拿到。空 sn 返回 no-op 闭包（防御性编程，
// 让 caller 不必每处都判空）。
func (s *Sequencer) Lock(sn string) func() {
	if sn == "" {
		return func() {}
	}
	mu := s.getOrCreate(sn)
	mu.Lock()
	return mu.Unlock
}

func (s *Sequencer) getOrCreate(sn string) *sync.Mutex {
	if v, ok := s.locks.Load(sn); ok {
		return v.(*sync.Mutex)
	}
	mu := &sync.Mutex{}
	actual, _ := s.locks.LoadOrStore(sn, mu)
	return actual.(*sync.Mutex)
}
