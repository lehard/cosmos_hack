package cad

import (
	"context"

	app "ant/internal/application/cad"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/infrastructure/transport/httpapi"
)

// Register объявляет операции модуля cad: импорт условной сборки КОМПАС-3D
// (FR-94, PRD §11.15, AD-18).
func Register(api *httpapi.API, q app.Queries, c app.Commands) {
	const owner = "cad"
	httpapi.Read(api, httpapi.Get("/cad/assemblies", "Условные сборки", "FR-94: импортированные сборки — состав и связи W-1, J-1, S-1."),
		platform.Action{ID: "cad.assembly.list", Owner: owner},
		func(ctx context.Context, in *struct{ httpapi.MomentQuery }, m platform.Moment) (app.CadAssemblyList, error) {
			return q.Assemblies(ctx, m)
		})
	httpapi.Do(api, httpapi.Post("/cad/assemblies", "Импортировать условную сборку",
		"FR-94: файл условной сборки по образцу ФЛ-100.00.000 СБ; geometry: null; связи переводятся в зоны и ограничения нормативного слоя."),
		platform.Action{ID: "cad.assembly.import", Class: platform.ClassRecord, Owner: owner, Emits: []catalog.Type{catalog.CadAssemblyImported}},
		func(ctx context.Context, in *struct{ Body app.ImportAssembly }) (platform.Receipt, error) {
			return c.ImportAssembly(ctx, in.Body)
		})
}
