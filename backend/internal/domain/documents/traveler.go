package documents

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
)

// Сопроводительная карта изделия (FR-65, каталог документов: «ядро»): по
// каждой операции — исполнитель, дата, подпись, фактические параметры и
// отметка ОТК; номера заявлений о несоответствии и разрешений на отклонение;
// итоговая годность. Это ровно те события, которые система пишет в историю
// изделия (research/documents-per-step.md §3.1), — карта собирается из
// журнала, вручную ничего не заполняется.

// Header — шапка карты: изделие, обозначение, изменение КД, задание, партии,
// версия техпроцесса.
type Header struct {
	ItemID      string   `json:"item_id,omitempty"`
	ItemTypeID  string   `json:"item_type_id,omitempty"`
	Revision    string   `json:"item_revision,omitempty"`
	OrderID     string   `json:"order_id,omitempty"`
	Lots        []string `json:"lots,omitempty"`
	ProcessHash string   `json:"process_version_hash,omitempty"`
	Source      *Ref     `json:"source,omitempty"`
}

// RowSig — подпись факта строки: кто, класс происхождения, уровень (AD-2, AD-13).
type RowSig struct {
	EventID    string `json:"event_id"`
	Person     string `json:"person,omitempty"`
	Provenance string `json:"provenance,omitempty"`
	Level      int    `json:"level"`
}

// Mark — отметка контроля по строке: результат контроля или решение ОТК.
type Mark struct {
	EventID string    `json:"event_id"`
	Label   string    `json:"label"`
	By      string    `json:"by,omitempty"`
	At      time.Time `json:"at"`
	// Human — решение человека (отметка ОТК «годен»); иначе — результат контроля.
	Human bool `json:"human,omitempty"`
}

// Row — строка карты: выполнение операции.
type Row struct {
	RunID      string     `json:"operation_run_id,omitempty"`
	StepKey    string     `json:"step_key"`
	Executor   string     `json:"executor,omitempty"`
	Equipment  string     `json:"equipment,omitempty"`
	Program    string     `json:"program,omitempty"`
	ReworkOf   string     `json:"rework_of,omitempty"`
	StartedAt  time.Time  `json:"started_at,omitzero"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Completion string     `json:"completion,omitempty"`
	Duration   string     `json:"duration,omitempty"`
	Sigs       []RowSig   `json:"sigs,omitempty"`
	Params     []string   `json:"params,omitempty"`
	// Regime — фактический режим оборудования за выполнение (сводки
	// параметров machinelogs: среднее, минимум, максимум, уставка, FR-121).
	Regime  []string `json:"regime,omitempty"`
	Checks  []Mark   `json:"checks,omitempty"`
	OTK     *Mark    `json:"otk,omitempty"`
	Remarks []string `json:"remarks,omitempty"`
	Sources []Ref    `json:"sources"`
}

// NCInfo — несоответствие изделия для карты, заявления и решения.
type NCInfo struct {
	NCID        string   `json:"nc_id"`
	Number      string   `json:"number"`
	Status      string   `json:"status"`
	StepKey     string   `json:"step_key,omitempty"`
	RunID       string   `json:"operation_run_id,omitempty"`
	Requirement string   `json:"requirement,omitempty"`
	Fact        string   `json:"fact,omitempty"`
	Severity    string   `json:"severity,omitempty"`
	DefectType  string   `json:"defect_type,omitempty"`
	FoundBy     string   `json:"found_by,omitempty"`
	FoundAt     string   `json:"found_at,omitempty"`
	FoundSource string   `json:"found_source,omitempty"`
	Confirmed   *RowSig  `json:"confirmed,omitempty"`
	ConfirmedAt string   `json:"confirmed_at,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
	Disposition string   `json:"disposition,omitempty"`
	Concession  string   `json:"concession,omitempty"`
	DocumentID  string   `json:"document_id,omitempty"`
	Sources     []Ref    `json:"sources"`
}

// Origin — происхождение сопроводительной карты для людей (FR-65, Д-65:
// платформа для людей — «Главный»).
const Origin = "Сопроводительная карта собрана системой «Главный» из истории изделия (журнала) — вручную не заполнялась"

// NCNumber — номер заявления о несоответствии для людей: тот же вид, что у
// nonconformity.NCNumber («НС-» + первые 8 знаков id), без импорта позднего
// модуля композиции (AD-40).
func NCNumber(ncID string) string {
	h := strings.ToUpper(strings.ReplaceAll(ncID, "-", ""))
	if len(h) > 8 {
		h = h[:8]
	}
	return "НС-" + h
}

