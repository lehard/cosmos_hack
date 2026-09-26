package process

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

func wantSteps(t *testing.T, s State, want ...string) {
	t.Helper()
	got := steps(s)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("изделие на шагах %v, ожидалось %v (токены %+v, отказы %+v)", got, want, s.Tokens, s.Refusals)
	}
}

func refusalCode(err error) errcodes.Code {
	var r *kernel.Refusal
	if errors.As(err, &r) {
		return r.Code
	}
	return ""
}

// weldOnce — от запуска до ЗТ-3: подготовка кромок, сварка в окне, камера.
func weldOnce(w *world) {
	w.register(0)
	w.start("P-1", "welding.edge_prep", 10)
	w.finish("P-1", 20)
	w.start("W-1", "welding.weld", 30)
	w.finish("W-1", 40)
	w.inspect("welding.kt3", 50, "defect_indicated", "U2")
}

func TestStartByMessageAndHappyPath(t *testing.T) {
	w := newWorld(t)
	weldOnce(w)
	s, _ := w.fold()
	wantSteps(t, s, "welding.zt3")
	w.decide("welding.zt3", ResolutionAccept, 60)
	w.add(catalog.InspectionResultRecorded, 65, map[string]any{"method": "visual_human", "phase": "before_zone_closure", "outcome": "no_defect_indicated",
		"processing_state": "completed", "zone_ids": []string{"Z9"}})
	w.start("A-1", "assembly.cover", 70)
	w.finish("A-1", 80)
	w.decide("final.zt6", ResolutionAccept, 90)
	s, rs := w.fold()
	if !s.Completed || s.Outcome != "released" || s.EndStep != "final.released" {
		t.Fatalf("изделие не выпущено: %+v", s)
	}
	thrown := reactionsOf(rs, catalog.OperationMessageThrown)
	if len(thrown) != 1 {
		t.Fatalf("сообщений «в 1С»: %d", len(thrown))
	}
	b, _ := json.Marshal(thrown[0].Data)
	if !strings.Contains(string(b), `"erp_action":"release"`) || !strings.Contains(string(b), w.in[len(w.in)-1].EventID) {
		t.Fatalf("сообщение без учётного действия или основания закрывающей точки: %s", b)
	}
	if n := len(reactionsOf(rs, catalog.OperationRunIntervalResolved)); n != 3 {
		t.Fatalf("интервалов выполнений: %d, ожидалось 3", n)
	}
	if n := len(reactionsOf(rs, catalog.OperationPreconditionFailed)); n != 0 {
		t.Fatalf("лишние нарушения предусловий: %+v", s.Breaches)
	}
}

// FR-11: истечение окна «кромки → сварка» уводит изделие на повторную подготовку.
func TestEdgeWindowExpiredSendsBackToPreparation(t *testing.T) {
	w := newWorld(t)
	w.register(0)
	w.start("P-1", "welding.edge_prep", 10)
	w.finish("P-1", 20) // окно 8 ч: до 10:00 от t0+2 ч
	s, _ := w.fold()
	wantSteps(t, s, "welding.weld")
	if len(s.Timers) != 1 || !s.Timers[0].DueAt.Equal(at(100)) {
		t.Fatalf("окно не взведено: %+v", s.Timers)
	}
	if d := s.Deadlines(w.env); len(d) != 1 || d[0].Kind != DeadlineTimer || d[0].ObligationID != s.Timers[0].ObligationID {
		t.Fatalf("срок окна для notifications: %+v", d)
	}
	// Сварщик начал через 9 ч после кромок: окно истекло в t0+10 ч.
	w.start("W-1", "welding.weld", 110)
	s, rs := w.fold()
	wantSteps(t, s, "welding.edge_prep")
	if s.Visits["P"] != 2 {
		t.Fatalf("подготовка кромок не повторена: посещений %d", s.Visits["P"])
	}
	br := reactionsOf(rs, catalog.OperationPreconditionFailed)
	if len(br) != 1 || !strings.Contains(string(mustJSON(t, br[0].Data)), `"precondition":"time_window"`) {
		t.Fatalf("нет нарушения окна: %+v", s.Breaches)
	}
	var left *Visit
	for i := range s.History {
		if s.History[i].Node == "W" {
			left = &s.History[i]
		}
	}
	if left == nil || left.Via != "timer" || !left.Left.Equal(at(100)) {
		t.Fatalf("сварка покинута не по таймеру в срок окна: %+v", left)
	}
	// Повторная подготовка и сварка в окне — дальше по маршруту.
	w.start("P-2", "welding.edge_prep", 120)
	w.finish("P-2", 130)
	w.start("W-2", "welding.weld", 140)
	s, _ = w.fold()
	wantSteps(t, s, "welding.weld")
	if len(s.Timers) != 0 {
		t.Fatalf("окно until_started не снято началом операции: %+v", s.Timers)
	}
	if r := s.Runs["P-2"]; r.ReworkOf != "P-1" || !r.Inferred || r.N != 2 {
		t.Fatalf("повтор не связан с предыдущим выполнением (FR-47): %+v", r)
	}
}

