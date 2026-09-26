package vision

import (
	"context"
	"slices"
	"strconv"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/vision"
)

// analyzerTypes — записи семейства analyzer (эмитент vision, AD-40).
var analyzerTypes = []catalog.Type{
	catalog.AnalyzerPassportAdmitted, catalog.AnalyzerPassportSuspended, catalog.AnalyzerPassportReinstated,
	catalog.AnalyzerPassportRetired, catalog.AnalyzerCheckRecorded,
}

// within — запись видна на момент m (AD-22): «как было» — occurred_at ≤ T,
// «что мы знали» — recorded_at ≤ T.
func within(m platform.Moment, r kernel.Record) bool {
	if m.AsOf == nil {
		return true
	}
	t := r.OccurredAt
	if m.Axis == platform.AxisRecorded {
		t = r.RecordedAt
	}
	return !t.After(*m.AsOf)
}

// registry — реестр паспортов и проверок на момент m (AD-22). Паспорта —
// нормативный слой всего предприятия; автооткат и возврат в прогоне сценария
// (AD-38) пишутся с run_id прогона и видны только в нём: записи без run_id —
// всем, с run_id — только своему прогону.
func (s *Service) registry(ctx context.Context, m platform.Moment) (dom.Registry, error) {
	var recs []kernel.Record
	for _, t := range analyzerTypes {
		q := appjournal.ReadQuery{EventType: string(t), Limit: 1000}
		for {
			page, err := s.d.Journal.Read(ctx, q)
			if err != nil {
				return dom.Registry{}, err
			}
			for _, e := range page {
				d, err := s.d.Codec.Decode(ctx, e)
				if err != nil {
					continue // нечитаемая запись паспорта — не паспорт (строже)
				}
				if within(m, d.Record) {
					recs = append(recs, d.Record)
				}
			}
			if len(page) < q.Limit {
				break
			}
			q.AfterSeq = int64(page[len(page)-1].Seq)
		}
	}
	return dom.Fold(recs), nil
}

func notFound(id string) error {
	return platform.Fail(errcodes.ApiNotFound, "object", "analyzer_passport", "id", id)
}

func ptr[T any](v T) *T { return &v }

// Analyzers — анализаторы с действующими паспортами (vision.analyzer.list):
// на анализатор — паспорт в действии, иначе последний по записи.
func (s *Service) Analyzers(ctx context.Context, m platform.Moment) (AnalyzerList, error) {
	if !s.live() {
		return s.Unimplemented.Analyzers(ctx, m)
	}
	reg, err := s.registry(ctx, m)
	if err != nil {
		return AnalyzerList{}, err
	}
	pick := map[string]dom.Passport{}
	order := []string{}
	for _, p := range reg.Passports {
		cur, ok := pick[p.AnalyzerID]
		if !ok {
			order = append(order, p.AnalyzerID)
		}
		better := !ok ||
			(p.Current().Status == dom.StatusActive && cur.Current().Status != dom.StatusActive) ||
			((p.Current().Status == dom.StatusActive) == (cur.Current().Status == dom.StatusActive) && p.Seq > cur.Seq)
		if better {
			pick[p.AnalyzerID] = p
		}
	}
	slices.Sort(order)
	out := AnalyzerList{Items: []AnalyzerSummary{}}
	for _, a := range order {
		p := pick[a]
		out.Items = append(out.Items, AnalyzerSummary{AnalyzerID: p.AnalyzerID, Title: p.Title, Kind: p.AnalyzerKind,
			PassportID: ptr(p.PassportID), Stage: ptr(p.Stage), TrustLevel: ptr(p.TrustLevel), Status: p.Current().Status,
			Versions: p.Versions.Map(), Provenance: provenance(p)})
	}
	return out, nil
}

func provenance(p dom.Passport) *string {
	if p.Provenance == "" {
		return nil
	}
	return ptr(p.Provenance)
}

