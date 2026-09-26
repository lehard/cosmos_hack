package signing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/signing"
)

// Проверка подписи команды человека (FR-66, FR-68, FR-69, FR-139, AD-10,
// AD-11, AD-13, AD-43) — одна функция для всех модулей, чьи операции несут
// уровень подписи ≥ 1 (quality, nonconformity, analysis, documents, signing):
// модуль передаёт, что именно должно быть подписано, и получает итог — кто
// подписал, каким путём (агент токена, бумага с заверением, демо-подписант),
// класс происхождения для записи и конверт, который пишется в журнал.
// Порт подписи один, путей два (FR-139).

// Expect — что должна подписать команда.
type Expect struct {
	// Class — класс пакета цифровой подписи: event (решение — событие-команда,
	// его подписывает агент токена), key-act (акт ключа), document-signature.
	Class string
	// EventType, CommandID, ItemID — событие-команда: event_id = command_id (AD-7).
	EventType string
	CommandID string
	ItemID    string
	// Data — ожидаемое содержимое data (сравнивается канонически); nil — не сравнивается.
	Data json.RawMessage
	// Actor — пользователь сеанса (Principal.PersonID): субъект ключа подписанта
	// (или заверитель — для бумаги), иначе «чужой ключ».
	Actor string
	// Level — уровень подписи операции (x-ant-action), Critical — критическое действие.
	Level    int
	Critical bool
	// Бумага (AD-43): документ и его отпечаток (пусто — лист решения Sheet),
	// этап, ожидаемый подписант этапа, допуск бумаги и полномочие заверителя.
	DocumentID        string
	DocDigest         string
	Stage             int
	Signer            string
	PaperAllowed      bool
	AttesterAuthority string
	// CoSigners — допускаются подписи других людей (вторая подпись акта ключа).
	CoSigners bool
}

// Accepted — итог принятой подписи.
type Accepted struct {
	// Status — valid или unsigned (демо без агента токена: «подпись не проверялась», Д-30).
	Status dom.Status
	// Method — token_agent | paper | demo_signer (document.signature.recorded.method).
	Method string
	// Provenance — класс происхождения подписи записи (AD-2): personal | paper | scenario.
	Provenance     string
	SignerPersonID string
	// Бумага: заверитель, учётный номер оригинала в архиве ОТК, адрес скана.
	AttestedBy      string
	PaperOriginalNo string
	ScanAddress     string
	// KeyRefs — действительные подписи; Profile — профиль пакета.
	KeyRefs []string
	Profile string
	// Envelope — подписанный конверт для записи в журнал как есть (nil — без подписи).
	Envelope []byte
	Payload  []byte
	// CoSigners — прочие подписанты пакета (человек, ключ, класс доверия).
	CoSigners []dom.Approval
	// ExtraValid — действительные подписи сверх перечня обязательных (подтверждение ротации действующим ключом).
	ExtraValid []string
	Verdict    dom.Verdict
	Note       string
}

// NotChecked — пометка решения без подписи в демо (Д-28, Д-30).
const NotChecked = "подпись не проверялась — демо без агента токена"

// Sheet — лист решения для печати на бумаге (AD-12, FR-139): канонический
// JSON того, что подписывает человек ручкой, его отпечаток и id (= command_id).
// Печатная рамка несёт QR ant:doc:‹command_id›:‹отпечаток›.
func Sheet(e Expect) (string, string, []byte, error) {
	if e.DocumentID != "" && e.DocDigest != "" {
		return e.DocumentID, e.DocDigest, nil, nil
	}
	c, err := dom.CanonicalOf(map[string]any{"format_version": 1, "event_type": e.EventType, "command_id": e.CommandID,
		"item_id": e.ItemID, "data": e.Data})
	if err != nil {
		return "", "", nil, err
	}
	return e.CommandID, dom.Digest(c), c, nil
}

// CheckCommand — подпись команды: цифровая (агент токена или demo-signer) или
// бумажная (заверение вторым человеком). raw — поле signature команды.
func (s *Service) CheckCommand(ctx context.Context, raw []byte, e Expect) (Accepted, error) {
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return s.unsigned(e)
	}
	env, payload, err := dom.ParseEnvelope(raw)
	if err != nil {
		return Accepted{}, fail(errcodes.SigningPackageTampered, err.Error())
	}
	if len(env.Signatures) == 0 {
		return s.unsigned(e)
	}
	if err := dom.CheckCanonical(payload); err != nil {
		return Accepted{}, fail(errcodes.SigningPackageTampered, "подписанное содержимое не в каноническом виде (AD-10)")
	}
	class, _, _ := dom.ParsePayloadType(env.PayloadType)
	if class == dom.ClassPaperAttestation {
		return s.checkPaper(ctx, raw, payload, e)
	}
	if class != e.Class {
		return Accepted{}, fail(errcodes.SigningPackageTampered, "подписан пакет класса "+class+", ожидался "+e.Class)
	}
	a, err := s.verify(ctx, raw, payload, class, e.Actor, e.CoSigners)
	if err != nil {
		return a, err
	}
	if err := checkEvent(payload, e); err != nil {
		return a, err
	}
	return a, nil
}

