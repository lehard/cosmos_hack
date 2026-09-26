package documents

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"ant/internal/domain/access"
	"ant/internal/domain/kernel"
)

// Модель документов изделия и объектов вне изделия (AD-12, AD-43).
// Документ — не файл: версия хранит канонический content, хеш отрисовки и
// отпечаток, вычисленные из событий-источников; подписи — записи
// document.signature.recorded; закрытие маршрута вычисляется заново по
// подписям (AD-43) и записывается реакцией document.route.closed.

// Статусы документа для людей и API.
const (
	// StatusLive — сопроводительная карта собирается из истории, версия ещё
	// не зафиксирована (нет document.version.drafted).
	StatusLive = "live"
	// StatusDrafted — версия сформирована, подписей маршрута ещё нет.
	StatusDrafted = "drafted"
	// StatusSigning — идёт сбор подписей.
	StatusSigning = "signing"
	// StatusReturned — подписант не согласовал версию (document.signature.declined).
	StatusReturned = "returned"
	// StatusRouteClosed — маршрут закрыт (document.route.closed).
	StatusRouteClosed = "route_closed"
	// StatusAnnulled — версия аннулирована.
	StatusAnnulled = "annulled"
)

// Способы подписи (document.signature.recorded.method) и подпись решением-источником.
const (
	MethodTokenAgent = "token_agent"
	MethodPaper      = "paper"
	MethodDevice     = "device"
	MethodDemo       = "demo_signer"
	// MethodSource — этап закрыт самим решением-источником (by_source):
	// отдельной записи подписи нет, подпись — у решения.
	MethodSource = "source_decision"
)

// Проверка подписей при закрытии маршрута (document.route.closed.verification).
const (
	VerificationFull = "full"
	VerificationDemo = "demo"
)

// Ref — ссылка на запись журнала: event_id и occurred_at (для причин реакций, AD-3, AD-37).
type Ref struct {
	ID string    `json:"id"`
	At time.Time `json:"at"`
}

func refOf(r kernel.Record) Ref { return Ref{ID: r.EventID, At: r.OccurredAt} }

// Signature — подпись этапа маршрута версии документа.
type Signature struct {
	EventID     string    `json:"event_id"`
	Version     int       `json:"version"`
	Stage       int       `json:"stage"`
	Person      string    `json:"person"`
	Method      string    `json:"method"`
	Digest      string    `json:"doc_digest"`
	Level       int       `json:"level"`
	AuthorityID string    `json:"authority_id,omitempty"`
	StampID     string    `json:"stamp_id,omitempty"`
	AttestedBy  string    `json:"attested_by,omitempty"`
	PaperNo     string    `json:"paper_original_no,omitempty"`
	ScanAddress string    `json:"scan_address,omitempty"`
	KeyRef      string    `json:"key_ref,omitempty"`
	Provenance  string    `json:"provenance,omitempty"`
	At          time.Time `json:"at"`
}

// Decline — подписант не согласовал версию (document.signature.declined).
type Decline struct {
	EventID string    `json:"event_id"`
	Stage   int       `json:"stage"`
	Person  string    `json:"person"`
	Comment string    `json:"comment"`
	At      time.Time `json:"at"`
}

// Version — версия документа (AD-12): не редактируется; новая версия —
// новый отпечаток, подписи прежней остаются при ней.
type Version struct {
	No         int             `json:"no"`
	Supersedes int             `json:"supersedes,omitempty"`
	Content    json.RawMessage `json:"content"`
	// BodyHash — хеш содержательной части (без номера версии и маршрута):
	// новая версия появляется, только если содержимое изменилось.
	BodyHash      string `json:"body_hash"`
	RenderingHash string `json:"rendering_hash"`
	Digest        string `json:"doc_digest"`
	// Stages — замороженный набор обязательных подписей (AD-43).
	Stages []access.ApprovalStage `json:"stages"`
	// Sources — события-источники; Trigger — запись, по которой сформирована версия.
	Sources []Ref     `json:"sources"`
	Trigger Ref       `json:"trigger"`
	At      time.Time `json:"at"`
	// SourceSigners — подписанты этапов by_source: этап → автор решения-источника.
	SourceSigners []Signature `json:"source_signers,omitempty"`
	Signatures    []Signature `json:"signatures,omitempty"`
	Declines      []Decline   `json:"declines,omitempty"`
	// Closed — маршрут закрыт: ClosedBy — засчитанные подписи, ClosedAt —
	// время последней из причин (AD-37), ClosedCauses — причины реакции.
	Closed       bool      `json:"closed,omitempty"`
	ClosedBy     []string  `json:"closed_by,omitempty"`
	ClosedAt     time.Time `json:"closed_at,omitzero"`
	ClosedCauses []Ref     `json:"closed_causes,omitempty"`
	Verification string    `json:"verification,omitempty"`
	Annulled     bool      `json:"annulled,omitempty"`
	AnnulledBy   string    `json:"annulled_by,omitempty"`
	AnnulReason  string    `json:"annul_reason,omitempty"`
}

