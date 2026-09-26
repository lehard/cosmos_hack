package world

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

// NC — несоответствие (FR-51…FR-53).
type NC struct {
	Spec        NCSpec
	ID, Number  string
	Items       []*Item
	SignalID    string
	SignalAt    time.Time
	ConfirmedAt time.Time
	StepKey     string
	Hyps        []HypVersion
	VerifiedAt  time.Time
}

// HypVersion — версия гипотез по несоответствию (incident.hypothesis.computed).
type HypVersion struct {
	V            int
	At           time.Time
	RevisedDueTo string
	Equipment    string // not_assessable | weak | possible | strong | confirmed
	Performer    string
	Incoming     string
	Missing      []string
	Categorical  bool
}

// Status — статус несоответствия на момент (draft → confirmed → disposition_set → verified → closed).
func (n *NC) Status(m *Model, t time.Time) string {
	switch {
	case !n.VerifiedAt.IsZero() && !n.VerifiedAt.After(t):
		return "verified"
	case n.DispositionAt(m) != nil && !n.DispositionAt(m).After(t):
		return "disposition_set"
	case !n.ConfirmedAt.After(t):
		return "confirmed"
	}
	return "draft"
}

// DispositionAt — момент решения по несоответствию: своё или группового НС-И1.
func (n *NC) DispositionAt(m *Model) *time.Time {
	if d := n.Spec.Disposition; d != nil {
		t := d.At.Time()
		return &t
	}
	if g := m.groupFor(n); g != nil && g.Spec.Disposition != nil {
		t := g.Spec.Disposition.At.Time()
		return &t
	}
	return nil
}

// Disposition — вид решения по несоответствию.
func (n *NC) Disposition(m *Model) string {
	if d := n.Spec.Disposition; d != nil {
		return d.Kind
	}
	if g := m.groupFor(n); g != nil && g.Spec.Disposition != nil {
		return g.Spec.Disposition.Kind
	}
	return "none"
}

func (m *Model) groupFor(n *NC) *NC {
	for _, g := range m.NCs {
		if g == n || len(g.Spec.Items) == 0 || g.ConfirmedAt.Before(n.ConfirmedAt) {
			continue
		}
		for _, it := range n.Items {
			if slices.Contains(g.Spec.Items, it.ID) {
				return g
			}
		}
	}
	return nil
}

// Incident — инцидент и версии области риска (FR-61, FR-62).
type Incident struct {
	Spec     IncidentSpec
	Versions []ScopeState
	Members  []*Item
	// Rings — компоненты партии вне изделий (кольца на складе) под блоком.
	Rings []string
}

// ScopeState — версия области: статусы изделий.
type ScopeState struct {
	Spec   ScopeVersion
	Status map[string]string // item id → confirmed | suspect | excluded | unknown
}

// Size — изделий в области (без исключённых).
func (s ScopeState) Size() int {
	n := 0
	for _, v := range s.Status {
		if v != "excluded" {
			n++
		}
	}
	return n
}

// Count — изделий с данным статусом.
func (s ScopeState) Count(status string) int {
	n := 0
	for _, v := range s.Status {
		if v == status {
			n++
		}
	}
	return n
}

// VersionAt — последняя версия области на момент t (nil — инцидента ещё нет).
func (in *Incident) VersionAt(t time.Time) *ScopeState {
	var out *ScopeState
	for i := range in.Versions {
		if in.Versions[i].Spec.At.Time().After(t) {
			break
		}
		out = &in.Versions[i]
	}
	return out
}

// ERPMsg — исходящее учётное сообщение 1С (AD-7: бизнес-ключ).
type ERPMsg struct {
	ID       string
	Action   string // accept_into_work | warehouse_transfer | scrap_transfer_rework | scrap_transfer_writeoff | return_to_supplier | release | rework_return
	Items    []*Item
	Lot      string
	Qty      int
	Route    string
	At       time.Time
	Basis    string
	Attempts []ERPAttempt
	Axis     string // ось «учёт в 1С» после квитанции
	After    bool   // после переделки
}

// AckAt — момент подтверждения 1С (нулевой — не подтверждено).
func (e *ERPMsg) AckAt() time.Time {
	for _, a := range e.Attempts {
		if a.Result == "acked" {
			return a.At.Time()
		}
	}
	return time.Time{}
}

// Task — задача или уведомление (FR-57; порождает notifications).
type Task struct {
	ID       string
	Kind     string
	Title    string
	Role     string
	Assignee string
	Location string
	Created  time.Time
	Done     time.Time
	Due      time.Time
	Ref      [2]string // вид, id
}

// ─────────────────────────────── сюжет ───────────────────────────────

