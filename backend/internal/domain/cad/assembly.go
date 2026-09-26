package cad

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Assembly — условная сборка на нашем языке: то, что адаптер КОМПАС прочитал
// из файла (или, в промышленном пути, из COM API7 / ЛОЦМАН:PLM). Формат файла
// сюда не попадает (AD-18).
type Assembly struct {
	// Designation — обозначение сборки по КД («ФЛ-100.00.000 СБ»).
	Designation string
	// AssemblyID — идентификатор сборки в файле («FL-100.00.000»); дочерние
	// позиции ссылаются на него полем Parent.
	AssemblyID string
	Name       string
	// Version — версия КД (литера изменения, «Б»), Letter — литера (О1).
	Version, Letter string
	// GeometryPresent — геометрия в файле есть; в условной сборке — нет
	// (geometry: null, кейс §5.3), GeometryNote — пояснение из файла.
	GeometryPresent bool
	GeometryNote    string
	// Source — откуда состав: система, вид выгрузки, время, путь к прямому подключению.
	Source Source
	// FormatVersion — версия формата файла.
	FormatVersion string
	// Digest — отпечаток файла (streebog256:…): идемпотентность импорта.
	Digest          string
	Components      []Component
	Links           []Link
	Characteristics []Characteristic
	// MappingRule, ConflictRule — правило сопоставления ID с 1С из файла (для людей).
	MappingRule, ConflictRule string
}

// Source — источник состава.
type Source struct {
	System, ExportKind, Note string
	// ExportedAt — когда состав выгружен (UTC; нулевое — неизвестно).
	ExportedAt time.Time
}

// Вид позиции состава (как в файле условной сборки).
const (
	KindDetail             = "detail"
	KindSubassembly        = "subassembly"
	KindStandardPart       = "standard_part"
	KindFastener           = "fastener"
	KindPurchasedEquipment = "purchased_equipment"
)

// Component — позиция состава.
type Component struct {
	Position    int
	Designation string
	Name        string
	Kind        string
	Qty         int
	Unit        string
	Material    string
	// MakeOrBuy — make (изготавливаем) или buy (покупное).
	MakeOrBuy string
	// Parent — обозначение (или AssemblyID) родителя в дереве.
	Parent string
	// Учёт: по партиям, по заводским номерам, со сроком хранения.
	LotTracked, SerialTracked, ShelfLifeTracked bool
	Note                                        string
}

// Вид связи сборки.
const (
	LinkWeld          = "weld"
	LinkBoltedJoint   = "bolted_joint"
	LinkSeal          = "seal"
	LinkThreadedJoint = "threaded_joint"
)

// Link — связь сборки: шов, болтовое или резьбовое соединение, уплотнение.
type Link struct {
	ID   string
	Type string
	// From, To — соединяемые позиции (обозначения); Between — пара для уплотнения.
	From, To string
	Between  []string
	// Part — позиция-уплотнение; Fasteners — крепёж соединения; Qty — число крепежа.
	Part      string
	Fasteners []string
	Qty       int
	// TighteningSequence — порядок затяжки; TorqueNm и TolerancePct — момент
	// и допуск (целые, AD-4); Locking — контровка.
	TighteningSequence string
	TorqueNm           int
	TolerancePct       int
	Locking            string
	// ReworkLimit — лимит ремонтов из файла (0 — по ТП, из нормативного слоя).
	ReworkLimit int
	// ClosesAccessTo — зоны, к которым соединение закрывает доступ, если заданы в файле явно.
	ClosesAccessTo []string
	Note           string
}

// Characteristic — важная характеристика с методами контроля.
type Characteristic struct {
	ID, Name string
	Control  []string
	Note     string
}

// Problem — нарушение структуры сборки, которое схема файла не ловит:
// поле (JSON Pointer в терминах файла) и что не так. Импорт с проблемой не
// принимается (кейс §5.3: связь на несуществующую позицию — карантин с кодом).
type Problem struct {
	Field  string
	Detail string
}

func (p Problem) Error() string { return p.Field + ": " + p.Detail }

