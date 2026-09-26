package vision

import (
	"strings"

	ev "ant/internal/contracts/events"
)

// Unknown — явное «неизвестно» в составляющей вектора версий («Соглашения/
// Неизвестность»): источник её не сообщил; значение не выдумывается.
const Unknown = "unknown"

// Versions — вектор версий наблюдения анализатора (AD-29, FR-98): все восемь
// составляющих контура контроля. Версия контракта ≠ версия анализатора ≠
// версия приложения (AD-20, кейс §4.7).
type Versions struct {
	ItemRevision     string `json:"item_revision"`
	RecipeRef        string `json:"recipe_ref"`
	CameraConfig     string `json:"camera_config"`
	Calibration      string `json:"calibration"`
	AnalyzerVersion  string `json:"analyzer_version"`
	ThresholdProfile string `json:"threshold_profile"`
	ContractVersion  string `json:"contract_version"`
	AppVersion       string `json:"app_version"`
}

// Components — имена составляющих в порядке AD-29 (ключи контракта
// analyzer_versions).
var Components = []string{"item_revision", "recipe_ref", "camera_config", "calibration",
	"analyzer_version", "threshold_profile", "contract_version", "app_version"}

// fields — составляющие вектора по именам Components.
func (v *Versions) fields() []*string {
	return []*string{&v.ItemRevision, &v.RecipeRef, &v.CameraConfig, &v.Calibration,
		&v.AnalyzerVersion, &v.ThresholdProfile, &v.ContractVersion, &v.AppVersion}
}

// Map — вектор как словарь «составляющая → значение» (формы ответов, карточка).
func (v Versions) Map() map[string]string {
	out := make(map[string]string, len(Components))
	for i, p := range v.fields() {
		out[Components[i]] = *p
	}
	return out
}

// Missing — составляющие, которых источник не сообщил (пусто или unknown).
func (v Versions) Missing() []string {
	var out []string
	for i, p := range v.fields() {
		if s := strings.TrimSpace(*p); s == "" || s == Unknown {
			out = append(out, Components[i])
		}
	}
	return out
}

// Complete — вектор, где несообщённые составляющие явно unknown, и их список
// (AD-29: вектор версий на каждом наблюдении). Наблюдение с неполным вектором
// не теряется (отсутствие результата ≠ годность), но неизвестная версия
// анализатора или карты контроля не совпадёт ни с одним паспортом — уровень
// доверия 0, контроль ручной (ограничивать безопаснее, AD-27).
func (v Versions) Complete() (Versions, []string) {
	miss := v.Missing()
	for _, p := range v.fields() {
		if strings.TrimSpace(*p) == "" {
			*p = Unknown
		}
	}
	return v, miss
}

// Contract — вектор в типе контракта событий (analyzer_versions).
func (v Versions) Contract() ev.AnalyzerVersions {
	c, _ := v.Complete()
	ptr := func(s string) *string { return &s }
	return ev.AnalyzerVersions{ItemRevision: ptr(c.ItemRevision), RecipeRef: c.RecipeRef, CameraConfig: ptr(c.CameraConfig),
		Calibration: ptr(c.Calibration), AnalyzerVersion: c.AnalyzerVersion, ThresholdProfile: ptr(c.ThresholdProfile),
		ContractVersion: c.ContractVersion, AppVersion: ptr(c.AppVersion)}
}

// FromContract — вектор из типа контракта; отсутствующие составляющие — пусто.
func FromContract(c ev.AnalyzerVersions) Versions {
	s := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	return Versions{ItemRevision: s(c.ItemRevision), RecipeRef: c.RecipeRef, CameraConfig: s(c.CameraConfig),
		Calibration: s(c.Calibration), AnalyzerVersion: c.AnalyzerVersion, ThresholdProfile: s(c.ThresholdProfile),
		ContractVersion: c.ContractVersion, AppVersion: s(c.AppVersion)}
}

// RecipeRef — ссылка на карту контроля `‹id›@‹версия›`.
func RecipeRef(id, version string) string {
	id, version = strings.TrimSpace(id), strings.TrimSpace(version)
	if id == "" {
		return ""
	}
	if version == "" {
		version = Unknown
	}
	return id + "@" + version
}