// unsigned — команда без подписи: в профилях demo и fixtures — принимается с
// пометкой «подпись не проверялась» (Д-30); иначе — нет пути подписи (FR-69).
func (s *Service) unsigned(e Expect) (Accepted, error) {
	if e.Level <= dom.Level0 || s.cfg.AllowUnsigned {
		return Accepted{Status: dom.StatusUnsigned, Method: dom.MethodTokenAgent, Provenance: dom.ProvPersonal,
			SignerPersonID: e.Actor, Note: NotChecked}, nil
	}
	return Accepted{}, fail(errcodes.SigningNoSignaturePath, "команда уровня подписи "+fmt.Sprint(e.Level)+" без подписи")
}

// verify — криптография, решение по пакету на текущей голове журнала и
// сверка подписанта с пользователем сеанса (AD-11: «чужой ключ»).
func (s *Service) verify(ctx context.Context, raw, payload []byte, class, actor string, cosigners bool) (Accepted, error) {
	v, keys, err := s.Judge(ctx, raw)
	if err != nil {
		return Accepted{}, err
	}
	a := Accepted{Verdict: v, Envelope: raw, Payload: payload, Profile: v.Required}
	for _, x := range v.Extra {
		if x.Status == dom.StatusValid {
			a.ExtraValid = append(a.ExtraValid, x.KeyRef)
		}
	}
	if ig, err := dom.IntegrityOf(payload); err == nil {
		a.Profile = ig.CryptoProfile
	}
	switch v.Status {
	case dom.StatusValid:
	case dom.StatusUnverifiable:
		return a, fail(errcodes.SigningKeyUnavailable, v.Detail, "key_ref", firstRef(v))
	case dom.StatusDoubtful, dom.StatusRejected:
		return a, fail(reasonCode(v.Reason), v.Detail, "key_ref", firstRef(v), "object_class", class, "required", v.Required)
	default:
		return a, fail(errcodes.SigningPackageTampered, v.Detail)
	}
	mine := false
	for _, sv := range v.Signers {
		a.KeyRefs = append(a.KeyRefs, sv.KeyRef)
		k, _ := keys.Key(sv.KeyRef)
		if sv.SubjectID == actor && (k.SubjectKind == dom.SubjectPerson || k.SubjectKind == dom.SubjectDemoPersona) {
			mine = true
			a.SignerPersonID = actor
			a.Method, a.Provenance = dom.MethodTokenAgent, dom.ProvPersonal
			if k.SubjectKind == dom.SubjectDemoPersona || k.Provenance == dom.ProvScenario {
				a.Method, a.Provenance = dom.MethodDemoSigner, dom.ProvScenario
			}
			continue
		}
		if !slices.ContainsFunc(a.CoSigners, func(x dom.Approval) bool { return x.PersonID == sv.SubjectID }) {
			a.CoSigners = append(a.CoSigners, dom.Approval{PersonID: sv.SubjectID, KeyRef: sv.KeyRef, Provenance: k.Provenance, Valid: true})
		}
	}
	if !mine || (!cosigners && len(a.CoSigners) > 0) {
		ref := firstRef(v)
		alert := dom.KeyAlert{Alert: "foreign_key", KeyRef: ref, Person: actor,
			Detail: "пользователь сеанса " + actor + " не является субъектом ключа подписи"}
		return a, s.raise(ctx, alert, fail(errcodes.SigningForeignKey, alert.Detail, "key_ref", ref))
	}
	a.Status = dom.StatusValid
	return a, nil
}

