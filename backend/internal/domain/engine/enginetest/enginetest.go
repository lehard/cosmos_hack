// Пакет enginetest — тестовый модуль-пустышка движка с одной реакцией: пока
// настоящих модулей нет (волна 4), на нём проверяются сценарий воркера,
// сравнение реакций, поздние события, детерминизм и сквозной путь
// «журнал → воркер → проекция → SSE» (эпик 07).
//
// Правило пустышки «test.signal»: если среди результатов контроля изделия
// (`inspection.result.recorded`) есть «признак дефекта» — защитная реакция
// `quality.signal.raised` в слоте (test.signal, item:‹id›, signal); причины —
// все такие результаты. Исправление результата (`corrects`) заменяет прежний.
// Позднее событие меняет причины — новая версия слота «пересмотрен из-за
// записи ‹id›» (FR-32); исправление на «годен» убирает реакцию — защита не
// снимается, движок ставит задачу пересмотра (AD-3).
//
// Слой: domain (AD-4) — чистая функция; только для тестов и заготовок.
package enginetest

import (
	"encoding/json"
	"maps"
	"slices"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	"ant/internal/domain/quality/qualitytest"
)

// RuleID — правило пустышки.
const RuleID = "test.signal"

// Fold — свёртка движка плюс реакция пустышки (сигнатура engine.Folder).
func Fold(b engine.Bundle, input []kernel.Record) (engine.Snapshot, []kernel.Reaction) {
	s, rs := engine.Fold(b, input)
	if re, ok := Signal(engine.SortInput(input)); ok {
		rs = append(rs, re)
		slices.SortFunc(rs, func(x, y kernel.Reaction) int {
			switch {
			case x.Slot.Key() < y.Slot.Key():
				return -1
			case x.Slot.Key() > y.Slot.Key():
				return 1
			}
			return 0
		})
	}
	return s, rs
}

// Signal — единственная реакция пустышки над упорядоченным входом.
func Signal(in []kernel.Record) (kernel.Reaction, bool) {
	failing := map[string]kernel.Record{}
	item := ""
	for _, r := range in {
		if r.Type != catalog.InspectionResultRecorded {
			continue
		}
		var d struct {
			Outcome string `json:"outcome"`
		}
		if err := json.Unmarshal(r.Data, &d); err != nil {
			continue
		}
		if r.Corrects != "" {
			delete(failing, r.Corrects)
		}
		if d.Outcome == string(ev.InspectionOutcomeDefectIndicated) {
			failing[r.EventID] = r
		}
		item = r.ItemID
	}
	if len(failing) == 0 {
		return kernel.Reaction{}, false
	}
	causes := make([]kernel.Record, 0, len(failing))
	for _, k := range slices.Sorted(maps.Keys(failing)) {
		causes = append(causes, failing[k])
	}
	subject := "item:" + item
	data := ev.QualitySignalRaisedV1{
		BasisKind:       ev.QualitySignalRaisedV1BasisKindInspectionResult,
		ReactionMapRef:  "test@1#1",
		ReactionOutcome: ev.QualitySignalRaisedV1ReactionOutcomeIsolate,
		Severity:        ev.SeverityMajor,
		SignalID:        ev.ObjectID("sig-" + item),
	}
	// Реакцию строит опора модуля-эмитента (AD-40): пустышка играет за quality.
	re, err := qualitytest.SignalRaised(kernel.Slot{RuleID: RuleID, Subject: subject, TriggerKey: "signal"}, data, causes...)
	if err != nil {
		panic(err)
	}
	re.RuleRev, re.AutomationMode = "test@1", 3
	return re, true
}
