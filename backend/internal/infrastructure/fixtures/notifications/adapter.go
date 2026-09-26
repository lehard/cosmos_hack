package notifications

import (
	"context"
	"slices"

	app "ant/internal/application/notifications"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
)

// Adapter — реализация fixtures ведущих портов модуля notifications (AD-36):
// сводка для шапки, «требует вашего внимания», лента тревог и задачи — из
// мира заготовок, по роли субъекта (ответ без роли — общий). Поверх мира —
// сессионное наложение (loader.Runtime.Record): отмеченная задача меняет
// состояние и уходит из открытых до сброса прогона. Адресность — на сервере,
// как у live: задачи персоны и её роли, а не общий ответ мира.
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// byRole — параметры с ролью субъекта (анонимный — без роли).
func byRole(ctx context.Context, kv ...string) map[string]string {
	out := map[string]string{}
	if p := platform.PrincipalFrom(ctx); !p.Anonymous() {
		out["role"] = p.Role
	}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i]] = kv[i+1]
	}
	return out
}

// Summary — сводка уведомлений для шапки (notifications.summary.read, FR-57):
// тревоги и информация — из мира, задачи и запросы решения — по адресным
// задачам субъекта с учётом отметок сессии.
func (a Adapter) Summary(ctx context.Context, m platform.Moment) (app.NotificationSummary, error) {
	s, err := respond[app.NotificationSummary](ctx, "notifications.summary.read", byRole(ctx), &m)
	if err != nil {
		return s, err
	}
	tl, err := a.Tasks(ctx, app.TaskFilter{State: "open"}, m, platform.Page{})
	if err != nil {
		return s, nil
	}
	if s.ByKind == nil {
		s.ByKind = &app.NotificationSummaryByKind{}
	}
	s.ByKind.Task, s.ByKind.DecisionRequest = 0, 0
	for _, t := range tl.Items {
		if decisionKind(t.Kind) {
			s.ByKind.DecisionRequest++
		} else {
			s.ByKind.Task++
		}
	}
	s.Unread = s.ByKind.Task + s.ByKind.DecisionRequest + s.ByKind.Alarm + s.ByKind.Info
	return s, nil
}

// decisionKind — вид задачи — запрос решения (как у live и фронта).
func decisionKind(kind string) bool {
	return kind == "decision_required" || kind == "review_after_new_data" || kind == "protection_basis_changed"
}

// Attention — блок «Требует вашего внимания» (notifications.attention.list, FR-8).
func (Adapter) Attention(ctx context.Context, m platform.Moment) (app.AttentionList, error) {
	return respond[app.AttentionList](ctx, "notifications.attention.list", byRole(ctx), &m)
}

// Alerts — лента тревог (notifications.alert.list).
func (Adapter) Alerts(ctx context.Context, m platform.Moment, _ platform.Page) (app.AlertList, error) {
	return respond[app.AlertList](ctx, "notifications.alert.list", byRole(ctx), &m)
}

// Tasks — задачи пользователя (notifications.task.list) с фильтром по
// состоянию и месту: ответ мира, поверх — отметки сессии, затем адресность.
func (Adapter) Tasks(ctx context.Context, f app.TaskFilter, m platform.Moment, _ platform.Page) (app.TaskList, error) {
	v, err := respond[app.TaskList](ctx, "notifications.task.list", byRole(ctx, "state", f.State, "location_id", f.LocationID), &m)
	if err != nil {
		return v, err
	}
	rt, err := loader.Default()
	if err != nil {
		return v, err
	}
	acks := acknowledged(ctx, rt, m)
	p := platform.PrincipalFrom(ctx)
	out := v.Items[:0]
	for _, x := range v.Items {
		if o, ok := acks[rt.Local(ctx, x.TaskID)]; ok && x.State == "open" {
			x.State = o
			x.Overdue = false
		}
		if !addressed(p, x) {
			continue
		}
		if f.State != "" && x.State != f.State {
			continue
		}
		if f.LocationID != "" && (x.LocationID == nil || *x.LocationID != f.LocationID) {
			continue
		}
		out = append(out, x)
	}
	v.Items = out
	return v, nil
}

// addressed — задача адресована субъекту (как live, notifications.viewer):
// задача персоны — только ей; задача роли без персоны — всем с этой ролью
// (с наследованием). Сверх live, по замыслу мира заготовок: руководитель
// (начальник ОТК, главный сварщик, начальник цеха) видит задачи подчинённой
// роли, а коллеги той же роли чужих персональных задач не видят. Анонимный
// (вызов без сеанса) — всё.
func addressed(p platform.Principal, t app.TaskEntry) bool {
	if p.Anonymous() {
		return true
	}
	if t.AssigneeID != nil && *t.AssigneeID != "" && *t.AssigneeID == p.PersonID {
		return true
	}
	if !slices.Contains(p.EffectiveRoles(), t.AssigneeRole) {
		return false
	}
	return t.AssigneeID == nil || *t.AssigneeID == "" || p.Role != t.AssigneeRole
}

// acknowledged — отметки задач сессии: id задачи (без префикса прогона) → итог.
func acknowledged(ctx context.Context, rt *loader.Runtime, m platform.Moment) map[string]string {
	out := map[string]string{}
	for _, f := range rt.Facts(ctx, &m, string(platform.EntityTask)) {
		if in, ok := f.Body.(app.AcknowledgeTask); ok {
			o := in.Outcome
			if o == "" {
				o = "done"
			}
			out[f.ID] = o
		}
	}
	return out
}

// AcknowledgeTask — отметить задачу (notifications.task.acknowledge): квитанция
// и отметка в сессии — задача получает состояние итога (выполнена, принята,
// отклонена) и уходит из открытых; сводка шапки пересчитывается.
func (Adapter) AcknowledgeTask(ctx context.Context, taskID string, in app.AcknowledgeTask) (platform.Receipt, error) {
	rt, err := loader.Default()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Record(ctx, "notifications.task.acknowledge", loader.ObjectRef{Kind: string(platform.EntityTask), ID: taskID}, in.CommandMeta(), in,
		loader.Change{Entity: string(platform.EntityTask), ID: "global"}, loader.Change{Entity: string(platform.EntityNotification), ID: "global"})
}
