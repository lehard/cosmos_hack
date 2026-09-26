package loader

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// FormatVersion — версия формата файлов заготовок.
const FormatVersion = 1

// Manifest — scenarios/fixtures/‹сценарий›/scenario.yaml: сценарий, его шаги и
// правила префикса прогона (AD-36, AD-38).
type Manifest struct {
	Format      int      `yaml:"format" json:"format"`
	ID          string   `yaml:"id" json:"id"`
	Title       string   `yaml:"title" json:"title"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Covers      []string `yaml:"covers,omitempty" json:"covers,omitempty"`
	Case        []string `yaml:"case,omitempty" json:"case,omitempty"`
	// InitialStep — шаг курсора, пока пульт не запускал прогон.
	InitialStep int `yaml:"initial_step" json:"initial_step"`
	// Enterprise — код предприятия в ID изделий (код:локальный_id, AD-16).
	Enterprise string `yaml:"enterprise" json:"enterprise"`
	// LocalIDs — регулярные выражения локальных ID сценария (изделия, партии,
	// носители, несоответствия, инциденты…): им адаптер добавляет префикс прогона.
	LocalIDs []string `yaml:"local_ids" json:"local_ids"`
	// Steps — оглавление шагов (номер, часы, заголовок, ожидание решения).
	Steps []StepHeader `yaml:"steps" json:"steps"`
}

// StepHeader — заголовок шага сценария.
type StepHeader struct {
	Step int `yaml:"step" json:"step"`
	// Clock — доменное «сейчас» шага (виртуальные часы сценария, AD-37), RFC 3339 UTC.
	Clock time.Time `yaml:"clock" json:"clock"`
	Title string    `yaml:"title" json:"title"`
	// Scenarios — какие сценарии процессной сессии (S01…S13) шаг продвигает.
	Scenarios []string `yaml:"scenarios,omitempty" json:"scenarios,omitempty"`
	// Wait — на этом шаге сценарий ждёт решения человека (FR-129).
	Wait *Wait `yaml:"wait,omitempty" json:"wait,omitempty"`
}

// Wait — ожидание решения человека на столе роли (FR-129): пока решение не
// принято, пульт не продвигает сценарий сам; принятое решение (команда action
// над объектом) двигает курсор на следующий шаг.
type Wait struct {
	Action string    `yaml:"action" json:"action"`
	Role   string    `yaml:"role" json:"role"`
	Object ObjectRef `yaml:"object" json:"object"`
	Title  string    `yaml:"title" json:"title"`
}

// ObjectRef — объект ожидания или изменения: вид сущности и id (без префикса прогона).
type ObjectRef struct {
	Kind string `yaml:"kind" json:"kind"`
	ID   string `yaml:"id" json:"id"`
}

// Change — изменение сущности на шаге — сообщение SSE entity_changed (AD-21).
type Change struct {
	Entity string `yaml:"entity" json:"entity"`
	ID     string `yaml:"id" json:"id"`
}

// StepFile — scenarios/fixtures/‹сценарий›/steps/NN.yaml: ответы операций,
// изменившиеся на этом шаге (шаги накопительные: ответ, не изменившийся с
// прошлого шага, берётся из прошлого).
type StepFile struct {
	StepHeader `yaml:",inline"`
	Format     int        `yaml:"format" json:"format"`
	Changes    []Change   `yaml:"changes,omitempty" json:"changes,omitempty"`
	Responses  []Response `yaml:"responses,omitempty" json:"responses,omitempty"`
}

// Response — ответ операции operationId при параметрах Params. Params —
// подмножество параметров вызова: из подходящих берётся самый точный (больше
// совпавших параметров). Status ≥ 400 — отказ с кодом Code (problem+json).
type Response struct {
	Op     string            `yaml:"op" json:"op"`
	Params map[string]string `yaml:"params,omitempty" json:"params,omitempty"`
	Status int               `yaml:"status,omitempty" json:"status,omitempty"`
	Code   string            `yaml:"code,omitempty" json:"code,omitempty"`
	// Body — тело ответа по схеме операции в contracts/openapi.yaml (make check).
	Body any `yaml:"body,omitempty" json:"body,omitempty"`
}

// Common — scenarios/fixtures/common/responses.yaml: ответы, общие для всех
// сценариев и шагов (демо-персоны, сеансы, столы ролей); ответ сценария с тем
// же ключом их перекрывает.
type Common struct {
	Format    int        `yaml:"format" json:"format"`
	Default   string     `yaml:"default_scenario" json:"default_scenario"`
	Responses []Response `yaml:"responses" json:"responses"`
}

// Key — ключ ответа: операция и канонические параметры.
func (r Response) Key() string { return r.Op + "?" + canonParams(r.Params) }

// BodyJSON — тело ответа в JSON.
func (r Response) BodyJSON() ([]byte, error) {
	if r.Body == nil {
		return []byte("null"), nil
	}
	return json.Marshal(normalizeYAML(r.Body))
}

// normalizeYAML приводит значения, прочитанные из YAML (map[string]any,
// map[any]any), к виду, который принимает encoding/json.
func normalizeYAML(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[k] = normalizeYAML(e)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[fmt.Sprint(k)] = normalizeYAML(e)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = normalizeYAML(e)
		}
		return out
	case time.Time:
		return x.UTC().Format("2006-01-02T15:04:05.000Z")
	default:
		return v
	}
}

func (m *Manifest) validate() error {
	if m.Format != FormatVersion {
		return fmt.Errorf("сценарий %s: формат %d, ожидается %d", m.ID, m.Format, FormatVersion)
	}
	if m.ID == "" || len(m.Steps) == 0 {
		return fmt.Errorf("сценарий %q: нет id или шагов", m.ID)
	}
	for i, s := range m.Steps {
		if s.Step != i {
			return fmt.Errorf("сценарий %s: шаг %d на месте %d", m.ID, s.Step, i)
		}
		if i > 0 && s.Clock.Before(m.Steps[i-1].Clock) {
			return fmt.Errorf("сценарий %s: часы шага %d раньше шага %d (AD-37)", m.ID, i, i-1)
		}
	}
	if m.InitialStep < 0 || m.InitialStep >= len(m.Steps) {
		return fmt.Errorf("сценарий %s: initial_step %d вне шагов", m.ID, m.InitialStep)
	}
	for _, p := range m.LocalIDs {
		if _, err := regexp.Compile("^(?:" + p + ")$"); err != nil {
			return fmt.Errorf("сценарий %s: local_ids %q: %w", m.ID, p, err)
		}
	}
	return nil
}
