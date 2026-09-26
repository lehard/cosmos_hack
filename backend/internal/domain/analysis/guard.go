package analysis

import (
	"strings"

	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// Гарды операций analysis над состоянием инцидента на basis_seq (AD-39):
// чистые функции; их вызывают api до записи (над проекцией как кэшем входа),
// стадия при применении (Stage не применяет решение без основания) и
// верификатор.

// GuardOpen — инцидент не закрыт.
func GuardOpen(v IncidentRecord) error {
	if v.Closed {
		return kernel.Refuse(errcodes.IncidentClosed, "incident_id", v.IncidentID)
	}
	return nil
}

// GuardNarrow — сужение области (FR-61, AD-27: разрешающее действие): только
// с основанием (доказательства и текст), только изделия текущей области.
func GuardNarrow(v IncidentRecord, items, evidence []string, reason string) error {
	if err := GuardOpen(v); err != nil {
		return err
	}
	if len(evidence) == 0 || strings.TrimSpace(reason) == "" {
		return kernel.Refuse(errcodes.IncidentBasisRequired)
	}
	for _, it := range items {
		if !v.InScope(it) {
			return kernel.Refuse(errcodes.IncidentItemNotInScope, "item_id", it, "incident_id", v.IncidentID)
		}
	}
	return nil
}

// GuardExpand — расширение области человеком (защитное): с основанием, хотя
// бы одно изделие ещё не в области.
func GuardExpand(v IncidentRecord, items []string, reason string) error {
	if err := GuardOpen(v); err != nil {
		return err
	}
	if strings.TrimSpace(reason) == "" {
		return kernel.Refuse(errcodes.IncidentBasisRequired)
	}
	for _, it := range items {
		if !v.InScope(it) {
			return nil
		}
	}
	r := kernel.Refuse(errcodes.ApiValidationFailed, "field", "item_ids", "reason", "все изделия уже в области риска")
	r.Detail = "Все изделия уже в области риска инцидента " + v.IncidentID
	return r
}

// GuardAssess — оценка изделия в инциденте: изделие в области, есть доказательства.
func GuardAssess(v IncidentRecord, item string, evidence []string) error {
	if err := GuardOpen(v); err != nil {
		return err
	}
	if len(evidence) == 0 {
		return kernel.Refuse(errcodes.IncidentBasisRequired)
	}
	if !v.InScope(item) {
		return kernel.Refuse(errcodes.IncidentItemNotInScope, "item_id", item, "incident_id", v.IncidentID)
	}
	return nil
}

// GuardConclude — вывод о причине (FR-59, AD-27: необратимое — только
// уполномоченный человек): «чем проверено» обязательно; ошибка исполнителя —
// только после письменного объяснения работника (incident.operator_error.confirmed).
func GuardConclude(v IncidentRecord, conclusion, category, verification string) error {
	if err := GuardOpen(v); err != nil {
		return err
	}
	if strings.TrimSpace(verification) == "" {
		r := kernel.Refuse(errcodes.ApiValidationFailed, "field", "verification", "reason", "нужно указать, чем проверено")
		return r
	}
	if conclusion == "confirmed" && category == "" {
		return kernel.Refuse(errcodes.ApiValidationFailed, "field", "category", "reason", "у подтверждённой причины нужна категория")
	}
	if conclusion == "confirmed" && category == CatPerformer && !v.OperatorError {
		return kernel.Refuse(errcodes.IncidentExplanationRequired)
	}
	return nil
}
