package world

import (
	"fmt"
	"slices"
	"strings"
	"time"

	erpapp "ant/internal/application/erp"
	ingestapp "ant/internal/application/ingest"
	journalapp "ant/internal/application/journal"
	mlapp "ant/internal/application/machinelogs"
	notifapp "ant/internal/application/notifications"
	opsapp "ant/internal/application/ops"
	"ant/internal/application/platform"
	visionapp "ant/internal/application/vision"
	"ant/internal/infrastructure/fixtures/loader"
)

// ─────────────────────────────── уведомления ───────────────────────────────

var roleAssignees = map[string][]string{
	"quality_inspector": {"INS-01", "INS-02"}, "head_of_qc": {"HQC-01", "INS-01", "INS-02"}, "technologist": {"TEC-01"}, "chief_welder": {"CWL-01", "TEC-01"},
	"site_foreman": {"FOR-WC", "FOR-AC", "FOR-MC"}, "head_of_workshop": {"FOR-WC", "FOR-AC"}, "storekeeper": {"STK-51"}, "administrator": {"ADM-01"},
	"production_manager": {}, "security_auditor": {},
}

func (c *Ctx) tasksFor(role string) []*Task {
	var out []*Task
	for _, t := range c.M.Tasks {
		if t.Created.After(c.T) {
			continue
		}
		if role == "" || slices.Contains(roleAssignees[role], t.Assignee) || t.Role == role {
			out = append(out, t)
		}
	}
	return out
}

// nodeName — имя узла BPMN действующей версии по step_key (AlertEntry.node_name);
// узла нет или у него нет имени — nil (фронт показывает step_key).
func (c *Ctx) nodeName(stepKey string) *string {
	if n := c.M.Bpmn[stepKey]; n != nil && n.Name != "" {
		return ptr(n.Name)
	}
	return nil
}

// stepNames — step_key → имя узла BPMN процесса фланца (UI-21).
func stepNames(nodes map[string]*BpmnNode) map[string]string {
	out := map[string]string{}
	for k, n := range nodes {
		if n != nil && n.Name != "" {
			out[k] = n.Name
		}
	}
	return out
}

func (c *Ctx) alerts() notifapp.AlertList {
	al := notifapp.AlertList{Items: []notifapp.AlertEntry{}}
	add := func(id string, at time.Time, kind string, f func(*notifapp.AlertEntry)) {
		if at.After(c.T) {
			return
		}
		a := notifapp.AlertEntry{AlertID: id, At: at, Kind: kind}
		f(&a)
		al.Items = append(al.Items, a)
	}
	for _, s := range c.M.Spec.Sources {
		if !s.Restored.Time().After(c.T) {
			continue
		}
		add("AL-"+s.ID, s.Alert.Time(), "anomaly", func(a *notifapp.AlertEntry) {
			a.Node, a.NodeName, a.Anomaly = ptr("welding.weld"), c.nodeName("welding.weld"), ptr("downtime_over_threshold")
			a.Ref = &platform.DrillRef{Entity: platform.EntityEquipment, ID: s.Equipment}
		})
	}
	for _, h := range c.M.Spec.ProcessHolds {
		add("AL-HOLD-"+h.Equipment, h.Set.Time(), "anomaly", func(a *notifapp.AlertEntry) {
			a.Node, a.NodeName, a.Anomaly = ptr("welding.weld"), c.nodeName("welding.weld"), ptr("downtime_over_threshold")
			a.Ref = &platform.DrillRef{Entity: platform.EntityEquipment, ID: h.Equipment}
		})
	}
	for _, n := range c.M.NCs {
		if n.Spec.Isolated.IsZero() || len(n.Items) == 0 {
			continue
		}
		it := n.Items[0]
		if !n.Spec.Isolated.Time().After(c.T) {
			continue
		}
		add("AL-ISO-"+n.ID, n.ConfirmedAt, "not_moved_to_isolator", func(a *notifapp.AlertEntry) {
			a.Item = ptr(FullID(it.ID))
			a.Ref = &platform.DrillRef{Entity: platform.EntityItem, ID: FullID(it.ID)}
		})
	}
	for _, v := range c.M.Spec.Integrity.Violations {
		add("AL-INT-"+v.Record, v.At.Time(), "integrity_violation", func(a *notifapp.AlertEntry) {
			a.Ref = &platform.DrillRef{Entity: platform.EntityIntegrity, ID: "global"}
		})
	}
	slices.SortStableFunc(al.Items, func(a, b notifapp.AlertEntry) int { return b.At.Compare(a.At) })
	return al
}

