package ops

import (
	"cmp"
	"maps"
	"slices"
	"strconv"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
)

// Состояния интеграции на столе администратора (FR-127, AD-18).
const (
	IntegrationOK       = "ok"
	IntegrationDegraded = "degraded"
	// IntegrationDisabled — система не включена (integrations.enabled) или
	// канал выключен модулем-владельцем.
	IntegrationDisabled = "disabled"
)

// Systems — внешние системы, которые стол администратора показывает всегда:
// учёт (1С), Галактика ERP и MES (кейс §3.2, FR-127). Прочие (КОМПАС, СКУД,
// УЦ, партнёр) — если включены или о них есть записи.
var Systems = []string{"onec", "galaktika", "mes"}

// IntegrationRecord — последняя запись ops.integration.degraded системы.
type IntegrationRecord struct {
	Seq    int64
	System string
	State  string
	Detail string
	At     time.Time
}

// Channel — живое состояние канала обмена от модуля-владельца интеграции
// (erp — 1С и Галактика, mes — MES): последняя сверка ответной стороны.
type Channel struct {
	System    string
	State     string
	Detail    string
	CheckedAt time.Time
}

// Integration — состояние интеграции для экрана.
type Integration struct {
	System string
	State  string
	Detail string
	// Since — с какого момента состояние (запись журнала или сверка канала).
	Since *time.Time
}

// ShouldRecord — писать ли ops.integration.degraded при наблюдённом state
// (AD-18): только переходы. Первое наблюдение «ok» не пишется — журнал хранит
// сбои и выход из них, а не каждую сверку; «disabled» — не состояние канала
// в контракте записи.
func ShouldRecord(last *IntegrationRecord, state string) bool {
	if state != IntegrationOK && state != IntegrationDegraded {
		return false
	}
	if last == nil {
		return state == IntegrationDegraded
	}
	return last.State != state
}

// DegradedEventID — event_id записи о переходе канала: UUIDv5 от системы,
// нового состояния и seq предыдущей записи системы (0 — записей не было).
// Две копии, увидевшие один переход, пишут одну запись (AD-7).
func DegradedEventID(system, state string, afterSeq int64) string {
	return kernel.UUIDv5(constants.NsAnt, string(catalog.OpsIntegrationDegraded)+"\x1f"+system+"\x1f"+state+"\x1f"+strconv.FormatInt(afterSeq, 10))
}

// Integrations сводит состояние интеграций (FR-127): живое состояние канала
// от модуля-владельца сильнее записи журнала; без канала — последняя запись
// ops.integration.degraded; без того и другого — «ok», если система включена,
// иначе «disabled». Порядок: Systems, затем прочие по имени.
func Integrations(enabled []string, channels []Channel, records []IntegrationRecord) []Integration {
	ch := map[string]Channel{}
	for _, c := range channels {
		ch[c.System] = c
	}
	rec := map[string]IntegrationRecord{}
	for _, r := range records {
		if p, ok := rec[r.System]; !ok || r.Seq > p.Seq {
			rec[r.System] = r
		}
	}
	on := map[string]bool{}
	for _, s := range enabled {
		on[s] = true
	}
	seen := map[string]bool{}
	order := slices.Clone(Systems)
	for _, s := range Systems {
		seen[s] = true
	}
	var extra []string
	for _, m := range []map[string]bool{on, keys(ch), keys(rec)} {
		for _, s := range slices.Sorted(maps.Keys(m)) {
			if !seen[s] {
				seen[s] = true
				extra = append(extra, s)
			}
		}
	}
	slices.SortFunc(extra, cmp.Compare[string])
	order = append(order, extra...)

	out := make([]Integration, 0, len(order))
	for _, s := range order {
		it := Integration{System: s}
		r, hasRec := rec[s]
		switch c, ok := ch[s]; {
		case ok:
			it.State, it.Detail = c.State, c.Detail
			since := c.CheckedAt
			if hasRec && r.State == c.State {
				since = r.At
			}
			if !since.IsZero() {
				it.Since = &since
			}
		case hasRec && on[s]:
			it.State, it.Detail = r.State, r.Detail
			at := r.At
			it.Since = &at
		case on[s]:
			it.State, it.Detail = IntegrationOK, "канал включён; сбоев сверки не записано"
		default:
			it.State, it.Detail = IntegrationDisabled, "не включена (integrations.enabled)"
		}
		out = append(out, it)
	}
	return out
}

func keys[V any](m map[string]V) map[string]bool {
	out := make(map[string]bool, len(m))
	for _, k := range slices.Sorted(maps.Keys(m)) {
		out[k] = true
	}
	return out
}
