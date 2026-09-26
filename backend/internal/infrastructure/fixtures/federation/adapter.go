package federation

import (
	"context"
	"slices"
	"sync"
	"time"

	app "ant/internal/application/federation"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/federation"
	"ant/internal/infrastructure/fixtures/loader"
	"ant/internal/infrastructure/security/profiles"
)

// Adapter — реализация fixtures ведущих портов модуля federation (AD-36,
// эпик 41): партнёры и выписки — ответы мира заготовок (scenarios/federation,
// подписи проверены при генерации). Команды мир не меняют (FR-129), но
// стенд в режиме заготовок показывает их до перезапуска процесса: принятая
// выписка, зарегистрированный партнёр и отправленная выписка — наложение в
// памяти поверх ответа мира. Приём выписки проверяет подписи по-настоящему
// (тот же domain/federation.VerifyExtract и ГОСТ Р 34.10-2012, что у live):
// изменённая выписка отклоняется 422 federation.extract_tampered.
type Adapter struct {
	mu       sync.Mutex
	partners []app.Partner
	extracts []app.PassportExtractView
}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

func respond[T any](ctx context.Context, op string, params map[string]string, m *platform.Moment) (T, error) {
	var out T
	rt, err := loader.Default()
	if err != nil {
		return out, err
	}
	err = rt.Respond(ctx, op, params, m, &out)
	return out, err
}

func decide(ctx context.Context, op, kind, id string, meta platform.CommandMeta) (platform.Receipt, error) {
	rt, err := loader.Default()
	if err != nil {
		return platform.Receipt{}, err
	}
	return rt.Decide(ctx, op, loader.ObjectRef{Kind: kind, ID: id}, meta)
}

