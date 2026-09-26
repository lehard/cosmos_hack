package agent

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"

	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// Хранилище ключа под PIN (AD-14, Д-72): ключ человека загружается файлом и
// хранится только зашифрованным; ключ шифрования выводится из PIN функцией
// argon2id, шифр — AES-256-GCM, открытые сведения (владелец, ключи, класс
// хранения) входят в AAD — подменить их, не зная PIN, нельзя.

// Ошибки хранилища.
var (
	// ErrPIN — неверный PIN (signing.pin_wrong).
	ErrPIN = errors.New("signing.pin_wrong")
	// ErrKeyFile — файл ключа не читается или ключ не сходится с открытым.
	ErrKeyFile = errors.New("agent: файл ключа не принят")
)

// MinPIN — минимальная длина PIN.
const MinPIN = 4

// FormatVersion — версия формата хранилища.
const FormatVersion = 1

// KeyFile — файл ключа персоны (формат infrastructure/security/profiles.Save:
// демо-ключи из .demo-keys/token-agent/).
type KeyFile struct {
	KeyRef       string `json:"key_ref"`
	Profile      string `json:"profile"`
	SecretHex    string `json:"secret_hex"`
	PublicKeyB64 string `json:"public_key_b64,omitempty"`
	Fingerprint  string `json:"fingerprint,omitempty"`
}

// KeyInfo — открытые сведения о ключе в хранилище.
type KeyInfo struct {
	KeyRef      string `json:"key_ref"`
	Profile     string `json:"profile"`
	Fingerprint string `json:"fingerprint"`
}

// ParseKeyFile — файл ключа: закрытый ключ и его открытые сведения. Если в
// файле есть открытый ключ — он обязан совпасть с выведенным из закрытого.
func ParseKeyFile(raw []byte) (KeyFile, *profiles.PrivateKey, error) {
	var f KeyFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return f, nil, fmt.Errorf("%w: %v", ErrKeyFile, err)
	}
	secret, err := hex.DecodeString(f.SecretHex)
	if err != nil || len(secret) == 0 {
		return f, nil, fmt.Errorf("%w: нет закрытого ключа (secret_hex)", ErrKeyFile)
	}
	k, err := profiles.FromSecret(f.KeyRef, f.Profile, secret)
	if err != nil {
		return f, nil, fmt.Errorf("%w: %v", ErrKeyFile, err)
	}
	if f.PublicKeyB64 != "" && f.PublicKeyB64 != k.PublicB64() {
		return f, nil, fmt.Errorf("%w: открытый ключ в файле не сходится с закрытым", ErrKeyFile)
	}
	f.PublicKeyB64, f.Fingerprint = k.PublicB64(), k.Fingerprint()
	return f, k, nil
}

// PersonOf — субъект ключа по соглашению демо-набора (эпик 05):
// ‹псевдоним›-ta@1, ‹псевдоним›-ta-pq@1 → ПСЕВДОНИМ.
func PersonOf(keyRef string) string {
	id, _, _ := strings.Cut(keyRef, "@")
	for _, suf := range []string{"-ta-pq", "-ta", "-pq"} {
		if s, ok := strings.CutSuffix(id, suf); ok {
			id = s
			break
		}
	}
	return strings.ToUpper(id)
}

// Bundle — открытое содержимое хранилища: ключи одного человека.
type Bundle struct {
	PersonID string    `json:"person_id"`
	Keys     []KeyFile `json:"keys"`
}

