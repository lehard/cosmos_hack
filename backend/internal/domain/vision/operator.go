package vision

import (
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// OperatorVision («Контроль действий оператора», FR-126): результат — гипотеза
// о действии, а не факт вины; распознавания лиц нет. Кто стоял у рабочего
// места, система знает только по входу на рабочее место (допуск access,
// AD-15): гипотеза относится к сеансу рабочего места на её время.

// WorkplaceSession — сеанс рабочего места по записям access.workplace.*.
type WorkplaceSession struct {
	WorkplaceID string    `json:"workplace_id"`
	SessionID   string    `json:"workplace_session_id"`
	PersonID    string    `json:"person_id"`
	From        time.Time `json:"from"`
	// To — конец сеанса (освобождение или отзыв допуска); нулевое — открыт.
	To time.Time `json:"to,omitzero"`
}

// Sessions — сеансы рабочих мест из записей access.workplace.admitted |
// released | revoked (порядок — occurred_at, AD-5).
func Sessions(records []kernel.Record) []WorkplaceSession {
	in := slices.Clone(records)
	slices.SortStableFunc(in, func(a, b kernel.Record) int {
		switch {
		case kernel.Less(a, b):
			return -1
		case kernel.Less(b, a):
			return 1
		}
		return 0
	})
	out := []WorkplaceSession{}
	end := func(id string, at time.Time) {
		for i := range out {
			if out[i].SessionID == id && out[i].To.IsZero() {
				out[i].To = at
			}
		}
	}
	for _, r := range in {
		switch r.Type {
		case catalog.AccessWorkplaceAdmitted:
			if d, err := kernel.Decode[ev.AccessWorkplaceAdmittedV1](r); err == nil {
				out = append(out, WorkplaceSession{WorkplaceID: string(d.WorkplaceID), SessionID: string(d.WorkplaceSessionID),
					PersonID: string(d.PersonID), From: r.OccurredAt})
			}
		case catalog.AccessWorkplaceReleased:
			if d, err := kernel.Decode[ev.AccessWorkplaceReleasedV1](r); err == nil {
				end(string(d.WorkplaceSessionID), r.OccurredAt)
			}
		case catalog.AccessWorkplaceRevoked:
			if d, err := kernel.Decode[ev.AccessWorkplaceRevokedV1](r); err == nil {
				end(string(d.WorkplaceSessionID), r.OccurredAt)
			}
		}
	}
	return out
}

// ExecutorAt — исполнитель, к которому относится гипотеза OperatorVision:
// сеанс рабочего места, открытый на момент at. Нет сеанса или их несколько —
// исполнитель неизвестен (ok = false): система не угадывает человека.
func ExecutorAt(sessions []WorkplaceSession, workplaceID string, at time.Time) (WorkplaceSession, bool) {
	var found []WorkplaceSession
	for _, s := range sessions {
		if s.WorkplaceID != workplaceID || at.Before(s.From) || (!s.To.IsZero() && !at.Before(s.To)) {
			continue
		}
		found = append(found, s)
	}
	if len(found) != 1 {
		return WorkplaceSession{}, false
	}
	return found[0], true
}
