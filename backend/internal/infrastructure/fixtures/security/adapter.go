package security

import (
	"context"
	"strconv"

	"ant/internal/application/platform"
	app "ant/internal/application/security"
)

// Adapter — реализация fixtures ведущих портов модуля security (AD-36):
// индикатор целостности «по данным сервера», отчёты верификатора, журнал
// критических действий и шина безопасности — из мира заготовок на шаге
// курсора (в главной истории нарушение появляется после подмены записи, S09).
type Adapter struct {
	// Unimplemented — операции, которых нет в мире заготовок, отвечают 501.
	app.Unimplemented
}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Integrity — состояние целостности журнала (security.integrity.read, AD-46).
func (Adapter) Integrity(ctx context.Context) (app.IntegrityStatus, error) {
	return respond[app.IntegrityStatus](ctx, "security.integrity.read", nil, nil)
}

// CriticalActions — журнал критических действий (security.critical_action.list,
// AD-28): отбор по группе, кто и объекту, страница по номеру CA — как у live.
func (Adapter) CriticalActions(ctx context.Context, f app.CriticalActionFilter, m platform.Moment, p platform.Page) (app.CriticalActionList, error) {
	v, err := respond[app.CriticalActionList](ctx, "security.critical_action.list", nil, &m)
	if err != nil {
		return v, err
	}
	before, _ := strconv.ParseInt(p.Cursor, 10, 64)
	limit := p.Limit
	if limit == 0 {
		limit = 50
	}
	out := app.CriticalActionList{Items: []app.CriticalAction{}}
	for _, x := range v.Items {
		if before > 0 && x.CANo >= before {
			continue
		}
		if (f.Group != "" && x.CAGroup != f.Group) || (f.ActorID != "" && x.ActorID != f.ActorID) {
			continue
		}
		if f.Object != nil && (x.Object.Entity != f.Object.Entity || (f.Object.ID != "" && x.Object.ID != f.Object.ID)) {
			continue
		}
		if len(out.Items) == limit {
			out.NextCursor = strconv.FormatInt(out.Items[len(out.Items)-1].CANo, 10)
			break
		}
		out.Items = append(out.Items, x)
	}
	return out, nil
}

// CriticalAction — запись CA-‹n› (security.critical_action.read).
func (Adapter) CriticalAction(ctx context.Context, caRef string) (app.CriticalAction, error) {
	return respond[app.CriticalAction](ctx, "security.critical_action.read", map[string]string{"ca_ref": caRef}, nil)
}

// Events — шина безопасности (security.event.list, AD-24): отбор по типу,
// страница по seq — как у live.
func (Adapter) Events(ctx context.Context, eventType string, m platform.Moment, p platform.Page) (app.SecurityEventList, error) {
	v, err := respond[app.SecurityEventList](ctx, "security.event.list", nil, &m)
	if err != nil {
		return v, err
	}
	before, _ := strconv.ParseInt(p.Cursor, 10, 64)
	limit := p.Limit
	if limit == 0 {
		limit = 50
	}
	out := app.SecurityEventList{Items: []app.SecurityEvent{}}
	for _, x := range v.Items {
		if (eventType != "" && x.EventType != eventType) || (before > 0 && x.Seq >= before) {
			continue
		}
		if len(out.Items) == limit {
			out.NextCursor = strconv.FormatInt(out.Items[len(out.Items)-1].Seq, 10)
			break
		}
		out.Items = append(out.Items, x)
	}
	return out, nil
}

// VerifierReports — отчёты верификатора, новые сверху (security.verifier_report.list, AD-46).
func (Adapter) VerifierReports(ctx context.Context, p platform.Page) (app.VerifierReportList, error) {
	v, err := respond[app.VerifierReportList](ctx, "security.verifier_report.list", nil, nil)
	if err != nil {
		return v, err
	}
	if p.Limit > 0 && len(v.Items) > p.Limit {
		v.Items = v.Items[:p.Limit]
	}
	return v, nil
}

// VerifierReport — отчёт верификатора целиком (security.verifier_report.read).
func (Adapter) VerifierReport(ctx context.Context, reportDigest string) (app.VerifierReport, error) {
	return respond[app.VerifierReport](ctx, "security.verifier_report.read", map[string]string{"report_digest": reportDigest}, nil)
}
