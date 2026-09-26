package platform

import (
	"context"
	"encoding/json"

	"ant/internal/contracts/errcodes"
)

// Подпись команды человека в общем декораторе (FR-66, FR-69, AD-10, AD-13,
// AD-14, Д-59): transport до вызова порта модуля передаёт запрос — операцию,
// параметры пути и тело как есть — порту SignatureChecker (реализация —
// модуль signing, CheckCommand с ожиданием ExpectRequest); принятая подпись
// кладётся в контекст, и модуль-исполнитель пишет её рядом с записью-решением
// (блок command конверта: signature, key_storage — CommandFields).

// SignedRequest — запрос команды для проверки подписи: подписывается
// JCS{operation, params, body без полей заголовка} (domain/signing.RequestData).
type SignedRequest struct {
	// Params — параметры пути операции.
	Params map[string]string
	// Body — тело запроса как пришло (байты, до разбора в тип команды).
	Body json.RawMessage
	// Meta — заголовок команды (command_id, basis_seq, policy_seq, signature).
	Meta CommandMeta
}

// Signature — принятая подпись команды.
type Signature struct {
	// Status — valid (подпись проверена) или unsigned (демо без агента токена, Д-30).
	Status string
	// Method — token_agent | paper | demo_signer.
	Method string
	// Provenance — класс происхождения записи (AD-2): personal | paper | scenario.
	Provenance string
	// SignerPersonID — подписант (у бумаги — подписавший ручкой), AttestedBy — заверитель.
	SignerPersonID string
	AttestedBy     string
	// KeyStorage, StorageVariant — класс хранения ключа по акту регистрации
	// (AD-11, AD-14, Д-72): hardware_token | software_browser; extension | page.
	KeyStorage     string
	StorageVariant string
	// KeyRefs — действительные подписи пакета.
	KeyRefs []string
	// ItemID — изделие под подписью (пусто — не указано клиентом): модуль
	// сверяет его с изделием, над которым исполняет команду.
	ItemID string
	// Envelope — конверт клиента как есть (поле signature запроса); nil — без подписи.
	Envelope json.RawMessage
}

// Signed — подпись проверена (не «подпись не проверялась»).
func (s Signature) Signed() bool { return s.Status == "valid" && len(s.Envelope) > 0 }

// CommandFields дописывает подпись в блок command конверта записи-решения
// (contracts/events/common/envelope.v1.json: command.signature,
// command.key_storage, command.signature_method): конверт клиента хранится
// рядом с записью как есть — его пересчитывает верификатор (AD-9).
func (s Signature) CommandFields(cmd map[string]any) {
	if !s.Signed() {
		return
	}
	cmd["signature"] = s.Envelope
	if s.Method != "" {
		cmd["signature_method"] = s.Method
	}
	if s.KeyStorage != "" {
		cmd["key_storage"] = s.KeyStorage
	}
	if s.StorageVariant != "" {
		cmd["key_storage_variant"] = s.StorageVariant
	}
}

// SignatureChecker — порт проверки подписи команды уровня ≥ 1 (Д-59).
// Возвращает принятую подпись; skip — операция проверяет подпись сама
// (акты ключей, подписи документов) или уровня подписи у неё нет.
type SignatureChecker interface {
	CheckRequest(ctx context.Context, act Action, rq SignedRequest) (sig Signature, skip bool, err error)
}

type signatureKey struct{}

// WithSignature кладёт принятую подпись команды в контекст.
func WithSignature(ctx context.Context, s Signature) context.Context {
	return context.WithValue(ctx, signatureKey{}, s)
}

// SignatureFrom — принятая подпись команды из контекста; нет — пусто.
func SignatureFrom(ctx context.Context) (Signature, bool) {
	s, ok := ctx.Value(signatureKey{}).(Signature)
	return s, ok
}

// CheckItem — изделие под подписью совпадает с изделием, над которым модуль
// исполняет команду (AD-14: подписано ровно то, что исполняется). Пустое с
// любой стороны — не сверяется (изделие связано параметрами пути под подписью).
func (s Signature) CheckItem(itemID string) error {
	if s.ItemID == "" || itemID == "" || s.ItemID == itemID {
		return nil
	}
	e := Fail(errcodes.SigningDocumentChanged, "doc_id", itemID)
	e.Detail = "подписано изделие " + s.ItemID + ", команда исполняется над " + itemID
	return e
}

// SignRecord — подпись команды из контекста в блок command записи-решения
// модуля-исполнителя (Д-59): сверка изделия под подписью и поля
// signature, signature_method, key_storage. Возвращает класс происхождения
// записи по способу подписи (personal | paper | scenario; пусто — оставить свой).
func SignRecord(ctx context.Context, cmd map[string]any, itemID string) (string, error) {
	s, ok := SignatureFrom(ctx)
	if !ok {
		return "", nil
	}
	if err := s.CheckItem(itemID); err != nil {
		return "", err
	}
	if cmd != nil {
		s.CommandFields(cmd)
	}
	if !s.Signed() {
		return "", nil
	}
	return s.Provenance, nil
}
