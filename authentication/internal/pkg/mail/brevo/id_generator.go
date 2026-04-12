package brevo

import (
	"sync"
)

type IDGenerator struct {
	idCounter int64
	mu        sync.Mutex
}

func (t *IDGenerator) NextID() int64 {
	t.mu.Lock()
	t.idCounter++
	t.mu.Unlock()
	return t.idCounter
}
