package notifications

import (
	"slices"
	"strings"
	"time"

	"ant/internal/contracts/statuses"
	"ant/internal/domain/item"
	"ant/internal/domain/kernel"
	"ant/internal/domain/nonconformity"
	"ant/internal/domain/process"
	"ant/internal/domain/quality"
)

// Выведение обязательств и задач из состояния модулей-владельцев (AD-4,
// AD-40): notifications стоит в композиции последним и на каждом шаге свёртки
// читает Upstream — изоляцию и «физически не перемещено», несоответствия,
// предъявления, сдерживание по области риска и доп. проверки у nonconformity
// (эпик 21), запросы задач у quality (эпик 20, State.Requests вида task).
// Обязательство или задача, чьё основание у владельца исчезло, снимается на
// этом же шаге («срок снят», «задача снята»).

// derive — сверить обязательства и задачи состояния с тем, что следует из
// состояния владельцев на этом шаге; r — запись шага (причина снятия).
func (s *State) derive(r kernel.Record, env Env, up Upstream) {
	if s.ItemID == "" {
		return
	}
	at := Cause{EventID: r.EventID, At: r.OccurredAt}
	wantO, wantT := s.wanted(env, up, at)

	for _, w := range wantO {
		o := s.obligation(w.ID)
		if o == nil {
			w.Open = true
			s.Obligations = append(s.Obligations, w)
			continue
		}
		// Что меняется у действующего срока: срок владельца, чьего решения
		// ждёт, заголовок; где стоит изделие — запоминается при постановке.
		o.FirstDue, o.WaitsOn, o.Title, o.OwnerRole = w.FirstDue, w.WaitsOn, w.Title, w.OwnerRole
		if o.StepKey == "" {
			o.StepKey = w.StepKey
		}
		if o.LocationID == "" {
			o.LocationID = w.LocationID
		}
		if !o.Open {
			// Основание вернулось (например, изоляция снова без решения).
			o.Open, o.Cleared, o.Causes = true, nil, w.Causes
		}
	}
	for i := range s.Obligations {
		o := &s.Obligations[i]
		if o.Open && !slices.ContainsFunc(wantO, func(w Obligation) bool { return w.ID == o.ID }) {
			c := at
			o.Open, o.Cleared = false, &c
		}
	}

	for _, w := range wantT {
		t := s.task(w.ID)
		if t == nil {
			w.Open = true
			s.Tasks = append(s.Tasks, w)
			continue
		}
		// Адресат задачи следует за изделием: кто держит его сейчас (FR-57).
		t.Role, t.Person, t.LocationID, t.Title, t.DueAt = w.Role, w.Person, w.LocationID, w.Title, w.DueAt
		t.ItemLabel, t.Operation, t.StepKey = w.ItemLabel, w.Operation, w.StepKey
		if !t.Open {
			t.Open, t.Closed, t.Causes = true, nil, w.Causes
		}
	}
	for i := range s.Tasks {
		t := &s.Tasks[i]
		if t.Open && !slices.ContainsFunc(wantT, func(w Task) bool { return w.ID == t.ID }) {
			c := at
			t.Open, t.Closed = false, &c
		}
	}
}

