package nonconformity

import (
	"ant/internal/contracts/statuses"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// Имена намерений модуля nonconformity (AD-30, AD-40): ось «сдерживание» и
// несоответствия меняет только nonconformity, другие модули выражают
// намерение его функциями.
const (
	// IntentContain — «сдерживание» (analysis: область риска инцидента).
	IntentContain = "contain"
	// IntentDraft — «черновик несоответствия» (quality: сигнал по карте
	// реакций; machinelogs: отклонение оборудования на выполнении изделия).
	IntentDraft = "draft_nonconformity"
)

// Containment — уровень сдерживания и его основание.
type Containment struct {
	Level  statuses.Containment
	Reason string
}

// Contain — функция-намерение nonconformity (AD-30, AD-40): поздний модуль
// (analysis — область риска инцидента) выражает намерение сдержать изделие;
// ось «сдерживание» меняет только nonconformity. Защитное действие при
// пересвёртке не снимается автоматически (AD-3, AD-27).
func Contain(from kernel.Module, c Containment, causes ...kernel.Record) kernel.Intent {
	return kernel.NewIntent(Module, from, IntentContain, c, causes...)
}

// DraftRequest — полезная нагрузка намерения «черновик несоответствия»:
// сгенерированный тип данных decision.nonconformity.drafted (контракт
// модуля nonconformity, contracts/events/decision). Тип лежит в пакете
// контрактов, поэтому его могут строить и модули, стоящие в композиции
// раньше nonconformity (quality, machinelogs), — им импорт
// domain/nonconformity запрещён порядком AD-1.
//
// Обязательно: SignalIds (хотя бы один сигнал). NcID можно оставить пустым —
// nonconformity присвоит детерминированный id по изделию и сигналам
// (DraftNCID). ReactionOutcome = isolate — дополнительно блок изделия
// правилом (decision.containment.applied, режим 1, FR-49).
type DraftRequest = ev.DecisionNonconformityDraftedV1

// Draft — функция-намерение nonconformity «черновик несоответствия» (FR-51,
// FR-50 режим 1, AD-40). Её вызывают модули, которым импорт nonconformity
// разрешён (analysis, стадия), а также сборка движка, подставляющая её в
// порты ранних модулей (как machinelogs.Registrar в стадии).
//
// Ранний модуль без порта строит то же намерение сам — только из пакетов
// kernel и contracts:
//
//	kernel.NewIntent("nonconformity", quality.Module, "draft_nonconformity",
//		ev.DecisionNonconformityDraftedV1{SignalIds: …, BasisKind: …, Severity: …}, причины…)
//
// Применяет намерение nonconformity.Apply: черновик появляется в состоянии
// несоответствий изделия, React пишет реакцию decision.nonconformity.drafted
// (слот nonconformity.draft / item:‹id› / nc_id). Повтор намерения с теми же
// сигналами ничего не меняет (тот же nc_id).
func Draft(from kernel.Module, d DraftRequest, causes ...kernel.Record) kernel.Intent {
	return kernel.NewIntent(Module, from, IntentDraft, d, causes...)
}

// draftFromRequest — данные черновика из полезной нагрузки намерения.
func draftFromRequest(p any) (DraftedData, bool) {
	switch v := p.(type) {
	case DraftedData:
		return v, true
	case *DraftedData:
		if v == nil {
			return DraftedData{}, false
		}
		return *v, true
	case DraftRequest:
		return draftedOf(v), true
	case *DraftRequest:
		if v == nil {
			return DraftedData{}, false
		}
		return draftedOf(*v), true
	}
	return DraftedData{}, false
}

func draftedOf(v DraftRequest) DraftedData {
	d := DraftedData{NCID: string(v.NcID)}
	for _, s := range v.SignalIds {
		d.SignalIDs = append(d.SignalIDs, string(s))
	}
	str := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	if v.DocumentID != nil {
		d.DocumentID = string(*v.DocumentID)
	}
	if v.BasisKind != nil {
		d.BasisKind = string(*v.BasisKind)
	}
	if v.Severity != nil {
		d.Severity = string(*v.Severity)
	}
	d.DefectTypeCode = str(v.DefectTypeCode)
	if v.ZoneID != nil {
		d.ZoneID = string(*v.ZoneID)
	}
	if v.StepKey != nil {
		d.StepKey = string(*v.StepKey)
	}
	if v.OperationRunID != nil {
		d.OperationRunID = string(*v.OperationRunID)
	}
	d.RequirementRef = str(v.RequirementRef)
	if v.ReactionOutcome != nil {
		d.ReactionOutcome = string(*v.ReactionOutcome)
	}
	d.ReactionMapRef = str(v.ReactionMapRef)
	d.ClosingPoint = str(v.ClosingPoint)
	if v.PresentationNo != nil {
		d.PresentationNo = *v.PresentationNo
	}
	for _, m := range v.MissingInformation {
		d.MissingInformation = append(d.MissingInformation, string(m))
	}
	return d
}