// Private — закрытые ключи набора.
func (b Bundle) Private() ([]*profiles.PrivateKey, error) {
	out := make([]*profiles.PrivateKey, 0, len(b.Keys))
	for _, f := range b.Keys {
		raw, _ := json.Marshal(f)
		_, k, err := ParseKeyFile(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, nil
}

// KDF — параметры argon2id.
type KDF struct {
	Name      string `json:"name"`
	Time      uint32 `json:"time"`
	MemoryKiB uint32 `json:"memory_kib"`
	Threads   uint8  `json:"threads"`
	SaltB64   string `json:"salt_b64"`
}

// DefaultKDF — параметры по умолчанию: 2 прохода, 32 МиБ, один поток —
// разблокировка в WASM за 1–3 с (RFC 9106, второй рекомендованный набор —
// 64 МиБ; снижено ради планшетов).
func DefaultKDF() KDF { return KDF{Name: "argon2id", Time: 2, MemoryKiB: 32 * 1024, Threads: 1} }

// Sealed — хранилище ключа под PIN.
type Sealed struct {
	FormatVersion int       `json:"format_version"`
	PersonID      string    `json:"person_id"`
	Keys          []KeyInfo `json:"keys"`
	// KeyStorage — hardware_token | software_browser; StorageVariant — extension | page.
	KeyStorage     string `json:"key_storage"`
	StorageVariant string `json:"storage_variant,omitempty"`
	SealedAt       string `json:"sealed_at"`
	KDF            KDF    `json:"kdf"`
	NonceB64       string `json:"nonce_b64"`
	CipherB64      string `json:"ciphertext_b64"`
}

// KeyRefs — ключи хранилища.
func (s Sealed) KeyRefs() []string {
	out := make([]string, 0, len(s.Keys))
	for _, k := range s.Keys {
		out = append(out, k.KeyRef)
	}
	return out
}

// aad — открытые сведения, связанные с шифртекстом.
func (s Sealed) aad() []byte {
	s.NonceB64, s.CipherB64 = "", ""
	b, _ := dom.CanonicalOf(s)
	return b
}

// SealOptions — что записать в открытые сведения хранилища.
type SealOptions struct {
	PersonID       string
	KeyStorage     string
	StorageVariant string
	Now            time.Time
	KDF            *KDF
}

// NewBundle — набор ключей из файлов; субъект — из опций или по имени ключа.
// Ключи одного субъекта, не больше одного на профиль (AD-11).
func NewBundle(files [][]byte, person string) (Bundle, error) {
	var b Bundle
	if len(files) == 0 {
		return b, fmt.Errorf("%w: не выбран ни один файл ключа", ErrKeyFile)
	}
	seen := map[string]bool{}
	for _, raw := range files {
		f, _, err := ParseKeyFile(raw)
		if err != nil {
			return b, err
		}
		if seen[f.Profile] {
			return b, fmt.Errorf("%w: два ключа профиля %s", ErrKeyFile, f.Profile)
		}
		seen[f.Profile] = true
		p := PersonOf(f.KeyRef)
		if person == "" {
			person = p
		}
		if p != person && person != "" && !strings.EqualFold(p, person) {
			return b, fmt.Errorf("%w: ключ %s не принадлежит %s", ErrKeyFile, f.KeyRef, person)
		}
		b.Keys = append(b.Keys, f)
	}
	if !seen[dom.ProfileGost] {
		return b, fmt.Errorf("%w: нужен ключ профиля gost (…-ta@1.key.json)", ErrKeyFile)
	}
	slices.SortFunc(b.Keys, func(x, y KeyFile) int { return strings.Compare(x.Profile, y.Profile) })
	b.PersonID = person
	return b, nil
}

// Seal — зашифровать набор ключей под PIN.
func Seal(b Bundle, pin string, o SealOptions, rnd io.Reader) (Sealed, error) {
	kdf, err := freshKDF(pin, o, rnd)
	if err != nil {
		return Sealed{}, err
	}
	return sealWith(b, deriveWith(kdf, pin), kdf, o, rnd)
}

// freshKDF — параметры argon2id с новой солью; PIN не короче MinPIN.
func freshKDF(pin string, o SealOptions, rnd io.Reader) (KDF, error) {
	if utf8.RuneCountInString(pin) < MinPIN {
		return KDF{}, fmt.Errorf("PIN — не короче %d символов", MinPIN)
	}
	kdf := DefaultKDF()
	if o.KDF != nil {
		kdf = *o.KDF
	}
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rnd, salt); err != nil {
		return KDF{}, err
	}
	kdf.SaltB64 = base64.StdEncoding.EncodeToString(salt)
	return kdf, nil
}

// sealWith — зашифровать набор готовым ключом из PIN (AES-256-GCM, свой nonce).
func sealWith(b Bundle, dk []byte, kdf KDF, o SealOptions, rnd io.Reader) (Sealed, error) {
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rnd, nonce); err != nil {
		return Sealed{}, err
	}
	s := Sealed{FormatVersion: FormatVersion, PersonID: b.PersonID, KeyStorage: o.KeyStorage, StorageVariant: o.StorageVariant,
		SealedAt: stamp(o.Now), KDF: kdf, NonceB64: base64.StdEncoding.EncodeToString(nonce)}
	if o.PersonID != "" {
		s.PersonID = o.PersonID
	}
	for _, k := range b.Keys {
		s.Keys = append(s.Keys, KeyInfo{KeyRef: k.KeyRef, Profile: k.Profile, Fingerprint: k.Fingerprint})
	}
	plain, err := json.Marshal(b)
	if err != nil {
		return Sealed{}, err
	}
	gcm, err := aead(dk)
	if err != nil {
		return Sealed{}, err
	}
	s.CipherB64 = base64.StdEncoding.EncodeToString(gcm.Seal(nil, nonce, plain, s.aad()))
	return s, nil
}

