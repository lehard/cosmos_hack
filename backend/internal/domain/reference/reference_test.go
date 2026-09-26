package reference_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/domain/kernel"
	ref "ant/internal/domain/reference"
)

type book struct {
	t   *testing.T
	rs  []kernel.Record
	seq int64
}

func (b *book) add(tp catalog.Type, at time.Time, data string) kernel.Record {
	b.t.Helper()
	if !json.Valid([]byte(data)) {
		b.t.Fatalf("неверный JSON: %s", data)
	}
	b.seq++
	r := kernel.Record{Seq: b.seq, EventID: fmt.Sprintf("00000000-0000-7000-8000-%012d", b.seq), Type: tp,
		OccurredAt: at, RecordedAt: at, Data: json.RawMessage(data)}
	b.rs = append(b.rs, r)
	return r
}

func (b *book) build() ref.Book {
	b.t.Helper()
	bk, err := ref.Build(b.rs)
	if err != nil {
		b.t.Fatal(err)
	}
	return bk
}

func msk(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, ref.Local)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}

// Производственный календарь РФ 2026 (Постановление Правительства о переносе
// выходных: 3 января → 9 января, 4 января → 31 декабря) — как в
// normative/reference/flange/calendar.yaml.
const cal2026 = `{"calendar_id":"CAL-ENT01","year":2026,"weekly_days_off":["sat","sun"],
 "non_working_days":["2026-01-01","2026-01-02","2026-01-03","2026-01-04","2026-01-05","2026-01-06","2026-01-07","2026-01-08","2026-01-09",
  "2026-02-23","2026-03-08","2026-03-09","2026-05-01","2026-05-09","2026-05-11","2026-06-12","2026-11-04","2026-12-31"],
 "shortened_days":["2026-04-30","2026-05-08","2026-06-11","2026-11-03"]}`

// FR-55: срок в рабочих днях — по производственному календарю с праздниками.
func TestCalendarWorkingDaysWithHolidays(t *testing.T) {
	b := &book{t: t}
	b.add(catalog.ReferenceCalendarDefined, msk("2025-12-01 09:00"), cal2026)
	c := b.build().Calendar()

	cases := []struct {
		from string
		days int
		want string
	}{
		// Четверг перед 1 мая: пт 1.05 праздник, сб-вс, пн 4.05 — первый рабочий.
		{"2026-04-30 10:00", 3, "2026-05-06 10:00"},
		// Пятница 8 мая: пн 11.05 — перенос за 9 мая.
		{"2026-05-08 15:00", 1, "2026-05-12 15:00"},
		// 30 декабря: 31.12 — перенесённый выходной, дальше 2027 без календаря (пн–пт).
		{"2026-12-30 09:00", 1, "2027-01-01 09:00"},
		// Новогодние каникулы: с 31.12.2025 по 11.01.2026, первый рабочий — 12.01.
		{"2026-01-02 12:00", 1, "2026-01-12 12:00"},
		// Поздний вечер по МСК — это уже следующий день по UTC; считается местная дата.
		{"2026-06-11 23:30", 1, "2026-06-15 23:30"},
	}
	for _, c0 := range cases {
		if got := c.AddWorkingDays(msk(c0.from), c0.days); !got.Equal(msk(c0.want)) {
			t.Errorf("%s + %d раб. дн.: %s, ждали %s", c0.from, c0.days, got.In(ref.Local), c0.want)
		}
	}
	// Без календаря — пн–пт: четверг 30.04 + 3 = вторник 5.05.
	if got := (ref.Calendar{}).AddWorkingDays(msk("2026-04-30 10:00"), 3); !got.Equal(msk("2026-05-05 10:00")) {
		t.Errorf("пн–пт: %s", got.In(ref.Local))
	}
	if !c.Shortened(msk("2026-11-03 12:00")) || c.Shortened(msk("2026-11-05 12:00")) {
		t.Error("сокращённые дни")
	}
	// 247 рабочих дней в 2026 году (производственный календарь РФ).
	n := 0
	for d := msk("2026-01-01 12:00"); d.Before(msk("2027-01-01 00:00")); d = d.Add(24 * time.Hour) {
		if c.Working(d) {
			n++
		}
	}
	if n != 247 {
		t.Errorf("рабочих дней в 2026: %d, ждали 247", n)
	}
	// Новый календарь года заменяет прежний целиком (позднее знание).
	b.add(catalog.ReferenceCalendarDefined, msk("2026-03-01 09:00"), `{"calendar_id":"CAL-ENT01","year":2026,"non_working_days":["2026-04-29"]}`)
	if c2 := b.build().Calendar(); c2.Working(msk("2026-04-29 10:00")) || !c2.Working(msk("2026-05-04 10:00")) {
		t.Error("новая запись календаря года не заменила прежнюю")
	}
}

