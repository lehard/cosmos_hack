package signing

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.stargrave.org/gogost/v7/gost34112012256"
)

// Единый формат подписанного пакета (AD-10, FR-67, кейс §6.3): конверт DSSE v1
// над каноническим JSON (RFC 8785). Здесь — только чистые функции разбора и
// кодирования; сама криптография (ГОСТ Р 34.10-2012, ML-DSA-65) — за портами
// Signer / Verifier (infrastructure/security/profiles), в домен не входит (AD-1).

// HashPrefix — префикс записи хеша формата цепочки v1 (Стрибог-256, AD-44).
const HashPrefix = "streebog256:"

// Envelope — конверт DSSE v1 (contracts/crypto/dsse-envelope.schema.json).
type Envelope struct {
	PayloadType string      `json:"payloadType"`
	Payload     string      `json:"payload"`
	Signatures  []Signature `json:"signatures"`
}

// Signature — одна подпись конверта: key_id@версия и подпись в base64.
type Signature struct {
	KeyID string `json:"keyid"`
	Sig   string `json:"sig"`
}

// Ошибки разбора пакета.
var (
	// ErrEnvelope — конверт не DSSE или нарушает закрытую схему.
	ErrEnvelope = errors.New("signing: конверт DSSE не по контракту")
	// ErrNotCanonical — подписываемые байты не совпадают со своим каноническим видом (AD-10).
	ErrNotCanonical = errors.New("signing: содержимое не в каноническом виде RFC 8785")
)

var (
	reKeyRef      = regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,95}@[1-9][0-9]{0,5}$`)
	rePayloadType = regexp.MustCompile(`^application/vnd\.ant\.([a-z][a-z-]*)\+json; v=([1-9][0-9]*)$`)
)

// ValidKeyRef — строка по шаблону key_id@версия.
func ValidKeyRef(s string) bool { return reKeyRef.MatchString(s) }

// SplitKeyRef — key_id и версия ключа из key_id@версия.
func SplitKeyRef(ref string) (string, int, error) {
	if !ValidKeyRef(ref) {
		return "", 0, fmt.Errorf("%w: key_ref %q", ErrEnvelope, ref)
	}
	i := strings.LastIndexByte(ref, '@')
	v, err := strconv.Atoi(ref[i+1:])
	return ref[:i], v, err
}

// PayloadType — application/vnd.ant.‹класс›+json; v=‹версия› (AD-10).
func PayloadType(class string, version int) string {
	return "application/vnd.ant." + class + "+json; v=" + strconv.Itoa(version)
}

// ParsePayloadType — класс и версия пакета из payloadType.
func ParsePayloadType(pt string) (string, int, error) {
	m := rePayloadType.FindStringSubmatch(pt)
	if m == nil {
		return "", 0, fmt.Errorf("%w: payloadType %q", ErrEnvelope, pt)
	}
	v, err := strconv.Atoi(m[2])
	return m[1], v, err
}

// PAE — Pre-Authentication Encoding DSSE v1: подпись ставится над ним, а не над
// base64 (contracts/crypto/README.md).
func PAE(payloadType string, payload []byte) []byte {
	var b bytes.Buffer
	b.WriteString("DSSEv1 ")
	b.WriteString(strconv.Itoa(len(payloadType)))
	b.WriteByte(' ')
	b.WriteString(payloadType)
	b.WriteByte(' ')
	b.WriteString(strconv.Itoa(len(payload)))
	b.WriteByte(' ')
	b.Write(payload)
	return b.Bytes()
}

// Canonical — JCS (RFC 8785) над JSON-значением raw (stdlib jsontext).
func Canonical(raw []byte) ([]byte, error) {
	v := jsontext.Value(bytes.Clone(raw))
	if err := v.Canonicalize(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotCanonical, err)
	}
	return []byte(v), nil
}

// CanonicalOf — канонический JSON значения v (json.Marshal → JCS).
func CanonicalOf(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return Canonical(raw)
}

// CheckCanonical — AD-10: подписываемые байты обязаны совпадать с каноническим
// видом своего разбора (без повторяющихся ключей, без пробелов, ключи по
// порядку) — иначе атака на расхождение парсеров.
func CheckCanonical(payload []byte) error {
	c, err := Canonical(payload)
	if err != nil {
		return err
	}
	if !bytes.Equal(c, payload) {
		return ErrNotCanonical
	}
	return nil
}

// ParseEnvelope разбирает конверт DSSE и возвращает его вместе с байтами
// содержимого. Проверяет закрытую схему: класс пакета, base64, key_id@версия.
// Пустой список подписей допустим только разбору — отказ «не подписано»
// ставит проверка (Judge).
func ParseEnvelope(raw []byte) (Envelope, []byte, error) {
	var e Envelope
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		return e, nil, fmt.Errorf("%w: %v", ErrEnvelope, err)
	}
	if _, _, err := ParsePayloadType(e.PayloadType); err != nil {
		return e, nil, err
	}
	p, err := base64.StdEncoding.DecodeString(e.Payload)
	if err != nil {
		return e, nil, fmt.Errorf("%w: payload не base64", ErrEnvelope)
	}
	if len(e.Signatures) > 8 {
		return e, nil, fmt.Errorf("%w: больше 8 подписей", ErrEnvelope)
	}
	for _, s := range e.Signatures {
		if !ValidKeyRef(s.KeyID) {
			return e, nil, fmt.Errorf("%w: keyid %q", ErrEnvelope, s.KeyID)
		}
		if _, err := base64.StdEncoding.DecodeString(s.Sig); err != nil {
			return e, nil, fmt.Errorf("%w: подпись %s не base64", ErrEnvelope, s.KeyID)
		}
	}
	return e, p, nil
}

// Seal — конверт без подписей над каноническим payload (профиль demo, Д-28, Д-30).
func Seal(payloadType string, payload []byte) Envelope {
	return Envelope{PayloadType: payloadType, Payload: base64.StdEncoding.EncodeToString(payload), Signatures: []Signature{}}
}

// Marshal — JSON конверта.
func (e Envelope) Marshal() []byte {
	b, _ := json.Marshal(e)
	return b
}

// Hash — Стрибог-256 (ГОСТ Р 34.11-2012) от склейки частей, сырые 32 байта.
func Hash(parts ...[]byte) []byte {
	h := gost34112012256.New()
	for _, p := range parts {
		h.Write(p)
	}
	return h.Sum(nil)
}

// Digest — H(b) в записи `streebog256:‹hex›`: отпечатки документов, адреса
// материалов (скан бумаги), отпечатки ключей (AD-12, AD-44).
func Digest(b []byte) string { return HashPrefix + hex.EncodeToString(Hash(b)) }

// ParseDigest — сырые 32 байта из `streebog256:‹64 hex›`.
func ParseDigest(s string) ([]byte, error) {
	h, ok := strings.CutPrefix(s, HashPrefix)
	if !ok || len(h) != 64 {
		return nil, fmt.Errorf("signing: ожидается streebog256:‹64 hex›: %q", s)
	}
	return hex.DecodeString(h)
}

// SignatureDigest — отпечаток одной подписи: лист сменного рапорта (AD-12,
// FR-66 уровень 3). Связывает содержимое и саму подпись:
// H(PAE(payloadType, payload) ‖ 0x00 ‖ keyid ‖ 0x00 ‖ подпись).
func SignatureDigest(payloadType string, payload []byte, s Signature) []byte {
	sig, _ := base64.StdEncoding.DecodeString(s.Sig)
	return Hash(PAE(payloadType, payload), []byte{0}, []byte(s.KeyID), []byte{0}, sig)
}
