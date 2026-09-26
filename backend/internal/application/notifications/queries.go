package notifications

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/application/platform"
	itemdom "ant/internal/domain/item"
	dom "ant/internal/domain/nonconformity"
	notif "ant/internal/domain/notifications"
)

// Чтение модуля notifications (FR-8, FR-57): адресно — только тем, у кого
// есть действие: задачи — исполнителю (роль с наследованием — Principal.EffectiveRoles, или псевдоним),
// тревоги и «требует вашего внимания» — владельцу срока и тем, кому он
// эскалирован, плюс обзорным ролям. Без сеанса (анонимный запрос) — всё.

// overview — роли, которым лента тревог и блок «требует вашего внимания»
// видны целиком (руководитель производства, начальник ОТК).
var overview = map[string]bool{"production_manager": true, "head_of_qc": true}

// viewer — субъект чтения.
type viewer struct {
	all    bool
	roles  []string
	person string
	// p — субъект сеанса: его область сужает задачи с местом (InScope).
	p platform.Principal
}

func viewerOf(ctx context.Context) viewer {
	p := platform.PrincipalFrom(ctx)
	if p.Anonymous() {
		return viewer{all: true}
	}
	// Роль с наследованием (начальник ОТК видит задачи контролёра, начальник
	// цеха — мастера) — из сеанса: одна функция наследования access (AD-15).
	return viewer{roles: p.EffectiveRoles(), person: p.PersonID, all: false, p: p}
}

// sees — задача видна субъекту: адресована ему и её место — в его области
// (InScope; задачи без места — только по адресности).
func (v viewer) sees(t notif.TaskRecord, places Places) bool {
	return v.addressed(t.Role, t.Person) && (v.all || InScope(v.p, t.LocationID, places))
}

// addressed — задача или уведомление адресовано субъекту.
func (v viewer) addressed(role, person string) bool {
	if v.all {
		return true
	}
	if person != "" {
		return person == v.person || (v.person == "" && slices.Contains(v.roles, role))
	}
	return slices.Contains(v.roles, role)
}

// oversees — срок виден субъекту: владелец, адресат эскалации или обзорная роль.
func (v viewer) oversees(o notif.ObligationRecord) bool {
	if v.all || (len(v.roles) > 0 && overview[v.roles[0]]) || slices.Contains(v.roles, o.OwnerRole) {
		return true
	}
	for _, e := range o.Escalations {
		if slices.Contains(v.roles, e.Role) {
			return true
		}
	}
	return false
}

// inRun — строка относится к прогону запроса (AD-38); без прогона — все.
func inRun(m platform.Moment, runID string) bool { return m.RunID == "" || m.RunID == runID }

// Summary — сводка для шапки (notifications.summary.read, FR-57):
// непрочитанные по видам — открытые задачи, запросы решения, тревоги и
// информация за сутки.
func (s *Service) Summary(ctx context.Context, m platform.Moment) (NotificationSummary, error) {
	if !s.live() {
		return s.Unimplemented.Summary(ctx, m)
	}
	now, err := s.now(ctx, m)
	if err != nil {
		return NotificationSummary{}, err
	}
	v := viewerOf(ctx)
	by := NotificationSummaryByKind{}
	ts, err := s.tasks(ctx)
	if err != nil {
		return NotificationSummary{}, err
	}
	for _, t := range ts {
		if t.State != notif.TaskOpen || !inRun(m, t.RunID) || t.CreatedAt.After(now) || !v.sees(t, s.cfg.Places) {
			continue
		}
		if decisionKind(t.Kind) {
			by.DecisionRequest++
		} else {
			by.Task++
		}
	}
	al, err := s.alerts(ctx, m, now, v)
	if err != nil {
		return NotificationSummary{}, err
	}
	by.Alarm = len(al)
	ns, err := s.notices(ctx)
	if err != nil {
		return NotificationSummary{}, err
	}
	for _, n := range ns {
		if n.State != notif.NoticeActive || !inRun(m, n.RunID) || n.At.After(now) || now.Sub(n.At) > 24*time.Hour || !v.addressed(n.Role, n.Person) {
			continue
		}
		switch n.Severity {
		case "info":
			by.Info++
		case "decision_request":
			by.DecisionRequest++
		}
	}
	return NotificationSummary{Unread: by.Info + by.Alarm + by.Task + by.DecisionRequest, ByKind: &by}, nil
}

