package world

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"
	"testing/fstest"
	"time"

	"go.yaml.in/yaml/v3"

	"ant/internal/infrastructure/fixtures/loader"
)

// FixturesDir — каталог заготовок от корня репозитория (AD-36).
const FixturesDir = "scenarios/fixtures"

// InputFiles — входы генератора от корня репозитория (кроме world.yaml сценариев):
// их копия встроена в бинарник (input/), чтобы мир строился без каталога репозитория.
var InputFiles = []string{"normative/policy/policy.v1.yaml", "normative/process/flange-process.bpmn", "normative/reference/flange/locations.yaml", ShiftsFile, ItemTypesFile, EquipmentFile,
	"normative/documents/templates.v1.yaml"}

// InputGlobs — входы генератора по шаблону.
var InputGlobs = []string{"normative/desks/*.yaml", FixturesDir + "/*/world.yaml", FederationDir + "/*.json"}

//go:embed all:input
var input embed.FS

// Inputs — встроенная копия входов генератора от корня репозитория
// (normative/policy, normative/desks, …; совпадение с репозиторием проверяет
// тест генератора). Её же читает вход демо-трека эпика 08 в cmd/ant (каталог
// политики и столы ролей), пока стартовая политика не приходит из журнала (эпик 05).
func Inputs() fs.FS {
	sub, err := fs.Sub(input, "input")
	if err != nil {
		panic(err) // каталог input встроен всегда
	}
	return sub
}

func init() {
	loader.Builtin = func() (*loader.Library, error) {
		sub, err := fs.Sub(input, "input")
		if err != nil {
			return nil, err
		}
		return LibraryFrom(sub)
	}
	loader.Holders = func(action string) []string {
		holdersOnce.Do(func() { holdersPol, _ = LoadPolicy(Inputs()) })
		if holdersPol == nil {
			return nil
		}
		return holdersPol.Holders(action)
	}
}

// holdersPol — стартовая политика для loader.Holders (читается один раз).
var (
	holdersOnce sync.Once
	holdersPol  *Policy
)

// LibraryFrom строит библиотеку заготовок в памяти из входов (fsys — корень репозитория).
func LibraryFrom(fsys fs.FS) (*loader.Library, error) {
	files, err := Generate(fsys)
	if err != nil {
		return nil, err
	}
	mfs := fstest.MapFS{}
	for name, b := range files {
		mfs[name] = &fstest.MapFile{Data: b}
	}
	return loader.LoadFS(mfs, ".")
}

// Worlds — описания сценариев scenarios/fixtures/*/world.yaml.
func Worlds(fsys fs.FS) ([]*Spec, error) {
	names, err := fs.Glob(fsys, FixturesDir+"/*/world.yaml")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	var out []*Spec
	for _, n := range names {
		b, err := fs.ReadFile(fsys, n)
		if err != nil {
			return nil, err
		}
		s := &Spec{}
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(s); err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		if s.ID != path.Base(path.Dir(n)) {
			return nil, fmt.Errorf("%s: id %q не совпадает с каталогом", n, s.ID)
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("нет %s/*/world.yaml", FixturesDir)
	}
	return out, nil
}

// Generate строит файлы заготовок (пути от scenarios/fixtures): common/, ‹сценарий›/scenario.yaml, steps/NN.yaml.
func Generate(fsys fs.FS) (map[string][]byte, error) {
	pol, err := LoadPolicy(fsys)
	if err != nil {
		return nil, err
	}
	bpmn, err := fs.ReadFile(fsys, "normative/process/flange-process.bpmn")
	if err != nil {
		return nil, err
	}
	specs, err := Worlds(fsys)
	if err != nil {
		return nil, err
	}
	shifts, err := LoadShifts(fsys)
	if err != nil {
		return nil, err
	}
	names, err := LoadNames(fsys)
	if err != nil {
		return nil, err
	}
	templates, err := LoadTemplates(fsys)
	if err != nil {
		return nil, err
	}
	fed, err := LoadFederation(fsys)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{"common/blobs/" + BpmnBlob: bpmn, "common/blobs/" + BracketBpmnBlob: BracketBpmn}
	var people []PersonRef
	seen := map[string]bool{}
	for _, s := range specs {
		for _, p := range s.People {
			if !seen[p.Person] {
				seen[p.Person] = true
				people = append(people, p)
			}
		}
		m, err := Build(s, pol, bpmn)
		if err != nil {
			return nil, err
		}
		m.shifts = shifts
		m.names = names
		m.templates = templates
		m.federation = fed
		if err := m.write(out); err != nil {
			return nil, err
		}
	}
	common, err := Common(pol, people)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "# СГЕНЕРИРОВАНО генератором мира заготовок (backend/internal/infrastructure/fixtures/world) — руками не править (AD-36).\n")
	fmt.Fprintf(&b, "# Общие ответы всех сценариев: демо-персоны, сеансы и столы ролей (normative/policy, normative/desks).\n")
	fmt.Fprintf(&b, "format: %d\ndefault_scenario: %s\n", loader.FormatVersion, specs[0].ID)
	writeResponses(&b, common)
	out["common/responses.yaml"] = b.Bytes()
	return out, nil
}

