package quality

import (
	"slices"
	"strings"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
)

// PassportsFrom — паспорта допуска анализатора из записей их потоков
// (analyzer.passport.admitted | suspended | reinstated | retired; эмитент —
// vision, AD-29). Чистая функция для BundleSource движка (эпик 17): записи
// паспортов лежат вне потока изделия, поэтому попадают в свёртку изделия
// через Env. Порядок — occurred_at → received_at → event_id (AD-5); откат
// действует с момента приостановки и не переписывает прежние наблюдения.
// Нечитаемые записи пропускаются: без паспорта уровень доверия 0 — строже.
func PassportsFrom(records []kernel.Record) []Passport {
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
	out := []Passport{}
	find := func(id string) *Passport {
		for i := range out {
			if out[i].PassportID == id {
				return &out[i]
			}
		}
		return nil
	}
	status := func(id string, st PassportStatus) {
		if p := find(id); p != nil {
			p.History = append(p.History, st)
		}
	}
	for _, r := range in {
		switch r.Type {
		case catalog.AnalyzerPassportAdmitted:
			d, err := kernel.Decode[ev.AnalyzerPassportAdmittedV1](r)
			if err != nil {
				continue
			}
			p := find(string(d.PassportID))
			if p == nil {
				out = append(out, Passport{PassportID: string(d.PassportID)})
				p = &out[len(out)-1]
			}
			p.RecipeRef, p.AnalyzerVersion = d.RecipeRef, d.Versions.AnalyzerVersion
			p.Stage, p.TrustLevel = string(d.Stage), d.TrustLevel
			p.History = append(p.History, PassportStatus{At: r.OccurredAt, Active: true, EventID: r.EventID, Note: "admitted"})
		case catalog.AnalyzerPassportSuspended:
			d, err := kernel.Decode[ev.AnalyzerPassportSuspendedV1](r)
			if err != nil {
				continue
			}
			status(string(d.PassportID), PassportStatus{At: r.OccurredAt, EventID: r.EventID, Note: "suspended:" + string(d.Trigger)})
		case catalog.AnalyzerPassportReinstated:
			d, err := kernel.Decode[ev.AnalyzerPassportReinstatedV1](r)
			if err != nil {
				continue
			}
			status(string(d.PassportID), PassportStatus{At: r.OccurredAt, Active: true, EventID: r.EventID, Note: "reinstated"})
		case catalog.AnalyzerPassportRetired:
			d, err := kernel.Decode[ev.AnalyzerPassportRetiredV1](r)
			if err != nil {
				continue
			}
			status(string(d.PassportID), PassportStatus{At: r.OccurredAt, EventID: r.EventID, Note: "retired"})
		}
	}
	slices.SortFunc(out, func(a, b Passport) int { return strings.Compare(a.PassportID, b.PassportID) })
	return out
}
