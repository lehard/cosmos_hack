package quality

import (
	"context"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	appquality "ant/internal/application/quality"
	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
)

// JournalPassports — записи паспортов допуска анализатора из журнала
// (analyzer.passport.*, потоки analyzer_passport:‹id›, эмитент vision, AD-29).
// Читает через порт журнала и кодек движка (повышение версий, AD-20).
type JournalPassports struct {
	Journal appjournal.JournalStore
	Codec   *engineapp.Codec
}

var _ appquality.PassportRecords = JournalPassports{}

// passportTypes — типы записей паспортов.
var passportTypes = []catalog.Type{
	catalog.AnalyzerPassportAdmitted, catalog.AnalyzerPassportSuspended,
	catalog.AnalyzerPassportReinstated, catalog.AnalyzerPassportRetired,
}

// PassportRecords — все записи паспортов в порядке seq.
func (j JournalPassports) PassportRecords(ctx context.Context) ([]kernel.Record, error) {
	out := []kernel.Record{}
	for _, t := range passportTypes {
		var after int64
		for {
			es, err := j.Journal.Read(ctx, appjournal.ReadQuery{EventType: string(t), AfterSeq: after, Limit: 1000})
			if err != nil {
				return nil, err
			}
			for _, e := range es {
				after = int64(e.Seq)
				d, err := j.Codec.Decode(ctx, e)
				if err != nil {
					continue // нечитаемый паспорт — не паспорт: уровень доверия 0 строже
				}
				out = append(out, d.Record)
			}
			if len(es) < 1000 {
				break
			}
		}
	}
	return out, nil
}
