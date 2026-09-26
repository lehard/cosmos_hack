package nonconformity

import (
	"context"
	"fmt"
	"slices"

	app "ant/internal/application/nonconformity"
	"ant/internal/application/platform"
	"ant/internal/infrastructure/fixtures/loader"
)

// Сессионное наложение решений контролёра и комиссии (FR-129, FR-19, FR-51):
// решение на точке предъявления, пересмотр, отклонение сигнала, подтверждение
// и решение по несоответствию мир заготовок не меняют, но стенд показывает их
// до сброса прогона — строка уходит из очереди «Ждут моего решения» или
// меняет группу, карточка показывает решение и новый статус.

// record — решение над объектом kind/id: квитанция и факт сессии.
func record(ctx context.Context, op, kind, id string, meta platform.CommandMeta, body any) (platform.Receipt, error) {
	rt, err := loader.Default()
	if err != nil {
		return platform.Receipt{}, err
	}
	touched := []loader.Change{{Entity: string(platform.EntityNonconformity), ID: "global"}, {Entity: string(platform.EntityTask), ID: "global"},
		{Entity: string(platform.EntityNotification), ID: "global"}}
	if kind != string(platform.EntityItem) && kind != "" {
		touched = append(touched, loader.Change{Entity: string(platform.EntityItem), ID: "global"})
	}
	return rt.Record(ctx, op, loader.ObjectRef{Kind: kind, ID: id}, meta, body, touched...)
}

// decisions — решения сессии на момент m: вид/id объекта (без префикса прогона) → факты.
type decisions struct {
	rt    *loader.Runtime
	ctx   context.Context
	facts map[string][]loader.Fact
}

func sessionDecisions(ctx context.Context, m platform.Moment) decisions {
	d := decisions{ctx: ctx, facts: map[string][]loader.Fact{}}
	rt, err := loader.Default()
	if err != nil {
		return d
	}
	d.rt = rt
	for _, f := range rt.Facts(ctx, &m, string(platform.EntityItem), string(platform.EntityNonconformity), "lot") {
		k := f.Kind + "/" + f.ID
		d.facts[k] = append(d.facts[k], f)
	}
	return d
}

// of — факты над объектом kind/id (id с префиксом прогона или без).
func (d decisions) of(kind, id string) []loader.Fact {
	if d.rt == nil || id == "" {
		return nil
	}
	return d.facts[kind+"/"+d.rt.Local(d.ctx, id)]
}

// has — над объектом есть решение одной из операций ops.
func (d decisions) has(kind, id string, ops ...string) bool {
	return slices.ContainsFunc(d.of(kind, id), func(f loader.Fact) bool { return slices.Contains(ops, f.Op) })
}

// queueRow — строка очереди с наложением: false — строка ушла из очереди.
func (d decisions) queueRow(x app.DecisionQueueRow) (app.DecisionQueueRow, bool) {
	item := string(platform.EntityItem)
	nc := ""
	if x.NCID != nil {
		nc = *x.NCID
	}
	switch x.Kind {
	case "presentation":
		return x, !d.has(item, x.ItemID, opResolve)
	case "review":
		return x, !d.has(item, x.ItemID, opReview)
	case "signal":
		if d.has(item, x.ItemID, opReject, opRecheck) || d.has(string(platform.EntityNonconformity), nc, opConfirm, opDisposition, opClose) {
			return x, false
		}
		if d.has(item, x.ItemID, opIsolate) {
			x.Kind = "isolated"
		}
	case "isolated":
		if d.has(string(platform.EntityNonconformity), nc, opDisposition, opClose) || d.has(item, x.ItemID, opRelease) {
			return x, false
		}
	}
	return x, true
}

// Операции решений.
const (
	opResolve     = "nonconformity.presentation.resolve"
	opReview      = "nonconformity.presentation.review"
	opReject      = "nonconformity.signal.reject"
	opRecheck     = "nonconformity.recheck.request"
	opIsolate     = "nonconformity.item.isolate"
	opConfirm     = "nonconformity.nonconformity.confirm"
	opDisposition = "nonconformity.disposition.set"
	opVerify      = "nonconformity.disposition.verify"
	opClose       = "nonconformity.nonconformity.close"
	opContainment = "nonconformity.containment.set"
	opRelease     = "nonconformity.containment.release"
)

// decisionRef — решение сессии строкой «решения людей» карточки.
func decisionRef(f loader.Fact) app.NCRecordRef {
	actor := f.Actor
	seq := f.Seq
	return app.NCRecordRef{EventID: fmt.Sprintf("EV-SESSION-%d", f.Seq), EventType: eventType(f.Op), Kind: "decision", Seq: &seq, OccurredAt: f.At,
		Author: &actor, Summary: summaryOf(f)}
}

