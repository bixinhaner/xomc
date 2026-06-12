package software

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// fairSlotPool 系统级（跨任务）升级设备槽位池：上限运行期可调，空闲槽位在多个
// 任务之间按"任务轮转"（round-robin）分配——每个有等待设备的任务轮流拿一个槽，
// 保证多任务并发时雨露均沾。若用简单 FIFO/信号量，先创建的任务会把全部槽位
// 占满，后创建的任务要整整等一轮设备升级时长才能启动第一台。
type fairSlotPool struct {
	mu    sync.Mutex
	limit int
	inUse int
	// queues 每任务一个 FIFO 等待队列；order 是有等待者的任务的轮转顺序，
	// rr 是轮转游标。任务队列清空后从 order 中摘除。
	queues map[uuid.UUID][]*slotWaiter
	order  []uuid.UUID
	rr     int
}

type slotWaiter struct {
	ready     chan struct{} // 拿到槽位时 close
	granted   bool
	cancelled bool
}

func newFairSlotPool(limit int) *fairSlotPool {
	if limit < 1 {
		limit = 1
	}
	return &fairSlotPool{
		limit:  limit,
		queues: make(map[uuid.UUID][]*slotWaiter),
	}
}

// SetLimit 调整上限；≤0 忽略（视为"未配置"）。调大后立即按轮转顺序放行等待者；
// 调小不打断已占用的槽位，只是要等占用数降回上限以下才再放行。
func (p *fairSlotPool) SetLimit(n int) {
	if n <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.limit = n
	p.grantLocked()
}

// Acquire 为 taskID 占用一个槽位；满载时进入该任务的等待队列（跨任务轮转放行），
// ctx 取消（任务急停）时放弃排队并返回 false。
func (p *fairSlotPool) Acquire(ctx context.Context, taskID uuid.UUID) bool {
	p.mu.Lock()
	// 快路径：有空槽且无人排队（有人排队时必须排队，否则插队破坏轮转公平）。
	if p.inUse < p.limit && len(p.order) == 0 {
		p.inUse++
		p.mu.Unlock()
		return true
	}
	w := &slotWaiter{ready: make(chan struct{})}
	p.queues[taskID] = append(p.queues[taskID], w)
	if len(p.queues[taskID]) == 1 {
		p.order = append(p.order, taskID)
	}
	p.mu.Unlock()

	select {
	case <-w.ready:
		return true
	case <-ctx.Done():
		p.mu.Lock()
		defer p.mu.Unlock()
		if w.granted {
			// 取消与放行竞态：槽位已发给我们但不再使用，退回并转授下一位。
			p.inUse--
			p.grantLocked()
			return false
		}
		w.cancelled = true // 惰性删除：放行循环遇到时跳过
		return false
	}
}

// Release 释放一个槽位并按轮转顺序转授下一个等待者。
func (p *fairSlotPool) Release() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.inUse > 0 {
		p.inUse--
	}
	p.grantLocked()
}

// grantLocked 在持锁状态下把空闲槽位按任务轮转发给等待者：每轮从 rr 指向的任务
// 队列头部取一个未取消的等待者放行，然后游标移到下一个任务。
func (p *fairSlotPool) grantLocked() {
	for p.inUse < p.limit && len(p.order) > 0 {
		if p.rr >= len(p.order) {
			p.rr = 0
		}
		tid := p.order[p.rr]
		q := p.queues[tid]

		var w *slotWaiter
		for len(q) > 0 {
			head := q[0]
			q = q[1:]
			if !head.cancelled {
				w = head
				break
			}
		}

		if len(q) == 0 {
			delete(p.queues, tid)
			p.order = append(p.order[:p.rr], p.order[p.rr+1:]...)
			// rr 不前移：摘除后原位置已是下一个任务。
		} else {
			p.queues[tid] = q
			p.rr++ // 本任务拿到一个槽，游标轮到下一个任务
		}

		if w != nil {
			w.granted = true
			close(w.ready)
			p.inUse++
		}
	}
}

// InUse 当前占用槽位数（监控/测试用）。
func (p *fairSlotPool) InUse() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.inUse
}