// decisionKind — задача — запрос решения (FR-57: «запрос решения — срок, эскалация»).
func decisionKind(kind string) bool {
	switch kind {
	case "decision_required", "review_after_new_data", "protection_basis_changed":
		return true
	}
	return false
}

// Attention — «требует вашего внимания» (FR-8): просроченные решения с ценой
// задержки — по одной строке на то, чьего решения ждут (изделие,
// несоответствие, инцидент): на сколько просрочено, сколько изделий и
// операций стоят.
func (s *Service) Attention(ctx context.Context, m platform.Moment) (AttentionList, error) {
	if !s.live() {
		return s.Unimplemented.Attention(ctx, m)
	}
	now, err := s.now(ctx, m)
	if err != nil {
		return AttentionList{}, err
	}
	v := viewerOf(ctx)
	all, err := s.obligations(ctx)
	if err != nil {
		return AttentionList{}, err
	}
	all = slices.DeleteFunc(all, func(o notif.ObligationRecord) bool { return !inRun(m, o.RunID) || o.SetAt.After(now) })
	type group struct {
		minutes int
		first   notif.ObligationRecord
	}
	groups := map[string]*group{}
	for _, o := range all {
		if !decisionBasis(o.Basis) || !o.Overdue(now) || !v.oversees(o) {
			continue
		}
		g := groups[o.WaitsOn]
		if g == nil {
			g = &group{first: o}
			groups[o.WaitsOn] = g
		}
		g.minutes = max(g.minutes, o.OverdueMinutes(now))
		if o.FirstDueAt.Before(g.first.FirstDueAt) {
			g.first = o
		}
	}
	out := AttentionList{Items: []AttentionEntry{}}
	for _, k := range sortedKeys(groups) {
		g := groups[k]
		items, ops := notif.Cost(all, k)
		mins := g.minutes
		target := g.first.Title
		kind, id := refOf(k)
		if kind == platform.EntityIncident {
			target = "Инцидент " + id
		}
		out.Items = append(out.Items, AttentionEntry{Kind: "overdue_decision", EntryID: "ATT-" + k, Target: &target,
			OverdueMinutes: &mins, Items: &items, Operations: &ops, Ref: &platform.DrillRef{Entity: kind, ID: id}})
	}
	// Сначала самые дорогие по задержке: больше стоящих изделий, дольше просрочено.
	slices.SortStableFunc(out.Items, func(a, b AttentionEntry) int {
		if *a.Items != *b.Items {
			return *b.Items - *a.Items
		}
		return *b.OverdueMinutes - *a.OverdueMinutes
	})
	return out, nil
}

// decisionBasis — срок ждёт решения человека (блок «требует вашего внимания»).
func decisionBasis(b string) bool {
	switch b {
	case notif.BasisIsolation, notif.BasisNC, notif.BasisIncidentScope, notif.BasisPresentation:
		return true
	}
	return false
}

// refOf — куда провалиться по «чьего решения ждут» (FR-7).
func refOf(stream string) (platform.EntityKind, string) {
	kind, id, ok := strings.Cut(stream, ":")
	if !ok {
		return platform.EntityItem, stream
	}
	switch kind {
	case "incident":
		return platform.EntityIncident, id
	case "nonconformity":
		return platform.EntityNonconformity, id
	}
	return platform.EntityItem, id
}

// Alerts — лента тревог (FR-8, FR-55): просроченные изоляции, «не перемещено
// в изолятор» после срока, точки предъявления сверх срока, эскалации — с
// ценой задержки; новые сверху.
func (s *Service) Alerts(ctx context.Context, m platform.Moment, p platform.Page) (AlertList, error) {
	if !s.live() {
		return s.Unimplemented.Alerts(ctx, m, p)
	}
	now, err := s.now(ctx, m)
	if err != nil {
		return AlertList{}, err
	}
	items, err := s.alerts(ctx, m, now, viewerOf(ctx))
	if err != nil {
		return AlertList{}, err
	}
	page, next := paginate(items, p)
	return AlertList{Items: page, NextCursor: next}, nil
}

