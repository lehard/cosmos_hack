package ingest

import (
	"hash/fnv"
	"regexp"
)

// ItemRef — ссылка источника на изделие через носитель (AD-16, AD-41).
type ItemRef struct {
	CarrierType         string
	Value               string
	IdentificationLevel string
}

// Binding — привязка события к изделию на приёме (AD-41, FR-34): внутренний
// item_id, значение носителя, основание и надёжность привязки.
type Binding struct {
	ItemID string
	// CarrierRef — `‹тип›:‹значение›` из item_ref (поле carrier_ref записи журнала).
	CarrierRef string
	// Basis — основание: internal_id | carrier.
	Basis string
	// Reliability — надёжность привязки: high | medium | low | unknown.
	Reliability string
	// NeedsLookup — носитель нужно разрешить по реестру носителей на occurred_at
	// (crossitem.ResolveCarrier); неоднозначное — в поток межизделийной стадии.
	NeedsLookup bool
	// Unbound — событие без изделия (оборудование, партия, задание).
	Unbound bool
}

var itemIDRe = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{0,15}:[A-Za-z0-9][A-Za-z0-9._/-]{0,95}$`)

// BindingReliability — надёжность привязки по уровню идентификации (кейс §4.6).
func BindingReliability(level string) string {
	switch level {
	case "unique":
		return "high"
	case "probable":
		return "medium"
	case "ambiguous":
		return "low"
	}
	return "unknown"
}

// Bind — простой случай разрешения носителя до выбора партиции (AD-41, FR-34):
// внутренний item_id в конверте или носитель вида internal_id — изделие сразу;
// иной носитель — нужен реестр носителей; ничего — событие без изделия.
// Внутренний ID из метки не выводится (AD-16): значение DataMatrix или QR
// остаётся носителем, а не item_id.
func Bind(itemID string, ref *ItemRef) Binding {
	var b Binding
	if ref != nil {
		b.CarrierRef = ref.CarrierType + ":" + ref.Value
	}
	switch {
	case itemID != "":
		b.ItemID, b.Basis, b.Reliability = itemID, "internal_id", "high"
		if ref != nil {
			b.Reliability = BindingReliability(ref.IdentificationLevel)
			if ref.CarrierType == "internal_id" && ref.Value == itemID {
				b.Reliability = "high"
			}
		}
	case ref != nil && ref.CarrierType == "internal_id" && itemIDRe.MatchString(ref.Value):
		b.ItemID, b.Basis, b.Reliability = ref.Value, "internal_id", "high"
	case ref != nil:
		b.Basis, b.Reliability, b.NeedsLookup = "carrier", BindingReliability(ref.IdentificationLevel), true
	default:
		b.Unbound = true
	}
	return b
}

// Partition — партиция изделия: hash(item_id) mod p (AD-6, AD-41), FNV-1a 32.
// Партиция — только по внутреннему ID.
func Partition(itemID string, p int) int {
	if p <= 0 {
		return 0
	}
	h := fnv.New32a()
	h.Write([]byte(itemID))
	return int(h.Sum32() % uint32(p))
}
