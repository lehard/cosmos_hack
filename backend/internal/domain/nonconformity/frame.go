package nonconformity

import (
	"encoding/json"
	"maps"
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/statuses"
	"ant/internal/domain/documents"
	"ant/internal/domain/item"
	"ant/internal/domain/kernel"
	"ant/internal/domain/machinelogs"
	"ant/internal/domain/process"
	"ant/internal/domain/quality"
	"ant/internal/domain/vision"
)

// Module — имя модуля-эмитента (AD-40).
const Module kernel.Module = "nonconformity"

// Правила модуля — слоты реакций (AD-3) и основания сдерживания (FR-50: у
// каждого автоматического действия — правило).
const (
	// RuleDraft — черновик несоответствия по намерению (режим 1, FR-50).
	RuleDraft = "nonconformity.draft"
	// RuleSignalContainment — блок изделия по реакции «изолировать» карты
	// реакций (FR-48, FR-49, режим 1).
	RuleSignalContainment = "nonconformity.signal_containment"
	// RuleSpecialProcess — блок изделия окна нарушения специального процесса (FR-151).
	RuleSpecialProcess = "nonconformity.special_process"
	// RuleIncidentScope — сдерживание по статусу изделия в области риска
	// инцидента (FR-62): блок / доп. проверка / наблюдать; снятие — только по
	// явно делегированному правилу (Env.DelegatedIncidentRelease) или человеком (AD-27).
	RuleIncidentScope = "nonconformity.incident_scope"
	// RuleIntentContain — сдерживание по намерению Contain другого модуля.
	RuleIntentContain = "nonconformity.intent_contain"
	// RuleCleanPoint — усиленный контроль точки чистоты (FR-49).
	RuleCleanPoint = "nonconformity.clean_point"
)

// Env — закреплённая при запуске изделия часть нормативного слоя, нужная
// модулю nonconformity (AD-17), и срез справочников (AD-31). Собирает движок
// из пакета версии; нулевое значение — умолчания фланца.
type Env struct {
	// ClosingPoints — закрывающие точки шагов предъявления (step_key → ZT-…);
	// пусто — DefaultClosingPoints.
	ClosingPoints map[string]string `json:"closing_points,omitempty"`
	// DelegatedIncidentRelease — снятие блока по области риска инцидента
	// делегировано правилу режима 2 (FR-50, FR-144). Нет — снятие ставит
	// задачу человеку: реакция сдерживания исчезает, движок пишет задачу
	// «основание защиты изменилось — пересмотрите» (AD-3), блок в оси остаётся.
	DelegatedIncidentRelease bool `json:"delegated_incident_release,omitempty"`
	// Quality — нормативная часть quality (план контроля, шаги предъявления)
	// для гарда решения на точке предъявления: quality.PresentationBlockers
	// (FR-35, FR-44). Нулевая — гард без проверки полноты контроля quality.
	Quality quality.Env `json:"-"`
	// Process — нормативный слой процесса изделия (закреплённая версия) для
	// гарда решения на точке предъявления: process.PresentationGuard (FR-19,
	// FR-21, FR-44; стык эпиков 17 и 21, эпик 16). Нулевой — без проверки.
	Process process.Env `json:"-"`
}

// Upstream — состояния модулей раньше nonconformity в композиции на этом шаге
// (только чтение, AD-40): поздний модуль видит вывод раннего, обратно — только
// через функцию-намерение раннего модуля.
type Upstream struct {
	Item        *item.State
	Process     *process.State
	Vision      *vision.State
	Quality     *quality.State
	Machinelogs *machinelogs.State
	Documents   *documents.State
}

// ── Reduce ──

