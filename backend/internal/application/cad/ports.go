package cad

import (
	"context"

	dom "ant/internal/domain/cad"
)

// Ведомые порты модуля cad (AD-18, AD-35: внешняя система — только за портом).

// Reader — чтение файла условной сборки: проверка схемой и версией формата,
// перевод в сборку на нашем языке. Адаптер: infrastructure/integration/cad/kompas
// (файл по образцу ФЛ-100.00.000 СБ); прямое подключение (COM API7, ЛОЦМАН:PLM)
// — другой адаптер того же порта. Ошибка формата — *FormatError.
type Reader interface {
	Read(ctx context.Context, fileName string, content []byte) (dom.Assembly, error)
}

// FormatError — файл не по контракту: поле (JSON Pointer) и что не так.
// Version — неизвестная версия формата (переобработка после повышателя, AD-20).
type FormatError struct {
	Field   string
	Detail  string
	Version bool
}

func (e *FormatError) Error() string {
	return "файл условной сборки: " + e.Field + ": " + e.Detail
}

// Intake — ведомый порт входа (AD-18: входящие — через обычный приём):
// пачка конвертов источника уходит в приём; ответ — итог по каждому в том же порядке.
type Intake interface {
	Submit(ctx context.Context, sourceID string, envelopes [][]byte) ([]IntakeOutcome, error)
}

// IntakeOutcome — итог приёма конверта.
type IntakeOutcome struct {
	EventID string
	// Outcome — accepted | accepted_with_flag | duplicate | conflict | quarantined.
	Outcome string
	Seq     int64
	Code    string
	Field   string
	Detail  string
}

// Nomenclature — номенклатура учётной системы для сверки обозначений КД
// (FR-95): наши типы изделий, у которых есть соответствие в системе. nil от
// порта (или порт не задан) — номенклатура ещё не получена, сверка не
// выполнялась. Реализация — над записями reference.external_id.mapped
// (проекция reference, эпик 19).
type Nomenclature interface {
	ItemTypes(ctx context.Context) (*dom.Nomenclature, error)
}
