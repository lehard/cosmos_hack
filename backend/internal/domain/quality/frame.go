package quality

import (
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/statuses"
	ev "ant/internal/contracts/events"
	"ant/internal/domain/item"
	"ant/internal/domain/kernel"
	"ant/internal/domain/process"
	"ant/internal/domain/vision"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "quality"

// State — состояние модуля quality в свёртке одного изделия (AD-5): входы
// (нормализованные факты и решения) и выводы, пересчитываемые из входов после
// каждой записи (derive). Выводы — единственный источник для реакций,
// проекций и поздних модулей композиции (nonconformity, notifications читают
// их через Upstream.Quality и Requests). Поля экспортируемые и сериализуются в
// JSON целиком — по ним считается хеш состояния (Д-22).
type State struct {
	// ItemID — изделие свёртки.
	ItemID string `json:"item_id,omitempty"`
	// Pos — число свёрнутых записей (порядок входа, AD-5); Current — event_id
	// последней свёрнутой записи.
	Pos     int    `json:"pos"`
	Current string `json:"current,omitempty"`

	// ── входы ──
	Observations  []Observation  `json:"observations,omitempty"`
	Runs          []Run          `json:"runs,omitempty"`
	Skips         []Skip         `json:"skips,omitempty"`
	Reports       []Report       `json:"reports,omitempty"`
	Presentations []Presentation `json:"presentations,omitempty"`
	Reviews       []Review       `json:"reviews,omitempty"`
	Rechecks      []Recheck      `json:"rechecks,omitempty"`
	Overrides     []Override     `json:"overrides,omitempty"`
	// Undecodable — записи, которые не удалось разобрать: факт не теряется молча.
	Undecodable []string `json:"undecodable,omitempty"`

	// ── выводы (derive) ──
	Defects    []Defect       `json:"defects,omitempty"`
	Signals    []Signal       `json:"signals,omitempty"`
	Points     []PointStatus  `json:"points,omitempty"`
	Types      []TypeCoverage `json:"types,omitempty"`
	Escapes    []Escape       `json:"escapes,omitempty"`
	AutoPasses []AutoPass     `json:"auto_passes,omitempty"`
	Requests   []Request      `json:"requests,omitempty"`
	// Axis — ось «состояние качества» (§3b PRD, AD-30); AxisBasis — записи-основания.
	Axis      statuses.Quality `json:"axis"`
	AxisBasis []string         `json:"axis_basis,omitempty"`
}

// Run — выполнение операции (operation.run.started): граница окна
// физического дефекта (FR-37) и сброс полноты участка при повторе (FR-47).
type Run struct {
	EventID    string    `json:"event_id"`
	Pos        int       `json:"pos"`
	OccurredAt time.Time `json:"occurred_at"`
	RunID      string    `json:"run_id"`
	StepKey    string    `json:"step_key"`
	Stage      string    `json:"stage"`
	ReworkOf   string    `json:"rework_of,omitempty"`
	// Inspection — выполнение шага контроля: окно дефекта не меняет.
	Inspection bool `json:"inspection,omitempty"`
}

// Skip — пропуск обязательной проверки исполнителем (operator.check.skipped).
type Skip struct {
	EventID         string    `json:"event_id"`
	Pos             int       `json:"pos"`
	OccurredAt      time.Time `json:"occurred_at"`
	StepKey         string    `json:"step_key"`
	InspectionPoint string    `json:"inspection_point,omitempty"`
	OperationRunID  string    `json:"operation_run_id,omitempty"`
}

// Report — сообщение исполнителя об отклонении (operator.deviation.reported).
type Report struct {
	EventID     string    `json:"event_id"`
	Pos         int       `json:"pos"`
	OccurredAt  time.Time `json:"occurred_at"`
	Zone        string    `json:"zone,omitempty"`
	Description string    `json:"description"`
}

// Presentation — предъявление на закрывающей точке (item.presentation.recorded)
// или решение по нему (decision.presentation.resolved, Resolution непуст).
type Presentation struct {
	EventID      string    `json:"event_id"`
	Pos          int       `json:"pos"`
	OccurredAt   time.Time `json:"occurred_at"`
	StepKey      string    `json:"step_key"`
	ClosingPoint string    `json:"closing_point,omitempty"`
	// Resolution — accept | accept_with_concession | reject | insufficient_data; пусто — только предъявлено.
	Resolution string `json:"resolution,omitempty"`
	// PresentationNo — номер предъявления (FR-19); ConcessionID — разрешение на отклонение.
	PresentationNo int    `json:"presentation_no,omitempty"`
	ConcessionID   string `json:"concession_id,omitempty"`
}

// Review — решение контролёра по сигналам: подтверждено несоответствие
// (decision.nonconformity.confirmed) или сигнал отклонён (decision.signal.rejected).
type Review struct {
	EventID    string    `json:"event_id"`
	Pos        int       `json:"pos"`
	OccurredAt time.Time `json:"occurred_at"`
	SignalIDs  []string  `json:"signal_ids"`
	// Verdict — confirmed | rejected.
	Verdict string `json:"verdict"`
	NCID    string `json:"nc_id,omitempty"`
}

// Recheck — запрошен повторный контроль (decision.recheck.requested).
type Recheck struct {
	EventID    string    `json:"event_id"`
	Pos        int       `json:"pos"`
	OccurredAt time.Time `json:"occurred_at"`
	Method     string    `json:"method"`
	Zones      []string  `json:"zones,omitempty"`
}

// Override — намерение SetQuality позднего модуля (AD-30): ось «состояние
// качества» меняет только quality.
type Override struct {
	Pos    int              `json:"pos"`
	Value  statuses.Quality `json:"value"`
	From   string           `json:"from"`
	Causes []string         `json:"causes,omitempty"`
}

// Env — см. env.go.

// Upstream — состояния модулей раньше quality в композиции на этом шаге
// (только чтение, AD-40): поздний модуль видит вывод раннего, обратно — только
// через функцию-намерение раннего модуля.
type Upstream struct {
	Item    *item.State
	Process *process.State
	Vision  *vision.State
}

// Reduce применяет запись входа изделия (факт, решение, адресованную запись
// стадии) к состоянию модуля и пересчитывает выводы (AD-5). Реакции в свёртку
// не входят (AD-3). Время приходит только из записей (AD-4, AD-37).
func Reduce(s State, r kernel.Record, env Env, up Upstream) State {
	_ = up
	s = s.clone()
	s.Pos++
	s.Current = r.EventID
	if s.ItemID == "" {
		s.ItemID = r.ItemID
	}
	if !s.reduce(r, env) {
		return s
	}
	s.derive(env)
	return s
}

// reduce — вход записи; false — запись модулю не интересна.
func (s *State) reduce(r kernel.Record, env Env) bool {
	bad := func() bool { s.Undecodable = append(s.Undecodable, r.EventID); return true }
	switch r.Type {
	case catalog.InspectionResultRecorded:
		d, err := kernel.Decode[ev.InspectionResultRecordedV1](r)
		if err != nil {
			return bad()
		}
		o := observe(r, d, s.Pos, env)
		o.Window = s.window(o.OperationRunID)
		// Исправление (FR-122) выводит прежнее наблюдение из выводов, но не удаляет его.
		if o.Corrects != "" {
			for i := range s.Observations {
				if s.Observations[i].EventID == o.Corrects {
					s.Observations[i].Superseded = true
				}
			}
		}
		s.Observations = append(s.Observations, o)
	case catalog.OperationRunStarted:
		d, err := kernel.Decode[ev.OperationRunStartedV1](r)
		if err != nil {
			return bad()
		}
		st, _ := env.step(string(d.StepKey))
		run := Run{EventID: r.EventID, Pos: s.Pos, OccurredAt: r.OccurredAt, RunID: string(d.OperationRunID),
			StepKey: string(d.StepKey), Stage: env.stageOf(string(d.StepKey)), Inspection: st.IsInspection()}
		if d.ReworkOf != nil {
			run.ReworkOf = string(*d.ReworkOf)
		}
		s.Runs = append(s.Runs, run)
	case catalog.OperatorCheckSkipped:
		d, err := kernel.Decode[ev.OperatorCheckSkippedV1](r)
		if err != nil {
			return bad()
		}
		sk := Skip{EventID: r.EventID, Pos: s.Pos, OccurredAt: r.OccurredAt, StepKey: string(d.StepKey)}
		if d.InspectionPoint != nil {
			sk.InspectionPoint = *d.InspectionPoint
		}
		if d.OperationRunID != nil {
			sk.OperationRunID = string(*d.OperationRunID)
		}
		s.Skips = append(s.Skips, sk)
	case catalog.OperatorDeviationReported:
		d, err := kernel.Decode[ev.OperatorDeviationReportedV1](r)
		if err != nil {
			return bad()
		}
		rp := Report{EventID: r.EventID, Pos: s.Pos, OccurredAt: r.OccurredAt, Description: string(d.Description)}
		if d.ZoneID != nil {
			rp.Zone = string(*d.ZoneID)
		}
		s.Reports = append(s.Reports, rp)
	case catalog.ItemPresentationRecorded:
		d, err := kernel.Decode[ev.ItemPresentationRecordedV1](r)
		if err != nil {
			return bad()
		}
		st, _ := env.step(string(d.StepKey))
		s.Presentations = append(s.Presentations, Presentation{EventID: r.EventID, Pos: s.Pos, OccurredAt: r.OccurredAt,
			StepKey: string(d.StepKey), ClosingPoint: st.ClosingPoint, PresentationNo: d.PresentationNo})
	case catalog.DecisionPresentationResolved:
		d, err := kernel.Decode[ev.DecisionPresentationResolvedV1](r)
		if err != nil {
			return bad()
		}
		pr := Presentation{EventID: r.EventID, Pos: s.Pos, OccurredAt: r.OccurredAt, StepKey: string(d.StepKey),
			ClosingPoint: d.ClosingPoint, Resolution: string(d.Resolution), PresentationNo: d.PresentationNo}
		if d.ConcessionID != nil {
			pr.ConcessionID = string(*d.ConcessionID)
		}
		s.Presentations = append(s.Presentations, pr)
	case catalog.DecisionNonconformityConfirmed:
		d, err := kernel.Decode[ev.DecisionNonconformityConfirmedV1](r)
		if err != nil {
			return bad()
		}
		s.Reviews = append(s.Reviews, Review{EventID: r.EventID, Pos: s.Pos, OccurredAt: r.OccurredAt,
			SignalIDs: ids(d.SignalIds), Verdict: "confirmed", NCID: string(d.NcID)})
	case catalog.DecisionSignalRejected:
		d, err := kernel.Decode[ev.DecisionSignalRejectedV1](r)
		if err != nil {
			return bad()
		}
		s.Reviews = append(s.Reviews, Review{EventID: r.EventID, Pos: s.Pos, OccurredAt: r.OccurredAt,
			SignalIDs: ids(d.SignalIds), Verdict: "rejected"})
	case catalog.DecisionRecheckRequested:
		d, err := kernel.Decode[ev.DecisionRecheckRequestedV1](r)
		if err != nil {
			return bad()
		}
		s.Rechecks = append(s.Rechecks, Recheck{EventID: r.EventID, Pos: s.Pos, OccurredAt: r.OccurredAt,
			Method: string(d.Method), Zones: ids(d.ZoneIds)})
	default:
		return false
	}
	return true
}

// window — окно физического дефекта наблюдения (FR-37): выполнение операции,
// указанное в наблюдении, если это известная операция (не контроль), иначе
// последнее начатое выполнение операции до наблюдения.
func (s *State) window(runID string) string {
	if runID != "" {
		for _, r := range s.Runs {
			if r.RunID == runID && !r.Inspection {
				return runID
			}
		}
	}
	for i := len(s.Runs) - 1; i >= 0; i-- {
		if !s.Runs[i].Inspection {
			return s.Runs[i].RunID
		}
	}
	return ""
}

// React вычисляет реакции модуля по состоянию после записи и намерения к
// ранним модулям (AD-3, AD-40): сигналы, дефекты и повторные наблюдения,
// «нет данных контроля», пропуски брака, автоматические пропуски к следующему
// контролю; продвижение токена точки предъявления — функцией process.
func React(s State, env Env, up Upstream) kernel.Output {
	_ = up
	var out kernel.Output
	if !env.IsZero() {
		// Без нормативного слоя реакций нет (см. derive): сверять не с чем.
		out.Reactions = reactions(s, env)
	}
	for _, p := range s.Presentations {
		// Решение человека на точке предъявления продвигает токен процесса
		// функцией-намерением process (AD-40, FR-44) — на шаге свёртки самой
		// записи-решения; process применяет его идемпотентно по решению.
		// Решение человека quality не подменяет: неполнота контроля (FR-35,
		// FR-14) видна на оси «состояние качества» и в PresentationBlockers.
		if p.EventID == s.Current && p.Resolution != "" {
			out.Intents = append(out.Intents, process.AdvancePresentation(Module, process.Presentation{
				StepKey: p.StepKey, Resolution: p.Resolution, DecisionEventID: p.EventID,
				PresentationNo: p.PresentationNo, ConcessionID: p.ConcessionID}, s.record(p.EventID)))
		}
	}
	return out
}

// Guard — доменный гард операций модуля quality (AD-39). Команд у модуля нет:
// результаты контроля — факты приёма, решения — nonconformity (AD-30).
// Гарду решения на точке предъявления (nonconformity) нужна PresentationBlockers.
func Guard(s State, env Env, up Upstream, cmd kernel.Command) error {
	_, _, _, _ = s, env, up, cmd
	return nil
}

// Apply применяет намерение, адресованное модулю quality (Intent.Target == Module),
// в конце шага свёртки (AD-40): SetQuality позднего модуля меняет ось
// «состояние качества». Неизвестное намерение — без изменений.
func Apply(s State, in kernel.Intent) State {
	if in.Target != Module || in.Name != IntentSetQuality {
		return s
	}
	v, ok := in.Payload.(statuses.Quality)
	if !ok {
		return s
	}
	s = s.clone()
	s.Overrides = append(s.Overrides, Override{Pos: s.Pos, Value: v, From: string(in.From), Causes: slices.Clone(in.Causes)})
	s.Axis, s.AxisBasis = axis(s)
	return s
}

// clone — копия состояния без общих срезов с предыдущей версией: свёртка
// хранит состояние на каждом шаге (запросы на момент, AD-22).
func (s State) clone() State {
	s.Observations = slices.Clone(s.Observations)
	s.Runs = slices.Clone(s.Runs)
	s.Skips = slices.Clone(s.Skips)
	s.Reports = slices.Clone(s.Reports)
	s.Presentations = slices.Clone(s.Presentations)
	s.Reviews = slices.Clone(s.Reviews)
	s.Rechecks = slices.Clone(s.Rechecks)
	s.Overrides = slices.Clone(s.Overrides)
	s.Undecodable = slices.Clone(s.Undecodable)
	return s
}

func ids[T ~string](in []T) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		out = append(out, string(v))
	}
	return out
}
