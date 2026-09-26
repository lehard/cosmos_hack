package agent

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"ant/internal/contracts/procs"
	dom "ant/internal/domain/signing"
	"ant/internal/infrastructure/security/profiles"
)

// Коды отказа агента (contracts/errors.yaml) — поле error.code протокола.
const (
	CodeLevel         = "signing.level_not_allowed"
	CodeUnknownFormat = "signing.unknown_doc_format"
	CodeChanged       = "signing.document_changed"
	CodeTampered      = "signing.package_tampered"
	CodeTokenMissing  = "signing.token_missing"
	CodePIN           = "signing.pin_wrong"
	CodeCancelled     = "signing.cancelled"
	CodeRate          = "signing.rate_limited"
	CodeInvalid       = "api.validation_failed"
)

// Error — отказ агента с кодом из каталога ошибок.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func refuse(code, format string, a ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, a...)}
}

// CodeOf — код отказа для ошибки агента.
func CodeOf(err error) string {
	var e *Error
	switch {
	case errors.As(err, &e):
		return e.Code
	case errors.Is(err, ErrPIN):
		return CodePIN
	case errors.Is(err, ErrKeyFile):
		return CodeInvalid
	}
	return CodeInvalid
}

// DocFormatVersions — известные агенту версии формата документа (AD-12):
// неизвестную агент отвергает явной ошибкой.
var DocFormatVersions = []int{1}

// Context — обстоятельства подписи, которые агент берёт сам, а не у страницы
// (AD-14): время клиента, рабочее место из своей конфигурации, последняя
// известная контрольная точка, обязательный профиль класса event.
type Context struct {
	Now            time.Time
	WorkplaceID    string
	SeenCheckpoint int64
	// Profile — gost | hybrid (hybrid — два ключа одного субъекта).
	Profile string
}

// Field — поле сводки уровня 2.
type Field struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Prepared — что будет подписано: канонические байты, отпечаток и сводка,
// посчитанные самим агентом (AD-14: страница может только показать свою копию).
type Prepared struct {
	Level          int      `json:"level"`
	PayloadType    string   `json:"payload_type"`
	EventType      string   `json:"event_type"`
	PayloadB64     string   `json:"payload_b64"`
	DocDigest      string   `json:"doc_digest"`
	Summary        []Field  `json:"summary"`
	Signers        []string `json:"signers"`
	PersonID       string   `json:"person_id"`
	ClientSignedAt string   `json:"client_signed_at"`
	// Batch — пачка уровня 2 (одно окно на все элементы).
	ItemID string `json:"item_id,omitempty"`
}

// Payload — канонические байты.
func (p Prepared) Payload() []byte {
	b, _ := base64.StdEncoding.DecodeString(p.PayloadB64)
	return b
}