// wanted — обязательства и задачи, которые следуют из состояния владельцев;
// cur — запись шага (основание нового срока, если у владельца нет своей).
func (s *State) wanted(env Env, up Upstream, cur Cause) ([]Obligation, []Task) {
	var os []Obligation
	var ts []Task
	subject := "item:" + s.ItemID
	label := item.LocalLabel(s.ItemID)
	if up.Item != nil {
		label = up.Item.DisplayLabel(s.ItemID)
	}
	obl := func(basis, kind, key, owner, title string, due time.Time, causes ...Cause) Obligation {
		return Obligation{
			ID: ObligationID(s.ItemID, basis+"/"+key), Key: basis + "/" + key, Kind: kind, Basis: basis,
			Subject: subject, ItemID: s.ItemID, OwnerRole: owner, StepKey: s.Where.StepKey, LocationID: s.Where.LocationID,
			Title: title, FirstDue: due.UTC(), Ladder: env.ladder(basis), Causes: causes,
		}
	}
	task := func(key, kind, role, title string, due *time.Time, causes ...Cause) Task {
		slot := kernel.Slot{RuleID: RuleTask, Subject: subject, TriggerKey: key}
		return Task{ID: TaskID(slot), Key: key, Kind: kind, Role: role, LocationID: s.Where.LocationID, Title: title,
			Subject: subject, DueAt: due, Causes: causes, ItemLabel: label}
	}

	// Сроки исполнителя процесса (эпик 17, process.State.Deadlines): окна
	// BPMN и нормы ожидания на точках предъявления — с obligation_id
	// процесса: «наступил срок» с ним же срабатывает таймер у process.
	gates := map[string]bool{}
	if p := up.Process; p != nil {
		var ds []process.Deadline
		if env.Process.Def != nil {
			ds = p.Deadlines(env.Process)
		} else {
			for _, tm := range p.Timers {
				due := tm.DueAt
				ds = append(ds, process.Deadline{ObligationID: tm.ObligationID, Kind: process.DeadlineTimer, StepKey: tm.StepKey, From: tm.ArmedAt, DueAt: &due})
			}
		}
		for _, d := range ds {
			c := cur // срок взведён записью этого шага (у существующего причины сохраняются)
			switch d.Kind {
			case process.DeadlineTimer:
				if d.DueAt == nil {
					continue
				}
				o := obl(BasisBPMNTimer, "bpmn_timer", d.ObligationID, RoleForeman, "Окно "+d.StepKey+": "+label, *d.DueAt, c)
				o.ID, o.StepKey, o.WaitsOn = d.ObligationID, d.StepKey, "item:"+s.ItemID
				os = append(os, o)
			case process.DeadlinePresentation:
				due := d.From.Add(d.Wait)
				if d.WorkDays > 0 {
					due = env.Calendar.AddWorkingDays(d.From, d.WorkDays)
				}
				o := obl(BasisPresentation, "presentation_wait", d.ObligationID, first(d.OwnerRole, RoleInspector),
					"Решение на точке предъявления "+d.StepKey+": "+label, due, c)
				o.ID, o.StepKey, o.WaitsOn = d.ObligationID, d.StepKey, "item:"+s.ItemID
				os = append(os, o)
				gates[d.StepKey] = true
			}
		}
	}

	// Задачи процесса: изделие вошло в шаг с действием человека — роль по
	// дорожке шага видит его в «Задачах»; действие продвигает токен, и задача
	// снимается на том же шаге свёртки (одно правило на все шаги BPMN).
	if p := up.Process; p != nil && env.Process.Def != nil {
		for _, h := range p.HumanSteps(env.Process) {
			t := task("process/"+h.Node+"/"+h.Operation, KindProcessStep, h.Role, stepTitle(h, label), h.DueAt, cur)
			t.Operation, t.StepKey = h.Operation, h.StepKey
			if h.Workshop != "" {
				t.LocationID = h.Workshop
			}
			ts = append(ts, t)
		}
	}

	if nc := up.Nonconformity; nc != nil {
		group := s.group(*nc)
		// FR-55: изолированное изделие ждёт решения; срок — по политике (3
		// рабочих дня по производственному календарю), срок из решения
		// «изолировать» — если задан (его считает nonconformity тем же календарём).
		if iso := nc.Isolation; iso != nil && !iso.Released {
			c := Cause{EventID: iso.EventID, At: iso.At}
			if !decided(*nc) {
				due := env.Calendar.AddWorkingDays(iso.At, env.DecisionWorkingDays)
				if iso.DecisionDueAt != nil {
					due = *iso.DecisionDueAt
				}
				o := obl(BasisIsolation, "nc_disposition", iso.EventID, RoleTechnologist, "Решение по изолированному изделию "+label, due, c)
				o.WaitsOn = group
				os = append(os, o)
				ts = append(ts, task("decide/"+iso.EventID, "decision_required", RoleTechnologist, o.Title, &o.FirstDue, c))
			}
			// FR-55: «изолировано в системе, физически не перемещено» —
			// задача с подтверждением тому, у кого изделие, и срок.
			if !iso.PhysicallyMoved {
				due := iso.At.Add(time.Duration(env.MoveWithinMin) * time.Minute)
				title := "Переместить в изолятор: " + label
				if iso.IsolatorLocationID != "" {
					title += " → " + iso.IsolatorLocationID
				}
				o := obl(BasisIsolationMove, "isolation_move", iso.EventID, RoleForeman, title, due, c)
				o.WaitsOn = group
				os = append(os, o)
				ts = append(ts, task("isolate_move/"+iso.EventID, "isolate_move", RoleForeman, title, &o.FirstDue, c))
			}
		}
		// Подтверждённое несоответствие без решения (изоляция его покрывает).
		if !nc.Isolated() {
			for _, n := range nc.NCs {
				if n.Status != nonconformity.StatusConfirmed || n.DispositionEventID != "" {
					continue
				}
				c := Cause{EventID: first(n.ConfirmedEventID, firstCause(n.Causes)), At: decisionAt(*nc, n.ConfirmedEventID, n.FoundAt)}
				o := obl(BasisNC, "nc_disposition", n.ID, RoleTechnologist, "Решение по несоответствию "+n.Number+" ("+label+")",
					env.Calendar.AddWorkingDays(c.At, env.DecisionWorkingDays), c)
				o.WaitsOn = group
				os = append(os, o)
				ts = append(ts, task("decide/"+n.ID, "decision_required", RoleTechnologist, o.Title, &o.FirstDue, c))
			}
		}
		// Сроки на точках предъявления (FR-8, FR-19): изделие ждёт контролёра.
		if p := nc.PendingPresentation(); p != nil && !gates[p.StepKey] {
			gate := first(p.ClosingPoint, p.StepKey)
			o := obl(BasisPresentation, "presentation_wait", p.EventID, RoleInspector, "Решение на точке предъявления "+gate+": "+label,
				p.At.Add(time.Duration(env.PresentationWaitMin)*time.Minute), Cause{EventID: p.EventID, At: p.At})
			o.StepKey, o.WaitsOn = p.StepKey, "item:"+s.ItemID
			os = append(os, o)
		}
		// FR-62: изделие в области риска инцидента на блоке или доп. проверке —
		// срок решения по области и адресная задача тому, у кого изделие
		// физически находится (место — по фактам перемещения и выполнения).
		for _, c := range nc.Containment {
			if c.Rule != nonconformity.RuleIncidentScope || c.Released {
				continue
			}
			// Решение по области для изделия уже принято: исключено с основанием
			// (основание — «наблюдать» или «снять») или решено несоответствие
			// изделия — срока «решение по области» и задачи нет; сам блок, если
			// остался, снимает человек (AD-27).
			if b := statuses.Containment(c.Basis); b == statuses.ContainmentObserve || b == statuses.ContainmentNone || decided(*nc) {
				continue
			}
			incident := strings.TrimPrefix(c.Key, "incident:")
			kind, what := "", ""
			switch statuses.Containment(c.Level) {
			case statuses.ContainmentItemHold, statuses.ContainmentLotHold:
				kind, what = "physical_move", "остановить и отложить (блок)"
			case statuses.ContainmentAdditionalCheck:
				kind, what = "recheck", "доп. проверка"
			default:
				continue
			}
			cause := cur
			if c.BasisEventID != "" {
				cause = Cause{EventID: c.BasisEventID, At: c.At}
			}
			o := obl(BasisIncidentScope, "containment_review", c.Key, RoleHeadOfQC, "Решение по области риска "+incident+": "+label,
				env.Calendar.AddWorkingDays(c.At, env.DecisionWorkingDays), cause)
			o.WaitsOn = c.Key
			os = append(os, o)
			ts = append(ts, task("incident/"+c.Key+"/"+kind, kind, RoleForeman, "Изделие "+label+" в области риска "+incident+": "+what, nil, cause))
		}
		// Д-81: приёмка отозвана пересмотром — пока блок человека по отзыву не
		// снят, задача тому, у кого изделие: остановить и отложить до решения.
		for _, p := range nc.Presentations {
			if !p.Revoked() || !slices.ContainsFunc(nc.Containment, func(c nonconformity.ContainmentSource) bool {
				return c.Key == p.ReviewEventID && !c.Released
			}) {
				continue
			}
			c := Cause{EventID: p.ReviewEventID, At: decisionAt(*nc, p.ReviewEventID, cur.At)}
			ts = append(ts, task("revoked/"+p.ReviewEventID, "physical_move", RoleForeman,
				"Приёмка "+first(p.ClosingPoint, p.StepKey)+" отозвана: остановить и отложить "+label+" до решения", nil, c))
		}
		// FR-52: назначенная доп. проверка — срок, пока нет нового результата контроля.
		for _, rc := range nc.Rechecks {
			at := decisionAt(*nc, rc.EventID, time.Time{})
			if at.IsZero() || rc.Done || s.LastInspectionAt.After(at) {
				continue
			}
			due, ok := ParseTime(rc.DueAt)
			if !ok {
				due = env.Calendar.AddWorkingDays(at, env.RecheckWorkingDays)
			}
			o := obl(BasisRecheck, "recheck", rc.EventID, RoleInspector, "Доп. проверка ("+rc.Method+"): "+label, due, Cause{EventID: rc.EventID, At: at})
			o.WaitsOn = "item:" + s.ItemID
			os = append(os, o)
		}
	}

	// Задачи по запросам quality (эпик 20, Д-43): recheck, isolate_move,
	// decision_required, inspection_missing — роль из запроса.
	if q := up.Quality; q != nil {
		for _, rq := range q.Requests {
			if rq.Kind != quality.RequestTask || rq.RoleID == "" || rq.TaskKind == "" {
				continue
			}
			cs := make([]Cause, 0, len(rq.Causes))
			for _, id := range rq.Causes {
				cs = append(cs, Cause{EventID: id, At: s.At})
			}
			t := task("quality/"+rq.Key, taskKind(rq.TaskKind), rq.RoleID, rq.Title, nil, cs...)
			if rq.TaskKind == "recheck" {
				// Доп. проверка по изделию (R-01, R-02: «оценка невозможна»):
				// кнопка — назначить её по изделию, без несоответствия; назначенная
				// доп. проверка задачу снимает (дальше — срок доп. проверки).
				if up.Nonconformity != nil && slices.ContainsFunc(up.Nonconformity.Rechecks, func(rc nonconformity.Recheck) bool { return !rc.Done }) {
					continue
				}
				t.Operation, t.StepKey = OpRecheckRequest, rq.StepKey
				t.Title = "Доп. проверка " + label + ": " + rq.Title
			}
			ts = append(ts, t)
		}
	}
	return os, ts
}

