package security

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля security (AD-36): индикатор
// целостности, журнал критических действий, шина безопасности, отчёты
// верификатора — стол Аудитора ИБ (только чтение).
type Queries interface {
	// Integrity — состояние целостности журнала «по данным сервера»
	// (security.integrity.read, AD-46): последний подписанный отчёт верификатора.
	Integrity(ctx context.Context) (IntegrityStatus, error)
	// CriticalActions — журнал критических действий (security.critical_action.list, AD-28, FR-77).
	CriticalActions(ctx context.Context, f CriticalActionFilter, m platform.Moment, p platform.Page) (CriticalActionList, error)
	// CriticalAction — запись CA-‹n› (security.critical_action.read).
	CriticalAction(ctx context.Context, caRef string) (CriticalAction, error)
	// Events — шина безопасности (security.event.list, AD-24, FR-118).
	Events(ctx context.Context, eventType string, m platform.Moment, p platform.Page) (SecurityEventList, error)
	// VerifierReports — отчёты верификатора (security.verifier_report.list, AD-46).
	VerifierReports(ctx context.Context, p platform.Page) (VerifierReportList, error)
	// VerifierReport — отчёт верификатора (security.verifier_report.read).
	VerifierReport(ctx context.Context, reportDigest string) (VerifierReport, error)
}

// CriticalActionFilter — отбор записей журнала критических действий.
type CriticalActionFilter struct {
	Group   string
	ActorID string
	Object  *platform.DrillRef
}

// Commands — ведущий порт команд модуля security (AD-36, AD-39). Команд API
// нет: записи CA пишет только CriticalActions.Execute в транзакции основной
// записи (AD-28), события шины — источники; у администратора приложения пути
// записи в журнал критических действий нет.
type Commands interface{}

// Unimplemented — заглушка портов security: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures, чтобы новые
// операции контракта не ломали сборку (реализация переопределяет метод).
type Unimplemented struct{}

func (Unimplemented) Integrity(context.Context) (IntegrityStatus, error) {
	return IntegrityStatus{}, platform.NotImplemented("security.integrity.read")
}

func (Unimplemented) CriticalActions(context.Context, CriticalActionFilter, platform.Moment, platform.Page) (CriticalActionList, error) {
	return CriticalActionList{}, platform.NotImplemented("security.critical_action.list")
}

func (Unimplemented) CriticalAction(context.Context, string) (CriticalAction, error) {
	return CriticalAction{}, platform.NotImplemented("security.critical_action.read")
}

func (Unimplemented) Events(context.Context, string, platform.Moment, platform.Page) (SecurityEventList, error) {
	return SecurityEventList{}, platform.NotImplemented("security.event.list")
}

func (Unimplemented) VerifierReports(context.Context, platform.Page) (VerifierReportList, error) {
	return VerifierReportList{}, platform.NotImplemented("security.verifier_report.list")
}

func (Unimplemented) VerifierReport(context.Context, string) (VerifierReport, error) {
	return VerifierReport{}, platform.NotImplemented("security.verifier_report.read")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
