package process

import (
	"strconv"
	"strings"
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

// Действия человека на шаге процесса (задачи ролей порождает процесс, а не
// сценарий): операция API, которой человек продвигает изделие, и роль по
// дорожке шага.
const (
	OpMovementReceive = "process.movement.receive"
	OpMovementSend    = "process.movement.send"
	OpOperationStart  = "process.operation.start"
	OpOperationFinish = "process.operation.finish"
)

// Роли исполнителей шагов (normative/policy/policy.v1.yaml).
const (
	RoleSiteForeman = "site_foreman"
	RolePerformer   = "performer"
	RoleStorekeeper = "storekeeper"
)

// movementRole — кто перемещает изделие на шаге дорожки: на складской
// дорожке (цех WS-SK — «Склад и входной контроль») — кладовщик, в цехах —
// мастер. Атрибута роли у дорожки в BPMN пока нет — проектное допущение.
func movementRole(workshop string) string {
	if workshop == "WS-SK" {
		return RoleStorekeeper
	}
	return RoleSiteForeman
}

// HumanStep — изделие стоит на шаге, где процесс ждёт действия человека:
// кто (роль), где (цех дорожки BPMN), что нажать (операция), с какого
// момента и срок по нормативу шага (норма ожидания или норма времени).
type HumanStep struct {
	StepKey   string     `json:"step_key"`
	Node      string     `json:"node"`
	Name      string     `json:"name"`
	Operation string     `json:"operation"`
	Role      string     `json:"role"`
	Workshop  string     `json:"workshop,omitempty"`
	Since     time.Time  `json:"since"`
	DueAt     *time.Time `json:"due_at,omitempty"`
}

// preparedBy — операция, для которой шаг n подготовительный: единственный
// следующий узел — операция с предусловием time_window, отсчитываемым от
// конца n (ref «‹хвост step_key n›<=…»); nil — n самостоятельная операция.
func preparedBy(d *Definition, n *Node) *Node {
	next := d.Next(n)
	if len(next) != 1 {
		return nil
	}
	nx := d.Node(next[0])
	if nx == nil || nx.Props.StepKind != stepKindOperation {
		return nil
	}
	short := n.StepKey()
	if i := strings.LastIndexByte(short, '.'); i >= 0 {
		short = short[i+1:]
	}
	for _, p := range nx.Preconditions {
		if p.Kind == "time_window" && strings.HasPrefix(p.Ref, short+"<=") {
			return nx
		}
	}
	return nil
}

// HumanSteps — шаги с действием человека, на которых сейчас стоят токены
// изделия (одно правило на все шаги процесса):
//   - перемещение: в пути или приёмка цехом (userTask) — «принять» мастером
//     цеха-получателя; ждёт отправки — «отправить» мастером цеха;
//   - операция: ждёт начала — «начать», начата — «выполнено»; исполнитель
//     участка дорожки.
//
// Контроль человеком (точки предъявления) — в очереди контролёра
// (nonconformity), машинный контроль и хранение — без задачи.
func (s State) HumanSteps(env Env) []HumanStep {
	if env.Def == nil || s.Completed {
		return nil
	}
	var out []HumanStep
	for _, t := range s.Tokens {
		n := env.Def.Node(t.Node)
		if n == nil || !n.IsTask() || n.IsHumanControl() || t.Phase == PhaseBlocked || t.Phase == PhaseStuck {
			continue
		}
		h := HumanStep{StepKey: t.StepKey, Node: n.ID, Name: n.Name, Workshop: env.Def.Workshop(n), Since: t.Since}
		var norm *int64
		switch n.Props.StepKind {
		case stepKindMovement:
			// Изолированное изделие перемещает в изолятор задача isolate_move.
			if s.Isolated && len(t.Stack) > 0 {
				continue
			}
			h.Role = movementRole(h.Workshop)
			if t.Phase == PhaseInTransit || n.Type == NodeUserTask {
				h.Operation = OpMovementReceive
			} else {
				h.Operation = OpMovementSend
			}
			if n.Norm != nil {
				norm = n.Norm.QueueNormMinutes
			}
		case stepKindOperation:
			h.Role = RolePerformer
			switch t.Phase {
			case PhaseInProgress, PhasePaused:
				h.Operation = OpOperationFinish
				if n.Norm != nil {
					norm = n.Norm.TimeMinutes
				}
			default:
				h.Operation = OpOperationStart
				if n.Norm != nil {
					norm = n.Norm.QueueNormMinutes
				}
				// Подготовительный шаг: следующая операция держит окно от его
				// конца (ant:precondition time_window «edge_prep<=PT8H» у сварки).
				// Исполнитель начинает сразу следующую операцию — подготовка
				// засчитывается по её началу; задача — «Начать» её.
				if nx := preparedBy(env.Def, n); nx != nil {
					h.StepKey, h.Node, h.Name = nx.StepKey(), nx.ID, nx.Name
					norm = nil
					if nx.Norm != nil {
						norm = nx.Norm.QueueNormMinutes
					}
				}
			}
		default:
			continue
		}
		if norm != nil && *norm > 0 {
			d := t.Since.Add(time.Duration(*norm) * time.Minute)
			h.DueAt = &d
		}
		out = append(out, h)
	}
	return out
}