// Reduce применяет запись входа изделия (факт, решение, адресованную запись
// стадии) к состоянию модуля (AD-5). Реакции в свёртку не входят (AD-3).
// Исходные записи не меняются: каждое решение — новая запись, и свёртка
// только добавляет его к состоянию (FR-51).
func Reduce(s State, r kernel.Record, env Env, up Upstream) State {
	_ = up
	s = s.clone()
	s.Effects = nil
	if r.OccurredAt.After(s.At) {
		s.At = r.OccurredAt
	}
	if s.ItemID == "" && r.ItemID != "" {
		s.ItemID = r.ItemID
	}
	switch r.Type {
	case catalog.OperationRunStarted:
		var d struct {
			OperatorID *string `json:"operator_id"`
		}
		if decode(r, &d) {
			s.Processed = true
			if d.OperatorID != nil && *d.OperatorID != "" {
				s.addParticipant(*d.OperatorID)
			}
		}
	case catalog.InspectionResultRecorded:
		s.Inspections = append(s.Inspections, r.EventID)
	case catalog.ItemPresentationRecorded:
		var d struct {
			StepKey        string `json:"step_key"`
			PresentationNo int    `json:"presentation_no"`
		}
		if decode(r, &d) {
			s.Presentations = append(s.Presentations, Presentation{
				EventID: r.EventID, StepKey: d.StepKey, ClosingPoint: env.ClosingPoint(d.StepKey),
				PresentationNo: d.PresentationNo, At: r.OccurredAt,
			})
		}
	case catalog.ItemInterventionOpened:
		s.InterventionOpen = true
	case catalog.ItemInterventionClosed:
		s.InterventionOpen = false
	case catalog.OperationMovementReceived:
		var d struct {
			DestinationKind string `json:"destination_kind"`
			ToLocationID    string `json:"to_location_id"`
		}
		// FR-55: приёмка в изоляторе снимает расхождение «изолировано в
		// системе, физически не перемещено».
		if decode(r, &d) && d.DestinationKind == "isolator" && s.Isolation != nil && !s.Isolation.Released {
			s.Isolation.PhysicallyMoved = true
			s.Isolation.MovedEventID = r.EventID
			if s.Isolation.IsolatorLocationID == "" {
				s.Isolation.IsolatorLocationID = d.ToLocationID
			}
		}
	case catalog.IncidentMembershipChanged:
		s.reduceIncident(r, env)
	case catalog.IncidentIncidentClosed:
		for i := range s.NCs {
			if s.NCs[i].Investigation == InvestigationOpen {
				s.NCs[i].Investigation = InvestigationClosed
			}
		}
	case catalog.DocumentRouteClosed:
		var d struct {
			DocumentID string `json:"document_id"`
		}
		if decode(r, &d) {
			for i := range s.NCs {
				n := &s.NCs[i]
				if n.DocumentID == d.DocumentID && n.Disposition != "" && !n.Executed {
					n.Executed, n.ApprovalsStatus = true, ApprovalsRouteClosed
					s.executed(*n, r)
				}
			}
		}
	case catalog.DecisionNonconformityRegistered:
		var d RegisteredData
		if decode(r, &d) && s.nc(d.NCID) == nil {
			n := NC{
				ID: d.NCID, Number: NCNumber(d.NCID), Origin: OriginSpecialProcess, FoundAt: r.OccurredAt,
				Causes: []string{r.EventID}, From: string(machinelogs.Module),
				Draft: DraftedData{
					NCID: d.NCID, SignalIDs: []string{d.ViolationWindowEventID}, BasisKind: "special_process_violation",
					StepKey: d.StepKey, OperationRunID: d.OperationRunID, Severity: "unknown",
				},
				Commission: true, WindowEventID: d.ViolationWindowEventID,
				// Регистрация правилом — уже несоответствие техпроцесса (FR-151):
				// контролёру подтверждать нечего, решение — за комиссией.
				Status: StatusConfirmed, Investigation: InvestigationOpen, Severity: "unknown",
			}
			s.NCs = append(s.NCs, n)
			s.addSource(ContainmentSource{Key: r.EventID, Level: string(statuses.ContainmentItemHold), By: ByRule,
				Rule: RuleSpecialProcess, NCID: d.NCID, At: r.OccurredAt, Reason: "Нарушение режима специального процесса (FR-151)"})
			// Ось качества правило не трогает: дефект не найден, изделие «не
			// годно и не брак — ждёт» решения комиссии (блок правилом выше;
			// S04-03). «Не годно» ставят решения людей.
		}
	case catalog.DecisionNonconformityConfirmed:
		var d ConfirmedData
		if decode(r, &d) {
			s.decision(r, d.NCID, "Несоответствие подтверждено: "+d.Reason.Text)
			if n := s.nc(d.NCID); n != nil && n.Status == StatusDraft {
				n.Status, n.ConfirmedEventID = StatusConfirmed, r.EventID
				n.Severity = d.Severity
				if d.DefectTypeCode != "" {
					n.DefectTypeCode = d.DefectTypeCode
				}
				if d.RequirementRef != "" {
					n.RequirementRef = d.RequirementRef
				}
				if d.FullAnalysisRequired {
					n.Investigation = InvestigationOpen
				}
				// Ось «состояние качества» по подтверждению quality ведёт сам:
				// он читает decision.nonconformity.confirmed (эпик 20).
			}
		}
	case catalog.DecisionSignalRejected:
		var d SignalRejectedData
		if decode(r, &d) {
			s.decision(r, "", "Сигнал отклонён: "+d.Reason.Text)
			s.Rejections = append(s.Rejections, Rejection{EventID: r.EventID, SignalIDs: slices.Clone(d.SignalIDs), Reason: d.Reason.Text})
			// Сдерживание правилом, основанное только на отклонённом сигнале,
			// снимает этим же решением человек (AD-27: разрешающее — человеком).
			for i := range s.Containment {
				c := &s.Containment[i]
				if c.By == ByRule && c.SignalID != "" && slices.Contains(d.SignalIDs, c.SignalID) {
					c.Released, c.ReleasedBy = true, r.EventID
				}
			}
			// Черновик, все сигналы которого отклонены, закрывается без
			// решения по изделию; сами сигналы и черновик не меняются (FR-51).
			for i := range s.NCs {
				n := &s.NCs[i]
				if n.Status != StatusDraft {
					continue
				}
				all := len(n.Draft.SignalIDs) > 0
				for _, sid := range n.Draft.SignalIDs {
					all = all && s.SignalRejected(sid)
				}
				if all {
					n.Status, n.Resolution, n.ClosedEventID = StatusClosed, ResolutionSignalRejected, r.EventID
					s.releaseNC(n.ID)
				}
			}
		}
	case catalog.DecisionRecheckRequested:
		var d RecheckData
		if decode(r, &d) {
			s.decision(r, "", "Назначена доп. проверка: "+d.Method)
			s.Rechecks = append(s.Rechecks, Recheck{EventID: r.EventID, Method: d.Method, DueAt: d.DueAt})
			// FR-52: изделие не маршрутизируется автоматически — ждёт доп. проверки.
			s.addSource(ContainmentSource{Key: r.EventID, Level: string(statuses.ContainmentAdditionalCheck), By: ByHuman,
				At: r.OccurredAt, Reason: d.Reason.Text})
		}
	case catalog.DecisionItemIsolated:
		var d IsolatedData
		if decode(r, &d) {
			s.decision(r, "", "Изделие изолировано: "+d.Reason.Text)
			iso := &Isolation{EventID: r.EventID, At: r.OccurredAt, IsolatorLocationID: d.IsolatorLocationID}
			if t, err := time.Parse(time.RFC3339Nano, d.DecisionDueAt); err == nil {
				t = t.UTC()
				iso.DecisionDueAt = &t
			}
			s.Isolation = iso
			s.addSource(ContainmentSource{Key: r.EventID, Level: string(statuses.ContainmentItemHold), By: ByHuman,
				At: r.OccurredAt, Reason: d.Reason.Text})
			s.effect(Effect{Kind: "isolate", Value: d.Reason.Text}, r)
		}
	case catalog.DecisionContainmentSet:
		var d ContainmentSetData
		if decode(r, &d) {
			s.decision(r, "", "Сдерживание: "+d.Level)
			if d.Level != string(statuses.ContainmentNone) {
				s.addSource(ContainmentSource{Key: r.EventID, Level: d.Level, By: ByHuman, At: r.OccurredAt, Reason: d.Reason.Text})
			}
		}
	case catalog.DecisionContainmentReleased:
		var d ContainmentReleasedData
		if decode(r, &d) {
			s.decision(r, "", "Сдерживание снято: "+d.Reason.Text)
			for i := range s.Containment {
				c := &s.Containment[i]
				if slices.Contains(d.ReleasedEventIDs, c.Key) || (c.NCID != "" && slices.Contains(d.ReleasedEventIDs, c.NCID)) {
					c.Released = true
				}
			}
			if s.Isolation != nil && slices.Contains(d.ReleasedEventIDs, s.Isolation.EventID) {
				s.Isolation.Released = true
			}
		}
	case catalog.DecisionDispositionSet:
		var d DispositionSetData
		if decode(r, &d) {
			s.decision(r, d.NCID, "Решение по несоответствию: "+d.Disposition)
			if n := s.nc(d.NCID); n != nil && (n.Status == StatusConfirmed) {
				n.Status, n.DispositionEventID = StatusDispositionSet, r.EventID
				n.Disposition, n.ScrapKind, n.ConcessionID, n.DocumentID = d.Disposition, d.ScrapKind, d.ConcessionID, d.DocumentID
				n.ApprovalsStatus = d.ApprovalsStatus
				if n.ApprovalsStatus == "" {
					n.ApprovalsStatus = ApprovalsRouteClosed
				}
				// FR-50 режим 4, AD-43: решение исполняется только после
				// закрытия маршрута подписей (или демо-заглушки порта).
				if n.ApprovalsStatus != ApprovalsPending {
					n.Executed = true
					s.executed(*n, r)
				}
			}
		}
	case catalog.DecisionDispositionVerified:
		var d DispositionVerifiedData
		if decode(r, &d) {
			s.decision(r, d.NCID, "Исполнение решения проверено")
			if n := s.nc(d.NCID); n != nil && n.Status == StatusDispositionSet {
				n.Status, n.VerifiedEventID = StatusVerified, r.EventID
				if n.Disposition == "repair" {
					// FR-53: после ремонта и повторного контроля — «годно по
					// разрешению на отклонение», не «годно».
					s.effect(Effect{Kind: "set_quality", Value: string(statuses.QualityAcceptedWithConcession)}, r)
				}
			}
		}
	case catalog.DecisionNonconformityClosed:
		var d ClosedData
		if decode(r, &d) {
			s.decision(r, d.NCID, "Несоответствие закрыто по изделию")
			if n := s.nc(d.NCID); n != nil && n.Status != StatusClosed {
				// Закрытие по изделию не закрывает системное расследование (FR-51).
				n.Status, n.ClosedEventID = StatusClosed, r.EventID
			}
		}
	case catalog.DecisionPresentationResolved:
		var d PresentationResolvedData
		if decode(r, &d) {
			s.decision(r, "", "Решение на точке предъявления "+d.ClosingPoint+": "+d.Resolution)
			for i := len(s.Presentations) - 1; i >= 0; i-- {
				p := &s.Presentations[i]
				if p.ResolvedEventID == "" && p.StepKey == d.StepKey && p.PresentationNo == d.PresentationNo {
					p.ResolvedEventID, p.Resolution = r.EventID, d.Resolution
					break
				}
			}
			// Продвижение токена точки (process.AdvancePresentation) и ось
			// качества по решению на точке ведёт quality: он читает
			// decision.presentation.resolved сам (эпик 20).
		}
	case catalog.DecisionPresentationReviewed:
		// Д-81: пересмотр — новая запись; прежнее решение остаётся в журнале.
		var d PresentationReviewedData
		if decode(r, &d) {
			s.decision(r, "", "Пересмотр решения на точке "+d.ClosingPoint+": "+d.Outcome)
			found := false
			for i := range s.Presentations {
				p := &s.Presentations[i]
				if p.ResolvedEventID != "" && p.ResolvedEventID == d.ReviewedEventID {
					p.ReviewEventID, p.ReviewOutcome, found = r.EventID, d.Outcome, true
				}
			}
			if found && d.Outcome == ReviewRevoked {
				// Отзыв приёмки ≠ «не годно»: несоответствие не создаётся,
				// решение по изделию не меняется; блок человека (снимает
				// уполномоченный) и качество «не проверено» — годность не доказана.
				s.addSource(ContainmentSource{Key: r.EventID, Level: string(statuses.ContainmentItemHold), By: ByHuman, At: r.OccurredAt,
					Reason: "Приёмка " + d.ClosingPoint + " отозвана при пересмотре: " + d.Reason.Text})
				s.effect(Effect{Kind: "set_quality", Value: string(statuses.QualityNotInspected)}, r)
			}
		}
	case catalog.DecisionReworkLimitWaived:
		var d ReworkWaivedData
		if decode(r, &d) {
			s.decision(r, "", "Разрешено сверх лимита доработок")
			if s.ReworkWaivers == nil {
				s.ReworkWaivers = map[string]int{}
			}
			s.ReworkWaivers[d.ZoneID] += d.ExtraAllowed
		}
	case catalog.DecisionCleanPointAssigned:
		var d CleanPointData
		if decode(r, &d) {
			s.addSource(ContainmentSource{Key: r.EventID, Level: string(statuses.ContainmentAdditionalCheck), By: ByRule,
				Rule: RuleCleanPoint, At: r.OccurredAt, Reason: "Точка чистоты после снятия остановки " + d.HoldID})
		}
	}
	s.fromQuality(up, r)
	s.fromDocuments(up, r)
	return s
}

