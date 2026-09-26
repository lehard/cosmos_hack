package quality

import (
	"context"

	engineapp "ant/internal/application/engine"
	"ant/internal/domain/engine"
	"ant/internal/domain/kernel"
	"ant/internal/domain/quality"
)

// PassportRecords — ведомый порт: записи потоков паспортов допуска
// анализатора (analyzer.passport.*, эмитент vision, AD-29) — вход
// quality.PassportsFrom.
type PassportRecords interface {
	PassportRecords(ctx context.Context) ([]kernel.Record, error)
}

// Bundles — BundleSource движка, дополняющий нормативный слой изделия частью
// модуля quality (AD-17): если источник версии (эпик 17: process) не собрал
// Env модуля quality, подставляется Env нормативного слоя; паспорта
// анализатора — из их потоков (срез на occurred_at наблюдения делает домен).
// Когда BundleSource процесса начнёт собирать Env сам (EnvFrom +
// PassportsFrom), декоратор ничего не меняет.
type Bundles struct {
	// Next — источник версии; nil — пустой нормативный слой.
	Next engineapp.BundleSource
	// Env — часть нормативного слоя quality по умолчанию (EnvFromFS).
	Env quality.Env
	// Passports — записи паспортов; nil — паспортов нет (уровень доверия 0).
	Passports PassportRecords
}

// Bundle — нормативный слой изделия с частью quality.
func (b Bundles) Bundle(ctx context.Context, itemID string, input []kernel.Record) (engine.Bundle, string, error) {
	next := b.Next
	if next == nil {
		next = engineapp.EmptyBundles{}
	}
	bd, rev, err := next.Bundle(ctx, itemID, input)
	if err != nil {
		return bd, rev, err
	}
	if bd.Quality.IsZero() {
		bd.Quality = b.Env
		if rev != "" {
			bd.Quality.RuleRev = rev
		}
	}
	if len(bd.Quality.Passports) == 0 && b.Passports != nil {
		recs, err := b.Passports.PassportRecords(ctx)
		if err != nil {
			return bd, rev, err
		}
		bd.Quality.Passports = quality.PassportsFrom(recs)
	}
	if rev == "" {
		rev = bd.Quality.RuleRev
	}
	return bd, rev, nil
}

var _ engineapp.BundleSource = Bundles{}
