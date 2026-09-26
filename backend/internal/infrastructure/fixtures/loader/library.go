package loader

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Library — все сценарии заготовок с разрешёнными (накопленными) ответами по шагам.
type Library struct {
	scenarios map[string]*Scenario
	order     []string
	common    map[string]*entry
	commonOps map[string][]*entry
	// blobs — большие неизменные строки (BPMN XML): в теле ответа строка
	// "@blob:‹имя›" заменяется содержимым common/blobs/‹имя› при ответе.
	blobs map[string][]byte
	// Default — сценарий курсора, пока пульт не запускал прогон.
	Default string
}

// Scenario — сценарий: оглавление и накопленные ответы каждого шага.
type Scenario struct {
	Manifest Manifest
	steps    []stepState
	localIDs *regexp.Regexp
}

type stepState struct {
	file  StepFile
	byKey map[string]*entry
	byOp  map[string][]*entry
}

type entry struct {
	resp Response
	body []byte
}

// LoadFS читает каталог заготовок root в fsys: common/responses.yaml и
// ‹сценарий›/scenario.yaml + steps/NN.yaml для каждого подкаталога со scenario.yaml.
func LoadFS(fsys fs.FS, root string) (*Library, error) {
	lib := &Library{scenarios: map[string]*Scenario{}, common: map[string]*entry{}, commonOps: map[string][]*entry{}, blobs: map[string][]byte{}}
	if blobs, err := fs.ReadDir(fsys, path.Join(root, "common", "blobs")); err == nil {
		for _, b := range blobs {
			if b.IsDir() {
				continue
			}
			data, err := fs.ReadFile(fsys, path.Join(root, "common", "blobs", b.Name()))
			if err != nil {
				return nil, err
			}
			lib.blobs[b.Name()] = data
		}
	}
	var common Common
	if err := readYAML(fsys, path.Join(root, "common", "responses.yaml"), &common); err == nil {
		if common.Format != FormatVersion {
			return nil, fmt.Errorf("common/responses.yaml: формат %d, ожидается %d", common.Format, FormatVersion)
		}
		lib.Default = common.Default
		for _, r := range common.Responses {
			e, err := newEntry(r)
			if err != nil {
				return nil, fmt.Errorf("common/responses.yaml: %w", err)
			}
			lib.common[r.Key()] = e
			lib.commonOps[r.Op] = append(lib.commonOps[r.Op], e)
		}
	} else if !isNotExist(err) {
		return nil, err
	}
	dirs, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("каталог заготовок %s: %w", root, err)
	}
	for _, d := range dirs {
		if !d.IsDir() || d.Name() == "common" {
			continue
		}
		dir := path.Join(root, d.Name())
		if _, err := fs.Stat(fsys, path.Join(dir, "scenario.yaml")); err != nil {
			continue
		}
		sc, err := loadScenario(fsys, dir)
		if err != nil {
			return nil, err
		}
		if sc.Manifest.ID != d.Name() {
			return nil, fmt.Errorf("%s: id сценария %q не совпадает с каталогом", dir, sc.Manifest.ID)
		}
		lib.scenarios[sc.Manifest.ID] = sc
		lib.order = append(lib.order, sc.Manifest.ID)
	}
	sort.Strings(lib.order)
	if len(lib.order) == 0 {
		return nil, fmt.Errorf("каталог заготовок %s: сценариев нет", root)
	}
	if lib.Default == "" {
		lib.Default = lib.order[0]
	}
	if _, ok := lib.scenarios[lib.Default]; !ok {
		return nil, fmt.Errorf("default_scenario %q: нет такого сценария", lib.Default)
	}
	return lib, nil
}

func loadScenario(fsys fs.FS, dir string) (*Scenario, error) {
	sc := &Scenario{}
	if err := readYAML(fsys, path.Join(dir, "scenario.yaml"), &sc.Manifest); err != nil {
		return nil, err
	}
	if err := sc.Manifest.validate(); err != nil {
		return nil, err
	}
	if len(sc.Manifest.LocalIDs) > 0 {
		sc.localIDs = regexp.MustCompile("^(?:" + strings.Join(sc.Manifest.LocalIDs, "|") + ")$")
	}
	prev := stepState{byKey: map[string]*entry{}, byOp: map[string][]*entry{}}
	for i, h := range sc.Manifest.Steps {
		var f StepFile
		name := path.Join(dir, "steps", fmt.Sprintf("%02d.yaml", i))
		if err := readYAML(fsys, name, &f); err != nil {
			return nil, err
		}
		if f.Format != FormatVersion || f.Step != i || !f.Clock.Equal(h.Clock) {
			return nil, fmt.Errorf("%s: формат, номер шага или часы не совпадают с scenario.yaml", name)
		}
		cur := stepState{file: f, byKey: make(map[string]*entry, len(prev.byKey)+len(f.Responses))}
		for k, e := range prev.byKey {
			cur.byKey[k] = e
		}
		for _, r := range f.Responses {
			e, err := newEntry(r)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			cur.byKey[r.Key()] = e
		}
		cur.byOp = make(map[string][]*entry)
		for _, e := range cur.byKey {
			cur.byOp[e.resp.Op] = append(cur.byOp[e.resp.Op], e)
		}
		sc.steps = append(sc.steps, cur)
		prev = cur
	}
	return sc, nil
}

