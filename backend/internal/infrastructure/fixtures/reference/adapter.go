package reference

import (
	app "ant/internal/application/reference"
	dom "ant/internal/domain/reference"
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
