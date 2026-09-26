package journal_test

import (
	"testing"

	"ant/internal/application/journal"
	"ant/internal/application/platform"
)

// Страницы журнала на заготовках: порядок «новые сверху», курсор, префикс
// семейства и поиск по event_id — как у live (интерфейс 6).
func TestPageEntries(t *testing.T) {
	var all []journal.JournalEntryView
	for i := int64(1); i <= 7; i++ {
		typ := "quality.signal.raised"
		if i%2 == 0 {
			typ = "item.item.registered"
		}
		all = append(all, journal.JournalEntryView{Seq: i, EventID: string(rune('a' + i)), EventType: typ, EntryKind: "fact"})
	}
	p := journal.PageEntries(all, journal.EntryFilter{Order: journal.OrderDesc}, platform.Page{Limit: 3})
	if len(p.Items) != 3 || p.Items[0].Seq != 7 || p.NextCursor != "5" {
		t.Fatalf("первая страница desc: %+v %q", p.Items, p.NextCursor)
	}
	p = journal.PageEntries(all, journal.EntryFilter{Order: journal.OrderDesc}, platform.Page{Limit: 3, Cursor: p.NextCursor})
	if len(p.Items) != 3 || p.Items[0].Seq != 4 {
		t.Fatalf("вторая страница desc: %+v", p.Items)
	}
	p = journal.PageEntries(all, journal.EntryFilter{EventType: "quality."}, platform.Page{Limit: 10})
	if len(p.Items) != 4 || p.Items[0].Seq != 1 || p.NextCursor != "" {
		t.Fatalf("префикс семейства: %+v", p.Items)
	}
	p = journal.PageEntries(all, journal.EntryFilter{EventID: "e"}, platform.Page{})
	if len(p.Items) != 1 || p.Items[0].Seq != 4 {
		t.Fatalf("по event_id: %+v", p.Items)
	}
}
