package erp

import (
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Module — модуль-эмитент семейства erp (AD-40).
const Module kernel.Module = "erp"

// Action — учётное действие порта учёта (наш язык, AD-18). Значения — из
// перечисления erp.posting.requested.action контракта.
type Action string

// Учётные действия порта учёта.
const (
	// AcceptIntoWork — «принято в работу»: выдача годного в производство (ЗТ-1).
	AcceptIntoWork Action = "accept_into_work"
	// WarehouseTransfer — «смена склада»: передача изделия между цехами (FR-130).
	WarehouseTransfer Action = "warehouse_transfer"
	// ScrapRework — «перевод в брак (переделка)», в том числе ремонт.
	ScrapRework Action = "scrap_transfer_rework"
	// ScrapWriteoff — «перевод в брак (списание)».
	ScrapWriteoff Action = "scrap_transfer_writeoff"
	// ScrapReprocess — «перевод в брак (переработка)»: разборка сборки.
	ScrapReprocess Action = "scrap_transfer_reprocess"
	// ReturnToSupplier — «возврат поставщику» с основанием претензии.
	ReturnToSupplier Action = "return_to_supplier"
	// ReturnFromDefect — «возврат из брака в производство» после удачной
	// переделки или ремонта (решение Д-17).
	ReturnFromDefect Action = "return_from_defect"
	// Release — «выпуск» годного (ЗТ-6), с признаком after_rework и номером
	// разрешения на отклонение.
	Release Action = "release"
	// InspectionResult — «результат контроля» по подписанной закрывающей
	// точке: движения не означает (кейс §3.3, Е-76).
	InspectionResult Action = "inspection_result"
)

// Actions — все учётные действия в порядке контракта.
var Actions = []Action{AcceptIntoWork, WarehouseTransfer, ScrapRework, ScrapWriteoff, ScrapReprocess,
	ReturnToSupplier, Release, InspectionResult, ReturnFromDefect}

// Valid — действие из словаря порта учёта.
func (a Action) Valid() bool {
	for _, x := range Actions {
		if x == a {
			return true
		}
	}
	return false
}

// DefectTransfer — перевод в брак (любой вид).
func (a Action) DefectTransfer() bool {
	return a == ScrapRework || a == ScrapWriteoff || a == ScrapReprocess
}

// Movement — действие меняет учёт движения (всё, кроме результата контроля).
func (a Action) Movement() bool { return a.Valid() && a != InspectionResult }

// Title — название действия для людей (журнал обмена, страница stand-а).
func (a Action) Title() string {
	switch a {
	case AcceptIntoWork:
		return "Принято в работу"
	case WarehouseTransfer:
		return "Смена склада"
	case ScrapRework:
		return "Перевод в брак (переделка)"
	case ScrapWriteoff:
		return "Перевод в брак (списание)"
	case ScrapReprocess:
		return "Перевод в брак (переработка)"
	case ReturnToSupplier:
		return "Возврат поставщику"
	case ReturnFromDefect:
		return "Возврат из брака в производство"
	case Release:
		return "Выпуск"
	case InspectionResult:
		return "Результат контроля"
	}
	return string(a)
}

// Axis — значение оси «учёт в 1С» после подтверждения действия учётной
// системой (AD-30: ось меняет только квитанция). Результат контроля оси не
// меняет (ok = false). «Возврат из брака в производство» возвращает изделие
// в работу — «принято в работу» (своего значения в словаре оси нет,
// contracts/statuses.yaml).
func Axis(a Action) (ev.AxisErpAccounting, bool) {
	switch a {
	case AcceptIntoWork, ReturnFromDefect:
		return ev.AxisErpAccountingAcceptedIntoWork, true
	case WarehouseTransfer:
		return ev.AxisErpAccountingMoved, true
	case ScrapRework, ScrapWriteoff, ScrapReprocess:
		return ev.AxisErpAccountingTransferredToScrap, true
	case ReturnToSupplier:
		return ev.AxisErpAccountingReturnedToSupplier, true
	case Release:
		return ev.AxisErpAccountingReleased, true
	}
	return "", false
}

// Входы порта учёта и записи, по которым модуль erp формирует исходящие
// сообщения и ведёт свои проекции (AD-18, AD-45).
var (
	// Triggers — записи, из которых складываются учётные сообщения.
	Triggers = []catalog.Type{
		catalog.ItemItemRegistered, catalog.ErpLotReceived, catalog.GenealogyLotIssued,
		catalog.OperationMessageThrown, catalog.DecisionPresentationResolved, catalog.DecisionLotResolved,
		catalog.DecisionDispositionSet, catalog.DecisionDispositionVerified,
	}
	// Exchange — записи обмена своего семейства (запрос, ответ, карантин,
	// решения человека): по ним ведётся очередь и журнал обмена.
	Exchange = []catalog.Type{
		catalog.ErpPostingRequested, catalog.ErpPostingResponded, catalog.ErpPostingQuarantined,
		catalog.ErpPostingResendRequested, catalog.ErpPostingCompensationDecided,
	}
)

// Is — тип t из списка ts.
func Is(t catalog.Type, ts []catalog.Type) bool {
	for _, x := range ts {
		if x == t {
			return true
		}
	}
	return false
}
