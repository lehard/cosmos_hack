package kompas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	app "ant/internal/application/cad"
	dom "ant/internal/domain/cad"
	"ant/internal/infrastructure/integration/schemacheck"
)

// SchemaPath — схема файла условной сборки (от contracts/).
const SchemaPath = "integrations/cad/assembly.schema.json"

// FormatMajor — мажорная версия формата, которую понимает адаптер.
const FormatMajor = "1"

// File — файл условной сборки (рукописные типы по схеме контракта; их
// соответствие схеме проверяет контрактный тест на эталоне).
type File struct {
	FormatVersion string `json:"format_version"`
	Source        struct {
		System     string `json:"system"`
		ExportKind string `json:"export_kind"`
		ExportedAt string `json:"exported_at"`
		Note       string `json:"note"`
	} `json:"source"`
	Assembly struct {
		AssemblyID      string          `json:"assembly_id"`
		Designation     string          `json:"designation"`
		Name            string          `json:"name"`
		Version         string          `json:"version"`
		LifecycleLetter string          `json:"lifecycle_letter"`
		Geometry        json.RawMessage `json:"geometry"`
		GeometryNote    string          `json:"geometry_note"`
		Critical        []struct {
			ID      string   `json:"id"`
			Name    string   `json:"name"`
			Control []string `json:"control"`
			Note    string   `json:"note"`
		} `json:"critical_characteristics"`
	} `json:"assembly"`
	Components []struct {
		Position         json.Number `json:"position"`
		Designation      string      `json:"designation"`
		Name             string      `json:"name"`
		Kind             string      `json:"kind"`
		Qty              json.Number `json:"qty"`
		Unit             string      `json:"unit"`
		Material         string      `json:"material"`
		MakeOrBuy        string      `json:"make_or_buy"`
		Parent           string      `json:"parent"`
		ShelfLifeTracked bool        `json:"shelf_life_tracked"`
		LotTracked       bool        `json:"lot_tracked"`
		SerialTracked    bool        `json:"serial_tracked"`
		Note             string      `json:"note"`
	} `json:"components"`
	Links []struct {
		Type               string      `json:"type"`
		ID                 string      `json:"id"`
		From               string      `json:"from"`
		To                 string      `json:"to"`
		Between            []string    `json:"between"`
		Part               string      `json:"part"`
		Fasteners          []string    `json:"fasteners"`
		Qty                json.Number `json:"qty"`
		TighteningSequence string      `json:"tightening_sequence"`
		TorqueNm           json.Number `json:"torque_nm"`
		TolerancePct       json.Number `json:"tolerance_pct"`
		Locking            string      `json:"locking"`
		ReworkLimit        json.Number `json:"rework_limit"`
		ClosesAccessTo     []string    `json:"closes_access_to"`
		Note               string      `json:"note"`
	} `json:"links"`
	Mapping struct {
		Rule             string `json:"rule"`
		ConflictBehavior string `json:"conflict_behavior"`
	} `json:"id_mapping_hint"`
}

// Reader — адаптер порта application/cad.Reader для файла условной сборки.
type Reader struct{}

var _ app.Reader = Reader{}

// Read — файл → сборка на нашем языке; ошибка формата — *app.FormatError.
func (Reader) Read(_ context.Context, _ string, content []byte) (dom.Assembly, error) {
	return Parse(content)
}

// Parse — проверка схемой и версией формата, перевод в dom.Assembly.
func Parse(content []byte) (dom.Assembly, error) {
	var head struct {
		FormatVersion *string `json:"format_version"`
	}
	if err := json.Unmarshal(content, &head); err != nil {
		return dom.Assembly{}, &app.FormatError{Field: "/", Detail: "файл не JSON: " + err.Error()}
	}
	if head.FormatVersion != nil {
		if major, _, _ := strings.Cut(*head.FormatVersion, "."); major != FormatMajor {
			return dom.Assembly{}, &app.FormatError{Field: "/format_version", Version: true,
				Detail: fmt.Sprintf("неизвестная версия формата %q — адаптер понимает %s.x; переобработка после появления повышателя", *head.FormatVersion, FormatMajor)}
		}
	}
	if err := schemacheck.Validate(SchemaPath, content); err != nil {
		f, d := schemacheck.Violation(err)
		return dom.Assembly{}, &app.FormatError{Field: f, Detail: d}
	}
	var f File
	dec := json.NewDecoder(bytes.NewReader(content))
	dec.UseNumber()
	if err := dec.Decode(&f); err != nil {
		return dom.Assembly{}, &app.FormatError{Field: "/", Detail: err.Error()}
	}
	return f.Domain(content)
}