func (m *Model) buildStory() error {
	for _, s := range m.Spec.NCs {
		n := &NC{Spec: s, ID: s.ID, Number: s.Label, SignalAt: s.Signal.Time(), ConfirmedAt: s.Confirmed.Time()}
		for _, id := range s.AllItems() {
			it := m.itemByID[id]
			if it == nil {
				return fmt.Errorf("%s: нет изделия %s", s.ID, id)
			}
			n.Items = append(n.Items, it)
		}
		if s.Signal.IsZero() {
			n.SignalAt = n.ConfirmedAt
		}
		n.SignalID = "SIG-" + strings.ReplaceAll(s.ID, "-", "")
		n.StepKey = "welding.kt3_radiography"
		if s.Source == "CAM-KT3" {
			n.StepKey = "welding.kt3_camera"
		}
		if s.Lot != "" {
			n.StepKey = "incoming.zt1_lot_acceptance"
		}
		if len(s.Items) > 1 {
			n.StepKey = "welding.zt3_acceptance"
		}
		m.NCs = append(m.NCs, n)
		m.ncByID[n.ID] = n
	}
	for _, s := range m.Spec.Signals {
		it := m.itemByID[s.Item]
		if it == nil {
			return fmt.Errorf("%s: нет изделия %s", s.ID, s.Item)
		}
		if s.Unable {
			prev := it.State(s.At.Time().Add(-time.Second)).Quality
			it.mark(s.At.Time(), "quality", "unable_to_assess")
			it.mark(s.Resolved.Time(), "quality", prev)
		} else {
			it.mark(s.At.Time(), "quality", "signal")
			it.mark(s.Rejected.Time(), "quality", "conforming")
		}
	}
	if err := m.applyNCs(); err != nil {
		return err
	}
	if err := m.buildIncidents(); err != nil {
		return err
	}
	for _, rv := range m.Spec.Reviews {
		it := m.itemByID[rv.Item]
		it.mark(rv.Flagged.Time(), "review", "flagged")
		it.mark(rv.Completed.Time(), "review", rv.Outcome)
	}
	for _, iv := range m.Spec.Interventions {
		it := m.itemByID[iv.Item]
		it.move(iv.Opened.Time(), "assembly.intervention", "in_progress", "WP-ASM-1")
		b := iv.BackToWC.Time()
		it.move(b, "welding.receive", "in_transit", "WS-WC")
		it.move(b.Add(minutes(5)), "welding.edge_prep", "in_queue", "ISO-WC")
	}
	m.buildERP()
	for _, it := range m.Items {
		it.sortTimeline()
	}
	m.buildTasks()
	m.buildEvents()
	return nil
}

// applyNCs — сигнал, подтверждение, изоляция, решение по изделию (FR-52, FR-53).
func (m *Model) applyNCs() error {
	for _, n := range m.NCs {
		s := n.Spec
		for _, it := range n.Items {
			if len(s.Items) == 0 {
				it.mark(n.SignalAt, "quality", "signal")
				it.move(n.ConfirmedAt, "welding.nonconformity", "isolated", "ST-WELD")
			}
			it.mark(n.ConfirmedAt, "quality", "nonconforming")
			if !s.Isolated.IsZero() {
				it.move(s.Isolated.Time(), "welding.nonconformity", "isolated", "ISO-WC")
			}
			if d := s.Disposition; d != nil {
				at := d.At.Time()
				it.mark(at, "disposition", d.Kind)
				st := it.State(at)
				switch {
				case d.Kind == "scrap":
					it.move(at, "welding.scrapped", "isolated", "ISO-WC")
				case d.Kind == "rework" && strings.HasPrefix(st.Step, "assembly."):
					it.move(at, "assembly.nonconformity", "isolated", "ISO-AC")
				case d.Kind == "rework":
					it.move(at, "welding.edge_prep", "in_queue", "ISO-WC")
				}
			}
		}
	}
	// Проверка исполнения переделки (FR-53): повторная ЗТ-3 пройдена.
	for _, n := range m.NCs {
		if len(n.Spec.Items) > 0 || n.Disposition(m) != "rework" || len(n.Items) != 1 {
			continue
		}
		for _, mk := range n.Items[0].marks {
			if mk.axis == "rework_done" {
				n.VerifiedAt = mk.at
			}
		}
		if n.ID == "NC-02" {
			n.VerifiedAt = time.Time{}
		}
	}
	for _, u := range m.Spec.ComponentUnlinks {
		it := m.itemByID[u.Item]
		it.move(u.At.Time(), "welding.edge_prep", "in_queue", "ISO-WC")
	}
	m.buildHypotheses()
	return nil
}

