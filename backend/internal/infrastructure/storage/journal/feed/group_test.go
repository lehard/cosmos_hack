package feed_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	app "ant/internal/application/journal"
	jc "ant/internal/contracts/journal"
	store "ant/internal/infrastructure/storage/journal"
	"ant/internal/infrastructure/storage/journal/feed"
	jt "ant/internal/infrastructure/storage/journal/journaltest"
)

// Группа потребителей (эпик 35): каждому имени записи отдаются только после
// его курсора, выход и курсоры всех имён — одна транзакция; имя с отставшим
// курсором догоняет, не повторяя остальным уже отданное.
func TestConsumeGroup(t *testing.T) {
	d := jt.NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	adm := d.AdminConn(t)
	if _, err := adm.Exec(ctx, `CREATE SCHEMA grp; CREATE TABLE grp.seen (name text, seq bigint, PRIMARY KEY (name, seq));
GRANT USAGE ON SCHEMA grp TO ant_app; GRANT ALL ON grp.seen TO ant_app`); err != nil {
		t.Fatal(err)
	}
	pool := d.AppPool(t)
	s := store.NewStore(pool, jt.SysClock{})
	leases := store.NewLeases(pool, jt.SysClock{})
	sig := store.NewListener(pool, nil)
	go func() { _ = sig.Run(ctx) }()
	const total = 20
	for i := range total {
		if _, err := s.Append(ctx, app.AppendRequest{Batch: []app.Pending{jt.Fact(fmt.Sprintf("ENT01:G-%d", i%4))}}); err != nil {
			t.Fatal(err)
		}
	}
	// «b» уже обработал первые 12 записей отдельным потребителем.
	if _, err := s.Append(ctx, app.AppendRequest{Consumer: &app.CursorAdvance{Name: "b", Partition: app.GlobalPartition, Seq: 12}}); err != nil {
		t.Fatal(err)
	}
	names := []string{"a", "b"}
	cctx, ccancel := context.WithCancel(ctx)
	c := feed.NewConsumer(s, leases, sig, feed.Options{Holder: "copy-1", TTL: 2 * time.Second, Batch: 7})
	errc := make(chan error, 1)
	go func() {
		errc <- c.ConsumeGroup(cctx, names, func(ctx context.Context, batch []jc.JournalEntry, from map[string]int64) (app.AppendRequest, error) {
			return app.AppendRequest{Project: func(ctx context.Context, _ app.AppendResult) error {
				tx, ok := store.Tx(ctx)
				if !ok {
					return errors.New("нет транзакции в Project")
				}
				for _, n := range names {
					for _, e := range batch {
						if int64(e.Seq) <= from[n] {
							continue
						}
						if _, err := tx.Exec(ctx, "INSERT INTO grp.seen (name, seq) VALUES ($1, $2)", n, e.Seq); err != nil {
							return err
						}
					}
				}
				return nil
			}}, nil
		})
	}()
	for {
		a, _ := s.Cursor(ctx, "a", app.GlobalPartition)
		b, _ := s.Cursor(ctx, "b", app.GlobalPartition)
		if a >= total && b >= total {
			break
		}
		select {
		case err := <-errc:
			t.Fatalf("группа остановилась: %v", err)
		case <-ctx.Done():
			t.Fatalf("курсоры a=%d b=%d не дошли до %d", a, b, total)
		case <-time.After(50 * time.Millisecond):
		}
	}
	ccancel()
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
	for n, want := range map[string]int64{"a": total, "b": total - 12} {
		var got int64
		if err := adm.QueryRow(ctx, "SELECT count(*) FROM grp.seen WHERE name = $1", n).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s: отдано %d записей, ожидалось %d", n, got, want)
		}
	}
}
