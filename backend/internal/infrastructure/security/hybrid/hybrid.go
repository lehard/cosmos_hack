// Пакет hybrid — подпись и проверка пакетов DSSE профиля hybrid (AD-10,
// AD-32, критерий О9): ГОСТ Р 34.10-2012 (256 бит, paramSetA, Стрибог-256;
// кодирование — contracts/crypto/README.md) и ML-DSA-65 (FIPS 204,
// crypto/mldsa, контекст `ant/‹payloadType›`); обе подписи обязательны,
// удаление любой — «понижение профиля». Им подписываются контрольные точки
// хранителя и отчёты верификатора (профиль hybrid по profiles.yaml); ключи —
// в томах keeper и verifier, открытые — в файле trust-anchors (AD-33).
//
// Слой: infrastructure/security — технический механизм защиты (AD-1).
// Криптография MVP — не СКЗИ; в промышленной эксплуатации — сертифицированное
// СКЗИ за портами Signer/Verifier (эпики 05, 27 сводят сюда общий профиль).
//
// Требования: FR-72, FR-76, AD-8, AD-10, AD-32, AD-46. Владелец: эпик 29.
package hybrid

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
	"strconv"
	"strings"

	"go.stargrave.org/gogost/v7/gost3410"
	"go.stargrave.org/gogost/v7/gost34112012256"
)

// Профили подписи (contracts/crypto/profiles.yaml).
const (
	ProfileGost = "gost"
	ProfilePQ   = "pq"
)

// PayloadType — payloadType класса пакета (contracts/crypto/payload-classes.yaml).
func PayloadType(class string) string { return "application/vnd.ant." + class + "+json; v=1" }

// PAE — Pre-Authentication Encoding DSSE v1.
func PAE(payloadType string, payload []byte) []byte {
	return []byte("DSSEv1 " + strconv.Itoa(len(payloadType)) + " " + payloadType + " " + strconv.Itoa(len(payload)) + " " + string(payload))
}

// Streebog — Стрибог-256.
func Streebog(b ...[]byte) []byte {
	h := gost34112012256.New()
	for _, p := range b {
		h.Write(p)
	}
	return h.Sum(nil)
}

// Fingerprint — отпечаток открытого ключа `streebog256:‹hex›`.
func Fingerprint(pub []byte) string { return "streebog256:" + hex.EncodeToString(Streebog(pub)) }

func curve() *gost3410.Curve { return gost3410.CurveIdtc26gost341012256paramSetA() }

// Envelope — конверт DSSE (contracts/crypto/dsse-envelope.schema.json).
type Envelope struct {
	PayloadType string      `json:"payloadType"`
	Payload     string      `json:"payload"`
	Signatures  []Signature `json:"signatures"`
}

// Signature — подпись конверта.
type Signature struct {
	KeyID string `json:"keyid"`
	Sig   string `json:"sig"`
}

// Signer — пара ключей субъекта профиля hybrid: ГОСТ и ML-DSA-65.
type Signer struct {
	// Subject — субъект (keeper, verifier); ключи — ‹subject›-gost@1 и ‹subject›-pq@1.
	Subject string
	gost    *gost3410.PrivateKey
	pq      *mldsa.PrivateKey
}

// KeyIDs — key_id@версия ключей подписанта (signers в payload, AD-10).
func (s *Signer) KeyIDs() []string { return []string{s.Subject + "-gost@1", s.Subject + "-pq@1"} }

// Sign — конверт DSSE класса class с двумя подписями над PAE.
func (s *Signer) Sign(class string, payload []byte) ([]byte, error) {
	pt := PayloadType(class)
	pae := PAE(pt, payload)
	gs, err := (&gost3410.PrivateKeyReverseDigest{Prv: s.gost}).Sign(rand.Reader, Streebog(pae), nil)
	if err != nil {
		return nil, fmt.Errorf("подпись ГОСТ: %w", err)
	}
	ps, err := s.pq.Sign(rand.Reader, pae, &mldsa.Options{Context: "ant/" + pt})
	if err != nil {
		return nil, fmt.Errorf("подпись ML-DSA: %w", err)
	}
	ids := s.KeyIDs()
	return json.Marshal(Envelope{PayloadType: pt, Payload: base64.StdEncoding.EncodeToString(payload), Signatures: []Signature{
		{KeyID: ids[0], Sig: base64.StdEncoding.EncodeToString(gs)},
		{KeyID: ids[1], Sig: base64.StdEncoding.EncodeToString(ps)},
	}})
}

