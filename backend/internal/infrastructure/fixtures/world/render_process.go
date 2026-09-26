package world

import (
	"fmt"
	"strings"
	"time"

	"ant/internal/application/platform"
	processapp "ant/internal/application/process"
	"ant/internal/infrastructure/fixtures/loader"
)

// BpmnBlob — имя BPMN XML в common/blobs (loader подставляет строку при ответе).
const BpmnBlob = "flange-process.bpmn"

// NodeCount — счётчики узла за период (FR-2, FR-3): сутки на часах шага.
type NodeCount struct {
	Queue, InProgress, Passed, Defects, NCs int
	WaitSum                                 time.Duration
}

func (c *Ctx) dayStart() time.Time {
	l := c.T.In(c.M.clk.loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, c.M.clk.loc).UTC()
}

// Counters — счётчики всех узлов на шаге.
func (c *Ctx) Counters() map[string]*NodeCount {
	out := map[string]*NodeCount{}
	get := func(k string) *NodeCount {
		if out[k] == nil {
			out[k] = &NodeCount{}
		}
		return out[k]
	}
	day := c.dayStart()
	for _, it := range c.Existing() {
		st := c.S(it)
		switch st.Position {
		case "in_progress", "at_inspection":
			get(st.Step).InProgress++
		case "completed":
		default:
			nc := get(st.Step)
			nc.Queue++
			var since time.Time
			for _, mv := range it.moves {
				if !mv.at.After(c.T) && mv.step == st.Step {
					if since.IsZero() {
						since = mv.at
					}
				} else if !mv.at.After(c.T) {
					since = time.Time{}
				}
			}
			if !since.IsZero() {
				nc.WaitSum += c.T.Sub(since)
			}
		}
		for i := 0; i+1 < len(it.moves); i++ {
			a, b := it.moves[i], it.moves[i+1]
			if a.step != b.step && b.at.After(day) && !b.at.After(c.T) {
				get(a.step).Passed++
			}
		}
	}
	for _, n := range c.M.NCs {
		if n.ConfirmedAt.After(c.T) {
			continue
		}
		if !n.ConfirmedAt.Before(day) {
			get(n.StepKey).Defects += len(n.Spec.Defects)
		}
		if n.Status(c.M, c.T) != "verified" {
			get(n.StepKey).NCs++
		}
	}
	return out
}

func (c *Ctx) mapCounters(cs map[string]*NodeCount) []processapp.MapNodeCounters {
	out := []processapp.MapNodeCounters{}
	for _, k := range sortedKeys(cs) {
		v := cs[k]
		mc := processapp.MapNodeCounters{StepKey: k, Queue: v.Queue, InProgress: v.InProgress, Passed: v.Passed, Defects: v.Defects}
		if v.NCs > 0 {
			mc.Nonconformities = ptr(v.NCs)
		}
		out = append(out, mc)
	}
	return out
}

// bottleneck — ограничение линии (FR-5): наибольшая очередь операции или контроля (≥ 3) и среднее ожидание.
func (c *Ctx) bottleneck(cs map[string]*NodeCount) (string, string) {
	best, bq := "", 2
	for _, k := range sortedKeys(cs) {
		n := c.M.Bpmn[k]
		if n == nil || n.Type == "exclusiveGateway" || strings.HasSuffix(k, "send_to_welding") || strings.HasSuffix(k, "send_to_assembly") || strings.HasSuffix(k, "released") || strings.HasSuffix(k, "nonconformity") || strings.HasSuffix(k, "scrapped") {
			continue
		}
		if cs[k].Queue > bq {
			best, bq = k, cs[k].Queue
		}
	}
	if best == "" {
		return "", ""
	}
	avg := cs[best].WaitSum / time.Duration(cs[best].Queue)
	return best, humanDuration(avg)
}

func humanDuration(d time.Duration) string {
	m := int(d.Minutes())
	if m < 60 {
		return fmt.Sprintf("%d мин", m)
	}
	if m < 48*60 {
		return fmt.Sprintf("%d ч %d мин", m/60, m%60)
	}
	return fmt.Sprintf("%d сут %d ч", m/1440, (m%1440)/60)
}