// PaperMark — статус бумажного экземпляра (document.paper.status_changed).
type PaperMark struct {
	EventID string    `json:"event_id"`
	Version int       `json:"version"`
	Status  string    `json:"status"`
	CopyNo  string    `json:"copy_no,omitempty"`
	At      time.Time `json:"at"`
}

// Doc — документ: шаблон, объект, версии.
type Doc struct {
	ID          string `json:"id"`
	TemplateRef string `json:"template_ref"`
	DocType     string `json:"doc_type"`
	Class       string `json:"class"`
	Title       string `json:"title"`
	// Subject — поток объекта документа (`item:‹id›`, `nonconformity:‹id›` …).
	Subject string `json:"subject"`
	// Key — объект внутри субъекта (несоответствие у решений и заявлений).
	Key string `json:"key,omitempty"`
	// Context — данные запроса или решения, из которых строится содержимое.
	Context  DocContext  `json:"context"`
	Versions []Version   `json:"versions,omitempty"`
	Paper    []PaperMark `json:"paper,omitempty"`
}

// DocContext — данные решения или запроса, по которым строится документ.
type DocContext struct {
	Decision    string   `json:"decision,omitempty"`
	Comment     string   `json:"comment,omitempty"`
	Author      string   `json:"author,omitempty"`
	At          string   `json:"at,omitempty"`
	Reason      string   `json:"reason,omitempty"`
	Concession  string   `json:"concession,omitempty"`
	ScrapKind   string   `json:"scrap_kind,omitempty"`
	Claim       string   `json:"claim,omitempty"`
	Sources     []string `json:"sources,omitempty"`
	SourceEvent string   `json:"source_event,omitempty"`
	// CustomerAcceptance — продукция с приёмкой представителя заказчика (режим 5).
	CustomerAcceptance bool `json:"customer_acceptance,omitempty"`
}

// Current — последняя версия (nil — версии нет).
func (d *Doc) Current() *Version {
	if len(d.Versions) == 0 {
		return nil
	}
	return &d.Versions[len(d.Versions)-1]
}

// Version — версия по номеру (0 — последняя).
func (d *Doc) Version(no int) *Version {
	if no <= 0 {
		return d.Current()
	}
	for i := range d.Versions {
		if d.Versions[i].No == no {
			return &d.Versions[i]
		}
	}
	return nil
}

// Status — статус документа по последней версии.
func (d *Doc) Status() string {
	v := d.Current()
	switch {
	case v == nil:
		return StatusLive
	case v.Annulled:
		return StatusAnnulled
	case v.Closed:
		return StatusRouteClosed
	case len(v.Declines) > 0:
		return StatusReturned
	case len(v.Signatures) > 0:
		return StatusSigning
	}
	return StatusDrafted
}

// PaperStatus — последний статус бумажного экземпляра версии no (пусто — не печатался).
func (d *Doc) PaperStatus(no int) string {
	st := ""
	for _, p := range d.Paper {
		if p.Version == no {
			st = p.Status
		}
	}
	return st
}

// Person — сотрудник по политике: роли (с наследованием), полномочия, виды
// клейм (AD-15, FR-145). Срез стартовой политики до проекции политики
// журнала (эпик 26).
type Person struct {
	ID          string   `json:"id"`
	Roles       []string `json:"roles"`
	Authorities []string `json:"authorities"`
	Stamps      []Stamp  `json:"stamps,omitempty"`
}

// Stamp — цифровое клеймо: номер и вид контроля (FR-145).
type Stamp struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

// People — сотрудники политики в порядке политики.
type People []Person

