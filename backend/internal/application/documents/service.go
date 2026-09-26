package documents

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	jc "ant/internal/contracts/journal"
	dom "ant/internal/domain/documents"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
)

// Service — реализация live ведущих портов модуля documents (AD-36).
// Документ изделия — часть свёртки изделия (AD-5, AD-22): чтение сворачивает
// вход изделия на момент той же функцией, что воркер; команды проверяют
// доменный гард над этой свёрткой и пишут одну запись в поток документа с
// item_id изделия, так что версию и закрытие маршрута вычисляет воркер
// (реакции document.version.drafted и document.route.closed). Документ вне
// изделия (лист утверждения версии процесса, «Запросить решение» по объекту
// вне изделия) — поток `document:‹id›`: api сворачивает его
// documents.FoldStream и пишет запрос, версию и закрытие маршрута в той же
// пачке, что и команду (AD-12, AD-42, AD-43).
// Без зависимостей (NewService()) — заглушка 501, как в волне 1.
type Service struct {
	Unimplemented
	d   Deps
	cfg Config
}

// Deps — зависимости live-реализации.
type Deps struct {
	// Journal — журнал (AD-44): чтение и единственная функция записи.
	Journal appjournal.JournalStore
	// Codec — чтение записей журнала в представление домена и сборка записей
	// ядра (реакции документов вне изделия).
	Codec *engineapp.Codec
	// Bundles — нормативный слой изделия с частью documents (Bundles этого
	// пакета поверх источника версии).
	Bundles engineapp.BundleSource
	// Fold — свёртка изделия; nil — domain/engine.Fold.
	Fold engine.Folder
	// Env — шаблоны и срез политики для документов вне изделия.
	Env dom.Env
	// DomainClock — доменное «сейчас» при приёме команды (AD-37).
	DomainClock appjournal.DomainClock
	// Now — InfraClock для received_at (AD-37); nil — time.Now.
	Now func() time.Time
	// Stamps — реестр действующих цифровых клейм (FR-145, эпик 37: проекция
	// политики access); nil — клеймо проверяется по срезу стартовой политики.
	Stamps StampRegistry
}

// StampRegistry — ведомый порт реестра цифровых клейм (FR-145): действующее
// в момент at клеймо сотрудника по виду контроля (выдано по приказу, в
// сроке, не отозвано). Реализация — access.StampRegistry над проекцией политики.
type StampRegistry interface {
	ValidStamp(ctx context.Context, personID, kind string, at time.Time) (stampID string, ok bool, err error)
}

// Config — параметры модуля.
type Config struct {
	// DomainBuild — domain_build записей (AD-9).
	DomainBuild string
	// Partitions — число партиций P (AD-6).
	Partitions int
	// ScenarioClock — режим часов scenario: recorded_at = доменное «сейчас» (AD-37).
	ScenarioClock bool
	// APIPrefix — префикс путей API для print_url (по умолчанию /api/v1).
	APIPrefix string
}

// Option — настройка Service.
type Option func(*Service)

// WithDeps — зависимости live-реализации.
func WithDeps(d Deps) Option { return func(s *Service) { s.d = d } }

// WithConfig — параметры модуля.
func WithConfig(c Config) Option { return func(s *Service) { s.cfg = c } }

// NewService создаёт реализацию live; без WithDeps — заглушка 501.
func NewService(opts ...Option) *Service {
	s := &Service{}
	for _, o := range opts {
		o(s)
	}
	if s.cfg.Partitions <= 0 {
		s.cfg.Partitions = 1
	}
	if s.cfg.APIPrefix == "" {
		s.cfg.APIPrefix = "/api/v1"
	}
	if s.d.Fold == nil {
		s.d.Fold = engine.Fold
	}
	if s.d.Bundles == nil {
		s.d.Bundles = Bundles{Env: s.d.Env}
	}
	if s.d.Now == nil {
		s.d.Now = time.Now
	}
	return s
}

func (s *Service) live() bool { return s.d.Journal != nil && s.d.Codec != nil }

var (
	_ Queries  = (*Service)(nil)
	_ Commands = (*Service)(nil)
)

