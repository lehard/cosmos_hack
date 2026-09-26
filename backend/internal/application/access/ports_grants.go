package access

import (
	"context"
	"slices"
	"strings"
	"time"

	documentsapp "ant/internal/application/documents"
	"ant/internal/application/platform"
	accessdom "ant/internal/domain/access"
)

// GrantDocuments — ведомый порт документов выдачи прав (AD-13, AD-43, эпик
// 26): маршрут подписей документа «Выдача ролей, полномочий, клейм» ведёт
// модуль documents; access исполняет выдачу только по закрытому маршруту и
// сам перепроверяет вторую подпись независимой стороны
// (accessdom.CheckApprovals) — модуль-исполнитель подписи не считает, но
// права подписанта на момент подписи проверяет его гард.
type GrantDocuments interface {
	GrantDocument(ctx context.Context, documentID string) (GrantDocument, error)
}

// GrantDocument — документ выдачи: что выдаёт, закрыт ли маршрут, чьи подписи засчитаны.
type GrantDocument struct {
	DocumentID string
	Template   string
	// Closed — маршрут закрыт (document.route.closed).
	Closed bool
	// Decision — решение документа (accessdom.FormatGrantDecision).
	Decision string
	// Approvals — засчитанные подписи текущей версии по этапам.
	Approvals []accessdom.Approval
}

// Cards — ведомый порт карточек решения редких подписантов (FR-136):
// видит ли сотрудник объект — документ, где он в маршруте (кандидат или уже
// подписал), или предмет такого документа (изделие и др.). Вызывается
// декоратором только для сотрудников «только карточка».
type Cards interface {
	Visible(ctx context.Context, personID string, obj platform.ObjectRef) (bool, error)
}

// DocumentsReader — чтение модуля documents, нужное мостам GrantDocuments и Cards
// (подмножество documentsapp.Queries).
type DocumentsReader interface {
	Documents(ctx context.Context, subject platform.DrillRef, m platform.Moment, p platform.Page) (documentsapp.DocumentList, error)
	Document(ctx context.Context, documentID string, version int, m platform.Moment) (documentsapp.DocumentView, error)
}

// DocumentsBridge — GrantDocuments и Cards над ведущим портом чтения
// documents (documents.document.read, documents.document.list): закрытость
// маршрута и засчитанные подписи — из замороженного набора документа (AD-43).
type DocumentsBridge struct {
	Docs DocumentsReader
}

var (
	_ GrantDocuments = DocumentsBridge{}
	_ Cards          = DocumentsBridge{}
)

// GrantDocument — документ выдачи по id: текущая версия, засчитанные подписи.
func (b DocumentsBridge) GrantDocument(ctx context.Context, documentID string) (GrantDocument, error) {
	v, err := b.Docs.Document(ctx, documentID, 0, platform.Moment{})
	if err != nil {
		return GrantDocument{}, err
	}
	g := GrantDocument{DocumentID: v.DocumentID, Template: v.Template, Closed: v.Status == "route_closed", Decision: findGrant(v.Content)}
	for _, s := range v.Signatures {
		if s.CurrentVersion && s.Verification != "invalid" {
			g.Approvals = append(g.Approvals, accessdom.Approval{PersonID: s.SignerPersonID, Stage: s.Stage, At: s.SignedAt})
		}
	}
	return g, nil
}

// Visible — документ адресован сотруднику (он кандидат этапа или подписал)
// либо объект — предмет такого документа.
func (b DocumentsBridge) Visible(ctx context.Context, personID string, obj platform.ObjectRef) (bool, error) {
	if obj.ID == "" {
		return true, nil
	}
	if obj.Kind == "document" {
		v, err := b.Docs.Document(ctx, obj.ID, 0, platform.Moment{})
		if err != nil {
			return false, nil
		}
		return addressed(v, personID), nil
	}
	list, err := b.Docs.Documents(ctx, platform.DrillRef{Entity: platform.EntityKind(obj.Kind), ID: obj.ID}, platform.Moment{}, platform.Page{Limit: 100})
	if err != nil {
		return false, nil
	}
	for _, d := range list.Items {
		v, err := b.Docs.Document(ctx, d.DocumentID, 0, platform.Moment{})
		if err == nil && addressed(v, personID) {
			return true, nil
		}
	}
	return false, nil
}

func addressed(v documentsapp.DocumentView, personID string) bool {
	for _, st := range v.Stages {
		if slices.Contains(st.Candidates, personID) || slices.Contains(st.SignedBy, personID) {
			return true
		}
	}
	return false
}

// findGrant — строка решения выдачи в содержимом документа (где бы её ни
// положил построитель содержимого documents).
func findGrant(v any) string {
	switch x := v.(type) {
	case string:
		if strings.HasPrefix(x, accessdom.GrantDecisionPrefix) {
			if _, ok := accessdom.ParseGrantDecision(x); ok {
				return x
			}
		}
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			if s := findGrant(x[k]); s != "" {
				return s
			}
		}
	case []any:
		for _, e := range x {
			if s := findGrant(e); s != "" {
				return s
			}
		}
	}
	return ""
}

// PolicyAuthorities — порт полномочий модуля signing (application/signing.Authorities,
// AD-11, AD-43) над проекцией политики вместо статической таблицы эпика 27:
// полномочие второй подписи и заверения бумаги — на позиции журнала seq,
// сфера сотрудника — по его действующим ролям.
type PolicyAuthorities struct {
	// Policy — проекция политики (Projection: политика на любой seq).
	Policy interface {
		PolicySource
		At(ctx context.Context, seq int64) (accessdom.Policy, error)
	}
	// Now — доменное «сейчас» (AD-37) для сроков полномочий; nil — сроки не проверяются.
	Now func(ctx context.Context) (time.Time, error)
}

// Has — у сотрудника есть полномочие на позиции журнала seq (0 — действующая политика).
func (a PolicyAuthorities) Has(ctx context.Context, personID, authorityID string, seq int64) (bool, error) {
	var pol accessdom.Policy
	var err error
	if seq > 0 {
		pol, err = a.Policy.At(ctx, seq)
	} else {
		pol, err = a.Policy.Policy(ctx)
	}
	if err != nil {
		return false, err
	}
	return pol.HasAuthority(personID, authorityID, "", a.now(ctx)), nil
}

// Domain — сфера сотрудника для второй подписи: qc / production / admin (AD-11).
func (a PolicyAuthorities) Domain(ctx context.Context, personID string) string {
	pol, err := a.Policy.Policy(ctx)
	if err != nil {
		return accessdom.DomainProduction
	}
	return pol.PersonDomain(personID, a.now(ctx))
}

func (a PolicyAuthorities) now(ctx context.Context) time.Time {
	if a.Now == nil {
		return time.Time{}
	}
	t, err := a.Now(ctx)
	if err != nil {
		return time.Time{}
	}
	return t
}