// Find — сотрудник по псевдониму.
func (ps People) Find(id string) (Person, bool) {
	for _, p := range ps {
		if p.ID == id {
			return p, true
		}
	}
	return Person{}, false
}

// HasRole — у сотрудника есть роль (с наследованием).
func (p Person) HasRole(r string) bool { return slices.Contains(p.Roles, r) }

// HasAuthority — у сотрудника есть полномочие.
func (p Person) HasAuthority(a string) bool { return slices.Contains(p.Authorities, a) }

// StampOf — клеймо вида kind (пусто — нет).
func (p Person) StampOf(kind string) string {
	for _, s := range p.Stamps {
		if s.Kind == kind {
			return s.ID
		}
	}
	return ""
}

// Step — шаг процесса для строк сопроводительной карты.
type Step struct {
	Title    string `json:"title"`
	Workshop string `json:"workshop,omitempty"`
}

// Env — нормативная часть модуля documents, закреплённая за изделием
// (AD-17): шаблоны документов, названия шагов процесса, срез политики для
// проверки подписей и режим проверки. Пустой Env (нет шаблонов) — модуль
// документов не активен: свёртка ничего не порождает (волна 2, тесты других
// модулей).
type Env struct {
	Templates Templates
	// Steps — названия шагов по step_key (из описания процесса версии).
	Steps map[string]Step
	// People — срез политики; пусто — полномочия подписантов не проверяются
	// (демо, пометка verification = demo).
	People People
	// Verification — full | demo (демо-профиль: подписи без агента токена, Д-30).
	Verification string
	// Policy — политика доступа на basis для обязательных подписей
	// (access.RequiredApprovals, AD-43, эпик 26): сфера выдачи прав, кандидаты
	// этапов. Пусто — этапы только по условиям маршрута. В каноническом JSON
	// документа не участвует.
	Policy access.Policy `json:"-"`
}

// Active — модуль документов включён (есть шаблоны).
func (e Env) Active() bool { return len(e.Templates) > 0 }

// verification — режим проверки для document.route.closed.
func (e Env) verification() string {
	if e.Verification == VerificationFull && len(e.People) > 0 {
		return VerificationFull
	}
	return VerificationDemo
}

// StepTitle — название шага или сам step_key.
func (e Env) StepTitle(stepKey string) string {
	if s, ok := e.Steps[stepKey]; ok && s.Title != "" {
		return s.Title
	}
	return stepKey
}

// PersonOf — псевдоним сотрудника из автора записи `key_id@версия` (AD-10).
func PersonOf(actor string) string {
	p, _, _ := strings.Cut(actor, "@")
	return p
}

// DocRef — документ изделия для паспорта и проекции (FR-65).
type DocRef struct {
	DocumentID  string `json:"document_id"`
	TemplateRef string `json:"template_ref"`
	DocType     string `json:"doc_type"`
	Title       string `json:"title"`
	// Status — drafted | in_route | closed | annulled (статусы документов паспорта).
	Status  string `json:"status"`
	Version int    `json:"version"`
	Digest  string `json:"doc_digest"`
}

// PassportStatus — статус документа в терминах паспорта изделия.
func PassportStatus(st string) string {
	switch st {
	case StatusSigning, StatusReturned:
		return "in_route"
	case StatusRouteClosed:
		return "closed"
	case StatusAnnulled:
		return "annulled"
	}
	return "drafted"
}

// Refs — документы изделия «собранные из истории» (FR-65): все документы с
// версиями и сопроводительная карта, которая собирается из истории, даже если
// версия ещё не зафиксирована.
func (s *State) Refs() []DocRef {
	var out []DocRef
	if s.TravelerRef != "" && len(s.Rows) > 0 && s.doc(TravelerID(s.itemID())) == nil {
		out = append(out, DocRef{DocumentID: TravelerID(s.itemID()), TemplateRef: s.TravelerRef, DocType: DocTraveler,
			Title: "Сопроводительная карта изделия", Status: "drafted"})
	}
	for i := range s.Docs {
		d := &s.Docs[i]
		r := DocRef{DocumentID: d.ID, TemplateRef: d.TemplateRef, DocType: d.DocType, Title: d.Title, Status: PassportStatus(d.Status())}
		if cur := d.Current(); cur != nil {
			r.Version, r.Digest = cur.No, cur.Digest
		}
		out = append(out, r)
	}
	return out
}
