package platform

import "time"

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

// Command — вход команды ведущего порта: метаданные и данные операции.
type Command[T any] struct {
	Meta CommandMeta
	Body T
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