// now — доменное «сейчас» (AD-37) в прогоне runID (AD-38).
func (s *Service) now(ctx context.Context, runID string) (time.Time, error) {
	if s.d.DomainClock == nil {
		return s.d.Now().UTC(), nil
	}
	if runID != "" {
		ctx = appjournal.WithRun(ctx, runID)
	}
	t, err := s.d.DomainClock.Now(ctx)
	return t.UTC().Truncate(time.Millisecond), err
}

// ── чтение журнала ──

// readAll — все записи по запросу постранично.
func (s *Service) readAll(ctx context.Context, q appjournal.ReadQuery) ([]jc.JournalEntry, error) {
	var out []jc.JournalEntry
	q.Limit = 1000
	for {
		page, err := s.d.Journal.Read(ctx, q)
		if err != nil {
			return nil, err
		}
		out = append(out, page...)
		if len(page) < q.Limit {
			return out, nil
		}
		q.AfterSeq = int64(page[len(page)-1].Seq)
	}
}

// within — запись видна на момент m (AD-22).
func within(m platform.Moment, occurred, recordedAt time.Time) bool {
	if m.AsOf == nil {
		return true
	}
	t := occurred
	if m.Axis == platform.AxisRecorded {
		t = recordedAt
	}
	return !t.After(*m.AsOf)
}

// recorded — записанная реакция: event_id, тип, слот.
type recorded struct {
	EventID string
	Type    catalog.Type
	Slot    kernel.Slot
	Seq     int64
}

// view — документ в состоянии на момент: состояние модуля (изделия или
// потока документа), нормативная часть, основание команд.
type view struct {
	ItemID string
	RunID  string
	State  dom.State
	Env    dom.Env
	// Input — вход свёртки (для доводов и типов записей).
	Input []kernel.Record
	// Reactions — записанные реакции (document.route.closed, drafted).
	Reactions []recorded
	// BasisSeq — seq последней записи (basis_seq ответов чтения).
	BasisSeq int64
}

func (v *view) record(id string) (kernel.Record, bool) {
	i := slices.IndexFunc(v.Input, func(r kernel.Record) bool { return r.EventID == id })
	if i < 0 {
		return kernel.Record{}, false
	}
	return v.Input[i], true
}

// decodeAll — записи журнала в представление домена на момент m: вход
// (всё, кроме записей роли worker) и записанные реакции.
func (s *Service) decodeAll(ctx context.Context, entries []jc.JournalEntry, m platform.Moment, v *view) error {
	for _, e := range entries {
		if e.Chain != "" && e.Chain != jc.JournalEntryChainMain {
			continue
		}
		d, err := s.d.Codec.Decode(ctx, e)
		if err != nil {
			return err
		}
		rec, _ := time.Parse(time.RFC3339Nano, e.RecordedAt)
		if !within(m, d.Record.OccurredAt, rec) {
			continue
		}
		if int64(e.Seq) > v.BasisSeq {
			v.BasisSeq = int64(e.Seq)
		}
		if v.RunID == "" && d.Record.RunID != "" {
			v.RunID = d.Record.RunID
		}
		switch {
		case d.Info.Type == catalog.OpsProcessingFailed || d.Info.Type == catalog.OpsProcessingRetried:
			continue
		case d.Info.Role == engineapp.RoleWorker || d.Record.Kind == catalog.KindReaction:
			x := recorded{EventID: d.Record.EventID, Type: d.Info.Type, Seq: int64(e.Seq)}
			if e.ReactionSlot != nil {
				x.Slot = kernel.Slot{RuleID: e.ReactionSlot.RuleID, Subject: e.ReactionSlot.Subject, TriggerKey: e.ReactionSlot.TriggerKey}
			}
			v.Reactions = append(v.Reactions, x)
			continue
		}
		v.Input = append(v.Input, d.Record)
	}
	return nil
}