// fromDocuments — решения режима 4–5 с подписями «ожидаются» исполняются,
// когда модуль documents закрыл маршрут подписей документа решения (AD-43,
// AD-40: documents стоит в композиции раньше; закрытие вычисляется в свёртке
// заново по подписям — documents.RouteClosed, эпик 28). Подписи
// nonconformity сам не считает.
func (s *State) fromDocuments(up Upstream, r kernel.Record) {
	if up.Documents == nil {
		return
	}
	for i := range s.NCs {
		n := &s.NCs[i]
		if n.Disposition == "" || n.Executed || n.ApprovalsStatus != ApprovalsPending || n.DocumentID == "" {
			continue
		}
		if documents.RouteClosed(*up.Documents, n.DocumentID) {
			n.Executed, n.ApprovalsStatus = true, ApprovalsRouteClosed
			s.executed(*n, r)
		}
	}
}

// fromQuality — намерения quality, выраженные данными его состояния
// (quality.State.Requests, AD-40): quality стоит в композиции раньше и не
// вызывает функции nonconformity, поэтому nonconformity читает запросы через
// Upstream.Quality на том же шаге свёртки. draft_nc — черновик
// несоответствия (реакция decision.nonconformity.drafted), contain —
// сдерживание правилом карты реакций (decision.containment.applied); task —
// зона notifications.
func (s *State) fromQuality(up Upstream, r kernel.Record) {
	if up.Quality == nil || s.ItemID == "" {
		return
	}
	q := up.Quality
	signal := func(id string) (quality.Signal, bool) {
		i := slices.IndexFunc(q.Signals, func(sg quality.Signal) bool { return sg.SignalID == id })
		if i < 0 {
			return quality.Signal{}, false
		}
		return q.Signals[i], true
	}
	for _, rq := range q.Requests {
		if rq.SignalID == "" || s.SignalRejected(rq.SignalID) {
			continue
		}
		ncID := DraftNCID(s.ItemID, []string{rq.SignalID})
		switch rq.Kind {
		case quality.RequestDraftNC:
			if s.nc(ncID) != nil {
				continue
			}
			sg, _ := signal(rq.SignalID)
			d := DraftedData{NCID: ncID, SignalIDs: []string{rq.SignalID}, BasisKind: basisOf(sg.Basis), Severity: sg.Severity,
				DefectTypeCode: sg.TypeCode, ZoneID: sg.Zone, StepKey: rq.StepKey, RequirementRef: sg.RequirementRef,
				ReactionOutcome: outcomeOf(sg.Assessment.Outcome), ReactionMapRef: rq.RuleRef}
			if d.Severity == "" {
				d.Severity = "unknown"
			}
			s.NCs = append(s.NCs, NC{
				ID: ncID, Number: NCNumber(ncID), Origin: OriginSignal, FoundAt: r.OccurredAt, Causes: slices.Clone(rq.Causes),
				From: string(quality.Module), Draft: d, Status: StatusDraft, Investigation: InvestigationNone, Severity: d.Severity,
				DefectTypeCode: d.DefectTypeCode, RequirementRef: d.RequirementRef, SignalID: rq.SignalID, AutomationMode: rq.AutomationMode,
			})
		case quality.RequestContain:
			key := "quality:" + rq.Key
			lvl := string(rq.Containment)
			if levelRank(lvl) == 0 {
				continue
			}
			i := slices.IndexFunc(s.Containment, func(c ContainmentSource) bool { return c.Key == key })
			if i >= 0 {
				// Защитное — поднимается само (AD-27); ниже — только человеком.
				if c := &s.Containment[i]; !c.Released && levelRank(lvl) > levelRank(c.Level) {
					c.Level, c.Basis = lvl, lvl
				}
				continue
			}
			s.Containment = append(s.Containment, ContainmentSource{Key: key, Level: lvl, Basis: lvl, By: ByRule,
				Rule: RuleSignalContainment, NCID: ncID, SignalID: rq.SignalID, At: r.OccurredAt, Causes: slices.Clone(rq.Causes),
				Reason: "Карта реакций " + rq.RuleRef})
		}
	}
}