// Prepare — разобрать запрос подписи, собрать подписываемое, посчитать
// отпечаток и сводку и проверить правила уровня (AD-13, AD-14, PRD §11.16).
// person и keys — владелец и ключи хранилища (открытые сведения).
func Prepare(b procs.SignBlock, person string, keys []KeyInfo, c Context) (Prepared, error) {
	p := Prepared{Level: b.Level, PayloadType: b.PayloadType, PersonID: person, ClientSignedAt: stamp(c.Now)}
	if b.Level != dom.Level1 && b.Level != dom.Level2 {
		return p, refuse(CodeLevel, "агент подписывает только уровни 1 и 2, запрошен %d", b.Level)
	}
	if b.DocFormatVersion != nil && !slices.Contains(DocFormatVersions, *b.DocFormatVersion) {
		return p, refuse(CodeUnknownFormat, "doc_format_version %d агенту неизвестен", *b.DocFormatVersion)
	}
	class, _, err := dom.ParsePayloadType(b.PayloadType)
	if err != nil {
		return p, refuse(CodeInvalid, "payload_type: %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(b.PayloadB64)
	if err != nil {
		return p, refuse(CodeInvalid, "payload_b64 — не base64")
	}
	signers, err := signersFor(keys, c.Profile)
	if err != nil {
		return p, err
	}
	p.Signers = signers
	var payload []byte
	switch {
	case b.CommandRequest != nil:
		if class != dom.ClassEvent {
			return p, refuse(CodeInvalid, "command_request — только для пакетов класса event, а не %s", class)
		}
		if b.EventType == nil || *b.EventType == "" {
			return p, refuse(CodeInvalid, "command_request без event_type")
		}
		payload, err = buildCommand(*b.CommandRequest, *b.EventType, b.Level, raw, person, signers, c)
		if err != nil {
			return p, err
		}
		p.EventType = *b.EventType
	default:
		// Готовое каноническое содержимое (событие-команда, документ, заверение).
		if err := dom.CheckCanonical(raw); err != nil {
			return p, refuse(CodeTampered, "содержимое не в каноническом виде (AD-10)")
		}
		payload = raw
		var ev struct {
			EventType string `json:"event_type"`
			Command   struct {
				SignatureLevel *int `json:"signature_level"`
			} `json:"command"`
			SignatureLevel *int `json:"signature_level"`
			Integrity      struct {
				Signers []string `json:"signers"`
			} `json:"integrity"`
			Signers []string `json:"signers"`
		}
		_ = json.Unmarshal(raw, &ev)
		p.EventType = ev.EventType
		if b.EventType != nil && *b.EventType != "" {
			if ev.EventType != "" && ev.EventType != *b.EventType {
				return p, refuse(CodeChanged, "в содержимом %s, а в запросе %s", ev.EventType, *b.EventType)
			}
			p.EventType = *b.EventType
		}
		lvl := ev.Command.SignatureLevel
		if lvl == nil {
			lvl = ev.SignatureLevel
		}
		if lvl != nil && *lvl != b.Level {
			return p, refuse(CodeLevel, "в содержимом уровень %d, в запросе %d", *lvl, b.Level)
		}
		declared := ev.Integrity.Signers
		if declared == nil {
			declared = ev.Signers
		}
		for _, s := range declared {
			if !slices.Contains(p.Signers, s) {
				return p, refuse(CodeTampered, "содержимое называет подписантом %s — такого ключа у агента нет", s)
			}
		}
		if len(declared) > 0 {
			p.Signers = declared
		}
	}
	// Уровень 1 — только вкомпилированный перечень (AD-13): «годен»,
	// заключения ОТК и критические действия уровнем 1 не подписываются.
	if b.Level == dom.Level1 && !dom.Level1Allowed(p.EventType) {
		return p, refuse(CodeLevel, "%s не входит в перечень уровня 1 — нужна подпись уровня 2 с окном подтверждения", orDash(p.EventType))
	}
	p.PayloadB64 = base64.StdEncoding.EncodeToString(payload)
	p.DocDigest = dom.Digest(payload)
	if b.ExpectedDocDigest != nil && *b.ExpectedDocDigest != "" && *b.ExpectedDocDigest != p.DocDigest {
		return p, refuse(CodeChanged, "сервер ожидает отпечаток %s, агент посчитал %s — подпись не выполнена", short(*b.ExpectedDocDigest), short(p.DocDigest))
	}
	p.Summary = summarize(p, b, payload)
	return p, nil
}

// signersFor — ключи подписи под обязательный профиль (AD-32): gost — один
// ключ ГОСТ; hybrid — ГОСТ и ML-DSA-65 одного субъекта.
func signersFor(keys []KeyInfo, profile string) ([]string, error) {
	var gost, pq string
	for _, k := range keys {
		switch k.Profile {
		case dom.ProfileGost:
			gost = k.KeyRef
		case dom.ProfilePQ:
			pq = k.KeyRef
		}
	}
	if gost == "" {
		return nil, refuse(CodeTokenMissing, "ключ не загружен")
	}
	if profile == dom.ProfileHybrid {
		if pq == "" {
			return nil, refuse(CodeTokenMissing, "обязателен профиль hybrid — загрузите и ключ ML-DSA (…-ta-pq@1.key.json)")
		}
		return []string{gost, pq}, nil
	}
	return []string{gost}, nil
}

// buildCommand — событие-команда по соглашению «подписан запрос»: тот же
// RequestData и CommandEvent, что у сервера (ExpectRequest) и demo-signer.
func buildCommand(cr procs.SignBlockCommandRequest, eventType string, level int, body []byte, person string, signers []string, c Context) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil || m == nil {
		return nil, refuse(CodeInvalid, "payload_b64 — не тело команды в JSON")
	}
	data, err := dom.RequestData(cr.Operation, map[string]string(cr.Params), m)
	if err != nil {
		return nil, refuse(CodeInvalid, "тело команды: %v", err)
	}
	cmdID, _ := m["command_id"].(string)
	if cmdID == "" {
		return nil, refuse(CodeInvalid, "в теле команды нет command_id (AD-7)")
	}
	seen := c.SeenCheckpoint
	if cr.SeenCheckpoint != nil && int64(*cr.SeenCheckpoint) > seen {
		seen = int64(*cr.SeenCheckpoint)
	}
	wp := c.WorkplaceID
	if wp == "" {
		wp, _ = m["workplace_id"].(string)
	}
	profile := dom.ProfileGost
	if len(signers) > 1 {
		profile = dom.ProfileHybrid
	}
	ce := dom.CommandEvent{EventType: eventType, CommandID: cmdID, ItemID: deref(cr.ItemID), RunID: deref(cr.RunID), SourceID: person,
		Data: data, Level: level, BasisSeq: num(m["basis_seq"]), PolicySeq: num(m["policy_seq"]), WorkplaceID: wp,
		SeenCheckpoint: seen, ClientSignedAt: stamp(c.Now), Profile: profile, Signers: signers}
	return ce.Canonical()
}

// Sign — подписать подготовленное ключами набора (конверт DSSE, AD-10).
func Sign(p Prepared, b Bundle) (dom.Envelope, error) {
	keys, err := b.Private()
	if err != nil {
		return dom.Envelope{}, err
	}
	return profiles.Signer{Keys: profiles.NewKeyring(keys...)}.Envelope(p.PayloadType, p.Payload(), p.Signers...)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func num(v any) int64 {
	switch x := v.(type) {
	case json.Number:
		n, _ := x.Int64()
		return n
	case float64:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	}
	return 0
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// short — отпечаток в короткой форме для людей: первые 16 знаков.
func short(d string) string {
	h := strings.TrimPrefix(d, dom.HashPrefix)
	if len(h) > 16 {
		h = h[:16]
	}
	return h
}
