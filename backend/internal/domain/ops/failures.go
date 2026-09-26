package ops

import (
	"cmp"
	"maps"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
)

// MaxErrorLen — предел текста ошибки в ops.processing.failed (схема: maxLength 4000).
const MaxErrorLen = 4000

// FailureEventID — event_id записи ops.processing.failed (AD-45):
// UUIDv5(NS_ANT, тип ‖ потребитель ‖ изделие ‖ seq). Повтор той же ошибки после
// сбоя процесса даёт тот же id — второго смысла в журнале не появляется (AD-7).
func FailureEventID(consumer, itemID string, seq int64) string {
	return kernel.UUIDv5(constants.NsAnt, string(catalog.OpsProcessingFailed)+"\x1f"+consumer+"\x1f"+itemID+"\x1f"+strconv.FormatInt(seq, 10))
}

// RetryEventID — event_id записи ops.processing.retried, которую пишет не
// человек, а `ant rebuild --item` (одна на сбой).
func RetryEventID(failureEventID string) string {
	return kernel.UUIDv5(constants.NsAnt, string(catalog.OpsProcessingRetried)+"\x1f"+failureEventID)
}

// FailureMessage — текст ошибки для журнала: не длиннее MaxErrorLen байт и
// без разрезанного посередине символа UTF-8.
func FailureMessage(msg string) string {
	if len(msg) <= MaxErrorLen {
		return msg
	}
	cut := msg[:MaxErrorLen]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}

// Failure — запись ops.processing.failed: потребитель не смог обработать
// запись изделия (AD-45).
type Failure struct {
	EventID  string
	Seq      int64
	ItemID   string
	Consumer string
	// FailedSeq — seq записи, на которой случилась ошибка.
	FailedSeq int64
	Error     string
	At        time.Time
}

// Retry — запись ops.processing.retried: повтор обработки после сбоя
// FailureEventID (решение администратора или `ant rebuild --item`).
type Retry struct {
	EventID        string
	Seq            int64
	ItemID         string
	FailureEventID string
}

// Stopped — изделие «обработка остановлена» и его последний сбой.
type Stopped struct {
	Failure Failure
	// Retries — сколько раз обработку изделия уже повторяли.
	Retries int
}

// StoppedItems — изделия, у которых последняя запись о сбое не снята
// повтором (AD-45). Повтор снимает сбой, на который он ссылается, и все более
// ранние сбои изделия: воркер после повтора сворачивает изделие целиком.
// Порядок — по seq сбоя (старые сверху), ответ детерминирован.
func StoppedItems(failures []Failure, retries []Retry) []Stopped {
	type state struct {
		last    *Failure
		cleared int64 // seq сбоя, до которого включительно всё снято повтором
		retries int
	}
	fs := slices.Clone(failures)
	slices.SortFunc(fs, func(a, b Failure) int { return cmp.Compare(a.Seq, b.Seq) })
	bySeq := map[string]int64{}
	items := map[string]*state{}
	for i := range fs {
		f := &fs[i]
		bySeq[f.EventID] = f.Seq
		st := items[f.ItemID]
		if st == nil {
			st = &state{}
			items[f.ItemID] = st
		}
		st.last = f
	}
	for _, r := range retries {
		st := items[r.ItemID]
		if st == nil {
			continue
		}
		st.retries++
		// Повтор без известного сбоя (ссылка на чужой id) снимает всё до себя.
		upTo, ok := bySeq[r.FailureEventID]
		if !ok || upTo > r.Seq {
			upTo = r.Seq
		}
		st.cleared = max(st.cleared, upTo)
	}
	var out []Stopped
	for _, id := range slices.Sorted(maps.Keys(items)) {
		st := items[id]
		if st.last == nil || st.last.Seq <= st.cleared {
			continue
		}
		out = append(out, Stopped{Failure: *st.last, Retries: st.retries})
	}
	slices.SortFunc(out, func(a, b Stopped) int { return cmp.Compare(a.Failure.Seq, b.Failure.Seq) })
	return out
}