// basisOf — основание сигнала quality в перечислении контракта черновика.
func basisOf(b string) string {
	switch b {
	case "inspection_result", "equipment_deviation", "check_skipped", "damage_on_receipt", "leak", "special_process_violation", "operator_report":
		return b
	}
	return "inspection_result"
}

// outcomeOf — реакция карты в перечислении контракта черновика.
func outcomeOf(o string) string {
	switch o {
	case "pass_to_next", "manual_review", "isolate", "question_to_technologist":
		return o
	}
	return ""
}

// reduceIncident — FR-62: действие по изделию в области риска инцидента.
// block / check / observe ставят или поднимают сдерживание правилом;
// release снимает его только по делегированному правилу (FR-144), иначе
// основание уходит, а блок остаётся до решения человека (AD-27, AD-3).
func (s *State) reduceIncident(r kernel.Record, env Env) {
	var d struct {
		IncidentID string `json:"incident_id"`
		Action     string `json:"action"`
	}
	if !decode(r, &d) || d.IncidentID == "" {
		return
	}
	key := "incident:" + d.IncidentID
	level := ""
	switch d.Action {
	case "block":
		level = string(statuses.ContainmentItemHold)
	case "check":
		level = string(statuses.ContainmentAdditionalCheck)
	case "observe":
		level = string(statuses.ContainmentObserve)
	case "release":
		level = string(statuses.ContainmentNone)
	default:
		return
	}
	i := slices.IndexFunc(s.Containment, func(c ContainmentSource) bool { return c.Key == key && !c.Released })
	if i < 0 {
		if level == string(statuses.ContainmentNone) {
			return
		}
		s.Containment = append(s.Containment, ContainmentSource{Key: key, Level: level, By: ByRule, Rule: RuleIncidentScope,
			At: r.OccurredAt, Reason: "Область риска инцидента " + d.IncidentID, Basis: level, BasisEventID: r.EventID})
		return
	}
	c := &s.Containment[i]
	c.Basis, c.BasisEventID = level, r.EventID
	switch {
	case levelRank(level) >= levelRank(c.Level):
		// Защитное — движок исполняет сам (AD-27).
		c.Level = level
	case env.DelegatedIncidentRelease:
		// Разрешающее по явно делегированному правилу режима 2 (FR-50).
		c.Level = level
		if level == string(statuses.ContainmentNone) {
			c.Released = true
		}
	}
}

