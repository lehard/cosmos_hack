package world

import (
	"strings"

	journalapp "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/fixtures/loader"
)

// Окно записи журнала на заготовках (интерфейс 6, journal.event.read):
// запись мира по event_id — текст для людей, источник словами, позиция в
// журнале, материалы наблюдения и числа режима. Ответ записи появляется на
// шаге, где запись стала известна (как journal.entry.read).

// eventView — запись мира в окне записи.
func (c *Ctx) eventView(e *Event) journalapp.JournalEventView {
	v := journalapp.JournalEventView{EventID: e.ID, EventType: e.Type, OccurredAt: e.Occurred.UTC(), RecordedAt: e.Recorded.UTC(), Late: e.Late,
		Text: optStr(e.Summary), SourceLabel: optStr(c.M.sourceLabel(e)), Kind: e.Kind, JournalSeq: e.Seq, Stream: e.Stream,
		Signers: []string{}, EvidenceRefs: []string{}}
	if info, ok := catalog.Lookup(catalog.Type(e.Type)); ok {
		v.EventTitle = info.Title
	}
	if x := e.Params["outcome"]; x != "" {
		v.Variant = ptr(x)
	}
	if len(e.Params) > 0 {
		v.Params = e.Params
	}
	if sk := sourceKindOf(e); sk != "" {
		v.SourceKind = ptr(sk)
	}
	if e.Item != nil {
		v.ItemID = ptr(FullID(e.Item.ID))
	}
	if e.Author != "" {
		v.Author = ptr(e.Author)
		v.Signers = []string{strings.ToLower(e.Author) + "@1"}
	}
	if refs := e.evidenceRefs(); len(refs) > 0 {
		v.EvidenceRefs = refs
	}
	if r := e.Reading; r != nil {
		v.Reading = &journalapp.JournalEventReading{Parameter: r.Parameter, Unit: r.Unit, Scale: r.Scale, SetpointNominal: r.SetpointNominal,
			SetpointMin: r.SetpointMin, SetpointMax: r.SetpointMax, ObservedMin: r.ObservedMin, ObservedMax: r.ObservedMax}
	}
	return v
}

// renderEvents — записи, ставшие известными на шаге (journal.event.read).
func renderEvents(c *Ctx) []loader.Response {
	var out []loader.Response
	for _, e := range c.M.Events {
		if e.Step == c.N {
			out = append(out, resp("journal.event.read", c.eventView(e), "event_id", e.ID))
		}
	}
	return out
}
