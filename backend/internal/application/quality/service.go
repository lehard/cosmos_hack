package quality

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	"ant/internal/domain/quality"
)

// ProjectionReader — ведомый порт чтения проекций модуля (engine.projections
// через application/engine.ProjectionStore; запись — только эффектами, AD-45).
type ProjectionReader interface {
	Get(ctx context.Context, name, key string) (json.RawMessage, bool, error)
}

// StateAt — ведомый порт состояния изделия на момент (AD-22): та же свёртка
// над префиксом входа (application/engine.StateQueries). Нужен для чтения на
// момент в прошлом (as_of); «сейчас» читается из проекции.
type StateAt func(ctx context.Context, itemID string, m platform.Moment) (engine.Snapshot, error)

// Service — реализация live ведущих портов модуля quality (AD-36): чтение
// проекций quality.item и quality.index, на момент в прошлом — свёртка на
// момент; карта реакций — из нормативного слоя. Без хранилища проекций
// (выгрузка OpenAPI, тесты контракта) операции отвечают 501.
type Service struct {
	Unimplemented
	store ProjectionReader
	at    StateAt
	env   quality.Env
}

// Option — настройка Service.
type Option func(*Service)

// WithStore — хранилище проекций (live).
func WithStore(r ProjectionReader) Option { return func(s *Service) { s.store = r } }

// WithStateAt — состояние изделия на момент (as_of).
func WithStateAt(f StateAt) Option { return func(s *Service) { s.at = f } }

// WithEnv — нормативный слой quality (карта реакций, классификатор).
func WithEnv(env quality.Env) Option { return func(s *Service) { s.env = env } }

// NewService создаёт реализацию live.
func NewService(opts ...Option) *Service {
	s := &Service{}
	for _, o := range opts {
		o(s)
	}
	return s
}

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// item — выводы quality по изделию на момент m; ok=false — данных контроля нет.
func (s *Service) item(ctx context.Context, itemID string, m platform.Moment) (ItemRecord, bool, error) {
	if m.AsOf != nil && s.at != nil {
		snap, err := s.at(ctx, itemID, m)
		if err != nil {
			return ItemRecord{}, false, err
		}
		v, err := itemView(itemID, snap, nil)
		if err != nil || v == nil {
			return ItemRecord{}, false, err
		}
		return v.(ItemRecord), true, nil
	}
	raw, ok, err := s.store.Get(ctx, ItemProjection, itemID)
	if err != nil || !ok {
		return ItemRecord{}, false, err
	}
	var rec ItemRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return ItemRecord{}, false, err
	}
	return rec, true, nil
}

// items — выводы по всем изделиям прогона (или по одному изделию).
func (s *Service) items(ctx context.Context, itemID string, m platform.Moment) ([]ItemRecord, error) {
	ids := []string{itemID}
	if itemID == "" {
		raw, ok, err := s.store.Get(ctx, IndexProjection, IndexKey(m.RunID))
		if err != nil {
			return nil, err
		}
		var ix IndexRecord
		if ok {
			if err := json.Unmarshal(raw, &ix); err != nil {
				return nil, err
			}
		}
		ids = ix.Items
	}
	out := []ItemRecord{}
	for _, id := range ids {
		rec, ok, err := s.item(ctx, id, m)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, rec)
		}
	}
	return out, nil
}

func notFound(what, key, id string) error {
	e := platform.Fail(errcodes.ApiNotFound, key, id)
	e.Detail = what + " " + id + " не найден(о)"
	return e
}

// page — страница среза по курсору-смещению.
func page[T any](all []T, p platform.Page) ([]T, string) {
	from, _ := strconv.Atoi(p.Cursor)
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	if from < 0 || from > len(all) {
		from = len(all)
	}
	to := min(from+limit, len(all))
	next := ""
	if to < len(all) {
		next = strconv.Itoa(to)
	}
	return all[from:to], next
}

// Signals — сигналы (quality.signal.list): поднятые реакцией quality.signal.raised.
func (s *Service) Signals(ctx context.Context, f SignalFilter, m platform.Moment, p platform.Page) (QualitySignalList, error) {
	if s.store == nil {
		return s.Unimplemented.Signals(ctx, f, m, p)
	}
	recs, err := s.items(ctx, f.ItemID, m)
	if err != nil {
		return QualitySignalList{}, err
	}
	all := []QualitySignal{}
	for _, rec := range recs {
		for _, sg := range rec.State.Signals {
			if sg.Raised && (f.State == "" || sg.State == f.State) {
				all = append(all, signalView(rec, sg))
			}
		}
	}
	slices.SortStableFunc(all, func(a, b QualitySignal) int {
		if c := b.RaisedAt.Compare(a.RaisedAt); c != 0 {
			return c
		}
		return strings.Compare(a.SignalID, b.SignalID)
	})
	items, next := page(all, p)
	return QualitySignalList{Items: items, NextCursor: next}, nil
}

// Signal — исходный сигнал с наблюдением (quality.signal.read, FR-38).
func (s *Service) Signal(ctx context.Context, signalID string, m platform.Moment) (QualitySignal, error) {
	if s.store == nil {
		return s.Unimplemented.Signal(ctx, signalID, m)
	}
	recs, err := s.items(ctx, "", m)
	if err != nil {
		return QualitySignal{}, err
	}
	for _, rec := range recs {
		for _, sg := range rec.State.Signals {
			if sg.SignalID == signalID && sg.Raised {
				return signalView(rec, sg), nil
			}
		}
	}
	return QualitySignal{}, notFound("Сигнал", "signal_id", signalID)
}