// Метки для людей.
var (
	resolutionLabel = map[string]string{
		"accept": "годен", "accept_with_concession": "годен по разрешению на отклонение",
		"reject": "не годен — возврат", "insufficient_data": "мало данных для решения",
	}
	outcomeLabel = map[string]string{
		"no_defect_indicated": "признаков дефекта нет", "defect_indicated": "признак дефекта",
		"unable_to_assess": "оценка невозможна",
	}
	dispositionLabel = map[string]string{
		"rework": "переделка", "repair": "ремонт", "use_as_is": "как есть",
		"scrap": "списать", "return_to_supplier": "вернуть поставщику",
	}
)

func label(m map[string]string, code string) string {
	if l, ok := m[code]; ok {
		return l
	}
	return code
}

// FormatTime — время в документе по соглашению (RFC 3339, UTC, три знака).
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// levelOf — уровень подписи факта по классу происхождения (AD-13):
// устройство — 0, факты исполнителя — 1, решения человека — 2.
func levelOf(r kernel.Record) int {
	switch {
	case r.Provenance == "device":
		return 0
	case r.Kind == catalog.KindDecision:
		return 2
	}
	return 1
}

func sigOf(r kernel.Record) RowSig {
	return RowSig{EventID: r.EventID, Person: PersonOf(r.Actor), Provenance: r.Provenance, Level: levelOf(r)}
}

// measurement — значение измерения целым с масштабом (AD-4, без float).
func measurement(m *ev.Measurement) string {
	if m == nil {
		return Empty
	}
	v, neg := m.Value, m.Value < 0
	if neg {
		v = -v
	}
	s := strconv.Itoa(v)
	if m.Scale > 0 {
		for len(s) <= m.Scale {
			s = "0" + s
		}
		s = s[:len(s)-m.Scale] + "." + s[len(s)-m.Scale:]
	}
	if neg {
		s = "-" + s
	}
	return s + " " + m.Unit
}

// row — последняя строка шага (по порядку начала) или строка выполнения.
func (s *State) row(runID, stepKey string) *Row {
	if runID != "" {
		for i := len(s.Rows) - 1; i >= 0; i-- {
			if s.Rows[i].RunID == runID {
				return &s.Rows[i]
			}
		}
	}
	if stepKey != "" {
		for i := len(s.Rows) - 1; i >= 0; i-- {
			if s.Rows[i].StepKey == stepKey {
				return &s.Rows[i]
			}
		}
	}
	return nil
}

// addParticipant — участник изготовления изделия (FR-56).
func (s *State) addParticipant(p string) {
	if p == "" || slices.Contains(s.Participants, p) {
		return
	}
	s.Participants = append(s.Participants, p)
	slices.Sort(s.Participants)
}