// executed — последствия исполняемого решения по несоответствию для ранних
// модулей (AD-30): «как есть» — «годно по разрешению на отклонение» (FR-53).
func (s *State) executed(n NC, r kernel.Record) {
	if n.Disposition == "use_as_is" {
		s.effect(Effect{Kind: "set_quality", Value: string(statuses.QualityAcceptedWithConcession)}, r)
	}
	switch n.Disposition {
	case "rework", "repair", "use_as_is":
		s.releaseDecided(n, r)
	}
}

// releaseDecided — исполняемое решение по изделию (переделка, ремонт, как
// есть) выводит изделие из изоляции на исполнение решения (BPMN: подпроцесс
// брака «изоляция → ЗТ-Р → итог», выход из него снимает изоляцию): снимаются
// изоляция и сдерживание, основанное на решённых несоответствиях — этом и
// подтверждённых до решения (Covered). Разрешающее действие — подписанное
// решение людей (AD-27). Сдерживание по области инцидента и доп. проверки не
// трогаются: их основания живут своим порядком.
func (s *State) releaseDecided(n NC, r kernel.Record) {
	decided := map[string]bool{n.ID: true}
	for _, o := range s.NCs {
		if o.ID != n.ID && o.Status == StatusConfirmed && !s.confirmedAt(o).After(r.OccurredAt) {
			decided[o.ID] = true
		}
	}
	for i := range s.Containment {
		c := &s.Containment[i]
		if !c.Released && c.NCID != "" && decided[c.NCID] {
			c.Released, c.ReleasedBy = true, r.EventID
		}
	}
	if s.Isolation != nil && !s.Isolation.Released {
		s.Isolation.Released = true
		for i := range s.Containment {
			if c := &s.Containment[i]; c.Key == s.Isolation.EventID && !c.Released {
				c.Released, c.ReleasedBy = true, r.EventID
			}
		}
	}
}