// Inspections — результаты контроля изделия (quality.inspection.list, FR-36).
func (s *Service) Inspections(ctx context.Context, itemID string, m platform.Moment, p platform.Page) (InspectionResultList, error) {
	if s.store == nil {
		return s.Unimplemented.Inspections(ctx, itemID, m, p)
	}
	rec, _, err := s.item(ctx, itemID, m)
	if err != nil {
		return InspectionResultList{}, err
	}
	all := []InspectionResult{}
	for _, o := range rec.State.Observations {
		all = append(all, inspectionView(itemID, o))
	}
	items, next := page(all, p)
	return InspectionResultList{Items: items, NextCursor: next}, nil
}

// Coverage — полнота контроля изделия (quality.coverage.read, FR-35, FR-14).
func (s *Service) Coverage(ctx context.Context, itemID string, m platform.Moment) (InspectionCoverage, error) {
	if s.store == nil {
		return s.Unimplemented.Coverage(ctx, itemID, m)
	}
	rec, _, err := s.item(ctx, itemID, m)
	if err != nil {
		return InspectionCoverage{}, err
	}
	out := InspectionCoverage{ItemID: itemID, Complete: true, Points: []CoveragePoint{}, BasisSeq: rec.BasisSeq,
		QualityState: string(rec.State.Axis)}
	if out.QualityState == "" {
		out.QualityState = "not_inspected"
	}
	points := rec.State.Points
	if len(points) == 0 {
		// Изделие без данных контроля: точки плана ещё не ждут результата.
		points = quality.PlanPoints(quality.State{}, s.env)
	}
	for _, p := range points {
		cp := CoveragePoint{StepKey: p.StepKey, InspectionPoint: firstNonEmpty(p.InspectionPoint, p.ClosingPoint), Method: p.Method,
			Required: p.Required, Status: p.Status}
		if p.MissingReason != "" {
			r := p.MissingReason
			cp.MissingReason = &r
		}
		if p.EventID != "" {
			id := p.EventID
			cp.EventID = &id
		}
		if p.Required && p.Status != "received" {
			out.Complete = false
		}
		out.Points = append(out.Points, cp)
	}
	for _, t := range rec.State.Types {
		out.Types = append(out.Types, DefectTypeCoverage{Code: t.Code, Name: t.Name, Severity: t.Severity, Status: t.Status,
			Methods: t.Methods, EventIDs: t.EventIDs})
	}
	return out, nil
}

// Defects — физические дефекты (quality.defect.list, FR-37): повторные
// наблюдения не множат; дефекты и изделия с дефектами — раздельно.
func (s *Service) Defects(ctx context.Context, itemID string, m platform.Moment, p platform.Page) (QualityDefectList, error) {
	if s.store == nil {
		return s.Unimplemented.Defects(ctx, itemID, m, p)
	}
	recs, err := s.items(ctx, itemID, m)
	if err != nil {
		return QualityDefectList{}, err
	}
	all := []QualityDefect{}
	withDefect := 0
	for _, rec := range recs {
		if len(rec.State.Defects) > 0 {
			withDefect++
		}
		for _, d := range rec.State.Defects {
			v := QualityDefect{DefectID: d.DefectID, ItemID: rec.ItemID, ZoneID: d.Zone, Location: d.Location,
				FirstObservationEventID: d.FirstObservation, Observations: len(d.Observations), IdentifiedAt: d.IdentifiedAt}
			if d.TypeCode != "" {
				c := d.TypeCode
				v.DefectTypeCode = &c
			}
			all = append(all, v)
		}
	}
	items, next := page(all, p)
	return QualityDefectList{Items: items, DefectCount: len(all), ItemsWithDefect: withDefect, NextCursor: next}, nil
}

// Escapes — пропуски брака (quality.escape.list, FR-100).
func (s *Service) Escapes(ctx context.Context, m platform.Moment, p platform.Page) (QualityEscapeList, error) {
	if s.store == nil {
		return s.Unimplemented.Escapes(ctx, m, p)
	}
	recs, err := s.items(ctx, "", m)
	if err != nil {
		return QualityEscapeList{}, err
	}
	all := []QualityEscape{}
	for _, rec := range recs {
		for _, e := range rec.State.Escapes {
			// event_id записи quality.escape.recorded первой версии слота (AD-3).
			slot := kernel.Slot{RuleID: quality.RuleEscape, Subject: "item:" + rec.ItemID, TriggerKey: e.DefectID}
			v := QualityEscape{EventID: kernel.Reaction{Slot: slot}.ID(1), DefectID: e.DefectID, ItemID: rec.ItemID, MissedObservationEventIDs: e.Missed,
				MethodCoversDefect: e.MethodCovers}
			for _, d := range rec.State.Defects {
				if d.DefectID == e.DefectID {
					v.RecordedAt = d.IdentifiedAt
				}
			}
			if e.AnalyzerVersion != "" {
				a := e.AnalyzerVersion
				v.AnalyzerVersion = &a
			}
			all = append(all, v)
		}
	}
	items, next := page(all, p)
	return QualityEscapeList{Items: items, NextCursor: next}, nil
}

// ReactionMap — действующая карта реакций (quality.reaction_map.read, FR-48, FR-50).
func (s *Service) ReactionMap(ctx context.Context, m platform.Moment) (ReactionMap, error) {
	if len(s.env.ReactionMap.Rules) == 0 {
		return s.Unimplemented.ReactionMap(ctx, m)
	}
	return reactionMapView(s.env), nil
}
