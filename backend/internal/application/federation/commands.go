package federation

import "ant/internal/application/platform"

// RegisterPartner — акт регистрации партнёра и его корней (federation.partner.registered,
// AD-19): администратор безопасности + вторая подпись начальника ОТК.
type RegisterPartner struct {
	platform.CommandHeader
	PartnerCode      string   `json:"partner_code" pattern:"^[A-Z0-9][A-Z0-9_-]{0,15}$"`
	Name             string   `json:"name" minLength:"1" maxLength:"256"`
	RootFingerprints []string `json:"root_fingerprints" minItems:"1"`
	Endpoint         string   `json:"endpoint,omitempty" maxLength:"512"`
	DocumentID       string   `json:"document_id" doc:"Документ акта с маршрутом подписей."`
}

// SendExtract — отправить выписку паспорта партнёру (FR-131): исходящая выписка —
// документ с маршрутом «контролёр ОТК (2) + ключ шлюза предприятия» (AD-19);
// отправляет роль outbox по записи federation.message.sent.
type SendExtract struct {
	platform.CommandHeader
	PartnerCode string            `json:"partner_code"`
	Kind        string            `json:"kind" enum:"passport_extract,risk_notice,claim_notice"`
	Subject     platform.DrillRef `json:"subject" doc:"Партия или изделие."`
	DocumentID  string            `json:"document_id" doc:"Документ выписки с закрытым маршрутом."`
}
