package profiles

import (
	"crypto/mldsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.stargrave.org/gogost/v7/gost3410"

	dom "ant/internal/domain/signing"
)

// Ключи профилей gost и pq (contracts/crypto/README.md):
//   - ГОСТ Р 34.10-2012, 256 бит, id-tc26-gost-3410-12-256-paramSetA:
//     дайджест Стрибог-256(PAE), число e — little-endian дайджеста (обёртки
//     ReverseDigest GoGOST), подпись s‖r big-endian 64 байта, открытый ключ
//     X‖Y little-endian 64 байта, закрытый — 32 байта little-endian;
//   - ML-DSA-65 (FIPS 204, crypto/mldsa), контекст `ant/‹payloadType›`,
//     закрытый ключ — зерно 32 байта, открытый — 1952 байта.
//
// Закрытые ключи людей и устройств сервер не хранит (AD-11): эти функции
// используют demo-signer (демо-персоны), edge-агент (устройства), роль ant
// (ключи движка и шлюзов в томе ant), ant init (генезис) и тесты.

func curve() *gost3410.Curve { return gost3410.CurveIdtc26gost341012256paramSetA() }

// PrivateKey — закрытый ключ одного профиля (gost или pq) с его key_ref.
type PrivateKey struct {
	Ref     string
	Profile string
	gost    *gost3410.PrivateKey
	pq      *mldsa.PrivateKey
}

// Generate создаёт ключ профиля gost или pq.
func Generate(ref, profile string) (*PrivateKey, error) {
	if !dom.ValidKeyRef(ref) {
		return nil, fmt.Errorf("profiles: key_ref %q", ref)
	}
	switch profile {
	case dom.ProfileGost:
		k, err := gost3410.GenPrivateKey(curve(), rand.Reader)
		if err != nil {
			return nil, err
		}
		return &PrivateKey{Ref: ref, Profile: profile, gost: k}, nil
	case dom.ProfilePQ:
		k, err := mldsa.GenerateKey(mldsa.MLDSA65())
		if err != nil {
			return nil, err
		}
		return &PrivateKey{Ref: ref, Profile: profile, pq: k}, nil
	}
	return nil, fmt.Errorf("profiles: ключ профиля %q не бывает (hybrid — два ключа gost и pq)", profile)
}

// FromSecret восстанавливает ключ из секрета: ГОСТ — 32 байта little-endian,
// ML-DSA — зерно 32 байта.
func FromSecret(ref, profile string, secret []byte) (*PrivateKey, error) {
	switch profile {
	case dom.ProfileGost:
		k, err := gost3410.NewPrivateKeyLE(curve(), secret)
		if err != nil {
			return nil, err
		}
		return &PrivateKey{Ref: ref, Profile: profile, gost: k}, nil
	case dom.ProfilePQ:
		k, err := mldsa.NewPrivateKey(mldsa.MLDSA65(), secret)
		if err != nil {
			return nil, err
		}
		return &PrivateKey{Ref: ref, Profile: profile, pq: k}, nil
	}
	return nil, fmt.Errorf("profiles: профиль %q", profile)
}

// Secret — закрытая часть ключа (для записи в том 0600).
func (k *PrivateKey) Secret() []byte {
	if k.gost != nil {
		return k.gost.RawLE()
	}
	return k.pq.Bytes()
}

// Public — открытый ключ в кодировке профиля.
func (k *PrivateKey) Public() []byte {
	if k.gost != nil {
		p, _ := k.gost.PublicKey()
		return p.RawLE()
	}
	return k.pq.PublicKey().Bytes()
}

// PublicB64 — открытый ключ в base64 (public_key_b64 акта регистрации).
func (k *PrivateKey) PublicB64() string { return base64.StdEncoding.EncodeToString(k.Public()) }

// Fingerprint — отпечаток открытого ключа streebog256:‹hex› (в сводке акта —
// «сверьте отпечаток», AD-11).
func (k *PrivateKey) Fingerprint() string { return dom.Digest(k.Public()) }

// Algorithm — алгоритм ключа.
func (k *PrivateKey) Algorithm() string { return dom.AlgorithmOf(k.Profile) }

