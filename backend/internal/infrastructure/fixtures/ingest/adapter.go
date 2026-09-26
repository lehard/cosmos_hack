package ingest

import app "ant/internal/application/ingest"

// Adapter — реализация fixtures ведущих портов модуля ingest. В волне 1 —
// заглушка: все операции отвечают 501 (app.Unimplemented).
type Adapter struct {
	app.Unimplemented
}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)
