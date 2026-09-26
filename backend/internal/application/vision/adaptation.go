package vision

import (
	"context"
	"slices"
	"strings"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/vision"
)

// Чтение контура адаптации VisionQC (эпик 40): пропуски брака со списком на
// перепроверку (FR-100), «какими версиями и почему» по наблюдению (FR-98),
// размеченные примеры из подтверждённых решений (FR-99), контроль дрейфа.
// Всё — свёртки записей журнала на момент (AD-22) в пределах прогона (AD-38).

// sameRun — запись относится к прогону m: живая работа — записи без run_id.
func sameRun(m platform.Moment, r kernel.Record) bool { return r.RunID == m.RunID }

// records — записи типа t на момент m в прогоне m (декодированные).
func (s *Service) records(ctx context.Context, t catalog.Type, m platform.Moment, run bool) ([]engineapp.Decoded, error) {
	q := appjournal.ReadQuery{EventType: string(t), Limit: 1000}
	var out []engineapp.Decoded
	for {
		page, err := s.d.Journal.Read(ctx, q)
		if err != nil {
			return nil, err
		}
		for _, e := range page {
			d, err := s.d.Codec.Decode(ctx, e)
			if err != nil {
				continue
			}
			if within(m, d.Record) && (!run || sameRun(m, d.Record)) {
				out = append(out, d)
			}
		}
		if len(page) < q.Limit {
			return out, nil
		}
		q.AfterSeq = int64(page[len(page)-1].Seq)
	}
}

// seen — наблюдения анализатора на момент m в прогоне m по event_id.
func (s *Service) seen(ctx context.Context, m platform.Moment) (map[string]dom.Seen, []dom.Seen, error) {
	ds, err := s.records(ctx, catalog.InspectionResultRecorded, m, true)
	if err != nil {
		return nil, nil, err
	}
	by := map[string]dom.Seen{}
	all := []dom.Seen{}
	for _, d := range ds {
		if o, ok := dom.SeenFrom(d.Record); ok {
			by[o.EventID] = o
			all = append(all, o)
		}
	}
	return by, all, nil
}

func strp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Escapes — пропуски брака с ранними наблюдениями и списком изделий на
// перепроверку (vision.escape.list, FR-100).
func (s *Service) Escapes(ctx context.Context, m platform.Moment) (AdaptationEscapeList, error) {
	if !s.live() {
		return s.Unimplemented.Escapes(ctx, m)
	}
	es, err := s.records(ctx, catalog.QualityEscapeRecorded, m, true)
	if err != nil {
		return AdaptationEscapeList{}, err
	}
	by, all, err := s.seen(ctx, m)
	if err != nil {
		return AdaptationEscapeList{}, err
	}
	// Последняя версия слота на дефект (пересвёртка пишет новую версию, AD-3).
	latest := map[string]engineapp.Decoded{}
	for _, d := range es {
		x, err := kernel.Decode[ev.QualityEscapeRecordedV1](d.Record)
		if err != nil {
			continue
		}
		k := d.Record.ItemID + "/" + string(x.DefectID)
		if p, ok := latest[k]; !ok || p.Record.Seq < d.Record.Seq {
			latest[k] = d
		}
	}
	out := AdaptationEscapeList{Items: []AdaptationEscape{}}
	keys := make([]string, 0, len(latest))
	for k := range latest {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		d := latest[k]
		x, _ := kernel.Decode[ev.QualityEscapeRecordedV1](d.Record)
		v := AdaptationEscape{EventID: d.Record.EventID, DefectID: string(x.DefectID), ItemID: d.Record.ItemID, MethodCoversDefect: x.MethodCoversDefect,
			RecordedAt: d.Record.OccurredAt.UTC(), AnalyzerVersion: x.AnalyzerVersion, Missed: []MissedObservation{}, Recheck: []RecheckItem{}}
		missed := []dom.Seen{}
		for _, id := range x.MissedObservationEventIds {
			o, ok := by[string(id)]
			if !ok {
				v.Missed = append(v.Missed, MissedObservation{EventID: string(id), Versions: map[string]string{}})
				continue
			}
			missed = append(missed, o)
			v.Missed = append(v.Missed, MissedObservation{EventID: o.EventID, ItemID: o.ItemID, OccurredAt: o.OccurredAt.UTC(),
				Point: strp(o.Point), Versions: o.Versions.Map()})
		}
		if x.MethodCoversDefect {
			for _, rc := range dom.RecheckList(missed, all, d.Record.ItemID) {
				v.Recheck = append(v.Recheck, RecheckItem{ItemID: rc.ItemID, ObservationEventIDs: rc.ObservationIDs, LastAt: rc.LastAt.UTC()})
			}
		}
		out.Items = append(out.Items, v)
	}
	slices.SortStableFunc(out.Items, func(a, b AdaptationEscape) int { return b.RecordedAt.Compare(a.RecordedAt) })
	return out, nil
}

