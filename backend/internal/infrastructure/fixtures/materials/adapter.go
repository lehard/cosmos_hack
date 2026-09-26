package materials

import (
	"context"

	app "ant/internal/application/materials"
	"ant/internal/infrastructure/fixtures/loader"
)

// Adapter — реализация fixtures ведущих портов модуля materials: метаданные и
// содержимое материалов мира заготовок (иллюстрации к наблюдениям камеры с
// пометкой «ИЛЛЮСТРАЦИЯ», FR-102); загрузка — 501 (в заготовках не пишем).
type Adapter struct {
	app.Unimplemented
}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Material — метаданные материала по адресу (materials.material.read).
func (Adapter) Material(ctx context.Context, address string) (app.MaterialInfo, error) {
	var out app.MaterialInfo
	rt, err := loader.Default()
	if err != nil {
		return out, err
	}
	err = rt.Respond(ctx, "materials.material.read", map[string]string{"address": address}, nil, &out)
	return out, err
}

// Content — байты материала по адресу (materials.material.content): в
// заготовках тело — байты (base64 в JSON; иллюстрации мира — SVG), тип
// содержимого — из метаданных материала.
func (a Adapter) Content(ctx context.Context, address string) (app.MaterialContent, error) {
	info, err := a.Material(ctx, address)
	if err != nil {
		return app.MaterialContent{}, err
	}
	rt, err := loader.Default()
	if err != nil {
		return app.MaterialContent{}, err
	}
	var b []byte
	if err := rt.Respond(ctx, "materials.material.content", map[string]string{"address": address}, nil, &b); err != nil {
		return app.MaterialContent{}, err
	}
	return app.MaterialContent{MediaType: info.MediaType, Bytes: b}, nil
}
