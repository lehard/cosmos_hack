package ingest

import (
	"context"
	"time"
)

// FaultKind — вид сбоя stand-а (таблица сбоев PRD §4.3, research
// data-collection-sources-and-failures.md): включается только со страницы
// тестовых сценариев через служебный порт (AD-18).
type FaultKind string

// Виды сбоев stand-а.
const (
	// FaultOffline — stand не отвечает и не отдаёт данные (недоступность).
	FaultOffline FaultKind = "offline"
	// FaultDelay — задержка ответа или отправки на Param мс.
	FaultDelay FaultKind = "delay"
	// FaultDuplicate — повторная доставка каждого сообщения (at-least-once).
	FaultDuplicate FaultKind = "duplicate"
	// FaultDrop — пропуск сообщений: разрыв номеров источника.
	FaultDrop FaultKind = "drop"
	// FaultReorder — нарушение порядка отправки.
	FaultReorder FaultKind = "reorder"
	// FaultClockSkew — часы stand-а сдвинуты на Param мс.
	FaultClockSkew FaultKind = "clock_skew"
	// FaultCorrupt — сообщение не по контракту (кривое).
	FaultCorrupt FaultKind = "corrupt"
	// FaultError — ответ ошибкой (5xx, отказ протокола).
	FaultError FaultKind = "error"
)

// FaultKinds — все виды сбоев.
var FaultKinds = []FaultKind{FaultOffline, FaultDelay, FaultDuplicate, FaultDrop, FaultReorder, FaultClockSkew, FaultCorrupt, FaultError}

// Fault — включённый сбой: вид, параметр (мс или доля в bp) и срок (нулевой — до снятия).
type Fault struct {
	Kind  FaultKind `json:"kind"`
	Param int64     `json:"param,omitempty"`
	Until time.Time `json:"until,omitzero"`
}

// StandInfo — stand внешней системы или оборудования (AD-18): что эмулирует,
// где граница эмуляции (NFR-TEST-2), какие сбои поддерживает и какие включены.
type StandInfo struct {
	Name      string      `json:"name"`
	Emulates  string      `json:"emulates"`
	Boundary  string      `json:"boundary"`
	Supported []FaultKind `json:"supported"`
	Active    []Fault     `json:"active"`
}

// StandControl — служебный порт сбоев stand-ов (AD-18): страница тестовых
// сценариев (эпики 14, 32, 36) включает и снимает сбои через него, не трогая
// сами stand-ы. Реализация — каркас infrastructure/integration/ingest/stands,
// роль stands (одна копия).
type StandControl interface {
	// Stands — зарегистрированные stand-ы и их сбои.
	Stands(ctx context.Context) ([]StandInfo, error)
	// SetFault включает сбой у stand-а name.
	SetFault(ctx context.Context, name string, f Fault) error
	// ClearFaults снимает все сбои stand-а name (пусто — у всех).
	ClearFaults(ctx context.Context, name string) error
}
