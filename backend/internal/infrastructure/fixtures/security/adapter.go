package security

import (
	"context"

	app "ant/internal/application/security"
)

// Adapter — реализация fixtures ведущих портов модуля security (AD-36):
// индикатор целостности «по данным сервера» — из мира заготовок на шаге
// курсора (в главной истории нарушение появляется после подмены записи, S09).
type Adapter struct {
	// Unimplemented — операции, которых нет в мире заготовок, отвечают 501.
	app.Unimplemented
}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Integrity — состояние целостности журнала (security.integrity.read, AD-46).
func (Adapter) Integrity(ctx context.Context) (app.IntegrityStatus, error) {
	return respond[app.IntegrityStatus](ctx, "security.integrity.read", nil, nil)
}
