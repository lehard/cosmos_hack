package process

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/platform"
	"ant/internal/contracts/bpmnext"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	dp "ant/internal/domain/process"
)

// ProjectionReader — ведомый порт чтения проекций (engineapp.ProjectionStore):
// значение проекции по ключу; запись — только эффектами движка (AD-45).
type ProjectionReader interface {
	Get(ctx context.Context, name, key string) (json.RawMessage, bool, error)
}

// ItemStates — ведомый порт состояния изделия на момент (AD-22): та же
// свёртка, что у воркера, над префиксом входа (engineapp.StateQueries с Bundles).
type ItemStates interface {
	Item(ctx context.Context, itemID string, m platform.Moment) (engineapp.ItemAt, error)
}

// Fact — факт исполнителя или мастера с терминала (FR-137) для записи через
// приём (FR-141: ручной ввод — полноправный источник с пометкой).
type Fact struct {
	Meta       platform.CommandMeta
	EventType  catalog.Type
	ItemID     string
	OccurredAt *time.Time
	Data       map[string]any
}

// FactWriter — ведомый порт записи факта через конвейер приёма (схема,
// дубли, привязка к изделию, журнал): адаптер над ingest в cmd/ant.
type FactWriter interface {
	Submit(ctx context.Context, f Fact) (platform.Receipt, error)
}

// NodeCounterSet — счётчики узлов, ограничение линии, аномалии и узлы без
// данных за период (FR-2, FR-3, FR-5).
type NodeCounterSet struct {
	Counters   []MapNodeCounters
	Bottleneck *MapBottleneck
	Anomalies  []MapNodeAnomaly
	DataGaps   []string
	BasisSeq   int64
}

// NodeCounterSource — ведомый порт счётчиков узлов живой карты: их считает
// analytics по строкам вклада изделий (AD-21, AD-45, эпик 25) — те же
// числа, что на вкладке «Аналитика» и в контрольных картах. nil — счётчики
// считает process по положению токенов (режим без analytics).
type NodeCounterSource interface {
	NodeCounters(ctx context.Context, versionID string, q LiveMapQuery, m platform.Moment) (NodeCounterSet, error)
}

// DomainClock — доменное «сейчас» при приёме команды (AD-37).
type DomainClock func(ctx context.Context) (time.Time, error)

// LiveService — реализация live ведущих портов модуля process (AD-36): живая
// карта и карточки узлов — по проекциям изделий (сейчас) или свёрткой на
// момент (AD-22); версии — из хранилища версий; черновик версии — проверка
// при загрузке (FR-13); операции и перемещения — гард над свёрткой изделия
// (AD-39) и факт через приём.
type LiveService struct {
	Unimplemented
	Store   ProjectionReader
	States  ItemStates
	Library VersionStore
	Bundles *Bundles
	Facts   FactWriter
	Clock   DomainClock
	// Counters — счётчики узлов от analytics; nil — свои по токенам.
	Counters NodeCounterSource
	// Recorder — запись решений normative.version.* в журнал; nil — не пишутся.
	Recorder *Recorder
}

var (
	_ Queries  = (*LiveService)(nil)
	_ Commands = (*LiveService)(nil)
)

func get[T any](ctx context.Context, s ProjectionReader, name, k string) (T, bool, error) {
	var v T
	raw, ok, err := s.Get(ctx, name, k)
	if err != nil || !ok {
		return v, ok, err
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, false, fmt.Errorf("проекция %s[%s]: %w", name, k, err)
	}
	return v, true, nil
}

func notFound(object, id string) error {
	e := platform.Fail(errcodes.ApiNotFound, "object", object, "id", id)
	e.Detail = object + " «" + id + "» не найден"
	return e
}