// Окно срабатывает и по obligation.due.reached без других записей.
func TestTimerFiresByDueReached(t *testing.T) {
	w := newWorld(t)
	w.register(0)
	w.start("P-1", "welding.edge_prep", 10)
	w.finish("P-1", 20)
	s, _ := w.fold()
	w.add(catalog.ObligationDueReached, 100, map[string]any{"obligation_id": s.Timers[0].ObligationID, "due_at": "2026-09-22T16:00:00.000Z"})
	s, _ = w.fold()
	wantSteps(t, s, "welding.edge_prep")
}

// FR-11, AD-17: подпроцесс брака вызывается из двух мест и возвращает в точку вызова.
func TestNonconformitySubprocessReturnsToCallPoint(t *testing.T) {
	w := newWorld(t)
	weldOnce(w)
	w.decide("welding.zt3", ResolutionReject, 60)
	s, _ := w.fold()
	wantSteps(t, s, "nc.isolation")
	if tk := s.TokenAt("nc.isolation"); len(tk.Stack) != 1 || tk.Stack[0].Call != "NC1" {
		t.Fatalf("нет кадра вызова NC1: %+v", tk)
	}
	w.add(catalog.DecisionItemIsolated, 62, map[string]any{"reason": map[string]string{"code": "nc", "text": "брак"}})
	w.add(catalog.OperationMovementReceived, 63, map[string]any{"to_location_id": "ISO-1", "destination_kind": "isolator", "inspection_on_receipt": "no_damage", "received_by": "M-1"})
	s, _ = w.fold()
	wantSteps(t, s, "nc.disposition")
	if p, _ := s.Primary(w.env); p.Position != PosIsolated {
		t.Fatalf("положение в подпроцессе брака: %s", p.Position)
	}
	w.dispose("rework", 70)
	s, _ = w.fold()
	// Возврат в точку вызова NC1 → итог rework_or_repair → повторная подготовка кромок.
	wantSteps(t, s, "welding.edge_prep")
	if tk := s.TokenAt("welding.edge_prep"); len(tk.Stack) != 0 || tk.Vars[VarNCOutcome].S != "rework_or_repair" {
		t.Fatalf("возврат без итога подпроцесса: %+v", tk)
	}
	// Второе место вызова — после ЗТ-6: возврат на установку крышки.
	w.start("P-2", "welding.edge_prep", 80)
	w.finish("P-2", 90)
	w.start("W-2", "welding.weld", 100)
	w.finish("W-2", 110)
	w.inspect("welding.kt3", 120, "no_defect_indicated", "U1", "U2")
	w.decide("welding.zt3", ResolutionAccept, 130)
	w.inspect("assembly.cover", 135, "no_defect_indicated", "Z9")
	s, _ = w.fold()
	wantSteps(t, s, "assembly.cover")
	w.start("A-1", "assembly.cover", 140)
	w.finish("A-1", 150)
	w.decide("final.zt6", ResolutionReject, 160)
	w.dispose("rework", 170) // изоляции нет в данных — догон до ЗТ-Р с пометкой «нет данных»
	s, _ = w.fold()
	wantSteps(t, s, "assembly.cover")
	if !slices.ContainsFunc(s.Gaps, func(g Gap) bool { return g.StepKey == "nc.isolation" }) {
		t.Fatalf("пропуск изоляции не помечен «нет данных»: %+v", s.Gaps)
	}
	// Списание во втором вызове — terminate завершает все токены (Д-4).
	w.start("A-2", "assembly.cover", 180)
	w.finish("A-2", 190)
	w.decide("final.zt6", ResolutionReject, 200)
	w.dispose("scrap", 210)
	s, _ = w.fold()
	if !s.Completed || s.Outcome != "terminated" || s.EndStep != "final.scrapped" || len(s.Tokens) != 0 {
		t.Fatalf("списание не завершило изделие: %+v", s)
	}
}

