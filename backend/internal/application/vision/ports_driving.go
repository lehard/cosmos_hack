package vision

import (
	"context"

	"ant/internal/application/platform"
)

// Queries — ведущий порт чтения модуля vision (AD-36).
type Queries interface {
	// Analyzers — анализаторы и уровень доверия паспортов (vision.analyzer.list).
	Analyzers(ctx context.Context, m platform.Moment) (AnalyzerList, error)
	// Passport — паспорт допуска (vision.passport.read).
	Passport(ctx context.Context, passportID string, m platform.Moment) (AnalyzerPassport, error)
	// Checks — отчёты проверки анализатора (vision.check.list).
	Checks(ctx context.Context, passportID string, m platform.Moment, p platform.Page) (AnalyzerCheckList, error)
}

// Commands — ведущий порт команд модуля vision (AD-39).
type Commands interface {
	AdmitPassport(ctx context.Context, in AdmitPassport) (platform.Receipt, error)
	ReinstatePassport(ctx context.Context, passportID string, in ReinstatePassport) (platform.Receipt, error)
	RetirePassport(ctx context.Context, passportID string, in RetirePassport) (platform.Receipt, error)
}

// Unimplemented — заглушка портов vision: каждая операция отвечает 501
// api.not_implemented. Встраивается в реализации live и fixtures.
type Unimplemented struct{}

func ni(op string) error { return platform.NotImplemented(op) }

func (Unimplemented) Analyzers(context.Context, platform.Moment) (AnalyzerList, error) {
	return AnalyzerList{}, ni("vision.analyzer.list")
}
func (Unimplemented) Passport(context.Context, string, platform.Moment) (AnalyzerPassport, error) {
	return AnalyzerPassport{}, ni("vision.passport.read")
}
func (Unimplemented) Checks(context.Context, string, platform.Moment, platform.Page) (AnalyzerCheckList, error) {
	return AnalyzerCheckList{}, ni("vision.check.list")
}
func (Unimplemented) AdmitPassport(context.Context, AdmitPassport) (platform.Receipt, error) {
	return platform.Receipt{}, ni("vision.passport.admit")
}
func (Unimplemented) ReinstatePassport(context.Context, string, ReinstatePassport) (platform.Receipt, error) {
	return platform.Receipt{}, ni("vision.passport.reinstate")
}
func (Unimplemented) RetirePassport(context.Context, string, RetirePassport) (platform.Receipt, error) {
	return platform.Receipt{}, ni("vision.passport.retire")
}

var (
	_ Queries  = Unimplemented{}
	_ Commands = Unimplemented{}
)
