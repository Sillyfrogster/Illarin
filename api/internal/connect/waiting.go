package connect

import (
	"sync"

	"github.com/google/uuid"
)

type hub struct {
	mu      sync.Mutex
	waiting map[uuid.UUID]*waiter
	limit   int
}

type waiter struct {
	work       chan struct{}
	superseded chan struct{}
}

func newHub(limit int) *hub {
	return &hub{waiting: make(map[uuid.UUID]*waiter), limit: limit}
}

func (h *hub) hold(appID uuid.UUID) (*waiter, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	previous, held := h.waiting[appID]
	if !held && len(h.waiting) >= h.limit {
		return nil, false
	}
	if held {
		close(previous.superseded)
	}
	current := &waiter{
		work:       make(chan struct{}, 1),
		superseded: make(chan struct{}),
	}
	h.waiting[appID] = current
	return current, true
}

func (h *hub) release(appID uuid.UUID, held *waiter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.waiting[appID] == held {
		delete(h.waiting, appID)
	}
}

func (h *hub) signal(appID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	held, waiting := h.waiting[appID]
	if !waiting {
		return
	}
	select {
	case held.work <- struct{}{}:
	default:
	}
}
