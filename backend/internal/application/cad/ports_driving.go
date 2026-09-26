package cad

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля cad (AD-36).
type Queries interface {
	// Assemblies — импортированные условные сборки (FR-94) (cad.assembly.list).
	Assemblies(ctx context.Context, m platform.Moment) (CadAssemblyList, error)
}

// Commands — ведущий порт команд модуля cad (AD-39).
type Commands interface {
	// ImportAssembly — cad.assembly.import.
	ImportAssembly(ctx context.Context, in ImportAssembly) (platform.Receipt, error)
}

// Unimplemented — заглушка портов cad: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func (Unimplemented) Assemblies(context.Context, platform.Moment) (CadAssemblyList, error) {
	return CadAssemblyList{}, platform.NotImplemented("cad.assembly.list")
}

func (Unimplemented) ImportAssembly(context.Context, ImportAssembly) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("cad.assembly.import")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
