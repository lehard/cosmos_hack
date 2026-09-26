package analytics

import (
	"encoding/json"
	"fmt"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	domain "ant/internal/domain/analytics"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	dp "ant/internal/domain/process"
)

// Имена глобальных проекций показателей (писатель — analytics, AD-45).
const (
	// ProjectionEquipment — простои оборудования и остановки точек процесса.
	ProjectionEquipment = "analytics.equipment"
	// ProjectionIncident — гипотезы, причины и ошибки исполнителей по инцидентам.
	ProjectionIncident = "analytics.incident"
)

// Contributions — источник вкладов показателей для движка (AD-45): воркер
// вызывает его при каждой пересвёртке изделия и заменяет строки изделия
// целиком в той же транзакции, что и курсор. Регистрация — Register.
func Contributions(itemID string, s engine.Snapshot, input []kernel.Record) ([]engineapp.Contribution, error) {
	rows := domain.Contribute(itemID, input)
	if ps, ok := positions(s); ok {
		// «Очередь» и «в работе» сейчас — по положению изделия в процессе (эпик 16).
		rows = domain.ApplyPositions(itemID, rows, ps)
	}
	out := make([]engineapp.Contribution, 0, len(rows))
	for _, r := range rows {
		c, err := EncodeRow(itemID, r)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// positions — положение изделия по процессу из итога свёртки (process
// State.Positions над закреплённой версией): очередь — в очереди шага и на
// точке предъявления; работа — выполнение и контроль. Версии нет — ok = false
// (очередь и работа — из фактов, как до эпика 17).
func positions(s engine.Snapshot) ([]domain.Position, bool) {
	env := s.Process.Env()
	if env.Def == nil {
		return nil, false
	}
	var out []domain.Position
	for _, p := range s.Process.Positions(env) {
		if p.Join {
			continue
		}
		var kind string
		switch p.Position {
		case dp.PosInQueue, dp.PosAtGate:
			kind = domain.RowQueue
		case dp.PosInProgress, dp.PosAtInspection:
			kind = domain.RowInProgress
		default:
			continue
		}
		out = append(out, domain.Position{Step: p.StepKey, Kind: kind, Since: p.Since})
	}
	return out, true
}

// Register подключает показатели к движку: вклады изделий и глобальные
// проекции оборудования и инцидентов (роль projector). Одна строка в
// engineRegistry (cmd/ant).
func Register(r *engineapp.Registry) error {
	r.AddContributor(Contributions)
	if err := r.AddGlobal(engineapp.GlobalProjection{
		Name: ProjectionEquipment, Writer: "analytics", Keys: domain.EquipmentKeys,
		Step: step(domain.EquipmentStep), Entity: liveMap,
	}); err != nil {
		return err
	}
	return r.AddGlobal(engineapp.GlobalProjection{
		Name: ProjectionIncident, Writer: "analytics", Keys: domain.IncidentKeys,
		Step: step(domain.IncidentStep), Entity: liveMap,
	})
}

// liveMap — изменение показателей приходит клиенту как live_map (ключи кэша
// аналитики — под live_map, эпик 15).
func liveMap(string) (platform.EntityKind, string, bool) {
	return platform.EntityLiveMap, "global", true
}

// step — (состояние JSON, запись) → состояние JSON над чистой свёрткой домена.
func step[S any](f func(S, string, kernel.Record) S) func(string, json.RawMessage, kernel.Record) (json.RawMessage, error) {
	return func(key string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
		var s S
		if len(prev) > 0 {
			if err := json.Unmarshal(prev, &s); err != nil {
				return nil, fmt.Errorf("проекция %s: %w", key, err)
			}
		}
		b, err := engine.Canonical(f(s, key, r))
		if err != nil {
			return nil, err
		}
		return b, nil
	}
}