// Validate — структурные правила сборки (чистая функция): позиции и
// обозначения уникальны и не пусты, родитель каждой позиции существует и
// дерево без циклов, связи ссылаются на существующие позиции, геометрия
// отсутствует явно или присутствует явно, версия КД задана.
func Validate(a Assembly) []Problem {
	var ps []Problem
	add := func(field, format string, args ...any) {
		ps = append(ps, Problem{Field: field, Detail: fmt.Sprintf(format, args...)})
	}
	if strings.TrimSpace(a.Designation) == "" {
		add("/assembly/designation", "пустое обозначение сборки")
	}
	if a.AssemblyID == "" {
		add("/assembly/assembly_id", "пустой идентификатор сборки")
	}
	if strings.TrimSpace(a.Version) == "" {
		add("/assembly/version", "не задана версия КД: изделие исполняется по версии, действовавшей при запуске")
	}
	if len(a.Components) == 0 {
		add("/components", "в сборке нет позиций")
	}
	byDes := map[string]int{}
	positions := map[int]bool{}
	for i, c := range a.Components {
		f := "/components/" + strconv.Itoa(i)
		if strings.TrimSpace(c.Designation) == "" || ItemTypeID(c.Designation) == "" {
			add(f+"/designation", "пустое обозначение позиции — ошибка импорта, а не новая деталь")
			continue
		}
		if positions[c.Position] {
			add(f+"/position", "позиция %d повторяется", c.Position)
		}
		positions[c.Position] = true
		if _, dup := byDes[c.Designation]; dup {
			add(f+"/designation", "обозначение «%s» повторяется", c.Designation)
		}
		byDes[c.Designation] = i
		if c.Qty < 1 {
			add(f+"/qty", "количество должно быть не меньше 1")
		}
	}
	for i, c := range a.Components {
		if c.Parent != a.AssemblyID && c.Parent != a.Designation {
			if _, ok := byDes[c.Parent]; !ok {
				add("/components/"+strconv.Itoa(i)+"/parent", "родитель «%s» не найден в составе", c.Parent)
			}
		}
	}
	if len(ps) == 0 {
		for i, c := range a.Components {
			if cyc := cycle(a, byDes, c); cyc {
				add("/components/"+strconv.Itoa(i)+"/parent", "цикл в дереве состава через «%s»", c.Designation)
				break
			}
		}
	}
	known := func(d string) bool { _, ok := byDes[d]; return ok }
	ids := map[string]bool{}
	for i, l := range a.Links {
		f := "/links/" + strconv.Itoa(i)
		if l.ID == "" {
			add(f+"/id", "у связи нет идентификатора")
		} else if ids[l.ID] {
			add(f+"/id", "связь %s повторяется", l.ID)
		}
		ids[l.ID] = true
		refs := map[string]string{"/from": l.From, "/to": l.To, "/part": l.Part}
		for _, k := range slices.Sorted(maps.Keys(refs)) {
			if v := refs[k]; v != "" && !known(v) {
				add(f+k, "связь %s ссылается на несуществующую позицию «%s»", l.ID, v)
			}
		}
		for j, v := range l.Between {
			if !known(v) {
				add(f+"/between/"+strconv.Itoa(j), "связь %s ссылается на несуществующую позицию «%s»", l.ID, v)
			}
		}
		for j, v := range l.Fasteners {
			if !known(v) {
				add(f+"/fasteners/"+strconv.Itoa(j), "связь %s ссылается на несуществующий крепёж «%s»", l.ID, v)
			}
		}
		switch l.Type {
		case LinkSeal:
			if len(l.Between) != 2 || l.Part == "" {
				add(f, "уплотнение %s: нужны пара позиций between и позиция-уплотнение part", l.ID)
			}
		default:
			if l.From == "" || l.To == "" {
				add(f, "связь %s: нужны позиции from и to", l.ID)
			}
		}
	}
	return ps
}

// cycle — путь от позиции к корню повторяет позицию.
func cycle(a Assembly, byDes map[string]int, c Component) bool {
	seen := map[string]bool{c.Designation: true}
	p := c.Parent
	for p != a.AssemblyID && p != a.Designation {
		if seen[p] {
			return true
		}
		seen[p] = true
		i, ok := byDes[p]
		if !ok {
			return false
		}
		p = a.Components[i].Parent
	}
	return false
}
