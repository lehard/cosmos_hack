package nonconformity_test

import (
	"testing"
	"time"

	app "ant/internal/application/nonconformity"
)

// Окно операции участка: без остановки — «Остановить точку» доступна; при
// остановке — «Снять» у обладателя полномочия, «Остановить» — нет (интерфейс 6).
func TestStationActions(t *testing.T) {
	a := app.StationActions("welding.weld", "Сварка", "IS-2", nil, false)
	if len(a) != 1 || a[0].Operation != "nonconformity.process_hold.set" || !a[0].Allowed || a[0].EquipmentID != "IS-2" {
		t.Fatalf("без остановки: %+v", a)
	}
	holds := []app.StationHold{{HoldID: "HOLD-IS-2-0923", Reason: "НС-01", Since: time.Now(), Level: "process_point_stop", ReleaseCondition: "ремонт"}}
	a = app.StationActions("welding.weld", "Сварка", "IS-2", holds, false)
	if len(a) != 2 || a[0].Allowed || a[1].Allowed || a[1].HoldID != "HOLD-IS-2-0923" {
		t.Fatalf("остановлено, без полномочия: %+v", a)
	}
	if a = app.StationActions("welding.weld", "Сварка", "IS-2", holds, true); !a[1].Allowed {
		t.Fatalf("с полномочием: %+v", a)
	}
}
