package loader

import (
	"context"
	"sync"

	"ant/internal/application/platform"
)

// MemoryCursor — курсор заготовок в памяти процесса (ключ fixture_cursor =
// memory, AD-35): для тестов и одной копии api. Общий для копий курсор —
// infrastructure/storage/fixtures (Postgres).
type MemoryCursor struct {
	mu sync.Mutex
	st platform.CursorState
}

// NewMemoryCursor — пустой курсор: мир заготовок встаёт на сценарий по умолчанию.
func NewMemoryCursor() *MemoryCursor { return &MemoryCursor{} }

// Current — текущее положение.
func (c *MemoryCursor) Current(context.Context) (platform.CursorState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.st, nil
}

// Move ставит курсор.
func (c *MemoryCursor) Move(_ context.Context, to platform.CursorState) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.st = to
	return nil
}

var _ platform.FixtureCursor = (*MemoryCursor)(nil)
