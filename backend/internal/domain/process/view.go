package process

import (
	"strconv"
	"time"

	"ant/internal/contracts/constants"
	"ant/internal/domain/kernel"
)

// Представления состояния исполнителя для поздних модулей композиции
// (notifications — сроки, AD-4; quality и nonconformity — точки предъявления)
// и для проекций карты (положение изделия по step_key, AD-17).

// Виды сроков (obligation.due.set.kind).
const (
	DeadlineTimer        = "bpmn_timer"
	DeadlinePresentation = "presentation_wait"
)

// Deadline — срок, который notifications превращает в obligation.due.set
// (AD-4: сроки эмитит только notifications, due_at — по производственному
// календарю). DueAt — срок окна BPMN (абсолютный); у ожидания на точке
// предъявления — From + Wait или WorkDays рабочих дней (считает notifications).
type Deadline struct {
	ObligationID string        `json:"obligation_id"`
	Kind         string        `json:"kind"`
	StepKey      string        `json:"step_key"`
	From         time.Time     `json:"from"`
	DueAt        *time.Time    `json:"due_at,omitempty"`
	Wait         time.Duration `json:"wait,omitempty"`
	WorkDays     int           `json:"work_days,omitempty"`
	OwnerRole    string        `json:"owner_role,omitempty"`
}

// Deadlines — действующие сроки изделия: взведённые таймеры-окна и ожидание
// на точках предъявления (FR-19: просрочка → эскалация). Окно, истекшее без
// новой записи, срабатывает по obligation.due.reached с тем же ObligationID.
func (s State) Deadlines(env Env) []Deadline {
	var out []Deadline
	for _, tm := range s.Timers {
		due := tm.DueAt
		out = append(out, Deadline{ObligationID: tm.ObligationID, Kind: DeadlineTimer, StepKey: tm.StepKey, From: tm.ArmedAt, DueAt: &due})
	}
	for _, t := range s.Tokens {
		n := env.Def.Node(t.Node)
		if n == nil || n.Presentation == nil {
			continue
		}
		p := n.Presentation
		d := Deadline{Kind: DeadlinePresentation, StepKey: n.StepKey(), From: t.Since, OwnerRole: p.Role,
			ObligationID: kernel.UUIDv5(constants.NsAnt, "process.presentation\x1f"+s.ItemID+"\x1f"+n.ID+"\x1f"+strconv.Itoa(s.Gates[n.StepKey()].Count))}
		if p.WaitLimitMinutes != nil {
			d.Wait = time.Duration(*p.WaitLimitMinutes) * time.Minute
		}
		if p.WaitLimitWorkDays != nil {
			d.WorkDays = int(*p.WaitLimitWorkDays)
		}
		if d.Wait == 0 && d.WorkDays == 0 {
			continue
		}
		out = append(out, d)
	}
	return out
}

// TokenView — положение токена изделия для карты (PRD §3b).
type TokenView struct {
	StepKey  string    `json:"step_key"`
	Node     string    `json:"node"`
	Position string    `json:"position"`
	Phase    string    `json:"phase"`
	Since    time.Time `json:"since"`
	Workshop string    `json:"workshop,omitempty"`
	RunID    string    `json:"operation_run_id,omitempty"`
	// Join — токен ждёт другие ветки на слиянии (на карте не считается).
	Join bool `json:"join,omitempty"`
}

// Positions — положение каждого токена изделия.
func (s State) Positions(env Env) []TokenView {
	var out []TokenView
	for _, t := range s.Tokens {
		n := env.Def.Node(t.Node)
		if n == nil {
			continue
		}
		v := TokenView{StepKey: t.StepKey, Node: t.Node, Phase: t.Phase, Since: t.Since, RunID: t.RunID, Workshop: env.Def.Workshop(n)}
		v.Join = n.IsGateway() && len(n.In) > 1 && t.Phase == PhaseWaiting
		v.Position = s.position(t, n)
		out = append(out, v)
	}
	return out
}

func (s State) position(t Token, n *Node) string {
	switch {
	case s.Isolated && len(t.Stack) > 0:
		return PosIsolated
	case t.Phase == PhaseInTransit:
		return PosInTransit
	case n.IsPresentationPoint():
		return PosAtGate
	case n.Props.StepKind == stepKindStorage:
		return PosInStorage
	case n.Props.StepKind == stepKindAutomated || n.Props.StepKind == stepKindHuman:
		return PosAtInspection
	case t.Phase == PhaseInProgress || t.Phase == PhasePaused:
		return PosInProgress
	}
	return PosInQueue
}

// Primary — главное положение изделия на карте: токен не на слиянии с
// наименьшим id; изделие завершено — последний шаг и «завершено».
func (s State) Primary(env Env) (TokenView, bool) {
	if s.Completed {
		return TokenView{StepKey: s.EndStep, Position: PosCompleted}, true
	}
	ps := s.Positions(env)
	for _, p := range ps {
		if !p.Join {
			return p, true
		}
	}
	if len(ps) > 0 {
		return ps[0], true
	}
	return TokenView{}, false
}
