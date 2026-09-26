package kernel

import (
	"encoding/json"
	"fmt"
	"time"

	"ant/internal/contracts/catalog"
)

// Module — имя модуля системы (AD-1, AD-40): эмитент типов записей и владелец операций.
type Module string

// Record — запись журнала в представлении домена: вход свёртки изделия или
// межизделийной стадии (AD-5, AD-42). Открытые поля — по контракту записи
// (AD-44); Data — канонический JSON поля data, уже повышенный до текущей
// версии схемы (upcast, AD-20).
type Record struct {
	// Seq — позиция в основной цепочке (порядок знания, AD-37).
	Seq int64
	// EventID — UUIDv7 факта или решения, UUIDv5 реакции (AD-44).
	EventID string
	// Type — тип записи из каталога.
	Type catalog.Type
	// SchemaVersion — версия схемы data после повышения.
	SchemaVersion int
	// Kind — вид записи (AD-2).
	Kind catalog.Kind
	// SourceID, SourceKind — источник факта (AD-2, FR-140).
	SourceID   string
	SourceKind string
	// Provenance — класс происхождения подписи (AD-2).
	Provenance string
	// ItemID — внутренний ID изделия (AD-41); пусто у записей вне изделия.
	ItemID string
	// Stream — поток записи `item:‹id›`, `‹вид›:‹id›` или `global` (AD-39).
	Stream string
	// RunID — прогон сценария (AD-38).
	RunID string
	// OccurredAt, ReceivedAt, RecordedAt — время возникновения, приёма и
	// доменное время записи (AD-37). Порядок в изделии — occurred → received → event_id (AD-5).
	OccurredAt time.Time
	ReceivedAt time.Time
	RecordedAt time.Time
	// CorrelationID, CausationID — сквозная цепочка и непосредственная причина.
	CorrelationID string
	CausationID   string
	// BasisSeq — для реакций и решений: seq, на котором вычислено основание (AD-39).
	BasisSeq int64
	// Corrects — event_id исправляемой записи (FR-122) или пусто.
	Corrects string
	// Data — канонический JSON поля data.
	Data json.RawMessage
}

// Decode разбирает Data записи в сгенерированный тип данных события.
func Decode[T any](r Record) (T, error) {
	var v T
	if err := json.Unmarshal(r.Data, &v); err != nil {
		return v, fmt.Errorf("запись %s (%s): %w", r.EventID, r.Type, err)
	}
	return v, nil
}

// Less — порядок записей в пределах изделия: occurred_at → received_at → event_id (AD-5).
func Less(a, b Record) bool {
	if !a.OccurredAt.Equal(b.OccurredAt) {
		return a.OccurredAt.Before(b.OccurredAt)
	}
	if !a.ReceivedAt.Equal(b.ReceivedAt) {
		return a.ReceivedAt.Before(b.ReceivedAt)
	}
	return a.EventID < b.EventID
}

// MaxOccurred — наибольший occurred_at среди причин: время реакции и адресованной
// записи (AD-37). Пустой список — нулевое время.
func MaxOccurred(causes []Record) time.Time {
	var t time.Time
	for _, c := range causes {
		if c.OccurredAt.After(t) {
			t = c.OccurredAt
		}
	}
	return t
}