func newEntry(r Response) (*entry, error) {
	if r.Op == "" {
		return nil, fmt.Errorf("ответ без op")
	}
	b, err := r.BodyJSON()
	if err != nil {
		return nil, fmt.Errorf("%s: тело: %w", r.Op, err)
	}
	return &entry{resp: r, body: b}, nil
}

func readYAML(fsys fs.FS, name string, v any) error {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// BlobPrefix — строка тела "@blob:‹имя›" заменяется содержимым common/blobs/‹имя›.
const BlobPrefix = "@blob:"

// expandBlobs подставляет большие строки в тело ответа.
func (l *Library) expandBlobs(body []byte) ([]byte, error) {
	marker := []byte(`"` + BlobPrefix)
	if !bytes.Contains(body, marker) {
		return body, nil
	}
	var out bytes.Buffer
	for {
		i := bytes.Index(body, marker)
		if i < 0 {
			out.Write(body)
			return out.Bytes(), nil
		}
		j := bytes.IndexByte(body[i+len(marker):], '"')
		if j < 0 {
			return nil, fmt.Errorf("незакрытая строка @blob")
		}
		name := string(body[i+len(marker) : i+len(marker)+j])
		data, ok := l.blobs[name]
		if !ok {
			return nil, fmt.Errorf("нет common/blobs/%s", name)
		}
		enc, err := json.Marshal(string(data))
		if err != nil {
			return nil, err
		}
		out.Write(body[:i])
		out.Write(enc)
		body = body[i+len(marker)+j+1:]
	}
}

// Scenarios — id сценариев по алфавиту.
func (l *Library) Scenarios() []string { return slices.Clone(l.order) }

// Scenario — сценарий по id.
func (l *Library) Scenario(id string) (*Scenario, bool) {
	s, ok := l.scenarios[id]
	return s, ok
}

// Steps — число шагов сценария.
func (s *Scenario) Steps() int { return len(s.steps) }

// Header — заголовок шага n.
func (s *Scenario) Header(n int) StepHeader { return s.Manifest.Steps[n] }

// Changes — изменения сущностей на шаге n (для SSE).
func (s *Scenario) Changes(n int) []Change { return s.steps[n].file.Changes }

// StepAt — последний шаг с часами ≤ t (не больше limit); раньше шага 0 — 0.
func (s *Scenario) StepAt(t time.Time, limit int) int {
	n := 0
	for i := 0; i <= limit && i < len(s.Manifest.Steps); i++ {
		if s.Manifest.Steps[i].Clock.After(t) {
			break
		}
		n = i
	}
	return n
}

// Resolution — итог поиска ответа.
type Resolution int

const (
	// Found — ответ найден.
	Found Resolution = iota
	// NoParams — у операции есть заготовки, но не для этих параметров (404).
	NoParams
	// NoOperation — операция в мире заготовок не наполнена (501).
	NoOperation
)

// resolve — ответ операции op при параметрах params на шаге n: из подходящих
// (параметры ответа — подмножество параметров вызова) — самый точный; ответ
// сценария перекрывает общий.
func (l *Library) resolve(s *Scenario, n int, op string, params map[string]string) (*entry, Resolution) {
	cands := s.steps[n].byOp[op]
	if e := best(cands, params); e != nil {
		return e, Found
	}
	if e := best(l.commonOps[op], params); e != nil {
		return e, Found
	}
	if len(cands) > 0 || len(l.commonOps[op]) > 0 {
		return nil, NoParams
	}
	// Операция появится на более позднем шаге — это «не найдено», а не «не наполнено».
	for i := n + 1; i < len(s.steps); i++ {
		if len(s.steps[i].byOp[op]) > 0 {
			return nil, NoParams
		}
	}
	return nil, NoOperation
}

func best(cands []*entry, params map[string]string) *entry {
	var out *entry
	for _, e := range cands {
		if !subset(e.resp.Params, params) {
			continue
		}
		if out == nil || len(e.resp.Params) > len(out.resp.Params) ||
			(len(e.resp.Params) == len(out.resp.Params) && e.resp.Key() < out.resp.Key()) {
			out = e
		}
	}
	return out
}

func subset(sub, of map[string]string) bool {
	for k, v := range sub {
		if of[k] != v {
			return false
		}
	}
	return true
}

func canonParams(p map[string]string) string {
	if len(p) == 0 {
		return ""
	}
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(p[k])
	}
	return b.String()
}

// Walk обходит все ответы библиотеки (общие и все шаги всех сценариев) — для
// проверки тел схемами openapi.yaml.
func (l *Library) Walk(fn func(where string, r Response, body []byte) error) error {
	keys := make([]string, 0, len(l.common))
	for k := range l.common {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		e := l.common[k]
		if err := fn("common", e.resp, e.body); err != nil {
			return err
		}
	}
	for _, id := range l.order {
		s := l.scenarios[id]
		for i, st := range s.steps {
			for _, r := range st.file.Responses {
				e := st.byKey[r.Key()]
				if err := fn(fmt.Sprintf("%s/steps/%02d", id, i), e.resp, e.body); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