// loadItem — документы изделия на момент m: та же свёртка, что у воркера
// (AD-22), ничего не пишет.
func (s *Service) loadItem(ctx context.Context, itemID string, m platform.Moment) (*view, error) {
	entries, err := s.readAll(ctx, appjournal.ReadQuery{ItemID: itemID})
	if err != nil {
		return nil, err
	}
	v := &view{ItemID: itemID}
	if err := s.decodeAll(ctx, entries, m, v); err != nil {
		return nil, err
	}
	if len(v.Input) == 0 {
		return nil, notFound("Изделие", itemID)
	}
	b, _, err := s.d.Bundles.Bundle(ctx, itemID, v.Input)
	if err != nil {
		return nil, err
	}
	v.Env = b.Documents
	if !v.Env.Active() {
		return nil, platform.NotImplemented("documents: нормативный слой документов не подключён")
	}
	if err := func() (err error) {
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("свёртка изделия %s: %v", itemID, p)
			}
		}()
		snap, _ := s.d.Fold(b, v.Input)
		v.State = snap.Documents
		return nil
	}(); err != nil {
		return nil, err
	}
	return v, nil
}

// loadStream — документ вне изделия: поток `document:‹id›` на момент m.
func (s *Service) loadStream(ctx context.Context, documentID string, m platform.Moment) (*view, error) {
	entries, err := s.readAll(ctx, appjournal.ReadQuery{Stream: streamOf(documentID)})
	if err != nil {
		return nil, err
	}
	v := &view{Env: s.d.Env}
	if err := s.decodeAll(ctx, entries, m, v); err != nil {
		return nil, err
	}
	v.State = dom.FoldStream(v.Input, v.Env)
	return v, nil
}

func streamOf(documentID string) string { return "document:" + documentID }

func notFound(object, id string) error {
	e := platform.Fail(errcodes.ApiNotFound, "object", object, "id", id)
	e.Detail = object + " " + id + " не найден"
	return e
}

// locate — где живёт документ: у изделия (item_id записей потока документа
// или id сопроводительной карты) или в собственном потоке.
func (s *Service) locate(ctx context.Context, documentID string) (itemID string, err error) {
	if it, ok := strings.CutPrefix(documentID, dom.PrefixTraveler); ok {
		return it, nil
	}
	page, err := s.d.Journal.Read(ctx, appjournal.ReadQuery{Stream: streamOf(documentID), Limit: 1})
	if err != nil {
		return "", err
	}
	if len(page) == 0 {
		return "", notFound("Документ", documentID)
	}
	if page[0].ItemID != nil {
		return *page[0].ItemID, nil
	}
	return "", nil
}

// load — документ и его состояние на момент m.
func (s *Service) load(ctx context.Context, documentID string, m platform.Moment) (*view, *dom.Doc, error) {
	if !s.live() {
		return nil, nil, platform.NotImplemented("documents")
	}
	itemID, err := s.locate(ctx, documentID)
	if err != nil {
		return nil, nil, err
	}
	var v *view
	if itemID != "" {
		v, err = s.loadItem(ctx, itemID, m)
	} else {
		v, err = s.loadStream(ctx, documentID, m)
	}
	if err != nil {
		return nil, nil, err
	}
	if d := v.State.Doc(documentID); d != nil {
		return v, d, nil
	}
	if documentID == dom.TravelerID(itemID) {
		if d, ok := v.State.Traveler(v.Env); ok {
			return v, &d, nil
		}
	}
	return nil, nil, notFound("Документ", documentID)
}

// template — шаблон документа.
func (v *view) template(d *dom.Doc) (dom.Template, error) {
	t, ok := v.Env.Templates.ByRef(d.TemplateRef)
	if !ok {
		return dom.Template{}, platform.Fail(errcodes.ApiNotFound, "object", "Шаблон", "id", d.TemplateRef)
	}
	return t, nil
}

// version — версия документа для показа: записанная или (у карты без
// версий, version = 0) текущий сбор из истории (live).
func (v *view) version(d *dom.Doc, no int) (*dom.Version, dom.Built, bool, error) {
	t, err := v.template(d)
	if err != nil {
		return nil, dom.Built{}, false, err
	}
	if no <= 0 && d.Current() == nil {
		pv, b, _, err := v.State.Preview(v.Env, d)
		if err != nil {
			return nil, dom.Built{}, false, err
		}
		return &pv, b, true, nil
	}
	ver := d.Version(no)
	if ver == nil {
		return nil, dom.Built{}, false, notFound("Версия документа", fmt.Sprintf("%s@%d", d.ID, no))
	}
	b, err := dom.Rebuild(t, ver)
	return ver, b, false, err
}
