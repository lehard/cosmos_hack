package nonconformity

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"
	"uuid"

	appjournal "ant/internal/application/journal"
	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
	dom "ant/internal/domain/nonconformity"
)

// Команды модуля nonconformity в режиме live (AD-39): состояние изделия —
// свёртка его входа из журнала (не проекция); доменный гард модуля-владельца
// операции; одна запись-решение с проверками потоков гарда и, для решений по
// разрешению на отклонение, атомарным расходом лимита. Классы (AD-27):
// подтвердить, изолировать, сдерживание, доп. проверка — защитные;
// отклонить сигнал, принять на точке, снять блок — разрешающие; решение по
// несоответствию — необратимое (режимы 3–5, маршрут подписей).

// reasonOf — основание из тела команды.
func reasonOf(r NCReason) dom.Reason {
	x := dom.Reason{Text: strings.TrimSpace(r.Text)}
	if r.Code != nil {
		x.Code = *r.Code
	}
	return x
}

func optReason(r *NCReason) *dom.Reason {
	if r == nil {
		return nil
	}
	x := reasonOf(*r)
	return &x
}

// itemCommand — решение над изделием: свёртка на «сейчас», гард, запись.
type itemCommand struct {
	Action string
	Type   catalog.Type
	Meta   platform.CommandMeta
	// Data — данные записи (они же Payload команды для гарда).
	Data any
	// Extra — дополнительные проверки (разрешение на отклонение) после гарда.
	Extra func(v *itemView, now time.Time) ([]appjournal.Check, error)
}

func (s *Service) onItem(ctx context.Context, v *itemView, c itemCommand) (platform.Receipt, error) {
	if r, ok, err := s.replayed(ctx, "item:"+v.ItemID, c.Type, c.Meta.CommandID); err != nil || ok {
		return r, err
	}
	now, err := s.now(ctx, v.RunID)
	if err != nil {
		return platform.Receipt{}, err
	}
	actor := platform.PrincipalFrom(ctx).PersonID
	stream := "item:" + v.ItemID
	basis := c.Meta.BasisSeq
	if basis <= 0 {
		basis = v.BasisSeq
	}
	cmd := kernel.Command{Action: c.Action, CommandID: c.Meta.CommandID, Actor: actor, Object: stream, BasisSeq: basis,
		PolicySeq: c.Meta.PolicySeq, GuardStreams: []string{stream}, OccurredAt: now, SignatureLevel: 2, Payload: c.Data}
	if err := dom.Guard(v.State(), v.Env, v.Upstream(), cmd); err != nil {
		return platform.Receipt{}, err
	}
	switch p := c.Data.(type) {
	case dom.PresentationResolvedData:
		if err := s.gateAuthority(v, actor, p.StepKey); err != nil {
			return platform.Receipt{}, err
		}
	case dom.PresentationReviewedData:
		// Д-81: пересматривает обладатель полномочия той же точки.
		if err := s.gateAuthority(v, actor, p.StepKey); err != nil {
			return platform.Receipt{}, err
		}
	}
	checks := []appjournal.Check{{Stream: stream, BasisSeq: basis}}
	if c.Extra != nil {
		more, err := c.Extra(v, now)
		if err != nil {
			return platform.Receipt{}, err
		}
		checks = append(checks, more...)
	}
	meta := c.Meta
	meta.BasisSeq = basis
	r, err := s.write(ctx, decision{Type: c.Type, Stream: stream, ItemID: v.ItemID, RunID: v.RunID, Data: c.Data, Meta: meta,
		Actor: actor, OccurredAt: now, Checks: checks, SignatureLevel: 2, GuardStreams: []string{stream}})
	return r, err
}

