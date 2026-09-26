package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	jc "ant/internal/contracts/journal"
	"ant/internal/domain/crossitem"
)

// Реестр носителей для приёма (AD-16, AD-41) — ОБХОД до эпика 18.
//
// TODO(18): реестр носителей межизделийной стадии эпика 18 заменяет этот
// файл при слиянии. До него приём не мог привязать событие источника
// (камера, маркировщик) к изделию по носителю — живой путь UJ-6 обрывался на
// первом сигнале камеры. Здесь — минимальный реестр по записям журнала
// item.carrier.applied / item.carrier.removed с item_id: «носитель действовал
// на момент» (снятие и замена — replaces_value — закрывают интервал
// включительно: событие, пришедшее по старой метке в момент перемаркировки,
// ещё привязывается к изделию).

// carrierSpan — интервал действия носителя у изделия.
type carrierSpan struct {
	item  string
	from  time.Time
	until *time.Time
}

// journalCarriers — CarrierRegistry над журналом: кэш с догрузкой новых
// записей носителей по seq при каждом запросе.
type journalCarriers struct {
	store appjournal.JournalStore

	mu    sync.Mutex
	after map[catalog.Type]int64
	spans map[string][]carrierSpan // значение носителя → интервалы
}

var _ crossitem.CarrierRegistry = (*journalCarriers)(nil)

// carriers — реестр носителей ядра процесса (один на процесс).
func (c *core) carriers() crossitem.CarrierRegistry {
	c.carrierOnce.Do(func() {
		c.carrierReg = &journalCarriers{store: c.journal, after: map[catalog.Type]int64{}, spans: map[string][]carrierSpan{}}
	})
	return c.carrierReg
}

// Lookup — изделия, у которых носитель ref действовал на момент at.
func (r *journalCarriers) Lookup(ref crossitem.CarrierRef, at time.Time) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refresh(ctx)
	var out []string
	for _, s := range r.spans[ref.Value] {
		if s.from.After(at) || (s.until != nil && at.After(*s.until)) {
			continue
		}
		dup := false
		for _, x := range out {
			dup = dup || x == s.item
		}
		if !dup {
			out = append(out, s.item)
		}
	}
	return out
}

// refresh — догрузить записи носителей каждого типа после своего курсора.
func (r *journalCarriers) refresh(ctx context.Context) {
	for _, t := range []catalog.Type{catalog.ItemCarrierApplied, catalog.ItemCarrierRemoved} {
		for {
			page, err := r.store.Read(ctx, appjournal.ReadQuery{EventType: string(t), AfterSeq: r.after[t], Limit: 500})
			if err != nil || len(page) == 0 {
				break
			}
			for _, e := range page {
				r.apply(ctx, e)
				r.after[t] = int64(e.Seq)
			}
			if len(page) < 500 {
				break
			}
		}
	}
}

func (r *journalCarriers) apply(ctx context.Context, e jc.JournalEntry) {
	t := catalog.Type(e.EventType)
	if (t != catalog.ItemCarrierApplied && t != catalog.ItemCarrierRemoved) || e.ItemID == nil || *e.ItemID == "" {
		return
	}
	var d struct {
		Value         string  `json:"value"`
		ReplacesValue *string `json:"replaces_value"`
	}
	if !entryData(ctx, r.store, e, &d) || d.Value == "" {
		return
	}
	at, err := time.Parse(time.RFC3339Nano, e.OccurredAt)
	if err != nil {
		return
	}
	item := *e.ItemID
	closeSpan := func(value string) {
		for i := range r.spans[value] {
			s := &r.spans[value][i]
			if s.item == item && s.until == nil {
				u := at
				s.until = &u
			}
		}
	}
	if t == catalog.ItemCarrierRemoved {
		closeSpan(d.Value)
		return
	}
	if d.ReplacesValue != nil && *d.ReplacesValue != "" {
		closeSpan(*d.ReplacesValue)
	}
	r.spans[d.Value] = append(r.spans[d.Value], carrierSpan{item: item, from: at})
}

// entryData — поле data события из конверта DSSE записи.
func entryData(ctx context.Context, store appjournal.JournalStore, e jc.JournalEntry, v any) bool {
	env, err := store.Open(ctx, e)
	if err != nil {
		return false
	}
	var dsse struct {
		Payload string `json:"payload"`
	}
	if json.Unmarshal(env.Raw, &dsse) != nil {
		return false
	}
	payload, err := base64.StdEncoding.DecodeString(dsse.Payload)
	if err != nil {
		return false
	}
	var ev struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(payload, &ev) != nil {
		return false
	}
	return json.Unmarshal(ev.Data, v) == nil
}
