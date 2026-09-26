package access

import (
	"context"

	"ant/internal/application/platform"
)

// AdminQueries — чтение администрирования доступа (FR-78…FR-85, AD-15); входит в Queries.
type AdminQueries interface {
	// Persons — сотрудники с ролями и учётными записями (FR-78) (access.person.list).
	Persons(ctx context.Context, m platform.Moment, p platform.Page) (AccessPersonList, error)
	// Person — сотрудник (access.person.read).
	Person(ctx context.Context, personID string, m platform.Moment) (AccessPerson, error)
	// Roles — роли и полномочия политики (AD-15) (access.role.list).
	Roles(ctx context.Context, m platform.Moment) (AccessRoleList, error)
	// Grants — история выдачи прав (Аудитор ИБ) (access.grant.list).
	Grants(ctx context.Context, personID string, m platform.Moment, p platform.Page) (AccessGrantHistory, error)
	// Stamps — цифровые клейма (FR-145) (access.stamp.list).
	Stamps(ctx context.Context, personID string, m platform.Moment) (AccessStampList, error)
	// Assignments — назначения на посты в смене (FR-81) (access.assignment.list).
	Assignments(ctx context.Context, shiftID, workshop string, m platform.Moment) (AccessAssignmentList, error)
	// Qualifications — квалификации (FR-80) (access.qualification.list).
	Qualifications(ctx context.Context, personID string, m platform.Moment) (AccessQualificationList, error)
	// Audit — параметры аудита policy.audit.* (access.audit.read).
	Audit(ctx context.Context, m platform.Moment) (AccessAuditParameters, error)
	// AssessGrant — чья подпись нужна для выдачи (access.grant.assess, эпик 26).
	AssessGrant(ctx context.Context, q GrantQuery) (GrantAssessment, error)
}

// AdminCommands — команды администрирования доступа и терминала исполнителя; входят в Commands.
type AdminCommands interface {
	// RegisterPerson — access.person.register.
	RegisterPerson(ctx context.Context, in RegisterPerson) (platform.Receipt, error)
	// ActivateAccount — access.account.activate.
	ActivateAccount(ctx context.Context, personID string, in ActivateAccount) (platform.Receipt, error)
	// GrantPolicy — access.policy.grant.
	GrantPolicy(ctx context.Context, in GrantPolicy) (platform.Receipt, error)
	// RevokePolicy — access.policy.revoke.
	RevokePolicy(ctx context.Context, in RevokePolicy) (platform.Receipt, error)
	// SetAssignment — access.assignment.set.
	SetAssignment(ctx context.Context, in SetAssignment) (platform.Receipt, error)
	// ClearAssignment — access.assignment.clear.
	ClearAssignment(ctx context.Context, in ClearAssignment) (platform.Receipt, error)
	// GrantQualification — access.qualification.grant.
	GrantQualification(ctx context.Context, personID string, in GrantQualification) (platform.Receipt, error)
	// RevokeQualification — access.qualification.revoke.
	RevokeQualification(ctx context.Context, personID string, in RevokeQualification) (platform.Receipt, error)
	// SetAuditParameters — access.audit.set_parameters.
	SetAuditParameters(ctx context.Context, in SetAuditParameters) (platform.Receipt, error)
	// AdmitWorkplace — access.workplace.admit.
	AdmitWorkplace(ctx context.Context, workplaceID string, in AdmitWorkplace) (platform.Receipt, error)
	// ReleaseWorkplace — access.workplace.release.
	ReleaseWorkplace(ctx context.Context, workplaceID string, in ReleaseWorkplace) (platform.Receipt, error)
	// ConfirmStep — access.operator.confirm_step.
	ConfirmStep(ctx context.Context, workplaceID string, in ConfirmStep) (platform.Receipt, error)
	// ReportDeviation — access.operator.report_deviation.
	ReportDeviation(ctx context.Context, workplaceID string, in ReportDeviation) (platform.Receipt, error)
	// RequestInspection — access.operator.request_inspection.
	RequestInspection(ctx context.Context, workplaceID string, in RequestInspection) (platform.Receipt, error)
}

func (Unimplemented) Persons(context.Context, platform.Moment, platform.Page) (AccessPersonList, error) {
	return AccessPersonList{}, platform.NotImplemented("access.person.list")
}

func (Unimplemented) Person(context.Context, string, platform.Moment) (AccessPerson, error) {
	return AccessPerson{}, platform.NotImplemented("access.person.read")
}

func (Unimplemented) Roles(context.Context, platform.Moment) (AccessRoleList, error) {
	return AccessRoleList{}, platform.NotImplemented("access.role.list")
}

func (Unimplemented) Grants(context.Context, string, platform.Moment, platform.Page) (AccessGrantHistory, error) {
	return AccessGrantHistory{}, platform.NotImplemented("access.grant.list")
}

func (Unimplemented) Stamps(context.Context, string, platform.Moment) (AccessStampList, error) {
	return AccessStampList{}, platform.NotImplemented("access.stamp.list")
}

func (Unimplemented) Assignments(context.Context, string, string, platform.Moment) (AccessAssignmentList, error) {
	return AccessAssignmentList{}, platform.NotImplemented("access.assignment.list")
}

func (Unimplemented) Qualifications(context.Context, string, platform.Moment) (AccessQualificationList, error) {
	return AccessQualificationList{}, platform.NotImplemented("access.qualification.list")
}

func (Unimplemented) Audit(context.Context, platform.Moment) (AccessAuditParameters, error) {
	return AccessAuditParameters{}, platform.NotImplemented("access.audit.read")
}

func (Unimplemented) AssessGrant(context.Context, GrantQuery) (GrantAssessment, error) {
	return GrantAssessment{}, platform.NotImplemented("access.grant.assess")
}

func (Unimplemented) RegisterPerson(context.Context, RegisterPerson) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.person.register")
}

func (Unimplemented) ActivateAccount(context.Context, string, ActivateAccount) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.account.activate")
}

func (Unimplemented) GrantPolicy(context.Context, GrantPolicy) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.policy.grant")
}

func (Unimplemented) RevokePolicy(context.Context, RevokePolicy) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.policy.revoke")
}

func (Unimplemented) SetAssignment(context.Context, SetAssignment) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.assignment.set")
}

func (Unimplemented) ClearAssignment(context.Context, ClearAssignment) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.assignment.clear")
}

func (Unimplemented) GrantQualification(context.Context, string, GrantQualification) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.qualification.grant")
}

func (Unimplemented) RevokeQualification(context.Context, string, RevokeQualification) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.qualification.revoke")
}

func (Unimplemented) SetAuditParameters(context.Context, SetAuditParameters) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.audit.set_parameters")
}

func (Unimplemented) AdmitWorkplace(context.Context, string, AdmitWorkplace) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.workplace.admit")
}

func (Unimplemented) ReleaseWorkplace(context.Context, string, ReleaseWorkplace) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.workplace.release")
}

func (Unimplemented) ConfirmStep(context.Context, string, ConfirmStep) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.operator.confirm_step")
}

func (Unimplemented) ReportDeviation(context.Context, string, ReportDeviation) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.operator.report_deviation")
}

func (Unimplemented) RequestInspection(context.Context, string, RequestInspection) (platform.Receipt, error) {
	return platform.Receipt{}, platform.NotImplemented("access.operator.request_inspection")
}
