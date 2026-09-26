package analytics

import (
	"context"
	"encoding/json"
	"slices"

	engineapp "ant/internal/application/engine"
	domain "ant/internal/domain/analytics"
)

// ProjectionSource — строки вклада и проекции в памяти (журнал движка в
// памяти, enginemem): то, что пишут эффекты движка.
type ProjectionSource interface {
	Contributions(item string) []engineapp.Contribution
	Projection(name string) map[string]json.RawMessage
}

// MemStore — Store над хранилищем эффектов в памяти (тесты, AD-45): строки
// вклада читаются по списку изделий.
type MemStore struct {
	Src   ProjectionSource
	Items func() []string
}

var _ Store = MemStore{}

// Rows — строки вклада изделий.
func (m MemStore) Rows(_ context.Context, metrics ...string) ([]domain.Row, error) {
	var out []domain.Row
	for _, it := range m.Items() {
		for _, c := range m.Src.Contributions(it) {
			if len(metrics) > 0 && !slices.Contains(metrics, c.Metric) {
				continue
			}
			r, err := DecodeRow(c.ItemID, c.Metric, c.Slice, c.Value, c.Sources)
			if err != nil {
				return nil, err
			}
			out = append(out, r)
		}
	}
	return out, nil
}

// Equipment — проекции оборудования.
func (m MemStore) Equipment(context.Context) ([]domain.Equipment, error) {
	return memProj[domain.Equipment](m.Src.Projection(ProjectionEquipment))
}

// Incidents — проекции инцидентов.
func (m MemStore) Incidents(context.Context) ([]domain.Incident, error) {
	return memProj[domain.Incident](m.Src.Projection(ProjectionIncident))
}

func memProj[T any](p map[string]json.RawMessage) ([]T, error) {
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	out := make([]T, 0, len(keys))
	for _, k := range keys {
		var v T
		if err := json.Unmarshal(p[k], &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}