// FR-17: поверка на дату операции; интервалы действия для предусловий процесса.
func TestEquipmentVerificationOnDate(t *testing.T) {
	b := &book{t: t}
	b.add(catalog.ReferenceEquipmentDefined, msk("2026-01-01 00:00"), `{"equipment_id":"CMM-1","kind":"cmm","location_id":"ST-CMM","name":"КИМ",
		"is_measuring_instrument":true,"valid_from":"2025-12-31T21:00:00.000Z"}`)
	b.add(catalog.ReferenceEquipmentDefined, msk("2026-01-01 00:00"), `{"equipment_id":"CNC-1","kind":"cnc_machine","location_id":"ST-CNC","name":"ЧПУ",
		"is_measuring_instrument":false,"valid_from":"2025-12-31T21:00:00.000Z"}`)
	bk := b.build()
	if st := bk.EquipmentStatusAt("CMM-1", msk("2026-02-01 10:00")); st.Valid || st.Reason != ref.ReasonNotVerified {
		t.Fatalf("средство измерений без поверки: %+v", st)
	}
	if st := bk.EquipmentStatusAt("CNC-1", msk("2026-02-01 10:00")); !st.Valid {
		t.Fatalf("станок без поверки (не СИ): %+v", st)
	}
	if st := bk.EquipmentStatusAt("XX", msk("2026-02-01 10:00")); st.Valid || st.Reason != ref.ReasonUnknown {
		t.Fatalf("неизвестное оборудование: %+v", st)
	}
	b.add(catalog.ReferenceEquipmentVerified, msk("2026-01-15 09:00"), `{"equipment_id":"CMM-1","kind":"calibration","result":"valid","valid_until":"2026-03-31"}`)
	bk = b.build()
	for _, c := range []struct {
		at     string
		valid  bool
		reason string
	}{
		{"2026-01-15 08:59", false, ref.ReasonNotVerified},
		{"2026-01-15 09:00", true, ""},
		{"2026-03-31 23:59", true, ""},
		{"2026-04-01 00:00", false, ref.ReasonExpired},
	} {
		if st := bk.EquipmentStatusAt("CMM-1", msk(c.at)); st.Valid != c.valid || st.Reason != c.reason {
			t.Errorf("%s: %+v", c.at, st)
		}
	}
	vs := bk.EquipmentValidity("CMM-1")
	if len(vs) != 1 || !vs[0].From.Equal(msk("2026-01-15 09:00")) || vs[0].Until == nil || !vs[0].Until.Equal(msk("2026-04-01 00:00")) {
		t.Fatalf("интервалы: %+v", vs)
	}
	// Непригодность прерывает действующую поверку; новая поверка — снова годно.
	b.add(catalog.ReferenceEquipmentVerified, msk("2026-02-10 12:00"), `{"equipment_id":"CMM-1","kind":"verification","result":"invalid"}`)
	b.add(catalog.ReferenceEquipmentVerified, msk("2026-02-20 12:00"), `{"equipment_id":"CMM-1","kind":"calibration","result":"valid","valid_until":"2027-02-19"}`)
	vs = b.build().EquipmentValidity("CMM-1")
	if len(vs) != 2 || !vs[0].Until.Equal(msk("2026-02-10 12:00")) || !vs[1].From.Equal(msk("2026-02-20 12:00")) {
		t.Fatalf("интервалы после непригодности: %+v", vs)
	}
	// Вывод из эксплуатации — новая версия с valid_until.
	b.add(catalog.ReferenceEquipmentDefined, msk("2026-06-01 00:00"), `{"equipment_id":"CNC-1","kind":"cnc_machine","location_id":"ST-CNC","name":"ЧПУ",
		"is_measuring_instrument":false,"valid_from":"2025-12-31T21:00:00.000Z","valid_until":"2026-05-31T21:00:00.000Z"}`)
	bk = b.build()
	if !bk.EquipmentStatusAt("CNC-1", msk("2026-05-31 23:00")).Valid || bk.EquipmentStatusAt("CNC-1", msk("2026-06-01 00:00")).Valid {
		t.Fatal("вывод из эксплуатации")
	}
	// Срез на basis_seq: до записи поверок — поверки нет.
	if bk.Slice(ref.Filter{BasisSeq: 2}).EquipmentStatusAt("CMM-1", msk("2026-03-01 10:00")).Valid {
		t.Fatal("срез на basis_seq видит позднюю поверку")
	}
}

