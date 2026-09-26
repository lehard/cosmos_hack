package nonconformity

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/nonconformity"
)

// Чтение модуля nonconformity в режиме live (AD-22): каждое изделие — та же
// свёртка, что у воркера, над входом из журнала на момент; очередь и список —
// по изделиям индекса (черновики, регистрации, изоляции, предъявления).

// at — доменное «сейчас» ответа: момент запроса или часы прогона.
func (s *Service) at(ctx context.Context, m platform.Moment, runID string) time.Time {
	if m.AsOf != nil {
		return *m.AsOf
	}
	t, err := s.now(ctx, runID)
	if err != nil {
		return s.d.Now().UTC()
	}
	return t
}

// Queue — очередь «Ждут моего решения» (PRD §3a): сигналы на рассмотрение,
// изолированные и ждущие решения изделия со сроком, точки предъявления;
// сортировка по риску (тяжесть, затем срок) или по сроку.
func (s *Service) Queue(ctx context.Context, f QueueFilter, m platform.Moment, p platform.Page) (DecisionQueue, error) {
	if !s.live() {
		return s.Unimplemented.Queue(ctx, f, m, p)
	}
	items, err := s.indexItems(ctx, m)
	if err != nil {
		return DecisionQueue{}, err
	}
	rows := []DecisionQueueRow{}
	for _, it := range items {
		v, err := s.loadItem(ctx, it, m)
		if err != nil {
			continue
		}
		rows = append(rows, s.queueRows(v, s.at(ctx, m, v.RunID))...)
		rows = append(rows, s.reviewRows(ctx, v)...)
	}
	sevRank := map[string]int{"critical": 0, "major": 1, "minor": 2, "unknown": 3}
	due := func(r DecisionQueueRow) int64 {
		if r.DueAt == nil {
			return 1 << 62
		}
		return r.DueAt.Unix()
	}
	slices.SortStableFunc(rows, func(a, b DecisionQueueRow) int {
		if a.Overdue != b.Overdue {
			if a.Overdue {
				return -1
			}
			return 1
		}
		if d := sevRank[a.Severity] - sevRank[b.Severity]; d != 0 {
			return d
		}
		if da, db := due(a), due(b); da != db {
			if da < db {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ObjectID, b.ObjectID)
	})
	for i := range rows {
		rows[i].RiskRank = i
	}
	if f.Sort == "deadline" {
		slices.SortStableFunc(rows, func(a, b DecisionQueueRow) int {
			da, db := due(a), due(b)
			switch {
			case da < db:
				return -1
			case da > db:
				return 1
			}
			return a.RiskRank - b.RiskRank
		})
	}
	out := rows[:0]
	for _, r := range rows {
		if f.Kind == "" || r.Kind == f.Kind {
			out = append(out, r)
		}
	}
	return DecisionQueue{Items: page(out, p)}, nil
}

func page[T any](xs []T, p platform.Page) []T {
	if p.Limit > 0 && len(xs) > p.Limit {
		return xs[:p.Limit]
	}
	return xs
}

// queueRows — строки очереди изделия.
func (s *Service) queueRows(v *itemView, now time.Time) []DecisionQueueRow {
	st := v.State()
	var rows []DecisionQueueRow
	label := v.ItemID
	var isoDue *time.Time
	overdue := false
	if st.Isolated() && st.Isolation.DecisionDueAt != nil {
		isoDue = st.Isolation.DecisionDueAt
		overdue = now.After(*isoDue)
	}
	waiting := false
	for _, n := range st.NCs {
		id := n.ID
		switch n.Status {
		case dom.StatusDraft:
			rows = append(rows, DecisionQueueRow{Kind: "signal", ObjectID: n.ID, NCID: &id, ItemID: v.ItemID, ItemLabel: label,
				StepKey: n.Draft.StepKey, Title: ncTitle(v, n), Reason: essence(v, n), Severity: severity(n.Severity), DueAt: isoDue, Overdue: overdue, BasisSeq: v.BasisSeq})
			waiting = true
		case dom.StatusConfirmed:
			// Подтверждено — ждёт решения по несоответствию (у спецпроцесса — комиссии).
			rows = append(rows, DecisionQueueRow{Kind: "isolated", ObjectID: n.ID, NCID: &id, ItemID: v.ItemID, ItemLabel: label,
				StepKey: n.Draft.StepKey, Title: ncTitle(v, n), Reason: essence(v, n), Severity: severity(n.Severity), DueAt: isoDue, Overdue: overdue, BasisSeq: v.BasisSeq})
			waiting = true
		}
	}
	if st.Isolated() && !waiting {
		rows = append(rows, DecisionQueueRow{Kind: "isolated", ObjectID: v.ItemID, ItemID: v.ItemID, ItemLabel: label,
			Title: "Изолировано — ждёт решения", Severity: "unknown", DueAt: isoDue, Overdue: overdue, BasisSeq: v.BasisSeq})
	}
	if pr := st.PendingPresentation(); pr != nil {
		no := pr.PresentationNo
		title := "Предъявление на " + pr.ClosingPoint
		if pr.ClosingPoint == "" {
			title = "Предъявление"
			if name := stepName(v, pr.StepKey); name != "" {
				title += ": " + name
			}
		}
		if no > 1 {
			title += " (повторное)"
		}
		rows = append(rows, DecisionQueueRow{Kind: "presentation", ObjectID: pr.EventID, ItemID: v.ItemID, ItemLabel: label,
			StepKey: pr.StepKey, Title: title, Severity: "unknown", PresentationN: &no, BasisSeq: v.BasisSeq})
	}
	return rows
}

// reviewRows — строки «решение принято до новых данных — пересмотрите»
// (AD-3; реакция движка task.task.created вида review_after_new_data): что
// пришло после решения — словами (название вида записи и оборудования), id
// пришедшей записи — source_event_id. Решение, принятое позже пришедших
// данных, пересмотр закрывает.
func (s *Service) reviewRows(ctx context.Context, v *itemView) []DecisionQueueRow {
	var rows []DecisionQueueRow
	for _, re := range v.Computed {
		d, ok := re.Data.(ev.TaskTaskCreatedV1)
		if re.Type != catalog.TaskTaskCreated || !ok || d.Kind != ev.TaskTaskCreatedV1KindReviewAfterNewData {
			continue
		}
		dec, ok := v.Record(re.Slot.TriggerKey)
		if !ok {
			continue
		}
		var last *kernel.Record
		for _, id := range re.Causes {
			r, ok := v.Record(id)
			if !ok || r.EventID == dec.EventID || (last != nil && r.Seq <= last.Seq) {
				continue
			}
			rr := r
			last = &rr
		}
		if last == nil || s.redecided(v, dec, *last) {
			continue
		}
		var pd struct {
			StepKey      string `json:"step_key"`
			ClosingPoint string `json:"closing_point"`
		}
		_ = json.Unmarshal(dec.Data, &pd)
		title := "Решение"
		if pd.ClosingPoint != "" {
			title += " " + pd.ClosingPoint
		}
		title += " принято до новых данных — пересмотрите: пришло «" + summaryOf(*last) + "»"
		var eq struct {
			EquipmentID string `json:"equipment_id"`
		}
		if json.Unmarshal(last.Data, &eq) == nil && eq.EquipmentID != "" && s.d.Equipment != nil {
			if name, ok := s.d.Equipment.EquipmentName(ctx, eq.EquipmentID, last.OccurredAt); ok {
				title += " от «" + name + "»"
			}
		}
		src, since := last.EventID, re.OccurredAt
		rows = append(rows, DecisionQueueRow{Kind: "review", ObjectID: dec.EventID, ItemID: v.ItemID, ItemLabel: v.ItemID, StepKey: pd.StepKey,
			Title: title, Severity: "major", BasisSeq: v.BasisSeq, SourceEventID: &src, ReviewSince: &since})
	}
	return rows
}

// redecided — после пришедшей записи last по изделию принято новое решение
// того же вида: пересмотр выполнен.
func (s *Service) redecided(v *itemView, dec, last kernel.Record) bool {
	for _, r := range v.Input {
		if r.Kind == catalog.KindDecision && r.Type == dec.Type && r.EventID != dec.EventID && r.Seq > last.Seq {
			return true
		}
	}
	return false
}

func severity(s string) string {
	switch s {
	case "critical", "major", "minor":
		return s
	}
	return "unknown"
}

// ncTitle — заголовок несоответствия для людей, без кодов: вид дефекта и
// зона — по справочникам (нет в справочнике — не называются), шаг — по
// описанию процесса.
func ncTitle(v *itemView, n dom.NC) string {
	if n.Origin == dom.OriginSpecialProcess {
		return "Нарушение режима специального процесса — решение комиссии"
	}
	what := "Признак дефекта"
	if l := nameIn(v.Labels.defects, &n.DefectTypeCode); l != nil {
		what += " «" + *l + "»"
	} else if n.DefectTypeCode != "" {
		what += " (вид не из классификатора)"
	}
	zone := n.Draft.ZoneID
	if l := nameIn(v.Labels.zones, &zone); l != nil {
		what += " · " + *l
	}
	if name := stepName(v, n.Draft.StepKey); name != "" {
		what += " — после шага «" + name + "»"
	}
	if n.Status == dom.StatusConfirmed {
		return "Подтверждено: " + strings.TrimPrefix(what, "Признак ")
	}
	return what
}

// essence — суть несоответствия для строки очереди: вид дефекта и зона по
// справочникам; ничего не нашлось — nil.
func essence(v *itemView, n dom.NC) *string {
	var parts []string
	if l := nameIn(v.Labels.defects, &n.DefectTypeCode); l != nil {
		parts = append(parts, *l)
	}
	zone := n.Draft.ZoneID
	if l := nameIn(v.Labels.zones, &zone); l != nil {
		parts = append(parts, *l)
	}
	if len(parts) == 0 {
		return nil
	}
	s := strings.Join(parts, " · ")
	return &s
}

// stepName — название шага по описанию процесса изделия; нет — "".
func stepName(v *itemView, stepKey string) string {
	if def := v.Env.Process.Def; def != nil && stepKey != "" {
		if nd := def.ByStep(stepKey); nd != nil {
			return nd.Name
		}
	}
	return ""
}

// List — несоответствия (журнал регистрации) с фильтром по изделию и статусу.
func (s *Service) List(ctx context.Context, f NCFilter, m platform.Moment, p platform.Page) (NCList, error) {
	if !s.live() {
		return s.Unimplemented.List(ctx, f, m, p)
	}
	items := []string{f.ItemID}
	if f.ItemID == "" {
		var err error
		if items, err = s.indexItems(ctx, m); err != nil {
			return NCList{}, err
		}
	}
	out := []NCSummary{}
	for _, it := range items {
		v, err := s.loadItem(ctx, it, m)
		if err != nil {
			if f.ItemID != "" {
				return NCList{}, err
			}
			continue
		}
		st := v.State()
		for _, n := range st.NCs {
			s.ncItems.Store(n.ID, it)
			if f.Status != "" && n.Status != f.Status {
				continue
			}
			sum := NCSummary{NCID: n.ID, Number: n.Number, Status: n.Status, ItemID: it, ItemLabel: it, Severity: severity(n.Severity),
				StepKey: n.Draft.StepKey, Disposition: dispositionOf(n), FoundAt: n.FoundAt,
				InvestigationStatus: n.Investigation, Containment: st.ContainmentLevel(), Commission: n.Commission}
			if n.DefectTypeCode != "" {
				c := n.DefectTypeCode
				sum.DefectTypeCode = &c
				sum.DefectTypeLabel = nameIn(v.Labels.defects, &c)
			}
			out = append(out, sum)
		}
	}
	slices.SortStableFunc(out, func(a, b NCSummary) int { return b.FoundAt.Compare(a.FoundAt) })
	return NCList{Items: page(out, p)}, nil
}

func dispositionOf(n dom.NC) string {
	if n.Disposition == "" {
		return "none"
	}
	return n.Disposition
}

// Concessions — разрешения на отклонение, применимые к изделию (FR-54): в
// области действия изделия; без изделия — все.
func (s *Service) Concessions(ctx context.Context, itemID string, m platform.Moment) (ConcessionList, error) {
	if !s.live() {
		return s.Unimplemented.Concessions(ctx, itemID, m)
	}
	book, err := s.concessionBook(ctx, m)
	if err != nil {
		return ConcessionList{}, err
	}
	now := s.at(ctx, m, "")
	out := ConcessionList{Items: []Concession{}}
	for _, c := range book.Sorted() {
		if itemID != "" && !c.InScope(itemID) {
			continue
		}
		g := c.Grant
		title := g.Title
		if title == "" {
			title = g.Number
		}
		limit := g.Limit
		v := Concession{ConcessionID: g.ConcessionID, Title: title, Kind: g.Kind, DocumentID: g.DocumentID, Limit: &limit,
			Used: c.Used(), ValidUntil: c.ValidUntil, Status: c.Status(now), ScopeItemIDs: g.ScopeItemIDs, BasisSeq: c.StreamSeq}
		if v.DocumentID == "" {
			v.DocumentID = g.ConcessionID
		}
		v.Number = optStr(g.Number)
		v.RequirementRef = optStr(g.RequirementRef)
		v.ScopeRangeFrom = optStr(g.ScopeRangeFrom)
		v.ScopeRangeTo = optStr(g.ScopeRangeTo)
		out.Items = append(out.Items, v)
	}
	return out, nil
}

func optStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ── карточка ──

// signalData — данные сигнала quality.signal.raised (реакция quality).
type signalData struct {
	SignalID             string `json:"signal_id"`
	BasisKind            string `json:"basis_kind"`
	DefectTypeCode       string `json:"defect_type_code"`
	DefectTypeKnown      *bool  `json:"defect_type_known"`
	ZoneID               string `json:"zone_id"`
	Severity             string `json:"severity"`
	RequirementRef       string `json:"requirement_ref"`
	AnalyzerConfidenceBP *int   `json:"analyzer_confidence_bp"`
	ObservationQualityBP *int   `json:"observation_quality_bp"`
	ReactionOutcome      string `json:"reaction_outcome"`
	ReactionMapRef       string `json:"reaction_map_ref"`
}

// inspectionData — поля результата контроля, нужные карточке.
type inspectionData struct {
	Method               string   `json:"method"`
	Phase                string   `json:"phase"`
	StepKey              string   `json:"step_key"`
	InspectionPoint      string   `json:"inspection_point"`
	OperationRunID       string   `json:"operation_run_id"`
	Outcome              string   `json:"outcome"`
	ZoneIDs              []string `json:"zone_ids"`
	AnalyzerConfidenceBP *int     `json:"analyzer_confidence_bp"`
	ObservationQualityBP *int     `json:"observation_quality_bp"`
	Stages               []struct {
		Stage        string `json:"stage"`
		Version      string `json:"version"`
		ConfidenceBP *int   `json:"confidence_bp"`
		OutputNote   string `json:"output_note"`
	} `json:"stages"`
	Versions     map[string]any `json:"versions"`
	EvidenceRefs []struct {
		MaterialAddress string `json:"material_address"`
	} `json:"evidence_refs"`
}

// runData — поля выполнения операции.
type runData struct {
	OperationRunID string  `json:"operation_run_id"`
	OperationCode  string  `json:"operation_code"`
	StepKey        string  `json:"step_key"`
	EquipmentID    string  `json:"equipment_id"`
	OperatorID     *string `json:"operator_id"`
	ProgramRef     string  `json:"program_ref"`
}

// Card — карточка несоответствия (FR-51): исходный сигнал, анализ системы
// (все версии вывода), решения людей и итоговый статус — раздельно.
func (s *Service) Card(ctx context.Context, ncID string, m platform.Moment) (NCCard, error) {
	if !s.live() {
		return s.Unimplemented.Card(ctx, ncID, m)
	}
	v, n, err := s.loadNC(ctx, ncID, m)
	if err != nil {
		return NCCard{}, err
	}
	c := s.card(v, n, s.at(ctx, m, v.RunID))
	s.withLabels(ctx, v, &c)
	if n.Origin == dom.OriginSpecialProcess {
		// FR-151, S05 (NC-G1): состав группового несоответствия окна.
		items, err := s.ncGroup(ctx, ncID, m)
		if err != nil {
			return NCCard{}, err
		}
		c.GroupItemIDs = items
	}
	return c, nil
}

func (s *Service) card(v *itemView, n dom.NC, now time.Time) NCCard {
	st := v.State()
	c := NCCard{
		NCID: n.ID, Number: n.Number, Status: n.Status, ItemID: v.ItemID, ItemLabel: v.ItemID,
		Axes: axes(st), BasisSeq: v.BasisSeq, InvestigationStatus: n.Investigation, Origin: n.Origin, Commission: n.Commission,
		PhysicallyNotMoved: st.PhysicallyNotMoved(), HumanDecisions: []NCRecordRef{},
	}
	if n.Resolution != "" {
		r := n.Resolution
		c.Resolution = &r
	}
	if n.ApprovalsStatus != "" {
		a := n.ApprovalsStatus
		c.ApprovalsStatus = &a
	}
	if n.Draft.ReactionMapRef != "" {
		r := n.Draft.ReactionMapRef
		c.RuleRev = &r
	}
	c.Happened = happened(v, n)
	c.Evidence = evidence(v, n)
	c.SystemAnalysis = analysis(v, n, c.Happened)
	for _, d := range st.Decisions {
		if d.NCID != "" && d.NCID != n.ID {
			continue
		}
		seq := d.Seq
		author := strings.SplitN(d.Actor, "@", 2)[0]
		ref := NCRecordRef{EventID: d.EventID, EventType: d.Type, Kind: "decision", Seq: &seq, OccurredAt: d.At, Summary: d.Summary}
		if author != "" {
			ref.Author = &author
		}
		c.HumanDecisions = append(c.HumanDecisions, ref)
	}
	if pr := st.PendingPresentation(); pr != nil {
		c.Presentation = &NCPresentationContext{StepKey: pr.StepKey, ClosingPoint: pr.ClosingPoint, PresentationNo: pr.PresentationNo, EventID: pr.EventID,
			MethodEventIDs: slices.Clone(st.Inspections)}
	}
	if iso := st.Isolation; iso != nil && !iso.Released {
		ni := &NCIsolation{EventID: iso.EventID, IsolatedAt: iso.At, DecisionDueAt: iso.DecisionDueAt, PhysicallyMoved: iso.PhysicallyMoved}
		ni.IsolatorLocationID = optStr(iso.IsolatorLocationID)
		ni.Overdue = iso.DecisionDueAt != nil && now.After(*iso.DecisionDueAt)
		c.Isolation = ni
	}
	for _, src := range st.Containment {
		if src.Released {
			continue
		}
		x := NCContainmentSource{Key: src.Key, Level: src.Level, By: src.By, Reason: src.Reason,
			BasisGone: src.By == dom.ByRule && levelRankOf(src.Basis) < levelRankOf(src.Level)}
		if src.Rule != "" {
			r := src.Rule
			x.RuleID = &r
		}
		c.Containment = append(c.Containment, x)
	}
	c.ToDecide = toDecide(st, n)
	return c
}

func levelRankOf(l string) int {
	return slices.Index([]string{"none", "observe", "additional_check", "item_hold", "lot_hold"}, l)
}

// axes — оси статуса изделия (PRD §3b, AD-30) в смысле контракта: ось
// «решение по изделию» и «сдерживание» — nonconformity; положение и качество
// — выводом из решений (владельцы process и quality применяют их по
// намерениям); учёт в 1С — только по квитанции erp.
func axes(st dom.State) NCItemAxes {
	a := NCItemAxes{Position: "in_progress", Quality: "not_inspected", Disposition: st.Disposition(), Containment: st.ContainmentLevel(),
		ErpAccounting: "not_sent"}
	if !st.Processed {
		a.Position = "in_queue"
	}
	if st.PendingPresentation() != nil {
		a.Position = "at_presentation_point"
	}
	if st.Isolated() {
		a.Position = "isolated"
	}
	if len(st.Inspections) > 0 {
		a.Quality = "conforming"
	}
	for _, n := range st.NCs {
		switch {
		case n.Status == dom.StatusDraft:
			a.Quality = "signal"
		case (n.Disposition == "use_as_is" && n.Executed) || (n.Disposition == "repair" && n.Status == dom.StatusVerified):
			a.Quality = "accepted_with_concession"
		case n.Status != dom.StatusClosed || n.Resolution == "":
			a.Quality = "nonconforming"
		}
	}
	return a
}

// toDecide — допустимые по состоянию решения (права — access.permission.list).
func toDecide(st dom.State, n dom.NC) NCToDecide {
	var ops []string
	add := func(op ...string) { ops = append(ops, op...) }
	switch n.Status {
	case dom.StatusDraft:
		add(dom.ActConfirm, dom.ActRejectSignal, dom.ActRecheck)
		if pr := st.PendingPresentation(); pr != nil && !st.Blocked() {
			add(dom.ActPresentation)
		}
		if !st.Isolated() {
			add(dom.ActIsolate)
		}
	case dom.StatusConfirmed:
		add(dom.ActDisposition, dom.ActRecheck)
		if !st.Isolated() {
			add(dom.ActIsolate)
		}
	case dom.StatusDispositionSet:
		if n.Executed {
			switch n.Disposition {
			case "rework", "repair":
				add(dom.ActVerify)
			default:
				add(dom.ActClose)
			}
		}
	case dom.StatusVerified:
		add(dom.ActClose)
	}
	if st.Blocked() || st.ContainmentLevel() != "none" {
		add(dom.ActContainmentRelease)
	}
	t := NCToDecide{Decisions: ops, ConcessionRequired: n.Status == dom.StatusConfirmed}
	if iso := st.Isolation; iso != nil && !iso.Released {
		t.DecisionDueAt = iso.DecisionDueAt
	}
	if t.Decisions == nil {
		t.Decisions = []string{}
	}
	return t
}

func ref(r kernel.Record, summary string) NCRecordRef {
	seq := r.Seq
	kind := string(r.Kind)
	switch kind {
	case "fact", "reaction", "decision", "service":
	default:
		kind = "fact"
	}
	x := NCRecordRef{EventID: r.EventID, EventType: string(r.Type), Kind: kind, Seq: &seq, OccurredAt: r.OccurredAt, Summary: summary}
	if r.Kind == catalog.KindFact && r.SourceKind != "" {
		switch r.SourceKind {
		case "manual_entry", "machine", "sensor", "camera", "external_system", "import":
			sk := r.SourceKind
			x.SourceKind = &sk
		}
	}
	if a := strings.SplitN(r.Actor, "@", 2)[0]; a != "" {
		x.Author = &a
	}
	x.Reading = readingOf(r)
	return x
}

// happened — зона «что произошло» (FR-51): до операции / операция / после.
func happened(v *itemView, n dom.NC) NCHappened {
	h := NCHappened{Before: []NCRecordRef{}, During: []NCRecordRef{}, After: []NCRecordRef{}}
	runID := n.Draft.OperationRunID
	var start, end *time.Time
	for _, r := range v.Input {
		var d runData
		switch r.Type {
		case catalog.OperationRunStarted:
			if json.Unmarshal(r.Data, &d) != nil || (runID != "" && d.OperationRunID != runID) {
				continue
			}
			if runID == "" && n.Draft.StepKey != "" && d.StepKey != n.Draft.StepKey {
				continue
			}
			t := r.OccurredAt
			start = &t
			op := &NCOperationContext{OperationRunID: d.OperationRunID, StepKey: d.StepKey, Label: d.OperationCode, StartedAt: &t}
			op.EquipmentID, op.ProgramRef, op.PerformerID = optStr(d.EquipmentID), optStr(d.ProgramRef), nil
			if d.OperatorID != nil && *d.OperatorID != "" {
				p := *d.OperatorID
				op.PerformerID = &p
			}
			if op.Label == "" {
				op.Label = d.StepKey
			}
			h.Operation = op
			runID = d.OperationRunID
		case catalog.OperationRunFinished:
			if json.Unmarshal(r.Data, &d) == nil && d.OperationRunID == runID && h.Operation != nil {
				t := r.OccurredAt
				end = &t
				h.Operation.FinishedAt = &t
			}
		}
	}
	for _, r := range v.Input {
		if r.Type == catalog.OperationRunStarted || r.Type == catalog.OperationRunFinished {
			continue
		}
		if r.Kind == catalog.KindDecision {
			continue
		}
		x := ref(r, summaryOf(r))
		switch {
		case start != nil && r.OccurredAt.Before(*start):
			h.Before = append(h.Before, x)
		case start != nil && (end == nil || !r.OccurredAt.After(*end)) && !r.OccurredAt.Before(*start) && r.Type != catalog.InspectionResultRecorded:
			h.During = append(h.During, x)
		default:
			h.After = append(h.After, x)
		}
	}
	return h
}

// summaryOf — строка записи для людей.
func summaryOf(r kernel.Record) string {
	switch r.Type {
	case catalog.InspectionResultRecorded:
		var d inspectionData
		_ = json.Unmarshal(r.Data, &d)
		out := "Контроль " + d.Method
		switch d.Outcome {
		case "defect_indicated":
			out += ": признаки дефекта"
		case "no_defect_indicated":
			out += ": признаков дефекта нет"
		case "unable_to_assess":
			out += ": оценка невозможна"
		}
		if d.InspectionPoint != "" {
			out += " (" + d.InspectionPoint + ")"
		}
		return out
	case catalog.DecisionNonconformityRegistered:
		return "Несоответствие зарегистрировано правилом (окно нарушения специального процесса)"
	case catalog.DecisionCleanPointAssigned:
		return "Точка чистоты: усиленный контроль"
	case catalog.IncidentMembershipChanged:
		return "Статус в области риска инцидента изменён"
	}
	if info, ok := catalog.Lookup(r.Type); ok && info.Title != "" {
		return info.Title
	}
	return string(r.Type)
}

// evidence — зона «доказательства»: исходные сигналы (отдельно от анализа
// системы), требование, история зоны.
func evidence(v *itemView, n dom.NC) NCEvidence {
	e := NCEvidence{Signals: []NCSourceSignal{}, ZoneHistory: []NCRecordRef{}}
	signals := map[string]signalData{}
	signalRef := map[string]NCRecordRef{}
	for _, r := range v.Reactions {
		if r.Record.Type != catalog.QualitySignalRaised {
			continue
		}
		var d signalData
		if json.Unmarshal(r.Record.Data, &d) == nil && slices.Contains(n.Draft.SignalIDs, d.SignalID) {
			signals[d.SignalID] = d
			signalRef[d.SignalID] = ref(r.Record, "Сигнал: "+d.ReactionOutcome)
		}
	}
	for _, cr := range v.Computed {
		if cr.Type != catalog.QualitySignalRaised {
			continue
		}
		b, _ := json.Marshal(cr.Data)
		var d signalData
		if json.Unmarshal(b, &d) == nil && slices.Contains(n.Draft.SignalIDs, d.SignalID) {
			if _, ok := signals[d.SignalID]; !ok {
				signals[d.SignalID] = d
			}
		}
	}
	// Результат контроля — основание черновика (причина намерения).
	var insp *kernel.Record
	for _, id := range n.Causes {
		if r, ok := v.Record(id); ok && (r.Type == catalog.InspectionResultRecorded || insp == nil) {
			rr := r
			insp = &rr
		}
	}
	for _, sid := range n.Draft.SignalIDs {
		d, ok := signals[sid]
		if !ok {
			d = signalData{SignalID: sid, BasisKind: n.Draft.BasisKind, DefectTypeCode: n.Draft.DefectTypeCode, ZoneID: n.Draft.ZoneID,
				Severity: n.Draft.Severity, ReactionOutcome: n.Draft.ReactionOutcome, ReactionMapRef: n.Draft.ReactionMapRef}
		}
		sig := NCSourceSignal{SignalID: sid, BasisKind: basisKind(d.BasisKind), Severity: severity(d.Severity),
			AnalyzerConfidenceBP: d.AnalyzerConfidenceBP, ObservationQualityBP: d.ObservationQualityBP,
			Stages: []NCAnalyzerStage{}, EvidenceRefs: []string{}}
		sig.DefectTypeCode = optStr(d.DefectTypeCode)
		sig.DefectTypeKnown = d.DefectTypeCode != "" && (d.DefectTypeKnown == nil || *d.DefectTypeKnown)
		sig.ZoneID = optStr(d.ZoneID)
		if rr, ok := signalRef[sid]; ok {
			sig.Record = rr
		}
		if insp != nil {
			var id inspectionData
			if json.Unmarshal(insp.Data, &id) == nil {
				for _, st := range id.Stages {
					x := NCAnalyzerStage{Stage: st.Stage, Version: st.Version, ConfidenceBP: st.ConfidenceBP, OutputNote: st.OutputNote}
					sig.Stages = append(sig.Stages, x)
				}
				if sig.AnalyzerConfidenceBP == nil {
					sig.AnalyzerConfidenceBP = id.AnalyzerConfidenceBP
				}
				if sig.ObservationQualityBP == nil {
					sig.ObservationQualityBP = id.ObservationQualityBP
				}
				if len(id.Versions) > 0 {
					sig.Versions = map[string]string{}
					for k, val := range id.Versions {
						if s, ok := val.(string); ok {
							sig.Versions[k] = s
						}
					}
				}
				for _, er := range id.EvidenceRefs {
					if er.MaterialAddress != "" {
						sig.EvidenceRefs = append(sig.EvidenceRefs, er.MaterialAddress)
					}
				}
			}
			if sig.Record.EventID == "" {
				sig.Record = ref(*insp, summaryOf(*insp))
			}
		}
		if sig.Record.EventID == "" && len(n.Causes) > 0 {
			sig.Record = NCRecordRef{EventID: n.Causes[0], EventType: "", Kind: "fact", OccurredAt: n.FoundAt, Summary: "Основание черновика"}
			if r, ok := v.Record(n.Causes[0]); ok {
				sig.Record = ref(r, summaryOf(r))
			}
		}
		if sig.Record.EventType == "" {
			sig.Record.EventType = string(catalog.DecisionNonconformityDrafted)
			sig.Record.Kind = "reaction"
			sig.Record.OccurredAt = n.FoundAt
		}
		e.Signals = append(e.Signals, sig)
	}
	req := n.RequirementRef
	if req == "" {
		for _, d := range signals {
			if d.RequirementRef != "" {
				req = d.RequirementRef
			}
		}
	}
	if req != "" {
		e.Requirement = &NCRequirement{Characteristic: req}
	}
	zone := n.Draft.ZoneID
	for _, r := range v.Input {
		if r.Type != catalog.InspectionResultRecorded {
			continue
		}
		var d inspectionData
		if json.Unmarshal(r.Data, &d) != nil {
			continue
		}
		if zone == "" || slices.Contains(d.ZoneIDs, zone) {
			e.ZoneHistory = append(e.ZoneHistory, ref(r, summaryOf(r)))
		}
	}
	return e
}

func basisKind(b string) string {
	switch b {
	case "inspection_result", "equipment_deviation", "check_skipped", "damage_on_receipt", "leak", "special_process_violation", "operator_report":
		return b
	}
	return "inspection_result"
}

func outcome(o string) string {
	switch o {
	case "pass_to_next", "manual_review", "isolate", "question_to_technologist":
		return o
	}
	return "manual_review"
}

// analysis — анализ системы и «почему система это предлагает» (FR-51): все
// версии вывода по слотам (сигнал quality, черновик и сдерживание
// nonconformity) с причиной пересмотра (AD-3, FR-32).
func analysis(v *itemView, n dom.NC, h NCHappened) NCSystemAnalysis {
	a := NCSystemAnalysis{Versions: []NCConclusionVersion{}, Why: []string{}, Alternatives: []string{}, MissingInformation: []string{}}
	for _, r := range v.Reactions {
		if r.Reaction == nil {
			continue
		}
		var oc string
		switch r.Record.Type {
		case catalog.QualitySignalRaised:
			var d signalData
			if json.Unmarshal(r.Record.Data, &d) != nil || !slices.Contains(n.Draft.SignalIDs, d.SignalID) {
				continue
			}
			oc = d.ReactionOutcome
		case catalog.DecisionNonconformityDrafted:
			if r.Reaction.Slot.TriggerKey != n.ID {
				continue
			}
			oc = n.Draft.ReactionOutcome
		case catalog.DecisionContainmentApplied:
			if !slices.ContainsFunc(n.Causes, func(id string) bool { return slices.Contains(r.Reaction.Causes, id) }) &&
				r.Reaction.Slot.TriggerKey != "nc:"+n.ID && !slices.Contains(n.Causes, r.Reaction.Slot.TriggerKey) {
				continue
			}
			oc = "isolate"
		default:
			continue
		}
		cv := NCConclusionVersion{Version: max(r.Reaction.Version, 1), EventID: r.Record.EventID, RuleID: r.Reaction.RuleID,
			AutomationMode: min(max(r.Reaction.AutomationMode, 1), 5), Outcome: outcome(oc), RevisedDueTo: r.Reaction.RevisedDueTo,
			RecordedAt: r.RecordedAt, Causes: []NCRecordRef{}}
		if r.Reaction.RuleRev != "" {
			rev := r.Reaction.RuleRev
			cv.RuleRev = &rev
		}
		for _, id := range r.Reaction.Causes {
			if c, ok := v.Record(id); ok {
				cv.Causes = append(cv.Causes, ref(c, summaryOf(c)))
			}
		}
		a.Versions = append(a.Versions, cv)
	}
	slices.SortStableFunc(a.Versions, func(x, y NCConclusionVersion) int {
		if c := x.RecordedAt.Compare(y.RecordedAt); c != 0 {
			return c
		}
		return x.Version - y.Version
	})
	if len(a.Versions) == 0 && n.Origin == dom.OriginSignal {
		// Реакция ещё не записана воркером — вывод свёртки (версия 1).
		cv := NCConclusionVersion{Version: 1, EventID: kernel.Reaction{Slot: kernel.Slot{RuleID: dom.RuleDraft, Subject: "item:" + v.ItemID, TriggerKey: n.ID}}.ID(1),
			RuleID: dom.RuleDraft, AutomationMode: 1, Outcome: outcome(n.Draft.ReactionOutcome), RecordedAt: n.FoundAt, Causes: []NCRecordRef{}}
		for _, id := range n.Causes {
			if c, ok := v.Record(id); ok {
				cv.Causes = append(cv.Causes, ref(c, summaryOf(c)))
			}
		}
		a.Versions = append(a.Versions, cv)
	}
	// Почему система это предлагает.
	switch n.Origin {
	case dom.OriginSpecialProcess:
		a.Why = append(a.Why, "Операция — специальный процесс: нарушение режима оборудования само по себе нарушение техпроцесса (FR-151).",
			"Изделие прошло операцию в окне нарушения — зарегистрировано несоответствие, даже если дефект при контроле не найден.",
			"Решение по изделию принимает комиссия по маршруту подписей.")
	default:
		if n.Draft.ReactionMapRef != "" {
			a.Why = append(a.Why, "Карта реакций "+n.Draft.ReactionMapRef+": «"+outcomeText(n.Draft.ReactionOutcome)+"».")
		}
		if n.Severity != "" && n.Severity != "unknown" {
			a.Why = append(a.Why, "Тяжесть признака — "+severityText(n.Severity)+".")
		}
		if n.Draft.ReactionOutcome == "isolate" {
			a.Why = append(a.Why, "Правило изолирует изделие до решения контролёра: ограничивать автоматически безопаснее, чем разрешать (FR-144).")
		}
		if len(a.Why) == 0 {
			a.Why = append(a.Why, "Признак дефекта по результату контроля — нужно решение контролёра; сигнал ≠ подтверждённое несоответствие.")
		}
		for _, sig := range evidence(v, n).Signals {
			if sig.ObservationQualityBP != nil && *sig.ObservationQualityBP < 5000 {
				a.Alternatives = append(a.Alternatives, "Низкое качество наблюдения — признак может быть артефактом съёмки; рассмотрите доп. проверку.")
			}
			if !sig.DefectTypeKnown {
				a.Alternatives = append(a.Alternatives, "Вид дефекта не определён классификатором — вывод о виде не делается.")
			}
		}
	}
	// Нехватка сведений — перечислением (как в разборе обстоятельств).
	miss := slices.Clone(n.Draft.MissingInformation)
	if op := h.Operation; op != nil {
		if op.PerformerID == nil {
			miss = append(miss, "operator_unknown")
		}
		if op.FinishedAt == nil {
			miss = append(miss, "cycle_end_time_unknown")
		}
		if len(h.During) == 0 {
			miss = append(miss, "equipment_log_missing")
		}
		before, after := false, false
		for _, r := range h.Before {
			before = before || r.EventType == string(catalog.InspectionResultRecorded)
		}
		for _, r := range h.After {
			after = after || r.EventType == string(catalog.InspectionResultRecorded)
		}
		if !before {
			miss = append(miss, "no_observation_before_operation")
		}
		if !after {
			miss = append(miss, "no_observation_after_operation")
		}
	}
	slices.Sort(miss)
	a.MissingInformation = append(a.MissingInformation, slices.Compact(miss)...)
	a.MissingInformationCodes = slices.Clone(a.MissingInformation)
	return a
}

func outcomeText(o string) string {
	switch o {
	case "pass_to_next":
		return "пропустить к следующему контролю"
	case "manual_review":
		return "ручной осмотр"
	case "isolate":
		return "изолировать"
	case "question_to_technologist":
		return "вопрос технологу"
	}
	return o
}

func severityText(s string) string {
	switch s {
	case "critical":
		return "критическая"
	case "major":
		return "значительная"
	case "minor":
		return "малозначительная"
	}
	return "не определена"
}
