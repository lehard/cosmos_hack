package signing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	ingest "ant/internal/application/ingest"
	app "ant/internal/application/signing"
	dom "ant/internal/domain/signing"
)

// IngestKeys — адаптер порта приёма application/ingest.KeyRegistry (FR-26,
// FR-70, AD-11) над реестром ключей модуля signing: источник
// зарегистрирован, если у него есть ключ по акту (устройство — актом ввода,
// шлюз и агент токена — актом регистрации); ключ отозван или заменён
// ротацией — «отозван»; открытая часть недоступна — «не проверяемо».
// Источник ключа человека — его агент токена: source_id = псевдоним.
type IngestKeys struct {
	Registry *app.Registry
	// Sources — источники, допустимые без собственного ключа (шлюз ant-ingest и т. п.).
	Sources []string
}

var _ ingest.KeyRegistry = IngestKeys{}

// Source — зарегистрирован ли источник.
func (k IngestKeys) Source(ctx context.Context, sourceID string) error {
	for _, s := range k.Sources {
		if s == sourceID {
			return nil
		}
	}
	keys, _, _, err := k.Registry.Snapshot(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys.Keys() {
		if key.SubjectID == sourceID && key.Revoked == nil {
			return nil
		}
	}
	return ingest.ErrUnknownSource
}

// Key — ключ по key_id@версия.
func (k IngestKeys) Key(ctx context.Context, ref string) (ingest.KeyInfo, error) {
	keys, _, incomplete, err := k.Registry.Snapshot(ctx)
	if err != nil {
		return ingest.KeyInfo{}, err
	}
	key, ok := keys.Key(ref)
	switch {
	case !ok && incomplete:
		return ingest.KeyInfo{}, fmt.Errorf("%w: реестр ключей прочитан не полностью", app.ErrKeyUnavailable)
	case !ok:
		return ingest.KeyInfo{}, ingest.ErrUnknownKey
	case len(key.PublicKey()) == 0:
		return ingest.KeyInfo{}, fmt.Errorf("%w: открытой части ключа %s нет", app.ErrKeyUnavailable, ref)
	}
	return ingest.KeyInfo{KeyRef: ref, SourceID: key.SubjectID, Provenance: key.Provenance,
		Revoked: key.Revoked != nil || key.RotatedBy != ""}, nil
}

// Bootstrap — затравка реестра: открытые ключи, принятые до журнала (блок
// генезиса эпика 05; до него — файл открытых ключей демо-персон и системных
// ключей, который пишут ant и demo-signer). Закрытых ключей в нём нет.
type Bootstrap struct {
	Version int               `json:"version"`
	Keys    []BootstrapKey    `json:"keys"`
	Note    string            `json:"note,omitempty"`
	Extra   map[string]string `json:"extra,omitempty"`
}

// BootstrapKey — открытый ключ затравки и его класс доверия.
type BootstrapKey struct {
	app.RegistrationData
	Provenance string `json:"provenance"`
}

// LoadBootstrap читает файл затравки; нет файла — пусто.
func LoadBootstrap(path string) ([]dom.Registration, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f Bootstrap
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	out := make([]dom.Registration, 0, len(f.Keys))
	for _, k := range f.Keys {
		g, err := k.Registration()
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", path, k.KeyRef, err)
		}
		if g.Fingerprint != "" && len(g.PublicKey()) > 0 && g.Fingerprint != dom.Digest(g.PublicKey()) {
			return nil, fmt.Errorf("%s: отпечаток ключа %s не совпадает", path, k.KeyRef)
		}
		g.Provenance = k.Provenance
		out = append(out, g)
	}
	return out, nil
}

// MergeBootstrap дописывает ключи в файл затравки (0644 — только открытые
// части): существующие key_ref не перезаписываются.
func MergeBootstrap(path string, keys []BootstrapKey) error {
	var f Bootstrap
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &f); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	f.Version = 1
	f.Note = "Открытые ключи затравки реестра (генезис демо до эпика 05). Закрытых ключей здесь нет."
	have := map[string]bool{}
	for _, k := range f.Keys {
		have[k.KeyRef] = true
	}
	for _, k := range keys {
		if !have[k.KeyRef] {
			f.Keys = append(f.Keys, k)
			have[k.KeyRef] = true
		}
	}
	b, _ := json.MarshalIndent(f, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// BootstrapOf — запись затравки для открытого ключа.
func BootstrapOf(ref, kind, subject, profile, pubB64, fingerprint, provenance string, classes []string, validFrom time.Time) BootstrapKey {
	return BootstrapKey{RegistrationData: app.RegistrationData{KeyRef: ref, SubjectKind: kind, SubjectID: subject, ProfileID: profile,
		Algorithm: dom.AlgorithmOf(profile), PublicKeyB64: pubB64, Fingerprint: fingerprint, PayloadClasses: classes,
		SubjectConfirmation: dom.ConfirmGenesis, ValidFrom: validFrom.UTC().Format(app.TimeLayout)}, Provenance: provenance}
}
