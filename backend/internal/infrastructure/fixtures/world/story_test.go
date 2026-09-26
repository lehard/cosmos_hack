package world

import (
	"io/fs"
	"testing"

	simapp "ant/internal/application/simulation"
)

func readBpmn() ([]byte, error) {
	return fs.ReadFile(repoFS(), "normative/process/flange-process.bpmn")
}

func buildMain(t *testing.T) *Model {
	t.Helper()
	pol, err := LoadPolicy(repoFS())
	if err != nil {
		t.Fatal(err)
	}
	specs, err := Worlds(repoFS())
	if err != nil {
		t.Fatal(err)
	}
	bpmn, err := readBpmn()
	if err != nil {
		t.Fatal(err)
	}
	m, err := Build(specs[0], pol, bpmn)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// TestMainStory — сверка мира с процессной сессией: область 34 → 13 → 6, итоги
// главной истории (scenarios-flange-expected.yaml, main_story_totals) — табло
// «ожидалось → получилось» на последнем шаге целиком зелёное.
func TestMainStory(t *testing.T) {
	m := buildMain(t)
	in := m.Incident("RS-01")
	if in == nil {
		t.Fatal("нет RS-01")
	}
	for i, want := range []int{34, 13, 6, 6} {
		if got := in.Versions[i].Size(); got != want {
			t.Errorf("RS-01 v%d: %d, ожидалось %d", i+1, got, want)
		}
	}
	if got := in.Versions[3].Count("excluded"); got != 28 {
		t.Errorf("RS-01 исключено: %d, ожидалось 28", got)
	}
	last := len(m.Steps) - 1
	c := m.ctx(last)
	b := renderSimulation(c)[0].Body
	for _, r := range b.(simapp.Board).Rows {
		if r.Status != "passed" {
			t.Errorf("табло %s «%s»: %s, получено %v, ожидалось %s", r.AssertionID, r.Title, r.Status, deref(r.Actual), r.Expected)
		}
	}
	// Где 34 детали в Ср 11:10 (S05-03): 21 — участок сварки, 8 — кладовая, 4 — сборка, 1 — стенд.
	at := m.clk.at(23, 11, 10)
	where := map[string]int{}
	for id := range in.Versions[0].Status {
		st := m.itemByID[id].State(at)
		switch {
		case st.Location == "WH-WC":
			where["storage"]++
		case st.Step == "testing.leak_test":
			where["leak"]++
		case len(st.Step) > 9 && st.Step[:9] == "assembly.":
			where["assembly"]++
		default:
			where["welding"]++
		}
	}
	if where["welding"] != 21 || where["storage"] != 8 || where["assembly"] != 4 || where["leak"] != 1 {
		t.Errorf("положение 34 изделий в Ср 11:10: %v", where)
	}
}

func deref(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}