func renderNotifications(c *Ctx) []loader.Response {
	var out []loader.Response
	al := c.alerts()
	att := notifapp.AttentionList{Items: []notifapp.AttentionEntry{}}
	for _, n := range c.M.NCs {
		if len(n.Spec.Items) > 0 || len(n.Items) == 0 || n.ConfirmedAt.After(c.T) {
			continue
		}
		if d := n.DispositionAt(c.M); d != nil && !d.After(c.T) {
			continue
		}
		due := workingDaysAfter(c.M, n.ConfirmedAt, 3)
		if due.Before(c.T) {
			att.Items = append(att.Items, notifapp.AttentionEntry{Kind: "overdue_decision", EntryID: "ATT-" + n.ID, Target: ptr(n.Number), OverdueMinutes: ptr(int(c.T.Sub(due).Minutes())), Items: ptr(1), Operations: ptr(1), Ref: &platform.DrillRef{Entity: platform.EntityNonconformity, ID: n.ID}})
		}
	}
	for _, in := range c.M.Incidents {
		if in.Spec.CauseConfirmed.IsZero() || in.Spec.CauseConfirmed.Time().After(c.T) {
			continue
		}
		att.Items = append(att.Items, notifapp.AttentionEntry{Kind: "unverified_measures", EntryID: "ATT-MEASURE-" + in.Spec.ID, N: ptr(1), Ref: &platform.DrillRef{Entity: platform.EntityIncident, ID: in.Spec.ID}})
	}
	for _, role := range append([]string{""}, sortedKeys(roleAssignees)...) {
		tl := notifapp.TaskList{Items: []notifapp.TaskEntry{}}
		sum := notifapp.NotificationSummary{ByKind: &notifapp.NotificationSummaryByKind{}}
		for _, t := range c.tasksFor(role) {
			state := "open"
			if !t.Done.IsZero() && !t.Done.After(c.T) {
				state = "done"
			}
			te := notifapp.TaskEntry{TaskID: t.ID, Kind: t.Kind, Title: t.Title, State: state, AssigneeRole: t.Role, AssigneeID: ptr(t.Assignee), CreatedAt: t.Created, DueAt: tptr(t.Due),
				Overdue: !t.Due.IsZero() && t.Due.Before(c.T) && state == "open", Ref: &platform.DrillRef{Entity: platform.EntityKind(t.Ref[0]), ID: t.Ref[1]}}
			if t.Location != "" {
				te.LocationID = ptr(t.Location)
			}
			tl.Items = append(tl.Items, te)
			if state == "open" {
				if t.Kind == "decision_required" || t.Kind == "review_after_new_data" {
					sum.ByKind.DecisionRequest++
				} else {
					sum.ByKind.Task++
				}
			}
		}
		slices.SortStableFunc(tl.Items, func(a, b notifapp.TaskEntry) int { return b.CreatedAt.Compare(a.CreatedAt) })
		if role == "" || role == "production_manager" || role == "administrator" || role == "security_auditor" || role == "head_of_qc" {
			for _, a := range al.Items {
				if c.T.Sub(a.At) < 24*time.Hour {
					sum.ByKind.Alarm++
				}
			}
		}
		sum.Unread = sum.ByKind.Task + sum.ByKind.DecisionRequest + sum.ByKind.Alarm + sum.ByKind.Info
		kv := []string{}
		if role != "" {
			kv = []string{"role", role}
		}
		out = append(out, resp("notifications.task.list", tl, kv...), resp("notifications.summary.read", sum, kv...))
	}
	out = append(out, resp("notifications.attention.list", att), resp("notifications.alert.list", al))
	return out
}

// ─────────────────────────────── 1С ───────────────────────────────

func (c *Ctx) erpMessage(e *ERPMsg) erpapp.ErpMessage {
	msg := erpapp.ErpMessage{BusinessKey: e.ID, ExternalSystem: "onec", Action: e.Action, MessageVersion: 1, Status: "sent", AfterRework: e.After, RequestedAt: e.At, Attempts: []erpapp.ErpMessageAttempt{}, BasisSeq: c.EntitySeq(e.ID)}
	if len(e.Items) == 1 {
		msg.ItemID = ptr(FullID(e.Items[0].ID))
	}
	if e.Lot != "" {
		msg.LotID = ptr(e.Lot)
	}
	for _, ev := range c.M.Events {
		if ev.Type == "erp.posting.requested" && ev.Entity.ID == e.ID {
			msg.RequestEventID = ev.ID
		}
	}
	for _, a := range e.Attempts {
		if a.At.Time().After(c.T) {
			continue
		}
		at := erpapp.ErpMessageAttempt{At: a.At.Time(), Outcome: "accepted"}
		switch a.Result {
		case "acked":
			msg.Status, msg.AccountingState = "acknowledged", ptr(e.Axis)
			if a.Doc != "" {
				at.ExternalDocumentRef = ptr(a.Doc)
			}
		case "error_retryable":
			at.Outcome, at.ErrorCode, at.ErrorMessage = "transport_error", ptr(a.Code), ptr(fmt.Sprintf("HTTP %d: 1С недоступна — повтор с тем же номером сообщения", a.HTTP))
		case "error_final":
			msg.Status = "rejected"
			at.Outcome, at.ErrorCode, at.ErrorMessage = "rejected", ptr(a.Code), ptr(fmt.Sprintf("HTTP %d: не найден договор с поставщиком — задача администратору", a.HTTP))
		}
		msg.Attempts = append(msg.Attempts, at)
	}
	return msg
}