// AD-38: прогон видит общий справочник и свои записи, но не записи других прогонов.
func TestSliceRun(t *testing.T) {
	b := &book{t: t}
	b.add(catalog.ReferenceEquipmentDefined, msk("2026-01-01 00:00"), `{"equipment_id":"IS-1","kind":"welding_source","location_id":"ST-WELD","name":"ИС-1",
		"is_measuring_instrument":false,"valid_from":"2025-12-31T21:00:00.000Z"}`)
	r := b.add(catalog.ReferenceEquipmentVerified, msk("2026-02-01 00:00"), `{"equipment_id":"IS-1","kind":"verification","result":"invalid"}`)
	b.rs[len(b.rs)-1].RunID = "run-A"
	_ = r
	bk := b.build()
	if !bk.Slice(ref.Filter{RunID: "run-B"}).EquipmentStatusAt("IS-1", msk("2026-03-01 10:00")).Valid {
		t.Fatal("прогон B видит запись прогона A")
	}
	if bk.Slice(ref.Filter{RunID: "run-A"}).EquipmentStatusAt("IS-1", msk("2026-03-01 10:00")).Valid {
		t.Fatal("прогон A не видит своей записи")
	}
	f := ref.ItemFilter([]kernel.Record{{Seq: 7, RunID: "run-A"}, {Seq: 12, RunID: "run-A"}})
	if f.BasisSeq != 12 || f.RunID != "run-A" {
		t.Fatalf("фильтр изделия: %+v", f)
	}
}

// FR-95, AD-18: конфликт соответствий — сигнал, не перезапись.
func TestExternalIDConflict(t *testing.T) {
	b := &book{t: t}
	b.add(catalog.ReferenceExternalIdMapped, msk("2026-01-01 00:00"), `{"system":"onec","object_kind":"order","external_id":"ЗП-0917","internal_id":"ORD-0917"}`)
	b.add(catalog.ReferenceExternalIdMapped, msk("2026-01-02 00:00"), `{"system":"onec","object_kind":"order","external_id":"ЗП-0917","internal_id":"ORD-0917"}`)
	bk := b.build()
	if id, conflict, ok := bk.Resolve("onec", "order", "ЗП-0917"); !ok || conflict || id != "ORD-0917" {
		t.Fatalf("повтор соответствия — не конфликт: %s %v %v", id, conflict, ok)
	}
	b.add(catalog.ReferenceExternalIdMapped, msk("2026-01-03 00:00"), `{"system":"onec","object_kind":"order","external_id":"ЗП-0917","internal_id":"ORD-9999"}`)
	bk = b.build()
	id, conflict, ok := bk.Resolve("onec", "order", "ЗП-0917")
	if !ok || !conflict || id != "ORD-0917" {
		t.Fatalf("конфликт перезаписал соответствие: %s %v %v", id, conflict, ok)
	}
	if vs := bk.MappingsView("onec"); len(vs) != 2 || !vs[0].Conflict || !vs[1].Conflict || vs[1].Effective {
		t.Fatalf("строки соответствий: %+v", vs)
	}
}

