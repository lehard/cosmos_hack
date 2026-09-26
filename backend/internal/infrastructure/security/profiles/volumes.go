package profiles

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	dom "ant/internal/domain/signing"
)

// Ключи в томах (AD-11, AD-33; эпик 05 — ant init). Форматы файлов — те, что
// читают процессы-владельцы ключей:
//   - ant, demo-signer, агент токена — ‹key_ref›.key.json (Save/Load);
//   - хранитель и верификатор (эпик 29) — ‹dir›/‹субъект›.gost и
//     ‹dir›/‹субъект›.pq: hex закрытого ключа ГОСТ (LE) и зерна ML-DSA-65;
//     ключи ‹субъект›-gost@1 и ‹субъект›-pq@1 (профиль hybrid);
//   - edge-агент — ‹source_id›.key: hex закрытого ключа ГОСТ (LE), ключ
//     device-‹source_id›@1.
// Все файлы закрытых ключей — 0400; существующие не перезаписываются: повтор
// ant init и порядок «init / keeper -init» не важны.

// HybridRefs — key_ref ключей субъекта профиля hybrid (соглашение томов keeper
// и verifier): ‹субъект›-gost@1, ‹субъект›-pq@1.
func HybridRefs(subject string) []string { return []string{subject + "-gost@1", subject + "-pq@1"} }

// EnsureHybrid — ключи субъекта hybrid в dir (‹субъект›.gost и ‹субъект›.pq):
// нет — создаются (0400), есть — читаются.
func EnsureHybrid(dir, subject string) (gost, pq *PrivateKey, err error) {
	refs := HybridRefs(subject)
	if gost, err = ensureRaw(filepath.Join(dir, subject+".gost"), refs[0], dom.ProfileGost); err != nil {
		return nil, nil, err
	}
	if pq, err = ensureRaw(filepath.Join(dir, subject+".pq"), refs[1], dom.ProfilePQ); err != nil {
		return nil, nil, err
	}
	return gost, pq, nil
}

// EnsureDeviceKey — ключ устройства для edge-агента: ‹dir›/‹source_id›.key
// (hex ГОСТ LE, 0400), key_ref device-‹source_id›@1.
func EnsureDeviceKey(dir, sourceID string) (*PrivateKey, error) {
	return ensureRaw(filepath.Join(dir, sourceID+".key"), DeviceRef(sourceID), dom.ProfileGost)
}

// DeviceRef — key_ref ключа устройства (соглашение edge-агента).
func DeviceRef(sourceID string) string { return "device-" + strings.ToLower(sourceID) + "@1" }