// FR-18: четвёртая доработка зоны при лимите 3 без разрешения блокируется.
func TestFourthReworkOfZoneIsBlocked(t *testing.T) {
	w := newWorld(t)
	weldOnce(w) // первая сварка; дефект в зоне U2
	h := 60
	for i := 1; i <= 3; i++ {
		w.decide("welding.zt3", ResolutionReject, h)
		w.dispose("repair", h+5)
		w.start(runID("P", i+1), "welding.edge_prep", h+10)
		w.finish(runID("P", i+1), h+15)
		w.start(runID("W", i+1), "welding.weld", h+20)
		w.finish(runID("W", i+1), h+25)
		w.inspect("welding.kt3", h+30, "defect_indicated", "U2")
		h += 40
	}
	// Три доработки зоны U2 выполнены; четвёртая — гард отказывает.
	w.decide("welding.zt3", ResolutionReject, h)
	w.dispose("repair", h+5)
	w.start(runID("P", 5), "welding.edge_prep", h+10)
	s, _ := w.fold()
	// Подготовка кромок — тоже петля с лимитом 3: пятое выполнение = 4-я доработка.
	if tk := s.TokenAt("welding.edge_prep"); tk == nil || tk.Phase != PhaseBlocked || tk.Block != "rework_limit" {
		t.Fatalf("4-я доработка петли кромок не заблокирована: %+v", tk)
	}
	w.add(catalog.DecisionReworkLimitWaived, h+12, map[string]any{"zone_id": "welding.edge_prep", "used": 4, "limit": 3, "extra_allowed": 1,
		"reason": map[string]string{"code": "waiver", "text": "разрешение"}})
	w.finish(runID("P", 5), h+15)
	s, _ = w.fold()
	wantSteps(t, s, "welding.weld")
	err := Guard(s, w.env, Upstream{}, kernel.Command{Action: "process.operation.start", OccurredAt: at(h + 20),
		Payload: StartCommand{StepKey: "welding.weld", RunID: "W-5", OperatorID: "WLD-01"}})
	if refusalCode(err) != errcodes.ProcessReworkLimitExceeded {
		t.Fatalf("гард пропустил 4-ю доработку зоны: %v", err)
	}
	var r *kernel.Refusal
	errors.As(err, &r)
	if r.Params["zone"] != "U2" || r.Params["used"] != "4" || r.Params["limit"] != "3" {
		t.Fatalf("параметры отказа: %+v", r.Params)
	}
	// Внешний факт той же 4-й доработки принимается с реакцией-блоком (AD-30).
	w.start("W-5", "welding.weld", h+20)
	s, rs := w.fold()
	if tk := s.TokenAt("welding.weld"); tk.Phase != PhaseBlocked {
		t.Fatalf("сварка сверх лимита не заблокирована: %+v", tk)
	}
	found := false
	for _, re := range reactionsOf(rs, catalog.OperationPreconditionFailed) {
		b := string(mustJSON(t, re.Data))
		if strings.Contains(b, `"precondition":"rework_limit"`) && strings.Contains(b, `"subject_ref":"U2"`) && strings.Contains(b, `"mode":"block"`) {
			found = true
		}
	}
	if !found {
		t.Fatalf("нет реакции-блока rework_limit по зоне U2: %+v", s.Breaches)
	}
	// Разрешение уполномоченного снимает блок.
	w.add(catalog.DecisionReworkLimitWaived, h+22, map[string]any{"zone_id": "U2", "used": 4, "limit": 3, "extra_allowed": 1,
		"reason": map[string]string{"code": "waiver", "text": "разрешение"}})
	w.finish("W-5", h+30)
	s, _ = w.fold()
	wantSteps(t, s, "welding.kt3")
}

