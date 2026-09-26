package item

import (
	"slices"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// RuleIdentification — правило «потеря или неоднозначность идентификации →
// идентификация под сомнением → изоляция до повторной идентификации» (AD-16).
const RuleIdentification = "item.identification_questioned"

// questionedData — data item.identification.questioned v1.
type questionedData struct {
	CauseKind  string   `json:"cause_kind"`
	Candidates []string `json:"candidates,omitempty"`
	Basis      []string `json:"basis"`
}

// React вычисляет реакции модуля по состоянию после записи (AD-3, AD-40):
// «идентификация под сомнением» — по одной реакции на вопрос. Реакция
// вычисляется и после подтверждения идентификации: это факт истории, а не
// действующая защита, поэтому снятие сомнения человеком не выглядит
// исчезновением защитной реакции (AD-3).
func React(s State, env Env) kernel.Output {
	_ = env
	var out kernel.Output
	for _, q := range s.Questions {
		causes := make([]kernel.Record, 0, len(q.Basis))
		for _, b := range q.Basis {
			causes = append(causes, kernel.Record{EventID: b, OccurredAt: q.At})
		}
		re, err := kernel.NewReaction(Module, catalog.ItemIdentificationQuestioned, QuestionSlot(s.ItemID, q.Key),
			questionedData{CauseKind: q.Cause, Candidates: slices.Clone(q.Candidates), Basis: slices.Clone(q.Basis)}, causes...)
		if err != nil {
			panic(err) // тип и эмитент закреплены кодом
		}
		re.AutomationMode = 1
		out.Reactions = append(out.Reactions, re)
	}
	return out
}

// Полезные нагрузки команд модуля item для гарда (заполняет application/item).
type (
	// CarrierCmd — нанести или снять носитель.
	CarrierCmd struct {
		Type, Value string
		// Replaces — значение заменяемого носителя (перемаркировка).
		Replaces string
	}
	// ConfirmCmd — подтвердить идентификацию.
	ConfirmCmd struct{ QuestionedEventID string }
	// InterventionCmd — открыть или закрыть вмешательство.
	InterventionCmd struct {
		ID    string
		Zones []string
	}
	// AssemblyCmd — установить компонент.
	AssemblyCmd struct {
		ComponentItemID string
		ComponentLotID  string
	}
)

// Guard — доменный гард операций модуля item (AD-39): состояние изделия на
// basis_seq и команда → nil или *kernel.Refusal с кодом из contracts/errors.yaml.
// Межизделийные предусловия (носитель у другого изделия, компонент в другой
// сборке, цикл) проверяет application/item по генеалогии стадии.
func Guard(s State, env Env, cmd kernel.Command) error {
	if cmd.Action == "item.item.register" {
		if s.Registered {
			return kernel.Refuse(errcodes.ItemAlreadyRegistered, "item_id", s.ItemID)
		}
		return nil
	}
	if !s.Registered {
		return kernel.Refuse(errcodes.ItemNotRegistered, "item_id", s.ItemID)
	}
	switch cmd.Action {
	case "item.carrier.apply":
		if c, ok := cmd.Payload.(CarrierCmd); ok && c.Replaces != "" &&
			!slices.ContainsFunc(s.Carriers, func(x Carrier) bool { return x.Active() && x.Value == c.Replaces }) {
			return kernel.Refuse(errcodes.ItemCarrierNotActive, "carrier_ref", c.Replaces)
		}
	case "item.carrier.remove":
		c, _ := cmd.Payload.(CarrierCmd)
		if !slices.ContainsFunc(s.Carriers, func(x Carrier) bool { return x.Active() && x.Type == c.Type && x.Value == c.Value }) {
			return kernel.Refuse(errcodes.ItemCarrierNotActive, "carrier_ref", c.Type+":"+c.Value)
		}
	case "item.identification.confirm":
		c, _ := cmd.Payload.(ConfirmCmd)
		for _, q := range s.OpenQuestions() {
			if c.QuestionedEventID == "" || s.questionMatches(q, c.QuestionedEventID) {
				return nil
			}
		}
		return kernel.Refuse(errcodes.ItemIdentificationNotQuestioned, "questioned_event_id", c.QuestionedEventID)
	case "item.intervention.open":
		c, _ := cmd.Payload.(InterventionCmd)
		if t, ok := env.Types[s.ItemTypeID]; ok && len(t.Zones) > 0 {
			for _, z := range c.Zones {
				if !slices.ContainsFunc(t.Zones, func(d ZoneDef) bool { return d.ID == z }) {
					return kernel.Refuse(errcodes.ItemZoneUnknown, "zone_id", z, "item_type_id", s.ItemTypeID)
				}
			}
		}
	case "item.intervention.close":
		c, _ := cmd.Payload.(InterventionCmd)
		if !slices.ContainsFunc(s.OpenInterventions(), func(iv Intervention) bool { return iv.ID == c.ID }) {
			return kernel.Refuse(errcodes.ItemInterventionNotOpen, "intervention_id", c.ID)
		}
	case "item.presentation.record", "item.assembly.record", "item.release.record":
		if s.Released {
			return kernel.Refuse(errcodes.ItemAlreadyReleased, "item_id", s.ItemID)
		}
		// Изоляция до повторной идентификации (AD-16): изделие с сомнением
		// дальше не предъявляется, не собирается и не выпускается.
		if s.Questioned() {
			return kernel.Refuse(errcodes.ItemIdentificationQuestioned, "item_id", s.ItemID)
		}
		if cmd.Action == "item.release.record" && len(s.OpenInterventions()) > 0 {
			return kernel.Refuse(errcodes.NonconformityInterventionOpen)
		}
		if c, ok := cmd.Payload.(AssemblyCmd); ok && c.ComponentItemID != "" {
			if c.ComponentItemID == s.ItemID {
				return kernel.Refuse(errcodes.ItemAssemblyCycle, "component_item_id", c.ComponentItemID, "item_id", s.ItemID)
			}
			if slices.ContainsFunc(s.Components, func(x Component) bool { return x.ItemID == c.ComponentItemID }) {
				return kernel.Refuse(errcodes.ItemComponentAlreadyAssembled, "component_item_id", c.ComponentItemID, "assembly_item_id", s.ItemID)
			}
		}
	}
	return nil
}
