package documents

import (
	"time"

	"ant/internal/application/platform"
)

// Формы ответов модуля documents (AD-12, AD-13, AD-43; FR-65, FR-136, FR-139).
// Документ — детерминированная проекция журнала: шаблон@версия × события-источники;
// не редактируется, новая версия — новый отпечаток.

// DocumentApprovalStage — этап маршрута подписей, вычисленный один раз при
// document.version.drafted функцией access.RequiredApprovals (AD-43).
type DocumentApprovalStage struct {
	Stage          int      `json:"stage" minimum:"1" doc:"Номер этапа по порядку."`
	Authority      string   `json:"authority" doc:"Полномочие этапа."`
	StampKind      string   `json:"stamp_kind,omitempty" doc:"Вид цифрового клейма (для действий контроля, FR-145)."`
	Quorum         string   `json:"quorum" enum:"one,all,k_of_n" doc:"Сколько подписей: 1 | все | k из n."`
	Required       int      `json:"required" minimum:"1" doc:"Сколько подписей нужно на этапе."`
	Level          int      `json:"level" minimum:"0" maximum:"3" doc:"Уровень подписи (AD-13)."`
	PaperAllowed   bool     `json:"paper_allowed" doc:"Бумага с заверением допустима на этапе (AD-43)."`
	ExternalParty  string   `json:"external_party,omitempty" doc:"Внешняя сторона: ВП, партнёр."`
	Candidates     []string `json:"candidates" doc:"Кто может подписать этап (псевдонимы)."`
	SignedBy       []string `json:"signed_by" doc:"Кто уже подписал (по текущему отпечатку)."`
	Status         string   `json:"status" enum:"pending,in_progress,done" doc:"Состояние этапа."`
	SeparationNote string   `json:"separation_note,omitempty" doc:"Разделение обязанностей на этапе (FR-56)."`
}

// DocumentSignatureView — подпись документа со статусом проверки (FR-68, AD-43).
type DocumentSignatureView struct {
	EventID         string    `json:"event_id" doc:"Запись document.signature.recorded."`
	Stage           int       `json:"stage" minimum:"1"`
	Method          string    `json:"method" enum:"token_agent,paper,device,demo_signer"`
	SignerPersonID  string    `json:"signer_person_id"`
	AttestedBy      string    `json:"attested_by,omitempty" doc:"Заверитель бумажной подписи (AD-43)."`
	Level           int       `json:"level" minimum:"0" maximum:"3"`
	SignedAt        time.Time `json:"signed_at"`
	DocDigest       string    `json:"doc_digest" doc:"Отпечаток, над которым поставлена подпись."`
	CurrentVersion  bool      `json:"current_version" doc:"Подпись над текущей версией; false — «по прежней версии», не засчитывается."`
	Verification    string    `json:"verification" enum:"valid,invalid,unverifiable,pending" doc:"Статус автоматической проверки подписи (FR-68); недоступный ключ — unverifiable, никогда не valid (AD-32)."`
	ProvenanceClass string    `json:"provenance_class" enum:"personal,paper,partner,scenario,genesis,server_attested,device" doc:"Класс происхождения подписи (AD-2)."`
}

// DocumentSummary — документ в списке объекта (FR-65).
type DocumentSummary struct {
	DocumentID  string            `json:"document_id"`
	Version     int               `json:"version" minimum:"1"`
	Template    string            `json:"template" doc:"Шаблон@версия из нормативного слоя."`
	Title       string            `json:"title"`
	Subject     platform.DrillRef `json:"subject" doc:"Объект документа: изделие, несоответствие, партия…"`
	Status      string            `json:"status" enum:"requested,drafted,signing,route_closed,annulled"`
	DocDigest   string            `json:"doc_digest,omitempty"`
	DraftedAt   *time.Time        `json:"drafted_at,omitempty"`
	ClosedAt    *time.Time        `json:"closed_at,omitempty"`
	PaperStatus string            `json:"paper_status,omitempty" enum:"printed,signed,destroyed" doc:"Статус бумажного экземпляра."`
}

// DocumentList — документы объекта.
type DocumentList struct {
	Items      []DocumentSummary `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// DocumentView — документ для подписи (AD-12): канонический content, отпечаток,
// хеш отрисовки, маршрут с прогрессом подписей. Агент токена сам пересчитывает
// отпечаток из content (AD-14).
type DocumentView struct {
	DocumentID        string                  `json:"document_id"`
	Version           int                     `json:"version" minimum:"1"`
	Template          string                  `json:"template"`
	DocFormatVersion  int                     `json:"doc_format_version" minimum:"1"`
	Title             string                  `json:"title"`
	Subject           platform.DrillRef       `json:"subject"`
	Status            string                  `json:"status" enum:"requested,drafted,signing,route_closed,annulled"`
	Content           map[string]any          `json:"content" doc:"Канонические данные документа (JCS без rendering_hash)."`
	RenderingHash     string                  `json:"rendering_hash" doc:"H(render(шаблон@версия, content))."`
	DocDigest         string                  `json:"doc_digest" doc:"Отпечаток документа streebog256:…; в QR печатной рамки."`
	SummaryFields     []DocumentSummaryField  `json:"summary_fields" doc:"Поля сводки уровня 2 — входят в отпечаток (AD-12)."`
	SourceEventIDs    []string                `json:"source_event_ids" doc:"События-источники документа."`
	Stages            []DocumentApprovalStage `json:"stages" doc:"Обязательные подписи — замороженный набор (AD-43)."`
	Signatures        []DocumentSignatureView `json:"signatures"`
	SupersedesVersion *int                    `json:"supersedes_version,omitempty" doc:"Предыдущая версия документа."`
	QR                string                  `json:"qr" doc:"ant:doc:‹id›:‹отпечаток› — для печатной рамки."`
	BasisSeq          int64                   `json:"basis_seq"`
}

// DocumentSummaryField — поле сводки для окна подтверждения.
type DocumentSummaryField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
}

// DocumentRendering — каноническая отрисовка HTML (AD-12); печатная рамка в неё не входит.
type DocumentRendering struct {
	DocumentID    string `json:"document_id"`
	Version       int    `json:"version" minimum:"1"`
	RenderingHash string `json:"rendering_hash"`
	HTML          string `json:"html" doc:"Каноническая отрисовка; серверного PDF нет."`
}

// DecisionCard — карточка «требуется ваше решение» для редких подписантов
// (FR-136): что решается, почему именно вы, основания и альтернативы, что
// будет после подписи.
type DecisionCard struct {
	DocumentID     string                 `json:"document_id"`
	Version        int                    `json:"version" minimum:"1"`
	Title          string                 `json:"title"`
	Question       string                 `json:"question" doc:"Что решается — одной фразой."`
	WhyYou         string                 `json:"why_you" doc:"Почему решение за вами: полномочие, этап маршрута."`
	Subject        platform.DrillRef      `json:"subject"`
	Basis          []platform.DrillRef    `json:"basis" doc:"Основания — записи и объекты."`
	SummaryFields  []DocumentSummaryField `json:"summary_fields"`
	Stage          DocumentApprovalStage  `json:"stage" doc:"Ваш этап маршрута."`
	OtherSigners   []string               `json:"other_signers" doc:"Кто ещё подписывает (подписант видит весь маршрут, AD-43)."`
	DueAt          *time.Time             `json:"due_at,omitempty" doc:"Срок решения."`
	AfterSignature string                 `json:"after_signature" doc:"Что произойдёт после подписи."`
	DocDigest      string                 `json:"doc_digest"`
	BasisSeq       int64                  `json:"basis_seq"`
}
