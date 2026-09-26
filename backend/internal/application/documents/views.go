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
	Method          string    `json:"method" enum:"token_agent,paper,device,demo_signer,source_decision" doc:"source_decision — этап закрыт самим решением-источником (by_source)."`
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
	Status      string            `json:"status" enum:"requested,drafted,signing,route_closed,annulled,live,returned" doc:"live — сопроводительная карта собирается из истории, версия ещё не зафиксирована; returned — подписант вернул версию с замечанием."`
	DocDigest   string            `json:"doc_digest,omitempty"`
	DraftedAt   *time.Time        `json:"drafted_at,omitempty"`
	ClosedAt    *time.Time        `json:"closed_at,omitempty"`
	PaperStatus string            `json:"paper_status,omitempty" enum:"printed,signed,destroyed" doc:"Статус бумажного экземпляра."`
	// DocType, Class, Versions — вид документа (построитель шаблона), класс и
	// число версий (AD-12: новая версия — новый отпечаток).
	DocType  string `json:"doc_type,omitempty" doc:"Вид документа: traveler, nc_statement, nc_disposition, generic."`
	Class    string `json:"class,omitempty" enum:"record,decision,requirement,input" doc:"Класс документа: запись, решение, требование, вход."`
	Versions int    `json:"versions,omitempty" doc:"Сколько версий сформировано."`
}

