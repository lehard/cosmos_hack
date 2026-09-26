package reference

import (
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "reference"

// Виды изменений справочника (reference.change.affects_item.reference_kind).
const (
	KindItemType  = "item_type"
	KindEquipment = "equipment"
	KindLocation  = "location"
)

// AffectsItemData — data reference.change.affects_item v1 (AD-31, AD-42).
type AffectsItemData struct {
	ReferenceEventID string `json:"reference_event_id"`
	ReferenceKind    string `json:"reference_kind"`
	ValidFrom        string `json:"valid_from"`
}

// AffectsItem — функция-намерение модуля reference «изменение справочника с
// действием в прошлом затрагивает изделие» (AD-31, AD-42): строит
// адресованную запись reference.change.affects_item в поток изделия.
// Затронутые изделия находит детерминированным запросом межизделийная стадия
// (domain/crossitem): reference стоит в порядке импорта раньше и генеалогии
// не видит, поэтому стадия вызывает эту функцию владельца типа — как порт
// machinelogs.Registrar вызывает функцию nonconformity. Ключ адресованной
// записи — `‹id записи справочника›|‹изделие›`: повтор даёт тот же event_id.
func AffectsItem(itemID string, ref kernel.Record, kind string, validFrom time.Time) (kernel.Addressed, error) {
	d := AffectsItemData{ReferenceEventID: ref.EventID, ReferenceKind: kind, ValidFrom: validFrom.UTC().Format("2006-01-02T15:04:05.000Z")}
	return kernel.NewAddressed(Module, catalog.ReferenceChangeAffectsItem, "item:"+itemID, ref.EventID+"|"+itemID, d, ref)
}