// items — представления изделий прогона на момент m.
func (s *LiveService) items(ctx context.Context, m platform.Moment) ([]ItemView, error) {
	idx, _, err := get[Index](ctx, s.Store, ProjectionIndex, IndexKey(m.RunID))
	if err != nil {
		return nil, err
	}
	out := make([]ItemView, 0, len(idx.Items))
	for _, e := range idx.Items {
		if m.AsOf != nil && e.RegisteredAt.After(*m.AsOf) {
			continue
		}
		if m.AsOf == nil {
			v, ok, err := get[ItemView](ctx, s.Store, ProjectionItem, e.ItemID)
			if err != nil {
				return nil, err
			}
			if ok {
				out = append(out, v)
			}
			continue
		}
		at, err := s.States.Item(ctx, e.ItemID, m)
		if err != nil {
			if pe, ok := platform.AsError(err); ok && pe.Code == errcodes.ApiNotFound {
				continue
			}
			return nil, err
		}
		if at.Snapshot.Process.StartedAt == nil && at.Snapshot.Process.VersionHash == "" {
			continue
		}
		out = append(out, ViewOf(e.ItemID, at.Snapshot))
	}
	return out, nil
}

// version — запись версии по id; пусто — действующая.
func (s *LiveService) version(ctx context.Context, id string) (VersionRecord, error) {
	if id == "" {
		vs, err := s.Library.List(ctx)
		if err != nil {
			return VersionRecord{}, err
		}
		if v, ok := Active(vs); ok {
			return v, nil
		}
		return VersionRecord{}, notFound("Действующая версия процесса", "active")
	}
	v, ok, err := s.Library.ByID(ctx, id)
	if err != nil {
		return v, err
	}
	if !ok {
		return v, notFound("Версия процесса", id)
	}
	return v, nil
}

// definition — модель версии для показа (без проверки подписей: карта и
// карточка показывают версию как хранится; исполнение проверяет Bundles).
func definition(v VersionRecord) (*dp.Definition, error) {
	d, vs := dp.Parse(v.XML)
	if d == nil {
		return nil, fmt.Errorf("версия %s: %v", v.ID, vs)
	}
	return d, nil
}

// period — границы периода счётчиков (FR-3) по доменному времени: конец —
// момент запроса или самое позднее время изделий.
func period(q LiveMapQuery, m platform.Moment, views []ItemView) (time.Time, time.Time) {
	var to time.Time
	for _, v := range views {
		if v.Now.After(to) {
			to = v.Now
		}
	}
	if m.AsOf != nil {
		to = *m.AsOf
	}
	if q.To != nil {
		to = *q.To
	}
	var span time.Duration
	switch q.Period {
	case "day":
		span = 24 * time.Hour
	case "week":
		span = 7 * 24 * time.Hour
	case "month":
		span = 30 * 24 * time.Hour
	default:
		span = 8 * time.Hour // смена
	}
	from := to.Add(-span)
	if q.From != nil {
		from = *q.From
	}
	return from, to
}

// nodeStat — счётчики узла по изделиям (FR-2): очередь, в работе, выполнено
// за период, признаки дефекта; ожидание в очереди — для ограничения и аномалий.
type nodeStat struct {
	queue, inProgress, passed, defects int
	wait                               time.Duration
	maxWait                            time.Duration
	items                              []string
}

func stats(views []ItemView, from, to time.Time) map[string]*nodeStat {
	out := map[string]*nodeStat{}
	at := func(k string) *nodeStat {
		if out[k] == nil {
			out[k] = &nodeStat{}
		}
		return out[k]
	}
	for _, v := range views {
		for _, t := range v.Tokens {
			if t.Join {
				continue
			}
			st := at(t.StepKey)
			st.items = append(st.items, v.ItemID)
			switch t.Phase {
			case dp.PhaseInProgress, dp.PhasePaused, dp.PhaseInTransit:
				st.inProgress++
			default:
				st.queue++
				w := v.Now.Sub(t.Since)
				st.wait += w
				st.maxWait = max(st.maxWait, w)
			}
		}
		for _, h := range v.Visits {
			if h.Left != nil && h.Via == "completed" && !h.Left.Before(from) && !h.Left.After(to) {
				at(h.StepKey).passed++
			}
		}
		for step, n := range v.Defects {
			at(step).defects += n
		}
	}
	return out
}

func summaryOf(v ItemView) string {
	switch {
	case v.Completed && v.Outcome == "released":
		return "released"
	case v.Completed:
		return "scrapped"
	case v.Containment == "block":
		return "hold"
	case v.Isolated || v.Primary != nil && v.Primary.Position == dp.PosIsolated:
		return "pending_decision"
	}
	return "in_process"
}

