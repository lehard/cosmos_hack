package notifications

import (
	"testing"

	"ant/internal/application/platform"
)

// Задача от действия человека со стола (факт без run_id) — в прогоне своего
// изделия (id ‹ENT›:‹run_id›/…), чужого прогона — нет.
func TestInRunItem(t *testing.T) {
	m := platform.Moment{RunID: "show-is2-20260921-1"}
	cases := []struct {
		run, item string
		want      bool
	}{
		{"show-is2-20260921-1", "", true},
		{"", "ENT01:show-is2-20260921-1/I-9DE0BA5E", true},
		{"", "ENT01:show-is2-20260921-2/I-9DE0BA5E", false},
		{"", "ENT01:F-017", false},
		{"other", "ENT01:show-is2-20260921-1/I-9DE0BA5E", false},
	}
	for _, c := range cases {
		if got := inRunItem(m, c.run, c.item); got != c.want {
			t.Errorf("inRunItem(%q, %q) = %v", c.run, c.item, got)
		}
	}
	if !inRunItem(platform.Moment{}, "", "ENT01:F-017") {
		t.Error("без прогона в запросе — все")
	}
}
