package quality

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"go.yaml.in/yaml/v3"

	"ant/internal/contracts/bpmnext"
	"ant/internal/contracts/normative"
	"ant/internal/domain/quality"
)

// Пути файлов нормативного слоя, из которых собирается quality.Env (от корня
// репозитория, как в normative.version.loaded.components[].path, AD-17).
const (
	PathClassifier  = "normative/defects/classifier.v1.yaml"
	PathReactionMap = "normative/reactions/reaction-map.v1.yaml"
	PathProcess     = "normative/process/flange-process.bpmn"
	PathItemTypes   = "normative/reference/flange/item-types.yaml"
)

// NormFiles — содержимое файлов нормативного слоя для модуля quality.
type NormFiles struct {
	Classifier  []byte
	ReactionMap []byte
	Process     []byte
	// ItemTypes — номенклатура с зонами (необязательно: без неё вид зоны неизвестен).
	ItemTypes []byte
}

// ReadNormFiles читает файлы нормативного слоя из fsys (корень — корень репозитория).
func ReadNormFiles(fsys fs.FS) (NormFiles, error) {
	var f NormFiles
	var err error
	if f.Classifier, err = fs.ReadFile(fsys, PathClassifier); err != nil {
		return f, err
	}
	if f.ReactionMap, err = fs.ReadFile(fsys, PathReactionMap); err != nil {
		return f, err
	}
	if f.Process, err = fs.ReadFile(fsys, PathProcess); err != nil {
		return f, err
	}
	if b, err := fs.ReadFile(fsys, PathItemTypes); err == nil {
		f.ItemTypes = b
	} else if !errors.Is(err, fs.ErrNotExist) {
		return f, err
	}
	return f, nil
}

// EnvFrom собирает часть нормативного слоя модуля quality (AD-17): классификатор
// видов дефектов (FR-125), карту реакций (FR-48), точки контроля из расширения
// BPMN (FR-12, FR-14) и зоны изделия. rev — ревизия нормативного слоя
// (normative_rev реакций). Паспорта анализатора — отдельно (quality.PassportsFrom).
//
// Эту функцию вызывает BundleSource движка (эпик 17: process) для версии
// процесса, закреплённой при запуске изделия.
func EnvFrom(f NormFiles, rev string) (quality.Env, error) {
	env := quality.Env{RuleRev: rev}
	if err := yamlInto(f.Classifier, &env.Classifier); err != nil {
		return env, fmt.Errorf("классификатор дефектов: %w", err)
	}
	if err := yamlInto(f.ReactionMap, &env.ReactionMap); err != nil {
		return env, fmt.Errorf("карта реакций: %w", err)
	}
	steps, err := StepsFromBPMN(f.Process)
	if err != nil {
		return env, fmt.Errorf("процесс: %w", err)
	}
	env.Steps = steps
	if len(f.ItemTypes) > 0 {
		var it normative.ItemTypesSeed
		if err := yamlInto(f.ItemTypes, &it); err != nil {
			return env, fmt.Errorf("номенклатура: %w", err)
		}
		env.Zones = zonesOf(it)
	}
	return env, nil
}

// EnvFromFS — EnvFrom по файлам из fsys.
func EnvFromFS(fsys fs.FS, rev string) (quality.Env, error) {
	f, err := ReadNormFiles(fsys)
	if err != nil {
		return quality.Env{}, err
	}
	return EnvFrom(f, rev)
}

// yamlInto — YAML → JSON → сгенерированный тип (у типов контрактов теги json).
func yamlInto(b []byte, v any) error {
	var raw any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return err
	}
	j, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(j))
	return dec.Decode(v)
}

// zonesOf — зоны всех типов изделий и соединения (W-1 — вид своих участков).
func zonesOf(it normative.ItemTypesSeed) []quality.ZoneSpec {
	out := []quality.ZoneSpec{}
	seen := map[string]bool{}
	add := func(id, kind string) {
		if id == "" || kind == "" || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, quality.ZoneSpec{ID: id, Kind: kind})
	}
	for _, t := range it.ItemTypes {
		kinds := map[string]string{}
		for _, z := range t.Zones {
			add(z.ZoneID, string(z.Kind))
			kinds[z.ZoneID] = string(z.Kind)
		}
		for _, l := range t.Links {
			if len(l.Zones) > 0 {
				add(l.LinkID, kinds[l.Zones[0]])
			}
		}
	}
	return out
}

// StepsFromBPMN — шаги процесса со свойствами контроля из расширения
// urn:ant:bpmn-ext:1 (AD-17) в порядке описания: для каждого узла с
// ant:properties/@stepKey — вид шага, точка контроля, закрывающая точка,
// методы с покрытием, требования КД, зоны, карта реакций.
func StepsFromBPMN(b []byte) ([]quality.StepSpec, error) {
	dec := xml.NewDecoder(bytes.NewReader(b))
	out := []quality.StepSpec{}
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "extensionElements" {
			continue
		}
		var ext bpmnext.ExtensionElements
		if err := dec.DecodeElement(&ext, &se); err != nil {
			return nil, err
		}
		if len(ext.Properties) == 0 || ext.Properties[0].StepKey == "" {
			continue
		}
		p := ext.Properties[0]
		stage, _, _ := strings.Cut(p.StepKey, ".")
		st := quality.StepSpec{StepKey: p.StepKey, Order: len(out) + 1, Stage: stage, StepKind: p.StepKind,
			InspectionPoint: p.InspectionPoint, ClosingPoint: p.ClosingPoint, SpecialProcess: p.SpecialProcess != nil && *p.SpecialProcess}
		for _, in := range ext.Inspection {
			spec := quality.InspectionSpec{Method: in.Method, Phase: in.Phase, Coverage: strings.Fields(in.Coverage), RecipeRef: in.RecipeRef}
			if in.ObservationQualityMinBp != nil {
				spec.ObservationQualityMinBP = int(*in.ObservationQualityMinBp)
			}
			st.Inspections = append(st.Inspections, spec)
		}
		for _, r := range ext.Requirement {
			st.Requirements = append(st.Requirements, quality.Requirement{Characteristic: r.Characteristic, Tolerance: r.Tolerance, KDRef: r.KdRef})
		}
		for _, z := range ext.ZoneRef {
			st.Zones = append(st.Zones, z.Zone)
		}
		if len(ext.ReactionMap) > 0 {
			st.ReactionMapRef = ext.ReactionMap[0].Ref
		}
		out = append(out, st)
	}
	return out, nil
}
