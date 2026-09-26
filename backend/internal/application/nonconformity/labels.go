package nonconformity

import (
	"context"
	"time"

	"ant/internal/domain/engine"
)

// Человеческие названия рядом с кодами карточки и списка несоответствий
// (интерфейс стола контролёра): вид дефекта — классификатор видов дефектов
// (normative/defects, FR-125), зона — зоны типа изделия по КД (справочник
// номенклатуры, FR-46), оборудование — справочник оборудования (эпик 19).
// Названия не выдумываются: кода нет в справочнике — поля *_label нет.

// EquipmentNames — ведомый порт: название оборудования по справочнику на
// момент at (reference.equipment.defined); нет в справочнике — "", false.
// Реализация — reference.EquipmentNames; nil — названий оборудования нет.
type EquipmentNames interface {
	EquipmentName(ctx context.Context, equipmentID string, at time.Time) (string, bool)
}

// labels — названия кодов из нормативного слоя изделия (тот же Bundle, что у свёртки).
type labels struct {
	defects map[string]string
	zones   map[string]string
}

// labelsOf — названия видов дефектов и зон из нормативного слоя изделия.
func labelsOf(b engine.Bundle) labels {
	l := labels{defects: map[string]string{}, zones: map[string]string{}}
	for _, d := range b.Quality.Classifier.DefectTypes {
		if d.Code != "" && d.Name != "" {
			l.defects[d.Code] = d.Name
		}
	}
	for _, t := range b.Item.Types {
		for _, z := range t.Zones {
			if z.ID != "" && z.Name != "" {
				l.zones[z.ID] = z.Name
			}
		}
	}
	return l
}

// nameIn — название кода из словаря; нет кода или названия — nil (поле не отдаётся).
func nameIn(dict map[string]string, code *string) *string {
	if code == nil || *code == "" {
		return nil
	}
	if s, ok := dict[*code]; ok {
		return &s
	}
	return nil
}

// withLabels — названия рядом с кодами карточки: вид дефекта и зона сигнала,
// оборудование операции (на время начала операции).
func (s *Service) withLabels(ctx context.Context, v *itemView, c *NCCard) {
	for i := range c.Evidence.Signals {
		sig := &c.Evidence.Signals[i]
		sig.DefectTypeLabel = nameIn(v.Labels.defects, sig.DefectTypeCode)
		sig.ZoneLabel = nameIn(v.Labels.zones, sig.ZoneID)
	}
	op := c.Happened.Operation
	if op == nil || op.EquipmentID == nil || *op.EquipmentID == "" || s.d.Equipment == nil {
		return
	}
	at := time.Time{}
	if op.StartedAt != nil {
		at = *op.StartedAt
	}
	if name, ok := s.d.Equipment.EquipmentName(ctx, *op.EquipmentID, at); ok && name != "" {
		op.EquipmentLabel = &name
	}
}