func label(itemID string) string {
	if _, local, ok := strings.Cut(itemID, ":"); ok {
		return local
	}
	return itemID
}

// LiveMap — живая карта на момент (process.live_map.read, FR-1…FR-5, FR-130;
// AD-17: счётчики по step_key, изделия прежних версий — на узлах своего
// step_key с пометкой версии).
func (s *LiveService) LiveMap(ctx context.Context, q LiveMapQuery, m platform.Moment) (LiveMap, error) {
	if s.Store == nil || s.Library == nil {
		return LiveMap{}, platform.NotImplemented("process.live_map.read")
	}
	shown, err := s.version(ctx, q.ProcessVersionID)
	if err != nil {
		return LiveMap{}, err
	}
	def, err := definition(shown)
	if err != nil {
		return LiveMap{}, err
	}
	views, err := s.items(ctx, m)
	if err != nil {
		return LiveMap{}, err
	}
	from, to := period(q, m, views)
	st := stats(views, from, to)
	lm := LiveMap{BpmnXML: string(shown.XML), Counters: []MapNodeCounters{}, Items: []MapItem{}, Anomalies: []MapNodeAnomaly{}, DataGaps: []string{}}
	all, err := s.Library.List(ctx)
	if err != nil {
		return LiveMap{}, err
	}
	active, _ := Active(all)
	inWork := map[string]int{}
	gaps := map[string]bool{}
	for _, v := range views {
		lm.BasisSeq = max(lm.BasisSeq, v.BasisSeq)
		for _, g := range v.Gaps {
			gaps[g.StepKey] = true
		}
		if v.Completed || v.Primary == nil {
			continue
		}
		inWork[v.VersionID]++
		lm.Items = append(lm.Items, MapItem{ItemID: v.ItemID, Label: label(v.ItemID), StepKey: v.Primary.StepKey, Position: v.Primary.Position,
			Summary: summaryOf(v), ProcessVersionID: v.VersionID})
	}
	for _, v := range all {
		if v.Status != dp.StatusActive && v.Status != dp.StatusRetired && inWork[v.ID] == 0 {
			continue
		}
		ref := MapVersionRef{ProcessVersionID: v.ID, Label: v.Label, IsCurrent: v.ID == active.ID, Items: inWork[v.ID]}
		lm.Versions = append(lm.Versions, ref)
		if v.ID == shown.ID {
			lm.ProcessVersion = ref
		}
	}
	if lm.ProcessVersion.ProcessVersionID == "" {
		lm.ProcessVersion = MapVersionRef{ProcessVersionID: shown.ID, Label: shown.Label, IsCurrent: shown.ID == active.ID, Items: inWork[shown.ID]}
		lm.Versions = append(lm.Versions, lm.ProcessVersion)
	}
	var best string
	var bestScore time.Duration
	for _, id := range def.Order {
		n := def.Nodes[id]
		k := n.StepKey()
		if k == "" {
			continue
		}
		c := st[k]
		if c == nil {
			c = &nodeStat{}
		}
		lm.Counters = append(lm.Counters, MapNodeCounters{StepKey: k, Queue: c.queue, InProgress: c.inProgress, Passed: c.passed, Defects: c.defects})
		if c.queue > 0 {
			// FR-5: ограничение — наибольшее ожидание при наибольшей загрузке.
			if score := c.wait; score > bestScore {
				best, bestScore = k, score
			}
			if n.Norm != nil && n.Norm.QueueNormMinutes != nil && c.maxWait > time.Duration(*n.Norm.QueueNormMinutes)*time.Minute {
				lm.Anomalies = append(lm.Anomalies, MapNodeAnomaly{StepKey: k, Kind: "queue_above_norm", Threshold: fmt.Sprintf("%d мин", *n.Norm.QueueNormMinutes)})
			}
			if p := n.Presentation; p != nil && p.WaitLimitMinutes != nil && c.maxWait > time.Duration(*p.WaitLimitMinutes)*time.Minute {
				lm.Anomalies = append(lm.Anomalies, MapNodeAnomaly{StepKey: k, Kind: "wait_above_norm", Threshold: fmt.Sprintf("%d мин", *p.WaitLimitMinutes)})
			}
		}
		if gaps[k] {
			lm.DataGaps = append(lm.DataGaps, k)
		}
	}
	if best != "" {
		c := st[best]
		lm.Bottleneck = &MapBottleneck{StepKey: best, Wait: fmt.Sprintf("%d мин", int((c.wait / time.Duration(c.queue)).Minutes()))}
	}
	if s.Counters != nil {
		// Счётчики, ограничение и аномалии — от analytics (одни числа на карте
		// и в аналитике); пропуски данных — объединение с пропусками исполнителя.
		ncs, err := s.Counters.NodeCounters(ctx, shown.ID, q, m)
		if err != nil {
			return LiveMap{}, err
		}
		lm.Counters, lm.Bottleneck, lm.Anomalies = ncs.Counters, ncs.Bottleneck, ncs.Anomalies
		if lm.Counters == nil {
			lm.Counters = []MapNodeCounters{}
		}
		if lm.Anomalies == nil {
			lm.Anomalies = []MapNodeAnomaly{}
		}
		for _, g := range ncs.DataGaps {
			if !slices.Contains(lm.DataGaps, g) {
				lm.DataGaps = append(lm.DataGaps, g)
			}
		}
		lm.BasisSeq = max(lm.BasisSeq, ncs.BasisSeq)
	}
	return lm, nil
}

