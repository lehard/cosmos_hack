package httpapi

import (
	"reflect"
	"time"

	"ant/internal/application/platform"
)

// MomentQuery — параметры момента чтения (AD-21, AD-22, AD-37, AD-38);
// встраивается во вход каждой операции чтения состояния.
type MomentQuery struct {
	Axis  platform.Axis `query:"axis" doc:"Ось момента: occurred — «как было» (по умолчанию), recorded — «что мы знали» (AD-37)."`
	AsOf  string        `query:"as_of" format:"date-time" doc:"Момент (RFC 3339 UTC); пусто — «сейчас». Задан — воспроизведение: команды выключены (AD-21)."`
	RunID string        `query:"run_id" maxLength:"128" doc:"Прогон сценария: данные в его пространстве имён (AD-38)."`
}

// Moment разбирает параметры момента.
func (q *MomentQuery) Moment() (platform.Moment, error) {
	return platform.ParseMoment(string(q.Axis), q.AsOf, q.RunID)
}

// PageQuery — параметры страницы списка.
type PageQuery struct {
	Cursor string `query:"cursor" maxLength:"512" doc:"Курсор следующей страницы из ответа; пусто — первая страница."`
	Limit  int    `query:"limit" minimum:"0" maximum:"500" doc:"Размер страницы (по умолчанию 50)."`
}

// Page — страница для порта.
func (q *PageQuery) Page() platform.Page {
	l := q.Limit
	if l == 0 {
		l = 50
	}
	return platform.Page{Cursor: q.Cursor, Limit: l}
}

// CommandBody — метаданные команды в теле (AD-7, AD-39, AD-14); встраивается
// в тело каждой команды.
type CommandBody struct {
	CommandID   string          `json:"command_id" format:"uuid" doc:"UUIDv7 клиента; повтор с тем же id возвращает прежний ответ (AD-7). У подписанной команды — event_id пакета."`
	BasisSeq    int64           `json:"basis_seq" minimum:"0" doc:"seq, на котором клиент видел объект (basis_seq из ответа чтения): после него в потоках гарда не должно быть новых записей guard_relevant, иначе 409 journal.stale_state (AD-39)."`
	PolicySeq   int64           `json:"policy_seq" minimum:"0" doc:"Версия политики, по которой показаны права (policy_seq сеанса); изменилась — 409 journal.stale_policy (AD-39)."`
	WorkplaceID string          `json:"workplace_id,omitempty" maxLength:"128" doc:"Рабочее место сеанса (барьер 2, AD-15)."`
	Signature   *SignedEnvelope `json:"signature,omitempty" doc:"Подписанный пакет DSSE для операций с уровнем подписи ≥ 1 (AD-10, AD-13, AD-14): подписывает агент токена, сервер сверяет отпечаток."`
}

// CommandMeta — метаданные команды для порта.
func (b *CommandBody) CommandMeta() platform.CommandMeta {
	m := platform.CommandMeta{CommandID: b.CommandID, BasisSeq: b.BasisSeq, PolicySeq: b.PolicySeq, WorkplaceID: b.WorkplaceID}
	if b.Signature != nil {
		m.Signature = b.Signature.raw()
	}
	return m
}

// SignedEnvelope — конверт DSSE (contracts/crypto/dsse-envelope.schema.json, AD-10).
type SignedEnvelope struct {
	PayloadType string              `json:"payloadType" maxLength:"256" doc:"application/vnd.ant.‹класс›+json; v=‹версия›."`
	Payload     string              `json:"payload" contentEncoding:"base64" doc:"Канонический JSON (RFC 8785) подписываемого содержимого, base64."`
	Signatures  []EnvelopeSignature `json:"signatures" minItems:"1" doc:"Подписи."`
}

// EnvelopeSignature — подпись в конверте DSSE.
type EnvelopeSignature struct {
	KeyID string `json:"keyid" maxLength:"128" doc:"key_id@версия подписанта."`
	Sig   string `json:"sig" contentEncoding:"base64" doc:"Подпись, base64."`
}

func (e *SignedEnvelope) raw() []byte {
	b, _ := jsonMarshal(e)
	return b
}

// commandMeta достаёт метаданные команды из поля Body входа (если тело встраивает CommandBody).
func commandMeta(in any) (platform.CommandMeta, bool) {
	v := reflect.ValueOf(in)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return platform.CommandMeta{}, false
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return platform.CommandMeta{}, false
	}
	f := v.FieldByName("Body")
	if !f.IsValid() {
		return platform.CommandMeta{}, false
	}
	if f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return platform.CommandMeta{}, false
		}
		f = f.Elem()
	}
	if !f.CanAddr() {
		return platform.CommandMeta{}, false
	}
	c, ok := f.Addr().Interface().(interface{ CommandMeta() platform.CommandMeta })
	if !ok {
		return platform.CommandMeta{}, false
	}
	return c.CommandMeta(), true
}

// hasCommandMeta — проверка на этапе регистрации: тело команды встраивает CommandBody.
func hasCommandMeta[I any]() bool {
	t := reflect.TypeFor[I]()
	f, ok := t.FieldByName("Body")
	if !ok {
		return false
	}
	bt := f.Type
	if bt.Kind() == reflect.Pointer {
		bt = bt.Elem()
	}
	return reflect.PointerTo(bt).Implements(reflect.TypeFor[interface{ CommandMeta() platform.CommandMeta }]())
}

// Meta — заголовки ответа каждой операции: режим ведущих портов Ant-Backend
// (AD-36, FR-150). Встраивается в выход операции (Out, NoBody или свой).
type Meta struct {
	Backend platform.Mode `header:"Ant-Backend" doc:"Режим ведущих портов, отдавших ответ: метка fixtures | live на виджете (AD-36, FR-150)."`
}

func (m *Meta) setMode(x platform.Mode) { m.Backend = x }

// Out — выход операции: заголовки Meta и тело.
type Out[T any] struct {
	Meta
	Body T
}

// OK — выход с телом.
func OK[T any](v T) *Out[T] { return &Out[T]{Body: v} }

// NoBody — выход без тела (204): только заголовки.
type NoBody struct {
	Meta
}

// Receipt — тело ответа на команду (AD-7, AD-28).
type Receipt struct {
	CommandID  string     `json:"command_id" doc:"id команды."`
	Seq        int64      `json:"seq" doc:"Позиция записи-решения в журнале; для следующей команды по объекту — новый basis_seq."`
	EventIDs   []string   `json:"event_ids" doc:"Записанные записи журнала."`
	CARef      string     `json:"ca_ref,omitempty" doc:"Номер критического действия CA-‹n› (AD-28)."`
	RecordedAt *time.Time `json:"recorded_at,omitempty" doc:"Доменное время записи (AD-37)."`
	Replayed   bool       `json:"replayed" doc:"Повтор с тем же command_id — возвращён прежний ответ (AD-7)."`
}

// ReceiptOf — тело ответа из квитанции порта.
func ReceiptOf(r platform.Receipt) *Out[Receipt] {
	out := Receipt{CommandID: r.CommandID, Seq: r.Seq, EventIDs: r.EventIDs, CARef: r.CARef, Replayed: r.Replayed}
	if out.EventIDs == nil {
		out.EventIDs = []string{}
	}
	if !r.RecordedAt.IsZero() {
		t := r.RecordedAt
		out.RecordedAt = &t
	}
	return OK(out)
}

// Cmd — команда для порта из тела, встраивающего CommandBody.
func Cmd[T any](meta interface{ CommandMeta() platform.CommandMeta }, body T) platform.Command[T] {
	return platform.Command[T]{Meta: meta.CommandMeta(), Body: body}
}
