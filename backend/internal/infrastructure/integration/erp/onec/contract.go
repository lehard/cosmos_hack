package onec

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"ant/internal/contracts/schemas"
)

// Пути контракта 1С во встроенной копии contracts/.
const (
	contractDir = "integrations/erp/1c/"
	// ContractVersion — версия контракта HTTP-сервиса qc, которую знает адаптер.
	ContractVersion = "qc.v1"
	schemaPosting   = contractDir + "qc.v1/posting.schema.json"
	schemaInspect   = contractDir + "qc.v1/inspection-result.schema.json"
	schemaReceipt   = contractDir + "qc.v1/receipt.schema.json"
	schemaError     = contractDir + "qc.v1/error.schema.json"
	schemaAbout     = contractDir + "qc.v1/about.schema.json"
	manifestPath    = contractDir + "metadata-manifest.json"
)

// Manifest — ожидаемые метаданные OData 1С (contracts/integrations/erp/1c/metadata-manifest.json).
type Manifest struct {
	Manifest string `json:"manifest"`
	Contract string `json:"contract"`
	OData    struct {
		Version   string `json:"version"`
		Namespace string `json:"namespace"`
		Container string `json:"container"`
		Path      string `json:"path"`
	} `json:"odata"`
	QC struct {
		Path     string `json:"path"`
		Contract string `json:"contract"`
	} `json:"qc"`
	Entities []Entity `json:"entities"`
}

// Entity — сущность OData: реквизиты и табличные части.
type Entity struct {
	Name       string     `json:"name"`
	Title      string     `json:"title"`
	Key        string     `json:"key"`
	Properties []Property `json:"properties"`
	Tables     []Table    `json:"tables"`
}

// Table — табличная часть сущности (коллекция строк).
type Table struct {
	Name       string     `json:"name"`
	Properties []Property `json:"properties"`
}

// Property — реквизит: имя, тип EDM, опирается ли на него адаптер.
type Property struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

var (
	manifestOnce sync.Once
	manifest     Manifest
	manifestErr  error
)

// LoadManifest — манифест из встроенного контракта.
func LoadManifest() (Manifest, error) {
	manifestOnce.Do(func() {
		b, err := schemas.FS.ReadFile("contracts/" + manifestPath)
		if err != nil {
			manifestErr = err
			return
		}
		manifestErr = json.Unmarshal(b, &manifest)
	})
	return manifest, manifestErr
}

// Entity — сущность манифеста по имени.
func (m Manifest) Entity(name string) (Entity, bool) {
	for _, e := range m.Entities {
		if e.Name == name {
			return e, true
		}
	}
	return Entity{}, false
}

// EDMX — документ `$metadata` OData v3 (CSDL 3.0), как его отдаёт 1С.
type EDMX struct {
	XMLName      xml.Name `xml:"Edmx"`
	Version      string   `xml:"Version,attr"`
	DataServices struct {
		Schemas []edmSchema `xml:"Schema"`
	} `xml:"DataServices"`
}

type edmSchema struct {
	Namespace    string           `xml:"Namespace,attr"`
	EntityTypes  []edmEntityType  `xml:"EntityType"`
	ComplexTypes []edmComplexType `xml:"ComplexType"`
}

type edmEntityType struct {
	Name       string        `xml:"Name,attr"`
	Properties []edmProperty `xml:"Property"`
}

type edmComplexType struct {
	Name       string        `xml:"Name,attr"`
	Properties []edmProperty `xml:"Property"`
}

type edmProperty struct {
	Name     string `xml:"Name,attr"`
	Type     string `xml:"Type,attr"`
	Nullable string `xml:"Nullable,attr"`
}

// RowType — имя комплексного типа строки табличной части (как в 1С).
func RowType(entity, table string) string { return entity + "_" + table + "_RowType" }