// nodeKind — вид узла для карточки и читаемой версии (перечисление контракта).
func nodeKind(n *dp.Node) string {
	switch n.Type {
	case dp.NodeServiceTask:
		return "automatedInspection"
	case dp.NodeUserTask:
		if n.Props.StepKind == "human_inspection" {
			return "humanInspection"
		}
		return "operation"
	case dp.NodeTask:
		return "operation"
	case dp.NodeParallelGateway:
		return "parallelGateway"
	case dp.NodeInclusiveGateway:
		return "inclusiveGateway"
	case dp.NodeExclusiveGateway:
		return "conditionalFlow"
	case dp.NodeStartEvent:
		return "startEvent"
	case dp.NodeEndEvent:
		return "endEvent"
	case dp.NodeBoundaryEvent:
		return "timer"
	case dp.NodeCallActivity:
		return "callActivity"
	case dp.NodeSubProcess:
		return "subprocess"
	}
	return "intermediateEvent"
}

// properties — наши свойства узла (ant:properties) с именами атрибутов XML.
func properties(p bpmnext.Properties) map[string]any {
	out := map[string]any{}
	put := func(k, v string) {
		if v != "" {
			out[k] = v
		}
	}
	put("stepKind", p.StepKind)
	put("workshop", p.Workshop)
	put("warehouse", p.Warehouse)
	put("closesZoneAccess", p.ClosesZoneAccess)
	put("paperAttester", p.PaperAttester)
	put("closingPoint", p.ClosingPoint)
	put("inspectionPoint", p.InspectionPoint)
	put("operationCode", p.OperationCode)
	put("reworkLimitScope", p.ReworkLimitScope)
	put("erpAction", p.ErpAction)
	put("triggerEventType", p.TriggerEventType)
	put("timerScope", p.TimerScope)
	put("outcome", p.Outcome)
	if p.SpecialProcess != nil {
		out["specialProcess"] = *p.SpecialProcess
	}
	if p.BufferPlace != nil {
		out["bufferPlace"] = *p.BufferPlace
	}
	if p.ReworkLimit != nil {
		out["reworkLimit"] = *p.ReworkLimit
	}
	return out
}