// buildHypotheses — версии гипотез (FR-59): S03 v1, S07 v2, S05 v3, подтверждение.
func (m *Model) buildHypotheses() {
	late := time.Time{}
	lateID := ""
	if len(m.Spec.LateEvents) > 0 {
		late, lateID = m.Spec.LateEvents[0].Received.Time(), m.Spec.LateEvents[0].ID
	}
	for _, n := range m.NCs {
		if n.Spec.Cause == nil {
			continue
		}
		cause := n.Spec.Cause
		switch cause.Category {
		case "incoming":
			n.Hyps = []HypVersion{{V: 1, At: n.SignalAt.Add(time.Minute), Incoming: "strong", Equipment: "weak", Performer: "weak", Missing: []string{"other"}}}
		default:
			if n.ID == "NC-01" {
				n.Hyps = []HypVersion{
					{V: 1, At: n.SignalAt, Equipment: "not_assessable", Performer: "possible", Missing: []string{"equipment_log_missing", "no_observation_after_operation"}},
					{V: 2, At: late.Add(time.Minute), RevisedDueTo: lateID, Equipment: "strong", Performer: "possible"},
					{V: 3, At: m.clk.at(23, 13, 30), RevisedDueTo: "NC-02,NC-03", Equipment: "strong", Performer: "weak"},
				}
			} else if len(n.Spec.Items) == 0 {
				n.Hyps = []HypVersion{{V: 1, At: n.SignalAt.Add(time.Minute), Equipment: "strong", Performer: "weak"}}
			}
		}
		if len(n.Hyps) > 0 {
			last := n.Hyps[len(n.Hyps)-1]
			last.V++
			last.At = cause.At.Time()
			last.RevisedDueTo = ""
			last.Categorical = true
			last.Missing = nil
			switch cause.Category {
			case "equipment":
				last.Equipment = "confirmed"
			case "incoming":
				last.Incoming = "confirmed"
			}
			n.Hyps = append(n.Hyps, last)
		}
	}
}

// buildIncidents — области риска из данных (FR-61): отсчёт после последней
// подтверждённо годной сварки, сужение по основаниям версий.
func (m *Model) buildIncidents() error {
	for _, s := range m.Spec.Incidents {
		in := &Incident{Spec: s}
		cur := map[string]string{}
		for _, v := range s.Versions {
			at := v.At.Time()
			next := map[string]string{}
			for k, x := range cur {
				next[k] = x
			}
			switch v.Rule {
			case "after_anchor":
				if m.anchor == nil {
					return fmt.Errorf("%s: нет изделия-отсчёта (anchor)", s.ID)
				}
				anchorEnd := firstWeld(m.anchor).To
				trigger := m.ncByID[s.Trigger]
				for _, it := range m.Items {
					w := firstWeld(it)
					if w == nil || !w.To.After(anchorEnd) || w.From.After(at) {
						continue
					}
					next[it.ID] = "suspect"
					if trigger != nil && slices.Contains(trigger.Spec.AllItems(), it.ID) {
						next[it.ID] = "confirmed"
					}
				}
			case "exclude_source":
				for id, x := range next {
					if x == "suspect" && firstWeld(m.itemByID[id]).Equipment == v.Source {
						next[id] = "excluded"
					}
				}
			case "exclude_before":
				for id, x := range next {
					w := firstWeld(m.itemByID[id])
					if x == "suspect" && !w.To.After(v.Before.Time()) {
						next[id] = "excluded"
					}
				}
				for _, id := range v.Unknown {
					next[id] = "unknown"
				}
			case "confirm_all":
				for id, x := range next {
					if x != "excluded" {
						next[id] = "confirmed"
					}
				}
			case "lot_members":
				for _, it := range m.Items {
					if it.RingLot == v.Lot && firstWeld(it) != nil {
						next[it.ID] = "suspect"
					}
				}
				for _, id := range v.Confirmed {
					next[id] = "confirmed"
				}
				for _, l := range m.Spec.Lots {
					if l.ID == v.Lot {
						rs, _ := expandRange(l.Rings)
						for _, r := range rs {
							used := false
							for _, it := range m.Items {
								used = used || it.Ring == r
							}
							if !used {
								in.Rings = append(in.Rings, r)
							}
						}
					}
				}
			case "exclude_items":
				for _, id := range v.Exclude {
					next[id] = "excluded"
				}
			default:
				return fmt.Errorf("%s v%d: неизвестное правило %q", s.ID, v.V, v.Rule)
			}
			// Оси изделий: статус в инциденте и сдерживание (блок снимает человек тем же шагом сужения).
			for _, id := range sortedKeys(next) {
				it := m.itemByID[id]
				x := next[id]
				if cur[id] == x {
					continue
				}
				it.mark(at, "incident:"+s.ID, x)
				switch {
				case x == "excluded":
					it.mark(at, "containment", "none")
				case cur[id] == "" && x != "excluded":
					it.mark(at, "containment", "item_hold")
				}
			}
			cur = next
			in.Versions = append(in.Versions, ScopeState{Spec: v, Status: next})
		}
		for _, id := range sortedKeys(cur) {
			in.Members = append(in.Members, m.itemByID[id])
		}
		sort.SliceStable(in.Members, func(i, j int) bool { return in.Members[i].ID < in.Members[j].ID })
		m.Incidents = append(m.Incidents, in)
	}
	return nil
}

