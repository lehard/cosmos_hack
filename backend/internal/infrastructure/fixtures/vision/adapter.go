package vision

import (
	"context"

	"ant/internal/application/platform"
	app "ant/internal/application/vision"
)

// Adapter — реализация fixtures ведущих портов модуля vision (AD-36):
// анализаторы VisionQC, паспорта допуска и проверки — из мира заготовок.
type Adapter struct{}

// New создаёт адаптер заготовок.
func New() *Adapter { return &Adapter{} }

var (
	_ app.Queries  = (*Adapter)(nil)
	_ app.Commands = (*Adapter)(nil)
)

func passport(id string) map[string]string { return map[string]string{"passport_id": id} }

// Analyzers — анализаторы и их паспорта (vision.analyzer.list).
func (Adapter) Analyzers(ctx context.Context, m platform.Moment) (app.AnalyzerList, error) {
	return respond[app.AnalyzerList](ctx, "vision.analyzer.list", nil, &m)
}

// Passport — паспорт допуска анализатора (vision.passport.read).
func (Adapter) Passport(ctx context.Context, passportID string, m platform.Moment) (app.AnalyzerPassport, error) {
	return respond[app.AnalyzerPassport](ctx, "vision.passport.read", passport(passportID), &m)
}

// Checks — проверки анализатора (vision.check.list).
func (Adapter) Checks(ctx context.Context, passportID string, m platform.Moment, _ platform.Page) (app.AnalyzerCheckList, error) {
	return respond[app.AnalyzerCheckList](ctx, "vision.check.list", passport(passportID), &m)
}

// AdmitPassport — допуск версии анализатора (vision.passport.admit).
func (Adapter) AdmitPassport(ctx context.Context, in app.AdmitPassport) (platform.Receipt, error) {
	return decide(ctx, "vision.passport.admit", "analyzer_passport", "", in.CommandMeta())
}

// ReinstatePassport — вернуть анализатор после отката (vision.passport.reinstate).
func (Adapter) ReinstatePassport(ctx context.Context, passportID string, in app.ReinstatePassport) (platform.Receipt, error) {
	return decide(ctx, "vision.passport.reinstate", "analyzer_passport", passportID, in.CommandMeta())
}

// RetirePassport — вывести паспорт (vision.passport.retire).
func (Adapter) RetirePassport(ctx context.Context, passportID string, in app.RetirePassport) (platform.Receipt, error) {
	return decide(ctx, "vision.passport.retire", "analyzer_passport", passportID, in.CommandMeta())
}