func renderERP(c *Ctx) []loader.Response {
	var out []loader.Response
	orders := erpapp.ErpOrderList{Items: []erpapp.ErpOrder{}}
	for _, o := range c.M.Spec.Orders {
		launched := 0
		for _, it := range c.Existing() {
			if it.Order == o.ID {
				launched++
			}
		}
		orders.Items = append(orders.Items, erpapp.ErpOrder{OrderID: o.ID, ExternalSystem: "onec", ExternalNumber: o.Label, ItemTypeID: flangeType, ItemRevision: ptr("Б"), Quantity: o.Qty, Launched: launched, DueDate: ptr(o.Due), ReceivedAt: o.Received.Time()})
	}
	out = append(out, resp("erp.order.list", orders))
	list := erpapp.ErpMessageList{Items: []erpapp.ErpMessage{}}
	queued, rejected := 0, 0
	var last time.Time
	for _, e := range c.M.ERP {
		if e.At.After(c.T) {
			continue
		}
		m := c.erpMessage(e)
		if len(m.Attempts) == 0 {
			m.Status = "queued"
			queued++
		}
		if m.Status == "rejected" {
			rejected++
		}
		if n := len(m.Attempts); n > 0 && m.Attempts[n-1].At.After(last) {
			last = m.Attempts[n-1].At
		}
		list.Items = append(list.Items, m)
		out = append(out, resp("erp.message.read", m, "business_key", e.ID))
	}
	slices.SortStableFunc(list.Items, func(a, b erpapp.ErpMessage) int { return b.RequestedAt.Compare(a.RequestedAt) })
	if len(list.Items) > 100 {
		list.Items = list.Items[:100] // последние 100: страница экрана; одно сообщение — erp.message.read
	}
	out = append(out, resp("erp.message.list", list))
	ch := erpapp.ErpChannel{System: "onec", State: "ok", Endpoint: "http://stands:8090/onec", Stand: true, ContractVersion: "1.0", LastExchangeAt: tptr(last), Queued: queued, Quarantined: rejected}
	if rejected > 0 {
		ch.State, ch.Detail = "degraded", ptr("Ошибка данных: не найден договор с поставщиком — нужна правка соответствий")
	}
	out = append(out, resp("erp.channel.list", erpapp.ErpChannelList{Items: []erpapp.ErpChannel{ch}}))
	return out
}

// ─────────────────────────────── журнал ───────────────────────────────

func (c *Ctx) entryView(e *Event) journalapp.JournalEntryView {
	v := journalapp.JournalEntryView{Seq: e.Seq, EventID: e.ID, EventType: e.Type, SchemaVersion: 1, EntryKind: e.Kind, Chain: "main", SourceID: e.Source, ProvenanceClass: e.Provenance,
		Stream: e.Stream, OccurredAt: e.Occurred, ReceivedAt: e.Recorded, RecordedAt: e.Recorded, CommittedAt: e.Recorded, CorrelationID: e.ID, Signers: []string{}, SignatureStatus: "not_checked",
		Data: map[string]any{"summary": e.Summary}}
	if sk := sourceKindOf(e); sk != "" {
		v.SourceKind = ptr(sk)
	}
	if e.Item != nil {
		v.ItemID = ptr(FullID(e.Item.ID))
	}
	for k, p := range e.Params {
		v.Data[k] = p
	}
	if e.Author != "" {
		v.Signers = []string{strings.ToLower(e.Author) + "@1"}
	}
	return v
}

