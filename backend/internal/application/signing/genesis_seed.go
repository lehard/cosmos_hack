package signing

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"ant/internal/contracts/catalog"
	ev "ant/internal/contracts/events"
	"ant/internal/contracts/normative"
	dom "ant/internal/domain/signing"
)

// Состав блока генезиса из затравки normative/ (AD-33, normative/README.md):
// режим часов, реестр профилей, ключи, стартовая политика (сотрудники, роли,
// назначения, полномочия, клейма, квалификации, разделение обязанностей,
// параметры аудита), справочники (места, номенклатура, оборудование и
// поверка, календарь, смены — составом модуля reference) и нормативный слой v1 (normative.version.loaded с
// отпечатками файлов и паспорта допуска анализаторов) с подписями кворума
// ключами демо-персон из этого же блока. Дальше части слоя меняются только
// новыми записями через маршрут подписей модулей-владельцев.

// Пути затравки от корня репозитория (они же components[].path, AD-17).
const (
	SeedPolicyFile    = "normative/policy/policy.v1.yaml"
	SeedLocationsFile = "normative/reference/flange/locations.yaml"
	SeedItemTypesFile = "normative/reference/flange/item-types.yaml"
	SeedEquipmentFile = "normative/reference/flange/equipment.yaml"
	SeedCalendarFile  = "normative/reference/flange/calendar.yaml"
	SeedLotsFile      = "normative/reference/flange/lot-templates.yaml"
	SeedPassportsFile = "normative/vision/analyzer-passports.v1.yaml"
	SeedProcessFile   = "normative/process/flange-process.bpmn"
)

// Полномочия стартовой политики, чьи держатели подписывают нормативный слой
// v1 в генезисе (кворум, FR-23) и допуск анализаторов (AD-29).
const (
	AuthorityQuorum            = "quorum_signature"
	AuthorityAnalyzerAdmission = "analyzer_admission"
)

// Режимы часов журнала (time.clock.mode_set, AD-37).
const (
	ClockSystem   = "system"
	ClockScenario = "scenario"
)

// PersonaKeyRef — ключ ГОСТ демо-персоны (класс scenario, AD-33): ‹псевдоним›@1
// в нижнем регистре — то же соглашение, что у demo-signer.
func PersonaKeyRef(persona string) string { return strings.ToLower(persona) + "@1" }

// PersonaPQRef — ключ ML-DSA-65 демо-персоны (подписанты актов — hybrid, AD-32).
func PersonaPQRef(persona string) string { return strings.ToLower(persona) + "-pq@1" }

// GenesisInput — входы состава блока.
type GenesisInput struct {
	// Profile — профиль окружения: паспорта анализаторов — только в профилях затравки.
	Profile string
	// ClockMode — режим часов журнала (system | scenario).
	ClockMode string
	// ProcessVersionID — id стартовой версии процесса (process.SeedVersionID).
	ProcessVersionID string
	// Keys — открытые ключи блока по порядку (движок, шлюзы, хранитель,
	// верификатор, устройства, демо-персоны, корни партнёра).
	Keys []RegistrationData
	// KeeperFingerprint — отпечаток ключа хранителя (параметры аудита).
	KeeperFingerprint string
	// ValidFrom — начало действия стартовой политики и справочников (доменное время).
	ValidFrom time.Time
	// Anchor — открытые ключи якоря.
	Anchor []dom.AnchorKey
	// References — стартовые справочники (reference.* модуля reference).
	References []GenesisRecord
}

// Seed — разобранная затравка.
type Seed struct {
	Policy    normative.PolicySeed
	Locations normative.LocationsSeed
	ItemTypes normative.ItemTypesSeed
	Equipment normative.EquipmentSeed
	Calendar  normative.CalendarSeed
	Lots      normative.LotTemplatesSeed
	Passports normative.AnalyzerPassportsSeed
	Process   []byte
	// Files — все файлы normative/ (кроме README) по порядку путей: состав
	// нормативного слоя v1 и его отпечатки.
	Files []SeedFile
}

// SeedFile — файл нормативного слоя.
type SeedFile struct {
	Path  string
	Bytes []byte
}

// LoadSeed читает затравку из fsys (корень — корень репозитория).
func LoadSeed(fsys fs.FS) (Seed, error) {
	var s Seed
	var err error
	for _, x := range []struct {
		p string
		v any
	}{{SeedPolicyFile, &s.Policy}, {SeedLocationsFile, &s.Locations}, {SeedItemTypesFile, &s.ItemTypes},
		{SeedEquipmentFile, &s.Equipment}, {SeedCalendarFile, &s.Calendar}, {SeedLotsFile, &s.Lots}, {SeedPassportsFile, &s.Passports}} {
		if err = decodeSeed(fsys, x.p, x.v); err != nil {
			return s, err
		}
	}
	if s.Process, err = fs.ReadFile(fsys, SeedProcessFile); err != nil {
		return s, err
	}
	err = fs.WalkDir(fsys, "normative", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || path.Base(p) == "README.md" {
			return err
		}
		b, err := fs.ReadFile(fsys, p)
		s.Files = append(s.Files, SeedFile{Path: p, Bytes: b})
		return err
	})
	slices.SortFunc(s.Files, func(a, b SeedFile) int { return strings.Compare(a.Path, b.Path) })
	return s, err
}

