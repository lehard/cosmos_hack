package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	"ant/internal/contracts/normative"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/vision"
)

// timeLayout — время в конверте: RFC 3339 UTC, три знака после секунд.
const timeLayout = "2006-01-02T15:04:05.000Z"

// Relay — сценарий «наблюдения внешней системы → события контракта» на краю
// (FR-38, FR-97, FR-126, AD-29): вектор версий на каждом наблюдении (неполный
// — явное unknown и пометка в ограничениях), детерминированный event_id,
// иллюстрация открытого набора только если хранилище её приняло (FR-102).
// Результат — события для локального входа edge-агента (source_id, номер и
// подпись ставит агент, AD-7).
type Relay struct {
	// Catalog — каталог иллюстраций (normative/vision/illustrations.v*.yaml).
	Catalog normative.IllustrationsCatalog
	// Files — файлы набора на краю; nil — офлайн их нет.
	Files IllustrationFiles
	// Materials — хранилище материалов; nil — приложить нельзя.
	Materials MaterialSink
}

// EventID — event_id наблюдения: UUIDv5 от системы, прогона и id результата у
// системы. Повтор той же системы — тот же event_id: приём даст «дубль» (AD-7).
func EventID(system, runID, observationID string) string {
	return kernel.UUIDv5(constants.NsAnt, "vision/"+system+"/"+runID+"/"+observationID)
}

