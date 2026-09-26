package federation

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"ant/internal/domain/signing"
)

// Выписка паспорта (FR-131, FR-132; AD-19, AD-10; docs/federation.md §4) —
// подписанный пакет единого формата: конверт DSSE v1 над каноническим JSON
// (RFC 8785), класс пакета passport-extract. Внутри — предмет (партия или
// изделие с глобальным ID «код_предприятия:локальный_id»), происхождение
// (плавка, сертификат, химсостав), результаты контроля, решения, подписанты
// с актами их ключей (подписаны корнем предприятия-отправителя) и
// контрольная точка хранителя отправителя (AD-8).
//
// Получатель проверяет выписку сам, без доступа к журналу отправителя:
// корни партнёра — из нашего акта регистрации партнёра, ключи сотрудников
// партнёра — из актов внутри выписки (доверие выводится из регистрации
// партнёра, отдельный акт на каждый ключ у нас не нужен, AD-19).

// ExtractClass — класс пакета выписки (payloadType application/vnd.ant.passport-extract+json; v=1).
const ExtractClass = "passport-extract"

// KeyActClass — класс пакета акта ключа подписанта внутри выписки.
const KeyActClass = "key-act"

// ExtractPayloadType — payloadType выписки v1.
var ExtractPayloadType = signing.PayloadType(ExtractClass, 1)

// KeyActPayloadType — payloadType акта ключа v1.
var KeyActPayloadType = signing.PayloadType(KeyActClass, 1)

// Статусы происхождения (federation.extract.received.origin_status).
const (
	// OriginVerified — подписи людей, акты их ключей и контрольная точка хранителя
	// отправителя проверяются: «происхождение подтверждено».
	OriginVerified = "verified"
	// OriginServerOnly — подпись есть, но без актов ключей людей или контрольной
	// точки: «происхождение подтверждено сервером отправителя».
	OriginServerOnly = "server_confirmed_only"
	// OriginUnverified — подписи проверить нельзя (партнёр не зарегистрирован,
	// ключи неизвестны, подписей нет): «происхождение не подтверждено» (FR-132).
	OriginUnverified = "unverified"
	// OriginNotApplicable — исходящая выписка: проверяет получатель.
	OriginNotApplicable = "not_applicable"
)

// Итог проверки одной подписи.
const (
	SigValid        = "valid"
	SigInvalid      = "invalid"
	SigUnverifiable = "unverifiable"
)

// ErrTampered — содержимое выписки изменено после подписи (или не в каноническом
// виде): выписка отклоняется при импорте (FR-132), код federation.extract_tampered.
var ErrTampered = errors.New("federation: выписка изменена после подписи")

// ErrFormat — пакет не выписка паспорта (не DSSE, другой класс, не тот формат).
var ErrFormat = errors.New("federation: пакет не выписка паспорта")

// Subject — предмет выписки: партия или изделие.
type Subject struct {
	Kind     string `json:"kind"`      // lot | item
	GlobalID string `json:"global_id"` // код_предприятия:локальный_id
	Label    string `json:"label,omitempty"`
	Quantity string `json:"quantity,omitempty"`
	// RecipientRef — ID предмета у получателя (партия по накладной или
	// заказу): под ним выписка становится корнем генеалогии у получателя.
	RecipientRef string `json:"recipient_ref,omitempty"`
}

// Origin — происхождение: плавка, материал, сертификат, химсостав.
type Origin struct {
	HeatNo      string            `json:"heat_no,omitempty"`
	Material    string            `json:"material,omitempty"`
	Standard    string            `json:"standard,omitempty"`
	Certificate string            `json:"certificate,omitempty"`
	Chemistry   map[string]string `json:"chemistry,omitempty"`
	Mechanical  map[string]string `json:"mechanical,omitempty"`
}

// Control — результат контроля у отправителя.
type Control struct {
	Name   string `json:"name"`
	Result string `json:"result"`
	Value  string `json:"value,omitempty"`
	Norm   string `json:"norm,omitempty"`
	By     string `json:"by,omitempty"` // key_ref подписанта
}

// Signer — подписант выписки: ключ и акт его регистрации у отправителя,
// подписанный корнем предприятия (конверт DSSE класса key-act).
type Signer struct {
	KeyRef string `json:"key_ref"`
	Role   string `json:"role,omitempty"`
	Name   string `json:"name,omitempty"`
	// KeyAct — акт ключа: конверт DSSE (JSON) над KeyAct, подписанный корнем.
	KeyAct json.RawMessage `json:"key_act,omitempty"`
}

// KeyAct — содержимое акта ключа подписанта (отправитель подписывает корнем).
type KeyAct struct {
	Enterprise string `json:"enterprise"`
	KeyRef     string `json:"key_ref"`
	Profile    string `json:"profile"`
	PublicB64  string `json:"public_key_b64"`
	Role       string `json:"role,omitempty"`
}

