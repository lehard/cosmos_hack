package vision

import (
	"context"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	ev "ant/internal/contracts/events"
	dom "ant/internal/domain/vision"
)

func reason(r AnalyzerReason) ev.Reason {
	out := ev.Reason{Text: ev.Text(r.Text)}
	if r.Code != nil && *r.Code != "" {
		c := ev.Code(*r.Code)
		out.Code = &c
	}
	return out
}

func versionsOf(m map[string]string) dom.Versions {
	return dom.Versions{ItemRevision: m["item_revision"], RecipeRef: m["recipe_ref"], CameraConfig: m["camera_config"],
		Calibration: m["calibration"], AnalyzerVersion: m["analyzer_version"], ThresholdProfile: m["threshold_profile"],
		ContractVersion: m["contract_version"], AppVersion: m["app_version"]}
}

// AdmitPassport — допуск версии анализатора (vision.passport.admit, FR-98):
// гард над реестром паспортов, протокол допуска с закрытым маршрутом (AD-43;
// в demo и fixtures — заглушка demo_stub), запись analyzer.passport.admitted.
// Экзамен, тень и пилот перед допуском — эпик 40.
func (s *Service) AdmitPassport(ctx context.Context, in AdmitPassport) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.AdmitPassport(ctx, in)
	}
	meta := in.CommandMeta()
	if r, ok, err := s.replayed(ctx, dom.Stream(in.PassportID), catalog.AnalyzerPassportAdmitted, meta); err != nil || ok {
		return r, err
	}
	reg, err := s.registry(ctx, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	kind := in.AnalyzerKind
	if kind == "" {
		kind = dom.KindVisionQC
	}
	a := dom.Admission{PassportID: in.PassportID, AnalyzerID: in.AnalyzerID, AnalyzerKind: kind, Stage: in.Stage,
		TrustLevel: in.TrustLevel, RecipeRef: in.RecipeRef, Versions: versionsOf(in.Versions),
		PreviousPassportID: in.PreviousPassportID, DocumentID: in.DocumentID}
	if err := dom.GuardAdmit(reg, a); err != nil {
		return platform.Receipt{}, err
	}
	st, err := s.d.Routes.Approvals(ctx, dom.ActionAdmit, in.DocumentID)
	if err != nil {
		return platform.Receipt{}, err
	}
	if st != ApprovalsClosed && st != ApprovalsDemoStub {
		return platform.Receipt{}, platform.Fail(errcodes.AnalyzerAdmissionRouteOpen, "document_id", in.DocumentID)
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	d := ev.AnalyzerPassportAdmittedV1{PassportID: ev.ObjectID(a.PassportID), Stage: ev.AnalyzerPassportAdmittedV1Stage(a.Stage),
		TrustLevel: a.TrustLevel, RecipeRef: a.RecipeRef, Versions: a.Versions.Contract(), DocumentID: ev.ObjectID(a.DocumentID),
		PreviousPassportID: oid(a.PreviousPassportID), AnalyzerID: oid(a.AnalyzerID), Title: sp(in.Title)}
	k := ev.AnalyzerPassportAdmittedV1AnalyzerKind(kind)
	d.AnalyzerKind = &k
	return s.write(ctx, catalog.AnalyzerPassportAdmitted, dom.Stream(a.PassportID), d, meta, now, "")
}

// ReinstatePassport — вернуть анализатор после отката (FR-101): только
// начальник ОТК и только из действующей приостановки (AD-27).
func (s *Service) ReinstatePassport(ctx context.Context, passportID string, in ReinstatePassport) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.ReinstatePassport(ctx, passportID, in)
	}
	if r, ok, err := s.replayed(ctx, dom.Stream(passportID), catalog.AnalyzerPassportReinstated, in.CommandMeta()); err != nil || ok {
		return r, err
	}
	reg, err := s.registry(ctx, platform.Moment{RunID: in.RunID})
	if err != nil {
		return platform.Receipt{}, err
	}
	principal := platform.PrincipalFrom(ctx)
	p, err := dom.GuardReinstate(reg, passportID, in.SuspensionEventID, principal.Role, principal.HasRole(dom.RoleHeadOfQC))
	if err != nil {
		return platform.Receipt{}, err
	}
	// Возврат действует там же, где приостановка: в прогоне сценария — в нём (AD-38).
	susp, _ := p.LastSuspension()
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	d := ev.AnalyzerPassportReinstatedV1{PassportID: ev.ObjectID(passportID), SuspensionEventID: ev.UUID(in.SuspensionEventID), Reason: reason(in.Reason)}
	return s.write(ctx, catalog.AnalyzerPassportReinstated, dom.Stream(passportID), d, in.CommandMeta(), now, susp.RunID)
}

// RetirePassport — вывести паспорт из действия; старые наблюдения сохраняют
// «проанализировано версией …» (FR-98).
func (s *Service) RetirePassport(ctx context.Context, passportID string, in RetirePassport) (platform.Receipt, error) {
	if !s.live() {
		return s.Unimplemented.RetirePassport(ctx, passportID, in)
	}
	if r, ok, err := s.replayed(ctx, dom.Stream(passportID), catalog.AnalyzerPassportRetired, in.CommandMeta()); err != nil || ok {
		return r, err
	}
	reg, err := s.registry(ctx, platform.Moment{})
	if err != nil {
		return platform.Receipt{}, err
	}
	if _, err := dom.GuardRetire(reg, passportID); err != nil {
		return platform.Receipt{}, err
	}
	now, err := s.now(ctx)
	if err != nil {
		return platform.Receipt{}, err
	}
	d := ev.AnalyzerPassportRetiredV1{PassportID: ev.ObjectID(passportID), Reason: reason(in.Reason)}
	return s.write(ctx, catalog.AnalyzerPassportRetired, dom.Stream(passportID), d, in.CommandMeta(), now, "")
}