// checkEvent — подписано ровно то, что исполняет команда: тип, command_id,
// изделие, data; уровень подписи допустим для типа (AD-13).
func checkEvent(payload []byte, e Expect) error {
	var ev struct {
		EventID   string          `json:"event_id"`
		EventType string          `json:"event_type"`
		ItemID    string          `json:"item_id"`
		Data      json.RawMessage `json:"data"`
		Command   struct {
			CommandID      string `json:"command_id"`
			SignatureLevel int    `json:"signature_level"`
		} `json:"command"`
	}
	if err := json.Unmarshal(payload, &ev); err != nil {
		return fail(errcodes.SigningPackageTampered, "подписанное содержимое — не событие-команда")
	}
	switch {
	case e.EventType != "" && ev.EventType != e.EventType:
		return fail(errcodes.SigningDocumentChanged, "подписано "+ev.EventType+", команда исполняет "+e.EventType, "doc_id", e.CommandID)
	case e.CommandID != "" && ev.EventID != e.CommandID && ev.Command.CommandID != e.CommandID:
		return fail(errcodes.SigningDocumentChanged, "подпись относится к другой команде", "doc_id", e.CommandID)
	case e.ItemID != "" && ev.ItemID != e.ItemID:
		return fail(errcodes.SigningDocumentChanged, "подписано другое изделие", "doc_id", e.CommandID)
	}
	if e.Data != nil {
		a, err1 := dom.Canonical(ev.Data)
		b, err2 := dom.Canonical(e.Data)
		if err1 != nil || err2 != nil || !bytes.Equal(a, b) {
			return fail(errcodes.SigningDocumentChanged, "подписанное содержимое расходится с командой — подпишите заново", "doc_id", e.CommandID)
		}
	}
	if err := dom.CheckLevel(ev.EventType, e.Level, ev.Command.SignatureLevel, e.Critical); err != nil {
		return fail(errcodes.SigningLevelNotAllowed, err.Error(), "action_id", ev.EventType,
			"required", fmt.Sprint(dom.MinLevel(ev.EventType, e.Level, e.Critical)), "actual", fmt.Sprint(ev.Command.SignatureLevel))
	}
	return nil
}

// checkPaper — бумага с заверением (AD-43, FR-139): заверитель — пользователь
// сеанса — подписал заверение уровнем 2; QR со скана — этого документа;
// заверитель ≠ подписант; полномочие заверителя; скан загружен.
func (s *Service) checkPaper(ctx context.Context, raw, payload []byte, e Expect) (Accepted, error) {
	var pa dom.PaperAttestation
	if err := json.Unmarshal(payload, &pa); err != nil {
		return Accepted{}, fail(errcodes.SigningPackageTampered, "заверение не по контракту")
	}
	a, err := s.verify(ctx, raw, payload, dom.ClassPaperAttestation, e.Actor, false)
	if err != nil {
		return a, err
	}
	docID, digest, _, err := Sheet(e)
	if err != nil {
		return a, err
	}
	facts := dom.PaperFacts{}
	if s.d.Scans != nil && s.d.QR != nil {
		img, err := s.d.Scans.Scan(ctx, pa.ScanAddress)
		if err != nil {
			return a, fail(errcodes.SigningQrMismatch, "скан "+pa.ScanAddress+" не найден в хранилище материалов", "qr_digest", "—", "doc_digest", digest)
		}
		facts.ScanAddress = dom.Digest(img)
		if facts.ScanQR, err = s.d.QR.ReadQR(img); err != nil {
			return a, fail(errcodes.SigningQrMismatch, "QR на скане не читается: "+err.Error(), "qr_digest", "—", "doc_digest", digest)
		}
	}
	auth := e.AttesterAuthority
	if auth == "" {
		auth = dom.AuthPaperAttestation
	}
	if s.d.Authorities != nil {
		head, _ := s.head(ctx)
		facts.AttesterHasAuthority, _ = s.d.Authorities.Has(ctx, e.Actor, auth, head)
	}
	signer := e.Signer
	exp := dom.PaperExpect{DocumentID: docID, DocDigest: digest, Stage: e.Stage, Signer: signer, PaperAllowed: e.PaperAllowed, AttesterAuthority: auth}
	if exp.Stage == 0 {
		exp.Stage = pa.Stage
	}
	if err := dom.CheckPaper(exp, pa, facts); err != nil {
		return a, paperError(err, facts.ScanQR, digest, exp.Stage)
	}
	a.Method, a.Provenance = dom.MethodPaper, dom.ProvPaper
	a.SignerPersonID, a.AttestedBy = pa.SignerPersonID, pa.AttestedBy
	a.PaperOriginalNo, a.ScanAddress = pa.PaperOriginalNo, pa.ScanAddress
	return a, nil
}

func paperError(err error, qr, digest string, stage int) error {
	_, qdg, _ := dom.ParseQR(qr)
	switch {
	case errors.Is(err, dom.ErrQR):
		return fail(errcodes.SigningQrMismatch, err.Error(), "qr_digest", qdg, "doc_digest", digest)
	case errors.Is(err, dom.ErrAttesterSigner):
		return fail(errcodes.SigningAttesterIsSigner, err.Error())
	case errors.Is(err, dom.ErrPaperForbidden):
		return fail(errcodes.SigningPaperForbidden, err.Error(), "stage", fmt.Sprint(stage))
	case errors.Is(err, dom.ErrLevel):
		return fail(errcodes.SigningLevelNotAllowed, err.Error(), "action_id", "documents.paper.attest", "required", "2", "actual", "1")
	}
	return fail(errcodes.ApiValidationFailed, err.Error(), "field", "signature", "reason", err.Error())
}

