package vision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"time"

	"go.yaml.in/yaml/v3"

	appjournal "ant/internal/application/journal"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/constants"
	ev "ant/internal/contracts/events"
	jc "ant/internal/contracts/journal"
	"ant/internal/contracts/normative"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/vision"
)

// Файлы затравки модуля vision в normative/ (пути от корня репозитория).
const (
	PassportsSeedFile = "normative/vision/analyzer-passports.v1.yaml"
	IllustrationsFile = "normative/vision/illustrations.v1.yaml"
)

// decodeYAML — YAML затравки в сгенерированный тип через JSON (проверки
// обязательных полей делает сгенерированный UnmarshalJSON).
func decodeYAML[T any](fsys fs.FS, name string) (T, error) {
	var out T
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return out, err
	}
	var raw any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return out, fmt.Errorf("%s: %w", name, err)
	}
	if err := remarshal(raw, &out); err != nil {
		return out, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// LoadPassportsSeed — стартовые паспорта демо (normative/vision/analyzer-passports.v1.yaml).
func LoadPassportsSeed(fsys fs.FS) (normative.AnalyzerPassportsSeed, error) {
	return decodeYAML[normative.AnalyzerPassportsSeed](fsys, PassportsSeedFile)
}

// LoadIllustrations — каталог иллюстраций (normative/vision/illustrations.v1.yaml).
func LoadIllustrations(fsys fs.FS) (normative.IllustrationsCatalog, error) {
	return decodeYAML[normative.IllustrationsCatalog](fsys, IllustrationsFile)
}

// SeedConfig — параметры записи затравки.
type SeedConfig struct {
	// Profile — профиль окружения (demo | fixtures | load | prod).
	Profile     string
	DomainBuild string
	Partitions  int
	// Now — InfraClock для received_at; nil — time.Now.
	Now func() time.Time
}

// SeedEventID — event_id записи допуска из затравки: UUIDv5 от версии
// затравки и паспорта. Повторный запуск migrate даёт дубль, а не вторую запись.
func SeedEventID(version int, passportID string) string {
	return kernel.UUIDv5(constants.NsAnt, "vision.seed/v"+strconv.Itoa(version)+"/"+passportID)
}

// SeedPassports записывает стартовые паспорта допуска (analyzer.passport.admitted,
// происхождение genesis, AD-33) в профилях затравки; в prod — никогда.
// Возвращает число новых записей: уже записанные — дубль, пропускаются.
// Иначе камеры демо работают на уровне доверия 0 (только запись, AD-29).
// Полный маршрут допуска (экзамен, тень, пилот, протокол) — эпик 40.
func SeedPassports(ctx context.Context, j appjournal.JournalStore, seed normative.AnalyzerPassportsSeed, cfg SeedConfig) (int, error) {
	if cfg.Profile == "prod" || !slices.Contains(seed.Profiles, normative.AnalyzerPassportsSeedProfilesElem(cfg.Profile)) {
		return 0, nil
	}
	at, err := time.Parse(time.RFC3339Nano, seed.AdmittedAt)
	if err != nil {
		return 0, fmt.Errorf("затравка паспортов: admitted_at: %w", err)
	}
	if cfg.Partitions <= 0 {
		cfg.Partitions = 1
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	n := 0
	for _, ps := range seed.Passports {
		v := ps.Versions
		vers := dom.Versions{ItemRevision: v.ItemRevision, RecipeRef: v.RecipeRef, CameraConfig: v.CameraConfig, Calibration: v.Calibration,
			AnalyzerVersion: v.AnalyzerVersion, ThresholdProfile: v.ThresholdProfile, ContractVersion: v.ContractVersion, AppVersion: v.AppVersion}
		if miss := vers.Missing(); len(miss) > 0 || vers.RecipeRef != ps.RecipeRef {
			return n, fmt.Errorf("затравка паспортов: %s: вектор версий неполон %v или не той карты контроля", ps.PassportID, miss)
		}
		kind := ev.AnalyzerPassportAdmittedV1AnalyzerKind(ps.AnalyzerKind)
		title := ps.Title
		aid := ev.ObjectID(ps.AnalyzerID)
		d := ev.AnalyzerPassportAdmittedV1{PassportID: ev.ObjectID(ps.PassportID), Stage: ev.AnalyzerPassportAdmittedV1Stage(ps.Stage),
			TrustLevel: ps.TrustLevel, RecipeRef: ps.RecipeRef, Versions: vers.Contract(), DocumentID: ev.ObjectID(ps.DocumentID),
			AnalyzerID: &aid, AnalyzerKind: &kind, Title: &title}
		id := SeedEventID(seed.Version, ps.PassportID)
		stream := dom.Stream(ps.PassportID)
		p, err := pending(entry{ID: id, Type: catalog.AnalyzerPassportAdmitted, Stream: stream, Data: d, OccurredAt: at,
			ReceivedAt: now(), SourceID: SourceGenesis, Signer: "genesis@1", Provenance: jc.JournalEntryProvenanceClassGenesis,
			Command: map[string]any{"command_id": id, "basis_seq": 0, "guard_streams": []string{stream}, "policy_seq": 0, "signature_level": 0}},
			cfg.DomainBuild, cfg.Partitions)
		if err != nil {
			return n, err
		}
		_, err = j.Append(ctx, appjournal.AppendRequest{Batch: []appjournal.Pending{p}})
		switch {
		case errors.Is(err, appjournal.ErrDuplicate):
			continue
		case err != nil:
			return n, fmt.Errorf("затравка паспортов: %s: %w", ps.PassportID, err)
		}
		n++
	}
	return n, nil
}

// remarshal — значение YAML (map[string]any) в тип через JSON.
func remarshal(raw any, out any) error {
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
