package process

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	dp "ant/internal/domain/process"
)

// Проекции модуля process (AD-45; писатель — process). Пишутся только
// эффектами в транзакции Append движка: проекция изделия — воркером по итогу
// свёртки, перечень изделий прогона — ролью projector; пересобираются
// `ant rebuild`. Ключи перечня содержат прогон сценария (AD-38).
const (
	// ProjectionItem — положение изделия в процессе (ось «положение», AD-30):
	// токены по step_key, версия, посещения шагов, пропуски данных.
	ProjectionItem = "process.item"
	// ProjectionIndex — изделия прогона (ключ — run_id; вне прогона — «-»)
	// с закреплённой версией: перечень для живой карты.
	ProjectionIndex = "process.index"
	// ProjectionRuns — выполнение операции → изделие (ключ — operation_run_id):
	// команды паузы и конца операции с терминала знают только выполнение.
	ProjectionRuns = "process.runs"
	// ProjectionOpenRuns — незавершённые выполнения (один ключ OpenRunsKey):
	// гард «одна операция на человека на посту» (FR-137).
	ProjectionOpenRuns = "process.open_runs"
	// OpenRunsKey — ключ перечня незавершённых выполнений.
	OpenRunsKey = "open"
)

// OpenRun — незавершённое выполнение: кто, на каком посту, над каким изделием.
type OpenRun struct {
	RunID     string `json:"operation_run_id"`
	ItemID    string `json:"item_id"`
	Operator  string `json:"operator_id,omitempty"`
	Workplace string `json:"workplace_id,omitempty"`
}

// OpenRuns — значение проекции process.open_runs.
type OpenRuns struct {
	Runs []OpenRun `json:"runs"`
}

// RunRef — значение проекции process.runs.
type RunRef struct {
	ItemID  string `json:"item_id"`
	StepKey string `json:"step_key,omitempty"`
	// EventID — запись «начато» (event_id = command_id команды со стола):
	// повтор той же команды — не второй старт.
	EventID string `json:"event_id,omitempty"`
}

// Показатели вклада изделия для аналитики (AD-45, эпик 25): строки по шагу
// и суткам (срез «‹step_key›|ГГГГ-ММ-ДД», значение — штуки).
const (
	// MetricStepPassed — выполнено шагов (посещение завершено).
	MetricStepPassed = "process.step.passed"
	// MetricStepGaps — шаги, пройденные без данных (оценка невозможна).
	MetricStepGaps = "process.step.gaps"
	// MetricStepDefects — признаки дефекта на шаге (различные зоны в посещении).
	MetricStepDefects = "process.step.defects"
)

// ItemView — значение проекции process.item.
type ItemView struct {
	ItemID        string             `json:"item_id"`
	RunID         string             `json:"run_id,omitempty"`
	VersionID     string             `json:"version_id,omitempty"`
	VersionHash   string             `json:"version_hash,omitempty"`
	VersionLabel  string             `json:"version_label,omitempty"`
	Refused       string             `json:"refused,omitempty"`
	RefusedDetail string             `json:"refused_detail,omitempty"`
	BasisSeq      int64              `json:"basis_seq"`
	Now           time.Time          `json:"now"`
	StartedAt     *time.Time         `json:"started_at,omitempty"`
	Completed     bool               `json:"completed,omitempty"`
	Outcome       string             `json:"outcome,omitempty"`
	EndStep       string             `json:"end_step,omitempty"`
	CompletedAt   *time.Time         `json:"completed_at,omitempty"`
	Isolated      bool               `json:"isolated,omitempty"`
	Containment   string             `json:"containment,omitempty"`
	Tokens        []dp.TokenView     `json:"tokens"`
	Primary       *dp.TokenView      `json:"primary,omitempty"`
	Visits        []dp.Visit         `json:"visits"`
	Gaps          []dp.Gap           `json:"gaps,omitempty"`
	Defects       map[string]int     `json:"defects,omitempty"`
	Gates         map[string]dp.Gate `json:"gates,omitempty"`
	Breaches      []dp.Breach        `json:"breaches,omitempty"`
	Refusals      []dp.Refusal       `json:"refusals,omitempty"`
	Deadlines     []dp.Deadline      `json:"deadlines,omitempty"`
}