func runID(p string, i int) string { return p + "-" + itoa(i) }

// FR-19, FR-44: без подписи изделие не проходит точку предъявления.
func TestPresentationPointRequiresSignature(t *testing.T) {
	w := newWorld(t)
	weldOnce(w)
	// Решение без подписи человека (запись сервера) — не пропускает.
	r := w.decide("welding.zt3", ResolutionAccept, 60)
	w.in[len(w.in)-1].Provenance = "server_attested"
	// Факты следующего шага не двигают изделие через точку.
	w.start("A-1", "assembly.cover", 70)
	w.finish("A-1", 80)
	s, _ := w.fold()
	wantSteps(t, s, "welding.zt3")
	if !slices.ContainsFunc(s.Refusals, func(x Refusal) bool {
		return x.EventID == r.EventID && x.Code == string(errcodes.NonconformityGateWithoutSignature)
	}) {
		t.Fatalf("отказ по решению без подписи не записан: %+v", s.Refusals)
	}
	err := Guard(s, w.env, Upstream{}, kernel.Command{Action: "process.operation.start", OccurredAt: at(70),
		Payload: StartCommand{StepKey: "assembly.cover", RunID: "A-2"}})
	if refusalCode(err) != errcodes.NonconformityGateWithoutSignature {
		t.Fatalf("гард пропустил операцию за точкой предъявления: %v", err)
	}
	// Намерение quality без подписанного решения — тоже не двигает.
	s2 := Apply(s, AdvancePresentation("quality", Presentation{StepKey: "welding.zt3", Resolution: ResolutionAccept}, r))
	wantSteps(t, s2, "welding.zt3")
	// Подписанное решение — проходит; повтор тем же решением (намерение
	// quality после продвижения самим process) ничего не меняет.
	signed := w.decide("welding.zt3", ResolutionAccept, 90)
	s, _ = w.fold()
	wantSteps(t, s, "assembly.cover")
	s3 := Apply(s, AdvancePresentation("quality", Presentation{StepKey: "welding.zt3", Resolution: ResolutionAccept, DecisionEventID: signed.EventID}))
	wantSteps(t, s3, "assembly.cover")
	if g := s.Gates["welding.zt3"]; g.Count != 1 || g.Authority != "qc_acceptance" || g.Last != ResolutionAccept {
		t.Fatalf("точка предъявления: %+v", g)
	}
}

// FR-19: счёт предъявлений и полномочие повторного предъявления.
func TestRepeatedPresentationRequiresHigherAuthority(t *testing.T) {
	w := newWorld(t)
	weldOnce(w)
	w.decide("welding.zt3", ResolutionInsufficientData, 60)
	w.inspect("welding.kt3", 70, "no_defect_indicated", "U1", "U2")
	s, _ := w.fold()
	wantSteps(t, s, "welding.zt3")
	if g := s.Gates["welding.zt3"]; g.Count != 2 || g.Authority != "qc_acceptance_repeat" {
		t.Fatalf("повторное предъявление: %+v", g)
	}
	if tk := s.TokenAt("welding.zt3"); tk.Vars[VarPresentationNo].I != 2 {
		t.Fatalf("presentation.no = %+v", tk.Vars[VarPresentationNo])
	}
}

