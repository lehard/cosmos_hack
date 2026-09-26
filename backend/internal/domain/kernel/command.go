package kernel

import (
	"time"

	"ant/internal/contracts/errcodes"
)

// Command — команда человека в представлении домена (AD-39, AD-7): операция
// (x-ant-action id), автор, основание и данные. Доменное «сейчас» (OccurredAt)
// ставит application при приёме (AD-37); время клиента в порядок не входит.
type Command struct {
	// Action — id операции `‹модуль›.‹объект›.‹действие›` (AD-40).
	Action string
	// CommandID — UUIDv7 клиента; у подписанной команды — event_id пакета (AD-7).
	CommandID string
	// Actor — псевдоним сотрудника; OnBehalfOf — заместитель по делегированию.
	Actor      string
	OnBehalfOf string
	// Object — поток объекта команды (`item:‹id›`, `nonconformity:‹id›` …).
	Object string
	// BasisSeq, PolicySeq — на каком seq проверено состояние и политика (AD-39).
	BasisSeq  int64
	PolicySeq int64
	// GuardStreams — потоки, которые проверил гард.
	GuardStreams []string
	// OccurredAt — доменное «сейчас» при приёме команды (AD-37).
	OccurredAt time.Time
	// SignatureLevel — уровень подписи 0–3 (AD-13).
	SignatureLevel int
	// Payload — данные команды (тип задаёт модуль-владелец операции).
	Payload any
}

// Refusal — отказ доменного гарда или правила с кодом из contracts/errors.yaml
// (FR-28, AD-39). Реализует error.
type Refusal struct {
	Code   errcodes.Code
	Params map[string]string
	// Detail — пояснение по-русски; пусто — шаблон из каталога.
	Detail string
}

func (r *Refusal) Error() string {
	if r.Detail != "" {
		return string(r.Code) + ": " + r.Detail
	}
	return string(r.Code)
}

// Refuse — отказ с кодом и параметрами шаблона (пары ключ, значение).
func Refuse(code errcodes.Code, kv ...string) *Refusal {
	r := &Refusal{Code: code}
	if len(kv) > 1 {
		r.Params = make(map[string]string, len(kv)/2)
		for i := 0; i+1 < len(kv); i += 2 {
			r.Params[kv[i]] = kv[i+1]
		}
	}
	return r
}

// Guard — сигнатура доменного гарда (AD-39): состояние на basis_seq и команда
// → nil или *Refusal. Гард — чистая функция модуля-владельца операции; её
// вызывают api до записи, свёртка при применении и верификатор.
type Guard[S any] func(state S, cmd Command) error
