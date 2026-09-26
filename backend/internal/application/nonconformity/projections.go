package nonconformity

import (
	engineapp "ant/internal/application/engine"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/nonconformity"
)

// ItemProjection — проекция изделия модуля nonconformity (AD-45, один
// писатель): оси «решение по изделию» и «сдерживание», изоляция и
// несоответствия изделия. Пишет воркер эффектами в транзакции Append вместе
// с курсором; её сверяют верификатор и `make tamper` («проекция сдерживания
// расходится с журналом», AD-28). Экраны модуля читают свёртку (AD-22).
const ItemProjection = "nonconformity.item"

// ItemSummary — значение проекции nonconformity.item.
type ItemSummary struct {
	ItemID             string                  `json:"item_id"`
	Containment        string                  `json:"containment"`
	Disposition        string                  `json:"disposition"`
	Isolated           bool                    `json:"isolated"`
	PhysicallyNotMoved bool                    `json:"physically_not_moved"`
	NCs                []NCBrief               `json:"ncs"`
	Holds              []dom.ContainmentSource `json:"containment_sources,omitempty"`
}

// NCBrief — несоответствие в проекции изделия.
type NCBrief struct {
	NCID          string `json:"nc_id"`
	Status        string `json:"status"`
	Investigation string `json:"investigation"`
	Disposition   string `json:"disposition,omitempty"`
	Executed      bool   `json:"executed,omitempty"`
	Commission    bool   `json:"commission,omitempty"`
}

// ItemView — чистая функция итога свёртки изделия → значение проекции (nil —
// у изделия нет ничего по модулю).
func ItemView(itemID string, s engine.Snapshot, _ []kernel.Reaction) (any, error) {
	st := s.Nonconformity
	if len(st.NCs) == 0 && len(st.Containment) == 0 && st.Isolation == nil {
		return nil, nil
	}
	v := ItemSummary{ItemID: itemID, Containment: st.ContainmentLevel(), Disposition: st.Disposition(), Isolated: st.Isolated(),
		PhysicallyNotMoved: st.PhysicallyNotMoved(), NCs: []NCBrief{}}
	for _, n := range st.NCs {
		v.NCs = append(v.NCs, NCBrief{NCID: n.ID, Status: n.Status, Investigation: n.Investigation, Disposition: n.Disposition,
			Executed: n.Executed, Commission: n.Commission})
	}
	for _, c := range st.Containment {
		if !c.Released {
			v.Holds = append(v.Holds, c)
		}
	}
	return v, nil
}

// RegisterProjections регистрирует проекции модуля в реестре движка
// (cmd/ant engineRegistry, одна строка на модуль).
func RegisterProjections(r *engineapp.Registry) error {
	return r.AddItem(engineapp.ItemProjection{Name: ItemProjection, Writer: dom.Module, View: ItemView})
}
