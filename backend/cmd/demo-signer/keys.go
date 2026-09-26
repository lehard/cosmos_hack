package main

import (
	"errors"
	"os"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
	storesigning "ant/internal/infrastructure/storage/signing"
)

// Persona — демо-персона: псевдоним из стартовой политики и её ключи
// класса scenario (ГОСТ — ‹псевдоним›@1, ML-DSA-65 — ‹псевдоним›-pq@1).
type Persona struct {
	PersonaID string `json:"persona_id"`
	PersonID  string `json:"person_id"`
	RoleID    string `json:"role_id"`
	KeyRef    string `json:"key_ref"`
}

// DefaultPersonas — персоны стартовой политики на случай, если файла нет.
var DefaultPersonas = []Persona{{PersonaID: "INS-01", RoleID: "quality_inspector"}, {PersonaID: "INS-02", RoleID: "quality_inspector"},
	{PersonaID: "HQC-01", RoleID: "head_of_qc"}, {PersonaID: "FOR-WC", RoleID: "site_foreman"}, {PersonaID: "TEC-01", RoleID: "technologist"},
	{PersonaID: "PM-01", RoleID: "production_manager"}, {PersonaID: "ADM-01", RoleID: "administrator"}, {PersonaID: "AUD-01", RoleID: "security_auditor"}}

// Personas — персоны из normative/policy/policy.v1.yaml (persons).
func Personas(path string) ([]Persona, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fill(slices.Clone(DefaultPersonas)), nil
	}
	if err != nil {
		return nil, err
	}
	var p struct {
		Persons []struct {
			ID    string `yaml:"id"`
			Roles []struct {
				Role string `yaml:"role"`
			} `yaml:"roles"`
		} `yaml:"persons"`
	}
	if err := yaml.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	var out []Persona
	for _, x := range p.Persons {
		role := ""
		if len(x.Roles) > 0 {
			role = x.Roles[0].Role
		}
		out = append(out, Persona{PersonaID: x.ID, RoleID: role})
	}
	return fill(out), nil
}

func fill(ps []Persona) []Persona {
	for i := range ps {
		ps[i].PersonID = ps[i].PersonaID
		ps[i].KeyRef = GostRef(ps[i].PersonaID)
	}
	return ps
}

// GostRef, PQRef — ключи персоны (соглашение демо-набора, AD-33).
func GostRef(persona string) string { return strings.ToLower(persona) + "@1" }

// PQRef — ключ ML-DSA-65 персоны (подписанты актов — hybrid, AD-32).
func PQRef(persona string) string { return strings.ToLower(persona) + "-pq@1" }

// KeySet — ключи персон.
type KeySet struct {
	Personas []Persona
	Ring     *profiles.Keyring
}

// Persona — персона по псевдониму.
func (k *KeySet) Persona(id string) (Persona, bool) {
	for _, p := range k.Personas {
		if p.PersonaID == id {
			return p, true
		}
	}
	return Persona{}, false
}

// EnsureKeys — ключи персон из тома dir; недостающие создаются (0600).
func EnsureKeys(dir string, ps []Persona) (*KeySet, error) {
	ring, err := profiles.LoadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		if _, err := ring.Ensure(dir, GostRef(p.PersonaID), dom.ProfileGost); err != nil {
			return nil, err
		}
		if _, err := ring.Ensure(dir, PQRef(p.PersonaID), dom.ProfilePQ); err != nil {
			return nil, err
		}
	}
	return &KeySet{Personas: ps, Ring: ring}, nil
}

// LoadKeys — ключи персон из тома dir: их создаёт и регистрирует блоком
// генезиса ant init (эпик 05, AD-33); ключа нет — ошибка (свой ключ,
// которого нет в генезисе, ant всё равно не примет).
func LoadKeys(dir string, ps []Persona) (*KeySet, error) {
	ring, err := profiles.LoadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		for _, ref := range []string{GostRef(p.PersonaID), PQRef(p.PersonaID)} {
			if _, ok := ring.Key(ref); !ok {
				return nil, errors.New("ключа " + ref + " в томе " + dir + " нет — сначала ant init (генезис)")
			}
		}
	}
	return &KeySet{Personas: ps, Ring: ring}, nil
}

// PersonaClasses — классы пакетов ключей демо-персон.
var PersonaClasses = []string{dom.ClassEvent, dom.ClassDocumentSignature, dom.ClassPaperAttestation, dom.ClassShiftReport, dom.ClassKeyAct}

// WriteBootstrap — открытые ключи персон в файл затравки реестра (субъект —
// demo_persona, класс scenario, AD-11, AD-26). Устарело с эпиком 05: ключи
// регистрирует блок генезиса; файл остаётся для тестов без журнала.
func WriteBootstrap(path string, set *KeySet) error {
	var out []storesigning.BootstrapKey
	for _, p := range set.Personas {
		for _, ref := range []string{GostRef(p.PersonaID), PQRef(p.PersonaID)} {
			k, _ := set.Ring.Key(ref)
			out = append(out, storesigning.BootstrapOf(ref, dom.SubjectDemoPersona, p.PersonaID, k.Profile, k.PublicB64(), k.Fingerprint(),
				dom.ProvScenario, PersonaClasses, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
		}
	}
	return storesigning.MergeBootstrap(path, out)
}
