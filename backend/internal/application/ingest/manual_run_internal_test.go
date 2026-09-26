package ingest

import "testing"

func TestRunOfItem(t *testing.T) {
	for id, want := range map[string]string{
		"ENT01:show-is2-20260921-1/I-9DE0BA5E": "show-is2-20260921-1",
		"ENT01:F-017":                          "",
		"F-017":                                "",
		"ENT01:/I-1":                           "",
	} {
		if got := runOfItem(id); got != want {
			t.Errorf("runOfItem(%q) = %q, ждали %q", id, got, want)
		}
	}
}
