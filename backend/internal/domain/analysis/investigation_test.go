package analysis_test

import (
	"errors"
	"testing"

	"ant/internal/contracts/errcodes"
	"ant/internal/domain/analysis"
	"ant/internal/domain/kernel"
)

// Стадия расследования и гард закрытия (кейс §2.3, FR-64): закрыть можно,
// только когда отвечены обе причины и все меры признаны эффективными.
func TestInvestigationStageAndClose(t *testing.T) {
	f := analysis.InvestigationFacts{IncidentID: "RS-1", Causes: map[string]string{}}
	f.Counts.Add(analysis.StatusSuspect)
	if got := analysis.InvestigationStage(f); got != analysis.StageScopeDefined {
		t.Fatalf("стадия без гипотез: %s", got)
	}
	if got := analysis.InvestigationNextStep(f); got == "" {
		t.Fatal("нет «что дальше» при изделиях под подозрением")
	}
	f.Hypotheses = 1
	if got := analysis.InvestigationStage(f); got != analysis.StageHypothesis {
		t.Fatalf("стадия с гипотезой: %s", got)
	}
	f.Causes[analysis.BranchWhyMade] = "confirmed"
	if got := analysis.InvestigationStage(f); got != analysis.StageCauseConfirmed {
		t.Fatalf("стадия с причиной: %s", got)
	}
	code := func(err error) errcodes.Code {
		var r *kernel.Refusal
		if !errors.As(err, &r) {
			t.Fatalf("ожидался отказ, получено %v", err)
		}
		return r.Code
	}
	if c := code(analysis.GuardCloseInvestigation(f)); c != errcodes.IncidentCauseBranchOpen {
		t.Fatalf("без ветки why_missed: %s", c)
	}
	f.Causes[analysis.BranchWhyMissed] = "not_established"
	if c := code(analysis.GuardCloseInvestigation(f)); c != errcodes.IncidentEffectivenessUnchecked {
		t.Fatalf("без мер: %s", c)
	}
	f.Actions = []string{analysis.ActionEffective, analysis.ActionImplemented}
	if got := analysis.InvestigationStage(f); got != analysis.StageEffectivenessCheck {
		t.Fatalf("стадия с внедрённой мерой: %s", got)
	}
	if c := code(analysis.GuardCloseInvestigation(f)); c != errcodes.IncidentEffectivenessUnchecked {
		t.Fatalf("мера не оценена: %s", c)
	}
	f.Actions = []string{analysis.ActionEffective, analysis.ActionEffective}
	if err := analysis.GuardCloseInvestigation(f); err != nil {
		t.Fatalf("всё отвечено и эффективно, а отказ: %v", err)
	}
	f.Closed = true
	if got := analysis.InvestigationStage(f); got != analysis.StageClosed || analysis.InvestigationNextStep(f) != "" {
		t.Fatalf("закрытое: стадия %s, дальше %q", got, analysis.InvestigationNextStep(f))
	}
}
