package item

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"

	"go.yaml.in/yaml/v3"

	engineapp "ant/internal/application/engine"
	"ant/internal/contracts/normative"
	"ant/internal/domain/engine"
	dom "ant/internal/domain/item"
	"ant/internal/domain/kernel"
)

// PathItemTypes — номенклатура с зонами и связями по КД (AD-31, FR-46).
const PathItemTypes = "normative/reference/flange/item-types.yaml"

// EnvFromFS — Env модуля item по номенклатуре нормативного слоя (корень fsys —
// корень репозитория или встроенная копия).
func EnvFromFS(fsys fs.FS) (dom.Env, error) {
	b, err := fs.ReadFile(fsys, PathItemTypes)
	if err != nil {
		return dom.Env{}, err
	}
	return EnvFromYAML(b)
}

// EnvFromYAML — Env модуля item по содержимому item-types.yaml.
func EnvFromYAML(b []byte) (dom.Env, error) {
	var raw any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return dom.Env{}, err
	}
	j, err := json.Marshal(raw)
	if err != nil {
		return dom.Env{}, err
	}
	var seed normative.ItemTypesSeed
	if err := json.NewDecoder(bytes.NewReader(j)).Decode(&seed); err != nil {
		return dom.Env{}, err
	}
	env := dom.Env{Types: map[string]dom.TypeDef{}}
	for _, t := range seed.ItemTypes {
		d := dom.TypeDef{Marking: string(t.Marking)}
		for _, z := range t.Zones {
			d.Zones = append(d.Zones, dom.ZoneDef{ID: z.ZoneID, Name: z.Name, Kind: string(z.Kind)})
		}
		for _, l := range t.Links {
			d.Links = append(d.Links, dom.LinkDef{ID: l.LinkID, Kind: string(l.Kind), Zones: l.Zones, ClosesAccessTo: l.ClosesAccessTo})
		}
		env.Types[t.ID] = d
	}
	return env, nil
}

// Bundles — BundleSource движка, дополняющий нормативный слой изделия частью
// модуля item (AD-17): если источник версии (эпик 17) не собрал Env модуля
// item, подставляется номенклатура нормативного слоя. Декорирует Next.
type Bundles struct {
	// Next — источник версии; nil — пустой нормативный слой.
	Next engineapp.BundleSource
	// Env — часть нормативного слоя item по умолчанию (EnvFromFS).
	Env dom.Env
}

// Bundle — нормативный слой изделия с частью item.
func (b Bundles) Bundle(ctx context.Context, itemID string, input []kernel.Record) (engine.Bundle, string, error) {
	next := b.Next
	if next == nil {
		next = engineapp.EmptyBundles{}
	}
	bd, rev, err := next.Bundle(ctx, itemID, input)
	if err != nil {
		return bd, rev, err
	}
	if len(bd.Item.Types) == 0 {
		bd.Item = b.Env
	}
	return bd, rev, nil
}

var _ engineapp.BundleSource = Bundles{}