// Anchor — открытый ключ из trust-anchors.
type Anchor struct {
	KeyID       string `json:"keyid"`
	Subject     string `json:"subject"`
	Profile     string `json:"profile"`
	PublicKey   string `json:"public_key_b64"`
	Fingerprint string `json:"fingerprint"`
}

// Anchors — файл trust-anchors (AD-33): открытые ключи хранителя и
// верификатора с отпечатками, ожидаемые профили классов; лежит в томе keeper,
// в verifier монтируется только он. До эпика 05 (генезис) его создаёт
// `keeper -init`; в промышленной эксплуатации он подписан Аудитором ИБ и
// передаётся на отчуждаемом носителе.
type Anchors struct {
	FormatVersion int               `json:"format_version"`
	Keys          []Anchor          `json:"keys"`
	Profiles      map[string]string `json:"object_profiles"`
	// AnchorFingerprint — отпечаток ключей якоря генезиса (пишет ant init, эпик 05).
	AnchorFingerprint string `json:"anchor_fingerprint,omitempty"`
	// GenesisDigest — отпечаток генезиса: первая контрольная точка хранителя.
	GenesisDigest string `json:"genesis_digest,omitempty"`
	Note          string `json:"note,omitempty"`
}

// LoadAnchors читает trust-anchors и его отпечаток (streebog256 байтов файла).
func LoadAnchors(path string) (Anchors, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Anchors{}, "", err
	}
	var a Anchors
	if err := json.Unmarshal(b, &a); err != nil {
		return Anchors{}, "", fmt.Errorf("trust-anchors: %w", err)
	}
	return a, Fingerprint(b), nil
}

// Errors of verification.
var (
	ErrDowngrade  = errors.New("отвергнуто: понижение профиля — нет обеих подписей hybrid")
	ErrBadSig     = errors.New("отвергнуто: подпись не сходится")
	ErrUnknownKey = errors.New("не проверяемо: ключа нет в trust-anchors")
	ErrClass      = errors.New("отвергнуто: подпись не по назначению (другой класс пакета)")
)

// Verify проверяет конверт класса class подписями субъекта subject: обе
// подписи hybrid обязательны; лишние и повторные не засчитываются (AD-10).
// Возвращает payload.
func (a Anchors) Verify(env []byte, class, subject string) ([]byte, error) {
	var e Envelope
	if err := json.Unmarshal(env, &e); err != nil {
		return nil, fmt.Errorf("конверт DSSE: %w", err)
	}
	if e.PayloadType != PayloadType(class) {
		return nil, fmt.Errorf("%w: %s", ErrClass, e.PayloadType)
	}
	payload, err := base64.StdEncoding.DecodeString(e.Payload)
	if err != nil {
		return nil, fmt.Errorf("payload: %w", err)
	}
	pae := PAE(e.PayloadType, payload)
	ok := map[string]bool{}
	for _, s := range e.Signatures {
		i := slices.IndexFunc(a.Keys, func(k Anchor) bool { return k.KeyID == s.KeyID && k.Subject == subject })
		if i < 0 || ok[a.Keys[i].Profile] {
			continue
		}
		k := a.Keys[i]
		sig, err := base64.StdEncoding.DecodeString(s.Sig)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrBadSig, s.KeyID)
		}
		pub, err := base64.StdEncoding.DecodeString(k.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("trust-anchors: %s: %w", k.KeyID, err)
		}
		if Fingerprint(pub) != k.Fingerprint {
			return nil, fmt.Errorf("trust-anchors: %s: отпечаток не сходится с ключом", k.KeyID)
		}
		switch k.Profile {
		case ProfileGost:
			p, err := gost3410.NewPublicKeyLE(curve(), pub)
			if err != nil {
				return nil, err
			}
			good, err := (gost3410.PublicKeyReverseDigest{Pub: p}).VerifyDigest(Streebog(pae), sig)
			if err != nil || !good {
				return nil, fmt.Errorf("%w: %s (ГОСТ Р 34.10-2012)", ErrBadSig, s.KeyID)
			}
		case ProfilePQ:
			p, err := mldsa.NewPublicKey(mldsa.MLDSA65(), pub)
			if err != nil {
				return nil, err
			}
			if err := mldsa.Verify(p, pae, sig, &mldsa.Options{Context: "ant/" + e.PayloadType}); err != nil {
				return nil, fmt.Errorf("%w: %s (ML-DSA-65)", ErrBadSig, s.KeyID)
			}
		default:
			continue
		}
		ok[k.Profile] = true
	}
	if !ok[ProfileGost] || !ok[ProfilePQ] {
		if len(ok) == 0 && !slices.ContainsFunc(a.Keys, func(k Anchor) bool { return k.Subject == subject }) {
			return nil, fmt.Errorf("%w: %s", ErrUnknownKey, subject)
		}
		return nil, ErrDowngrade
	}
	return payload, nil
}

