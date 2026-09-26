package signing

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	dom "ant/internal/domain/signing"
)

// Проверка подписи команд модулей решений в общем декораторе (Д-59, FR-66,
// FR-69, FR-139, AD-13, AD-14): quality, nonconformity, analysis, documents
// и прочие команды с уровнем подписи ≥ 1 по каталогу операций. Transport
// передаёт запрос как пришёл; здесь — ожидание ExpectRequest (подписан
// JCS{operation, params, body}) и CheckCommand: подпись агента токена или
// ключа в браузере, demo-signer, бумага с заверением («лист решения»,
// doc_id = command_id). Без подписи — в профилях demo и fixtures принимается
// с пометкой «подпись не проверялась» (Д-30), иначе signing.no_signature_path.

var _ platform.SignatureChecker = (*Service)(nil)

// CheckRequest — порт platform.SignatureChecker. skip — операция проверяет
// подпись сама (акты ключей и сменный рапорт модуля signing, подписи этапов
// документов классом document-signature) или не пишет записей.
func (s *Service) CheckRequest(ctx context.Context, act platform.Action, rq platform.SignedRequest) (platform.Signature, bool, error) {
	if s.d.Registry == nil || act.SignatureLevel < 1 {
		return platform.Signature{}, true, nil
	}
	if ownCheck(act) {
		if slices.Contains(act.Emits, catalog.DocumentSignatureRecorded) {
			return s.checkDocumentSignature(ctx, rq)
		}
		return platform.Signature{}, true, nil
	}
	body, err := requestBody(rq.Body)
	if err != nil {
		// Тело не JSON-объект — разбор отвергнет transport (400); подписывать нечего.
		return platform.Signature{}, true, nil
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	et := signedEventType(rq.Meta.Signature, act.Emits)
	e, err := ExpectRequest(act.ID, rq.Params, body, et, rq.Meta.CommandID, rq.Params["item_id"], actor, act.SignatureLevel, act.Critical)
	if err != nil {
		return platform.Signature{}, false, err
	}
	// Бумага без документа — «лист решения» (Д-59): заверитель — пользователь
	// сеанса, подписант — указанный в заверении.
	e.PaperAllowed = true
	a, err := s.CheckCommand(ctx, rq.Meta.Signature, e)
	if err != nil {
		return platform.Signature{}, false, err
	}
	out := platform.Signature{Status: string(a.Status), Method: a.Method, Provenance: a.Provenance, SignerPersonID: a.SignerPersonID,
		AttestedBy: a.AttestedBy, KeyStorage: a.KeyStorage, StorageVariant: a.StorageVariant, KeyRefs: a.KeyRefs}
	if a.Status == dom.StatusValid {
		out.Envelope = json.RawMessage(bytes.Clone(a.Envelope))
		out.ItemID = signedItem(a.Payload)
	}
	return out, false, nil
}

// ownCheck — операции, которые проверяют подпись сами другим классом пакета.
func ownCheck(act platform.Action) bool {
	if act.Owner == "signing" || len(act.Emits) == 0 {
		return true
	}
	return slices.Contains(act.Emits, catalog.DocumentSignatureRecorded)
}

// requestBody — тело запроса объектом с точными числами (как у клиента).
func requestBody(raw json.RawMessage) (map[string]any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// signedEventType — тип записи под подписью, если он из эмитируемых
// операцией (у операций с несколькими типами клиент выбирает свой); иначе —
// первый эмитируемый тип.
func signedEventType(raw []byte, emits []catalog.Type) string {
	if len(emits) == 0 {
		return ""
	}
	if _, payload, err := dom.ParseEnvelope(raw); err == nil {
		var ev struct {
			EventType string `json:"event_type"`
		}
		if json.Unmarshal(payload, &ev) == nil && slices.Contains(emits, catalog.Type(ev.EventType)) {
			return ev.EventType
		}
	}
	return string(emits[0])
}

// signedItem — item_id под подписью (пусто — клиент изделие не указал).
func signedItem(payload []byte) string {
	var ev struct {
		ItemID string `json:"item_id"`
	}
	_ = json.Unmarshal(payload, &ev)
	return ev.ItemID
}

// checkDocumentSignature — подпись этапа документа пакетом класса
// document-signature (AD-12, AD-14, эпик 28): агент токена подписал
// содержимое документа (signing_payload_b64), отпечаток содержимого равен
// doc_digest команды, подписант — пользователь сеанса. Без конверта —
// прежний путь модуля documents (подпись над отпечатком или демо, Д-30).
func (s *Service) checkDocumentSignature(ctx context.Context, rq platform.SignedRequest) (platform.Signature, bool, error) {
	env, payload, err := dom.ParseEnvelope(rq.Meta.Signature)
	if err != nil || len(env.Signatures) == 0 {
		return platform.Signature{}, true, nil
	}
	if class, _, _ := dom.ParsePayloadType(env.PayloadType); class != dom.ClassDocumentSignature {
		return platform.Signature{}, true, nil
	}
	body, err := requestBody(rq.Body)
	if err != nil {
		return platform.Signature{}, false, fail(errcodes.ApiValidationFailed, "тело команды не JSON-объект", "field", "body", "reason", err.Error())
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	a, err := s.verify(ctx, rq.Meta.Signature, payload, dom.ClassDocumentSignature, actor, false)
	if err != nil {
		return platform.Signature{}, false, err
	}
	if want, _ := body["doc_digest"].(string); want != "" && dom.Digest(payload) != want {
		return platform.Signature{}, false, fail(errcodes.SigningDocumentChanged, "подписано содержимое с отпечатком "+dom.Digest(payload)+
			", а команда называет "+want+" — подпишите заново", "doc_id", rq.Params["document_id"])
	}
	return platform.Signature{Status: string(a.Status), Method: a.Method, Provenance: a.Provenance, SignerPersonID: a.SignerPersonID,
		KeyStorage: a.KeyStorage, StorageVariant: a.StorageVariant, KeyRefs: a.KeyRefs,
		Envelope: json.RawMessage(bytes.Clone(rq.Meta.Signature))}, false, nil
}