// reduceTraveler — факты и решения, из которых собирается карта.
func (s *State) reduceTraveler(r kernel.Record) {
	switch r.Type {
	case catalog.ItemItemRegistered:
		d, err := kernel.Decode[ev.ItemItemRegisteredV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		src := refOf(r)
		s.Header = Header{ItemID: string(d.ItemID), ItemTypeID: string(d.ItemTypeID), Revision: d.ItemRevision, ProcessHash: string(d.ProcessVersionHash), Source: &src}
		if d.OrderID != nil {
			s.Header.OrderID = string(*d.OrderID)
		}
		for _, l := range d.LotIds {
			s.Header.Lots = append(s.Header.Lots, string(l))
		}
	case catalog.OperationRunStarted:
		d, err := kernel.Decode[ev.OperationRunStartedV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		row := Row{RunID: string(d.OperationRunID), StepKey: string(d.StepKey), StartedAt: r.OccurredAt, Sigs: []RowSig{sigOf(r)}, Sources: []Ref{refOf(r)}}
		if d.OperatorID != nil {
			row.Executor = *d.OperatorID
		}
		if row.Executor == "" {
			row.Executor = PersonOf(r.Actor)
		}
		if d.OperationStartedAt != nil {
			row.StartedAt = time.Time(*d.OperationStartedAt).UTC()
		}
		if d.EquipmentID != nil {
			row.Equipment = string(*d.EquipmentID)
			row.Params = append(row.Params, "Оборудование: "+row.Equipment)
		}
		if d.ProgramRef != nil {
			row.Program = *d.ProgramRef
			row.Params = append(row.Params, "Программа: "+row.Program)
		}
		if d.ReworkOf != nil {
			row.ReworkOf = string(*d.ReworkOf)
			row.Remarks = append(row.Remarks, "Повтор выполнения "+row.ReworkOf)
		}
		s.Rows = append(s.Rows, row)
		s.addParticipant(row.Executor)
		s.addParticipant(PersonOf(r.Actor))
	case catalog.OperationRunFinished:
		d, err := kernel.Decode[ev.OperationRunFinishedV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		row := s.row(string(d.OperationRunID), "")
		if row == nil {
			return
		}
		fin := r.OccurredAt
		if d.OperationFinishedAt != nil {
			fin = time.Time(*d.OperationFinishedAt).UTC()
		}
		row.FinishedAt, row.Completion = &fin, string(d.Completion)
		if d.ReportedDuration != nil {
			row.Duration = strconv.Itoa(d.ReportedDuration.Value) + " " + string(d.ReportedDuration.Unit)
			row.Params = append(row.Params, "Длительность: "+row.Duration)
		}
		row.Sigs = append(row.Sigs, sigOf(r))
		row.Sources = append(row.Sources, refOf(r))
		s.addParticipant(PersonOf(r.Actor))
	case catalog.InspectionResultRecorded:
		d, err := kernel.Decode[ev.InspectionResultRecordedV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		run, step := "", ""
		if d.OperationRunID != nil {
			run = string(*d.OperationRunID)
		}
		if d.StepKey != nil {
			step = string(*d.StepKey)
		}
		row := s.row(run, step)
		if row == nil {
			s.Rows = append(s.Rows, Row{StepKey: step, StartedAt: r.OccurredAt, Sources: []Ref{}})
			row = &s.Rows[len(s.Rows)-1]
		}
		by := PersonOf(r.Actor)
		if d.InspectorID != nil && *d.InspectorID != "" {
			by = *d.InspectorID
		}
		row.Checks = append(row.Checks, Mark{EventID: r.EventID, Label: "контроль «" + string(d.Method) + "»: " + label(outcomeLabel, string(d.Outcome)), By: by, At: r.OccurredAt})
		for _, m := range d.Measurements {
			row.Params = append(row.Params, m.Characteristic+": "+measurement(m.Value)+" ("+string(m.Verdict)+")")
		}
		row.Sources = append(row.Sources, refOf(r))
	case catalog.DecisionPresentationResolved:
		d, err := kernel.Decode[ev.DecisionPresentationResolvedV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		row := s.row("", string(d.StepKey))
		if row == nil {
			s.Rows = append(s.Rows, Row{StepKey: string(d.StepKey), StartedAt: r.OccurredAt, Sources: []Ref{}})
			row = &s.Rows[len(s.Rows)-1]
		}
		l := label(resolutionLabel, string(d.Resolution))
		if d.ConcessionID != nil {
			l += " № " + string(*d.ConcessionID)
			row.Remarks = append(row.Remarks, "Разрешение на отклонение "+string(*d.ConcessionID))
		}
		if d.PresentationNo > 1 {
			l += " (предъявление " + strconv.Itoa(d.PresentationNo) + ")"
		}
		row.OTK = &Mark{EventID: r.EventID, Label: l, By: PersonOf(r.Actor), At: r.OccurredAt, Human: true}
		row.Sources = append(row.Sources, refOf(r))
	case catalog.ItemReleaseRecorded:
		d, err := kernel.Decode[ev.ItemReleaseRecordedV1](r)
		if err != nil {
			s.bad(r)
			return
		}
		src := refOf(r)
		st := "годен — сдан на склад " + string(d.WarehouseID)
		if d.ConcessionID != nil {
			st = "годен по разрешению на отклонение № " + string(*d.ConcessionID) + " — сдан на склад " + string(d.WarehouseID)
		}
		s.Final = &Final{Status: st, At: FormatTime(r.OccurredAt), Source: src}
	}
}

// fromMachinelogs — фактический режим оборудования строк карты из профилей
// выполнения machinelogs (раньше documents в композиции, AD-40).
func (s *State) fromMachinelogs(up Upstream) {
	if up.Machinelogs == nil {
		return
	}
	for i := range s.Rows {
		r := &s.Rows[i]
		p, ok := up.Machinelogs.Profile(r.RunID)
		if r.RunID == "" || !ok {
			continue
		}
		var regime []string
		for _, ps := range p.Parameters {
			regime = append(regime, paramLine(ps))
		}
		if p.ToolID != "" {
			regime = append(regime, "Инструмент: "+p.ToolID)
		}
		if p.ManualMode {
			regime = append(regime, "Ручное изменение режима во время выполнения")
		}
		r.Regime = regime
	}
}

// paramLine — сводка параметра режима для людей (целые с масштабом, AD-4).
func paramLine(p machinelogs.ParamSummary) string {
	m := func(x *machinelogs.Measure) string {
		if x == nil {
			return Empty
		}
		return measurement(&ev.Measurement{Value: int(x.Value), Scale: x.Scale, Unit: x.Unit})
	}
	l := p.Parameter + ": ср. " + m(p.Mean) + " (мин. " + m(p.Min) + ", макс. " + m(p.Max) + ")"
	if p.Setpoint != nil {
		l += ", уставка " + m(p.Setpoint.Lower) + " … " + m(p.Setpoint.Upper)
	}
	if in := p.InRange(); in != nil && !*in {
		l += " — вне уставки"
	}
	return l
}

// Final — итоговая годность изделия.
type Final struct {
	Status string `json:"status"`
	At     string `json:"at,omitempty"`
	Source Ref    `json:"source"`
}

// travelerBody — содержательная часть сопроводительной карты (без номера
// версии и маршрута) и её события-источники.
func (s *State) travelerBody(env Env) (map[string]any, []Ref) {
	var sources []Ref
	if s.Header.Source != nil {
		sources = append(sources, *s.Header.Source)
	}
	rows := make([]any, 0, len(s.Rows))
	otk := 0
	for i, r := range s.Rows {
		sources = append(sources, r.Sources...)
		date := FormatTime(r.StartedAt)
		if r.FinishedAt != nil {
			date += " — " + FormatTime(*r.FinishedAt)
		}
		var sigs []string
		for _, sg := range r.Sigs {
			who := sg.Person
			if who == "" {
				who = "устройство"
			}
			sigs = append(sigs, who+" (подпись "+sg.Provenance+", ур. "+strconv.Itoa(sg.Level)+")")
		}
		mark, markBy := "", ""
		switch {
		case r.OTK != nil:
			mark, markBy = r.OTK.Label+" — "+r.OTK.By+", "+FormatTime(r.OTK.At), r.OTK.By
			otk++
		case len(r.Checks) > 0:
			c := r.Checks[len(r.Checks)-1]
			mark = c.Label + " — " + c.By + ", " + FormatTime(c.At)
		}
		remarks := slices.Clone(r.Remarks)
		for _, n := range s.NCs {
			if (n.RunID != "" && n.RunID == r.RunID) || (n.RunID == "" && n.StepKey == r.StepKey) {
				remarks = append(remarks, "Заявление о несоответствии "+n.Number)
			}
		}
		rows = append(rows, map[string]any{
			"no": i + 1, "step_key": r.StepKey, "operation": env.StepTitle(r.StepKey), "operation_run_id": r.RunID,
			"executor": r.Executor, "date": date, "signature": sigs, "params": nonNil(append(slices.Clone(r.Params), r.Regime...)), "otk": mark, "otk_by": markBy,
			"completion": r.Completion, "remarks": nonNil(remarks),
		})
	}
	ncs := make([]any, 0, len(s.NCs))
	for _, n := range s.NCs {
		sources = append(sources, n.Sources...)
		disp := ""
		if n.Disposition != "" {
			disp = label(dispositionLabel, n.Disposition)
		}
		ncs = append(ncs, map[string]any{"nc_id": n.NCID, "number": n.Number, "status": n.Status, "disposition": disp, "concession": n.Concession})
	}
	final := map[string]any{"status": "в работе", "at": ""}
	if s.Final != nil {
		final = map[string]any{"status": s.Final.Status, "at": s.Final.At}
		sources = append(sources, s.Final.Source)
	}
	body := map[string]any{
		"item": map[string]any{
			"item_id": s.Header.ItemID, "item_type_id": s.Header.ItemTypeID, "item_revision": s.Header.Revision,
			"order_id": s.Header.OrderID, "lots": nonNil(s.Header.Lots), "process_version_hash": s.Header.ProcessHash,
		},
		"rows": rows, "rows_count": len(rows), "otk_count": otk, "nc_count": len(ncs), "nonconformities": ncs, "final": final,
		"origin": Origin,
	}
	if body["item"].(map[string]any)["item_id"] == "" {
		body["item"].(map[string]any)["item_id"] = s.ItemID
	}
	return body, sources
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