// Намерение quality с подписанным решением продвигает точку (AD-40).
func TestAdvancePresentationIntent(t *testing.T) {
	w := newWorld(t)
	weldOnce(w)
	s, _ := w.fold()
	dec := w.decide("welding.zt3", ResolutionAccept, 60)
	// Запись решения видна process (подпись), продвигает намерение.
	s = Reduce(s, dec, w.env, Upstream{})
	wantSteps(t, s, "assembly.cover") // process продвигает и сам — идемпотентно
	s = Apply(s, AdvancePresentation("quality", Presentation{StepKey: "welding.zt3", Resolution: ResolutionAccept}, dec))
	wantSteps(t, s, "assembly.cover")
}

// FR-20: операция, закрывающая доступ к зоне, требует завершённой проверки зоны.
func TestHiddenWorkRequiresZoneCheck(t *testing.T) {
	w := newWorld(t)
	weldOnce(w)
	w.decide("welding.zt3", ResolutionAccept, 60)
	s, _ := w.fold()
	err := Guard(s, w.env, Upstream{}, kernel.Command{Action: "process.operation.start", OccurredAt: at(70), Payload: StartCommand{StepKey: "assembly.cover"}})
	var r *kernel.Refusal
	if !errors.As(err, &r) || r.Code != errcodes.ProcessZoneCheckRequired || r.Params["zone"] != "Z9" {
		t.Fatalf("гард без проверки зоны: %v", err)
	}
	w.start("A-1", "assembly.cover", 70)
	s, rs := w.fold()
	if len(reactionsOf(rs, catalog.OperationPreconditionFailed)) != 1 || s.TokenAt("assembly.cover").Phase != PhaseBlocked {
		t.Fatalf("факт скрытой работы без проверки зоны: %+v", s.Breaches)
	}
}

// FR-21: при открытом вмешательстве приёмка запрещена; закрытие с перепроверкой — разрешает.
func TestInterventionBlocksAcceptance(t *testing.T) {
	w := newWorld(t)
	weldOnce(w)
	w.decide("welding.zt3", ResolutionAccept, 60)
	w.inspect("assembly.cover", 65, "no_defect_indicated", "Z9")
	w.start("A-1", "assembly.cover", 70)
	w.finish("A-1", 80)
	w.add(catalog.ItemInterventionOpened, 85, map[string]any{"intervention_id": "IV-1", "zone_ids": []string{"Z9"}, "purpose": map[string]string{"code": "fod", "text": "поиск"}})
	w.decide("final.zt6", ResolutionAccept, 90)
	s, _ := w.fold()
	wantSteps(t, s, "final.zt6")
	if s.Zones["Z9"].Outdated != true {
		t.Fatalf("результаты зоны не помечены «устарели»: %+v", s.Zones["Z9"])
	}
	if refusalCode(PresentationGuard(s, w.env, "final.zt6", ResolutionAccept)) != errcodes.NonconformityInterventionOpen {
		t.Fatal("гард приёмки при открытом вмешательстве")
	}
	w.add(catalog.ItemInterventionClosed, 95, map[string]any{"intervention_id": "IV-1", "recheck_event_ids": []string{}})
	w.decide("final.zt6", ResolutionAccept, 100)
	s, _ = w.fold()
	if !s.Completed {
		t.Fatalf("после закрытия вмешательства приёмка не прошла: %+v", s.Tokens)
	}
}