// BuildMetadata — `$metadata` по манифесту (им пользуется stand: каркас
// stand-а — из контракта, AD-18). rename — переименовать реквизиты
// (сбой «несовместимые метаданные» для проверки О8).
func BuildMetadata(m Manifest, rename map[string]string) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<edmx:Edmx xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx" Version="1.0">` + "\n")
	b.WriteString(`<edmx:DataServices xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata" m:DataServiceVersion="3.0" m:MaxDataServiceVersion="3.0">` + "\n")
	fmt.Fprintf(&b, `<Schema xmlns="http://schemas.microsoft.com/ado/2009/11/edm" Namespace="%s">`+"\n", esc(m.OData.Namespace))
	name := func(p string) string {
		if r, ok := rename[p]; ok {
			return r
		}
		return p
	}
	for _, e := range m.Entities {
		fmt.Fprintf(&b, `<EntityType Name="%s"><Key><PropertyRef Name="%s"/></Key>`, esc(e.Name), esc(e.Key))
		for _, p := range e.Properties {
			nullable := "true"
			if p.Name == e.Key {
				nullable = "false"
			}
			fmt.Fprintf(&b, `<Property Name="%s" Type="%s" Nullable="%s"/>`, esc(name(p.Name)), esc(p.Type), nullable)
		}
		for _, t := range e.Tables {
			fmt.Fprintf(&b, `<Property Name="%s" Type="Collection(%s.%s)" Nullable="false"/>`, esc(name(t.Name)), esc(m.OData.Namespace), esc(RowType(e.Name, t.Name)))
		}
		b.WriteString("</EntityType>\n")
		for _, t := range e.Tables {
			fmt.Fprintf(&b, `<ComplexType Name="%s">`, esc(RowType(e.Name, t.Name)))
			for _, p := range t.Properties {
				fmt.Fprintf(&b, `<Property Name="%s" Type="%s" Nullable="true"/>`, esc(name(p.Name)), esc(p.Type))
			}
			b.WriteString("</ComplexType>\n")
		}
	}
	fmt.Fprintf(&b, `<EntityContainer Name="%s" m:IsDefaultEntityContainer="true">`, esc(m.OData.Container))
	for _, e := range m.Entities {
		fmt.Fprintf(&b, `<EntitySet Name="%s" EntityType="%s.%s"/>`, esc(e.Name), esc(m.OData.Namespace), esc(e.Name))
	}
	b.WriteString("</EntityContainer>\n</Schema>\n</edmx:DataServices>\n</edmx:Edmx>\n")
	return b.Bytes()
}

func esc(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// CompareMetadata — расхождения `$metadata` ответной стороны с манифестом:
// нет сущности, нет обязательного реквизита, другой тип (AD-18). Пусто —
// совпадает. Лишние сущности и реквизиты расхождением не считаются.
func CompareMetadata(m Manifest, raw []byte) ([]string, error) {
	var doc EDMX
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("$metadata не разобран: %w", err)
	}
	types := map[string]map[string]string{}
	for _, s := range doc.DataServices.Schemas {
		for _, e := range s.EntityTypes {
			types[e.Name] = props(e.Properties)
		}
		for _, c := range s.ComplexTypes {
			types[c.Name] = props(c.Properties)
		}
	}
	var diff []string
	check := func(owner string, want []Property) {
		have, ok := types[owner]
		if !ok {
			diff = append(diff, "нет типа "+owner)
			return
		}
		for _, p := range want {
			if !p.Required {
				continue
			}
			t, ok := have[p.Name]
			switch {
			case !ok:
				diff = append(diff, fmt.Sprintf("%s: нет реквизита %s", owner, p.Name))
			case t != p.Type:
				diff = append(diff, fmt.Sprintf("%s.%s: тип %s, ожидался %s", owner, p.Name, t, p.Type))
			}
		}
	}
	for _, e := range m.Entities {
		check(e.Name, e.Properties)
		for _, t := range e.Tables {
			check(RowType(e.Name, t.Name), t.Properties)
			if _, ok := types[e.Name][t.Name]; !ok {
				if _, has := types[e.Name]; has {
					diff = append(diff, fmt.Sprintf("%s: нет табличной части %s", e.Name, t.Name))
				}
			}
		}
	}
	slices.Sort(diff)
	return slices.Compact(diff), nil
}

func props(ps []edmProperty) map[string]string {
	out := make(map[string]string, len(ps))
	for _, p := range ps {
		out[p.Name] = p.Type
	}
	return out
}

// Проверка сообщений схемами контракта qc.v1 (тем же валидатором, что приём, AD-20).

var (
	valMu    sync.Mutex
	valC     *jsonschema.Compiler
	valCache = map[string]*jsonschema.Schema{}
)

type fsLoader struct{}

func (fsLoader) Load(url string) (any, error) {
	p, ok := strings.CutPrefix(url, schemas.BaseURI)
	if !ok {
		return nil, fmt.Errorf("схема %s вне contracts/", url)
	}
	b, err := schemas.FS.ReadFile("contracts/" + p)
	if err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(b))
}

// Validate — документ doc по схеме контракта path (от contracts/); ошибка —
// что не так.
func Validate(path string, doc []byte) error {
	valMu.Lock()
	if valC == nil {
		valC = jsonschema.NewCompiler()
		valC.DefaultDraft(jsonschema.Draft7)
		valC.UseLoader(fsLoader{})
	}
	sch, ok := valCache[path]
	if !ok {
		var err error
		if sch, err = valC.Compile(schemas.BaseURI + path); err != nil {
			valMu.Unlock()
			return err
		}
		valCache[path] = sch
	}
	valMu.Unlock()
	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(doc))
	if err != nil {
		return err
	}
	return sch.Validate(v)
}
