package nctest

import (
	"context"
	"errors"
	"testing"
	"time"

	engineapp "ant/internal/application/engine"
	"ant/internal/application/engine/enginemem"
	appjournal "ant/internal/application/journal"
	app "ant/internal/application/nonconformity"
	"ant/internal/application/platform"
)

// World — журнал в памяти, воркер движка со свёрткой DraftingFold и сервис
// модуля nonconformity над ними (AD-36: те же порты, что в живом ядре).
type World struct {
	T      *testing.T
	J      *enginemem.Journal
	Codec  *engineapp.Codec
	Worker *engineapp.WorkerService
	Feed   *enginemem.Feed
	Part   engineapp.Partition
	Now    time.Time
}

// NewWorld — мир в памяти с одной партицией.
func NewWorld(t *testing.T) *World {
	j := enginemem.New(nil)
	part := engineapp.Partition{Number: 0, Epoch: 1}
	j.SetEpoch(appjournal.PartitionLease(0), 1)
	codec := &engineapp.Codec{Store: j, Sealer: enginemem.Sealer{KeyRef: "engine@1"}, KeyRef: "engine@1", DomainBuild: "test", Partitions: 1}
	feed := &enginemem.Feed{J: j, Parts: []engineapp.Partition{part}}
	reg := engineapp.NewRegistry()
	if err := app.RegisterProjections(reg); err != nil {
		t.Fatal(err)
	}
	w := engineapp.NewWorker(engineapp.WorkerConfig{Feed: feed, Codec: codec, Fold: DraftingFold, Projections: reg})
	return &World{T: t, J: j, Codec: codec, Worker: w, Feed: feed, Part: part, Now: T0.Add(8 * time.Hour)}
}

// Service — сервис модуля над миром; routes — порт «маршрут закрыт».
func (w *World) Service(routes app.RouteGate) *app.Service {
	return app.NewService(
		app.WithDeps(app.Deps{Journal: w.J, Codec: w.Codec, Fold: DraftingFold, Routes: routes, DomainClock: clockAt{&w.Now},
			Now: func() time.Time { return w.Now }}),
		app.WithConfig(app.Config{DomainBuild: "streebog256:" + zeros, Partitions: 1}),
	)
}

const zeros = "0000000000000000000000000000000000000000000000000000000000000000"

type clockAt struct{ t *time.Time }

func (c clockAt) Now(context.Context) (time.Time, error) { return *c.t, nil }

// Add — записи в журнал одной пачкой.
func (w *World) Add(ps ...appjournal.Pending) {
	w.T.Helper()
	if _, err := w.J.Append(context.Background(), appjournal.AppendRequest{Batch: ps}); err != nil {
		w.T.Fatal(err)
	}
}

// Settle — воркер обрабатывает весь необработанный вход (AD-5).
func (w *World) Settle() {
	w.T.Helper()
	for range 20 {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		works, err := w.Feed.Next(ctx, w.Part)
		cancel()
		if errors.Is(err, context.DeadlineExceeded) {
			return
		}
		if err != nil {
			w.T.Fatal(err)
		}
		if err := w.Worker.Process(context.Background(), w.Part, works); err != nil {
			w.T.Fatal(err)
		}
	}
	w.T.Fatal("воркер не успокоился")
}

// As — контекст с субъектом person в роли role.
func As(person, role string) context.Context {
	return platform.WithPrincipal(context.Background(), platform.Principal{PersonID: person, Role: role})
}
