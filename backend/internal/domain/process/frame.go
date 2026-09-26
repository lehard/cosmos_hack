package process

import (
	"strings"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/item"
	"ant/internal/domain/kernel"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "process"

// Правила реакций модуля (слот реакции, AD-3).
const (
	RulePrecondition = "process.precondition"
	RuleMessage      = "process.message"
	RuleInterval     = "process.interval"
)

// Env — закреплённая при запуске изделия часть нормативного слоя, нужная
// модулю process (AD-17), и срез справочников на occurred_at (AD-31).
// Собирает BundleSource (application/process) по хешу версии из
// item.item.registered.
type Env struct {
	// VersionID, VersionHash, Label — закреплённая версия процесса.
	VersionID   string
	VersionHash string
	Label       string
	// Def — описание процесса версии; nil — версия не найдена.
	Def *Definition
	// Refusal — версию исполнять нельзя (FR-23: изменённое содержимое,
	// неполный кворум, версия не найдена): токены не двигаются, гарды
	// отказывают этим кодом.
	Refusal *kernel.Refusal
	// Plan — план контроля типа изделия: значения переменных plan.* (нет —
	// plan.radiography_required = true: ограничивать безопаснее, AD-27).
	Plan map[string]bool
	// Reference — срез справочников для предусловий (FR-17).
	Reference Reference
}

func (e Env) planRadiography() bool {
	if v, ok := e.Plan[VarPlanRadiography]; ok {
		return v
	}
	return true
}

// Upstream — состояния модулей раньше process в композиции на этом шаге
// (только чтение, AD-40): поздний модуль видит вывод раннего, обратно — только
// через функцию-намерение раннего модуля.
type Upstream struct {
	Item *item.State
}

// Reduce применяет запись входа изделия (факт, решение, адресованную запись
// стадии) к исполнителю (AD-5). Реакции в свёртку не входят (AD-3). Перед
// записью срабатывают окна, истёкшие строго раньше её occurred_at (таймер —
// «окно истекло», AD-17): время приходит только из записей (AD-4).
func Reduce(s State, r kernel.Record, env Env, up Upstream) State {
	_ = up
	s = s.clone()
	s.env = env
	if r.ItemID != "" && s.ItemID == "" {
		s.ItemID = r.ItemID
	}
	if s.RunID == "" {
		s.RunID = r.RunID
	}
	if r.Type == catalog.ItemItemRegistered {
		if d, ok := decode[registeredData](r); ok {
			s.VersionHash = d.Hash
		}
		s.VersionID = env.VersionID
	}
	if env.Refusal != nil {
		s.Refused, s.RefusedWhy = string(env.Refusal.Code), env.Refusal.Error()
		return s
	}
	if env.Def == nil {
		if r.Type == catalog.ItemItemRegistered {
			s.Refused, s.RefusedWhy = string(errcodes.ProcessVersionTampered), "версия процесса изделия не найдена"
		}
		return s
	}
	m := &machine{s: &s, env: env, d: env.Def, cause: causeOf(r)}
	if !s.Completed {
		m.fireDue(r.OccurredAt)
	}
	m.at = r.OccurredAt
	if m.at.Before(s.Now) {
		// Позднее по occurred_at не бывает (вход упорядочен, AD-5); защита.
		m.at = s.Now
	}
	if !s.Completed {
		m.apply(r)
	} else {
		m.noteLate(r)
	}
	if r.OccurredAt.After(s.Now) {
		s.Now = r.OccurredAt
	}
	return s
}

// noteLate — факт после завершения изделия: не двигает токены (FR-44).
func (m *machine) noteLate(r kernel.Record) {
	switch r.Type {
	case catalog.OperationRunStarted, catalog.DecisionPresentationResolved, catalog.OperationMovementSent:
		m.refuse("unrouted", "", errcodes.ProcessPreconditionFailed, "изделие уже завершено (%s)", m.s.Outcome)
	case catalog.ItemInterventionOpened:
		m.apply(r)
	}
}

// Apply применяет намерение, адресованное модулю process (Intent.Target ==
// Module), в конце шага свёртки (AD-40). Неизвестное намерение — без изменений.
// Нормативный слой изделия — тот, с которым свёрнута последняя запись
// (Reduce запоминает Env в состоянии вне JSON и хеша).
func Apply(s State, in kernel.Intent) State {
	return ApplyWith(s, in, s.env)
}

// ApplyWith — Apply с нормативным слоем изделия: продвижение точки
// предъявления требует описания процесса.
func ApplyWith(s State, in kernel.Intent, env Env) State {
	switch in.Name {
	case IntentIsolate:
		s = s.clone()
		s.Isolated = true
		return s
	case IntentAdvancePresentation:
		p, ok := in.Payload.(Presentation)
		if !ok || env.Def == nil || env.Refusal != nil || s.Completed {
			return s
		}
		if p.DecisionEventID == "" && len(in.Causes) > 0 {
			p.DecisionEventID = in.Causes[0]
		}
		s = s.clone()
		m := &machine{s: &s, env: env, d: env.Def, at: s.Now, cause: Cause{EventID: p.DecisionEventID, OccurredAt: s.Now}}
		m.advanceGate(p)
		return s
	}
	return s
}

// React вычисляет реакции модуля по состоянию после записи (AD-3):
// нарушения предусловий (operation.precondition.failed), сообщения «в 1С»
// (operation.message.thrown) и интервалы выполнений операций
// (operation.run.interval_resolved — их ждёт machinelogs, AD-42). Сроки и
// эскалации (таймеры, ожидание на точке предъявления) — не реакции process:
// их эмитит notifications по State.Deadlines (AD-4).
func React(s State, env Env, up Upstream) kernel.Output {
	_, _ = env, up
	var out kernel.Output
	subject := "item:" + s.ItemID
	add := func(t catalog.Type, slot kernel.Slot, data any, causes []Cause) {
		recs := make([]kernel.Record, 0, len(causes))
		for _, c := range causes {
			recs = append(recs, c.Record())
		}
		re, err := kernel.NewReaction(Module, t, slot, data, recs...)
		if err != nil {
			panic(err) // тип и эмитент постоянны: ошибка — рассинхронизация с каталогом
		}
		out.Reactions = append(out.Reactions, re)
	}
	for _, b := range s.Breaches {
		d := ev.OperationPreconditionFailedV1{StepKey: ev.StepKey(b.StepKey), Precondition: ev.OperationPreconditionFailedV1Precondition(b.Precondition),
			Mode: ev.OperationPreconditionFailedV1Mode(b.Mode)}
		if b.RunID != "" {
			id := ev.ObjectID(b.RunID)
			d.OperationRunID = &id
		}
		if b.Subject != "" {
			id := ev.ObjectID(b.Subject)
			d.SubjectRef = &id
		}
		if b.Detail != "" {
			det := b.Detail
			d.Detail = &det
		}
		add(catalog.OperationPreconditionFailed, kernel.Slot{RuleID: RulePrecondition, Subject: subject, TriggerKey: b.Key}, d, b.Causes)
	}
	for _, t := range s.Thrown {
		basis := make([]ev.UUID, 0, len(t.Basis))
		for _, b := range t.Basis {
			basis = append(basis, ev.UUID(b))
		}
		d := ev.OperationMessageThrownV1{StepKey: ev.StepKey(t.StepKey), MessageRef: t.MessageRef,
			ErpAction: ev.OperationMessageThrownV1ErpAction(t.ErpAction), ClosingBasis: basis}
		add(catalog.OperationMessageThrown, kernel.Slot{RuleID: RuleMessage, Subject: subject, TriggerKey: t.Node + "#" + itoa(t.Visit)}, d, t.Causes)
	}
	for _, k := range sortedKeys(s.Runs) {
		r := s.Runs[k]
		add(catalog.OperationRunIntervalResolved, kernel.Slot{RuleID: RuleInterval, Subject: subject, TriggerKey: r.RunID}, intervalData(r), r.Causes)
	}
	return out
}

// intervalData — данные operation.run.interval_resolved (время — RFC 3339 с
// миллисекундами, «Соглашения/Время»). Границы, переданные источником, —
// source_reported; взятые из времени записей или входа на шаг — system_computed.
type intervalJSON struct {
	OperationRunID string `json:"operation_run_id"`
	EquipmentID    string `json:"equipment_id,omitempty"`
	IntervalStart  string `json:"interval_start"`
	IntervalEnd    string `json:"interval_end,omitempty"`
	IntervalOrigin string `json:"interval_origin"`
}

func intervalData(r Run) intervalJSON {
	d := intervalJSON{OperationRunID: r.RunID, EquipmentID: r.Equipment, IntervalStart: ts(r.StartedAt), IntervalOrigin: "system_computed"}
	if r.FinishedAt != nil {
		d.IntervalEnd = ts(*r.FinishedAt)
	}
	if r.SourceStart && (r.FinishedAt == nil || r.SourceEnd) {
		d.IntervalOrigin = "source_reported"
	}
	return d
}

func ts(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}

// Команды process над изделием в представлении домена (kernel.Command.Payload).

// StartCommand — начать операцию (process.operation.start).
type StartCommand struct {
	StepKey     string
	RunID       string
	ReworkOf    string
	OperatorID  string
	EquipmentID string
}

// RunCommand — пауза, продолжение, конец операции (process.operation.pause | resume | finish).
type RunCommand struct {
	RunID string
}

// MoveCommand — отправить или принять изделие (process.movement.send | receive).
type MoveCommand struct {
	StepKey         string
	DestinationKind string
}

// Guard — доменный гард операций модуля process (AD-39): состояние изделия на
// basis_seq и команда → nil или *kernel.Refusal с кодом из contracts/errors.yaml.
// Его вызывают api до записи, свёртка при применении и верификатор.
func Guard(s State, env Env, up Upstream, cmd kernel.Command) error {
	_ = up
	if !strings.HasPrefix(cmd.Action, "process.operation.") && !strings.HasPrefix(cmd.Action, "process.movement.") {
		return nil
	}
	if env.Refusal != nil {
		return env.Refusal
	}
	if env.Def == nil {
		return kernel.Refuse(errcodes.ProcessVersionTampered, "version_id", s.VersionID)
	}
	if s.Completed {
		return kernel.Refuse(errcodes.ProcessPreconditionFailed, "condition", "изделие не завершено ("+s.Outcome+")")
	}
	switch cmd.Action {
	case "process.operation.start":
		c, ok := cmd.Payload.(StartCommand)
		if !ok {
			return nil
		}
		return GuardStart(s, env, c, cmd.OccurredAt)
	case "process.operation.pause", "process.operation.resume", "process.operation.finish":
		c, ok := cmd.Payload.(RunCommand)
		if !ok {
			return nil
		}
		if _, ok := s.Runs[c.RunID]; !ok {
			return kernel.Refuse(errcodes.ProcessPreconditionFailed, "condition", "выполнение "+c.RunID+" начато")
		}
	case "process.movement.send":
		if s.Containment == "block" {
			return kernel.Refuse(errcodes.NonconformityItemBlocked)
		}
	}
	return nil
}

// GuardStart — гард начала операции (FR-17, FR-18, FR-19, FR-20, FR-21, FR-44):
// изделие стоит на шаге (не за точкой предъявления без подписи), предусловия
// с блокировкой выполнены, лимит доработок не исчерпан.
func GuardStart(s State, env Env, c StartCommand, at time.Time) error {
	d := env.Def
	n := d.ByStep(c.StepKey)
	if n == nil {
		return kernel.Refuse(errcodes.ProcessPreconditionFailed, "condition", "шаг "+c.StepKey+" есть в версии процесса изделия")
	}
	t := s.TokenAt(c.StepKey)
	if t == nil {
		m := &machine{s: &s, env: env, d: d}
		if g := m.gateBefore(n); g != "" {
			gn := d.ByStep(g)
			who := ""
			if gn != nil && gn.Presentation != nil {
				gs := s.Gates[g]
				who = requiredAuthority(gn, max(gs.Count, 1))
			}
			return kernel.Refuse(errcodes.NonconformityGateWithoutSignature, "role", who)
		}
		return kernel.Refuse(errcodes.ProcessPreconditionFailed, "condition", "изделие на шаге "+c.StepKey+" (сейчас: "+strings.Join(s.Steps(), ", ")+")")
	}
	if t.Phase == PhaseBlocked {
		return kernel.Refuse(errcodes.ProcessPreconditionFailed, "condition", "блок снят ("+t.Block+")")
	}
	zones := s.zonesFor(n)
	if s.runsAt(n.ID, "") == 0 {
		zones = n.Zones
	}
	for _, f := range s.checks(env, n, c.OperatorID, c.EquipmentID, zones, at, true) {
		if f.Mode == preconditionBlock {
			r := kernel.Refuse(f.Code, f.Params...)
			r.Detail = f.Detail
			return r
		}
	}
	return nil
}

// PresentationGuard — может ли решение resolution быть принято на точке
// предъявления stepKey (для гарда decision.presentation.resolved модуля
// nonconformity, FR-19, FR-21, FR-44): изделие стоит на точке, приёмка — не
// при открытом вмешательстве. Требуемое полномочие — State.Gates[stepKey].Authority.
func PresentationGuard(s State, env Env, stepKey, resolution string) error {
	if env.Refusal != nil {
		return env.Refusal
	}
	n := env.Def.ByStep(stepKey)
	if n == nil || !n.IsPresentationPoint() {
		return kernel.Refuse(errcodes.ProcessPreconditionFailed, "condition", "шаг "+stepKey+" — точка предъявления")
	}
	if s.TokenAt(stepKey) == nil {
		return kernel.Refuse(errcodes.ProcessPreconditionFailed, "condition", "изделие на точке "+stepKey)
	}
	if (resolution == ResolutionAccept || resolution == ResolutionAcceptWithConcession) && len(s.OpenInterventions()) > 0 {
		return kernel.Refuse(errcodes.NonconformityInterventionOpen)
	}
	return nil
}