func renderJournal(c *Ctx) []loader.Response {
	var out []loader.Response
	vis := c.Visible()
	list := journalapp.JournalEntryList{Items: []journalapp.JournalEntryView{}}
	for i := len(vis) - 1; i >= 0 && len(list.Items) < 50; i-- {
		list.Items = append(list.Items, c.entryView(vis[i]))
	}
	out = append(out, resp("journal.entry.list", list))
	// Журнал по изделию — в паспорте (item.passport.read); здесь — последние 50 записей
	// и каждая запись по seq (переход к записи).
	for _, e := range vis {
		if e.Step == c.N {
			out = append(out, resp("journal.entry.read", c.entryView(e), "seq", fmt.Sprint(e.Seq)))
		}
	}
	head := journalapp.JournalHead{Seq: c.Seq(), ClockMode: "scenario", RecordedAt: tptr(c.T)}
	out = append(out, resp("journal.head.read", head))
	tl := journalapp.TimelineData{From: c.M.Steps[0], To: c.T, Marks: []journalapp.TimelineMark{}}
	for _, e := range vis {
		var kind string
		switch e.Type {
		case "decision.process_hold.set":
			kind = "process_stop"
		case "incident.scope.computed", "security.integrity.violated", "ingest.anomaly.flagged":
			kind = "spike"
		case "task.task.created", "ingest.source.loss_suspected":
			kind = "escalation"
		default:
			continue
		}
		mk := journalapp.TimelineMark{MarkID: e.ID, At: e.Occurred, Kind: kind, Title: ptr(e.Summary)}
		if e.Entity.Entity != "" {
			mk.Ref = &platform.DrillRef{Entity: platform.EntityKind(e.Entity.Entity), ID: e.Entity.ID}
		} else if e.Item != nil {
			mk.Ref = &platform.DrillRef{Entity: platform.EntityItem, ID: FullID(e.Item.ID)}
		}
		tl.Marks = append(tl.Marks, mk)
	}
	out = append(out, resp("journal.timeline.read", tl))
	return out
}

// ─────────────────────────────── оборудование ───────────────────────────────

var equipmentTitle = map[string][2]string{
	"CNC-1": {"Станок ЧПУ", "WP-CNC-1"}, "IS-1": {"Сварочный источник ИС-1", "WP-WELD-1"}, "IS-2": {"Сварочный источник ИС-2", "WP-WELD-2"},
	"CMM-1": {"Координатно-измерительная машина", "WP-CMM-1"}, "XRAY-LAB": {"Рентгеновский контроль (лаборатория НК)", "LAB-NDT"},
	"LEAK-1": {"Стенд с течеискателем", "WP-LEAK-1"}, "TW-1": {"Ключ с регистрацией момента", "WP-ASM-1"},
}

func (c *Ctx) equipment(id string) mlapp.EquipmentState {
	t := equipmentTitle[id]
	st := mlapp.EquipmentState{EquipmentID: id, Title: t[0], StationID: t[1], Execution: "idle", ControllerMode: "automatic", Condition: "normal", Warnings: []mlapp.EquipmentWarning{},
		Verification: mlapp.EquipmentVerification{Status: "valid"}, SourceKind: "machine", UpdatedAt: tptr(c.T)}
	if id == "IS-1" || id == "IS-2" {
		st.SpecialProcess, st.ProgramRef = true, "ПС-4"
		st.Verification = mlapp.EquipmentVerification{Status: "unknown"}
	}
	for _, it := range c.M.Items {
		for _, r := range it.Runs {
			if r.Equipment == id && !r.From.After(c.T) && r.To.After(c.T) {
				st.Execution, st.CurrentRunID = "running", r.ID
			}
		}
	}
	for _, s := range c.M.Spec.Sources {
		if s.Equipment == id && !s.Alert.Time().After(c.T) && s.Restored.Time().After(c.T) {
			st.Condition, st.Execution = "unknown", "unknown"
			st.Warnings = append(st.Warnings, mlapp.EquipmentWarning{Kind: "other", Text: "Нет данных от источника: связь со шлюзом потеряна", Since: s.Lost.Time()})
		}
	}
	for _, le := range c.M.Spec.LateEvents {
		if le.Source == "edge-weld-2" && id == "IS-2" && !le.Received.Time().After(c.T) {
			st.Condition = "warning"
			st.Warnings = append(st.Warnings, mlapp.EquipmentWarning{Kind: "out_of_setpoint", Text: fmt.Sprintf("Ток вне уставки: %d–182 А при 160 ± 10 А", le.CurrentA), Since: le.Occurred.Time()})
		}
	}
	for _, h := range c.M.Spec.ProcessHolds {
		if h.Equipment == id && !h.Set.Time().After(c.T) {
			st.Execution, st.Condition = "stopped", "fault"
		}
	}
	return st
}

