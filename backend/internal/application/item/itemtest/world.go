// Пакет itemtest — опора тестов модулей item и crossitem на фейках (журнал в
// памяти enginemem) и на своей БД (make dev-db): факты — как их записал бы
// приём (эпик 06), команды — живыми операциями item и crossitem, шаг
// межизделийной стадии — настоящим StageRunner (crossitem.Fold со всеми
// подключёнными модулями, AD-42), проекции изделий — тем же кодом, что
// `ant rebuild` (AD-45).
//
// Слой: application (тестовая опора; без драйверов и сети). В сборку ролей не входит.
package itemtest

import (
	"context"
	"strconv"
	"time"

	crossitemapp "ant/internal/application/crossitem"
	engineapp "ant/internal/application/engine"
	itemapp "ant/internal/application/item"
	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	"ant/internal/domain/crossitem"
	dom "ant/internal/domain/item"
	"ant/internal/domain/kernel"
)

// T0 — начало сценария.
var T0 = time.Date(2026, 9, 26, 5, 0, 0, 0, time.UTC)

// Env — номенклатура сценария: фланец с зонами и соединением J-1.
var Env = dom.Env{Types: map[string]dom.TypeDef{"FL-100.00.000": {
	Marking: "dpm_datamatrix",
	Zones:   []dom.ZoneDef{{ID: "W-1.U1", Name: "Шов W-1, участок У1"}, {ID: "S-1", Name: "Канавка уплотнения"}, {ID: "CAV", Name: "Полость"}},
	Links:   []dom.LinkDef{{ID: "J-1", Kind: "bolted_joint", Zones: []string{"J-1"}, ClosesAccessTo: []string{"S-1", "CAV"}}},
}}}

// Clock — доменные часы команд (AD-37), двигает сценарий.
type Clock struct{ T time.Time }

// Now — доменное «сейчас».
func (c *Clock) Now(context.Context) (time.Time, error) { return c.T, nil }

// World — журнал, кодек, проекции, стадия и живые операции.
type World struct {
	Journal  appjournal.JournalStore
	Codec    *engineapp.Codec
	Store    engineapp.ProjectionStore
	Registry *engineapp.Registry
	Clock    *Clock
	Item     *itemapp.Service
	Cross    *crossitemapp.Service
	stage    crossitem.Stage
	after    int64
	n        int
	tag      string
}

// New — мир над журналом и хранением проекций; tag различает прогоны в одной БД.
func New(j appjournal.JournalStore, codec *engineapp.Codec, store engineapp.ProjectionStore, tag string) *World {
	w := &World{Journal: j, Codec: codec, Store: store, Registry: itemapp.MustRegister(engineapp.NewRegistry()), Clock: &Clock{T: T0}, tag: tag}
	wr := itemapp.JournalWriter{Journal: j, DomainBuild: codec.DomainBuild, Partitions: codec.Partitions}
	w.Item = itemapp.NewLive(itemapp.Config{Codec: codec, Projections: store, Bundles: itemapp.Bundles{Env: Env}, Writer: wr, Clock: w.Clock, Env: Env})
	w.Cross = crossitemapp.NewLive(crossitemapp.Config{Projections: store, Writer: wr, Clock: w.Clock})
	return w
}

// ID — детерминированный UUID записи мира.
func (w *World) ID(kind string) string {
	w.n++
	return kernel.UUIDv5(constants.NsAnt, "itemtest\x1f"+w.tag+"\x1f"+kind+"\x1f"+strconv.Itoa(w.n))
}

// Cmd — заголовок команды с новым command_id.
func (w *World) Cmd() platform.CommandHeader { return platform.CommandHeader{CommandID: w.ID("cmd")} }

// At — момент сценария (минуты от T0); часы команд сдвигаются туда же.
func (w *World) At(min int) time.Time {
	t := T0.Add(time.Duration(min) * time.Minute)
	w.Clock.T = t
	return t
}

// Fact пишет запись как приём (эпик 06): поток изделия или объекта; carrier —
// носитель события без изделия (`тип:значение`, AD-41). Возвращает event_id.
func (w *World) Fact(ctx context.Context, t catalog.Type, item, stream, carrier string, at time.Time, data any) (string, error) {
	info, _ := catalog.Lookup(t)
	id := w.ID(string(t))
	if stream == "" && item != "" {
		stream = "item:" + item
	}
	p, err := w.Codec.Encode(ctx, engineapp.Out{EventID: id, Type: t, Kind: info.Kind, Stream: stream, ItemID: item,
		OccurredAt: at, Correlation: id, Data: data})
	if err != nil {
		return "", err
	}
	if carrier != "" {
		p.Entry.CarrierRef = &carrier
	}
	_, err = w.Journal.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}})
	return id, err
}

// Settle — межизделийная стадия до неподвижной точки (StageRunner над
// crossitem.Fold, AD-42; состояние стадии и реестр носителей — эффектами в
// той же пачке) и пересборка проекций item.* из журнала (Rebuilder, AD-45).
func (w *World) Settle(ctx context.Context) error {
	runner := &crossitemapp.StageRunner{Codec: w.Codec}
	for range 20 {
		es, err := w.readAfter(ctx, w.after)
		if err != nil {
			return err
		}
		if len(es) == 0 {
			break
		}
		next, rq, err := runner.Apply(ctx, w.stage, es)
		if err != nil {
			return err
		}
		w.after = int64(es[len(es)-1].Seq)
		w.stage = next
		if len(rq.Batch) == 0 && len(rq.Effects) == 0 {
			continue
		}
		if _, err := w.Journal.Append(ctx, rq); err != nil {
			return err
		}
	}
	_, err := (&engineapp.Rebuilder{Codec: w.Codec, Registry: w.Registry, Bundles: itemapp.Bundles{Env: Env}}).RebuildAll(ctx)
	return err
}

// Stage — состояние стадии мира.
func (w *World) Stage() crossitem.Stage { return w.stage }

func (w *World) readAfter(ctx context.Context, after int64) ([]jcEntry, error) {
	var out []jcEntry
	for {
		es, err := w.Journal.Read(ctx, appjournal.ReadQuery{AfterSeq: after, Limit: 1000})
		if err != nil {
			return nil, err
		}
		out = append(out, es...)
		if len(es) < 1000 {
			return out, nil
		}
		after = int64(es[len(es)-1].Seq)
	}
}

// Principal — контекст с автором команды.
func Principal(ctx context.Context, person string) context.Context {
	return platform.WithPrincipal(ctx, platform.Principal{PersonID: person, Role: "quality_inspector"})
}