// Observation — «какими версиями и почему» по наблюдению анализатора
// (vision.observation.read, FR-98): паспорт — на момент наблюдения в его прогоне.
func (s *Service) Observation(ctx context.Context, eventID string, m platform.Moment) (ObservationAccount, error) {
	if !s.live() {
		return s.Unimplemented.Observation(ctx, eventID, m)
	}
	ds, err := s.records(ctx, catalog.InspectionResultRecorded, m, false)
	if err != nil {
		return ObservationAccount{}, err
	}
	var o dom.Seen
	found := false
	for _, d := range ds {
		if strings.EqualFold(d.Record.EventID, eventID) {
			o, found = dom.SeenFrom(d.Record)
			break
		}
	}
	if !found {
		return ObservationAccount{}, platform.Fail(errcodes.ApiNotFound, "object", "observation", "id", eventID)
	}
	reg, err := s.registry(ctx, platform.Moment{Axis: m.Axis, AsOf: m.AsOf, RunID: o.RunID})
	if err != nil {
		return ObservationAccount{}, err
	}
	a := dom.Explain(o, reg)
	v := ObservationAccount{EventID: o.EventID, ItemID: strp(o.ItemID), OccurredAt: o.OccurredAt.UTC(), Point: strp(o.Point), Outcome: o.Outcome,
		QualityBP: o.QualityBP, ConfidenceBP: o.ConfidenceBP, Versions: o.Versions.Map(), MissingVersions: a.Missing, PassportID: strp(a.PassportID),
		StatusThen: a.StatusThen, StatusNow: a.StatusNow, LevelThen: a.LevelThen, AllowedAutoActions: a.AllowedThen, Suspicious: a.Suspicious,
		Reasons: a.Reasons}
	for _, st := range o.Stages {
		v.Stages = append(v.Stages, ObservationStage{Name: st.Name, Version: st.Version, ConfidenceBP: st.ConfidenceBP, Output: strp(st.Output)})
	}
	return v, nil
}

// Examples — размеченные примеры из подтверждённых решений (vision.example.list,
// FR-99): решение контролёра по сигналу анализатора → наблюдение, ответ
// анализатора и ответ эксперта раздельно.
func (s *Service) Examples(ctx context.Context, m platform.Moment) (LabeledExampleList, error) {
	if !s.live() {
		return s.Unimplemented.Examples(ctx, m)
	}
	sigs, err := s.records(ctx, catalog.QualitySignalRaised, m, true)
	if err != nil {
		return LabeledExampleList{}, err
	}
	causes := map[string][]string{}
	for _, d := range sigs {
		x, err := kernel.Decode[ev.QualitySignalRaisedV1](d.Record)
		if err != nil || d.Reaction == nil {
			continue
		}
		causes[string(x.SignalID)] = d.Reaction.Causes
	}
	by, _, err := s.seen(ctx, m)
	if err != nil {
		return LabeledExampleList{}, err
	}
	out := LabeledExampleList{Items: []LabeledExample{}}
	add := func(d engineapp.Decoded, verdict string, signals []ev.ObjectID, expertDefect *string, reason string, label bool) {
		for _, sg := range signals {
			for _, c := range causes[string(sg)] {
				o, ok := by[c]
				if !ok {
					continue
				}
				x := LabeledExample{DecisionEventID: d.Record.EventID, Verdict: verdict, DecidedAt: d.Record.OccurredAt.UTC(), ObservationEventID: o.EventID,
					ItemID: strp(o.ItemID), AnalyzerOutcome: o.Outcome, AnalyzerConfidence: o.ConfidenceBP, ExpertDefect: expertDefect,
					ExpertReason: strp(reason), Versions: o.Versions.Map(), ForAdaptation: label}
				if len(o.Defects) > 0 {
					x.AnalyzerDefect = strp(o.Defects[0].Code)
				}
				out.Items = append(out.Items, x)
			}
		}
	}
	conf, err := s.records(ctx, catalog.DecisionNonconformityConfirmed, m, true)
	if err != nil {
		return LabeledExampleList{}, err
	}
	for _, d := range conf {
		x, err := kernel.Decode[ev.DecisionNonconformityConfirmedV1](d.Record)
		if err != nil {
			continue
		}
		add(d, "confirmed", x.SignalIds, x.DefectTypeCode, string(x.Reason.Text), true)
	}
	rej, err := s.records(ctx, catalog.DecisionSignalRejected, m, true)
	if err != nil {
		return LabeledExampleList{}, err
	}
	for _, d := range rej {
		x, err := kernel.Decode[ev.DecisionSignalRejectedV1](d.Record)
		if err != nil {
			continue
		}
		add(d, "rejected", x.SignalIds, nil, string(x.Reason.Text), x.LabelForAdaptation != nil && *x.LabelForAdaptation)
	}
	slices.SortStableFunc(out.Items, func(a, b LabeledExample) int { return b.DecidedAt.Compare(a.DecidedAt) })
	return out, nil
}

// monitor — контроль дрейфа паспорта в прогоне m из состояния правила автоотката.
func (s *Service) monitor(ctx context.Context, passportID string, m platform.Moment) *AnalyzerMonitor {
	if s.d.Watch == nil {
		return nil
	}
	raw, ok, err := s.d.Watch.Get(ctx, StateWatch, StateWatchKey)
	if err != nil || !ok {
		return nil
	}
	w, err := dom.UnmarshalState(raw)
	if err != nil {
		return nil
	}
	out := &AnalyzerMonitor{Window: dom.DriftWindow, QualityMinBP: dom.DriftQualityBP, RecentQualityBP: []int{}}
	for _, p := range w.Passports {
		if p.PassportID != passportID {
			continue
		}
		rw, _ := p.RunOf(m.RunID)
		out.Seen = rw.Seen
		for _, f := range rw.Recent {
			out.RecentQualityBP = append(out.RecentQualityBP, f.QualityBP)
		}
	}
	return out
}