func (c *Ctx) runProfile(r *OpRun) mlapp.RunProfile {
	p := mlapp.RunProfile{OperationRunID: r.ID, ItemID: FullID(r.Item.ID), StepKey: r.StepKey, EquipmentID: r.Equipment, OperatorID: ptr(r.Performer), StartedAt: r.From,
		IntervalOrigin: "system_computed", ProgramRef: r.Program, Parameters: []mlapp.CycleParameter{}, Events: []mlapp.EquipmentEventRow{}, SpecialProcess: r.Kind == "welding"}
	if !r.To.After(c.T) {
		p.FinishedAt = tptr(r.To)
	}
	if r.Kind == "machining" || r.Kind == "leak_test" {
		p.IntervalOrigin = "source_reported"
	}
	if r.Kind == "welding" {
		cp := mlapp.CycleParameter{Parameter: "current_a", Scale: 0, Unit: "А", Setpoint: "160 ± 10 А"}
		if !r.LogLost && !r.LogReceived.IsZero() && !r.LogReceived.After(c.T) {
			cp.Value = ptr(int64(r.CurrentA[1]))
			cp.InRange = ptr(r.CurrentA[1] <= 170 && r.CurrentA[0] >= 150)
			p.Violation = !*cp.InRange
		}
		p.Parameters = append(p.Parameters, cp)
	}
	for _, e := range c.Visible() {
		if e.Params["operation_run_id"] == r.ID {
			p.Events = append(p.Events, mlapp.EquipmentEventRow{EventID: e.ID, EventType: e.Type, Layer: layerOf(e.Type), Seq: e.Seq, OccurredAt: e.Occurred, Summary: e.Summary, SourceKind: sourceKindOf(e)})
		}
	}
	return p
}

func layerOf(t string) string {
	switch {
	case strings.HasPrefix(t, "equipment.deviation"):
		return "deviation"
	case strings.HasPrefix(t, "equipment.cycle"):
		return "how"
	case strings.HasPrefix(t, "equipment.program") || strings.HasPrefix(t, "equipment.tool"):
		return "with_what"
	}
	return "what"
}

func renderMachinelogs(c *Ctx) []loader.Response {
	var out []loader.Response
	list := mlapp.EquipmentList{Items: []mlapp.EquipmentState{}}
	for _, id := range sortedKeys(equipmentTitle) {
		st := c.equipment(id)
		list.Items = append(list.Items, st)
		out = append(out, resp("machinelogs.equipment.read", st, "equipment_id", id))
		tl := mlapp.EquipmentTimeline{EquipmentID: id, From: c.dayStart().Add(-24 * time.Hour), To: c.T, Rows: []mlapp.EquipmentEventRow{}}
		for _, e := range c.Visible() {
			if e.Params["equipment_id"] == id && strings.HasPrefix(e.Type, "equipment.") || e.Entity.ID == id && strings.HasPrefix(e.Type, "equipment.") {
				if e.Occurred.Before(tl.From) {
					continue
				}
				tl.Rows = append(tl.Rows, mlapp.EquipmentEventRow{EventID: e.ID, EventType: e.Type, Layer: layerOf(e.Type), Seq: e.Seq, OccurredAt: e.Occurred, Summary: e.Summary, Params: e.Params, SourceKind: sourceKindOf(e)})
			}
		}
		out = append(out, resp("machinelogs.timeline.read", tl, "equipment_id", id))
	}
	out = append(out, resp("machinelogs.equipment.list", list))
	for _, it := range c.Existing() {
		for _, r := range it.Runs {
			if !r.From.After(c.T) {
				out = append(out, resp("machinelogs.run_profile.read", c.runProfile(r), "run_id", r.ID))
			}
		}
	}
	vl := mlapp.ViolationList{Items: []mlapp.ViolationWindow{}}
	for _, le := range c.M.Spec.LateEvents {
		if le.Received.Time().After(c.T) {
			continue
		}
		w := mlapp.ViolationWindow{EquipmentID: "IS-2", StepKey: "welding.weld", WindowStart: le.Occurred.Time(), DeviationEventIDs: []string{}, OperationRunIDs: []string{}, Items: []platform.DrillRef{}, Nonconformities: []platform.DrillRef{}}
		for _, h := range c.M.Spec.ProcessHolds {
			if !h.Set.Time().After(c.T) {
				w.WindowEnd = tptr(h.Set.Time())
			}
		}
		for _, e := range c.Visible() {
			if e.Type == "equipment.deviation.detected" {
				w.DeviationEventIDs = append(w.DeviationEventIDs, e.ID)
			}
		}
		for _, it := range c.M.Items {
			for _, r := range it.Runs {
				if r.Equipment == "IS-2" && r.To.After(le.Occurred.Time()) && r.From.Before(c.M.clk.at(23, 11, 12)) && r.ReworkOf == "" {
					w.OperationRunIDs = append(w.OperationRunIDs, r.ID)
					w.Items = append(w.Items, platform.DrillRef{Entity: platform.EntityItem, ID: FullID(it.ID)})
				}
			}
		}
		if g := c.M.NC("NC-G1"); g != nil && !g.ConfirmedAt.After(c.T) {
			w.Nonconformities = append(w.Nonconformities, platform.DrillRef{Entity: platform.EntityNonconformity, ID: g.ID})
		}
		vl.Items = append(vl.Items, w)
	}
	out = append(out, resp("machinelogs.violation.list", vl))
	return out
}

