package nonconformity

import (
	"slices"
	"strconv"
	"strings"

	"ant/internal/contracts/errcodes"
	"ant/internal/contracts/statuses"
	"ant/internal/domain/kernel"
	"ant/internal/domain/process"
	"ant/internal/domain/quality"
)

// Операции модуля (x-ant-action, AD-40) — ключи гарда.
const (
	ActConfirm            = "nonconformity.nonconformity.confirm"
	ActRejectSignal       = "nonconformity.signal.reject"
	ActRecheck            = "nonconformity.recheck.request"
	ActIsolate            = "nonconformity.item.isolate"
	ActPresentation       = "nonconformity.presentation.resolve"
	ActLot                = "nonconformity.lot.resolve"
	ActDisposition        = "nonconformity.disposition.set"
	ActVerify             = "nonconformity.disposition.verify"
	ActContainmentSet     = "nonconformity.containment.set"
	ActContainmentRelease = "nonconformity.containment.release"
	ActConcessionGrant    = "nonconformity.concession.grant"
	ActConcessionRevoke   = "nonconformity.concession.revoke"
	ActReworkWaive        = "nonconformity.rework_limit.waive"
	ActProcessHoldSet     = "nonconformity.process_hold.set"
	ActProcessHoldRelease = "nonconformity.process_hold.release"
	ActClose              = "nonconformity.nonconformity.close"
)

// Guard — доменный гард операций модуля nonconformity над изделием (AD-39):
// состояние изделия на basis_seq и команда → nil или *kernel.Refusal с кодом
// из contracts/errors.yaml. Его вызывают api до записи, свёртка при
// применении и верификатор. Payload команды — данные записи, которую
// операция пишет (DispositionSetData и т. п.); Actor — псевдоним сотрудника.
//
// Гарды объектов вне изделия (разрешение на отклонение, остановка точки
// процесса) — ConcessionGuard и HoldGuard.
func Guard(s State, env Env, up Upstream, cmd kernel.Command) error {
	_, _ = env, up
	switch p := cmd.Payload.(type) {
	case ConfirmedData:
		n, err := s.mustNC(p.NCID)
		if err != nil {
			return err
		}
		if n.Status != StatusDraft {
			return invalid(cmd.Action, n)
		}
		for _, sid := range p.SignalIDs {
			if !slices.Contains(n.Draft.SignalIDs, sid) || s.SignalRejected(sid) {
				return refuse(errcodes.NonconformityInvalidTransition, "action", "подтвердить", "nc_id", n.ID,
					"status", "сигнал "+sid+" не относится к черновику или отклонён")
			}
		}
	case SignalRejectedData:
		// FR-52: отклонение без причины невозможно.
		if strings.TrimSpace(p.Reason.Text) == "" {
			return kernel.Refuse(errcodes.NonconformityRejectReasonRequired)
		}
		for _, sid := range p.SignalIDs {
			for _, n := range s.NCs {
				if slices.Contains(n.Draft.SignalIDs, sid) && n.Status != StatusDraft {
					return invalid(cmd.Action, n)
				}
			}
			if s.SignalRejected(sid) {
				return refuse(errcodes.NonconformityInvalidTransition, "action", "отклонить сигнал", "nc_id", sid, "status", "сигнал уже отклонён")
			}
		}
	case IsolatedData:
		if s.Isolated() {
			return refuse(errcodes.NonconformityInvalidTransition, "action", "изолировать", "nc_id", s.ItemID, "status", "уже в изоляции")
		}
	case PresentationResolvedData:
		return guardPresentation(s, env, up, cmd, p)
	case DispositionSetData:
		n, err := s.mustNC(p.NCID)
		if err != nil {
			return err
		}
		if n.Status != StatusConfirmed {
			return invalid(cmd.Action, n)
		}
		switch p.Disposition {
		case "repair", "use_as_is":
			// FR-53, FR-54: ремонт и «как есть» — только по действующему
			// разрешению на отклонение; применимость проверяет ConcessionGuard.
			if p.ConcessionID == "" {
				return kernel.Refuse(errcodes.NonconformityConcessionRequired, "decision", DispositionLabel(p.Disposition))
			}
		case "return_to_supplier":
			// FR-53: вернуть поставщику — только необработанное.
			if s.Processed {
				return kernel.Refuse(errcodes.NonconformityReturnOnlyUnprocessed, "item_id", s.ItemID)
			}
		}
	case DispositionVerifiedData:
		n, err := s.mustNC(p.NCID)
		if err != nil {
			return err
		}
		if n.Status != StatusDispositionSet || !n.Executed {
			return invalid(cmd.Action, n)
		}
		for _, id := range p.RecheckEventIDs {
			if !slices.Contains(s.Inspections, id) {
				return kernel.Refuse(errcodes.NonconformityMethodResultMissing, "method", "повторного контроля", "inspection_point", id)
			}
		}
	case ClosedData:
		n, err := s.mustNC(p.NCID)
		if err != nil {
			return err
		}
		switch {
		case n.Status == StatusVerified:
		case n.Status == StatusDispositionSet && n.Executed && (n.Disposition == "scrap" || n.Disposition == "use_as_is" || n.Disposition == "return_to_supplier"):
		default:
			return invalid(cmd.Action, n)
		}
	case ContainmentSetData:
		if levelRank(p.Level) == 0 {
			return refuse(errcodes.NonconformityInvalidTransition, "action", "установить сдерживание", "nc_id", s.ItemID, "status", "уровень «none» — это снятие")
		}
	case ContainmentReleasedData:
		matched := false
		for _, c := range s.Containment {
			hit := slices.Contains(p.ReleasedEventIDs, c.Key) || (c.NCID != "" && slices.Contains(p.ReleasedEventIDs, c.NCID))
			if c.Released || !hit {
				continue
			}
			matched = true
			// Снятие блока при действующем основании правила (активный
			// инцидент, окно спецпроцесса) — отказ: сначала основание (AD-39).
			if c.By == ByRule && levelRank(c.Basis) >= levelRank(string(statuses.ContainmentItemHold)) && c.Rule == RuleIncidentScope {
				return kernel.Refuse(errcodes.NonconformityItemBlocked)
			}
		}
		if s.Isolation != nil && !s.Isolation.Released && slices.Contains(p.ReleasedEventIDs, s.Isolation.EventID) {
			matched = true
		}
		if !matched {
			return refuse(errcodes.NonconformityInvalidTransition, "action", "снять сдерживание", "nc_id", s.ItemID, "status", "нет действующего сдерживания с такими записями")
		}
	}
	return nil
}

