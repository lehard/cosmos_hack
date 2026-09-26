package loader

import (
	"encoding/json"
	"strings"
)

// AD-38: локальные ID изделий, партий, носителей и прочих объектов сценария
// получают префикс прогона. В заготовках они хранятся без префикса; адаптер
// берёт run_id из курсора и вставляет «‹run_id›/» перед локальной частью:
// F-017 → run-1/F-017, ENT01:F-017 → ENT01:run-1/F-017,
// item:ENT01:F-017 → item:ENT01:run-1/F-017. Входные параметры — обратно.

// applyRun вставляет префикс прогона в строки тела, совпадающие с локальными ID.
func (s *Scenario) applyRun(body []byte, runID string) ([]byte, error) {
	if runID == "" || s.localIDs == nil {
		return body, nil
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, err
	}
	v = s.walkRun(v, runID)
	return json.Marshal(v)
}

func (s *Scenario) walkRun(v any, runID string) any {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			x[k] = s.walkRun(e, runID)
		}
		return x
	case []any:
		for i, e := range x {
			x[i] = s.walkRun(e, runID)
		}
		return x
	case string:
		return s.PrefixID(x, runID)
	default:
		return v
	}
}

// PrefixID добавляет префикс прогона к локальной части id (последнему сегменту
// после «:»), если она — локальный ID сценария.
func (s *Scenario) PrefixID(id, runID string) string {
	if runID == "" || s.localIDs == nil {
		return id
	}
	head, tail := "", id
	if i := strings.LastIndexByte(id, ':'); i >= 0 {
		head, tail = id[:i+1], id[i+1:]
	}
	if !s.localIDs.MatchString(tail) {
		return id
	}
	return head + runID + "/" + tail
}

// StripRun снимает префикс прогона с id из входных параметров.
func StripRun(id, runID string) string {
	if runID == "" {
		return id
	}
	return strings.Replace(id, runID+"/", "", 1)
}
