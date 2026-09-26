package main

import (
	"context"
	"sync"
	"time"

	analyticsapp "ant/internal/application/analytics"
	referenceapp "ant/internal/application/reference"
	dom "ant/internal/domain/reference"
	referencefx "ant/internal/infrastructure/fixtures/reference"
	"ant/internal/infrastructure/storage/journal/clock"
	referencestore "ant/internal/infrastructure/storage/reference"
)

// Сборка модуля reference (эпик 19): справочники — версионируемые записи
// журнала (AD-31). Срез справочника на basis_seq изделия — внешний слой
// нормативного слоя изделия (process.go: bundleSource) для предусловий
// процесса и сроков notifications; календарь сроков nonconformity
// (nonconformity.go); график смен analytics (analytics.go); живые операции
// reference.*; стартовые справочники — записи блока генезиса (ant init, эпик 05).

// referenceLive — live-реализация операций reference для роли api: чтение —
// срез справочника из журнала ядра на момент (AD-22); команды — гард и
// решение в журнал с проверкой AD-39; доменное «сейчас» — часы журнала (AD-37).
func referenceLive(ctx context.Context, env *environment) (*referenceapp.Service, error) {
	c, err := env.core(ctx)
	if err != nil {
		return nil, err
	}
	return referenceapp.NewService(
		referenceapp.WithSource(c.refSource),
		referenceapp.WithWriter(referenceapp.JournalWriter{Journal: c.journal, DomainBuild: c.codec.DomainBuild, Now: c.codec.Now}),
		referenceapp.WithClock(c.domainClock()),
	), nil
}

// referenceFixtures — заготовки reference: книга стартовых справочников
// (встроенная копия normative/reference) без журнала.
var referenceFixtures = sync.OnceValue(func() *referencefx.Adapter {
	recs, err := referencestore.SeedRecords()
	if err != nil {
		return referencefx.New()
	}
	b, err := referenceapp.SeedBook(recs)
	if err != nil {
		return referencefx.New()
	}
	return referencefx.NewWithBook(b)
})

// referenceShifts — график смен справочника для analytics (FR-81): смена
// на момент по срезу прогона; одна книга на запрос показателей.
type referenceShifts struct{ src referenceapp.BookSource }

// Schedule — график смен прогона run.
func (r referenceShifts) Schedule(ctx context.Context, run string) (analyticsapp.ShiftLookup, error) {
	b, err := r.src.Book(ctx, 0)
	if err != nil {
		return nil, err
	}
	sl := b.Slice(dom.Filter{RunID: run})
	if len(sl.Shifts) == 0 {
		return nil, nil
	}
	memo := map[int64]analyticsapp.ShiftSpan{}
	miss := map[int64]bool{}
	return func(t time.Time) (analyticsapp.ShiftSpan, bool) {
		k := t.Unix() / 60 // по минутам
		if s, ok := memo[k]; ok {
			return s, true
		}
		if miss[k] {
			return analyticsapp.ShiftSpan{}, false
		}
		s, ok := sl.ShiftAt(t, "")
		active := ok
		if !ok {
			s, ok = sl.ShiftBefore(t, "")
		}
		if !ok {
			miss[k] = true
			return analyticsapp.ShiftSpan{}, false
		}
		label := s.Name
		if label == "" {
			label = s.ShiftID
		}
		span := analyticsapp.ShiftSpan{ID: s.ID, Label: label + " " + s.From.In(dom.Local).Format("02.01"), From: s.From, To: s.To, Active: active}
		memo[k] = span
		return span, true
	}, nil
}

var _ analyticsapp.Shifts = referenceShifts{}
