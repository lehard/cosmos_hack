package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	simapp "ant/internal/application/simulation"
	sim "ant/internal/domain/simulation"
)

// StepRequest — запрос шага (contracts/internal/demo-signer.openapi.yaml).
type StepRequest struct {
	RunID      string `json:"run_id"`
	ScenarioID string `json:"scenario_id"`
	StepID     string `json:"step_id"`
	PersonaID  string `json:"persona_id"`
	// Request — что ant просит подписать и отправить: операция, параметры
	// пути и тело с подставленными значениями прогона.
	Request *Request `json:"request,omitempty"`
}

// Request — запрос операции API.
type Request struct {
	Operation string            `json:"operation"`
	Params    map[string]string `json:"params,omitempty"`
	Body      map[string]any    `json:"body,omitempty"`
}

// StepResult — итог шага.
type StepResult struct {
	Status    string `json:"status"` // signed | attested | skipped | mismatch
	CommandID string `json:"command_id,omitempty"`
	DocDigest string `json:"doc_digest,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// Steps — шаги сценариев из scenarios/definitions (том только для чтения).
type Steps struct {
	Root string
}

// ErrMismatch — запрос ant не совпал с шагом сценария: не подписывается.
var ErrMismatch = errors.New("запрос не совпадает с шагом сценария")

// Find — шаг решения сценария: по метке (label) или по «‹операция›#‹n›» —
// n-е решение этой операции в карточке (с 1).
func (s Steps) Find(ctx context.Context, scenarioID, stepID string) (sim.Step, error) {
	defs := simapp.FSDefinitions{FS: os.DirFS(s.Root)}
	cat, err := defs.Catalog(ctx)
	if err != nil {
		return sim.Step{}, err
	}
	run := scenarioID
	for _, c := range cat.Entries {
		if c.ID == scenarioID && c.Run != "" {
			run = c.Run
		}
	}
	b, err := defs.Bundle(ctx, run)
	if err != nil {
		return sim.Step{}, err
	}
	op, nth, hasNth := strings.Cut(stepID, "#")
	for _, sc := range b.Scenarios {
		if sc.ID != scenarioID && scenarioID != run {
			continue
		}
		n := 0
		for _, st := range sc.Steps {
			if st.Decision == "" {
				continue
			}
			if v, ok := b.World.Aliases.People[st.Actor]; ok && v != "" {
				st.Actor = v // персона контракта после сведения имён (worlds/‹мир›.yaml)
			}
			if st.Label != "" && st.Label == stepID {
				return st, nil
			}
			if hasNth && st.Decision == op {
				n++
				if fmt.Sprint(n) == nth {
					return st, nil
				}
			}
		}
	}
	return sim.Step{}, fmt.Errorf("%w: шага %s в сценарии %s нет", ErrMismatch, stepID, scenarioID)
}

// Match — AD-26: подписывается только запрос, совпадающий с шагом: та же
// операция, та же персона, те же буквальные значения параметров и тела;
// плейсхолдеры определения ({item:…}, {ref:…}) принимают значение прогона.
func Match(st sim.Step, persona string, rq Request) error {
	if st.Decision != rq.Operation {
		return fmt.Errorf("%w: операция %s, в шаге %s", ErrMismatch, rq.Operation, st.Decision)
	}
	if st.Actor != "" && st.Actor != persona {
		return fmt.Errorf("%w: персона %s, в шаге %s", ErrMismatch, persona, st.Actor)
	}
	params := map[string]any{}
	for k, v := range rq.Params {
		params[k] = v
	}
	if err := literal("params", st.Params, params); err != nil {
		return err
	}
	return literal("body", st.Body, rq.Body)
}

// literal — буквальные значения шаблона want есть в got (лишние поля got —
// заголовок команды и подставленные значения — допускаются).
func literal(path string, want, got any) error {
	switch w := want.(type) {
	case nil:
		return nil
	case string:
		if strings.Contains(w, "{") && strings.Contains(w, "}") {
			return nil // плейсхолдер: значение прогона
		}
		if g, ok := got.(string); ok && g == w {
			return nil
		}
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			break
		}
		keys := make([]string, 0, len(w))
		for k := range w {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if err := literal(path+"."+k, w[k], g[k]); err != nil {
				return err
			}
		}
		return nil
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			break
		}
		for i := range w {
			if err := literal(fmt.Sprintf("%s[%d]", path, i), w[i], g[i]); err != nil {
				return err
			}
		}
		return nil
	default:
		if equalJSON(w, got) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s = %v, в шаге %v", ErrMismatch, path, got, want)
}

func equalJSON(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	var xa, ya any
	_ = json.Unmarshal(x, &xa)
	_ = json.Unmarshal(y, &ya)
	return reflect.DeepEqual(xa, ya)
}
