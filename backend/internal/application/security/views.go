package security

import "time"

// IntegrityStatus — индикатор целостности на столах (AD-46, FR-73): ant забирает
// последний подписанный отчёт верификатора у хранителя; индикатор помечен «по
// данным сервера» и желтеет сам, если свежего отчёта нет дольше двух интервалов.
type IntegrityStatus struct {
	Status          string     `json:"status" enum:"ok,violated,stale,unknown" doc:"ok — последний отчёт верификатора «цело»; stale — свежего отчёта нет дольше двух интервалов."`
	CheckedAt       *time.Time `json:"checked_at,omitempty" doc:"Время отчёта верификатора."`
	ReportRef       string     `json:"report_ref,omitempty" doc:"Отпечаток отчёта верификатора."`
	IntervalSeconds int        `json:"interval_seconds" minimum:"1" doc:"Интервал проверок верификатора."`
	ServerSide      bool       `json:"server_side" doc:"Всегда true — показывается «по данным сервера»."`
}
