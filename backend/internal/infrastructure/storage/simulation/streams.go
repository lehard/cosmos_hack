package simulation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sim "ant/internal/domain/simulation"
)

// Потоки прогона по умолчанию (AD-26: «потоки событий — JSONL»):
// scenarios/definitions/streams/‹прогон›.events.jsonl — исходные события
// источников в порядке доставки (строка — одно событие с моментом и видом
// доставки); ‹прогон›.steps.jsonl — шаги людей и служебные шаги (их сверяет
// demo-signer: подписывает только совпадающие с шагом запросы); ‹прогон›.truth.json
// — «истина» генератора и счётчики шума. Файлы строятся генератором из
// определений (make sim-streams) и в сборке сверяются с ним: руками не правятся.

// GoldenRunID — run_id потоков по умолчанию: ‹прогон›-‹seed› (AD-38).
func GoldenRunID(run string, seed int64) string {
	return strings.ToLower(run) + "-" + fmt.Sprint(seed)
}

type eventLine struct {
	DeliverAt string          `json:"deliver_at"`
	Delivery  sim.Delivery    `json:"delivery"`
	SourceKey string          `json:"source_key"`
	Scenario  string          `json:"scenario"`
	Label     string          `json:"label,omitempty"`
	Contract  bool            `json:"contract"`
	Event     json.RawMessage `json:"event"`
}

// Render — содержимое файлов потоков плана: имя файла → байты.
func Render(p *sim.Plan) (map[string][]byte, error) {
	var ev, st bytes.Buffer
	for _, e := range p.Emissions {
		line := eventLine{DeliverAt: sim.FormatTime(e.DeliverAt), Delivery: e.Delivery, SourceKey: e.SourceKey,
			Scenario: e.Scenario, Label: e.Label, Contract: e.Contract, Event: e.Event}
		b, err := json.Marshal(line)
		if err != nil {
			return nil, err
		}
		ev.Write(b)
		ev.WriteByte('\n')
	}
	for _, a := range p.Actions {
		b, err := json.Marshal(a)
		if err != nil {
			return nil, err
		}
		st.Write(b)
		st.WriteByte('\n')
	}
	tr, err := json.MarshalIndent(struct {
		RunID string    `json:"run_id"`
		Run   string    `json:"run"`
		Seed  int64     `json:"seed"`
		Start string    `json:"start"`
		End   string    `json:"end"`
		Truth sim.Truth `json:"truth"`
	}{p.RunID, p.RunDef, p.Seed, sim.FormatTime(p.Start), sim.FormatTime(p.End), p.Truth}, "", " ")
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		p.RunDef + ".events.jsonl": ev.Bytes(),
		p.RunDef + ".steps.jsonl":  st.Bytes(),
		p.RunDef + ".truth.json":   append(tr, '\n'),
	}, nil
}

// WriteStreams пишет потоки в каталог dir.
func WriteStreams(dir string, files map[string][]byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, b := range files {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}
