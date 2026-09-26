package reference

import (
	"context"

	"ant/internal/application/platform"
	app "ant/internal/application/reference"
	dom "ant/internal/domain/reference"
	"ant/internal/infrastructure/fixtures/loader"
)

// Adapter — реализация fixtures ведущих портов модуля reference (AD-36):
// чтение — те же сценарии приложения над книгой стартовых справочников
// (затравка normative/reference, её собирает cmd/ant), без журнала; команды
// в режиме заготовок не исполняются (501). Без книги — все операции 501.
type Adapter struct {
	*app.Service
}

// New создаёт адаптер заготовок без книги (все операции 501).
func New() *Adapter { return &Adapter{Service: app.NewService()} }

// NewWithBook — адаптер заготовок над книгой справочников.
func NewWithBook(b dom.Book) *Adapter {
	return &Adapter{Service: app.NewService(app.WithSource(app.StaticSource{B: b}))}
}

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Shifts — график смен (reference.shift.list, FR-81): в режиме заготовок —
// смены мира на часах шага курсора (у книги справочников нет конкретных смен,
// а часы live — не часы мира); место — фильтр по location_id.
func (a *Adapter) Shifts(ctx context.Context, locationID string, m platform.Moment) (app.RefShiftList, error) {
	rt, err := loader.Default()
	if err != nil {
		return a.Service.Shifts(ctx, locationID, m)
	}
	var v app.RefShiftList
	if err := rt.Respond(ctx, "reference.shift.list", nil, &m, &v); err != nil {
		return a.Service.Shifts(ctx, locationID, m)
	}
	if locationID == "" {
		return v, nil
	}
	out := app.RefShiftList{Items: []app.RefShift{}}
	for _, s := range v.Items {
		if s.LocationID == locationID {
			out.Items = append(out.Items, s)
		}
	}
	return out, nil
}
