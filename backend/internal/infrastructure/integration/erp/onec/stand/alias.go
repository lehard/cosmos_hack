package stand

import (
	"context"
	"net/http"

	app "ant/internal/application/ingest"
	"ant/internal/infrastructure/integration/ingest/stands"
)

// AliasName — имя stand-а 1С в определениях сценариев (`stand: onec`,
// scenarios/definitions): сбои по нему включают те же сбои stand-а «1c».
const AliasName = "onec"

// Alias — второе имя stand-а для служебного порта сбоев: общие сбои, без
// своего протокола (страница и протокол — только /stand/1c/).
func Alias(s *Stand) stands.Stand { return alias{s} }

type alias struct{ s *Stand }

func (a alias) Info() app.StandInfo {
	i := a.s.Info()
	i.Name, i.Emulates = AliasName, "то же, что stand «1c» (имя stand-а 1С в сценариях): "+i.Emulates
	return i
}

func (a alias) Faults() *stands.FaultSwitch { return a.s.Faults() }

func (a alias) Handler() http.Handler { return nil }

func (a alias) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