// Judge — решение по конверту на текущей голове журнала (AD-10, AD-32):
// реестр ключей и профилей на момент проверки. Для записанных записей
// момент подписи — их seq и committed_at (JudgeAt).
func (s *Service) Judge(ctx context.Context, raw []byte) (dom.Verdict, *dom.Registry, error) {
	head, err := s.head(ctx)
	if err != nil {
		return dom.Verdict{}, nil, err
	}
	return s.JudgeAt(ctx, raw, head+1, s.now())
}

// JudgeAt — решение по конверту на позиции seq и моменте committed_at.
func (s *Service) JudgeAt(ctx context.Context, raw []byte, seq int64, at time.Time) (dom.Verdict, *dom.Registry, error) {
	keys, book, _, err := s.d.Registry.Snapshot(ctx)
	if err != nil {
		return dom.Verdict{}, nil, err
	}
	if s.d.Crypto == nil {
		return dom.Verdict{Status: dom.StatusUnverifiable, Reason: dom.ReasonUnavailable, Detail: "проверка подписи не подключена"}, keys, nil
	}
	env, payload, checks, err := s.d.Crypto.Check(ctx, raw)
	if err != nil {
		return dom.Verdict{}, keys, fail(errcodes.SigningPackageTampered, err.Error())
	}
	class, _, _ := dom.ParsePayloadType(env.PayloadType)
	ig, err := dom.IntegrityOf(payload)
	if err != nil {
		return dom.Verdict{}, keys, fail(errcodes.SigningPackageTampered, "нет блока целостности под подписью")
	}
	present := make([]string, 0, len(env.Signatures))
	for _, sg := range env.Signatures {
		present = append(present, sg.KeyID)
	}
	j := dom.Judgement{Class: class, Declared: ig, Present: present, Crypto: checks, Seq: seq, At: at}
	return dom.Judge(keys, book, j), keys, nil
}

// raise — тревога по ключу в шину безопасности (AD-24) вместе с отказом.
func (s *Service) raise(ctx context.Context, a dom.KeyAlert, err error) error {
	if s.d.Alerts == nil || s.d.Journal == nil {
		return err
	}
	if ps, e := s.d.Alerts.KeyAlert(ctx, a); e == nil && len(ps) > 0 {
		_, _ = s.d.Journal.Append(ctx, journal.AppendRequest{Batch: ps})
	}
	return err
}

func firstRef(v dom.Verdict) string {
	for _, sv := range v.Signers {
		if sv.Status != dom.StatusValid {
			return sv.KeyRef
		}
	}
	if len(v.Signers) > 0 {
		return v.Signers[0].KeyRef
	}
	return ""
}

// reasonCode — код отказа по причине решения.
func reasonCode(r string) errcodes.Code {
	switch r {
	case dom.ReasonDowngrade:
		return errcodes.SigningProfileDowngrade
	case dom.ReasonRevoked, dom.ReasonKeyInactive, dom.ReasonCompromised:
		return errcodes.SigningKeyRevoked
	case dom.ReasonUnavailable:
		return errcodes.SigningKeyUnavailable
	case dom.ReasonUnknownKey, dom.ReasonClassForbidden:
		return errcodes.SigningForeignKey
	}
	return errcodes.SigningPackageTampered
}

// fail — ошибка порта с кодом и параметрами шаблона.
func fail(code errcodes.Code, detail string, kv ...string) *platform.Error {
	e := platform.Fail(code, kv...)
	e.Detail = detail
	return e
}

// SignerKey — ключ ГОСТ человека по соглашению демо-набора: ‹псевдоним›@1 в нижнем регистре.
func SignerKey(person string) string { return signerRef(person) }

// ExpectRequest — ожидание подписи команды по соглашению «подписан запрос»
// (domain/signing.RequestData): модуль-исполнитель передаёт operationId,
// параметры пути, тело команды, тип записи-решения и уровень операции.
func ExpectRequest(operation string, params map[string]string, body any, eventType, commandID, itemID, actor string, level int, critical bool) (Expect, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return Expect{}, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return Expect{}, err
	}
	data, err := dom.RequestData(operation, params, m)
	if err != nil {
		return Expect{}, err
	}
	return Expect{Class: dom.ClassEvent, EventType: eventType, CommandID: commandID, ItemID: itemID, Data: data, Actor: actor,
		Level: level, Critical: critical}, nil
}