// confirmedAt — когда несоответствие стало подтверждённым: решение
// контролёра или (регистрация правилом) момент регистрации.
func (s State) confirmedAt(n NC) time.Time {
	if n.ConfirmedEventID != "" {
		for _, d := range s.Decisions {
			if d.EventID == n.ConfirmedEventID {
				return d.At
			}
		}
	}
	return n.FoundAt
}

// Covered — подтверждённое несоответствие без своего решения покрыто
// исполняемым решением по изделию, принятым после его подтверждения (ось
// «решение по изделию» — одна на изделие, AD-30): групповое решение комиссии
// (NC-G1) решает судьбу изделия и по его собственным несоответствиям,
// подтверждённым раньше. Подтверждённое позже — ждёт нового решения.
func (s State) Covered(n NC) bool {
	if n.Status != StatusConfirmed {
		return false
	}
	at := s.confirmedAt(n)
	for _, o := range s.NCs {
		if o.ID == n.ID || o.Disposition == "" || !o.Executed || o.DispositionEventID == "" {
			continue
		}
		for _, d := range s.Decisions {
			if d.EventID == o.DispositionEventID && !d.At.Before(at) {
				return true
			}
		}
	}
	return false
}

func (s *State) decision(r kernel.Record, ncID, summary string) {
	s.Decisions = append(s.Decisions, DecisionRef{EventID: r.EventID, Seq: r.Seq, Type: string(r.Type), Actor: r.Actor,
		At: r.OccurredAt, NCID: ncID, Summary: summary})
}