// Events — события контракта из наблюдений.
func (r Relay) Events(ctx context.Context, s Signals) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(s.Inspections)+len(s.Actions))
	for _, in := range s.Inspections {
		e, err := r.inspection(ctx, s.System, in)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	for _, a := range s.Actions {
		e, err := action(s.System, a)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func envelope(t catalog.Type, id, runID string, it *ItemRef, occurred string, data any) (map[string]any, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("%s: data: %w", t, err)
	}
	e := map[string]any{"event_id": id, "event_type": string(t), "schema_version": 1, "occurred_at": occurred,
		"source_kind": "camera", "reliability": "high", "correlation_id": id, "causation_id": nil, "data": json.RawMessage(raw)}
	if runID != "" {
		e["run_id"] = runID
	}
	if it != nil && it.Value != "" {
		lvl := it.IdentificationLevel
		if lvl == "" {
			lvl = "unique"
		}
		e["item_ref"] = map[string]any{"carrier_type": it.CarrierType, "value": it.Value, "identification_level": lvl}
	}
	return e, nil
}

func bpp(p *int) *ev.Bp {
	if p == nil {
		return nil
	}
	v := ev.Bp(min(max(*p, 0), 10000))
	return &v
}

func sp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func oid(s string) *ev.ObjectID {
	if s == "" {
		return nil
	}
	v := ev.ObjectID(s)
	return &v
}

// versionsNote — ограничение «вектор версий неполный» (AD-29).
func versionsNote(v dom.Versions) (ev.AnalyzerVersions, []string) {
	_, miss := v.Complete()
	if len(miss) == 0 {
		return v.Contract(), nil
	}
	return v.Contract(), []string{"вектор версий неполный, неизвестно: " + strings.Join(miss, ", ")}
}

func (r Relay) inspection(ctx context.Context, system string, in Inspection) (map[string]any, error) {
	vers, lim := versionsNote(in.Versions)
	d := ev.InspectionResultRecordedV1{
		ObservationID: oid(in.ObservationID), Method: ev.InspectionMethodCamera, Phase: ev.InspectionResultRecordedV1Phase(in.Phase),
		StepKey: nil, InspectionPoint: sp(in.Point), Outcome: ev.InspectionOutcome(in.Outcome),
		ProcessingState: ev.ProcessingState(in.ProcessingState), AnalyzerConfidenceBp: bpp(in.ConfidenceBP),
		ObservationQualityBp: bpp(in.QualityBP), Versions: &vers, EquipmentID: oid(in.EquipmentID),
	}
	if d.Phase == "" {
		d.Phase = ev.InspectionResultRecordedV1PhaseAfterOperation
	}
	if in.StepKey != "" {
		k := ev.StepKey(in.StepKey)
		d.StepKey = &k
	}
	if in.UnableReason != "" {
		u := ev.InspectionResultRecordedV1UnableReason(in.UnableReason)
		d.UnableReason = &u
	}
	if in.Recommendation != "" {
		rc := ev.InspectionResultRecordedV1Recommendation(in.Recommendation)
		d.Recommendation = &rc
	}
	for _, st := range in.Stages {
		d.Stages = append(d.Stages, ev.AnalyzerStage{Stage: ev.Code(st.Name), Version: st.Version, ConfidenceBp: bpp(st.ConfidenceBP), OutputNote: sp(st.Output)})
	}
	codes := []string{}
	for _, f := range in.Findings {
		df := ev.Defect{ZoneID: ev.ObjectID(f.Zone), Severity: ev.Severity(f.Severity), StageConfidenceBp: bpp(f.ConfidenceBP), Location: sp(f.Location)}
		if df.Severity == "" {
			df.Severity = ev.SeverityUnknown
		}
		if f.DefectCode != "" {
			c := f.DefectCode
			df.DefectTypeCode = &c
			codes = append(codes, c)
		}
		if f.Note != "" {
			t := ev.Text(f.Note)
			df.Description = &t
		}
		d.Defects = append(d.Defects, df)
	}
	for _, z := range in.Zones {
		d.ZoneIds = append(d.ZoneIds, ev.ObjectID(z))
	}
	d.Limitations = append(append([]string{}, in.Limitations...), lim...)
	if in.Simulated {
		d.Limitations = append(d.Limitations, "результат получен системой в режиме симуляции")
	}
	if ref, note, ok := r.illustrate(ctx, in.Outcome, codes); ok {
		d.EvidenceRefs = append(d.EvidenceRefs, ref)
	} else if note != "" {
		d.Limitations = append(d.Limitations, note)
	}
	for i, l := range d.Limitations {
		d.Limitations[i] = clip(l, 256)
	}
	if len(d.Limitations) == 0 {
		d.Limitations = nil
	}
	id := EventID(system, in.RunID, in.ObservationID)
	return envelope(catalog.InspectionResultRecorded, id, in.RunID, in.Item, in.OccurredAt.UTC().Format(timeLayout), d)
}

// illustrate — иллюстрация к наблюдению (FR-102): выбрать класс набора,
// взять файл на краю, отдать хранилищу; не вышло — ссылка и метаданные в
// ограничениях, материала нет (фиктивное наличие не показывается).
func (r Relay) illustrate(ctx context.Context, outcome string, codes []string) (ev.EvidenceRef, string, bool) {
	ch, ok := dom.ChooseIllustration(r.Catalog, outcome, codes)
	if !ok {
		return ev.EvidenceRef{}, "", false
	}
	if r.Files == nil {
		return ev.EvidenceRef{}, ch.Unavailable("файлов набора нет офлайн"), false
	}
	b, err := r.Files.Sample(ch.Dataset.ID, ch.Class.DatasetClass, ch.Class.Sample)
	if err != nil {
		return ev.EvidenceRef{}, ch.Unavailable("файла образца нет офлайн"), false
	}
	if r.Materials == nil {
		return ev.EvidenceRef{}, ch.Unavailable("хранилище материалов недоступно"), false
	}
	note := ch.SourceNote()
	addr, err := r.Materials.PutIllustration(ctx, b, ch.Class.MediaType, note)
	if err != nil || addr == "" {
		return ev.EvidenceRef{}, ch.Unavailable("хранилище материалов не приняло файл"), false
	}
	return ev.EvidenceRef{MaterialAddress: ev.Digest(addr), MediaType: ch.Class.MediaType, Kind: ev.EvidenceRefKindIllustration,
		IsIllustration: true, SourceNote: &note}, "", true
}

func action(system string, a OperatorAction) (map[string]any, error) {
	// Ограничений у operator.action.observed в схеме нет: неполный вектор
	// виден по явному unknown в самом векторе.
	vers, _ := versionsNote(a.Versions)
	d := ev.OperatorActionObservedV1{Observation: ev.OperatorActionObservedV1Observation(a.Action), WorkplaceID: oid(a.WorkplaceID),
		TpStep: sp(a.TPStep), AnalyzerConfidenceBp: bpp(a.ConfidenceBP), ObservationQualityBp: bpp(a.QualityBP), Versions: &vers}
	id := EventID(system, a.RunID, a.HypothesisID)
	return envelope(catalog.OperatorActionObserved, id, a.RunID, a.Item, a.OccurredAt.UTC().Format(timeLayout), d)
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