// Node — карточка узла (process.node.read, FR-154, FR-156).
func (s *LiveService) Node(ctx context.Context, versionID, stepKey string, m platform.Moment) (ProcessNodeCard, error) {
	if s.Store == nil || s.Library == nil {
		return ProcessNodeCard{}, platform.NotImplemented("process.node.read")
	}
	v, err := s.version(ctx, versionID)
	if err != nil {
		return ProcessNodeCard{}, err
	}
	def, err := definition(v)
	if err != nil {
		return ProcessNodeCard{}, err
	}
	n := def.ByStep(stepKey)
	if n == nil {
		return ProcessNodeCard{}, notFound("Шаг процесса", stepKey)
	}
	views, err := s.items(ctx, m)
	if err != nil {
		return ProcessNodeCard{}, err
	}
	from, to := period(LiveMapQuery{}, m, views)
	c := stats(views, from, to)[stepKey]
	if c == nil {
		c = &nodeStat{}
	}
	card := ProcessNodeCard{ProcessVersionID: v.ID, StepKey: stepKey, ElementID: n.ID, Name: n.Name, Kind: nodeKind(n), Lane: def.LaneName(n),
		Documentation: n.Documentation, Properties: properties(n.Props), NormRefs: []NormRef{},
		Counters: MapNodeCounters{StepKey: stepKey, Queue: c.queue, InProgress: c.inProgress, Passed: c.passed, Defects: c.defects},
		Items:    []platform.DrillRef{}, Nonconformities: []platform.DrillRef{}}
	if card.Name == "" {
		card.Name = n.ID
	}
	if ws := def.Workshop(n); ws != "" {
		card.Properties["workshop"] = ws
	}
	for _, nr := range n.NormRefs {
		card.NormRefs = append(card.NormRefs, NormRef{Standard: nr.Standard, Clause: nr.Clause, Check: nr.Check, SystemAction: nr.SystemAction})
	}
	for _, id := range slices.Compact(sortedStrings(c.items)) {
		card.Items = append(card.Items, platform.DrillRef{Entity: platform.EntityItem, ID: id})
	}
	return card, nil
}

func (s *LiveService) quorum(ctx context.Context, v VersionRecord) (*VersionQuorum, error) {
	q := QuorumVerifier(RecordedQuorum{})
	if s.Bundles != nil && s.Bundles.Quorum != nil {
		q = s.Bundles.Quorum
	}
	a, err := q.Approval(ctx, v)
	if err != nil {
		return nil, err
	}
	have, need := a.Have()
	return &VersionQuorum{Have: have, Need: need}, nil
}

// Versions — версии процесса (process.version.list, FR-22).
func (s *LiveService) Versions(ctx context.Context, m platform.Moment) (ProcessVersionList, error) {
	if s.Library == nil {
		return ProcessVersionList{}, platform.NotImplemented("process.version.list")
	}
	vs, err := s.Library.List(ctx)
	if err != nil {
		return ProcessVersionList{}, err
	}
	inWork := map[string]int{}
	if s.Store != nil {
		views, err := s.items(ctx, m)
		if err != nil {
			return ProcessVersionList{}, err
		}
		for _, v := range views {
			if !v.Completed {
				inWork[v.VersionID]++
			}
		}
	}
	out := ProcessVersionList{Items: []ProcessVersionSummary{}}
	for _, v := range vs {
		q, err := s.quorum(ctx, v)
		if err != nil {
			return out, err
		}
		out.Items = append(out.Items, ProcessVersionSummary{VersionID: v.ID, Label: v.Label, Status: v.Status, Hash: v.Hash, CreatedAt: v.CreatedAt,
			EffectiveFrom: v.EffectiveFrom, Quorum: q, ItemsInWork: inWork[v.ID]})
	}
	return out, nil
}

// Version — версия в читаемом виде (process.version.read, FR-24).
func (s *LiveService) Version(ctx context.Context, versionID string, m platform.Moment) (ProcessVersion, error) {
	_ = m
	if s.Library == nil {
		return ProcessVersion{}, platform.NotImplemented("process.version.read")
	}
	v, err := s.version(ctx, versionID)
	if err != nil {
		return ProcessVersion{}, err
	}
	def, err := definition(v)
	if err != nil {
		return ProcessVersion{}, err
	}
	q, err := s.quorum(ctx, v)
	if err != nil {
		return ProcessVersion{}, err
	}
	out := ProcessVersion{VersionID: v.ID, Label: v.Label, Status: v.Status, Hash: v.Hash, CreatedAt: v.CreatedAt, EffectiveFrom: v.EffectiveFrom,
		Quorum: q, Elements: []ProcessElement{}}
	if v.Author != "" {
		a := v.Author
		out.Author = &a
	}
	for _, id := range def.Order {
		n := def.Nodes[id]
		e := ProcessElement{ID: n.ID, Kind: nodeKind(n), Name: n.Name, Properties: properties(n.Props), Next: def.Next(n)}
		if k := n.StepKey(); k != "" {
			e.StepKey = &k
		}
		if l := def.LaneName(n); l != "" {
			e.Lane = &l
		}
		out.Elements = append(out.Elements, e)
	}
	return out, nil
}