func (s *State) effect(e Effect, r kernel.Record) {
	e.Cause, e.CauseAt = r.EventID, r.OccurredAt
	s.Effects = append(s.Effects, e)
}

func (s *State) addParticipant(p string) {
	if i, ok := slices.BinarySearch(s.Participants, p); !ok {
		s.Participants = slices.Insert(s.Participants, i, p)
	}
}

func (s *State) addSource(c ContainmentSource) {
	if slices.ContainsFunc(s.Containment, func(x ContainmentSource) bool { return x.Key == c.Key }) {
		return
	}
	if c.Basis == "" {
		c.Basis = c.Level
	}
	s.Containment = append(s.Containment, c)
}

// releaseNC — черновик закрыт отклонением всех его сигналов: сдерживание
// правилом по этому черновику снимает то же решение человека (AD-27).
func (s *State) releaseNC(ncID string) {
	for i := range s.Containment {
		c := &s.Containment[i]
		if c.NCID == ncID && c.By == ByRule && !c.Released {
			c.Released = true
		}
	}
}

// clone — копия состояния: свёртка чистая, прежнее значение не меняется.
func (s State) clone() State {
	out := s
	out.NCs = slices.Clone(s.NCs)
	out.Rejections = slices.Clone(s.Rejections)
	out.Containment = slices.Clone(s.Containment)
	if s.Isolation != nil {
		iso := *s.Isolation
		out.Isolation = &iso
	}
	out.Rechecks = slices.Clone(s.Rechecks)
	out.Presentations = slices.Clone(s.Presentations)
	out.Decisions = slices.Clone(s.Decisions)
	out.Participants = slices.Clone(s.Participants)
	out.Inspections = slices.Clone(s.Inspections)
	out.ReworkWaivers = maps.Clone(s.ReworkWaivers)
	out.Effects = slices.Clone(s.Effects)
	return out
}