// Partners — партнёры мира и зарегистрированные в этом процессе (federation.partner.list).
func (a *Adapter) Partners(ctx context.Context, m platform.Moment) (app.PartnerList, error) {
	l, err := respond[app.PartnerList](ctx, "federation.partner.list", nil, &m)
	if err != nil {
		return l, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, p := range a.partners {
		i := slices.IndexFunc(l.Items, func(x app.Partner) bool { return x.PartnerCode == p.PartnerCode })
		if i >= 0 {
			l.Items[i] = p
		} else {
			l.Items = append(l.Items, p)
		}
	}
	return l, nil
}

// Extracts — выписки (federation.extract.list): мир и принятые/отправленные здесь.
func (a *Adapter) Extracts(ctx context.Context, direction, partnerCode string, m platform.Moment, _ platform.Page) (app.PassportExtractList, error) {
	l, err := respond[app.PassportExtractList](ctx, "federation.extract.list", nil, &m)
	if err != nil {
		return l, err
	}
	a.mu.Lock()
	var mine []app.PassportExtract
	for _, v := range a.extracts {
		if !slices.ContainsFunc(l.Items, func(x app.PassportExtract) bool {
			return x.ExtractDigest == v.Extract.ExtractDigest && x.Direction == v.Extract.Direction
		}) {
			mine = append(mine, v.Extract)
		}
	}
	a.mu.Unlock()
	slices.Reverse(mine)
	return app.PassportExtractList{Items: app.Filter(append(mine, l.Items...), direction, partnerCode)}, nil
}

// Extract — выписка с подписями (federation.extract.read).
func (a *Adapter) Extract(ctx context.Context, digest string) (app.PassportExtractView, error) {
	a.mu.Lock()
	for _, v := range a.extracts {
		if v.Extract.ExtractDigest == digest {
			a.mu.Unlock()
			return v, nil
		}
	}
	a.mu.Unlock()
	return respond[app.PassportExtractView](ctx, "federation.extract.read", map[string]string{"extract_digest": digest}, nil)
}

// RegisterPartner — акт регистрации партнёра (federation.partner.register):
// тот же гард, что у live; партнёр виден до перезапуска процесса.
func (a *Adapter) RegisterPartner(ctx context.Context, in app.RegisterPartner) (platform.Receipt, error) {
	if err := app.CheckRegister(in); err != nil {
		return platform.Receipt{}, err
	}
	rc, err := decide(ctx, "federation.partner.register", "partner", in.PartnerCode, in.CommandMeta())
	if err != nil {
		return rc, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	p := app.Partner{PartnerCode: in.PartnerCode, Name: in.Name, RootFingerprints: in.RootFingerprints, Endpoint: in.Endpoint,
		DocumentID: in.DocumentID, RegisteredAt: at(rc), Channel: "unknown"}
	a.partners = slices.DeleteFunc(a.partners, func(x app.Partner) bool { return x.PartnerCode == in.PartnerCode })
	a.partners = append(a.partners, p)
	return rc, nil
}

// ReceiveExtract — принять выписку партнёра (federation.extract.receive,
// FR-132): настоящая проверка подписей по корням партнёра из списка
// партнёров; изменённая — 422; принятая видна в списке до перезапуска.
func (a *Adapter) ReceiveExtract(ctx context.Context, in app.ReceiveExtract) (platform.Receipt, error) {
	ps, err := a.Partners(ctx, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	roots := app.RootsOf(ps.Items, in.PartnerCode)
	partner := in.PartnerCode
	if roots == nil {
		partner = ""
	}
	raw := []byte(in.Envelope)
	v, err := dom.VerifyExtract(raw, partner, roots, profiles.Verify)
	if err != nil {
		return platform.Receipt{}, app.RejectTampered(err)
	}
	rc, err := decide(ctx, "federation.extract.receive", "lot", v.Digest, in.CommandMeta())
	if err != nil {
		return rc, err
	}
	view := app.ViewOf(v, raw, "incoming", in.PartnerCode, v.Digest, at(rc), nil)
	a.remember(view)
	return rc, nil
}

// SendExtract — отправить выписку партнёру (federation.extract.send): на
// заготовках пакет не собирается и не подписывается (ключа шлюза у стенда
// нет) — в списке появляется исходящая запись «ждёт квитанции».
func (a *Adapter) SendExtract(ctx context.Context, in app.SendExtract) (platform.Receipt, error) {
	ps, err := a.Partners(ctx, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	if app.RootsOf(ps.Items, in.PartnerCode) == nil {
		return platform.Receipt{}, platform.Fail(errcodes.ApiNotFound, "object", "Партнёр", "id", in.PartnerCode)
	}
	rc, err := decide(ctx, "federation.extract.send", "partner", in.PartnerCode, in.CommandMeta())
	if err != nil {
		return rc, err
	}
	sub := in.Subject
	e := app.PassportExtract{ExtractDigest: "pending:" + rc.CommandID, Direction: "outgoing", PartnerCode: in.PartnerCode,
		OriginStatus: dom.OriginNotApplicable, Subject: &sub, DocumentID: in.DocumentID, MessageID: rc.CommandID, At: at(rc), Label: sub.ID}
	if len(rc.EventIDs) > 0 {
		e.MessageID = rc.EventIDs[0]
	}
	a.remember(app.PassportExtractView{Extract: e, Content: map[string]any{}, Signatures: []app.PartnerSignature{},
		OriginReason: "отправлено через порт межзаводского обмена; квитанция партнёра придёт событием"})
	return rc, nil
}

func (a *Adapter) remember(v app.PassportExtractView) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.extracts = slices.DeleteFunc(a.extracts, func(x app.PassportExtractView) bool {
		return x.Extract.ExtractDigest == v.Extract.ExtractDigest && x.Extract.Direction == v.Extract.Direction
	})
	a.extracts = append(a.extracts, v)
}

func at(rc platform.Receipt) time.Time {
	if rc.RecordedAt.IsZero() {
		return time.Now().UTC()
	}
	return rc.RecordedAt.UTC()
}
