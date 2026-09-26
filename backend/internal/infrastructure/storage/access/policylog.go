package access

import (
	"context"
	"slices"

	app "ant/internal/application/access"
	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	jc "ant/internal/contracts/journal"
	accessdom "ant/internal/domain/access"
)

// PolicyLog — записи политики из журнала (порт application/access.PolicyLog,
// AD-15: источник правды политики — журнал): policy.* и access.person/account
// по индексу (event_type, seq), расшифрованные и повышенные до текущей версии
// схемы тем же кодеком, что и у движка (AD-20).
type PolicyLog struct {
	Journal appjournal.JournalStore
	Codec   *engineapp.Codec
	// Signal — «есть новое» (LISTEN/NOTIFY); nil — проекция перечитывает по интервалу.
	Signal appjournal.Signal
}

var _ app.PolicyLog = PolicyLog{}

// readLimit — записей за одно чтение.
const readLimit = 1000

// Since — записи политики после seq по возрастанию seq.
func (l PolicyLog) Since(ctx context.Context, afterSeq int64) ([]accessdom.Record, error) {
	var out []accessdom.Record
	for _, t := range accessdom.Types {
		after := afterSeq
		for {
			es, err := l.Journal.Read(ctx, appjournal.ReadQuery{EventType: string(t), AfterSeq: after, Limit: readLimit})
			if err != nil {
				return nil, err
			}
			for _, e := range es {
				d, err := l.Codec.Decode(ctx, e)
				if err != nil {
					return nil, err
				}
				out = append(out, accessdom.Record{Seq: d.Record.Seq, Type: string(t), Data: d.Record.Data, OccurredAt: d.Record.OccurredAt,
					Genesis: e.ProvenanceClass == jc.JournalEntryProvenanceClassGenesis})
				after = int64(e.Seq)
			}
			if len(es) < readLimit {
				break
			}
		}
	}
	slices.SortFunc(out, func(a, b accessdom.Record) int {
		switch {
		case a.Seq < b.Seq:
			return -1
		case a.Seq > b.Seq:
			return 1
		}
		return 0
	})
	return out, nil
}

// Head — seq головы по сигналу «есть новое»; нет сигнала — 0.
func (l PolicyLog) Head() int64 {
	if l.Signal == nil {
		return 0
	}
	return l.Signal.Head()
}