// ensureRaw — файл «hex секрета» профиля profile: читается или создаётся.
func ensureRaw(path, ref, profile string) (*PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err == nil {
		raw, err := hex.DecodeString(strings.TrimSpace(string(b)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		return FromSecret(ref, profile, raw)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	k, err := Generate(ref, profile)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o400)
	if err != nil {
		return nil, err
	}
	if _, err := f.WriteString(hex.EncodeToString(k.Secret()) + "\n"); err != nil {
		_ = f.Close()
		return nil, err
	}
	return k, f.Close()
}

// PublicSet — открытые ключи по key_ref (порт PublicKeys над набором в памяти:
// ключи якоря и блока генезиса, тесты).
type PublicSet map[string]PublicKey

// PublicKey — открытый ключ набора.
type PublicKey struct {
	Profile string
	Key     []byte
}

// PublicKey — профиль и байты ключа ref; нет — ErrKeyUnavailable.
func (p PublicSet) PublicKey(_ context.Context, ref string) (string, []byte, error) {
	k, ok := p[ref]
	if !ok {
		return "", nil, ErrKeyUnavailable
	}
	return k.Profile, k.Key, nil
}

// Anchor — открытый ключ в файле trust-anchors (формат эпика 29).
type Anchor struct {
	KeyID       string `json:"keyid"`
	Subject     string `json:"subject"`
	Profile     string `json:"profile"`
	PublicKey   string `json:"public_key_b64"`
	Fingerprint string `json:"fingerprint"`
}

// TrustAnchors — файл trust-anchors (AD-33): отпечатки якоря, хранителя и
// верификатора, отпечаток генезиса, ожидаемые профили. Лежит в томе keeper,
// в verifier монтируется только он (копия — в томе verifier). В промышленной
// эксплуатации его подписывает Аудитор ИБ и передаёт на отчуждаемом носителе.
type TrustAnchors struct {
	FormatVersion int               `json:"format_version"`
	Keys          []Anchor          `json:"keys"`
	Profiles      map[string]string `json:"object_profiles"`
	// AnchorFingerprint — общий отпечаток ключей якоря (заголовок генезиса).
	AnchorFingerprint string `json:"anchor_fingerprint,omitempty"`
	GenesisDigest     string `json:"genesis_digest,omitempty"`
	Note              string `json:"note,omitempty"`
}

// AnchorOf — запись trust-anchors для ключа.
func AnchorOf(subject string, k *PrivateKey) Anchor {
	return Anchor{KeyID: k.Ref, Subject: subject, Profile: k.Profile, PublicKey: k.PublicB64(), Fingerprint: k.Fingerprint()}
}

// AnchorFromPublic — запись trust-anchors для открытого ключа.
func AnchorFromPublic(subject, ref, profile string, pub []byte) Anchor {
	return Anchor{KeyID: ref, Subject: subject, Profile: profile, PublicKey: base64.StdEncoding.EncodeToString(pub), Fingerprint: dom.Digest(pub)}
}

// ErrOtherGenesis — trust-anchors уже закрепляет другой генезис (другой домен доверия).
var ErrOtherGenesis = errors.New("trust-anchors закрепляет другой генезис — это другой домен доверия (сбросьте тома или окружение)")

// WriteTrustAnchors дописывает в trust-anchors (path) ключи, отпечаток якоря и
// генезиса: нет файла — создаёт; в файле (например, от keeper -init эпика 29)
// нет генезиса — дополняет; тот же генезис — ничего; другой — ErrOtherGenesis.
// Запись атомарна (tmp + rename), файл 0444.
func WriteTrustAnchors(path string, add TrustAnchors) error {
	var cur TrustAnchors
	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(b, &cur); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	case errors.Is(err, os.ErrNotExist):
		cur = TrustAnchors{FormatVersion: 1}
	default:
		return err
	}
	if cur.GenesisDigest != "" {
		if cur.GenesisDigest == add.GenesisDigest {
			return nil
		}
		return fmt.Errorf("%s: %w", path, ErrOtherGenesis)
	}
	for _, k := range add.Keys {
		i := slices.IndexFunc(cur.Keys, func(x Anchor) bool { return x.KeyID == k.KeyID })
		switch {
		case i < 0:
			cur.Keys = append(cur.Keys, k)
		case cur.Keys[i].Fingerprint != k.Fingerprint:
			return fmt.Errorf("%s: ключ %s в файле другой — тома не от одной установки", path, k.KeyID)
		}
	}
	if cur.Profiles == nil {
		cur.Profiles = map[string]string{}
	}
	for c, p := range add.Profiles {
		if _, ok := cur.Profiles[c]; !ok {
			cur.Profiles[c] = p
		}
	}
	cur.AnchorFingerprint, cur.GenesisDigest = add.AnchorFingerprint, add.GenesisDigest
	if add.Note != "" {
		cur.Note = add.Note
	}
	out, _ := json.MarshalIndent(cur, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	if err := os.WriteFile(tmp, append(out, '\n'), 0o444); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadTrustAnchors читает trust-anchors; нет файла — пусто.
func LoadTrustAnchors(path string) (TrustAnchors, error) {
	var t TrustAnchors
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return t, nil
	}
	if err != nil {
		return t, err
	}
	return t, json.Unmarshal(b, &t)
}