// Файлы ключей субъекта в его томе: ‹dir›/‹subject›.gost (закрытый ключ LE,
// hex) и ‹dir›/‹subject›.pq (семя ML-DSA-65, hex), 0400.

// Generate создаёт ключи субъекта в dir, если их нет, и возвращает открытые
// ключи для trust-anchors.
func Generate(dir, subject string) ([]Anchor, error) {
	gp, pp := filepath.Join(dir, subject+".gost"), filepath.Join(dir, subject+".pq")
	if _, err := os.Stat(gp); errors.Is(err, os.ErrNotExist) {
		prv, err := gost3410.GenPrivateKey(curve(), rand.Reader)
		if err != nil {
			return nil, err
		}
		if err := writeKey(gp, prv.RawLE()); err != nil {
			return nil, err
		}
	}
	if _, err := os.Stat(pp); errors.Is(err, os.ErrNotExist) {
		seed := make([]byte, mldsa.PrivateKeySize)
		if _, err := rand.Read(seed); err != nil {
			return nil, err
		}
		if err := writeKey(pp, seed); err != nil {
			return nil, err
		}
	}
	s, err := Load(dir, subject)
	if err != nil {
		return nil, err
	}
	return s.Anchors()
}

func writeKey(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(hex.EncodeToString(raw)+"\n"), 0o400)
}

func readKey(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(strings.TrimSpace(string(b)))
}

// Load читает ключи субъекта из dir.
func Load(dir, subject string) (*Signer, error) {
	graw, err := readKey(filepath.Join(dir, subject+".gost"))
	if err != nil {
		return nil, fmt.Errorf("ключ ГОСТ %s: %w", subject, err)
	}
	g, err := gost3410.NewPrivateKeyLE(curve(), graw)
	if err != nil {
		return nil, err
	}
	seed, err := readKey(filepath.Join(dir, subject+".pq"))
	if err != nil {
		return nil, fmt.Errorf("ключ ML-DSA %s: %w", subject, err)
	}
	p, err := mldsa.NewPrivateKey(mldsa.MLDSA65(), seed)
	if err != nil {
		return nil, err
	}
	return &Signer{Subject: subject, gost: g, pq: p}, nil
}

// Anchors — открытые ключи подписанта для trust-anchors.
func (s *Signer) Anchors() ([]Anchor, error) {
	gpub, err := s.gost.PublicKey()
	if err != nil {
		return nil, err
	}
	graw, praw := gpub.RawLE(), s.pq.PublicKey().Bytes()
	ids := s.KeyIDs()
	return []Anchor{
		{KeyID: ids[0], Subject: s.Subject, Profile: ProfileGost, PublicKey: base64.StdEncoding.EncodeToString(graw), Fingerprint: Fingerprint(graw)},
		{KeyID: ids[1], Subject: s.Subject, Profile: ProfilePQ, PublicKey: base64.StdEncoding.EncodeToString(praw), Fingerprint: Fingerprint(praw)},
	}, nil
}
