package cad

import (
	"time"

	"ant/internal/application/platform"
)

// CadComponent — позиция условной сборки: позиция в файле, обозначение КД,
// количество и учёт; наш тип изделия и родитель в дереве состава.
type CadComponent struct {
	Position    string `json:"position"`
	Designation string `json:"designation"`
	Quantity    int    `json:"quantity" minimum:"1"`
	LotTracked  bool   `json:"lot_tracked"`
	// Расширение эпика 31 (совместимо): дерево состава и учёт.
	ItemTypeID       string `json:"item_type_id,omitempty" doc:"Наш тип изделия по обозначению КД (FR-95)."`
	Name             string `json:"name,omitempty"`
	Kind             string `json:"kind,omitempty" doc:"detail, subassembly, standard_part, fastener, purchased_equipment, other."`
	ParentItemTypeID string `json:"parent_item_type_id,omitempty" doc:"Родитель в дереве состава."`
	TotalQuantity    int    `json:"total_quantity,omitempty" doc:"Количество на одну сборку верхнего уровня."`
	Tracking         string `json:"tracking,omitempty" doc:"serial, lot или none."`
	MakeOrBuy        string `json:"make_or_buy,omitempty" doc:"make или buy."`
	ShelfLifeTracked bool   `json:"shelf_life_tracked,omitempty"`
	Material         string `json:"material,omitempty"`
}

// CadLink — связь сборки (сварной шов W-1, болтовое соединение J-1,
// уплотнение S-1, резьбовое соединение): переводится в зоны и ограничения
// нормативного слоя (AD-18, PRD §11.15).
type CadLink struct {
	LinkID     string   `json:"link_id"`
	Kind       string   `json:"kind"`
	Components []string `json:"components"`
	LinkType   string   `json:"link_type,omitempty" doc:"Вид связи в файле, если шире kind."`
	ZoneID     string   `json:"zone_id,omitempty"`
	Note       string   `json:"note,omitempty"`
}

// CadZone — зона изделия из связи сборки (FR-46).
type CadZone struct {
	ZoneID     string `json:"zone_id"`
	Name       string `json:"name"`
	Kind       string `json:"kind" doc:"weld_section, joint, hole, surface, cavity, groove, other."`
	LinkID     string `json:"link_id"`
	ItemTypeID string `json:"item_type_id,omitempty" doc:"Тип изделия, на котором лежит зона."`
}

// CadConstraint — ограничение нормативного слоя из связи: лимит ремонтов шва
// (FR-18), «закрывает доступ к зоне» (FR-20), порядок установки (FR-17),
// крепёж, момент затяжки.
type CadConstraint struct {
	ConstraintID    string   `json:"constraint_id"`
	Kind            string   `json:"kind" doc:"rework_limit, closes_access, install_order, fastening, torque."`
	LinkID          string   `json:"link_id"`
	ZoneID          string   `json:"zone_id,omitempty"`
	Limit           *int     `json:"limit,omitempty" doc:"Лимит ремонтов зоны."`
	LimitSource     string   `json:"limit_source,omitempty" doc:"Файл сборки или шаг ТП нормативного слоя."`
	ClosesZoneIDs   []string `json:"closes_zone_ids,omitempty" doc:"Зоны, к которым соединение закрывает доступ."`
	FirstItemTypeID string   `json:"first_item_type_id,omitempty"`
	ThenItemTypeID  string   `json:"then_item_type_id,omitempty"`
	StepKey         string   `json:"step_key,omitempty" doc:"Шаг процесса, исполняющий ограничение."`
	Quantity        int      `json:"quantity,omitempty"`
	Value           *int     `json:"value,omitempty"`
	Unit            string   `json:"unit,omitempty"`
	TolerancePct    *int     `json:"tolerance_pct,omitempty"`
	Note            string   `json:"note,omitempty"`
}

// CadCharacteristic — важная характеристика с методами контроля (FR-14).
type CadCharacteristic struct {
	CharacteristicID string   `json:"characteristic_id"`
	Name             string   `json:"name"`
	Methods          []string `json:"methods,omitempty"`
	Note             string   `json:"note,omitempty"`
}

// CadDiscrepancy — расхождение с номенклатурой учётной системы (FR-95).
type CadDiscrepancy struct {
	ItemTypeID  string `json:"item_type_id"`
	Designation string `json:"designation"`
	System      string `json:"system" doc:"onec, galaktika, other."`
	Reason      string `json:"reason" doc:"not_in_erp — нет в номенклатуре; not_checked — сверка не выполнялась."`
	Note        string `json:"note,omitempty"`
}

// CadAssembly — импортированная условная сборка КОМПАС-3D (cad.assembly.imported, FR-94).
type CadAssembly struct {
	AssemblyDesignation string         `json:"assembly_designation" doc:"Например ФЛ-100.00.000 СБ."`
	Revision            string         `json:"revision"`
	FileDigest          string         `json:"file_digest" doc:"Отпечаток файла (streebog256:…)."`
	Components          []CadComponent `json:"components"`
	Links               []CadLink      `json:"links"`
	GeometryPresent     bool           `json:"geometry_present" doc:"Геометрия не импортируется (geometry: null)."`
	ImportedAt          time.Time      `json:"imported_at"`
	// Расширение эпика 31 (совместимо): тип изделия, зоны, ограничения, расхождения.
	AssemblyItemTypeID string              `json:"assembly_item_type_id,omitempty"`
	AssemblyName       string              `json:"assembly_name,omitempty"`
	LifecycleLetter    string              `json:"lifecycle_letter,omitempty"`
	GeometryNote       string              `json:"geometry_note,omitempty"`
	SourceSystem       string              `json:"source_system,omitempty"`
	Zones              []CadZone           `json:"zones,omitempty"`
	Constraints        []CadConstraint     `json:"constraints,omitempty"`
	Characteristics    []CadCharacteristic `json:"characteristics,omitempty"`
	Discrepancies      []CadDiscrepancy    `json:"discrepancies,omitempty"`
	Seq                int64               `json:"seq,omitempty" doc:"Позиция записи импорта в журнале."`
}

// CadAssemblyList — импортированные сборки.
type CadAssemblyList struct {
	Items []CadAssembly `json:"items"`
}

// ImportAssembly — импорт файла условной сборки (схема contracts/integrations/cad/assembly.schema.json, эпик 31).
type ImportAssembly struct {
	platform.CommandHeader
	FileName string `json:"file_name" maxLength:"256"`
	Content  string `json:"content" contentEncoding:"base64" doc:"Файл условной сборки (JSON), base64."`
}
