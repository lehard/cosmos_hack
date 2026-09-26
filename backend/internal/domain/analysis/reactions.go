package analysis

import "time"

// TimeLayout — время в данных записей: RFC 3339 UTC, ровно три знака после
// секунд («Соглашения/Время»).
const TimeLayout = "2006-01-02T15:04:05.000Z"

// ts — время по соглашению.
func ts(t time.Time) string { return t.UTC().Format(TimeLayout) }

// hypothesisData — гипотеза в data incident.hypothesis.computed.
type hypothesisData struct {
	Category      string   `json:"category"`
	ConfidenceBP  *int     `json:"confidence_bp,omitempty"`
	Supporting    []string `json:"supporting_event_ids"`
	Contradicting []string `json:"contradicting_event_ids,omitempty"`
}

// hypothesisComputedData — data incident.hypothesis.computed v1 (FR-58, FR-59).
type hypothesisComputedData struct {
	NcID              string           `json:"nc_id"`
	CausalWindowStart string           `json:"causal_window_start,omitempty"`
	CausalWindowEnd   string           `json:"causal_window_end,omitempty"`
	Hypotheses        []hypothesisData `json:"hypotheses"`
	Missing           []string         `json:"missing_information"`
	Categorical       bool             `json:"conclusion_is_categorical"`
}

// hypothesisComputed — версия вывода разбора для журнала (AD-3).
func hypothesisComputed(a Analysis) hypothesisComputedData {
	d := hypothesisComputedData{NcID: a.NCID, Hypotheses: []hypothesisData{}, Missing: append([]string{}, a.Missing...), Categorical: a.Categorical}
	if a.Window != nil {
		d.CausalWindowStart, d.CausalWindowEnd = ts(a.Window.Start), ts(a.Window.End)
	}
	for _, h := range a.Hypotheses {
		x := hypothesisData{Category: h.Category, ConfidenceBP: h.ConfidenceBP, Supporting: uuidsOnly(h.Supporting)}
		if c := uuidsOnly(h.Contradicting); len(c) > 0 {
			x.Contradicting = c
		}
		d.Hypotheses = append(d.Hypotheses, x)
	}
	return d
}

// uuidsOnly — только идентификаторы записей вида UUID (схема supporting_event_ids).
func uuidsOnly(xs []string) []string {
	out := []string{}
	for _, x := range xs {
		if isUUID(x) {
			out = append(out, x)
		}
	}
	return out
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return false
			}
		case (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'):
		default:
			return false
		}
	}
	return true
}