// DocumentList — документы объекта.
type DocumentList struct {
	Items      []DocumentSummary `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
	// CollectedFromHistory, ManualEntries — счётчик FR-65 «документов собрано
	// из истории — вручную не понадобилось».
	CollectedFromHistory *int `json:"collected_from_history,omitempty" doc:"FR-65: документов собрано из истории изделия."`
	ManualEntries        *int `json:"manual_entries,omitempty" doc:"FR-65: сколько полей заполнено вручную — у документов-проекций всегда 0."`
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
	Status            string                  `json:"status" enum:"requested,drafted,signing,route_closed,annulled,live,returned"`
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
	// DocType, Class, Route, Versions, Paper, Verification, Live — совместимые
	// дополнения эпика 28: маршрут в форме карточки подписи (эпик 11), список
	// версий, бумажные экземпляры, как проверены подписи.
	DocType      string               `json:"doc_type,omitempty" doc:"Вид документа."`
	Class        string               `json:"class,omitempty" enum:"record,decision,requirement,input"`
	Route        []DocumentRouteStage `json:"route,omitempty" doc:"Маршрут с подписями по этапам: засчитана или нет и почему (AD-43)."`
	Versions     []DocumentVersionRef `json:"versions,omitempty" doc:"Версии документа: у каждой свой отпечаток, подписи прежней остаются при ней (AD-12)."`
	Paper        []DocumentPaperMark  `json:"paper,omitempty" doc:"Бумажные экземпляры: напечатан, подписан, уничтожен (AD-12)."`
	Verification string               `json:"verification,omitempty" enum:"full,demo" doc:"Как проверены подписи при закрытии маршрута; demo — без агента токена (Д-30)."`
	Live         bool                 `json:"live,omitempty" doc:"Версия ещё не зафиксирована: показан текущий сбор из истории (номер — следующей версии)."`
	RouteClosed  *string              `json:"route_closed_event_id,omitempty" doc:"Реакция document.route.closed версии."`
}

// DocumentVersionRef — версия документа в списке версий.
type DocumentVersionRef struct {
	Version    int       `json:"version" minimum:"1"`
	DocDigest  string    `json:"doc_digest"`
	Status     string    `json:"status" enum:"drafted,signing,route_closed,annulled,returned"`
	DraftedAt  time.Time `json:"drafted_at"`
	Signatures int       `json:"signatures" doc:"Сколько подписей записано над этой версией."`
}

// DocumentPaperMark — статус бумажного экземпляра версии.
type DocumentPaperMark struct {
	EventID string    `json:"event_id"`
	Version int       `json:"version" minimum:"1"`
	Status  string    `json:"paper_status" enum:"printed,signed,destroyed"`
	CopyNo  string    `json:"copy_no,omitempty"`
	At      time.Time `json:"at"`
}

// DocumentStageSignature — подпись этапа маршрута (форма StageSignature
// эпика 11: подпись записи + засчитана ли модулем documents, AD-43).
type DocumentStageSignature struct {
	EventID          string    `json:"event_id" doc:"Запись document.signature.recorded или решение-источник (by_source)."`
	SignerID         string    `json:"signer_id" doc:"Псевдоним подписанта или id устройства."`
	Class            string    `json:"class" enum:"device,personal,paper,partner,server_attested,scenario,genesis" doc:"Класс происхождения подписи (AD-2)."`
	Level            int       `json:"level" minimum:"0" maximum:"3" doc:"Уровень подписи (AD-13)."`
	Check            string    `json:"check" enum:"valid,rejected,not_verifiable,unchecked" doc:"Проверка подписи (FR-68): демо без агента токена — unchecked."`
	AttestedBy       string    `json:"attested_by,omitempty" doc:"Заверитель бумажной подписи (AD-43)."`
	PaperOriginalRef string    `json:"paper_original_ref,omitempty" doc:"Учётный номер бумажного оригинала в архиве ОТК."`
	ScanAddress      string    `json:"scan_address,omitempty" doc:"Адрес скана в хранилище материалов."`
	Method           string    `json:"method" enum:"token_agent,paper,device,demo_signer,source_decision"`
	SignedAt         time.Time `json:"signed_at"`
	Counted          bool      `json:"counted" doc:"Засчитана модулем documents: текущий отпечаток, полномочие и клеймо, уровень, разделение обязанностей."`
	PreviousVersion  bool      `json:"previous_version,omitempty" doc:"Подпись по прежней версии — видна, но не засчитывается."`
	Why              string    `json:"why,omitempty" doc:"Почему не засчитана: prior_version, authority, no_stamp, level, separation_*, paper_forbidden, attester_*, surplus."`
}

// DocumentRouteStage — этап маршрута с подписями (форма RouteStage эпика 11:
// элемент required_approvals + подписи).
type DocumentRouteStage struct {
	Stage               int                      `json:"stage" minimum:"1"`
	Title               string                   `json:"title,omitempty"`
	Role                string                   `json:"role,omitempty"`
	AuthorityID         string                   `json:"authority_id"`
	AuthorityLabel      string                   `json:"authority_label" doc:"Кто подписывает этап — для людей."`
	StampKind           string                   `json:"stamp_kind,omitempty"`
	Quorum              string                   `json:"quorum" enum:"one,all,k_of_n"`
	K                   int                      `json:"k,omitempty"`
	Required            int                      `json:"required" minimum:"1" doc:"Сколько засчитанных подписей нужно на этапе."`
	SignatureLevel      int                      `json:"signature_level" minimum:"0" maximum:"3"`
	PaperAllowed        bool                     `json:"paper_allowed"`
	AttesterAuthorityID string                   `json:"attester_authority_id,omitempty"`
	ExternalParty       string                   `json:"external_party,omitempty" enum:"none,customer_representative,partner"`
	BySource            bool                     `json:"by_source,omitempty" doc:"Этап закрыт самим решением-источником."`
	Separation          []string                 `json:"separation,omitempty"`
	Done                bool                     `json:"done"`
	Signatures          []DocumentStageSignature `json:"signatures"`
}

// RoutedDocument — документ с маршрутом подписей (форма эпика 11).
type RoutedDocument struct {
	DocumentID  string               `json:"document_id"`
	Version     int                  `json:"version" minimum:"1"`
	TemplateRef string               `json:"template_ref"`
	DocType     string               `json:"doc_type"`
	Title       string               `json:"title"`
	DocDigest   string               `json:"doc_digest" doc:"Отпечаток — входит в QR бумажного экземпляра (AD-43)."`
	Status      string               `json:"status" enum:"drafted,in_route,closed,annulled" doc:"Те же значения, что у документов паспорта."`
	DraftedAt   time.Time            `json:"drafted_at"`
	Route       []DocumentRouteStage `json:"route"`
	BasisSeq    int64                `json:"basis_seq"`
}

// DecisionProposal — что предлагается подписать (FR-136).
type DecisionProposal struct {
	Kind      string `json:"kind" enum:"disposition,concession,presentation,other"`
	Code      string `json:"code" doc:"Код предложения: вариант решения, concession, accept…"`
	Summary   string `json:"summary"`
	ItemID    string `json:"item_id"`
	ItemLabel string `json:"item_label"`
	NCID      string `json:"nc_id,omitempty"`
	NCNumber  string `json:"nc_number,omitempty"`
}

// DecisionEscalation — почему решение пришло к подписанту (FR-50, FR-136).
type DecisionEscalation struct {
	Reason         string `json:"reason"`
	AutomationMode int    `json:"automation_mode" minimum:"0" maximum:"5"`
	RuleID         string `json:"rule_id,omitempty"`
	RuleRev        string `json:"rule_rev,omitempty"`
}

// DecisionEvidence — довод: запись журнала-источник документа.
type DecisionEvidence struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	OccurredAt time.Time `json:"occurred_at"`
}

// SimilarDecision — похожее прошлое решение (FR-136).
type SimilarDecision struct {
	RefID     string    `json:"ref_id"`
	Number    string    `json:"number"`
	Summary   string    `json:"summary"`
	Outcome   string    `json:"outcome" enum:"accepted,rejected"`
	DecidedAt time.Time `json:"decided_at"`
}

// AwaitingAttestation — бумажная подпись этапа ждёт заверения (FR-139, AD-43).
type AwaitingAttestation struct {
	Stage  int    `json:"stage"`
	Signer string `json:"signer"`
}

// DecisionRequest — карточка «требуется ваше решение» (форма эпика 11,
// FR-136): документ с маршрутом, что предлагается, почему к вам, доводы,
// похожие случаи, ваш этап, срок.
type DecisionRequest struct {
	Document            RoutedDocument       `json:"document"`
	Proposal            DecisionProposal     `json:"proposal"`
	Escalation          DecisionEscalation   `json:"escalation"`
	Evidence            []DecisionEvidence   `json:"evidence"`
	SimilarAccepted     []SimilarDecision    `json:"similar_accepted"`
	SimilarRejected     []SimilarDecision    `json:"similar_rejected"`
	MyStage             *int                 `json:"my_stage,omitempty" doc:"Этап, который ждёт подписи текущего пользователя; нет — не ваш черёд."`
	ExpectedSigner      *string              `json:"expected_signer,omitempty" doc:"Ожидаемый подписант этапа — он же в QR бумажного экземпляра."`
	DueAt               *time.Time           `json:"due_at,omitempty"`
	AwaitingAttestation *AwaitingAttestation `json:"awaiting_attestation,omitempty"`
}

// DecisionRequestList — запросы решения, ждущие подписи текущего пользователя.
type DecisionRequestList struct {
	Items []DecisionRequest `json:"items"`
}

// PrintView — печатная форма: каноническая отрисовка в печатной рамке с QR
// `ant:doc:‹id›:‹отпечаток›`, датой печати и колонтитулом «получено из
// системы» (AD-12: рамка в отрисовку и отпечаток не входит).
type PrintView struct {
	DocumentID    string    `json:"document_id"`
	Version       int       `json:"version" minimum:"1"`
	DocDigest     string    `json:"doc_digest"`
	RenderingHash string    `json:"rendering_hash"`
	QR            string    `json:"qr"`
	QRSVG         string    `json:"qr_svg" doc:"QR как SVG — рисует сервер (Д-30)."`
	PrintedAt     time.Time `json:"printed_at"`
	HTML          string    `json:"html" doc:"Полная страница для печати: отрисовка + рамка."`
}

// PrintAccepted — ответ на печать бумажного экземпляра: квитанция и что печатать.
type PrintAccepted struct {
	CommandID string   `json:"command_id"`
	Seq       int64    `json:"seq"`
	EventIDs  []string `json:"event_ids"`
	Replayed  bool     `json:"replayed"`
	Version   int      `json:"version" minimum:"1" doc:"Печатаемая версия (у живой карты — новая версия, зафиксированная этой печатью)."`
	DocDigest string   `json:"doc_digest"`
	QR        string   `json:"qr"`
	PrintURL  string   `json:"print_url" doc:"Страница для печати (GET)."`
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

// RequestAccepted — ответ на «Запросить решение» / новую версию: квитанция и
// документ. Version и DocDigest — версия, которую оформит свёртка (у
// документа изделия её запишет воркер той же функцией, AD-5).
type RequestAccepted struct {
	CommandID  string   `json:"command_id"`
	Seq        int64    `json:"seq"`
	EventIDs   []string `json:"event_ids"`
	Replayed   bool     `json:"replayed"`
	DocumentID string   `json:"document_id"`
	Version    int      `json:"version,omitempty" doc:"Версия, которую оформит запрос; 0 — содержимое не изменилось, новой версии нет."`
	DocDigest  string   `json:"doc_digest,omitempty"`
}