// dataGaps — узлы без данных источника (не «норма»): журнал ИС-2 не приходит (S07).
func (c *Ctx) dataGaps() []string {
	out := []string{}
	for _, s := range c.M.Spec.Sources {
		if !s.Lost.Time().After(c.T) && s.Restored.Time().After(c.T) {
			out = append(out, "welding.weld")
		}
	}
	return out
}

func (c *Ctx) anomalies(cs map[string]*NodeCount) []processapp.MapNodeAnomaly {
	out := []processapp.MapNodeAnomaly{}
	for _, h := range c.M.Spec.ProcessHolds {
		if !h.Set.Time().After(c.T) {
			out = append(out, processapp.MapNodeAnomaly{StepKey: "welding.weld", Kind: "downtime_over_threshold", Threshold: h.Equipment + ": остановка по качеству"})
		}
	}
	for _, k := range sortedKeys(cs) {
		if cs[k].Queue >= 8 && !strings.HasPrefix(k, "final.released") && c.M.Bpmn[k] != nil && c.M.Bpmn[k].Type != "exclusiveGateway" && !strings.Contains(k, "send_to") {
			out = append(out, processapp.MapNodeAnomaly{StepKey: k, Kind: "queue_above_norm", Threshold: "8 изделий"})
		}
	}
	return out
}

// openIncident — последний открытый к шагу инцидент (режим инцидента на карте по умолчанию).
func (c *Ctx) openIncident() *Incident {
	var out *Incident
	for _, in := range c.M.Incidents {
		if !in.Spec.Opened.Time().After(c.T) && (in.Spec.Closed.IsZero() || in.Spec.Closed.Time().After(c.T)) {
			out = in
		}
	}
	return out
}

func (c *Ctx) mapIncident(in *Incident) *processapp.MapIncident {
	v := in.VersionAt(c.T)
	if v == nil {
		return nil
	}
	return &processapp.MapIncident{IncidentID: in.Spec.ID, Label: in.Spec.Label, ScopeVersion: v.Spec.V, SizeAtCreation: in.Versions[0].Size(), Size: v.Size(), Basis: v.Spec.Basis}
}

func (c *Ctx) liveMap(cs map[string]*NodeCount, in *Incident) processapp.LiveMap {
	inWork := 0
	items := []processapp.MapItem{}
	for _, it := range c.Existing() {
		st := c.S(it)
		if st.Position == "completed" {
			continue
		}
		inWork++
		mi := processapp.MapItem{ItemID: FullID(it.ID), Label: it.Label, StepKey: st.Step, Position: st.Position, Summary: st.Summary, ProcessVersionID: ProcessVersionID}
		if in != nil {
			mi.IncidentStatus = st.Incidents[in.Spec.ID]
		}
		items = append(items, mi)
	}
	ver := processapp.MapVersionRef{ProcessVersionID: ProcessVersionID, Label: "v1", IsCurrent: true, Items: inWork}
	lm := processapp.LiveMap{ProcessVersion: ver, Versions: []processapp.MapVersionRef{ver}, BpmnXML: loader.BlobPrefix + BpmnBlob,
		Counters: c.mapCounters(cs), Items: items, Anomalies: c.anomalies(cs), DataGaps: c.dataGaps(), BasisSeq: c.Seq()}
	if k, w := c.bottleneck(cs); k != "" {
		lm.Bottleneck = &processapp.MapBottleneck{StepKey: k, Wait: w}
	}
	if in != nil {
		lm.Incident = c.mapIncident(in)
	}
	return lm
}