func (s *Service) item(ctx context.Context, itemID string, c itemCommand) (platform.Receipt, error) {
	v, err := s.loadItem(ctx, itemID, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	return s.onItem(ctx, v, c)
}

func (s *Service) nc(ctx context.Context, ncID string, c itemCommand) (platform.Receipt, error) {
	v, _, err := s.loadNC(ctx, ncID, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	return s.onItem(ctx, v, c)
}

// useConcession — применимость разрешения к изделию (гард книги разрешений)
// и проверки AD-39: поток разрешения не менялся (отзыв) и расход 1 из лимита
// атомарно с решением.
func (s *Service) useConcession(ctx context.Context, id, itemID, kind string, now time.Time) ([]appjournal.Check, error) {
	book, err := s.concessionBook(ctx, platform.Moment{})
	if err != nil {
		return nil, err
	}
	c := book.Items[id]
	if err := dom.ConcessionGuard(c, id, itemID, kind, now); err != nil {
		return nil, err
	}
	return []appjournal.Check{
		{Stream: "concession:" + id, BasisSeq: c.StreamSeq},
		{ConcessionID: id, Consume: 1},
	}, nil
}

// Confirm — подтвердить несоответствие (FR-52): защитное, критическое.
func (s *Service) Confirm(ctx context.Context, ncID string, in ConfirmNonconformity) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.Confirm(ctx, ncID, in)
	}
	return s.nc(ctx, ncID, itemCommand{Action: dom.ActConfirm, Type: catalog.DecisionNonconformityConfirmed, Meta: in.CommandMeta(),
		Data: dom.ConfirmedData{NCID: ncID, SignalIDs: in.SignalIDs, RequirementRef: in.RequirementRef, Severity: in.Severity,
			DefectTypeCode: in.DefectTypeCode, FullAnalysisRequired: in.FullAnalysisRequired, Reason: reasonOf(in.Reason)}})
}

// RejectSignal — отклонить сигнал с обязательной причиной (FR-52): исходный
// сигнал не меняется — решение отдельной записью (FR-51).
func (s *Service) RejectSignal(ctx context.Context, itemID string, in RejectSignal) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RejectSignal(ctx, itemID, in)
	}
	return s.item(ctx, itemID, itemCommand{Action: dom.ActRejectSignal, Type: catalog.DecisionSignalRejected, Meta: in.CommandMeta(),
		Data: dom.SignalRejectedData{SignalIDs: in.SignalIDs, Reason: reasonOf(in.Reason), LabelForAdaptation: in.LabelForAdaptation}})
}

// RequestRecheck — назначить доп. проверку (FR-52): изделие ждёт, не
// маршрутизируется автоматически.
func (s *Service) RequestRecheck(ctx context.Context, itemID string, in RequestRecheck) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RequestRecheck(ctx, itemID, in)
	}
	return s.item(ctx, itemID, itemCommand{Action: dom.ActRecheck, Type: catalog.DecisionRecheckRequested, Meta: in.CommandMeta(),
		Data: dom.RecheckData{Method: in.Method, ZoneIDs: in.ZoneIDs, DueAt: in.DueAt, Reason: reasonOf(in.Reason)}})
}

// Isolate — изолировать со сроком решения по производственному календарю
// (FR-55: по умолчанию 3 рабочих дня).
func (s *Service) Isolate(ctx context.Context, itemID string, in IsolateItem) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.Isolate(ctx, itemID, in)
	}
	v, err := s.loadItem(ctx, itemID, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	due := in.DecisionDueAt
	if due == "" {
		now, err := s.now(ctx, v.RunID)
		if err != nil {
			return platform.Receipt{}, err
		}
		due = fmtTime(s.d.Calendar.AddWorkingDays(now, s.cfg.IsolationWorkingDays))
	}
	return s.onItem(ctx, v, itemCommand{Action: dom.ActIsolate, Type: catalog.DecisionItemIsolated, Meta: in.CommandMeta(),
		Data: dom.IsolatedData{IsolatorLocationID: in.IsolatorLocationID, DecisionDueAt: due, Reason: reasonOf(in.Reason)}})
}