// SignPAE подписывает PAE пакета класса payloadType.
func (k *PrivateKey) SignPAE(payloadType string, pae []byte) ([]byte, error) {
	if k.gost != nil {
		return (&gost3410.PrivateKeyReverseDigest{Prv: k.gost}).Sign(rand.Reader, dom.Hash(pae), nil)
	}
	return k.pq.Sign(nil, pae, &mldsa.Options{Context: "ant/" + payloadType})
}

// Verify проверяет подпись sig над PAE открытым ключом pub профиля profile.
func Verify(profile string, pub []byte, payloadType string, pae, sig []byte) bool {
	switch profile {
	case dom.ProfileGost:
		if len(pub) != dom.GostPublicKeySize || len(sig) != dom.GostSignatureSize {
			return false
		}
		p, err := gost3410.NewPublicKeyLE(curve(), pub)
		if err != nil {
			return false
		}
		ok, err := gost3410.PublicKeyReverseDigest{Pub: p}.VerifyDigest(dom.Hash(pae), sig)
		return err == nil && ok
	case dom.ProfilePQ:
		p, err := mldsa.NewPublicKey(mldsa.MLDSA65(), pub)
		if err != nil {
			return false
		}
		return mldsa.Verify(p, pae, sig, &mldsa.Options{Context: "ant/" + payloadType}) == nil
	}
	return false
}

// keyFile — файл закрытого ключа в томе (0600): key_ref, профиль, секрет hex.
type keyFile struct {
	KeyRef    string `json:"key_ref"`
	Profile   string `json:"profile"`
	SecretHex string `json:"secret_hex"`
	// PublicB64, Fingerprint — справочно (сверка отпечатка при выдаче).
	PublicB64   string `json:"public_key_b64"`
	Fingerprint string `json:"fingerprint"`
}

// FileName — имя файла ключа в томе: ‹key_id›@‹версия›.key.json.
func FileName(ref string) string { return strings.ReplaceAll(ref, ":", "_") + ".key.json" }

// Save записывает ключ в каталог dir (0600); существующий файл не перезаписывается.
func Save(dir string, k *PrivateKey) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(keyFile{KeyRef: k.Ref, Profile: k.Profile, SecretHex: hex.EncodeToString(k.Secret()),
		PublicB64: k.PublicB64(), Fingerprint: k.Fingerprint()}, "", "  ")
	f, err := os.OpenFile(filepath.Join(dir, FileName(k.Ref)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// Load читает файл ключа.
func Load(path string) (*PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f keyFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	s, err := hex.DecodeString(f.SecretHex)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return FromSecret(f.KeyRef, f.Profile, s)
}

// Keyring — набор закрытых ключей процесса (том ant, demo-signer, edge-агент).
type Keyring struct {
	keys map[string]*PrivateKey
}

// NewKeyring — набор из ключей.
func NewKeyring(keys ...*PrivateKey) *Keyring {
	kr := &Keyring{keys: map[string]*PrivateKey{}}
	for _, k := range keys {
		kr.keys[k.Ref] = k
	}
	return kr
}

// LoadDir читает все *.key.json каталога dir; нет каталога — пустой набор.
func LoadDir(dir string) (*Keyring, error) {
	kr := NewKeyring()
	ents, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return kr, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".key.json") {
			continue
		}
		k, err := Load(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		kr.keys[k.Ref] = k
	}
	return kr, nil
}

// Ensure — ключ ref профиля profile из набора; нет — создаётся и сохраняется в dir.
func (kr *Keyring) Ensure(dir, ref, profile string) (*PrivateKey, error) {
	if k, ok := kr.keys[ref]; ok {
		return k, nil
	}
	k, err := Generate(ref, profile)
	if err != nil {
		return nil, err
	}
	if dir != "" {
		if err := Save(dir, k); err != nil {
			return nil, err
		}
	}
	kr.keys[ref] = k
	return k, nil
}

// Key — ключ по key_ref.
func (kr *Keyring) Key(ref string) (*PrivateKey, bool) {
	k, ok := kr.keys[ref]
	return k, ok
}

// Refs — key_ref всех ключей по порядку.
func (kr *Keyring) Refs() []string {
	out := make([]string, 0, len(kr.keys))
	for r := range kr.keys {
		out = append(out, r)
	}
	slices.Sort(out)
	return out
}
