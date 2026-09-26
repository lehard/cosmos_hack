package main

import (
	"context"
	"sync"

	journalapp "ant/internal/application/journal"
	journalfx "ant/internal/infrastructure/fixtures/journal"
)

// withLiveStream — заготовки journal (журнал, голова, таймлайн — из мира
// заготовок) с общим SSE-каналом: смена шага курсора заготовок и изменения
// движка по фактам живого приёма (профиль fixtures с ports.modules.ingest:
// live, AD-36). Живая подписка — только новые изменения (after = 0): seq
// заготовок и seq журнала — разные пространства, догонять по Last-Event-ID
// живой журнал не нужно.
func withLiveStream(fx *journalfx.Adapter, live journalapp.Queries) *mergedStream {
	return &mergedStream{Adapter: fx, live: live}
}

// mergedStream — адаптер заготовок journal с подменённой подпиской.
type mergedStream struct {
	*journalfx.Adapter
	live journalapp.Queries
}

// Subscribe открывает обе подписки и сливает их в одну.
func (m *mergedStream) Subscribe(ctx context.Context, afterSeq int64, runID string) (journalapp.Subscription, error) {
	fs, err := m.Adapter.Subscribe(ctx, afterSeq, runID)
	if err != nil {
		return nil, err
	}
	ls, err := m.live.Subscribe(ctx, 0, runID)
	if err != nil {
		fs.Close()
		return nil, err
	}
	return newMerged(fs, ls), nil
}

// merged — подписка-слияние: по горутине на источник, общая очередь.
type merged struct {
	subs   []journalapp.Subscription
	ch     chan journalapp.Change
	errc   chan error
	cancel context.CancelFunc
	once   sync.Once
	wg     sync.WaitGroup
}

func newMerged(subs ...journalapp.Subscription) *merged {
	ctx, cancel := context.WithCancel(context.Background())
	m := &merged{subs: subs, ch: make(chan journalapp.Change), errc: make(chan error, len(subs)), cancel: cancel}
	for _, s := range subs {
		m.wg.Go(func() {
			for {
				c, err := s.Next(ctx)
				if err != nil {
					m.errc <- err
					return
				}
				select {
				case m.ch <- c:
				case <-ctx.Done():
					return
				}
			}
		})
	}
	return m
}

// Next — следующее изменение любого источника; ошибка источника закрывает поток
// (клиент переподключится).
func (m *merged) Next(ctx context.Context) (journalapp.Change, error) {
	select {
	case c := <-m.ch:
		return c, nil
	case err := <-m.errc:
		return journalapp.Change{}, err
	case <-ctx.Done():
		return journalapp.Change{}, ctx.Err()
	}
}

// Close останавливает оба источника.
func (m *merged) Close() {
	m.once.Do(func() {
		m.cancel()
		for _, s := range m.subs {
			s.Close()
		}
		m.wg.Wait()
	})
}