// alerts — действующие тревоги субъекта: сроки, для которых записано
// «наступил срок», пока основание не снято.
func (s *Service) alerts(ctx context.Context, m platform.Moment, now time.Time, v viewer) ([]AlertEntry, error) {
	all, err := s.obligations(ctx)
	if err != nil {
		return nil, err
	}
	all = slices.DeleteFunc(all, func(o notif.ObligationRecord) bool { return !inRun(m, o.RunID) || o.SetAt.After(now) })
	out := []AlertEntry{}
	for _, o := range all {
		if o.State != notif.ObligationOpen || len(o.Reached) == 0 || !v.oversees(o) {
			continue
		}
		items, ops := notif.Cost(all, o.WaitsOn)
		mins := o.OverdueMinutes(now)
		a := AlertEntry{AlertID: "AL-" + o.ObligationID, At: o.FirstDueAt, Kind: alertKind(o.Basis), OverdueMinutes: &mins, Items: &items, Operations: &ops}
		switch a.Kind {
		case "overdue_isolation", "not_moved_to_isolator":
			a.Item = strp(o.ItemID)
			a.Ref = &platform.DrillRef{Entity: platform.EntityItem, ID: o.ItemID}
		case "gate_overdue":
			a.Item = strp(o.ItemID)
			gate := dom.DefaultClosingPoints[o.StepKey]
			if gate == "" {
				gate = o.StepKey
			}
			a.Gate = strp(gate)
			a.Ref = &platform.DrillRef{Entity: platform.EntityItem, ID: o.ItemID}
		default:
			a.Target = strp(o.Title)
			kind, id := refOf(o.WaitsOn)
			a.Ref = &platform.DrillRef{Entity: kind, ID: id}
		}
		out = append(out, a)
	}
	slices.SortStableFunc(out, func(a, b AlertEntry) int {
		if c := b.At.Compare(a.At); c != 0 {
			return c
		}
		return strings.Compare(a.AlertID, b.AlertID)
	})
	return out, nil
}

// alertKind — вид тревоги ленты по основанию срока (AlertEntry.kind).
func alertKind(basis string) string {
	switch basis {
	case notif.BasisIsolation:
		return "overdue_isolation"
	case notif.BasisIsolationMove:
		return "not_moved_to_isolator"
	case notif.BasisPresentation:
		return "gate_overdue"
	}
	return "escalation"
}

// Tasks — задачи и уведомления пользователя (notifications.task.list, FR-57):
// адресные и в области сеанса (место задачи входит в область персоны,
// InScope), со сроком и признаком просрочки; новые сверху.
func (s *Service) Tasks(ctx context.Context, f TaskFilter, m platform.Moment, p platform.Page) (TaskList, error) {
	if !s.live() {
		return s.Unimplemented.Tasks(ctx, f, m, p)
	}
	now, err := s.now(ctx, m)
	if err != nil {
		return TaskList{}, err
	}
	v := viewerOf(ctx)
	ts, err := s.tasks(ctx)
	if err != nil {
		return TaskList{}, err
	}
	out := []TaskEntry{}
	for _, t := range ts {
		if !inRun(m, t.RunID) || t.CreatedAt.After(now) || !v.sees(t, s.cfg.Places) {
			continue
		}
		if f.State != "" && t.State != f.State {
			continue
		}
		if f.LocationID != "" && t.LocationID != f.LocationID {
			continue
		}
		e := TaskEntry{TaskID: t.TaskID, Kind: t.Kind, Title: t.Title, State: t.State, AssigneeRole: t.Role, CreatedAt: t.CreatedAt,
			DueAt: t.DueAt, Overdue: t.State == notif.TaskOpen && t.DueAt != nil && t.DueAt.Before(now)}
		if t.Person != "" {
			e.AssigneeID = strp(t.Person)
		}
		if t.LocationID != "" {
			e.LocationID = strp(t.LocationID)
		}
		kind, id := refOf(t.Subject)
		e.Ref = &platform.DrillRef{Entity: kind, ID: id}
		if t.ItemID != "" && kind == platform.EntityItem {
			e.ItemID = strp(t.ItemID)
			label := t.ItemLabel
			if label == "" {
				label = itemdom.LocalLabel(t.ItemID)
			}
			e.ItemLabel = strp(label)
		}
		if t.Operation != "" {
			e.OperationID = strp(t.Operation)
		}
		if t.StepKey != "" {
			e.StepKey = strp(t.StepKey)
		}
		out = append(out, e)
	}
	slices.SortStableFunc(out, func(a, b TaskEntry) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.TaskID, b.TaskID)
	})
	page, next := paginate(out, p)
	return TaskList{Items: page, NextCursor: next}, nil
}

// paginate — страница списка: курсор — смещение.
func paginate[T any](items []T, p platform.Page) ([]T, string) {
	off, _ := strconv.Atoi(p.Cursor)
	if off < 0 || off > len(items) {
		off = len(items)
	}
	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}
	end := min(off+limit, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[off:end], next
}

func strp(s string) *string { return &s }

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
