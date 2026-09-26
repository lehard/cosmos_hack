package ops

import (
	"context"
	"strings"
	"sync"
	"time"

	app "ant/internal/application/ops"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/ops"
)

// Adapter — реализация fixtures ведущих портов модуля ops (AD-36): здоровье
// компонентов, остановленные изделия, настройки — из мира заготовок.
// Решения об интеграциях (эпик 48) мир не меняют (FR-129), но экран
// «Интеграции» стенда в режиме заготовок показывает их до перезапуска
// процесса: наложение в памяти поверх ответа мира.
type Adapter struct {
	mu        sync.Mutex
	overrides map[string]app.IntegrationDecision
}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Health — состояние компонентов (ops.health.read, FR-127).
func (*Adapter) Health(ctx context.Context) (app.OpsHealth, error) {
	return respond[app.OpsHealth](ctx, "ops.health.read", nil, nil)
}

// StoppedItems — изделия с остановленной обработкой (ops.stopped_item.list).
func (*Adapter) StoppedItems(ctx context.Context, _ platform.Page) (app.StoppedItemList, error) {
	return respond[app.StoppedItemList](ctx, "ops.stopped_item.list", nil, nil)
}

// Settings — настройки (ops.setting.list).
func (*Adapter) Settings(ctx context.Context) (app.SettingList, error) {
	return respond[app.SettingList](ctx, "ops.setting.list", nil, nil)
}

// RetryProcessing — повторить обработку изделия (ops.processing.retry).
func (*Adapter) RetryProcessing(ctx context.Context, itemID string, in app.RetryProcessing) (platform.Receipt, error) {
	return decide(ctx, "ops.processing.retry", "item", itemID, in.CommandMeta())
}

// DisableSource — отключить источник (ops.source.disable).
func (*Adapter) DisableSource(ctx context.Context, sourceID string, in app.SwitchSource) (platform.Receipt, error) {
	return decide(ctx, "ops.source.disable", "quarantine", sourceID, in.CommandMeta())
}

// EnableSource — включить источник (ops.source.enable).
func (*Adapter) EnableSource(ctx context.Context, sourceID string, in app.SwitchSource) (platform.Receipt, error) {
	return decide(ctx, "ops.source.enable", "quarantine", sourceID, in.CommandMeta())
}

// Integrations — экран «Интеграции» (ops.integration.list, FR-157): ответ
// мира заготовок и поверх — решения этого процесса.
func (a *Adapter) Integrations(ctx context.Context) (app.IntegrationList, error) {
	l, err := respond[app.IntegrationList](ctx, "ops.integration.list", nil, nil)
	if err != nil {
		return l, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, e := range l.Items {
		if d, ok := a.overrides[e.System]; ok {
			dd := d
			l.Items[i].State, l.Items[i].Default, l.Items[i].LastDecision, l.Items[i].BasisSeq = d.State, false, &dd, d.Seq
			if d.State == dom.SwitchDisabled && e.Channel != nil {
				l.Items[i].Channel = ptr(dom.IntegrationDisabled)
			} else if e.Channel != nil && *e.Channel == dom.IntegrationDisabled {
				l.Items[i].Channel = ptr(dom.IntegrationOK)
			}
		}
	}
	return l, nil
}

// SetIntegration — включить, выключить, «стенд ↔ реальная» (ops.integration.set):
// тот же доменный гард, что у live (профиль fixtures — стенд разрешён);
// квитанция — как у live, решение запоминается в памяти процесса.
func (a *Adapter) SetIntegration(ctx context.Context, system string, in app.SetIntegrationState) (platform.Receipt, error) {
	l, err := a.Integrations(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	var cur *app.IntegrationEntry
	for i := range l.Items {
		if l.Items[i].System == system {
			cur = &l.Items[i]
		}
	}
	if cur == nil {
		return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", "Интеграция", "id", system)
	}
	st := dom.SwitchState{System: system, Installed: cur.Installed, Stand: cur.StandAvailable, Real: cur.RealAvailable, State: cur.State}
	if rej := dom.CheckSet(l.Profile, st, in.State); rej != nil {
		e := platform.Fail(errcodes.Code(rej.Code), "system", system, "state", in.State, "profile", l.Profile)
		return platform.Receipt{}, e
	}
	rc, err := decide(ctx, "ops.integration.set", "integrity", system, in.CommandMeta())
	if err != nil {
		return rc, err
	}
	actor := strings.ToLower(platform.PrincipalFrom(ctx).PersonID)
	a.mu.Lock()
	if a.overrides == nil {
		a.overrides = map[string]app.IntegrationDecision{}
	}
	at := rc.RecordedAt
	if at.IsZero() {
		at = time.Now().UTC()
	}
	a.overrides[system] = app.IntegrationDecision{State: in.State, Previous: ptr(cur.State), Reason: in.Reason.Text, Actor: actor, At: at, Seq: rc.Seq}
	a.mu.Unlock()
	return rc, nil
}

// CheckIntegration — проверить соединение (ops.integration.check).
func (a *Adapter) CheckIntegration(ctx context.Context, system string, in app.CheckIntegration) (platform.Receipt, error) {
	return decide(ctx, "ops.integration.check", "integrity", system, in.CommandMeta())
}

func ptr[T any](v T) *T { return &v }