// ResolvePresentation — решение на точке предъявления (FR-19, FR-56):
// «Принять — передать дальше», принять по разрешению, вернуть, мало данных.
func (s *Service) ResolvePresentation(ctx context.Context, itemID string, in ResolvePresentation) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ResolvePresentation(ctx, itemID, in)
	}
	data := dom.PresentationResolvedData{StepKey: in.StepKey, ClosingPoint: in.ClosingPoint, Resolution: in.Resolution,
		PresentationNo: in.PresentationNo, ConcessionID: in.ConcessionID, MethodEventIDs: in.MethodEventIDs, Reason: optReason(in.Reason)}
	c := itemCommand{Action: dom.ActPresentation, Type: catalog.DecisionPresentationResolved, Meta: in.CommandMeta(), Data: data}
	if in.Resolution == "accept_with_concession" && in.ConcessionID != "" {
		c.Extra = func(v *itemView, now time.Time) ([]appjournal.Check, error) {
			return s.useConcession(ctx, in.ConcessionID, v.ItemID, "", now)
		}
	}
	return s.item(ctx, itemID, c)
}

// ReviewPresentation — пересмотр решения на точке, принятого до новых данных
// (FR-32, FR-146, Д-81): новая запись decision.presentation.reviewed поверх
// прежней. Точку, номер предъявления и шаг берёт из пересматриваемой записи;
// рассмотренные факты по умолчанию — новые факты открытого пересмотра.
func (s *Service) ReviewPresentation(ctx context.Context, itemID string, in ReviewPresentation) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ReviewPresentation(ctx, itemID, in)
	}
	v, err := s.loadItem(ctx, itemID, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	data := dom.PresentationReviewedData{ReviewedEventID: strings.ToLower(in.ReviewedEventID), Outcome: in.Outcome,
		NewFactIDs: slices.Clone(in.NewFactIDs), Reason: reasonOf(in.Reason)}
	if r, ok := v.Record(data.ReviewedEventID); ok && r.Type == catalog.DecisionPresentationResolved {
		var d dom.PresentationResolvedData
		if json.Unmarshal(r.Data, &d) == nil {
			data.StepKey, data.ClosingPoint, data.PresentationNo = d.StepKey, d.ClosingPoint, max(d.PresentationNo, 1)
		}
	}
	if len(data.NewFactIDs) == 0 {
		for _, rv := range s.reviewsOf(v) {
			if rv.decision.EventID == data.ReviewedEventID {
				for _, f := range rv.facts {
					data.NewFactIDs = append(data.NewFactIDs, f.EventID)
				}
			}
		}
	}
	if data.NewFactIDs == nil {
		data.NewFactIDs = []string{}
	}
	slices.Sort(data.NewFactIDs)
	data.NewFactIDs = slices.Compact(data.NewFactIDs)
	return s.onItem(ctx, v, itemCommand{Action: dom.ActPresentationReview, Type: catalog.DecisionPresentationReviewed, Meta: in.CommandMeta(), Data: data})
}

// ResolveLot — решение по партии входного контроля (ЗТ-1), поток lot:‹id›.
func (s *Service) ResolveLot(ctx context.Context, lotID string, in ResolveLot) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ResolveLot(ctx, lotID, in)
	}
	now, err := s.now(ctx, "")
	if err != nil {
		return platform.Receipt{}, err
	}
	stream := "lot:" + lotID
	meta := in.CommandMeta()
	var checks []appjournal.Check
	if meta.BasisSeq > 0 {
		checks = append(checks, appjournal.Check{Stream: stream, BasisSeq: meta.BasisSeq})
	}
	return s.write(ctx, decision{Type: catalog.DecisionLotResolved, Stream: stream, Meta: meta, Actor: platform.PrincipalFrom(ctx).PersonID,
		OccurredAt: now, Checks: checks, SignatureLevel: 2, GuardStreams: []string{stream},
		Data: dom.LotResolvedData{LotID: lotID, Resolution: in.Resolution, AcceptedQuantity: in.AcceptedQuantity,
			MethodEventIDs: in.MethodEventIDs, Reason: optReason(in.Reason)}})
}