// Domain — файл на нашем языке (перевод на наш язык).
func (f File) Domain(content []byte) (dom.Assembly, error) {
	a := dom.Assembly{Designation: f.Assembly.Designation, AssemblyID: f.Assembly.AssemblyID, Name: f.Assembly.Name,
		Version: f.Assembly.Version, Letter: f.Assembly.LifecycleLetter, GeometryNote: f.Assembly.GeometryNote,
		GeometryPresent: len(f.Assembly.Geometry) > 0 && string(f.Assembly.Geometry) != "null",
		FormatVersion:   f.FormatVersion, Digest: dom.Digest(content),
		Source:      dom.Source{System: f.Source.System, ExportKind: f.Source.ExportKind, Note: f.Source.Note},
		MappingRule: f.Mapping.Rule, ConflictRule: f.Mapping.ConflictBehavior}
	if t, err := time.Parse(time.RFC3339Nano, f.Source.ExportedAt); err == nil {
		a.Source.ExportedAt = t.UTC()
	}
	for _, c := range f.Assembly.Critical {
		a.Characteristics = append(a.Characteristics, dom.Characteristic{ID: c.ID, Name: c.Name, Control: c.Control, Note: c.Note})
	}
	for i, c := range f.Components {
		at := fmt.Sprintf("/components/%d", i)
		pos, err := whole(c.Position, at+"/position")
		if err != nil {
			return dom.Assembly{}, err
		}
		qty, err := whole(c.Qty, at+"/qty")
		if err != nil {
			return dom.Assembly{}, err
		}
		a.Components = append(a.Components, dom.Component{Position: pos, Designation: c.Designation, Name: c.Name, Kind: c.Kind, Qty: qty,
			Unit: c.Unit, Material: c.Material, MakeOrBuy: c.MakeOrBuy, Parent: c.Parent, LotTracked: c.LotTracked,
			SerialTracked: c.SerialTracked, ShelfLifeTracked: c.ShelfLifeTracked, Note: c.Note})
	}
	for i, l := range f.Links {
		at := fmt.Sprintf("/links/%d", i)
		x := dom.Link{ID: l.ID, Type: l.Type, From: l.From, To: l.To, Between: l.Between, Part: l.Part, Fasteners: l.Fasteners,
			TighteningSequence: l.TighteningSequence, Locking: l.Locking, ClosesAccessTo: l.ClosesAccessTo, Note: l.Note}
		var err error
		if x.Qty, err = whole(l.Qty, at+"/qty"); err != nil {
			return dom.Assembly{}, err
		}
		if x.TorqueNm, err = whole(l.TorqueNm, at+"/torque_nm"); err != nil {
			return dom.Assembly{}, err
		}
		if x.TolerancePct, err = whole(l.TolerancePct, at+"/tolerance_pct"); err != nil {
			return dom.Assembly{}, err
		}
		if x.ReworkLimit, err = whole(l.ReworkLimit, at+"/rework_limit"); err != nil {
			return dom.Assembly{}, err
		}
		a.Links = append(a.Links, x)
	}
	return a, nil
}

// whole — целое из числа файла: «12» и «12.0» — 12; дробь — ошибка формата
// (AD-4: в домене нет float, значения — целое + масштаб). Пусто — 0.
func whole(n json.Number, field string) (int, error) {
	if n == "" {
		return 0, nil
	}
	r, ok := new(big.Rat).SetString(string(n))
	if !ok || !r.IsInt() || !r.Num().IsInt64() {
		return 0, &app.FormatError{Field: field, Detail: fmt.Sprintf("ожидалось целое, получено %s (дробные значения — через масштаб)", n)}
	}
	return int(r.Num().Int64()), nil
}
