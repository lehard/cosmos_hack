package item

import "testing"

// Метка изделия для людей (кейс §4.6): номер детали, а не внутренний id и не
// «DM:F-001»; DM-код без номера — как есть.
func TestLabels(t *testing.T) {
	for id, want := range map[string]string{
		"ENT01:show-is2-20260921/F-001": "Ф-001",
		"ENT01:DM:F-002":                "Ф-002",
		"DM:F-003":                      "Ф-003",
		"ENT01:R-004":                   "К-004",
		"ENT01:I-1B7F8B19":              "I-1B7F8B19",
	} {
		if got := LocalLabel(id); got != want {
			t.Errorf("LocalLabel(%q) = %q, ждали %q", id, got, want)
		}
	}
	dm := func(v string) State {
		return State{Carriers: []Carrier{{Type: "dpm_datamatrix", Value: v}}}
	}
	if got := dm("show-is2-20260921/DM:F-001").DisplayLabel("ENT01:show-is2-20260921-1/I-3CDF7159"); got != "Ф-001" {
		t.Errorf("DM с номером: %q", got)
	}
	if got := dm("DM:7Q2X").DisplayLabel("ENT01:I-1"); got != "DM:7Q2X" {
		t.Errorf("DM без номера: %q", got)
	}
}
