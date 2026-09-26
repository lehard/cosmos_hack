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

// ReceiveExtract — принять выписку паспорта партнёра (FR-132, AD-19): порт
// межзаводского обмена передаёт подписанный пакет; получатель сам проверяет
// подписи цепочкой к корням партнёра из нашего акта регистрации. Изменённая
// выписка отклоняется (422 federation.extract_tampered); непроверяемая
// принимается с пометкой «происхождение не подтверждено».
type ReceiveExtract struct {
	platform.CommandHeader
	PartnerCode string `json:"partner_code" pattern:"^[A-Z0-9][A-Z0-9_-]{0,15}$" doc:"От кого пришла выписка (канал партнёра)."`
	Envelope    string `json:"envelope" minLength:"2" maxLength:"262144" doc:"Подписанный пакет выписки — конверт DSSE (JSON) как пришёл."`
}