// eventType — тип записи решения по операции (каталог событий).
func eventType(op string) string {
	switch op {
	case opResolve:
		return "decision.presentation.resolved"
	case opReview:
		return "decision.presentation.reviewed"
	case opReject:
		return "decision.signal.rejected"
	case opRecheck:
		return "decision.recheck.requested"
	case opIsolate:
		return "decision.item.isolated"
	case opConfirm:
		return "decision.nonconformity.confirmed"
	case opDisposition:
		return "decision.disposition.set"
	case opVerify:
		return "decision.disposition.verified"
	case opClose:
		return "decision.nonconformity.closed"
	case opContainment:
		return "decision.containment.set"
	case opRelease:
		return "decision.containment.released"
	}
	return "decision.recorded"
}

var resolutionText = map[string]string{"accept": "принять", "accept_with_concession": "принять по разрешению на отклонение", "reject": "не принять", "insufficient_data": "мало данных"}
var dispositionText = map[string]string{"rework": "доработать", "repair": "ремонт", "use_as_is": "использовать как есть", "scrap": "в брак", "return_to_supplier": "вернуть поставщику"}

// summaryOf — решение словами для людей.
func summaryOf(f loader.Fact) string {
	switch b := f.Body.(type) {
	case app.ResolvePresentation:
		return "Решение на точке предъявления: " + resolutionText[b.Resolution]
	case app.ReviewPresentation:
		if b.Outcome == "revoked" {
			return "Пересмотр: приёмка отозвана — " + b.Reason.Text
		}
		return "Пересмотр: решение оставлено в силе — " + b.Reason.Text
	case app.RejectSignal:
		return "Сигнал отклонён: " + b.Reason.Text
	case app.RequestRecheck:
		return "Назначена дополнительная проверка: " + b.Reason.Text
	case app.IsolateItem:
		return "Изделие изолировано: " + b.Reason.Text
	case app.ConfirmNonconformity:
		return "Несоответствие подтверждено: " + b.Reason.Text
	case app.SetDisposition:
		return "Решение по изделию: " + dispositionText[b.Disposition] + " — " + b.Reason.Text
	case app.VerifyDisposition:
		return "Исполнение решения подтверждено"
	case app.CloseNonconformity:
		return "Несоответствие закрыто"
	}
	return "Решение записано"
}

// card — карточка несоответствия с решениями сессии: статус, оси изделия,
// решения людей.
func (d decisions) card(c app.NCCard) app.NCCard {
	fs := append(slices.Clone(d.of(string(platform.EntityNonconformity), c.NCID)), d.of(string(platform.EntityItem), c.ItemID)...)
	if len(fs) == 0 {
		return c
	}
	slices.SortStableFunc(fs, func(a, b loader.Fact) int { return int(a.Seq - b.Seq) })
	c.HumanDecisions = slices.Clone(c.HumanDecisions)
	for _, f := range fs {
		switch b := f.Body.(type) {
		case app.ConfirmNonconformity:
			c.Status = "confirmed"
		case app.RejectSignal:
			c.Status, c.Resolution = "closed", ptr("signal_rejected")
		case app.IsolateItem:
			c.Axes.Containment, c.Axes.Position = "item_hold", "isolated"
		case app.SetDisposition:
			c.Status, c.Axes.Disposition = "disposition_set", b.Disposition
		case app.VerifyDisposition:
			c.Status = "verified"
		case app.CloseNonconformity:
			c.Status = "closed"
		case app.ReleaseContainment:
			c.Axes.Containment = "none"
		case app.SetContainment:
			if b.Level != "" {
				c.Axes.Containment = b.Level
			}
		case app.ResolvePresentation:
			c.Presentation = nil
		default:
			continue
		}
		c.HumanDecisions = append(c.HumanDecisions, decisionRef(f))
	}
	return c
}

// presentation — точка предъявления с решением сессии: кнопки решений
// недоступны, почему — «решение записано».
func (d decisions) presentation(v app.NCPresentationView) app.NCPresentationView {
	var last *loader.Fact
	for _, f := range d.of(string(platform.EntityItem), v.ItemID) {
		if f.Op == opResolve || f.Op == opReview {
			last = &f
		}
	}
	if last == nil {
		return v
	}
	why := summaryOf(*last) + fmt.Sprintf(" (запись № %d)", last.Seq)
	v.Presentation.AllowedResolutions = []string{}
	v.Actions = slices.Clone(v.Actions)
	for i := range v.Actions {
		v.Actions[i].Allowed, v.Actions[i].WhyAvailable = false, why
	}
	if last.Op == opReview {
		v.Review = nil
	}
	return v
}

func ptr[T any](v T) *T { return &v }