// decodeSeed — YAML затравки в сгенерированный тип через JSON (обязательные
// поля проверяет сгенерированный UnmarshalJSON).
func decodeSeed(fsys fs.FS, name string, out any) error {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return err
	}
	var raw any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	j, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := json.Unmarshal(j, out); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// NormativeComponents — состав нормативного слоя v1: путь и отпечаток каждого
// файла, отпечаток набора — H(JCS(components)) (normative.version.loaded).
func NormativeComponents(files []SeedFile) ([]map[string]string, string, error) {
	comps := make([]map[string]string, 0, len(files))
	for _, f := range files {
		comps = append(comps, map[string]string{"path": f.Path, "digest": dom.Digest(f.Bytes)})
	}
	c, err := dom.CanonicalOf(comps)
	if err != nil {
		return nil, "", err
	}
	return comps, dom.Digest(c), nil
}

// ComposeGenesis — записи блока генезиса по затравке и ключам.
func ComposeGenesis(in GenesisInput, s Seed) (GenesisSpec, error) {
	p := s.Policy
	code := p.EnterpriseCode
	if code == "" {
		code = s.Locations.Enterprise.Code
	}
	from := in.ValidFrom.UTC().Truncate(time.Millisecond).Format(TimeLayout)
	policyStream := "policy:" + strings.ToLower(code)
	spec := GenesisSpec{EnterpriseCode: code, NormativeVersionHash: dom.Digest(s.Process), Anchor: in.Anchor}
	add := func(t catalog.Type, stream string, data any, quorum ...string) {
		spec.Records = append(spec.Records, GenesisRecord{Type: t, Stream: stream, Data: data, Quorum: quorum})
	}
	mode := in.ClockMode
	if mode == "" {
		mode = ClockSystem
	}
	// Режим часов — свойство журнала (AD-37).
	add(catalog.TimeClockModeSet, "", map[string]any{"mode": mode})
	// Криптопрофили: объект → обязательный профиль (profiles.yaml, AD-32).
	byProfile := map[string][]string{}
	for _, c := range dom.Classes {
		pr := dom.DefaultObjectProfiles()[c]
		byProfile[pr] = append(byProfile[pr], c)
	}
	for _, pr := range []string{dom.ProfileHybrid, dom.ProfileGost} {
		add(catalog.KeyProfileRegistered, "", map[string]any{"profile_id": pr, "object_classes": byProfile[pr]})
	}
	// Ключи (AD-11, AD-33): подтверждение субъекта — genesis.
	have := map[string]bool{}
	for _, k := range in.Keys {
		k.SubjectConfirmation = dom.ConfirmGenesis
		if k.ValidFrom == "" {
			k.ValidFrom = from
		}
		id, _, err := dom.SplitKeyRef(k.KeyRef)
		if err != nil {
			return spec, err
		}
		have[k.KeyRef] = true
		add(catalog.KeyRegistrationRecorded, "key:"+id, k)
	}
	// Стартовая политика (AD-15): сотрудники, роли, назначения, полномочия,
	// клейма, квалификации, разделение обязанностей, параметры аудита.
	for _, x := range p.Persons {
		d := map[string]any{"person_id": x.ID, "display_name": x.Name}
		if len(x.Roles) > 0 {
			d["org_unit"] = x.Roles[0].Scope
		}
		add(catalog.AccessPersonRegistered, "person:"+x.ID, d)
	}
	for _, r := range p.Roles {
		d := map[string]any{"role_id": r.ID, "title": r.Title, "actions": nonNil(r.Actions)}
		if len(r.Inherits) > 0 {
			d["inherits"] = r.Inherits
		}
		add(catalog.PolicyRoleDefined, policyStream, d)
	}
	for _, x := range p.Persons {
		for _, r := range x.Roles {
			add(catalog.PolicyRoleAssigned, policyStream, map[string]any{"person_id": x.ID, "role_id": r.Role, "scope": r.Scope, "valid_from": from})
		}
	}
	holders := map[string][]string{}
	for _, g := range p.Grants.Authorities {
		d := map[string]any{"person_id": g.Person, "authority_id": g.Authority, "scope": g.Scope, "valid_from": from}
		if l := g.Limits; l != nil {
			lim := map[string]any{}
			if len(l.ItemTypeIds) > 0 {
				lim["item_type_ids"] = l.ItemTypeIds
			}
			if l.MaxSeverity != nil {
				lim["max_severity"] = string(*l.MaxSeverity)
			}
			d["limits"] = lim
		}
		holders[g.Authority] = append(holders[g.Authority], g.Person)
		add(catalog.PolicyAuthorityGranted, policyStream, d)
	}
	for _, st := range p.Grants.Stamps {
		add(catalog.PolicyStampIssued, policyStream, map[string]any{"person_id": st.Person, "stamp_id": StampID(st.StampID),
			"inspection_kind": st.Kind, "scope": st.Scope, "order_ref": st.OrderRef, "valid_from": from})
	}
	for _, q := range p.Grants.Qualifications {
		d := map[string]any{"person_id": q.Person, "qualification_id": q.Qualification, "scope": q.Scope, "valid_from": from, "valid_until": q.ValidUntil}
		if q.CertificateRef != nil {
			d["certificate_ref"] = *q.CertificateRef
		}
		add(catalog.AccessQualificationGranted, "person:"+q.Person, d)
	}
	for _, r := range p.SeparationOfDuties {
		add(catalog.PolicySodRuleSet, policyStream, map[string]any{"rule_id": r.ID, "statement": r.Statement,
			"conflicting_actions": r.ConflictingActions, "enabled": true})
	}
	audit := map[string]any{"checkpoint_interval_s": p.Audit.CheckpointIntervalS, "checkpoint_max_gap_s": p.Audit.CheckpointMaxGapS,
		"keeper_key_fingerprint": in.KeeperFingerprint}
	if len(p.Audit.SecurityBusSubscribers) > 0 {
		audit["security_bus_subscribers"] = p.Audit.SecurityBusSubscribers
	}
	add(catalog.PolicyAuditParametersSet, policyStream, audit)
	// Справочники (AD-31): записи модуля reference по затравке normative/reference
	// (номенклатура, места, оборудование и поверки, календарь, смены) — тем же
	// составом, что читает его проекция (эпик 19).
	spec.Records = append(spec.Records, in.References...)
	// Нормативный слой v1 с подписями кворума (FR-23, AD-17, AD-33).
	comps, bundle, err := NormativeComponents(s.Files)
	if err != nil {
		return spec, err
	}
	vid := in.ProcessVersionID
	add(catalog.NormativeVersionLoaded, "process_version:"+vid, map[string]any{"version_id": vid,
		"process_version_hash": spec.NormativeVersionHash, "bundle_digest": bundle, "components": comps},
		quorumOf(holders[AuthorityQuorum], have)...)
	// Паспорта допуска анализаторов демо (AD-29) — только в профилях затравки.
	if in.Profile != "prod" && slices.Contains(s.Passports.Profiles, normative.AnalyzerPassportsSeedProfilesElem(in.Profile)) {
		admitters := quorumOf(holders[AuthorityAnalyzerAdmission], have)
		for _, ps := range s.Passports.Passports {
			v := ps.Versions
			ptr := func(x string) *string { return &x }
			kind := ev.AnalyzerPassportAdmittedV1AnalyzerKind(ps.AnalyzerKind)
			aid := ev.ObjectID(ps.AnalyzerID)
			d := ev.AnalyzerPassportAdmittedV1{PassportID: ev.ObjectID(ps.PassportID), Stage: ev.AnalyzerPassportAdmittedV1Stage(ps.Stage),
				TrustLevel: ps.TrustLevel, RecipeRef: ps.RecipeRef, DocumentID: ev.ObjectID(ps.DocumentID), AnalyzerID: &aid,
				AnalyzerKind: &kind, Title: ptr(ps.Title), Versions: ev.AnalyzerVersions{ItemRevision: ptr(v.ItemRevision),
					RecipeRef: v.RecipeRef, CameraConfig: ptr(v.CameraConfig), Calibration: ptr(v.Calibration),
					AnalyzerVersion: v.AnalyzerVersion, ThresholdProfile: ptr(v.ThresholdProfile), ContractVersion: v.ContractVersion,
					AppVersion: ptr(v.AppVersion)}}
			if v.RecipeRef != ps.RecipeRef {
				return spec, fmt.Errorf("затравка паспортов: %s: вектор версий не той карты контроля", ps.PassportID)
			}
			add(catalog.AnalyzerPassportAdmitted, "analyzer_passport:"+ps.PassportID, d, admitters...)
		}
	}
	return spec, nil
}