// renderProcess — живая карта, карточки узлов, версии процесса (FR-1…FR-5, FR-9, FR-22…FR-24, FR-154).
func renderProcess(c *Ctx) []loader.Response {
	cs := c.Counters()
	out := []loader.Response{resp("process.live_map.read", c.liveMap(cs, c.openIncident()))}
	for _, in := range c.M.Incidents {
		if !in.Spec.Opened.Time().After(c.T) {
			out = append(out, resp("process.live_map.read", c.liveMap(cs, in), "incident_id", in.Spec.ID))
		}
	}
	// Карточки узлов.
	for _, n := range c.M.BpmnOrder {
		cnt := cs[n.StepKey]
		if cnt == nil {
			cnt = &NodeCount{}
		}
		card := processapp.ProcessNodeCard{ProcessVersionID: ProcessVersionID, StepKey: n.StepKey, ElementID: n.ID, Name: n.Name, Kind: nodeKind(n), Lane: n.Lane,
			Documentation: n.Doc, Properties: map[string]any{}, NormRefs: []processapp.NormRef{}, Items: []platform.DrillRef{}, Nonconformities: []platform.DrillRef{},
			Counters: processapp.MapNodeCounters{StepKey: n.StepKey, Queue: cnt.Queue, InProgress: cnt.InProgress, Passed: cnt.Passed, Defects: cnt.Defects}}
		if card.Name == "" {
			card.Name = n.ID
		}
		if cnt.NCs > 0 {
			card.Counters.Nonconformities = ptr(cnt.NCs)
		}
		for k, v := range n.Props {
			card.Properties[k] = v
		}
		for _, nr := range n.Norms {
			card.NormRefs = append(card.NormRefs, processapp.NormRef{Standard: nr.Standard, Clause: nr.Clause, Check: nr.Check, SystemAction: nr.Action})
		}
		for _, it := range c.Existing() {
			if st := c.S(it); st.Step == n.StepKey && st.Position != "completed" {
				card.Items = append(card.Items, platform.DrillRef{Entity: platform.EntityItem, ID: FullID(it.ID)})
			}
		}
		for _, nc := range c.M.NCs {
			if nc.StepKey == n.StepKey && !nc.ConfirmedAt.After(c.T) && nc.Status(c.M, c.T) != "verified" {
				card.Nonconformities = append(card.Nonconformities, platform.DrillRef{Entity: platform.EntityNonconformity, ID: nc.ID})
			}
		}
		out = append(out, resp("process.node.read", card, "version_id", ProcessVersionID, "step_key", n.StepKey))
	}
	if c.N == 0 {
		out = append(out, c.versions()...)
	}
	// Изделий в работе по версии меняется — список версий на каждом шаге.
	inWork := 0
	for _, it := range c.Existing() {
		if c.S(it).Position != "completed" {
			inWork++
		}
	}
	created := c.M.clk.at(14, 9, 0)
	sum := processapp.ProcessVersionSummary{VersionID: ProcessVersionID, Label: "v1", Status: "active", Hash: c.M.BpmnDigest, CreatedAt: created, EffectiveFrom: tptr(created), Quorum: &processapp.VersionQuorum{Have: 3, Need: 3}, ItemsInWork: inWork}
	out = append(out, resp("process.version.list", processapp.ProcessVersionList{Items: []processapp.ProcessVersionSummary{sum}}))
	return out
}

// versions — версия процесса в читаемом виде, разница и BPMN как загружен (неизменны).
func (c *Ctx) versions() []loader.Response {
	created := c.M.clk.at(14, 9, 0)
	v := processapp.ProcessVersion{VersionID: ProcessVersionID, Label: "v1", Status: "active", Hash: c.M.BpmnDigest, Author: ptr("TEC-01"), CreatedAt: created,
		EffectiveFrom: tptr(created), Quorum: &processapp.VersionQuorum{Have: 3, Need: 3}, Elements: []processapp.ProcessElement{}, BasisSeq: c.Seq()}
	for _, n := range c.M.BpmnOrder {
		e := processapp.ProcessElement{ID: n.ID, StepKey: ptr(n.StepKey), Kind: nodeKind(n), Name: n.Name, Properties: map[string]any{}, Next: n.Next}
		if n.Lane != "" {
			e.Lane = ptr(n.Lane)
		}
		for k, val := range n.Props {
			e.Properties[k] = val
		}
		v.Elements = append(v.Elements, e)
	}
	return []loader.Response{
		resp("process.version.read", v, "version_id", ProcessVersionID),
		resp("process.version.diff", processapp.ProcessVersionDiff{VersionID: ProcessVersionID, AgainstID: ProcessVersionID, Entries: []processapp.ProcessDiffEntry{}}, "version_id", ProcessVersionID),
		resp("process.version.bpmn", processapp.ProcessBpmn{VersionID: ProcessVersionID, Hash: c.M.BpmnDigest, BpmnXML: loader.BlobPrefix + BpmnBlob}, "version_id", ProcessVersionID),
	}
}