func firstWeld(it *Item) *OpRun {
	if it == nil {
		return nil
	}
	for _, r := range it.Runs {
		if r.Kind == "welding" {
			return r
		}
	}
	return nil
}

// Incident — инцидент по id.
func (m *Model) Incident(id string) *Incident {
	for _, in := range m.Incidents {
		if in.Spec.ID == id {
			return in
		}
	}
	return nil
}

// NC — несоответствие по id.
func (m *Model) NC(id string) *NC { return m.ncByID[id] }

// buildERP — сообщения 1С: явные из описания и автоматические учётные
// движения по маршруту (принято в работу, перемещения, выпуск; FR-91).
func (m *Model) buildERP() {
	explicit := map[string]bool{}
	for _, e := range m.Spec.ERP {
		msg := &ERPMsg{ID: e.ID, Action: e.Kind, Lot: e.Lot, Qty: e.Qty, Route: e.Route, At: e.At.Time(), Basis: e.Basis, Attempts: e.Attempts}
		for _, id := range e.Items {
			msg.Items = append(msg.Items, m.itemByID[id])
			explicit[e.Kind+"/"+id] = true
		}
		switch e.Kind {
		case "release_good":
			msg.Action, msg.Axis = "release", "released"
		case "scrap_transfer_rework":
			msg.Axis = "transferred_to_scrap"
		case "scrap_transfer_writeoff":
			msg.Axis, msg.After = "transferred_to_scrap", true
		case "return_to_supplier":
			msg.Axis = "returned_to_supplier"
		case "transfer":
			msg.Action, msg.Axis = "warehouse_transfer", "moved"
		case "rework_return_to_production":
			msg.Action, msg.Axis, msg.After = "scrap_transfer_reprocess", "accepted_into_work", true
		}
		m.ERP = append(m.ERP, msg)
		if ack := msg.AckAt(); !ack.IsZero() {
			for _, it := range msg.Items {
				it.mark(ack, "erp", msg.Axis)
			}
		}
	}
	// Автоматические учётные движения — по отметкам оси «учёт в 1С» маршрута.
	for _, it := range m.Items {
		for _, mk := range it.marks {
			if mk.axis != "erp" {
				continue
			}
			action := map[string]string{"accepted_into_work": "accept_into_work", "moved": "warehouse_transfer", "released": "release"}[mk.value]
			if action == "" || explicit["release_good/"+it.ID] && action == "release" {
				continue
			}
			covered := false
			for _, e := range m.ERP {
				if e.Axis == mk.value && e.AckAt().Equal(mk.at) && slices.Contains(e.Items, it) {
					covered = true
				}
			}
			if covered {
				continue
			}
			m.nextER++
			at := mk.at.Add(-time.Minute)
			msg := &ERPMsg{ID: fmt.Sprintf("OUT-%06d", 200000+m.nextER), Action: action, Items: []*Item{it}, At: at, Axis: mk.value,
				Attempts: []ERPAttempt{{At: T{t: mk.at}, Result: "acked", Doc: ""}}}
			m.ERP = append(m.ERP, msg)
		}
	}
	sort.SliceStable(m.ERP, func(i, j int) bool { return m.ERP[i].At.Before(m.ERP[j].At) })
}