// SetDisposition — решение по несоответствию (FR-53): переделка / ремонт /
// как есть / списать / вернуть поставщику. Ремонт и «как есть» — только по
// действующему разрешению на отклонение, расход лимита — атомарно с
// решением (FR-54, AD-39); исполнение — после закрытия маршрута подписей
// (режим 4, AD-43): состояние подписей даёт порт RouteGate.
func (s *Service) SetDisposition(ctx context.Context, ncID string, in SetDisposition) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.SetDisposition(ctx, ncID, in)
	}
	docID := "ncd-" + ncID
	approvals, err := s.d.Routes.Approvals(ctx, dom.ActDisposition, docID)
	if err != nil {
		return platform.Receipt{}, err
	}
	data := dom.DispositionSetData{NCID: ncID, Disposition: in.Disposition, ScrapKind: in.ScrapKind, ConcessionID: in.ConcessionID,
		DocumentID: docID, ClaimBasis: in.ClaimBasis, ApprovalsStatus: approvals, Reason: reasonOf(in.Reason)}
	c := itemCommand{Action: dom.ActDisposition, Type: catalog.DecisionDispositionSet, Meta: in.CommandMeta(), Data: data}
	if in.ConcessionID != "" && (in.Disposition == "repair" || in.Disposition == "use_as_is") {
		c.Extra = func(v *itemView, now time.Time) ([]appjournal.Check, error) {
			return s.useConcession(ctx, in.ConcessionID, v.ItemID, in.Disposition, now)
		}
	}
	return s.group(ctx, ncID, c, nil)
}

// VerifyDisposition — подтвердить выполнение решения повторной проверкой.
// Групповое несоответствие: проверка относится к изделиям, чьи результаты
// повторного контроля в ней указаны (у каждого изделия — свои).
func (s *Service) VerifyDisposition(ctx context.Context, ncID string, in VerifyDisposition) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.VerifyDisposition(ctx, ncID, in)
	}
	c := itemCommand{Action: dom.ActVerify, Type: catalog.DecisionDispositionVerified, Meta: in.CommandMeta(),
		Data: dom.DispositionVerifiedData{NCID: ncID, RecheckEventIDs: in.RecheckEventIDs}}
	var pick func(v *itemView) bool
	if len(in.RecheckEventIDs) > 0 {
		pick = func(v *itemView) bool {
			return slices.ContainsFunc(in.RecheckEventIDs, func(id string) bool { return slices.Contains(v.State().Inspections, id) })
		}
	}
	return s.group(ctx, ncID, c, pick)
}

// Close — закрыть несоответствие по изделию (системное расследование не закрывается, FR-51).
func (s *Service) Close(ctx context.Context, ncID string, in CloseNonconformity) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.Close(ctx, ncID, in)
	}
	return s.group(ctx, ncID, itemCommand{Action: dom.ActClose, Type: catalog.DecisionNonconformityClosed, Meta: in.CommandMeta(),
		Data: dom.ClosedData{NCID: ncID, Summary: in.Summary}}, nil)
}

// SetContainment — установить уровень сдерживания (FR-49): защитное.
func (s *Service) SetContainment(ctx context.Context, itemID string, in SetContainment) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.SetContainment(ctx, itemID, in)
	}
	return s.item(ctx, itemID, itemCommand{Action: dom.ActContainmentSet, Type: catalog.DecisionContainmentSet, Meta: in.CommandMeta(),
		Data: dom.ContainmentSetData{Level: in.Level, Reason: reasonOf(in.Reason)}})
}

// ReleaseContainment — снять сдерживание (AD-27: разрешающее, только
// человек; снятие блока ≠ годность).
func (s *Service) ReleaseContainment(ctx context.Context, itemID string, in ReleaseContainment) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ReleaseContainment(ctx, itemID, in)
	}
	return s.item(ctx, itemID, itemCommand{Action: dom.ActContainmentRelease, Type: catalog.DecisionContainmentReleased, Meta: in.CommandMeta(),
		Data: dom.ContainmentReleasedData{ReleasedEventIDs: in.ReleasedEventIDs, Reason: reasonOf(in.Reason)}})
}