// Diff — разница версий по step_key (process.version.diff, FR-24).
func (s *LiveService) Diff(ctx context.Context, versionID, againstID string, m platform.Moment) (ProcessVersionDiff, error) {
	_ = m
	if s.Library == nil {
		return ProcessVersionDiff{}, platform.NotImplemented("process.version.diff")
	}
	v, err := s.version(ctx, versionID)
	if err != nil {
		return ProcessVersionDiff{}, err
	}
	base, err := s.version(ctx, againstID)
	if err != nil {
		return ProcessVersionDiff{}, err
	}
	nd, err := definition(v)
	if err != nil {
		return ProcessVersionDiff{}, err
	}
	od, err := definition(base)
	if err != nil {
		return ProcessVersionDiff{}, err
	}
	return ProcessVersionDiff{VersionID: v.ID, AgainstID: base.ID, Entries: DiffDefinitions(od, nd)}, nil
}

// DiffDefinitions — читаемая разница: добавленные и удалённые шаги, новые
// точки предъявления, изменённые свойства (по step_key, AD-17).
func DiffDefinitions(old, new *dp.Definition) []ProcessDiffEntry {
	out := []ProcessDiffEntry{}
	name := func(n *dp.Node) string {
		if n.Name != "" {
			return n.Name
		}
		return n.StepKey()
	}
	for _, k := range sortedKeys(new.Steps) {
		nn := new.ByStep(k)
		on := old.ByStep(k)
		if on == nil {
			if nn.IsPresentationPoint() {
				out = append(out, ProcessDiffEntry{Kind: "presentationPointAdded", Step: name(nn)})
			} else {
				out = append(out, ProcessDiffEntry{Kind: "elementAdded", Element: name(nn)})
			}
			continue
		}
		if nn.IsPresentationPoint() && !on.IsPresentationPoint() {
			out = append(out, ProcessDiffEntry{Kind: "presentationPointAdded", Step: name(nn)})
		}
		op, np := properties(on.Props), properties(nn.Props)
		keys := map[string]bool{}
		for k := range op {
			keys[k] = true
		}
		for k := range np {
			keys[k] = true
		}
		for _, pk := range sortedKeys(keys) {
			if fmt.Sprint(op[pk]) != fmt.Sprint(np[pk]) {
				out = append(out, ProcessDiffEntry{Kind: "propertyChanged", Element: name(nn), Property: pk, From: op[pk], To: np[pk]})
			}
		}
	}
	for _, k := range sortedKeys(old.Steps) {
		if new.ByStep(k) == nil {
			out = append(out, ProcessDiffEntry{Kind: "elementRemoved", Element: name(old.ByStep(k))})
		}
	}
	return out
}

// Bpmn — BPMN XML версии как хранится и её хеш (process.version.bpmn, AD-17).
func (s *LiveService) Bpmn(ctx context.Context, versionID string) (ProcessBpmn, error) {
	if s.Library == nil {
		return ProcessBpmn{}, platform.NotImplemented("process.version.bpmn")
	}
	v, err := s.version(ctx, versionID)
	if err != nil {
		return ProcessBpmn{}, err
	}
	return ProcessBpmn{VersionID: v.ID, Hash: v.Hash, BpmnXML: string(v.XML)}, nil
}

// ── команды ──