// FR-23, AD-17: изменённый XML и неполный кворум не исполняются.
func TestModifiedVersionIsNotExecuted(t *testing.T) {
	orig := []byte(miniBPMN)
	pinned := VersionHash(orig)
	genesis := Approval{VersionID: "mini-1", Hash: pinned, Genesis: true}
	if r := CheckExecutable("mini-1", pinned, orig, genesis); r != nil {
		t.Fatalf("стартовая версия генезиса отклонена: %v", r)
	}
	tampered := []byte(strings.Replace(miniBPMN, `reworkLimit="3" reworkLimitScope="zone"`, `reworkLimit="30" reworkLimitScope="zone"`, 1))
	r := CheckExecutable("mini-1", pinned, tampered, genesis)
	if r == nil || r.Code != errcodes.ProcessVersionTampered {
		t.Fatalf("изменённый XML не обнаружен: %v", r)
	}
	unsigned := Approval{VersionID: "mini-2", Hash: pinned, Signatures: []Signature{{Role: "technologist", Authority: QuorumAuthority, Valid: true}}}
	r = CheckExecutable("mini-2", pinned, orig, unsigned)
	if r == nil || r.Code != errcodes.ProcessQuorumIncomplete || !strings.Contains(r.Params["who"], "production_manager") {
		t.Fatalf("неполный кворум: %v", r)
	}
	// Отказ версии: токены не двигаются, гарды отказывают тем же кодом.
	w := newWorld(t)
	w.env.Refusal = CheckExecutable("mini-1", pinned, tampered, genesis)
	weldOnce(w)
	s, rs := w.fold()
	if len(s.Tokens) != 0 || s.Refused != string(errcodes.ProcessVersionTampered) || len(rs) != 0 {
		t.Fatalf("изменённая версия исполнена: %+v", s)
	}
	if refusalCode(Guard(s, w.env, Upstream{}, kernel.Command{Action: "process.operation.start", Payload: StartCommand{StepKey: "welding.edge_prep"}})) != errcodes.ProcessVersionTampered {
		t.Fatal("гард принял команду по изменённой версии")
	}
}

// NFR-DET-1: свёртка детерминирована — два прогона дают одно состояние.
func TestFoldDeterministic(t *testing.T) {
	w := newWorld(t)
	weldOnce(w)
	w.decide("welding.zt3", ResolutionReject, 60)
	w.dispose("rework", 70)
	s1, r1 := w.fold()
	s2, r2 := w.fold()
	if string(mustJSON(t, s1)) != string(mustJSON(t, s2)) || string(mustJSON(t, r1)) != string(mustJSON(t, r2)) {
		t.Fatal("свёртка не детерминирована")
	}
}

// Пропущенное событие: конец операции дальнего шага догоняет изделие.
func TestCatchUpMarksGaps(t *testing.T) {
	w := newWorld(t)
	w.register(0)
	w.start("W-1", "welding.weld", 10) // кромки без данных
	s, _ := w.fold()
	wantSteps(t, s, "welding.weld")
	if len(s.Gaps) != 1 || s.Gaps[0].StepKey != "welding.edge_prep" {
		t.Fatalf("пропуск кромок не помечен: %+v", s.Gaps)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Эпик 16: команда начала операции согласована со свёрткой — токен на шаге
// без своих данных (подготовка кромок без факта) догоняется до операции, как
// догнал бы его факт operation.run.started; через точку предъявления — нет.
func TestStartGuardCatchesUpLikeFold(t *testing.T) {
	w := newWorld(t)
	w.register(0)
	s, _ := w.fold()
	wantSteps(t, s, "welding.edge_prep")
	err := Guard(s, w.env, Upstream{}, kernel.Command{Action: "process.operation.start", OccurredAt: at(5),
		Payload: StartCommand{StepKey: "welding.weld", RunID: "W-1", OperatorID: "WLD-01"}})
	if err != nil {
		t.Fatalf("начало сварки после шага без данных: %v", err)
	}
	wantSteps(t, s, "welding.edge_prep") // гард состояние не меняет
	err = Guard(s, w.env, Upstream{}, kernel.Command{Action: "process.operation.start", OccurredAt: at(5),
		Payload: StartCommand{StepKey: "assembly.cover", RunID: "A-1"}})
	if err == nil {
		t.Fatal("операция за точкой предъявления без подписи принята")
	}
}