// passportView — форма ответа паспорта.
func passportView(p dom.Passport, basis int64) AnalyzerPassport {
	v := AnalyzerPassport{PassportID: p.PassportID, AnalyzerID: p.AnalyzerID, Stage: p.Stage, TrustLevel: p.TrustLevel,
		RecipeRef: p.RecipeRef, Versions: p.Versions.Map(), Status: p.Current().Status, AdmittedAt: p.AdmittedAt.UTC(),
		DocumentID: p.DocumentID, AllowedAutoActions: dom.AllowedAutoActions(p.EffectiveLevel()), BasisSeq: basis,
		Provenance: provenance(p), Title: ptr(p.Title), AnalyzerKind: ptr(p.AnalyzerKind)}
	if p.PreviousPassportID != "" {
		v.PreviousPassportID = ptr(p.PreviousPassportID)
	}
	if st, ok := p.LastSuspension(); ok {
		v.Suspension = &AnalyzerSuspension{EventID: st.EventID, Trigger: st.Trigger, Fallback: st.Fallback, At: st.At.UTC(), Basis: st.Basis}
		if st.Note != "" {
			v.Suspension.Note = ptr(st.Note)
		}
		if st.RunID != "" {
			v.Suspension.RunID = ptr(st.RunID)
		}
		if st.FallbackPassportID != "" {
			v.Suspension.FallbackPassportID = ptr(st.FallbackPassportID)
		}
	}
	for _, h := range p.History {
		x := AnalyzerStatus{Status: h.Status, At: h.At.UTC(), EventID: h.EventID}
		if h.Trigger != "" {
			x.Trigger = ptr(h.Trigger)
		}
		if h.Note != "" {
			x.Note = ptr(h.Note)
		}
		v.History = append(v.History, x)
	}
	return v
}

// Passport — паспорт допуска (vision.passport.read).
func (s *Service) Passport(ctx context.Context, passportID string, m platform.Moment) (AnalyzerPassport, error) {
	if !s.live() {
		return s.Unimplemented.Passport(ctx, passportID, m)
	}
	reg, err := s.registry(ctx, m)
	if err != nil {
		return AnalyzerPassport{}, err
	}
	p, ok := reg.Passport(passportID)
	if !ok {
		return AnalyzerPassport{}, notFound(passportID)
	}
	v := passportView(p, reg.BasisSeq)
	v.Monitor = s.monitor(ctx, passportID, m)
	return v, nil
}

// Checks — отчёты проверки анализатора (vision.check.list) по порядку записи;
// курсор — номер следующего элемента.
func (s *Service) Checks(ctx context.Context, passportID string, m platform.Moment, pg platform.Page) (AnalyzerCheckList, error) {
	if !s.live() {
		return s.Unimplemented.Checks(ctx, passportID, m, pg)
	}
	reg, err := s.registry(ctx, m)
	if err != nil {
		return AnalyzerCheckList{}, err
	}
	if _, ok := reg.Passport(passportID); !ok {
		return AnalyzerCheckList{}, notFound(passportID)
	}
	all := reg.ChecksOf(passportID)
	from, _ := strconv.Atoi(pg.Cursor)
	from = min(max(from, 0), len(all))
	limit := pg.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	to := min(from+limit, len(all))
	out := AnalyzerCheckList{Items: []AnalyzerCheck{}}
	for _, c := range all[from:to] {
		out.Items = append(out.Items, AnalyzerCheck{EventID: c.EventID, PassportID: c.PassportID, CheckKind: c.Kind,
			EscapeRateBP: c.EscapeRateBP, FalseAlarmRateBP: c.FalseAlarmRateBP, DisagreementRateBP: c.DisagreementRateBP,
			Passed: c.Passed, OccurredAt: c.OccurredAt.UTC()})
	}
	if to < len(all) {
		out.NextCursor = strconv.Itoa(to)
	}
	return out, nil
}