// ─────────────────────────────── VisionQC ───────────────────────────────

func renderVision(c *Ctx) []loader.Response {
	if c.N != 0 {
		return nil
	}
	admitted := c.M.clk.at(1, 9, 0)
	vers := map[string]string{"analyzer": "vqc-weld 2.3.1", "recipe": "kt3-weld@1", "contract": "1.0", "threshold_profile": "TP-3"}
	p1 := visionapp.AnalyzerPassport{PassportID: "AP-KT3-3", AnalyzerID: "vqc-weld", Stage: "active", TrustLevel: 2, RecipeRef: "kt3-weld@1", Versions: vers, Status: "active", AdmittedAt: admitted,
		DocumentID: "DOC-AP-KT3-3", AllowedAutoActions: []string{"record", "protective"}, BasisSeq: c.Seq()}
	p2 := visionapp.AnalyzerPassport{PassportID: "AP-OV-1", AnalyzerID: "ov-asm", Stage: "pilot", TrustLevel: 1, RecipeRef: "ov-asm@1", Versions: map[string]string{"analyzer": "ov-asm 0.9.0", "contract": "1.0"}, Status: "active", AdmittedAt: admitted,
		DocumentID: "DOC-AP-OV-1", AllowedAutoActions: []string{"record"}, BasisSeq: c.Seq()}
	list := visionapp.AnalyzerList{Items: []visionapp.AnalyzerSummary{
		{AnalyzerID: "vqc-weld", Title: "Визуальный контроль шва (КТ-3)", Kind: "visionqc", PassportID: ptr(p1.PassportID), Stage: ptr("active"), TrustLevel: ptr(2), Status: "active", Versions: vers},
		{AnalyzerID: "ov-asm", Title: "Контроль действий оператора на сборке", Kind: "operatorvision", PassportID: ptr(p2.PassportID), Stage: ptr("pilot"), TrustLevel: ptr(1), Status: "active", Versions: p2.Versions},
	}}
	checks := visionapp.AnalyzerCheckList{Items: []visionapp.AnalyzerCheck{{EventID: c.M.eventID("check/AP-KT3-3"), PassportID: "AP-KT3-3", CheckKind: "reference_set", EscapeRateBP: ptr(120), FalseAlarmRateBP: ptr(450), Passed: true, OccurredAt: admitted.Add(-24 * time.Hour)}}}
	return []loader.Response{
		resp("vision.analyzer.list", list),
		resp("vision.passport.read", p1, "passport_id", p1.PassportID), resp("vision.passport.read", p2, "passport_id", p2.PassportID),
		resp("vision.check.list", checks, "passport_id", p1.PassportID), resp("vision.check.list", visionapp.AnalyzerCheckList{Items: []visionapp.AnalyzerCheck{}}),
	}
}

// ─────────────────────────────── приём ───────────────────────────────