// SealEach — каждый файл ключа в своё хранилище под одним PIN (Д-72, демо из
// одного браузера: ключи всех персон рабочего места). argon2id считается один
// раз — общая соль и параметры у всех хранилищ, — затем AES-256-GCM на каждый
// ключ со своим nonce; владелец хранилища — по имени ключа (PersonOf).
// existing — уже загруженное хранилище: новые ключи ложатся под тот же PIN
// (его соль), неверный PIN — ErrPIN. Ответ — хранилища (повтор key_ref в
// наборе — один раз) и ключ из PIN для памяти сеанса.
func SealEach(files [][]byte, pin string, o SealOptions, existing *Sealed, rnd io.Reader) ([]Sealed, []byte, error) {
	if len(files) == 0 {
		return nil, nil, fmt.Errorf("%w: не выбран ни один файл ключа", ErrKeyFile)
	}
	var kdf KDF
	var dk []byte
	if existing != nil {
		kdf, dk = existing.KDF, DeriveKey(*existing, pin)
		if _, err := Open(*existing, dk); err != nil {
			return nil, nil, err
		}
	} else {
		var err error
		if kdf, err = freshKDF(pin, o, rnd); err != nil {
			return nil, nil, err
		}
		dk = deriveWith(kdf, pin)
	}
	o.PersonID = ""
	seen := map[string]bool{}
	out := make([]Sealed, 0, len(files))
	for _, raw := range files {
		f, _, err := ParseKeyFile(raw)
		if err != nil {
			return nil, nil, err
		}
		if seen[f.KeyRef] {
			continue
		}
		seen[f.KeyRef] = true
		s, err := sealWith(Bundle{PersonID: PersonOf(f.KeyRef), Keys: []KeyFile{f}}, dk, kdf, o, rnd)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, s)
	}
	return out, dk, nil
}

// DeriveKey — ключ шифрования хранилища из PIN (argon2id). Результат можно
// держать в памяти сеанса браузера вместо PIN (разблокировано до блокировки).
func DeriveKey(s Sealed, pin string) []byte { return deriveWith(s.KDF, pin) }

func deriveWith(kdf KDF, pin string) []byte {
	salt, _ := base64.StdEncoding.DecodeString(kdf.SaltB64)
	return argon2.IDKey([]byte(pin), salt, kdf.Time, kdf.MemoryKiB, kdf.Threads, 32)
}

// Set — хранилища ключей одного человека (по одному на профиль), открываемые
// одним ключом из PIN: так расширение держит ключи многих персон и подписывает
// ключом вошедшего (gost — один ключ, hybrid — ГОСТ и ML-DSA).
type Set []Sealed

// ParseSet — одно хранилище (объект JSON) или несколько (массив).
func ParseSet(raw []byte) (Set, error) {
	var set Set
	raw = bytes.TrimSpace(raw)
	if len(raw) > 0 && raw[0] == '[' {
		if err := json.Unmarshal(raw, &set); err != nil {
			return nil, err
		}
	} else {
		var s Sealed
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, err
		}
		set = Set{s}
	}
	if len(set) == 0 {
		return nil, refuse(CodeTokenMissing, "ключ не загружен")
	}
	seenProfile := map[string]bool{}
	for _, s := range set {
		if len(s.Keys) == 0 {
			return nil, refuse(CodeTokenMissing, "ключ не загружен")
		}
		// Никакого «суперключа»: подпись — ключами одного человека.
		if s.PersonID != set[0].PersonID {
			return nil, refuse(CodeInvalid, "ключи разных людей (%s, %s) в одной подписи", set[0].PersonID, s.PersonID)
		}
		for _, k := range s.Keys {
			if seenProfile[k.Profile] {
				return nil, refuse(CodeInvalid, "два ключа профиля %s в одной подписи", k.Profile)
			}
			seenProfile[k.Profile] = true
		}
	}
	return set, nil
}

// PersonID — владелец ключей набора.
func (set Set) PersonID() string { return set[0].PersonID }

// Keys — открытые сведения всех ключей набора.
func (set Set) Keys() []KeyInfo {
	var out []KeyInfo
	for _, s := range set {
		out = append(out, s.Keys...)
	}
	return out
}

// Open — расшифровать все хранилища набора одним ключом из PIN.
func (set Set) Open(dk []byte) (Bundle, error) {
	b := Bundle{PersonID: set.PersonID()}
	for _, s := range set {
		bi, err := Open(s, dk)
		if err != nil {
			return Bundle{}, err
		}
		b.Keys = append(b.Keys, bi.Keys...)
	}
	return b, nil
}

// Open — расшифровать хранилище ключом из DeriveKey; неверный PIN — ErrPIN.
func Open(s Sealed, dk []byte) (Bundle, error) {
	var b Bundle
	if s.FormatVersion != FormatVersion {
		return b, fmt.Errorf("agent: формат хранилища %d не поддерживается", s.FormatVersion)
	}
	nonce, err1 := base64.StdEncoding.DecodeString(s.NonceB64)
	ct, err2 := base64.StdEncoding.DecodeString(s.CipherB64)
	if err1 != nil || err2 != nil {
		return b, fmt.Errorf("agent: хранилище повреждено")
	}
	gcm, err := aead(dk)
	if err != nil {
		return b, err
	}
	plain, err := gcm.Open(nil, nonce, ct, s.aad())
	if err != nil {
		return b, ErrPIN
	}
	if err := json.Unmarshal(plain, &b); err != nil {
		return b, err
	}
	if b.PersonID != s.PersonID {
		return b, fmt.Errorf("agent: владелец хранилища не сходится")
	}
	return b, nil
}

func aead(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, ErrPIN
	}
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(blk)
}

// TimeLayout — формат времени протокола (миллисекунды, UTC).
const TimeLayout = "2006-01-02T15:04:05.000Z"

func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(TimeLayout)
}