// DraftVersion — черновик версии из редактора (process.version.draft, FR-22,
// FR-25): проверка при загрузке (FR-13) — отказ с кодом и id элемента;
// байты XML — в хранилище версий модуля (id черновика — «draft-‹12 hex
// хеша›»), в журнал — решение normative.version.drafted в поток версии.
func (s *LiveService) DraftVersion(ctx context.Context, in DraftVersion) (platform.Receipt, error) {
	if s.Library == nil {
		return platform.Receipt{}, platform.NotImplemented("process.version.draft")
	}
	xml := []byte(in.BpmnXML)
	if _, vs := dp.Load(xml, dp.LoadOptions{}); len(dp.Errors(vs)) > 0 {
		return platform.Receipt{}, LoadError(dp.Errors(vs))
	}
	hash := dp.VersionHash(xml)
	id := "draft-" + strings.TrimPrefix(hash, "streebog256:")[:12]
	now := time.Now().UTC()
	if s.Clock != nil {
		t, err := s.Clock(ctx)
		if err != nil {
			return platform.Receipt{}, err
		}
		now = t
	}
	rec := VersionRecord{ID: id, Label: in.Label, Status: dp.StatusDraft, BaseVersionID: in.BaseVersionID, Author: platform.PrincipalFrom(ctx).PersonID,
		Hash: hash, XML: xml, CreatedAt: now}
	if err := s.Library.Save(ctx, rec); err != nil {
		return platform.Receipt{}, err
	}
	if s.Recorder == nil {
		return platform.Receipt{CommandID: in.CommandID, EventIDs: []string{id}, RecordedAt: now}, nil
	}
	data := map[string]any{"version_id": id, "label": in.Label, "process_version_hash": hash}
	if in.BaseVersionID != "" {
		data["base_version_id"] = in.BaseVersionID
	}
	return s.Recorder.Record(ctx, catalog.NormativeVersionDrafted, VersionStream(id), rec.Author, in.CommandMeta(), now, data)
}

// LoadError — отказ загрузки описания (FR-13): код и id первого нарушения,
// в пояснении — все нарушения «код [элемент]: причина».
func LoadError(vs []dp.Violation) error {
	first := vs[0]
	e := platform.Fail(first.Code, "element", first.Element, "detail", first.Detail, "reason", first.Detail)
	parts := make([]string, 0, len(vs))
	for _, v := range vs {
		parts = append(parts, v.String())
	}
	e.Detail = strings.Join(parts, "; ")
	return e
}

func (s *LiveService) now(ctx context.Context) (time.Time, error) {
	if s.Clock != nil {
		return s.Clock(ctx)
	}
	return time.Now().UTC(), nil
}

// guard — гард команды над свёрткой изделия на момент приёма (AD-39).
func (s *LiveService) guard(ctx context.Context, itemID, action string, payload any) error {
	now, err := s.now(ctx)
	if err != nil {
		return err
	}
	at, err := s.States.Item(ctx, itemID, platform.Moment{})
	if err != nil {
		return err
	}
	st := at.Snapshot.Process
	return dp.Guard(st, st.Env(), dp.Upstream{Item: &at.Snapshot.Item}, kernel.Command{Action: action, OccurredAt: now, Payload: payload})
}

func (s *LiveService) submit(ctx context.Context, meta platform.CommandMeta, t catalog.Type, itemID string, data map[string]any) (platform.Receipt, error) {
	if s.Facts == nil {
		return platform.Receipt{}, platform.NotImplemented(string(t))
	}
	for k, v := range data {
		if v == "" {
			delete(data, k)
		}
	}
	return s.Facts.Submit(ctx, Fact{Meta: meta, EventType: t, ItemID: itemID, Data: data})
}

func person(ctx context.Context) any {
	if p := platform.PrincipalFrom(ctx).PersonID; p != "" {
		return p
	}
	return nil
}

func personRef(ctx context.Context) string {
	if p := platform.PrincipalFrom(ctx).PersonID; p != "" {
		return p
	}
	return "unknown"
}

// StartOperation — начать операцию (process.operation.start, FR-137): гард
// предусловий (FR-17), лимита доработок (FR-18), точки предъявления (FR-19),
// скрытых работ (FR-20); факт — через приём.
func (s *LiveService) StartOperation(ctx context.Context, itemID string, in StartOperation) (platform.Receipt, error) {
	if s.States == nil {
		return platform.Receipt{}, platform.NotImplemented("process.operation.start")
	}
	who, _ := person(ctx).(string)
	if err := s.guard(ctx, itemID, "process.operation.start", dp.StartCommand{StepKey: in.StepKey, RunID: in.OperationRunID, ReworkOf: in.ReworkOf,
		OperatorID: who, EquipmentID: in.EquipmentID}); err != nil {
		return platform.Receipt{}, err
	}
	return s.submit(ctx, in.CommandMeta(), catalog.OperationRunStarted, itemID, map[string]any{
		"operation_run_id": in.OperationRunID, "operation_code": in.OperationCode, "step_key": in.StepKey, "station_id": in.StationID,
		"equipment_id": in.EquipmentID, "program_ref": in.ProgramRef, "rework_of": in.ReworkOf, "group_id": in.GroupID, "operator_id": person(ctx),
		"workplace_id": in.WorkplaceID})
}

