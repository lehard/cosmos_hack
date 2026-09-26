package world

import (
	"bytes"
	"fmt"
	"io/fs"
	"unicode"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// Названия рядом с кодами в заготовках (интерфейс стола контролёра): зона —
// зоны типа изделия по КД (normative/reference/flange/item-types.yaml),
// оборудование — справочник оборудования (equipment.yaml). Кода нет в
// справочнике — названия нет (поле *_label не отдаётся). Виды дефектов мира
// — свои коды сценария (burn_through, base_metal_pore, …) с названиями
// defectTitle: в классификаторе normative/defects у них другие коды.

// Файлы справочников от корня репозитория (входы генератора).
const (
	ItemTypesFile = "normative/reference/flange/item-types.yaml"
	EquipmentFile = "normative/reference/flange/equipment.yaml"
)

// Names — названия зон и оборудования из справочников.
type Names struct {
	Zones     map[string]string
	Equipment map[string]string
}

// LoadNames читает названия зон и оборудования из fsys (корень репозитория).
func LoadNames(fsys fs.FS) (Names, error) {
	n := Names{Zones: map[string]string{}, Equipment: map[string]string{}}
	var types struct {
		ItemTypes []struct {
			Zones []struct {
				ZoneID string `yaml:"zone_id"`
				Name   string `yaml:"name"`
			} `yaml:"zones"`
		} `yaml:"item_types"`
	}
	if err := readLoose(fsys, ItemTypesFile, &types); err != nil {
		return n, err
	}
	for _, t := range types.ItemTypes {
		for _, z := range t.Zones {
			if z.ZoneID != "" && z.Name != "" {
				n.Zones[z.ZoneID] = z.Name
			}
		}
	}
	var eq struct {
		Equipment []struct {
			ID   string `yaml:"id"`
			Name string `yaml:"name"`
		} `yaml:"equipment"`
	}
	if err := readLoose(fsys, EquipmentFile, &eq); err != nil {
		return n, err
	}
	for _, e := range eq.Equipment {
		if e.ID != "" && e.Name != "" {
			n.Equipment[e.ID] = e.Name
		}
	}
	return n, nil
}

// readLoose — YAML справочника без проверки лишних полей (нужны только названия).
func readLoose(fsys fs.FS, name string, v any) error {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return err
	}
	if err := yaml.NewDecoder(bytes.NewReader(b)).Decode(v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// nameOf — название из словаря; нет — nil.
func nameOf(dict map[string]string, code string) *string {
	if s, ok := dict[code]; ok && code != "" {
		return &s
	}
	return nil
}

// defectLabel — вид дефекта мира по-русски с заглавной (как названия
// справочников); неизвестный вид — nil.
func defectLabel(kind string) *string {
	t := defectTitle(kind)
	if t == "" {
		return nil
	}
	r, size := utf8.DecodeRuneInString(t)
	s := string(unicode.ToUpper(r)) + t[size:]
	return &s
}