func jsonLine(v any) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(err)
	}
	return strings.TrimRight(b.String(), "\n")
}

func writeResponses(b *bytes.Buffer, rs []loader.Response) {
	if len(rs) == 0 {
		return
	}
	b.WriteString("responses:\n")
	for _, r := range rs {
		fmt.Fprintf(b, "  - op: %s\n", r.Op)
		if len(r.Params) > 0 {
			fmt.Fprintf(b, "    params: %s\n", jsonLine(r.Params))
		}
		fmt.Fprintf(b, "    body: %s\n", jsonLine(r.Body))
	}
}

func utc(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05Z") }

// write — scenario.yaml и накопительные шаги: в файл шага идут только ответы,
// изменившиеся с прошлого шага (сравнение тел JSON).
func (m *Model) write(out map[string][]byte) error {
	dir := m.Spec.ID + "/"
	man := loader.Manifest{Format: loader.FormatVersion, ID: m.Spec.ID, Title: m.Spec.Title, Description: m.Spec.Description, Covers: m.Spec.Covers, Case: m.Spec.Case,
		InitialStep: m.Spec.InitialStep, StartStep: m.Spec.StartStep, Enterprise: enterprise, LocalIDs: m.Spec.LocalIDs}
	for i, s := range m.Spec.Steps {
		h := loader.StepHeader{Step: i, Clock: m.Steps[i], Title: s.Title, Scenarios: s.Scenarios}
		if s.Wait != nil {
			h.Wait = &loader.Wait{Action: s.Wait.Action, Role: s.Wait.Role, Object: loader.ObjectRef{Kind: s.Wait.Object.Kind, ID: s.Wait.Object.ID}, Title: s.Wait.Title}
		}
		man.Steps = append(man.Steps, h)
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "# СГЕНЕРИРОВАНО из world.yaml генератором мира заготовок — руками не править (AD-36).\n")
	fmt.Fprintf(&b, "format: %d\nid: %s\ntitle: %s\ndescription: %s\ncovers: %s\ncase: %s\ninitial_step: %d\nstart_step: %d\nenterprise: %s\nlocal_ids: %s\nsteps:\n",
		man.Format, man.ID, jsonLine(man.Title), jsonLine(man.Description), jsonLine(man.Covers), jsonLine(man.Case), man.InitialStep, man.StartStep, man.Enterprise, jsonLine(man.LocalIDs))
	for _, h := range man.Steps {
		fmt.Fprintf(&b, "  - {step: %d, clock: %s, title: %s", h.Step, utc(h.Clock), jsonLine(h.Title))
		if len(h.Scenarios) > 0 {
			fmt.Fprintf(&b, ", scenarios: %s", jsonLine(h.Scenarios))
		}
		if h.Wait != nil {
			fmt.Fprintf(&b, ", wait: %s", jsonLine(h.Wait))
		}
		b.WriteString("}\n")
	}
	out[dir+"scenario.yaml"] = b.Bytes()
	prev := map[string]string{}
	for i, h := range man.Steps {
		var sb bytes.Buffer
		fmt.Fprintf(&sb, "# СГЕНЕРИРОВАНО из world.yaml — руками не править (AD-36). Шаг накопительный: здесь только ответы, изменившиеся с прошлого шага.\n")
		fmt.Fprintf(&sb, "format: %d\nstep: %d\nclock: %s\ntitle: %s\n", loader.FormatVersion, i, utc(h.Clock), jsonLine(h.Title))
		if len(h.Scenarios) > 0 {
			fmt.Fprintf(&sb, "scenarios: %s\n", jsonLine(h.Scenarios))
		}
		if h.Wait != nil {
			fmt.Fprintf(&sb, "wait: %s\n", jsonLine(h.Wait))
		}
		if ch := m.Changes(i); len(ch) > 0 {
			sb.WriteString("changes:\n")
			for _, c := range ch {
				fmt.Fprintf(&sb, "  - %s\n", jsonLine(c))
			}
		}
		var changed []loader.Response
		seen := map[string]bool{}
		for _, r := range m.Render(i) {
			k := r.Key()
			if seen[k] {
				return fmt.Errorf("%s шаг %d: ответ %s дважды", m.Spec.ID, i, k)
			}
			seen[k] = true
			body := jsonLine(r.Body)
			if prev[k] != body {
				prev[k] = body
				changed = append(changed, r)
			}
		}
		writeResponses(&sb, changed)
		out[fmt.Sprintf("%ssteps/%02d.yaml", dir, i)] = sb.Bytes()
	}
	return nil
}
