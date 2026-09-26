package item

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/analysis"
	"ant/internal/domain/engine"
	dom "ant/internal/domain/item"
	"ant/internal/domain/kernel"
)

// Проекции модуля item (AD-45: один писатель — item; пишутся эффектами в
// транзакции Append вместе с курсором; пересобираются `ant rebuild`).
const (
	// ProjectionRow — строка изделия для списков (item.item.list): оси статуса,
	// шаг, партии, задание. Ключ — item_id. Проекция изделия: её заменяет воркер
	// по итогу каждой пересвёртки.
	ProjectionRow = "item.row"
	// ProjectionIndex — перечень изделий (глобальная, роль projector). Ключ
	// IndexKey; значение — IndexRecord.
	ProjectionIndex = "item.index"
	// IndexKey — ключ перечня изделий.
	IndexKey = "_all"
	// StageState — состояние межизделийной стадии (писатель crossitem,
	// application/crossitem.StateProjection): генеалогия, партии, носители.
	StageState    = "crossitem.stage"
	stageStateKey = "state"
)

// RowRecord — значение проекции item.row.
type RowRecord struct {
	ItemRow
	OrderID  string   `json:"order_id,omitempty"`
	LotIDs   []string `json:"lot_ids,omitempty"`
	BasisSeq int64    `json:"basis_seq"`
}

// IndexRecord — значение проекции item.index: все изделия по порядку регистрации.
type IndexRecord struct {
	IDs []string `json:"ids"`
}

// Register подключает проекции item к реестру движка (одна строка в cmd/ant
// engineRegistry).
func Register(reg *engineapp.Registry) error {
	if err := reg.AddItem(engineapp.ItemProjection{Name: ProjectionRow, Writer: dom.Module, View: rowView}); err != nil {
		return err
	}
	return reg.AddGlobal(engineapp.GlobalProjection{Name: ProjectionIndex, Writer: dom.Module, Keys: indexKeys, Step: indexStep,
		Entity: func(string) (platform.EntityKind, string, bool) { return "", "", false }})
}

// MustRegister — Register для сборки ролей: ошибка регистрации — ошибка сборки.
func MustRegister(reg *engineapp.Registry) *engineapp.Registry {
	if err := Register(reg); err != nil {
		panic(fmt.Errorf("проекции item: %w", err))
	}
	return reg
}

func rowView(itemID string, s engine.Snapshot, _ []kernel.Reaction) (any, error) {
	if !s.Item.Registered {
		return nil, nil
	}
	return RowRecord{ItemRow: rowOf(itemID, s), OrderID: s.Item.OrderID, LotIDs: s.Item.LotIDs, BasisSeq: s.BasisSeq}, nil
}

// rowOf — строка списка по итогу свёртки.
func rowOf(itemID string, s engine.Snapshot) ItemRow {
	r := ItemRow{ItemID: itemID, Label: Label(itemID), ItemTypeID: s.Item.ItemTypeID, StepKey: s.Item.StepKey, VersionLabel: s.Item.NormativeRev,
		Status: StatusOf(s), RunID: s.Item.RunID}
	if len(s.Item.LotIDs) > 0 {
		r.LotID = s.Item.LotIDs[0]
	}
	return r
}

func indexKeys(r kernel.Record) []string {
	if r.Type == catalog.ItemItemRegistered && r.ItemID != "" {
		return []string{IndexKey}
	}
	return nil
}

func indexStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	var v IndexRecord
	if len(prev) > 0 {
		if err := json.Unmarshal(prev, &v); err != nil {
			return nil, err
		}
	}
	if !slices.Contains(v.IDs, r.ItemID) {
		v.IDs = append(v.IDs, r.ItemID)
	}
	if v.IDs == nil {
		v.IDs = []string{}
	}
	return json.Marshal(v)
}

// Label — номер детали для людей: локальная часть ID без кода предприятия и
// префикса прогона; машинные F-/R-/C- — кириллицей, как на бирке.
func Label(itemID string) string {
	local := itemID
	if i := strings.IndexByte(local, ':'); i >= 0 {
		local = local[i+1:]
	}
	if i := strings.LastIndexByte(local, '/'); i >= 0 {
		local = local[i+1:]
	}
	for _, r := range [][2]string{{"F-", "Ф-"}, {"R-", "К-"}, {"C-", "КР-"}} {
		if strings.HasPrefix(local, r[0]) {
			return r[1] + local[len(r[0]):]
		}
	}
	return local
}

// StatusOf — оси статуса изделия (§3b PRD, AD-30) по итогу свёртки: каждую
// ось показывает модуль-владелец; пока владелец оси не выставил её в
// Snapshot, показывается вывод из записей изделия (изоляция при сомнении в
// идентификации, сдерживание по генеалогии) — сама ось не меняется.
func StatusOf(s engine.Snapshot) ItemStatus {
	st := ItemStatus{Position: "in_queue", Quality: "not_inspected", Disposition: "none", Containment: "none", ErpAccounting: "not_sent"}
	it := s.Item
	switch {
	case it.Released:
		st.Position = "completed"
	case it.Questioned():
		st.Position = "isolated"
	case len(it.Running) > 0:
		st.Position = "in_progress"
	}
	if q := string(s.Quality.Axis); q != "" {
		st.Quality = q
	}
	if h := it.HoldLevel(); h != "" {
		st.Containment = h
	}
	for _, m := range incidents(s.Analysis) {
		if m.Action == "block" && holdRank(st.Containment) < holdRank("item_hold") {
			st.Containment = "item_hold"
		}
	}
	st.Summary = Summary(st, s)
	return st
}

func holdRank(l string) int {
	return slices.Index([]string{"none", "observe", "additional_check", "item_hold", "lot_hold"}, l)
}

// Summary — сводный статус для списков и карты (словарь item_summary): из
// осей, сам осью не является.
func Summary(st ItemStatus, s engine.Snapshot) string {
	switch st.Disposition {
	case "scrap":
		return "scrapped"
	case "return_to_supplier":
		return "returned"
	case "rework":
		return "in_rework"
	case "repair":
		return "in_repair"
	case "use_as_is":
		return "accepted_with_concession"
	}
	switch {
	case st.Quality == "nonconforming":
		return "nonconforming"
	case st.Containment == "item_hold" || st.Containment == "lot_hold" || st.Position == "isolated":
		return "hold"
	case st.Quality == "signal":
		return "suspect"
	case st.Containment == "additional_check" || st.Quality == "unable_to_assess":
		return "reinspection_required"
	case st.Quality == "accepted_with_concession":
		return "accepted_with_concession"
	case st.Position == "completed":
		return "released"
	}
	for _, m := range incidents(s.Analysis) {
		switch m.Status {
		case "suspect", "unknown":
			return "suspect"
		case "excluded":
			return "cleared"
		}
	}
	return "in_process"
}

// incidents — последний статус изделия в каждом инциденте (FR-62).
func incidents(a analysis.State) []analysis.Membership {
	var out []analysis.Membership
	for _, m := range a.Incidents {
		if i := slices.IndexFunc(out, func(x analysis.Membership) bool { return x.IncidentID == m.IncidentID }); i >= 0 {
			out[i] = m
			continue
		}
		out = append(out, m)
	}
	return out
}
