package analytics

import (
	"encoding/json"
	"fmt"
	"time"

	engineapp "ant/internal/application/engine"
	domain "ant/internal/domain/analytics"
)

// Строка вклада домена ↔ строка вклада движка (AD-45): engine.contributions
// хранит (item_id, metric, slice, value, scale, sources); срез, момент,
// интервал и виды источников — в поле slice как канонический JSON (формат
// владеет analytics, движок его не толкует).

// sliceDoc — содержимое поля slice строки вклада.
type sliceDoc struct {
	At       time.Time   `json:"at"`
	Interval bool        `json:"iv,omitempty"`
	Until    *time.Time  `json:"until,omitempty"`
	Unit     string      `json:"u,omitempty"`
	Dims     domain.Dims `json:"d"`
	Kinds    []string    `json:"k,omitempty"`
}

// EncodeRow — строка вклада движка из строки домена.
func EncodeRow(itemID string, r domain.Row) (engineapp.Contribution, error) {
	doc := sliceDoc{At: r.At.UTC(), Interval: r.Interval, Unit: r.Unit, Dims: r.Dims, Kinds: r.Kinds}
	if r.Until != nil {
		u := r.Until.UTC()
		doc.Until = &u
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return engineapp.Contribution{}, err
	}
	return engineapp.Contribution{ItemID: itemID, Metric: r.Metric, Slice: string(b), Value: r.Value, Sources: r.Sources}, nil
}

// DecodeRow — строка домена из хранимой строки вклада.
func DecodeRow(itemID, metric, slice string, value int64, sources []string) (domain.Row, error) {
	var doc sliceDoc
	if err := json.Unmarshal([]byte(slice), &doc); err != nil {
		return domain.Row{}, fmt.Errorf("вклад %s/%s: срез: %w", itemID, metric, err)
	}
	return domain.Row{Metric: metric, Item: itemID, At: doc.At, Interval: doc.Interval, Until: doc.Until, Value: value,
		Unit: doc.Unit, Dims: doc.Dims, Sources: sources, Kinds: doc.Kinds}, nil
}
