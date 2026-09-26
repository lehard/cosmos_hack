package ingest

import (
	"slices"
	"time"
)

// SourceState — учёт source_seq одного источника (AD-7, FR-39, кейс §4.5):
// наибольший принятый номер, число принятых номеров и незакрытые разрывы.
// Учёт ведёт приём вместе с карантином; сигнал «потеря данных источника» —
// реакция планировщика после окна ожидания (DueLosses), досылка в окне его отменяет.
type SourceState struct {
	SourceID string
	// HighWater — наибольший принятый source_seq (0 — ещё ничего).
	HighWater int64
	// Received — сколько разных номеров принято.
	Received int64
	// Gaps — незакрытые разрывы в порядке номеров.
	Gaps []Gap
}

// Gap — разрыв номеров [From, To], открытый приёмом в OpenedAt.
type Gap struct {
	From, To int64
	OpenedAt time.Time
	// Reported — по разрыву уже записан ingest.source.loss_suspected.
	Reported bool
}

// SeqKind — что показал номер нового сообщения.
type SeqKind string

// Виды наблюдения номера.
const (
	// SeqNone — у сообщения нет source_seq (ручной ввод без терминала и т. п.).
	SeqNone SeqKind = "none"
	// SeqInOrder — следующий по порядку номер.
	SeqInOrder SeqKind = "in_order"
	// SeqGapOpened — номер с пропуском: открыт разрыв.
	SeqGapOpened SeqKind = "gap_opened"
	// SeqGapFilled — досылка: номер из открытого разрыва (позднее событие).
	SeqGapFilled SeqKind = "gap_filled"
	// SeqViolation — номер уже был у другого event_id или ниже учтённого без
	// разрыва: противоречие последовательности — флаг (FR-33, AD-5).
	SeqViolation SeqKind = "violation"
)

// SeqObservation — результат учёта номера.
type SeqObservation struct {
	Kind SeqKind
	// Gap — открытый разрыв (для SeqGapOpened).
	Gap Gap
}

// maxGaps — предел числа незакрытых разрывов у источника: дальше соседние
// разрывы сливаются (номера не теряются, только укрупняется учёт).
const maxGaps = 1024

// ObserveSeq учитывает номер seq нового (не повторного) сообщения источника,
// принятого в receivedAt. Чистая функция: state не меняется, возвращается новое.
func ObserveSeq(state SourceState, seq int64, receivedAt time.Time) (SourceState, SeqObservation) {
	if seq <= 0 {
		return state, SeqObservation{Kind: SeqNone}
	}
	st := state
	st.Gaps = slices.Clone(state.Gaps)
	switch {
	case seq == st.HighWater+1:
		st.HighWater = seq
		st.Received++
		return st, SeqObservation{Kind: SeqInOrder}
	case seq > st.HighWater+1:
		g := Gap{From: st.HighWater + 1, To: seq - 1, OpenedAt: receivedAt}
		st.Gaps = append(st.Gaps, g)
		if len(st.Gaps) > maxGaps {
			st.Gaps[1].From = st.Gaps[0].From
			st.Gaps = st.Gaps[1:]
		}
		st.HighWater = seq
		st.Received++
		return st, SeqObservation{Kind: SeqGapOpened, Gap: g}
	}
	for i, g := range st.Gaps {
		if seq < g.From || seq > g.To {
			continue
		}
		// FR-39: досылка закрывает часть разрыва.
		var repl []Gap
		if g.From < seq {
			repl = append(repl, Gap{From: g.From, To: seq - 1, OpenedAt: g.OpenedAt, Reported: g.Reported})
		}
		if seq < g.To {
			repl = append(repl, Gap{From: seq + 1, To: g.To, OpenedAt: g.OpenedAt, Reported: g.Reported})
		}
		st.Gaps = slices.Replace(st.Gaps, i, i+1, repl...)
		st.Received++
		return st, SeqObservation{Kind: SeqGapFilled}
	}
	return state, SeqObservation{Kind: SeqViolation}
}

// Missing — сколько номеров сейчас не хватает (сумма разрывов).
func (s SourceState) Missing() int64 {
	var n int64
	for _, g := range s.Gaps {
		n += g.To - g.From + 1
	}
	return n
}

// CompletenessBP — полнота источника в базисных пунктах (FR-41): принятые
// номера против ожидаемых 1…HighWater; 10000 — без пропусков.
func (s SourceState) CompletenessBP() int64 {
	if s.HighWater == 0 {
		return 10000
	}
	return (s.HighWater - s.Missing()) * 10000 / s.HighWater
}

// DueLosses — разрывы, не закрытые досылкой за окно ожидания window к моменту
// now и ещё не заявленные (AD-7): по каждому планировщик пишет
// ingest.source.loss_suspected. Возвращает новое состояние с пометкой Reported.
func DueLosses(state SourceState, now time.Time, window time.Duration) (SourceState, []Gap) {
	st := state
	st.Gaps = slices.Clone(state.Gaps)
	var due []Gap
	for i, g := range st.Gaps {
		if g.Reported || now.Before(g.OpenedAt.Add(window)) {
			continue
		}
		st.Gaps[i].Reported = true
		due = append(due, st.Gaps[i])
	}
	return st, due
}
