package analysis

import (
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/documents"
	"ant/internal/domain/item"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
	"ant/internal/domain/nonconformity"
	"ant/internal/domain/process"
	"ant/internal/domain/quality"
	"ant/internal/domain/vision"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "analysis"

// Env — закреплённая при запуске изделия часть нормативного слоя, нужная
// модулю analysis (AD-17: процесс, план контроля, карта реакций, шаблоны …), и срез
// справочников на occurred_at (AD-31). Собирает движок из пакета версии.
type Env struct{}

// Upstream — состояния модулей раньше analysis в композиции на этом шаге
// (только чтение, AD-40): поздний модуль видит вывод раннего, обратно — только
// через функцию-намерение раннего модуля.
type Upstream struct {
	Item          *item.State
	Process       *process.State
	Vision        *vision.State
	Quality       *quality.State
	Machinelogs   *machinelogs.State
	Documents     *documents.State
	Nonconformity *nonconformity.State
}

// Reduce применяет запись входа изделия (факт, решение, адресованную запись
// стадии) к состоянию модуля (AD-5). Реакции в свёртку не входят (AD-3).
func Reduce(s State, r kernel.Record, env Env, up Upstream) State {
	_, _ = env, up
	return reduceItem(s, r)
}

// RuleHypotheses — правило вывода разбора (AD-3: версия вывода разбора —
// реакция; слот — правило, изделие, несоответствие).
const RuleHypotheses = "analysis.hypotheses"

// React вычисляет реакции модуля по состоянию после записи и намерения к
// ранним модулям (AD-3, AD-40): версию вывода разбора incident.hypothesis.computed
// по каждому несоответствию изделия (FR-58, FR-59). Оборудование в свёртке
// изделия — только события, привязанные к изделию приёмом; временная линия
// оборудования (эпик 23) придёт через Upstream.Machinelogs (EquipmentFrom).
// Без событий оборудования выводов о нём версия не содержит и не категорична;
// полный разбор с портом оборудования даёт запрос analysis.circumstances.read.
func React(s State, env Env, up Upstream) kernel.Output {
	_ = env
	var out kernel.Output
	subject := itemSubject(s)
	for _, c := range s.Cases {
		a, ok := Analyze(s, s.ItemID, c.NCID, EquipmentFrom(up))
		if !ok {
			continue
		}
		causes := make([]kernel.Record, 0, len(a.Causes))
		for _, id := range a.Causes {
			causes = append(causes, kernel.Record{EventID: id, OccurredAt: occurredOf(s, id)})
		}
		re, err := kernel.NewReaction(Module, catalog.IncidentHypothesisComputed,
			kernel.Slot{RuleID: RuleHypotheses, Subject: subject, TriggerKey: c.NCID}, hypothesisComputed(a), causes...)
		if err != nil {
			continue
		}
		re.AutomationMode = 1
		out.Reactions = append(out.Reactions, re)
	}
	return out
}

// EquipmentFrom — события оборудования для разбора из состояния machinelogs в
// свёртке изделия (порт временной линии, AD-42). Эпик 23 наполняет
// machinelogs.State; до него — nil: «порт не подключён».
func EquipmentFrom(up Upstream) []EquipmentEvent {
	_ = up
	return nil
}

// itemSubject — субъект слота реакции: поток изделия (reaction_id различается
// у изделий группового несоответствия).
func itemSubject(s State) string { return "item:" + s.ItemID }

// occurredOf — время записи-причины (для occurred_at реакции = наибольший
// среди причин, AD-37).
func occurredOf(s State, id string) time.Time {
	for _, m := range s.Marks {
		if m.EventID == id {
			return m.OccurredAt
		}
	}
	for _, c := range s.Cases {
		if c.EventID == id {
			return c.At
		}
	}
	for _, f := range s.Findings {
		if f.EventID == id {
			return f.At
		}
	}
	return time.Time{}
}

// Guard — доменный гард операций модуля analysis над состоянием изделия (AD-39).
// Операции analysis — над потоками инцидентов: их гард — GuardIncident над
// состоянием инцидента; у изделия ограничений нет.
func Guard(s State, env Env, up Upstream, cmd kernel.Command) error {
	_, _, _, _ = s, env, up, cmd
	return nil
}