// FR-81: смена-шаблон повторяется в рабочие дни; смена на момент — по месту и предкам.
func TestShifts(t *testing.T) {
	b := &book{t: t}
	b.add(catalog.ReferenceCalendarDefined, msk("2025-12-01 09:00"), cal2026)
	b.add(catalog.ReferenceLocationDefined, msk("2026-01-01 00:00"), `{"location_id":"WS-WC","kind":"workshop","scope":"ent01/b1/wc","name":"Сварочный цех","valid_from":"2025-12-31T21:00:00.000Z"}`)
	b.add(catalog.ReferenceLocationDefined, msk("2026-01-01 00:00"), `{"location_id":"ST-WELD","kind":"station","parent_id":"WS-WC","scope":"ent01/b1/wc/weld","name":"Участок сварки","valid_from":"2025-12-31T21:00:00.000Z"}`)
	b.add(catalog.ReferenceShiftScheduled, msk("2026-01-01 00:00"), `{"shift_id":"SHIFT-1","location_id":"WS-WC","name":"Первая смена",
		"starts_at":"2026-01-12T05:00:00.000Z","ends_at":"2026-01-12T13:30:00.000Z","repeat_until":"2026-12-31T20:59:00.000Z","working_days_only":true}`)
	b.add(catalog.ReferenceShiftScheduled, msk("2026-01-01 00:00"), `{"shift_id":"SHIFT-2","location_id":"WS-WC","name":"Вторая смена",
		"starts_at":"2026-01-12T13:30:00.000Z","ends_at":"2026-01-12T22:00:00.000Z","repeat_until":"2026-12-31T20:59:00.000Z","working_days_only":true}`)
	bk := b.build()
	s, ok := bk.ShiftAt(msk("2026-05-06 10:00"), "ST-WELD")
	if !ok || s.ID != "SHIFT-1@2026-05-06" || !s.From.Equal(msk("2026-05-06 08:00")) || !s.To.Equal(msk("2026-05-06 16:30")) {
		t.Fatalf("первая смена: %+v %v", s, ok)
	}
	// Вторая смена идёт за полночь.
	if s, ok := bk.ShiftAt(msk("2026-05-07 00:30"), ""); !ok || s.ID != "SHIFT-2@2026-05-06" {
		t.Fatalf("вторая смена за полночь: %+v %v", s, ok)
	}
	// Праздник 1 мая — смен нет; последняя смена — вторая 30 апреля.
	if _, ok := bk.ShiftAt(msk("2026-05-01 10:00"), ""); ok {
		t.Fatal("смена в праздник")
	}
	if s, ok := bk.ShiftBefore(msk("2026-05-01 10:00"), ""); !ok || s.ID != "SHIFT-2@2026-04-30" {
		t.Fatalf("последняя смена перед праздником: %+v %v", s, ok)
	}
	if n := len(bk.ShiftsBetween(msk("2026-05-04 00:00"), msk("2026-05-09 00:00"), "WS-WC")); n != 10 {
		t.Fatalf("смен за 4–8 мая: %d", n)
	}
}

// AD-31: версия с датой действия; позднее знание побеждает.
func TestItemTypeVersions(t *testing.T) {
	b := &book{t: t}
	b.add(catalog.ReferenceItemTypeDefined, msk("2026-01-01 00:00"), `{"item_type_id":"FL-100.00.000","designation":"ФЛ-100.00.000 СБ","name":"Фланец","revision":"А","valid_from":"2025-12-31T21:00:00.000Z"}`)
	b.add(catalog.ReferenceItemTypeDefined, msk("2026-01-01 00:00"), `{"item_type_id":"FL-100.00.000","designation":"ФЛ-100.00.000 СБ","name":"Фланец","revision":"Б","valid_from":"2026-06-30T21:00:00.000Z"}`)
	bk := b.build()
	if x, ok := bk.ItemTypeAt("FL-100.00.000", msk("2026-03-01 00:00")); !ok || x.Data.Revision != "А" {
		t.Fatalf("до даты действия: %+v", x.Data)
	}
	if x, ok := bk.ItemTypeAt("FL-100.00.000", msk("2026-07-01 00:00")); !ok || x.Data.Revision != "Б" {
		t.Fatalf("после даты действия: %+v", x.Data)
	}
	if n := len(bk.ItemTypesAt(msk("2026-07-01 00:00"))); n != 1 {
		t.Fatalf("номенклатура: %d", n)
	}
}

// 2027: суббота 20 февраля — рабочая (перенос на 22 февраля).
func TestCalendarWorkingSaturday(t *testing.T) {
	b := &book{t: t}
	b.add(catalog.ReferenceCalendarDefined, msk("2026-10-01 09:00"), `{"calendar_id":"CAL-ENT01","year":2027,"weekly_days_off":["sat","sun"],
		"non_working_days":["2027-02-22","2027-02-23"],"working_days":["2027-02-20"]}`)
	c := b.build().Calendar()
	if !c.Working(msk("2027-02-20 10:00")) || c.Working(msk("2027-02-21 10:00")) || c.Working(msk("2027-02-22 10:00")) {
		t.Fatal("перенос выходного")
	}
	if got := c.AddWorkingDays(msk("2027-02-19 10:00"), 2); !got.Equal(msk("2027-02-24 10:00")) {
		t.Fatalf("пт 19.02 + 2: %s", got.In(ref.Local))
	}
}