// WaiveReworkLimit — разрешение сверх лимита доработок (FR-18).
func (s *Service) WaiveReworkLimit(ctx context.Context, itemID string, in WaiveReworkLimit) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.WaiveReworkLimit(ctx, itemID, in)
	}
	return s.item(ctx, itemID, itemCommand{Action: dom.ActReworkWaive, Type: catalog.DecisionReworkLimitWaived, Meta: in.CommandMeta(),
		Data: dom.ReworkWaivedData{ZoneID: in.ZoneID, Used: in.Used, Limit: in.Limit, ExtraAllowed: in.ExtraAllowed, Reason: reasonOf(in.Reason)}})
}

// GrantConcession — выдать разрешение на отклонение (FR-54, Д-24): лимит
// открывается атомарно с записью выдачи (journal.Append: ConcessionGrants).
func (s *Service) GrantConcession(ctx context.Context, in GrantConcession) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.GrantConcession(ctx, in)
	}
	id := strings.TrimSpace(in.ConcessionID)
	if id == "" {
		id = "CON-" + shortID(in.CommandID)
	}
	if r, ok, err := s.replayed(ctx, "concession:"+id, catalog.DecisionConcessionGranted, in.CommandID); err != nil || ok {
		return r, err
	}
	book, err := s.concessionBook(ctx, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	approvals, err := s.d.Routes.Approvals(ctx, dom.ActConcessionGrant, in.DocumentID)
	if err != nil {
		return platform.Receipt{}, err
	}
	data := dom.ConcessionGrantedData{ConcessionID: id, Number: in.Number, Title: in.Title, Kind: in.Kind, RequirementRef: in.RequirementRef,
		ScopeItemIDs: in.ScopeItemIDs, ScopeRangeFrom: in.ScopeRangeFrom, ScopeRangeTo: in.ScopeRangeTo, Limit: in.Limit,
		ValidUntil: in.ValidUntil, DocumentID: in.DocumentID, ApprovalsStatus: approvals, Reason: reasonOf(in.Reason)}
	if data.ValidUntil != "" {
		t, err := time.Parse(time.RFC3339Nano, data.ValidUntil)
		if err != nil {
			return platform.Receipt{}, platform.Fail(errcodes.ApiValidationFailed, "field", "valid_until", "reason", err.Error())
		}
		data.ValidUntil = fmtTime(t)
	}
	if err := dom.GrantGuard(book, data); err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx, "")
	if err != nil {
		return platform.Receipt{}, err
	}
	stream := "concession:" + id
	return s.write(ctx, decision{Type: catalog.DecisionConcessionGranted, Stream: stream, Data: data, Meta: in.CommandMeta(),
		Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now, SignatureLevel: 2, GuardStreams: []string{stream},
		Checks: []appjournal.Check{{Stream: stream, BasisSeq: 0}},
		Grants: []appjournal.ConcessionGrant{{ConcessionID: id, Limit: int64(in.Limit)}}})
}

// RevokeConcession — отозвать разрешение на отклонение (FR-54): расход
// прекращается; решения, принятые по нему, остаются в журнале.
func (s *Service) RevokeConcession(ctx context.Context, concessionID string, in RevokeConcession) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RevokeConcession(ctx, concessionID, in)
	}
	if r, ok, err := s.replayed(ctx, "concession:"+concessionID, catalog.DecisionConcessionRevoked, in.CommandID); err != nil || ok {
		return r, err
	}
	book, err := s.concessionBook(ctx, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := dom.RevokeGuard(book, concessionID); err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx, "")
	if err != nil {
		return platform.Receipt{}, err
	}
	stream := "concession:" + concessionID
	meta := in.CommandMeta()
	basis := meta.BasisSeq
	if basis <= 0 {
		basis = book.Items[concessionID].StreamSeq
	}
	meta.BasisSeq = basis
	return s.write(ctx, decision{Type: catalog.DecisionConcessionRevoked, Stream: stream, Meta: meta,
		Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now, SignatureLevel: 2, GuardStreams: []string{stream},
		Checks: []appjournal.Check{{Stream: stream, BasisSeq: basis}},
		Data:   dom.ConcessionRevokedData{ConcessionID: concessionID, Reason: reasonOf(in.Reason)}})
}

