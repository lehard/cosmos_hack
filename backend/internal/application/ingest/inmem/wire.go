package inmem

import (
	app "ant/internal/application/ingest"
)

// Core — приём целиком в памяти: сервис и его хранилища.
type Core struct {
	Service    *app.Service
	Journal    *Journal
	Registry   *Registry
	Quarantine *Quarantine
	Materials  *Materials
	Keys       *Keys
	Telemetry  *Telemetry
	Clock      *Clock
}

// NewCore собирает приём в памяти с часами clock (nil — системные).
func NewCore(cfg app.Config, clock *Clock, extra func(*app.Deps)) *Core {
	c := &Core{Journal: nil, Registry: NewRegistry(), Quarantine: NewQuarantine(), Materials: NewMaterials(),
		Keys: NewKeys(), Telemetry: NewTelemetry(), Clock: clock}
	d := app.Deps{Registry: c.Registry, Quarantine: c.Quarantine, Materials: c.Materials, Telemetry: c.Telemetry}
	if clock != nil {
		c.Journal = NewJournal(clock.At)
		d.DomainClock, d.InfraClock = clock, clock.Infra()
	} else {
		c.Journal = NewJournal(nil)
		d.DomainClock, d.InfraClock = SystemClock{}, SystemInfra{}
	}
	d.Journal = c.Journal
	if extra != nil {
		extra(&d)
	}
	c.Service = app.NewService(app.WithConfig(cfg), app.WithDeps(d))
	return c
}
