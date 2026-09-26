package mes

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля mes (AD-36).
type Queries interface {
	// Jobs — задания MES (FR-92) (mes.order.list).
	Jobs(ctx context.Context, m platform.Moment, p platform.Page) (MesJobList, error)
	// Blocks — блокировки в MES (FR-93) (mes.block.list).
	Blocks(ctx context.Context, m platform.Moment) (MesBlockList, error)
}

// Commands — ведущий порт команд модуля mes (AD-39).
type Commands interface {
}

// Unimplemented — заглушка портов mes: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func (Unimplemented) Jobs(context.Context, platform.Moment, platform.Page) (MesJobList, error) {
	return MesJobList{}, platform.NotImplemented("mes.order.list")
}

func (Unimplemented) Blocks(context.Context, platform.Moment) (MesBlockList, error) {
	return MesBlockList{}, platform.NotImplemented("mes.block.list")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
