package simulation

import (
	"context"
	"slices"
	"strings"
	"sync"

	app "ant/internal/application/simulation"
)

// MemoryRuns — состояние прогонов в памяти роли stands (одна копия, AD-6).
// Истина — журнал прогона (simulation.run.*, time.clock.ticked) и план
// генератора, детерминированный по seed; хранилище в Postgres (схема
// simulation) — следующий шаг, когда пульт понадобится нескольким копиям api.
type MemoryRuns struct {
	mu   sync.Mutex
	runs map[string]*app.RunState
}

// NewMemoryRuns — пустое хранилище.
func NewMemoryRuns() *MemoryRuns { return &MemoryRuns{runs: map[string]*app.RunState{}} }

var _ app.RunStore = (*MemoryRuns)(nil)

// Save — сохранить копию состояния.
func (m *MemoryRuns) Save(_ context.Context, r *app.RunState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[r.RunID] = app.Snapshot(r)
	return nil
}

// Load — копия состояния прогона.
func (m *MemoryRuns) Load(_ context.Context, runID string) (*app.RunState, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[runID]
	if !ok {
		return nil, false, nil
	}
	return app.Snapshot(r), true, nil
}

// List — копии всех прогонов по run_id.
func (m *MemoryRuns) List(context.Context) ([]*app.RunState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, 0, len(m.runs))
	for id := range m.runs {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, strings.Compare)
	out := make([]*app.RunState, 0, len(ids))
	for _, id := range ids {
		out = append(out, app.Snapshot(m.runs[id]))
	}
	return out, nil
}
