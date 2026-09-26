package item

// ItemLookup — результат поиска изделия по носителю (AD-16, AD-41).
type ItemLookup struct {
	ItemID     string `json:"item_id" doc:"Внутренний ID изделия: код_предприятия:локальный_id (AD-16)."`
	CarrierRef string `json:"carrier_ref,omitempty" doc:"Носитель, по которому найдено: ant:carrier:‹тип›:‹значение›."`
}