// OpRecheckRequest — операция «Назначить доп. проверку» по изделию.
const OpRecheckRequest = "nonconformity.recheck.request"

// KindProcessStep — вид задачи «шаг процесса ждёт действия человека».
const KindProcessStep = "process_step"

// stepTitle — что сделать, для людей: «Принять в цех Ф-001».
func stepTitle(h process.HumanStep, label string) string {
	name := h.Name
	if name == "" {
		name = h.StepKey
	}
	switch h.Operation {
	case process.OpMovementReceive:
		return "Принять в цех " + label
	case process.OpMovementSend:
		return "Отправить " + label + ": " + name
	case process.OpOperationStart:
		return "Начать: " + name + " — " + label
	case process.OpOperationFinish:
		return "Завершить: " + name + " — " + label
	}
	return name + " — " + label
}

// group — чьего решения ждут сроки изделия: инцидента, если изделие в его
// области на блоке (решение по области снимает блок всем изделиям сразу),
// иначе самого изделия. По группе сводится цена задержки (FR-8).
func (s *State) group(nc nonconformity.State) string {
	var keys []string
	for _, c := range nc.Containment {
		if c.Rule == nonconformity.RuleIncidentScope && !c.Released &&
			(statuses.Containment(c.Level) == statuses.ContainmentItemHold || statuses.Containment(c.Level) == statuses.ContainmentLotHold) {
			keys = append(keys, c.Key)
		}
	}
	if len(keys) == 0 {
		return "item:" + s.ItemID
	}
	slices.Sort(keys)
	return keys[0]
}

// decided — по изолированному изделию решение принято: у несоответствия
// решение (или оно закрыто).
func decided(nc nonconformity.State) bool {
	for _, n := range nc.NCs {
		if n.DispositionEventID != "" || n.Status == nonconformity.StatusClosed {
			return true
		}
	}
	return false
}

// decisionAt — время решения человека по event_id (def — если не найдено).
func decisionAt(nc nonconformity.State, eventID string, def time.Time) time.Time {
	for _, d := range nc.Decisions {
		if d.EventID == eventID {
			return d.At
		}
	}
	return def
}

// taskKind — вид задачи из перечисления контракта task.task.created
// (незнакомый вид запроса — other).
func taskKind(k string) string {
	switch k {
	case "physical_move", "isolate_move", "recheck", "decision_required", "review_after_new_data", "protection_basis_changed",
		"resign", "remark_carrier", "remove_temporary_carrier", "inspection_missing", "admin_resend", "process_step", "other":
		return k
	}
	return "other"
}

func firstCause(cs []string) string {
	if len(cs) == 0 {
		return ""
	}
	return cs[0]
}