// ViewOf — представление изделия по итогу свёртки (проекция и запросы на момент).
func ViewOf(itemID string, s engine.Snapshot) ItemView {
	st := s.Process
	env := st.Env()
	v := ItemView{ItemID: itemID, RunID: st.RunID, VersionID: st.VersionID, VersionHash: st.VersionHash, VersionLabel: env.Label,
		Refused: st.Refused, RefusedDetail: st.RefusedWhy, BasisSeq: s.BasisSeq, Now: st.Now, StartedAt: st.StartedAt,
		Completed: st.Completed, Outcome: st.Outcome, EndStep: st.EndStep, CompletedAt: st.CompletedAt, Isolated: st.Isolated,
		Containment: st.Containment, Tokens: []dp.TokenView{}, Visits: st.History, Gaps: st.Gaps, Defects: st.Defects, Gates: st.Gates,
		Breaches: st.Breaches, Refusals: st.Refusals}
	if v.Visits == nil {
		v.Visits = []dp.Visit{}
	}
	if env.Def != nil {
		v.Tokens = st.Positions(env)
		if p, ok := st.Primary(env); ok {
			v.Primary = &p
		}
		v.Deadlines = st.Deadlines(env)
	}
	return v
}

// IndexEntry — изделие в перечне прогона.
type IndexEntry struct {
	ItemID       string    `json:"item_id"`
	Hash         string    `json:"process_version_hash"`
	RegisteredAt time.Time `json:"registered_at"`
}

// Index — значение проекции process.index.
type Index struct {
	Items []IndexEntry `json:"items"`
}

// IndexKey — ключ перечня изделий прогона.
func IndexKey(runID string) string {
	if runID == "" {
		return "-"
	}
	return runID
}

// RegisterProjections регистрирует проекции process в реестре движка
// (cmd/ant/engine.go, engineRegistry) и вклады показателей шагов.
func RegisterProjections(reg *engineapp.Registry) error {
	if err := reg.AddItem(engineapp.ItemProjection{Name: ProjectionItem, Writer: dp.Module, View: itemView}); err != nil {
		return err
	}
	if err := reg.AddGlobal(engineapp.GlobalProjection{Name: ProjectionIndex, Writer: dp.Module, Keys: indexKeys, Step: indexStep}); err != nil {
		return err
	}
	if err := reg.AddGlobal(engineapp.GlobalProjection{Name: ProjectionRuns, Writer: dp.Module, Keys: runKeys, Step: runStep}); err != nil {
		return err
	}
	if err := reg.AddGlobal(engineapp.GlobalProjection{Name: ProjectionOpenRuns, Writer: dp.Module, Keys: openRunKeys, Step: openRunStep}); err != nil {
		return err
	}
	reg.AddContributor(contributions)
	return nil
}

type runRecord struct {
	RunID   string `json:"operation_run_id"`
	StepKey string `json:"step_key"`
}

func runKeys(r kernel.Record) []string {
	if r.Type != catalog.OperationRunStarted || r.ItemID == "" {
		return nil
	}
	var d runRecord
	if json.Unmarshal(r.Data, &d) != nil || d.RunID == "" {
		return nil
	}
	return []string{d.RunID}
}

func runStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	var d runRecord
	if err := json.Unmarshal(r.Data, &d); err != nil {
		return nil, err
	}
	// Первое «начато» выполнения остаётся: повтор id выполнения у другого
	// изделия — ошибка стола, а не переход выполнения (гард StartOperation).
	if len(prev) > 0 {
		var was RunRef
		if json.Unmarshal(prev, &was) == nil && was.ItemID != "" {
			return prev, nil
		}
	}
	return json.Marshal(RunRef{ItemID: r.ItemID, StepKey: d.StepKey, EventID: r.EventID})
}

func openRunKeys(r kernel.Record) []string {
	switch r.Type {
	case catalog.OperationRunStarted, catalog.OperationRunFinished:
		if r.ItemID != "" {
			return []string{OpenRunsKey}
		}
	}
	return nil
}

func openRunStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	var v OpenRuns
	if len(prev) > 0 {
		if err := json.Unmarshal(prev, &v); err != nil {
			return nil, err
		}
	}
	var d struct {
		RunID     string  `json:"operation_run_id"`
		Operator  *string `json:"operator_id"`
		Workplace *string `json:"workplace_id"`
		Station   *string `json:"station_id"`
	}
	if json.Unmarshal(r.Data, &d) != nil || d.RunID == "" {
		return prev, nil
	}
	v.Runs = slices.DeleteFunc(v.Runs, func(o OpenRun) bool { return o.RunID == d.RunID && o.ItemID == r.ItemID })
	if r.Type == catalog.OperationRunStarted {
		o := OpenRun{RunID: d.RunID, ItemID: r.ItemID}
		if d.Operator != nil {
			o.Operator = *d.Operator
		}
		switch {
		case d.Workplace != nil && *d.Workplace != "":
			o.Workplace = *d.Workplace
		case d.Station != nil:
			o.Workplace = *d.Station
		}
		v.Runs = append(v.Runs, o)
	}
	if v.Runs == nil {
		v.Runs = []OpenRun{}
	}
	return json.Marshal(v)
}

func itemView(itemID string, s engine.Snapshot, _ []kernel.Reaction) (any, error) {
	if s.Process.StartedAt == nil && s.Process.VersionHash == "" {
		return nil, nil
	}
	return ViewOf(itemID, s), nil
}

func indexKeys(r kernel.Record) []string {
	if r.Type != catalog.ItemItemRegistered || r.ItemID == "" {
		return nil
	}
	return []string{IndexKey(r.RunID)}
}

func indexStep(_ string, prev json.RawMessage, r kernel.Record) (json.RawMessage, error) {
	var idx Index
	if len(prev) > 0 {
		if err := json.Unmarshal(prev, &idx); err != nil {
			return nil, fmt.Errorf("process.index: %w", err)
		}
	}
	reg, _ := FindRegistration([]kernel.Record{r})
	if !slices.ContainsFunc(idx.Items, func(e IndexEntry) bool { return e.ItemID == r.ItemID }) {
		idx.Items = append(idx.Items, IndexEntry{ItemID: r.ItemID, Hash: reg.Hash, RegisteredAt: r.OccurredAt.UTC()})
	}
	return json.Marshal(idx)
}

func day(t time.Time) string { return t.UTC().Format("2006-01-02") }

// contributions — вклады изделия: выполненные шаги, пропуски данных и
// признаки дефекта по шагу и суткам (AD-45: заменяются целиком при пересвёртке).
func contributions(itemID string, s engine.Snapshot, _ []kernel.Record) ([]engineapp.Contribution, error) {
	st := s.Process
	type key struct{ metric, slice string }
	sum := map[key]int64{}
	src := map[key][]string{}
	for _, v := range st.History {
		if v.Left != nil && v.Via == "completed" && v.StepKey != "" {
			k := key{MetricStepPassed, v.StepKey + "|" + day(*v.Left)}
			sum[k]++
		}
	}
	for _, g := range st.Gaps {
		k := key{MetricStepGaps, g.StepKey + "|" + day(st.Now)}
		sum[k]++
		src[k] = append(src[k], g.EventID)
	}
	for _, step := range sortedKeys(st.Defects) {
		k := key{MetricStepDefects, step + "|" + day(st.Now)}
		sum[k] += int64(st.Defects[step])
	}
	keys := make([]key, 0, len(sum))
	for k := range sum {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b key) int {
		if a.metric != b.metric {
			if a.metric < b.metric {
				return -1
			}
			return 1
		}
		if a.slice < b.slice {
			return -1
		}
		if a.slice > b.slice {
			return 1
		}
		return 0
	})
	out := make([]engineapp.Contribution, 0, len(keys))
	for _, k := range keys {
		out = append(out, engineapp.Contribution{ItemID: itemID, Metric: k.metric, Slice: k.slice, Value: sum[k], Sources: slices.Compact(sortedStrings(src[k]))})
	}
	return out, nil
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func sortedStrings(xs []string) []string {
	out := slices.Clone(xs)
	slices.Sort(out)
	return out
}