func decode(r kernel.Record, v any) bool {
	return len(r.Data) > 0 && json.Unmarshal(r.Data, v) == nil
}

// ── Apply ──

// Apply применяет намерение, адресованное модулю nonconformity (Intent.Target
// == Module), в конце шага свёртки (AD-40): nonconformity — владелец своей оси
// и несоответствий и сам решает, как намерение меняет состояние.
// Неизвестное намерение — без изменений.
func Apply(s State, in kernel.Intent) State {
	switch in.Name {
	case IntentContain:
		c, ok := in.Payload.(Containment)
		if !ok || levelRank(string(c.Level)) == 0 {
			return s
		}
		s = s.clone()
		key := "intent:" + string(in.From) + ":" + joinIDs(in.Causes)
		s.addSource(ContainmentSource{Key: key, Level: string(c.Level), By: ByRule, Rule: RuleIntentContain, Reason: c.Reason, Causes: in.Causes, At: s.At})
	case IntentDraft:
		d, ok := draftFromRequest(in.Payload)
		if !ok || len(d.SignalIDs) == 0 {
			return s
		}
		s = s.clone()
		d.SignalIDs = slices.Compact(slices.Sorted(slices.Values(d.SignalIDs)))
		if d.NCID == "" {
			d.NCID = DraftNCID(s.ItemID, d.SignalIDs)
		}
		if s.nc(d.NCID) != nil {
			return s
		}
		// Сигнал, уже отклонённый контролёром, черновика не порождает.
		for _, sid := range d.SignalIDs {
			if s.SignalRejected(sid) {
				return s
			}
		}
		sev := d.Severity
		if sev == "" {
			sev = "unknown"
		}
		n := NC{
			ID: d.NCID, Number: NCNumber(d.NCID), Origin: OriginSignal, FoundAt: s.At, Causes: slices.Clone(in.Causes), From: string(in.From),
			Draft: d, Status: StatusDraft, Investigation: InvestigationNone, Severity: sev,
			DefectTypeCode: d.DefectTypeCode, RequirementRef: d.RequirementRef,
		}
		s.NCs = append(s.NCs, n)
		if d.ReactionOutcome == "isolate" {
			s.addSource(ContainmentSource{Key: "nc:" + d.NCID, Level: string(statuses.ContainmentItemHold), By: ByRule,
				Rule: RuleSignalContainment, NCID: d.NCID, Reason: "Карта реакций: изолировать", Causes: in.Causes, At: s.At})
		}
	}
	return s
}

func joinIDs(ids []string) string {
	out := ""
	for i, id := range ids {
		if i > 0 {
			out += ","
		}
		out += id
	}
	return out
}
