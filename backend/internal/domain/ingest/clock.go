package ingest

import "time"

// Flag — флаг аномалии входа; значения — перечисление `flag` схемы
// ingest.anomaly.flagged.v1 (совпадение проверяет тест).
type Flag string

// Флаги аномалий входа (FR-29, FR-33, AD-5).
const (
	FlagFutureTimestamp   Flag = "future_timestamp"
	FlagClockSkew         Flag = "clock_skew"
	FlagSequenceViolation Flag = "sequence_violation"
	FlagUnknownEnumValue  Flag = "unknown_enum_value"
	FlagUnknownDefectType Flag = "unknown_defect_type"
	FlagLateWrite         Flag = "late_write"
)

// Flags — все флаги в порядке контракта.
var Flags = []Flag{FlagFutureTimestamp, FlagClockSkew, FlagSequenceViolation, FlagUnknownEnumValue, FlagUnknownDefectType, FlagLateWrite}

// ClockPolicy — пороги проверки часов источника (конфигурация приёма).
type ClockPolicy struct {
	// FutureTolerance — насколько occurred_at может опережать received_at,
	// прежде чем событие считается «из будущего».
	FutureTolerance time.Duration
	// SkewThreshold — допустимое расхождение часов источника и ядра.
	SkewThreshold time.Duration
}

// DefaultClockPolicy — пороги по умолчанию: NTP через сеть даёт десятки мс,
// при асимметричных маршрутах — 100+ мс; флаг — при расхождении больше 2 с.
var DefaultClockPolicy = ClockPolicy{FutureTolerance: 2 * time.Second, SkewThreshold: 2 * time.Second}

// ClockFlag — флаг часов с расхождением в миллисекундах (плюс — источник впереди).
type ClockFlag struct {
	Flag   Flag
	SkewMS int64
}

// CheckClock — флаги часов (FR-33, AD-5): событие из будущего (occurred_at
// позже received_at больше допуска) и расхождение часов источника выше порога.
// Сдвиг оценивается по каждому источнику: sentAt — время отправки пачки по
// часам источника (edge-агент ставит его в пачку), received — время приёма ядром;
// задержка доставки буфера (поздняя пачка) сдвигом часов не считается.
// Порядок по occurred_at молча не подменяется — только флаг.
func CheckClock(occurred, received time.Time, sentAt *time.Time, p ClockPolicy) []ClockFlag {
	var out []ClockFlag
	if d := occurred.Sub(received); d > p.FutureTolerance {
		out = append(out, ClockFlag{Flag: FlagFutureTimestamp, SkewMS: d.Milliseconds()})
	}
	if sentAt != nil {
		d := sentAt.Sub(received)
		if d > p.SkewThreshold || -d > p.SkewThreshold {
			out = append(out, ClockFlag{Flag: FlagClockSkew, SkewMS: d.Milliseconds()})
		}
	}
	return out
}
