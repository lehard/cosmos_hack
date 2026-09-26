package documents

import (
	"strconv"
	"strings"

	"ant/internal/domain/access"
)

// Шаблон документа шага (AD-12, FR-65): часть нормативного слоя
// (normative/documents/templates.v‹N›.yaml, схема
// contracts/normative/templates.schema.json). Документ = шаблон@версия ×
// события-источники; построитель содержимого выбирается по doc_type.

// DocFormatVersion — версия формата документа (AD-12): правила content,
// отрисовки и отпечатка. Агент токена отвергает неизвестную версию явной
// ошибкой signing.unknown_doc_format; смена правил — новая версия.
const DocFormatVersion = 1

// Классы документов (каталог документов спайна).
const (
	ClassRecord      = "record"
	ClassDecision    = "decision"
	ClassRequirement = "requirement"
	ClassInput       = "input"
)

// Построители содержимого (doc_type шаблона).
const (
	// DocTraveler — сопроводительная карта изделия (ядро, FR-65).
	DocTraveler = "traveler"
	// DocNCStatement — заявление о несоответствии.
	DocNCStatement = "nc_statement"
	// DocNCDisposition — решение по несоответствующей продукции (режимы 4–5, AD-43).
	DocNCDisposition = "nc_disposition"
	// DocGeneric — документ по запросу человека: «Запросить решение», лист
	// утверждения версии процесса.
	DocGeneric = "generic"
)

// Идентификаторы шаблонов позвоночника (normative/documents/templates.v1.yaml).
const (
	TemplateTraveler        = "traveler"
	TemplateNCStatement     = "nc-statement"
	TemplateNCDisposition   = "nc-disposition"
	TemplateProcessApproval = "process-version-approval"
	TemplateDecisionRequest = "decision-request"
)

// Field — поле раздела или сводки: путь в content через точку и подпись.
type Field struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// Section — раздел канонической HTML-отрисовки.
type Section struct {
	// Kind — fields | table | route | text.
	Kind    string  `json:"kind"`
	Title   string  `json:"title,omitempty"`
	Fields  []Field `json:"fields,omitempty"`
	Rows    string  `json:"rows,omitempty"`
	Columns []Field `json:"columns,omitempty"`
	Key     string  `json:"key,omitempty"`
}

// Template — шаблон документа шага@версия.
type Template struct {
	ID      string              `json:"id"`
	Title   string              `json:"title"`
	Class   string              `json:"class"`
	Version int                 `json:"version,omitempty"`
	DocType string              `json:"doc_type,omitempty"`
	Subject string              `json:"subject,omitempty"`
	Summary []Field             `json:"summary,omitempty"`
	Route   []access.RouteStage `json:"route,omitempty"`
	Layout  []Section           `json:"layout,omitempty"`
}

// Ref — template_ref: `‹id›@‹версия›` (нет версии — 1).
func (t Template) Ref() string { return t.ID + "@" + strconv.Itoa(max(t.Version, 1)) }

// Complete — шаблон описан полностью: есть построитель и отрисовка. Шаблоны,
// только объявленные в перечне (эпик 44), документов не порождают.
func (t Template) Complete() bool { return t.DocType != "" && len(t.Layout) > 0 }

// Templates — шаблоны версии нормативного слоя в порядке файла.
type Templates []Template

// ByRef — шаблон по template_ref `‹id›@‹версия›` или по id (последняя версия).
func (ts Templates) ByRef(ref string) (Template, bool) {
	id, ver, hasVer := strings.Cut(ref, "@")
	var best Template
	found := false
	for _, t := range ts {
		if t.ID != id {
			continue
		}
		if hasVer {
			if strconv.Itoa(max(t.Version, 1)) == ver {
				return t, true
			}
			continue
		}
		if !found || max(t.Version, 1) > max(best.Version, 1) {
			best, found = t, true
		}
	}
	return best, found
}
