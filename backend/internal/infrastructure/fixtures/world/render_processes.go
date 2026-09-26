package world

import (
	_ "embed"

	"ant/internal/application/platform"
	processapp "ant/internal/application/process"
	"ant/internal/infrastructure/fixtures/loader"
)

// Выбор процесса (UI-11): в мире заготовок два процесса — основной процесс
// фланца (normative/process) и пример второго, кронштейна, только для
// показа выбора. Кронштейн изделий не имеет: карта пустая, версия одна.

// Процессы мира: id главного bpmn:process и название.
const (
	FlangeProcessID   = "Process_Flange"
	FlangeProcessName = "Фланец люка гермокорпуса в сборе"
	BracketProcessID  = "Process_Bracket"
	BracketProcess    = "Кронштейн крепления приборной панели"
	BracketVersionID  = "bracket-1"
	BracketBpmnBlob   = "bracket-process.bpmn"
)

// BracketBpmn — BPMN второго процесса-примера (не нормативный слой).
//
//go:embed demo/bracket-process.bpmn
var BracketBpmn []byte

// processList — список процессов на шаге (изделий в работе меняется).
func (c *Ctx) processList(inWork int) loader.Response {
	return resp("process.process.list", processapp.ProcessList{Items: []processapp.ProcessSummary{
		{ProcessID: FlangeProcessID, Name: FlangeProcessName, ActiveVersion: &processapp.ProcessVersionRef{VersionID: ProcessVersionID, Label: "v1"},
			Versions: 1, Status: "active", IsDefault: true, ItemsInWork: inWork},
		{ProcessID: BracketProcessID, Name: BracketProcess, ActiveVersion: &processapp.ProcessVersionRef{VersionID: BracketVersionID, Label: "v1"},
			Versions: 1, Status: "active", ItemsInWork: 0},
	}})
}

// bracket — живая карта, версии и карточки узлов второго процесса (неизменны,
// только на шаге 0: шаги накопительные).
func (c *Ctx) bracket() []loader.Response {
	byKey, order, err := ParseBpmn(BracketBpmn)
	if err != nil || len(byKey) == 0 {
		panic("world: BPMN кронштейна не разобран") // встроенный файл — ошибка сборки
	}
	digest := Digest(BracketBpmn)
	created := c.M.clk.at(14, 9, 0)
	ver := processapp.MapVersionRef{ProcessVersionID: BracketVersionID, Label: "v1", IsCurrent: true}
	counters := []processapp.MapNodeCounters{}
	for _, n := range order {
		counters = append(counters, processapp.MapNodeCounters{StepKey: n.StepKey})
	}
	lm := processapp.LiveMap{ProcessID: BracketProcessID, ProcessName: BracketProcess, ProcessVersion: ver, Versions: []processapp.MapVersionRef{ver},
		BpmnXML: loader.BlobPrefix + BracketBpmnBlob, Counters: counters, Items: []processapp.MapItem{}, Anomalies: []processapp.MapNodeAnomaly{},
		DataGaps: []string{}, BasisSeq: c.Seq()}
	quorum := &processapp.VersionQuorum{Have: 3, Need: 3}
	sum := processapp.ProcessVersionSummary{VersionID: BracketVersionID, ProcessID: BracketProcessID, Label: "v1", Status: "active", Hash: digest,
		CreatedAt: created, EffectiveFrom: tptr(created), Quorum: quorum}
	v := processapp.ProcessVersion{VersionID: BracketVersionID, Label: "v1", Status: "active", Hash: digest, Author: ptr("TEC-01"), CreatedAt: created,
		EffectiveFrom: tptr(created), Quorum: quorum, Elements: []processapp.ProcessElement{}, BasisSeq: c.Seq()}
	out := []loader.Response{}
	// Карта кронштейна — и с period: иначе ответ основного процесса с period
	// при равной точности перебил бы process_id (карта кронштейна от периода
	// не зависит — изделий нет).
	for _, kind := range periodKinds {
		out = append(out, resp("process.live_map.read", lm, periodParams(kind, "process_id", BracketProcessID)...),
			resp("process.live_map.read", lm, periodParams(kind, "process_version_id", BracketVersionID)...))
	}
	out = append(out,
		resp("process.version.list", processapp.ProcessVersionList{Items: []processapp.ProcessVersionSummary{sum}}, "process_id", BracketProcessID),
		resp("process.version.diff", processapp.ProcessVersionDiff{VersionID: BracketVersionID, AgainstID: BracketVersionID, Entries: []processapp.ProcessDiffEntry{}},
			"version_id", BracketVersionID),
		resp("process.version.bpmn", processapp.ProcessBpmn{VersionID: BracketVersionID, Hash: digest, BpmnXML: loader.BlobPrefix + BracketBpmnBlob},
			"version_id", BracketVersionID),
	)
	for _, n := range order {
		e := processapp.ProcessElement{ID: n.ID, StepKey: ptr(n.StepKey), Kind: nodeKind(n), Name: n.Name, Properties: map[string]any{}, Next: n.Next}
		if n.Lane != "" {
			e.Lane = ptr(n.Lane)
		}
		card := processapp.ProcessNodeCard{ProcessVersionID: BracketVersionID, StepKey: n.StepKey, ElementID: n.ID, Name: n.Name, Kind: nodeKind(n),
			Lane: n.Lane, Documentation: n.Doc, Properties: map[string]any{}, NormRefs: []processapp.NormRef{}, Items: []platform.DrillRef{},
			Nonconformities: []platform.DrillRef{}, Counters: processapp.MapNodeCounters{StepKey: n.StepKey}}
		for k, val := range n.Props {
			e.Properties[k] = val
			card.Properties[k] = val
		}
		v.Elements = append(v.Elements, e)
		out = append(out, resp("process.node.read", card, "version_id", BracketVersionID, "step_key", n.StepKey))
	}
	return append(out, resp("process.version.read", v, "version_id", BracketVersionID))
}
