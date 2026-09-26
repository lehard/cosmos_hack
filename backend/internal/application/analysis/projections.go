package analysis

import (
	"encoding/json"
	"fmt"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	dom "ant/internal/domain/analysis"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// Проекции модуля analysis (AD-45: один писатель — analysis; пишутся только
// эффектами в транзакции Append вместе с курсором; пересобираются `ant rebuild`).
const (
	// ProjectionCircumstances — проекция изделия analysis.circumstances: события
	// трёх дорожек, выполнения, результаты контроля, несоответствия и статус в
	// инцидентах (FR-58, FR-153). Ключ — item_id.
	ProjectionCircumstances = "analysis.circumstances"
	// ProjectionIncident — инциденты и версии области риска (FR-61, FR-62).
	// Ключ — incident_id; ключ dom.ListKey — список инцидентов.
	ProjectionIncident = "analysis.incident"
	// ProjectionNC — несоответствия для разбора (FR-59, FR-60, FR-135). Ключ —
	// nc_id; ключ dom.ListKey — список.
	ProjectionNC = "analysis.nc"
)

// ItemView — значение проекции analysis.circumstances.
type ItemView struct {
	ItemID   string    `json:"item_id"`
	BasisSeq int64     `json:"basis_seq"`
	State    dom.State `json:"state"`
}

// Register подключает проекции analysis к реестру движка (одна строка в
// cmd/ant engineRegistry): проекцию изделия ведёт воркер по итогу свёртки,
// глобальные — роль projector.
func Register(reg *engineapp.Registry) error {
	if err := reg.AddItem(engineapp.ItemProjection{Name: ProjectionCircumstances, Writer: dom.Module, View: circumstancesView}); err != nil {
		return err
	}
	if err := reg.AddGlobal(engineapp.GlobalProjection{Name: ProjectionIncident, Writer: dom.Module, Keys: dom.IncidentKeys,
		Step: stepIncident, Entity: entity(platform.EntityIncident)}); err != nil {
		return err
	}
	return reg.AddGlobal(engineapp.GlobalProjection{Name: ProjectionNC, Writer: dom.Module, Keys: dom.NCKeys,
		Step: stepNC, Entity: entity(platform.EntityNonconformity)})
}

// MustRegister — Register для сборки ролей: ошибка регистрации — ошибка сборки.
func MustRegister(reg *engineapp.Registry) *engineapp.Registry {
	if err := Register(reg); err != nil {
		panic(fmt.Errorf("проекции analysis: %w", err))
	}
	return reg
}

// circumstancesView — значение проекции изделия по итогу свёртки; у изделия
// без сведений для разбора строки нет.
func circumstancesView(itemID string, s engine.Snapshot, _ []kernel.Reaction) (any, error) {
	a := s.Analysis
	if len(a.Marks) == 0 && len(a.Runs) == 0 && len(a.Cases) == 0 && len(a.Incidents) == 0 && len(a.Equipment) == 0 {
		return nil, nil
	}
	return ItemView{ItemID: itemID, BasisSeq: s.BasisSeq, State: a}, nil
}

func stepIncident(key string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	if key == dom.ListKey {
		return stepList(prev, r)
	}
	var v dom.IncidentRecord
	if err := unmarshal(prev, &v); err != nil {
		return nil, err
	}
	return json.Marshal(dom.StepIncident(key, v, r))
}

func stepNC(key string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	if key == dom.ListKey {
		return stepList(prev, r)
	}
	var v dom.NCRecord
	if err := unmarshal(prev, &v); err != nil {
		return nil, err
	}
	return json.Marshal(dom.StepNC(key, v, r))
}

func stepList(prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	var l dom.ListRecord
	if err := unmarshal(prev, &l); err != nil {
		return nil, err
	}
	l = dom.StepList(l, dom.ListIDOf(r))
	if l.IDs == nil {
		l.IDs = []string{}
	}
	return json.Marshal(l)
}

func unmarshal(raw json.RawMessage, v any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, v)
}

// entity — сущность SSE для ключа проекции (список не сообщается).
func entity(kind platform.EntityKind) func(string) (platform.EntityKind, string, bool) {
	return func(key string) (platform.EntityKind, string, bool) {
		if key == dom.ListKey {
			return "", "", false
		}
		return kind, key, true
	}
}
