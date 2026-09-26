package process

import (
	"context"

	dp "ant/internal/domain/process"
)

// Действующая версия процесса для соседних модулей (эпик 39, AD-17): хеш —
// для закрепления за новыми изделиями (item), имена узлов по step_key — для
// подписей срезов и карт аналитики (UI-21). Соседи получают их через свои
// ведомые порты; адаптеры собирает cmd/ant над VersionStore.

// ActiveVersion — действующая версия основного процесса: последняя введённая
// в действие (FR-22). ok=false — действующей версии нет (хранилище пусто или
// все версии — черновики).
func ActiveVersion(ctx context.Context, store VersionStore) (VersionRecord, bool, error) {
	all, err := store.List(ctx)
	if err != nil {
		return VersionRecord{}, false, err
	}
	v, ok := Active(VersionsOf(all, DefaultProcess(all)))
	return v, ok, nil
}

// ActiveVersionHash — хеш действующей версии основного процесса ("" — нет
// действующей): его закрепляет за собой новое изделие (AD-17); изделия,
// запущенные раньше, доделываются по своей закреплённой версии.
func ActiveVersionHash(ctx context.Context, store VersionStore) (string, error) {
	v, ok, err := ActiveVersion(ctx, store)
	if err != nil || !ok {
		return "", err
	}
	return v.Hash, nil
}

// StepNames — step_key → имя узла BPMN версии v (узлы без имени пропускаются).
// Версия не разбирается — пустой словарь: подписи останутся кодами.
func StepNames(v VersionRecord) map[string]string {
	out := map[string]string{}
	d, _ := dp.Parse(v.XML)
	if d == nil {
		return out
	}
	for key, id := range d.Steps {
		if n := d.Nodes[id]; n != nil && n.Name != "" {
			out[key] = n.Name
		}
	}
	return out
}

// ActiveStepNames — имена узлов действующей версии основного процесса по step_key.
func ActiveStepNames(ctx context.Context, store VersionStore) (map[string]string, error) {
	v, ok, err := ActiveVersion(ctx, store)
	if err != nil || !ok {
		return map[string]string{}, err
	}
	return StepNames(v), nil
}
