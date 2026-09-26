package nonconformity

import (
	"encoding/json"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Числа режима оборудования в карточке несоответствия (зона «что произошло»,
// FR-51, FR-147): у отклонения режима (equipment.deviation.detected) и сводки
// цикла (equipment.cycle.summarized) — уставка и наблюдённые значения
// числами (measurement: мантисса, масштаб, единица), а не строками. Разные
// масштабы приводятся к наибольшему. Ряда замеров в журнале нет (сводка
// ссылается на сырые данные raw_ref) — его карточка не отдаёт.

type measurement struct {
	Value *int64 `json:"value"`
	Scale int    `json:"scale"`
	Unit  string `json:"unit"`
}

type tolerance struct {
	Nominal *measurement `json:"nominal"`
	Lower   *measurement `json:"lower"`
	Upper   *measurement `json:"upper"`
}

// readingOf — числа режима записи; не запись режима или нет чисел — nil.
func readingOf(r kernel.Record) *NCParameterReading {
	switch r.Type {
	case catalog.EquipmentDeviationDetected:
		var d struct {
			Parameter string       `json:"parameter"`
			Value     *measurement `json:"value"`
			Setpoint  *tolerance   `json:"setpoint"`
		}
		if json.Unmarshal(r.Data, &d) != nil {
			return nil
		}
		return reading(d.Parameter, d.Setpoint, d.Value, d.Value)
	case catalog.EquipmentCycleSummarized:
		var d struct {
			Parameters []struct {
				Parameter       string       `json:"parameter"`
				Min             *measurement `json:"min"`
				Max             *measurement `json:"max"`
				Setpoint        *tolerance   `json:"setpoint"`
				OutOfSetpointMS int64        `json:"out_of_setpoint_ms"`
			} `json:"parameters"`
		}
		if json.Unmarshal(r.Data, &d) != nil || len(d.Parameters) == 0 {
			return nil
		}
		// Параметр вне уставки дольше всех; иначе — первый.
		p := d.Parameters[0]
		for _, x := range d.Parameters[1:] {
			if x.OutOfSetpointMS > p.OutOfSetpointMS {
				p = x
			}
		}
		return reading(p.Parameter, p.Setpoint, p.Min, p.Max)
	}
	return nil
}

// reading — числа к общему масштабу; нет ни одного числа — nil.
func reading(param string, sp *tolerance, lo, hi *measurement) *NCParameterReading {
	var all []*measurement
	if sp != nil {
		all = append(all, sp.Nominal, sp.Lower, sp.Upper)
	}
	all = append(all, lo, hi)
	out := &NCParameterReading{Parameter: param}
	found := false
	for _, m := range all {
		if m == nil || m.Value == nil {
			continue
		}
		found = true
		if m.Scale > out.Scale {
			out.Scale = m.Scale
		}
		if out.Unit == "" {
			out.Unit = m.Unit
		}
	}
	if !found {
		return nil
	}
	at := func(m *measurement) *int64 {
		if m == nil || m.Value == nil || (m.Unit != "" && m.Unit != out.Unit) {
			return nil
		}
		v := *m.Value
		for i := m.Scale; i < out.Scale; i++ {
			v *= 10
		}
		return &v
	}
	if sp != nil {
		out.SetpointNominal, out.SetpointMin, out.SetpointMax = at(sp.Nominal), at(sp.Lower), at(sp.Upper)
	}
	out.ObservedMin, out.ObservedMax = at(lo), at(hi)
	return out
}
