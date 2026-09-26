package security

import (
	"encoding/json"
	"slices"
	"strings"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/statuses"
)

// Action — критическое действие MVP (FR-146, AD-28): тип основной записи,
// название действия, было → стало. Признак критичности и группа CA — в
// каталоге типов (AD-40); здесь — то, что видит Аудитор ИБ в журнале CA.
type Action struct {
	Type catalog.Type
	// Name — название действия (словарь продукта).
	Name string
	// Before, After — было → стало по умолчанию; уточняются данными записи.
	Before, After string
}

// Actions — действия MVP: не меньше девяти (AD-28), в том числе действия
// исполнителя. Остальные критические типы каталога получают запись CA с
// названием из каталога (ActionFor).
var Actions = []Action{
	// Решения по изделию и несоответствию (FR-146, минимум 6).
	{Type: catalog.DecisionNonconformityConfirmed, Name: "Подтвердить несоответствие", Before: "признак не подтверждён", After: "несоответствие подтверждено"},
	{Type: catalog.DecisionSignalRejected, Name: "Отклонить признак", Before: "признак несоответствия", After: "признак отклонён — изделие не несоответствующее"},
	{Type: catalog.DecisionContainmentSet, Name: "Блок", Before: "без блока", After: "заблокировано"},
	{Type: catalog.DecisionItemIsolated, Name: "Блок: изоляция изделия", Before: "в потоке", After: "изолировано"},
	{Type: catalog.DecisionContainmentReleased, Name: "Снять блок / выпуск", Before: "заблокировано", After: "разрешено"},
	{Type: catalog.DecisionPresentationResolved, Name: "Снять блок / выпуск: решение на точке предъявления", Before: "предъявлено", After: "решение принято"},
	{Type: catalog.DecisionDispositionSet, Name: "Назначить переделку / решение по изделию", Before: "решения нет", After: "решение принято"},
	// Пересмотр решения на точке (Д-81) — исправление прежнего решения новой записью.
	{Type: catalog.DecisionPresentationReviewed, Name: "Исправить прежнее решение: пересмотр решения на точке предъявления", Before: "решение принято до новых данных", After: "решение пересмотрено"},
	// Исправление прежнего решения — только новой записью (corrects), любой тип.
	// Действия исполнителя (AD-28): пропуск проверки, работа при отказе в
	// допуске, ручная смена режима оборудования, отключение источника.
	{Type: catalog.OperatorCheckSkipped, Name: "Пропуск обязательной проверки исполнителем", Before: "проверка обязательна", After: "проверка пропущена"},
	{Type: catalog.OperatorOverridePerformed, Name: "Работа в обход автоматики (при отказе в допуске)", Before: "автоматика действует", After: "ручное вмешательство"},
	{Type: catalog.SecurityAdmissionDenied, Name: "Отказ в допуске к рабочему месту", Before: "попытка допуска", After: "допуск отклонён"},
	{Type: catalog.OperatorModeChanged, Name: "Ручная смена режима оборудования", Before: "режим по программе", After: "режим изменён вручную"},
	{Type: catalog.OpsSourceDisabled, Name: "Отключение источника", Before: "источник включён", After: "источник отключён"},
	{Type: catalog.SecurityIdempotencyConflict, Name: "Конфликт целостности при приёме", Before: "принято первое содержимое", After: "пришло другое содержимое"},
	// Защищённые данные: реакции движка с последствиями для изделия.
	{Type: catalog.DecisionContainmentApplied, Name: "Блок по правилу", Before: "без блока", After: "сдерживание применено правилом"},
}

// CorrectionName — название действия «исправить прежнее решение» (AD-2:
// исправление — новое событие с corrects_event).
const CorrectionName = "Исправить прежнее решение новой записью"

// ActionFor — действие для типа записи: из перечня MVP или по каталогу
// (любой тип с признаком критичности). ok = false — тип не критический.
func ActionFor(t catalog.Type) (Action, catalog.Info, bool) {
	info, found := catalog.Lookup(t)
	if !found || !info.Critical {
		return Action{}, info, false
	}
	if i := slices.IndexFunc(Actions, func(a Action) bool { return a.Type == t }); i >= 0 {
		return Actions[i], info, true
	}
	return Action{Type: t, Name: info.Title, Before: "—", After: info.Title}, info, true
}

// Critical — тип записи требует записи в журнале критических действий.
func Critical(t catalog.Type) bool {
	info, ok := catalog.Lookup(t)
	return ok && info.Critical
}

// label — текст значения оси или словаря статусов (одно перечисление на продукт).
func label(axis, code string) string {
	for _, d := range statuses.Axes {
		if d.Name != axis {
			continue
		}
		for _, v := range d.Values {
			if v.Code == code {
				return v.Label
			}
		}
	}
	return code
}

// refine уточняет «было → стало» по данным основной записи: уровень
// сдерживания, вид решения по изделию.
func refine(a Action, data json.RawMessage) (before, after string) {
	before, after = a.Before, a.After
	var d struct {
		Level       string `json:"level"`
		Disposition string `json:"disposition"`
		Outcome     string `json:"outcome"`
		NewMode     string `json:"new_mode"`
		Change      string `json:"change"`
		StepKey     string `json:"step_key"`
	}
	if len(data) == 0 || json.Unmarshal(data, &d) != nil {
		return before, after
	}
	switch a.Type {
	case catalog.DecisionContainmentSet, catalog.DecisionContainmentApplied:
		switch d.Level {
		case "":
		case "item_hold", "lot_hold":
			after = "заблокировано: " + label("containment", d.Level)
		default:
			after = "сдерживание: " + label("containment", d.Level)
		}
	case catalog.DecisionDispositionSet:
		if d.Disposition != "" {
			after = label("disposition", d.Disposition)
		}
	case catalog.DecisionPresentationResolved:
		if d.Outcome != "" {
			after = "решение на точке предъявления: " + d.Outcome
		}
	case catalog.DecisionPresentationReviewed:
		switch d.Outcome {
		case "revoked":
			after = "приёмка отозвана: изделие заблокировано, годность не подтверждена"
		case "upheld":
			after = "прежнее решение оставлено в силе"
		}
	case catalog.OperatorModeChanged:
		if d.NewMode != "" {
			after = "режим изменён вручную: " + d.NewMode
		} else if d.Change != "" {
			after = "режим изменён вручную: " + d.Change
		}
	case catalog.OperatorCheckSkipped:
		if d.StepKey != "" {
			before = "проверка обязательна: " + d.StepKey
		}
	}
	return strings.TrimSpace(before), strings.TrimSpace(after)
}