func renderIngest(c *Ctx) []loader.Response {
	var out []loader.Response
	ql := ingestapp.QuarantineList{Items: []ingestapp.QuarantineEntry{}}
	open := 0
	for _, q := range c.M.Spec.Quarantine {
		if q.At.Time().After(c.T) {
			continue
		}
		e := ingestapp.QuarantineEntry{QuarantineID: q.ID, SourceID: q.Source, Fingerprint: Digest([]byte("q/" + q.ID)), MaterialAddress: Digest([]byte("m/" + q.ID)), ProblemCode: q.Code,
			Detail: ptr(fmt.Sprintf("Поле %s%s", q.Field, map[bool]string{true: ": «" + q.Value + "»"}[q.Value != ""])), QuarantinedAt: q.At.Time(), State: "open", BasisSeq: c.Seq()}
		if !q.Fixed.IsZero() && !q.Fixed.Time().After(c.T) {
			e.State = "accepted"
		} else {
			open++
		}
		ql.Items = append(ql.Items, e)
		full := e
		full.Content = ptr(fmt.Sprintf(`{"event_type":"inspection.result.recorded","schema_version":"%s","%s":%q}`, map[bool]string{true: q.Value, false: "1.0"}[q.Field == "schema_version"], q.Field, q.Value))
		out = append(out, resp("ingest.quarantine.read", full, "quarantine_id", q.ID))
	}
	out = append(out, resp("ingest.quarantine.list", ql))
	sources := ingestapp.SourceList{Items: []ingestapp.SourceView{}}
	for _, id := range []string{"edge-cnc-1", "edge-kt2", "edge-kt3", "edge-leak-1", "edge-weld-1", "edge-weld-2", "gw-ndt", "onec"} {
		sv := ingestapp.SourceView{SourceID: id, SourceKind: "machine", KeyRef: ptr(id + "@1"), State: "active", LastSeq: ptr(int64(4000 + len(id)*97)), LastReceivedAt: tptr(c.T), BasisSeq: c.Seq()}
		switch {
		case strings.HasPrefix(id, "edge-kt"):
			sv.SourceKind = "camera"
		case id == "gw-ndt":
			sv.SourceKind = "manual_entry"
		case id == "onec":
			sv.SourceKind, sv.KeyRef = "external_system", nil
		}
		for _, s := range c.M.Spec.Sources {
			if s.ID != id {
				continue
			}
			if !s.Alert.Time().After(c.T) && s.Restored.Time().After(c.T) {
				sv.State, sv.LastReceivedAt = "loss_suspected", tptr(s.Lost.Time())
			}
			if !s.Restored.Time().After(c.T) {
				sv.GapCount = 1
			}
		}
		for _, q := range c.M.Spec.Quarantine {
			if q.Source == id && !q.At.Time().After(c.T) && (q.Fixed.IsZero() || q.Fixed.Time().After(c.T)) {
				sv.Quarantined++
			}
		}
		sources.Items = append(sources.Items, sv)
	}
	out = append(out, resp("ingest.source.list", sources))
	var received, dups, lost int64 = int64(len(c.Visible())), 0, 0
	for _, s := range c.M.Spec.Sources {
		if !s.Restored.Time().After(c.T) {
			received += int64(s.Batch.Records)
			dups += int64(s.Batch.Duplicates) + 1 // + повтор кадра камеры КТ-3 (S06)
			lost += int64(s.Batch.Lost)
		}
	}
	quar := int64(0)
	for _, q := range c.M.Spec.Quarantine {
		if !q.At.Time().After(c.T) {
			quar++
		}
	}
	comp := 10000
	if received > 0 && lost > 0 {
		comp = int(10000 - lost*10000/(received+lost))
	}
	out = append(out, resp("ingest.metrics.read", ingestapp.IngestMetrics{WindowFrom: c.M.Steps[0], WindowTo: c.T, Received: received + dups + quar, Accepted: received, Duplicates: dups,
		Quarantined: quar, QuarantineOpen: int64(open), LatencyP50Ms: 40, LatencyP95Ms: 180, EventToScreenP95Ms: 900, CompletenessBP: comp}))
	return out
}

// ─────────────────────────────── эксплуатация ───────────────────────────────

func renderOps(c *Ctx) []loader.Response {
	h := opsapp.OpsHealth{Components: []opsapp.ComponentState{
		{Component: "ant/api", State: "ok", Instances: 1, CheckedAt: tptr(c.T)},
		{Component: "postgres", State: "ok", Instances: 1, CheckedAt: tptr(c.T)},
		{Component: "ant/worker", State: "not_implemented", Instances: 0, Detail: ptr("Режим заготовок: движок не участвует (AD-36)"), CheckedAt: tptr(c.T)},
	}, Queues: []opsapp.QueueState{}, Integrations: []opsapp.IntegrationState{{System: "onec", State: "ok"}}, StoppedItems: 0, Profile: "fixtures", Mode: platform.ModeFixtures, Version: "fixtures"}
	for _, q := range c.M.Spec.Quarantine {
		if !q.At.Time().After(c.T) && (q.Fixed.IsZero() || q.Fixed.Time().After(c.T)) {
			h.QuarantineOpen++
		}
	}
	for _, e := range c.M.ERP {
		for i, a := range e.Attempts {
			if a.Result == "error_final" && !a.At.Time().After(c.T) && (i+1 >= len(e.Attempts) || e.Attempts[i+1].At.Time().After(c.T)) {
				h.Integrations[0] = opsapp.IntegrationState{System: "onec", State: "degraded", Detail: ptr("Ошибка данных по " + e.ID + ": " + a.Code), Since: tptr(a.At.Time())}
			}
		}
	}
	verdict := "intact"
	checked := c.T.Truncate(15 * time.Minute)
	for _, v := range c.M.Spec.Integrity.Violations {
		if !v.At.Time().After(c.T) {
			verdict, checked = "violated", v.At.Time()
		}
	}
	h.Verifier = &opsapp.VerifierReportRef{Verdict: verdict, CheckedAt: checked, ReportRef: Digest([]byte("verifier/" + verdict))}
	out := []loader.Response{resp("ops.health.read", h), resp("ops.stopped_item.list", opsapp.StoppedItemList{Items: []opsapp.StoppedItem{}}),
		resp("ops.integration.list", c.integrations())}
	if c.N == 0 {
		mods := []opsapp.ModuleMode{}
		for _, m := range []string{"access", "analysis", "analytics", "erp", "ingest", "item", "journal", "machinelogs", "nonconformity", "notifications", "ops", "process", "quality", "security", "simulation", "vision"} {
			mods = append(mods, opsapp.ModuleMode{Module: m, Mode: platform.ModeFixtures})
		}
		out = append(out, resp("ops.setting.list", opsapp.SettingList{Ports: []opsapp.PortSetting{
			{Port: "journal_store", Adapter: "postgres", Zone: "storage", Replace: "BFT-журнал (описание)"},
			{Port: "fixture_cursor", Adapter: "postgres", Zone: "storage"},
			{Port: "publisher", Adapter: "http", Zone: "transport", Replace: "Kafka"},
			{Port: "telemetry", Adapter: "prometheus", Zone: "observability", Replace: "OTLP"},
			{Port: "access_control", Adapter: "permissive", Zone: "security", Replace: "Casbin"},
			{Port: "signer", Adapter: "demo_signer", Zone: "security", Replace: "PKCS#11, СКЗИ"},
		}, Modules: mods, Enabled: []string{"onec"}}))
	}
	return out
}

