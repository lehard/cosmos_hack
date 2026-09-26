package federation

import (
	"time"

	"ant/internal/application/platform"
)

// Формы ответов модуля federation (AD-19; FR-131…FR-134): партнёр — такая же
// копия ant; доверие ключам партнёра — от нашего акта регистрации партнёра.

// Partner — предприятие-партнёр и его корни доверия.
type Partner struct {
	PartnerCode      string    `json:"partner_code" doc:"Код предприятия-партнёра."`
	Name             string    `json:"name"`
	RootFingerprints []string  `json:"root_fingerprints" doc:"Отпечатки корней партнёра из нашего акта регистрации."`
	Endpoint         string    `json:"endpoint,omitempty" doc:"Адрес порта межзаводского обмена."`
	DocumentID       string    `json:"document_id" doc:"Акт регистрации партнёра (маршрут: администратор безопасности → начальник ОТК)."`
	RegisteredAt     time.Time `json:"registered_at"`
	Channel          string    `json:"channel" enum:"ok,degraded,unknown" doc:"Состояние канала обмена."`
}

// PartnerList — партнёры.
type PartnerList struct {
	Items []Partner `json:"items"`
}

// PassportExtract — выписка паспорта, входящая или исходящая (FR-131, FR-132).
type PassportExtract struct {
	ExtractDigest   string             `json:"extract_digest" doc:"Отпечаток выписки."`
	Direction       string             `json:"direction" enum:"incoming,outgoing"`
	PartnerCode     string             `json:"partner_code"`
	OriginStatus    string             `json:"origin_status" enum:"verified,server_confirmed_only,unverified,not_applicable" doc:"Происхождение подтверждено / подтверждено только сервером отправителя / не подтверждено (AD-19); у исходящей — not_applicable."`
	Subject         *platform.DrillRef `json:"subject,omitempty" doc:"Партия или изделие."`
	HeatNo          string             `json:"heat_no,omitempty" doc:"Плавка."`
	MaterialAddress string             `json:"material_address,omitempty" doc:"Исходные байты выписки в хранилище материалов."`
	DocumentID      string             `json:"document_id,omitempty" doc:"Исходящая выписка — документ с маршрутом (AD-19)."`
	MessageID       string             `json:"message_id,omitempty" doc:"Межзаводское сообщение."`
	Acknowledged    *bool              `nullable:"true" json:"acknowledged" doc:"Квитанция партнёра; null — ещё нет."`
	At              time.Time          `json:"at"`
}

// PassportExtractList — выписки.
type PassportExtractList struct {
	Items      []PassportExtract `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// PassportExtractView — выписка с содержимым: подписи класса partner и акты
// ключей подписантов (FR-133).
type PassportExtractView struct {
	Extract    PassportExtract    `json:"extract"`
	Content    map[string]any     `json:"content" doc:"Содержимое выписки."`
	Signatures []PartnerSignature `json:"signatures"`
	Checkpoint string             `json:"checkpoint,omitempty" doc:"Контрольная точка хранителя отправителя."`
}

// PartnerSignature — подпись внутри выписки, проверенная цепочкой к корням партнёра.
type PartnerSignature struct {
	KeyRef       string `json:"key_ref"`
	SignerRole   string `json:"signer_role,omitempty"`
	Verification string `json:"verification" enum:"valid,invalid,unverifiable"`
}
