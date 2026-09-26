package materials

import (
	"context"
	"encoding/hex"
	"slices"
	"sync"

	app "ant/internal/application/materials"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"

	"ant/internal/infrastructure/fixtures/loader"
	"go.stargrave.org/gogost/v7/gost34112012256"
)

// Adapter — реализация fixtures ведущих портов модуля materials: метаданные и
// содержимое материалов мира заготовок (иллюстрации к наблюдениям камеры с
// пометкой «ИЛЛЮСТРАЦИЯ», FR-102); загрузка (скан бумажной подписи для
// заверения, FR-139) — в память процесса: адрес содержимого как у live
// (H(байты), ГОСТ Р 34.11-2012), материал читается по адресу до перезапуска.
type Adapter struct {
	app.Unimplemented
	mu       sync.Mutex
	uploaded map[string]uploaded
}

// uploaded — материал, загруженный в этом процессе.
type uploaded struct {
	info  app.MaterialInfo
	bytes []byte
}

// Upload — загрузить материал (materials.material.upload): байты в памяти процесса.
func (a *Adapter) Upload(_ context.Context, in app.MaterialUpload) (app.MaterialInfo, error) {
	if len(in.Bytes) == 0 {
		return app.MaterialInfo{}, platform.Fail(errcodes.ApiValidationFailed, "field", "body", "reason", "пустой материал")
	}
	mt := in.MediaType
	if mt == "" {
		mt = "application/octet-stream"
	}
	h := gost34112012256.New()
	_, _ = h.Write(in.Bytes)
	addr := "streebog256:" + hex.EncodeToString(h.Sum(nil))
	info := app.MaterialInfo{MaterialAddress: addr, MediaType: mt, SizeBytes: int64(len(in.Bytes)), Kind: in.Kind, CapturedAt: in.CapturedAt,
		ItemID: in.ItemID, IsIllustration: in.IsIllustration, ProvenanceNote: in.ProvenanceNote}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.uploaded == nil {
		a.uploaded = map[string]uploaded{}
	}
	a.uploaded[addr] = uploaded{info: info, bytes: slices.Clone(in.Bytes)}
	return info, nil
}

// mine — материал, загруженный в этом процессе.
func (a *Adapter) mine(address string) (uploaded, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	u, ok := a.uploaded[address]
	return u, ok
}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

// Material — метаданные материала по адресу (materials.material.read).
func (a *Adapter) Material(ctx context.Context, address string) (app.MaterialInfo, error) {
	if u, ok := a.mine(address); ok {
		return u.info, nil
	}
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
func (a *Adapter) Content(ctx context.Context, address string) (app.MaterialContent, error) {
	if u, ok := a.mine(address); ok {
		return app.MaterialContent{MediaType: u.info.MediaType, Bytes: slices.Clone(u.bytes)}, nil
	}
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