// Root — корневой ключ предприятия-отправителя; его отпечаток должен быть в
// нашем акте регистрации партнёра. Ключ шлюза предприятия (уровень 0) —
// тоже корень: его подпись — «подтверждено сервером отправителя».
type Root struct {
	KeyRef    string `json:"key_ref"`
	Profile   string `json:"profile"`
	PublicB64 string `json:"public_key_b64"`
}

// Checkpoint — контрольная точка хранителя отправителя (AD-8).
type Checkpoint struct {
	Keeper   string `json:"keeper"`
	HeadSeq  int64  `json:"head_seq"`
	HeadHash string `json:"head_hash"`
	At       string `json:"at,omitempty"`
}

// Extract — содержимое выписки паспорта v1.
type Extract struct {
	FormatVersion int         `json:"format_version"`
	CryptoProfile string      `json:"crypto_profile"`
	Sender        string      `json:"sender"`
	Recipient     string      `json:"recipient,omitempty"`
	ExtractNo     string      `json:"extract_no"`
	IssuedAt      string      `json:"issued_at"`
	Subject       Subject     `json:"subject"`
	Origin        Origin      `json:"origin"`
	Control       []Control   `json:"control,omitempty"`
	Decisions     []string    `json:"decisions,omitempty"`
	Signers       []Signer    `json:"signers,omitempty"`
	Roots         []Root      `json:"roots,omitempty"`
	Checkpoint    *Checkpoint `json:"checkpoint,omitempty"`
	Sources       []string    `json:"sources,omitempty"` // ссылки на исходные объекты (FR-67)
}

// CryptoFunc — криптографическая проверка одной подписи над PAE открытым ключом
// профиля (порт Verifier: infrastructure/security/profiles.Verify). Домен
// криптографии не содержит (AD-1).
type CryptoFunc func(profile string, pub []byte, payloadType string, pae, sig []byte) bool

// SignatureCheck — проверка одной подписи выписки.
type SignatureCheck struct {
	KeyRef       string
	Role         string
	Name         string
	Human        bool // ключ сотрудника по акту (не корень/шлюз)
	Verification string
}

// Verification — итог проверки выписки получателем.
type Verification struct {
	Extract    Extract
	Payload    []byte
	Digest     string // отпечаток выписки: H(payload) — не зависит от набора подписей
	Origin     string
	Signatures []SignatureCheck
	// Checkpoint — контрольная точка есть в выписке.
	Checkpoint bool
	// Reason — пояснение статуса для человека.
	Reason string
}

// ExtractDigest — отпечаток выписки: Стрибог-256 канонического содержимого.
func ExtractDigest(payload []byte) string { return signing.Digest(payload) }

