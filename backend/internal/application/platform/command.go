package platform

import (
	"encoding/json"
	"time"

	"ant/internal/contracts/crypto"
)

// CommandMeta — метаданные команды API (AD-7, AD-39, AD-14): приходят в теле
// каждой команды и попадают в блок command конверта записи-решения.
type CommandMeta struct {
	// CommandID — UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7).
	CommandID string
	// BasisSeq — seq, на котором клиент видел объект (из ответа чтения): гард
	// проверяет, что после него в потоках нет новых записей guard_relevant (AD-39).
	BasisSeq int64
	// PolicySeq — версия политики, по которой клиенту показаны права (AD-15, AD-39).
	PolicySeq int64
	// WorkplaceID — рабочее место сеанса (барьер 2, AD-15).
	WorkplaceID string
	// Reason — обоснование человека, если операция его требует (например, отклонение сигнала).
	Reason string
	// Signature — подписанный пакет DSSE для операций уровня подписи ≥ 1 (AD-10, AD-13);
	// пусто — уровень 0 или подпись собирается маршрутом документа.
	Signature []byte
}

// CommandHeader — метаданные команды в теле запроса (AD-7, AD-39, AD-14):
// встраивается в тип данных каждой команды слоя application; transport
// проверяет при регистрации, что тело команды его содержит.
type CommandHeader struct {
	CommandID   string               `json:"command_id" format:"uuid" doc:"UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7). У подписанной команды — event_id пакета."`
	BasisSeq    int64                `json:"basis_seq" minimum:"0" doc:"seq, на котором клиент видел объект (basis_seq из ответа чтения): после него в потоках гарда не должно быть новых записей guard_relevant, иначе 409 journal.stale_state (AD-39)."`
	PolicySeq   int64                `json:"policy_seq" minimum:"0" doc:"Версия политики, по которой показаны права (policy_seq сеанса); изменилась — 409 journal.stale_policy (AD-39)."`
	WorkplaceID string               `json:"workplace_id,omitempty" maxLength:"128" doc:"Рабочее место сеанса (барьер 2, AD-15)."`
	Signature   *crypto.DsseEnvelope `json:"signature,omitempty" doc:"Подписанный пакет DSSE для операций с уровнем подписи ≥ 1 (AD-10, AD-13, AD-14): подписывает агент токена, сервер сверяет отпечаток."`
}

// CommandMeta — метаданные команды для сценария и декоратора.
func (h *CommandHeader) CommandMeta() CommandMeta {
	m := CommandMeta{CommandID: h.CommandID, BasisSeq: h.BasisSeq, PolicySeq: h.PolicySeq, WorkplaceID: h.WorkplaceID}
	if h.Signature != nil {
		m.Signature, _ = json.Marshal(h.Signature)
	}
	return m
}

// Receipt — квитанция принятой команды (AD-7, AD-28).
type Receipt struct {
	// CommandID — id команды.
	CommandID string
	// Seq — позиция записи-решения в журнале (порядок знания).
	Seq int64
	// EventIDs — записанные записи (решение и, если есть, документ, задача…).
	EventIDs []string
	// CARef — номер критического действия `CA-‹n›`, если операция критическая (AD-28).
	CARef string
	// RecordedAt — доменное время записи (AD-37).
	RecordedAt time.Time
	// Replayed — повтор с тем же command_id: возвращён прежний ответ (AD-7).
	Replayed bool
}

// Mode — режим ведущих портов, отдавших ответ (AD-36, FR-150).
type Mode string

const (
	// ModeFixtures — заготовки за ведущими портами.
	ModeFixtures Mode = "fixtures"
	// ModeLive — сценарии приложения над доменом и журналом.
	ModeLive Mode = "live"
)