// integrations — экран «Интеграции» (ops.integration.list, FR-157; эпик 48):
// в мире заготовок установлены стенды 1С, КОМПАС-3D, VisionQC, контроля
// действий оператора и демо-УЦ; Галактика, MES, СКУД и партнёры — «не
// установлена». Канал 1С — как в erp.channel.list того же шага.
func (c *Ctx) integrations() opsapp.IntegrationList {
	queued, rejected := int64(0), int64(0)
	var last time.Time
	var lastErr *opsapp.IntegrationError
	for _, e := range c.M.ERP {
		if e.At.After(c.T) {
			continue
		}
		m := c.erpMessage(e)
		switch {
		case len(m.Attempts) == 0:
			queued++
		case m.Status == "rejected":
			rejected++
		}
		if n := len(m.Attempts); n > 0 {
			a := m.Attempts[n-1]
			if a.At.After(last) {
				last = a.At
			}
			if a.Outcome == "rejected" && a.ErrorMessage != nil {
				lastErr = &opsapp.IntegrationError{At: a.At, Detail: *a.ErrorMessage}
			}
		}
	}
	since := c.M.Steps[0]
	check := func(ep string) *opsapp.IntegrationCheck {
		return &opsapp.IntegrationCheck{Result: "ok", Endpoint: ptr(ep), Detail: ptr("ответная сторона отвечает, версия контракта совпала"), At: since, Seq: 1}
	}
	onec := opsapp.IntegrationEntry{System: "onec", Installed: true, State: "stand", Default: true, StandAvailable: true,
		Channel: ptr("ok"), Endpoint: ptr("http://stands:8090/onec"), CheckedAt: tptr(c.T), Queued: ptr(queued), Quarantined: ptr(rejected),
		LastError: lastErr, LastCheck: check("http://stands:8090/onec"), BasisSeq: 0}
	if !last.IsZero() {
		onec.LastExchangeAt = tptr(last)
	}
	if rejected > 0 {
		onec.Channel, onec.Detail = ptr("degraded"), ptr("Ошибка данных: не найден договор с поставщиком — нужна правка соответствий")
	}
	stand := func(sys, ep string) opsapp.IntegrationEntry {
		return opsapp.IntegrationEntry{System: sys, Installed: true, State: "stand", Default: true, StandAvailable: true,
			Channel: ptr("ok"), Endpoint: ptr(ep), CheckedAt: tptr(c.T), LastExchangeAt: tptr(c.T), LastCheck: check(ep)}
	}
	absent := func(sys string) opsapp.IntegrationEntry {
		return opsapp.IntegrationEntry{System: sys, State: "disabled", Default: true}
	}
	// Эпик 37: СКУД — stand роли stands (журнал проходов skud.v1, кнопки проходов на /stand/skud/).
	skud := stand("skud", "http://stands:8491/stand/skud/api/v1")
	skud.Detail = ptr("Турникеты зон цехов: проходы сотрудников, присутствие на постах")
	ca := stand("ca", "https://ca.stand/ocsp")
	ca.Detail = ptr("Демо-УЦ: выпуск и отзыв сертификатов mTLS, OCSP")
	return opsapp.IntegrationList{Profile: "fixtures", Items: []opsapp.IntegrationEntry{
		onec, absent("galaktika"), absent("mes"), stand("kompas", "file:///var/lib/ant/exchange/kompas"), skud,
		ca, stand("visionqc", "http://stands:8090/visionqc"), stand("operatorvision", "http://stands:8090/operatorvision"), absent("partner"),
	}}
}