func guardPresentation(s State, env Env, up Upstream, cmd kernel.Command, p PresentationResolvedData) error {
	accept := p.Resolution == "accept" || p.Resolution == "accept_with_concession"
	// FR-56: участник изготовления не принимает изделие на точке предъявления.
	if accept && cmd.Actor != "" && s.Participant(cmd.Actor) {
		return kernel.Refuse(errcodes.AccessSeparationOfDuties)
	}
	for _, pr := range s.Presentations {
		if pr.StepKey == p.StepKey && pr.PresentationNo == p.PresentationNo && pr.ResolvedEventID != "" {
			return refuse(errcodes.NonconformityInvalidTransition, "action", "решение на точке предъявления", "nc_id", s.ItemID,
				"status", "предъявление №"+strconv.Itoa(p.PresentationNo)+" уже решено")
		}
	}
	// Изделие стоит на точке, приёмка — не при открытом вмешательстве: гард
	// точки предъявления модуля process над его состоянием (AD-40, эпик 16).
	if up.Process != nil && env.Process.Def != nil {
		if err := process.PresentationGuard(*up.Process, env.Process, p.StepKey, p.Resolution); err != nil {
			return err
		}
	}
	if !accept {
		return nil
	}
	if s.InterventionOpen {
		return kernel.Refuse(errcodes.NonconformityInterventionOpen)
	}
	if s.Blocked() {
		return kernel.Refuse(errcodes.NonconformityItemBlocked)
	}
	for _, n := range s.NCs {
		// Открытое несоответствие без исполняемого решения — изделие не
		// принимается (снятие блока ≠ годность, AD-30).
		decided := (n.Status == StatusDispositionSet && n.Executed) || n.Status == StatusVerified || s.Covered(n)
		if n.Open() && !decided {
			return kernel.Refuse(errcodes.NonconformityItemBlocked)
		}
	}
	for _, id := range p.MethodEventIDs {
		if !slices.Contains(s.Inspections, id) {
			return kernel.Refuse(errcodes.NonconformityMethodResultMissing, "method", "контроля", "inspection_point", p.ClosingPoint)
		}
	}
	// FR-35, FR-44, FR-48: полнота контроля и открытые сигналы участка —
	// чистая функция quality над его состоянием на basis_seq.
	if up.Quality != nil && !env.Quality.IsZero() {
		if bs := quality.PresentationBlockers(*up.Quality, env.Quality, p.StepKey); len(bs) > 0 {
			b := bs[0]
			if b.Code == "open_signal" {
				return kernel.Refuse(errcodes.NonconformityItemBlocked)
			}
			return kernel.Refuse(errcodes.NonconformityMethodResultMissing, "method", "контроля ("+blockerText(b.Code)+")", "inspection_point", b.StepKey)
		}
	}
	if p.Resolution == "accept_with_concession" && p.ConcessionID == "" {
		return kernel.Refuse(errcodes.NonconformityConcessionRequired, "decision", "Принять по разрешению на отклонение")
	}
	return nil
}

func blockerText(code string) string {
	switch code {
	case "inspection_missing":
		return "нет данных"
	case "inspection_pending":
		return "результат ещё не получен"
	case "unable_to_assess":
		return "оценка невозможна"
	}
	return code
}

// DispositionLabel — вариант решения по-русски (для текста отказа).
func DispositionLabel(d string) string {
	switch d {
	case "rework":
		return "Переделка"
	case "repair":
		return "Ремонт"
	case "use_as_is":
		return "Как есть"
	case "scrap":
		return "Списать"
	case "return_to_supplier":
		return "Вернуть поставщику"
	}
	return d
}

func (s State) mustNC(id string) (NC, error) {
	n, ok := s.NC(id)
	if !ok {
		return NC{}, kernel.Refuse(errcodes.ApiNotFound, "object", "Несоответствие", "id", id)
	}
	return n, nil
}

func invalid(action string, n NC) error {
	st := n.Status
	if n.Status == StatusDispositionSet && !n.Executed {
		st = "решение ждёт подписей"
	}
	return refuse(errcodes.NonconformityInvalidTransition, "action", ActionLabel(action), "nc_id", n.Number, "status", st)
}

func refuse(code errcodes.Code, kv ...string) error { return kernel.Refuse(code, kv...) }

// ActionLabel — название операции для людей.
func ActionLabel(action string) string {
	switch action {
	case ActConfirm:
		return "Подтвердить несоответствие"
	case ActRejectSignal:
		return "Отклонить сигнал"
	case ActDisposition:
		return "Решение по несоответствию"
	case ActVerify:
		return "Подтвердить выполнение решения"
	case ActClose:
		return "Закрыть несоответствие"
	}
	return action
}

func compactSorted(ids []string) []string {
	slices.Sort(ids)
	return slices.Compact(ids)
}
