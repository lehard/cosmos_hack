package reference

import (
	"context"
	"slices"
	"sync"

	engineapp "ant/internal/application/engine"
	appjournal "ant/internal/application/journal"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/reference"
)

// BookSource — ведомый порт справочника (AD-31, AD-45): все версии
// справочников, известные на basis_seq (basisSeq = 0 — всё записанное к
// моменту вызова). Срез по прогону, оси и дате делает вызывающий
// (dom.Book.Slice и функции «…At»).
type BookSource interface {
	Book(ctx context.Context, basisSeq int64) (dom.Book, error)
}

// JournalSource — BookSource над журналом: записи справочника читаются из
// журнала по типам (индекс event_type, seq) и сворачиваются чистой функцией
// dom.Apply — так же, как их построит верификатор (AD-9, AD-45: вход
// свёртки строится из журнала, а не из отстающей проекции, поэтому свёртке
// не нужно ждать курсора). Книга кэшируется в памяти процесса и дочитывается
// с последней позиции; версии хранят seq, поэтому срез на любой меньший
// basis_seq — фильтр той же книги.
type JournalSource struct {
	Journal appjournal.JournalStore
	Codec   *engineapp.Codec

	mu   sync.Mutex
	book dom.Book
	// upTo — все записи справочника с seq ≤ upTo уже в book.
	upTo int64
}

var _ BookSource = (*JournalSource)(nil)

// readLimit — размер страницы чтения журнала.
const readLimit = 1000

// Book — справочник, полный на basisSeq (0 — на голову журнала).
func (s *JournalSource) Book(ctx context.Context, basisSeq int64) (dom.Book, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if basisSeq > 0 && basisSeq <= s.upTo {
		return s.book, nil
	}
	target := basisSeq
	if target <= 0 {
		h, err := s.Journal.Head(ctx)
		if err != nil {
			return dom.Book{}, err
		}
		target = h.MainSeq
		if target <= s.upTo {
			return s.book, nil
		}
	}
	// Видимость по seq монотонна (AD-44): все записи с seq ≤ target уже
	// видны; записи позже target не применяются — их дочитает следующий вызов.
	var recs []kernel.Record
	for _, t := range dom.Types {
		after := s.upTo
		for {
			es, err := s.Journal.Read(ctx, appjournal.ReadQuery{EventType: string(t), AfterSeq: after, Limit: readLimit})
			if err != nil {
				return dom.Book{}, err
			}
			for _, e := range es {
				after = int64(e.Seq)
				if int64(e.Seq) > target {
					break
				}
				d, err := s.Codec.Decode(ctx, e)
				if err != nil {
					return dom.Book{}, err
				}
				recs = append(recs, d.Record)
			}
			if len(es) < readLimit || after > target {
				break
			}
		}
	}
	slices.SortFunc(recs, func(a, b kernel.Record) int { return int(a.Seq - b.Seq) })
	b := s.book
	for _, r := range recs {
		var err error
		if b, err = dom.Apply(b, r); err != nil {
			return dom.Book{}, err
		}
	}
	s.book, s.upTo = b, target
	return b, nil
}

// StaticSource — BookSource над готовой книгой (заготовки, тесты).
type StaticSource struct{ B dom.Book }

// Book — книга целиком.
func (s StaticSource) Book(context.Context, int64) (dom.Book, error) { return s.B, nil }
