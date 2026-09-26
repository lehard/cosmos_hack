package upcast

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// Повышатель, сгенерированный из таблицы contracts/events/upcasters, даёт ровно
// пример contracts/events/examples/versions/…v1-upcast-to-v2.json (FR-110).
func TestEquipmentStateChangedV1ToV2MatchesExample(t *testing.T) {
	load := func(name string) map[string]any {
		b, err := os.ReadFile("../../../../contracts/events/examples/versions/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var env struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(b, &env); err != nil {
			t.Fatal(err)
		}
		return env.Data
	}
	got, err := Upcast("equipment.state.changed", 1, 2, load("equipment.state.changed.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := load("equipment.state.changed.v1-upcast-to-v2.json"); !reflect.DeepEqual(got, want) {
		t.Fatalf("повышение v1 → v2 расходится с примером:\n got %v\nwant %v", got, want)
	}
	if _, err := Upcast("equipment.state.changed", 2, 3, got); err == nil {
		t.Fatal("нет повышателя v2 → v3 — ждали ошибку (карантин ingest.unknown_schema_version)")
	}
}
