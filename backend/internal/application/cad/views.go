package cad

import (
	"time"

	"ant/internal/application/platform"
)

// CadComponent — позиция условной сборки.
type CadComponent struct {
	Position    string `json:"position"`
	Designation string `json:"designation"`
	Quantity    int    `json:"quantity" minimum:"1"`
	LotTracked  bool   `json:"lot_tracked"`
}

// CadLink — связь сборки (сварной шов W-1, соединение J-1, крепёж S-1): переводится
// в зоны и ограничения нормативного слоя (AD-18, PRD §11.15).
type CadLink struct {
	LinkID     string   `json:"link_id"`
	Kind       string   `json:"kind"`
	Components []string `json:"components"`
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