// quorumOf — ключи ГОСТ держателей полномочия, зарегистрированные в блоке
// (в prod ключей людей init не создаёт — кворум ставится церемонией, AD-33).
func quorumOf(persons []string, have map[string]bool) []string {
	var out []string
	for _, p := range persons {
		if ref := PersonaKeyRef(p); have[ref] && !slices.Contains(out, ref) {
			out = append(out, ref)
		}
	}
	return out
}

func setPtr(d map[string]any, k string, v *string) {
	if v != nil && *v != "" {
		d[k] = *v
	}
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// StampID — идентификатор клейма ASCII (object_id контракта): кириллица
// затравки («ОТК-01-ВК») транслитерируется — «OTK-01-VK».
func StampID(s string) string {
	tr := map[rune]string{'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ж': "ZH", 'З': "Z", 'И': "I", 'Й': "Y",
		'К': "K", 'Л': "L", 'М': "M", 'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U", 'Ф': "F", 'Х': "KH",
		'Ц': "TS", 'Ч': "CH", 'Ш': "SH", 'Щ': "SCH", 'Ы': "Y", 'Э': "E", 'Ю': "YU", 'Я': "YA"}
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if t, ok := tr[r]; ok {
			b.WriteString(t)
		} else if r < 128 {
			b.WriteRune(r)
		}
	}
	return b.String()
}