// SetProcessHold — стоп точки процесса или критическая остановка (FR-49):
// сдерживание процесса — отдельный объект, поток equipment:‹точка›.
func (s *Service) SetProcessHold(ctx context.Context, in SetProcessHold) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.SetProcessHold(ctx, in)
	}
	id := strings.TrimSpace(in.HoldID)
	if id == "" {
		id = "HOLD-" + shortID(in.CommandID)
	}
	data := dom.ProcessHoldSetData{HoldID: id, Level: in.Level, EquipmentID: in.EquipmentID, ToolID: in.ToolID, ProgramRef: in.ProgramRef,
		StepKey: in.StepKey, IncidentID: in.IncidentID, ReleaseCondition: in.ReleaseCondition, Reason: reasonOf(in.Reason)}
	if st := dom.HoldStream(data); st != "" {
		if r, ok, err := s.replayed(ctx, st, catalog.DecisionProcessHoldSet, in.CommandID); err != nil || ok {
			return r, err
		}
	}
	book, err := s.holdBook(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	if err := dom.HoldGuard(book, kernel.Command{Action: dom.ActProcessHoldSet, Payload: data}); err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx, "")
	if err != nil {
		return platform.Receipt{}, err
	}
	stream := dom.HoldStream(data)
	return s.write(ctx, decision{Type: catalog.DecisionProcessHoldSet, Stream: stream, Data: data, Meta: in.CommandMeta(),
		Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now, SignatureLevel: 2, GuardStreams: []string{stream}})
}

// ReleaseProcessHold — снять остановку точки процесса (FR-49): разрешающее,
// только уполномоченный; дальше — точка чистоты на N изделий.
func (s *Service) ReleaseProcessHold(ctx context.Context, holdID string, in ReleaseProcessHold) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ReleaseProcessHold(ctx, holdID, in)
	}
	book, err := s.holdBook(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	data := dom.ProcessHoldReleasedData{HoldID: holdID, CleanPointItems: in.CleanPointItems, Reason: reasonOf(in.Reason)}
	if h, ok := book.Holds[holdID]; ok {
		if r, ok, err := s.replayed(ctx, h.Stream, catalog.DecisionProcessHoldReleased, in.CommandID); err != nil || ok {
			return r, err
		}
	}
	if err := dom.HoldGuard(book, kernel.Command{Action: dom.ActProcessHoldRelease, Payload: data}); err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx, "")
	if err != nil {
		return platform.Receipt{}, err
	}
	stream := book.Holds[holdID].Stream
	meta := in.CommandMeta()
	var checks []appjournal.Check
	if meta.BasisSeq > 0 {
		checks = append(checks, appjournal.Check{Stream: stream, BasisSeq: meta.BasisSeq})
	}
	return s.write(ctx, decision{Type: catalog.DecisionProcessHoldReleased, Stream: stream, Data: data, Meta: meta,
		Actor: platform.PrincipalFrom(ctx).PersonID, OccurredAt: now, Checks: checks, SignatureLevel: 2, GuardStreams: []string{stream}})
}

// shortID — короткий id из UUID команды (или нового UUIDv7).
func shortID(commandID string) string {
	id := strings.ReplaceAll(strings.ToLower(commandID), "-", "")
	if len(id) < 12 {
		id = strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	}
	return strings.ToUpper(id[len(id)-12:])
}

// fmtTime — время по соглашению (RFC 3339 UTC, три знака после секунд).
func fmtTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// gateAuthority — решение на точке предъявления принимает обладатель
// полномочия точки: process State.Gates[step].Authority (FR-19: повторное
// предъявление — полномочие выше). Стык эпиков 17 и 21 (эпик 16).
func (s *Service) gateAuthority(v *itemView, actor, stepKey string) error {
	g, ok := v.Snap.Process.Gate(stepKey)
	if !ok || g.Authority == "" || actor == "" || s.d.Authorities == nil {
		return nil
	}
	if s.d.Authorities.HasAuthority(actor, g.Authority) {
		return nil
	}
	return kernel.Refuse(errcodes.AccessSignatureRequired, "who", "обладатель полномочия «"+g.Authority+"»")
}
