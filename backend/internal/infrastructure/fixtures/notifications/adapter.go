package notifications

import (
	"context"

	app "ant/internal/application/notifications"
	"ant/internal/application/platform"
)

// Adapter — реализация fixtures ведущих портов модуля notifications (AD-36):
// сводка для шапки, «требует вашего внимания», лента тревог и задачи — из
// мира заготовок, по роли субъекта (ответ без роли — общий).
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

// Summary — сводка уведомлений для шапки (notifications.summary.read, FR-57).
func (Adapter) Summary(ctx context.Context, m platform.Moment) (app.NotificationSummary, error) {
	return respond[app.NotificationSummary](ctx, "notifications.summary.read", byRole(ctx), &m)
}

// Attention — блок «Требует вашего внимания» (notifications.attention.list, FR-8).
func (Adapter) Attention(ctx context.Context, m platform.Moment) (app.AttentionList, error) {
	return respond[app.AttentionList](ctx, "notifications.attention.list", byRole(ctx), &m)
}

// Alerts — лента тревог (notifications.alert.list).
func (Adapter) Alerts(ctx context.Context, m platform.Moment, _ platform.Page) (app.AlertList, error) {
	return respond[app.AlertList](ctx, "notifications.alert.list", byRole(ctx), &m)
}

// Tasks — задачи пользователя (notifications.task.list) с фильтром по состоянию и месту.
func (Adapter) Tasks(ctx context.Context, f app.TaskFilter, m platform.Moment, _ platform.Page) (app.TaskList, error) {
	v, err := respond[app.TaskList](ctx, "notifications.task.list", byRole(ctx, "state", f.State, "location_id", f.LocationID), &m)
	if err != nil {
		return v, err
	}
	out := v.Items[:0]
	for _, x := range v.Items {
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

// AcknowledgeTask — отметить задачу (notifications.task.acknowledge).
func (Adapter) AcknowledgeTask(ctx context.Context, taskID string, in app.AcknowledgeTask) (platform.Receipt, error) {
	return decide(ctx, "notifications.task.acknowledge", "task", taskID, in.CommandMeta())
}