// ParseExtract разбирает конверт выписки: DSSE, класс passport-extract,
// канонический JSON (AD-10) — иначе ErrFormat / ErrTampered.
func ParseExtract(raw []byte) (signing.Envelope, Extract, []byte, error) {
	env, payload, err := signing.ParseEnvelope(raw)
	if err != nil {
		return env, Extract{}, nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	if cls, _, _ := signing.ParsePayloadType(env.PayloadType); cls != ExtractClass {
		return env, Extract{}, nil, fmt.Errorf("%w: класс %q", ErrFormat, cls)
	}
	if err := signing.CheckCanonical(payload); err != nil {
		return env, Extract{}, nil, fmt.Errorf("%w: %v", ErrTampered, err)
	}
	var x Extract
	if err := json.Unmarshal(payload, &x); err != nil {
		return env, Extract{}, nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	if x.FormatVersion != 1 || x.Sender == "" || x.Subject.GlobalID == "" {
		return env, Extract{}, nil, fmt.Errorf("%w: нет format_version=1, sender или subject.global_id", ErrFormat)
	}
	return env, x, payload, nil
}

// VerifyExtract — самостоятельная проверка выписки получателем (FR-132, AD-19):
//   - корни из выписки учитываются, только если их отпечатки есть в нашем
//     акте регистрации партнёра (rootFingerprints);
//   - ключ сотрудника — из акта ключа внутри выписки, подписанного таким корнем;
//   - подпись известным ключом не сходится — выписка изменена: ErrTampered;
//   - сходятся подписи людей по актам и есть контрольная точка — verified;
//   - сходится хотя бы одна подпись (например, только шлюз) — server_confirmed_only;
//   - ни одной проверяемой подписи — unverified (принимается с пометкой).
//
// partner — код зарегистрированного партнёра, от которого пришла выписка
// ("" — партнёр не зарегистрирован: все подписи непроверяемы).
func VerifyExtract(raw []byte, partner string, rootFingerprints []string, verify CryptoFunc) (Verification, error) {
	env, x, payload, err := ParseExtract(raw)
	if err != nil {
		return Verification{}, err
	}
	v := Verification{Extract: x, Payload: payload, Digest: ExtractDigest(payload), Checkpoint: x.Checkpoint != nil && x.Checkpoint.HeadHash != ""}
	if partner != "" && x.Sender != partner {
		return v, fmt.Errorf("%w: отправитель %s, а канал партнёра %s", ErrFormat, x.Sender, partner)
	}
	// Корни: только зарегистрированные у нас (AD-19).
	keys := map[string]knownKey{}
	if partner != "" {
		for _, r := range x.Roots {
			pub, err := base64.StdEncoding.DecodeString(r.PublicB64)
			if err != nil || !slices.Contains(rootFingerprints, signing.Digest(pub)) {
				continue
			}
			keys[r.KeyRef] = knownKey{profile: r.Profile, pub: pub}
		}
	}
	// Ключи сотрудников — по актам, подписанным корнем.
	for _, s := range x.Signers {
		if _, ok := keys[s.KeyRef]; ok || len(s.KeyAct) == 0 {
			continue
		}
		if act, ok := checkKeyAct(s.KeyAct, x.Sender, s.KeyRef, keys, verify); ok {
			pub, _ := base64.StdEncoding.DecodeString(act.PublicB64)
			keys[s.KeyRef] = knownKey{profile: act.Profile, pub: pub, human: true}
		}
	}
	pae := signing.PAE(env.PayloadType, payload)
	valid, humans := 0, 0
	for _, s := range env.Signatures {
		c := SignatureCheck{KeyRef: s.KeyID, Verification: SigUnverifiable}
		for _, sg := range x.Signers {
			if sg.KeyRef == s.KeyID {
				c.Role, c.Name = sg.Role, sg.Name
			}
		}
		if k, ok := keys[s.KeyID]; ok {
			sig, _ := base64.StdEncoding.DecodeString(s.Sig)
			c.Human = k.human
			if verify(k.profile, k.pub, env.PayloadType, pae, sig) {
				c.Verification = SigValid
				valid++
				if k.human {
					humans++
				}
			} else {
				c.Verification = SigInvalid
			}
		}
		v.Signatures = append(v.Signatures, c)
	}
	for _, c := range v.Signatures {
		if c.Verification == SigInvalid {
			v.Origin, v.Reason = "", "подпись "+c.KeyRef+" не сходится с содержимым"
			return v, fmt.Errorf("%w: %s", ErrTampered, v.Reason)
		}
	}
	switch {
	case humans > 0 && v.Checkpoint:
		v.Origin, v.Reason = OriginVerified, "подписи сотрудников отправителя проверены по актам ключей и корням партнёра; есть контрольная точка хранителя"
	case valid > 0:
		v.Origin, v.Reason = OriginServerOnly, "проверена подпись сервера отправителя; нет подписей людей с актами ключей или контрольной точки"
	case partner == "":
		v.Origin, v.Reason = OriginUnverified, "отправитель не зарегистрирован как партнёр — корней для проверки нет"
	default:
		v.Origin, v.Reason = OriginUnverified, "ни одну подпись нельзя проверить корнями партнёра"
	}
	return v, nil
}

// checkKeyAct — акт ключа подписанта: класс key-act, предприятие-отправитель,
// тот же key_ref, подпись корнем из keys (только корни, не ключи людей).
func checkKeyAct(raw json.RawMessage, sender, keyRef string, keys map[string]knownKey, verify CryptoFunc) (KeyAct, bool) {
	env, payload, err := signing.ParseEnvelope(raw)
	if err != nil || env.PayloadType != KeyActPayloadType || signing.CheckCanonical(payload) != nil {
		return KeyAct{}, false
	}
	var act KeyAct
	if json.Unmarshal(payload, &act) != nil || act.Enterprise != sender || act.KeyRef != keyRef {
		return KeyAct{}, false
	}
	pae := signing.PAE(env.PayloadType, payload)
	for _, s := range env.Signatures {
		k, ok := keys[s.KeyID]
		if !ok || k.human {
			continue
		}
		sig, _ := base64.StdEncoding.DecodeString(s.Sig)
		if verify(k.profile, k.pub, env.PayloadType, pae, sig) {
			return act, true
		}
	}
	return KeyAct{}, false
}

// knownKey — открытый ключ, которому получатель доверяет: корень партнёра
// (human=false) или ключ сотрудника по акту, подписанному корнем (human=true).
type knownKey struct {
	profile string
	pub     []byte
	human   bool
}

// GlobalID — глобальный ID «код_предприятия:локальный_id» (соглашения спайна).
func GlobalID(enterprise, local string) string { return enterprise + ":" + local }

// SplitGlobalID — код предприятия и локальный ID.
func SplitGlobalID(g string) (string, string, bool) {
	return strings.Cut(g, ":")
}