// buildTasks — задачи людей по сюжету (FR-55, FR-57): придержать, перенести в
// изолятор, перепроверить, решить, пересмотреть, переотправить.
func (m *Model) buildTasks() {
	add := func(kind, title, role, assignee, loc string, created, done time.Time, ref [2]string) {
		m.Tasks = append(m.Tasks, &Task{ID: fmt.Sprintf("TASK-%03d", len(m.Tasks)+1), Kind: kind, Title: title, Role: role, Assignee: assignee, Location: loc, Created: created, Done: done, Ref: ref})
	}
	for _, in := range m.Incidents {
		if len(in.Versions) == 0 {
			continue
		}
		v1 := in.Versions[0]
		at := v1.Spec.At.Time()
		for _, id := range sortedKeys(v1.Status) {
			it := m.itemByID[id]
			if v1.Status[id] == "confirmed" {
				continue
			}
			st := it.State(at)
			assignee, role, loc := "FOR-WC", "site_foreman", "WS-WC"
			switch {
			case st.Location == "WH-WC":
				assignee, role = "STK-51", "storekeeper"
			case strings.HasPrefix(st.Step, "assembly.") || strings.HasPrefix(st.Step, "testing.") || strings.HasPrefix(st.Step, "final."):
				assignee, loc = "FOR-AC", "WS-AC"
			}
			add("other", fmt.Sprintf("Придержать %s: блок области риска %s", it.Label, in.Spec.ID), role, assignee, loc, at, at.Add(minutes(20)), [2]string{"item", FullID(id)})
		}
	}
	for _, n := range m.NCs {
		if !n.Spec.Isolated.IsZero() {
			it := n.Items[0]
			add("isolate_move", fmt.Sprintf("Перенести %s в изолятор и подтвердить", it.Label), "site_foreman", "FOR-WC", "WS-WC", n.ConfirmedAt, n.Spec.Isolated.Time(), [2]string{"item", FullID(it.ID)})
		}
		if d := n.DispositionAt(m); d != nil && len(n.Spec.Items) == 0 && n.Spec.Disposition == nil {
			continue
		}
	}
	recheck := m.clk.at(23, 12, 30)
	for _, id := range []string{"F-015", "F-019", "F-021", "F-023", "F-025"} {
		it := m.itemByID[id]
		var done time.Time
		for _, mv := range it.moves {
			if mv.step == "welding.zt3_acceptance" && mv.at.After(recheck) {
				done = mv.at
				break
			}
		}
		if id == "F-015" {
			done = m.clk.at(23, 14, 0)
		}
		add("recheck", fmt.Sprintf("Перепроверить %s: камера КТ-3 и рентген", it.Label), "quality_inspector", "INS-01", "WS-WC", recheck, done, [2]string{"item", FullID(id)})
	}
	for _, rv := range m.Spec.Reviews {
		it := m.itemByID[rv.Item]
		add("review_after_new_data", fmt.Sprintf("Пересмотрите решение %s по %s: принято до новых данных", rv.Gate, it.Label), "quality_inspector", rv.By, "WS-WC", rv.Flagged.Time(), rv.Completed.Time(), [2]string{"item", FullID(it.ID)})
	}
	add("other", "Найти и придержать Ф-015 в сборке: блок по качеству", "site_foreman", "FOR-AC", "WS-AC", m.clk.at(23, 12, 20), m.clk.at(23, 12, 30), [2]string{"item", FullID("F-015")})
	for _, st := range m.Spec.Steps {
		if st.Wait == nil {
			continue
		}
		who := map[string]string{"quality_inspector": "INS-01", "technologist": "TEC-01", "chief_welder": "CWL-01"}[st.Wait.Role]
		done := time.Time{}
		for i, s2 := range m.Spec.Steps {
			if s2.At.Time().Equal(st.At.Time()) && i+1 < len(m.Spec.Steps) {
				done = m.Spec.Steps[i+1].At.Time()
			}
		}
		kind := st.Wait.Object.Kind
		add("decision_required", st.Wait.Title, st.Wait.Role, who, "", st.At.Time(), done, [2]string{kind, st.Wait.Object.ID})
	}
	for _, e := range m.ERP {
		for i, a := range e.Attempts {
			if a.Result == "error_final" && i+1 < len(e.Attempts) {
				add("admin_resend", fmt.Sprintf("1С: ошибка данных по сообщению %s — %s", e.ID, a.Code), "administrator", "ADM-01", "", a.At.Time(), e.Attempts[i+1].At.Time(), [2]string{"erp_message", e.ID})
			}
		}
	}
	for _, s := range m.Spec.Sources {
		add("inspection_missing", "Нет данных от ИС-2 15 минут: журнал режима сварки не приходит", "administrator", "ADM-01", "", s.Alert.Time(), s.Restored.Time(), [2]string{"equipment", s.Equipment})
	}
	sort.SliceStable(m.Tasks, func(i, j int) bool { return m.Tasks[i].Created.Before(m.Tasks[j].Created) })
	for i, t := range m.Tasks {
		t.ID = fmt.Sprintf("TASK-%03d", i+1)
		if t.Kind == "decision_required" {
			t.Due = t.Created.Add(3 * 24 * time.Hour)
		}
	}
}