// runItem — изделие выполнения операции (проекция process.runs).
func (s *LiveService) runItem(ctx context.Context, runID string) (string, error) {
	ref, ok, err := get[RunRef](ctx, s.Store, ProjectionRuns, runID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", notFound("Выполнение операции", runID)
	}
	return ref.ItemID, nil
}

func (s *LiveService) runCmd(ctx context.Context, action string, t catalog.Type, runID string, meta platform.CommandMeta, data map[string]any) (platform.Receipt, error) {
	if s.States == nil || s.Store == nil {
		return platform.Receipt{}, platform.NotImplemented(action)
	}
	itemID, err := s.runItem(ctx, runID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := s.guard(ctx, itemID, action, dp.RunCommand{RunID: runID}); err != nil {
		return platform.Receipt{}, err
	}
	data["operation_run_id"] = runID
	return s.submit(ctx, meta, t, itemID, data)
}

// PauseOperation — пауза (process.operation.pause).
func (s *LiveService) PauseOperation(ctx context.Context, runID string, in PauseOperation) (platform.Receipt, error) {
	return s.runCmd(ctx, "process.operation.pause", catalog.OperationRunPaused, runID, in.CommandMeta(), map[string]any{"pause_reason": in.PauseReason, "note": in.Note})
}

// ResumeOperation — продолжение (process.operation.resume).
func (s *LiveService) ResumeOperation(ctx context.Context, runID string, in ResumeOperation) (platform.Receipt, error) {
	return s.runCmd(ctx, "process.operation.resume", catalog.OperationRunResumed, runID, in.CommandMeta(), map[string]any{})
}

// FinishOperation — конец операции (process.operation.finish).
func (s *LiveService) FinishOperation(ctx context.Context, runID string, in FinishOperation) (platform.Receipt, error) {
	return s.runCmd(ctx, "process.operation.finish", catalog.OperationRunFinished, runID, in.CommandMeta(), map[string]any{"completion": in.Completion})
}

// SendMovement — отправить изделие (process.movement.send, FR-16).
func (s *LiveService) SendMovement(ctx context.Context, itemID string, in SendMovement) (platform.Receipt, error) {
	if s.States == nil {
		return platform.Receipt{}, platform.NotImplemented("process.movement.send")
	}
	if err := s.guard(ctx, itemID, "process.movement.send", dp.MoveCommand{StepKey: in.StepKey}); err != nil {
		return platform.Receipt{}, err
	}
	return s.submit(ctx, in.CommandMeta(), catalog.OperationMovementSent, itemID, map[string]any{
		"from_location_id": in.FromLocationID, "to_location_id": in.ToLocationID, "container_id": in.ContainerID, "step_key": in.StepKey,
		"sent_by": personRef(ctx)})
}

// ReceiveMovement — принять изделие (process.movement.receive, FR-16, FR-137).
func (s *LiveService) ReceiveMovement(ctx context.Context, itemID string, in ReceiveMovement) (platform.Receipt, error) {
	if s.States == nil {
		return platform.Receipt{}, platform.NotImplemented("process.movement.receive")
	}
	if err := s.guard(ctx, itemID, "process.movement.receive", dp.MoveCommand{StepKey: in.StepKey, DestinationKind: in.DestinationKind}); err != nil {
		return platform.Receipt{}, err
	}
	return s.submit(ctx, in.CommandMeta(), catalog.OperationMovementReceived, itemID, map[string]any{
		"from_location_id": in.FromLocationID, "to_location_id": in.ToLocationID, "destination_kind": in.DestinationKind,
		"inspection_on_receipt": in.InspectionOnReceipt, "step_key": in.StepKey, "received_by": personRef(ctx)})
}
