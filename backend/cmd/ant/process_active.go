package main

import (
	"context"
	"sync"

	processapp "ant/internal/application/process"
)

// activeProcess — адаптер ведомых портов «действующая версия процесса» над
// хранилищем версий модуля process (эпик 39, AD-17): хеш — новым изделиям
// (item), имена узлов по step_key — подписям аналитики (UI-21). Имена
// разбираются из XML один раз на версию.
type activeProcess struct {
	store processapp.VersionStore

	mu    sync.Mutex
	hash  string
	names map[string]string
}

// ActiveVersionHash — хеш действующей версии основного процесса ("" — нет действующей).
func (a *activeProcess) ActiveVersionHash(ctx context.Context) (string, error) {
	return processapp.ActiveVersionHash(ctx, a.store)
}

// StepNames — step_key → имя узла BPMN действующей версии.
func (a *activeProcess) StepNames(ctx context.Context) (map[string]string, error) {
	v, ok, err := processapp.ActiveVersion(ctx, a.store)
	if err != nil || !ok {
		return nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.hash != v.Hash || a.names == nil {
		a.hash, a.names = v.Hash, processapp.StepNames(v)
	}
	return a.names, nil
}
