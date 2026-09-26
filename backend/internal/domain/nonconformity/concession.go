package nonconformity

import (
	"maps"
	"slices"
	"time"

	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	"ant/internal/domain/kernel"
)

// Разрешение на отклонение (FR-54, ГОСТ Р ИСО 9000 п. 3.12.5): номер, пункт
// КД/ТУ, область действия (изделия или диапазон номеров), лимит количества,
// срок, отзыв. Живёт в своём потоке `concession:‹id›` вне изделия (AD-39);
// лимит открывается атомарно с записью выдачи, расход — атомарно с решением
// (journal.Append: ConcessionGrants и Check.Consume). Книга разрешений —
// чистая свёртка записей выдачи, отзыва и решений, на которых разрешение
// израсходовано: её строят api для гарда, запросы и верификатор.

// Статусы разрешения (перечисление Concession.status).
const (
	ConcessionActive    = "active"
	ConcessionExhausted = "exhausted"
	ConcessionExpired   = "expired"
	ConcessionRevoked   = "revoked"
)

// ConcessionUse — решение, на котором разрешение израсходовано.
type ConcessionUse struct {
	EventID string    `json:"event_id"`
	ItemID  string    `json:"item_id"`
	NCID    string    `json:"nc_id,omitempty"`
	At      time.Time `json:"at"`
}

// ConcessionView — разрешение на отклонение по записям журнала.
type ConcessionView struct {
	Grant          ConcessionGrantedData `json:"grant"`
	GrantedEventID string                `json:"granted_event_id"`
	GrantedAt      time.Time             `json:"granted_at"`
	ValidUntil     *time.Time            `json:"valid_until,omitempty"`
	Revoked        bool                  `json:"revoked,omitempty"`
	RevokedEventID string                `json:"revoked_event_id,omitempty"`
	Uses           []ConcessionUse       `json:"uses,omitempty"`
	// StreamSeq — seq последней записи потока разрешения (basis_seq проверки AD-39).
	StreamSeq int64 `json:"stream_seq"`
}

// Used — сколько изделий уже по разрешению.
func (c ConcessionView) Used() int { return len(c.Uses) }

// Status — статус разрешения на момент at (доменное время, AD-37).
func (c ConcessionView) Status(at time.Time) string {
	switch {
	case c.Revoked:
		return ConcessionRevoked
	case c.ValidUntil != nil && !at.IsZero() && at.After(*c.ValidUntil):
		return ConcessionExpired
	case c.Grant.Limit > 0 && c.Used() >= c.Grant.Limit:
		return ConcessionExhausted
	}
	return ConcessionActive
}

// InScope — изделие в области действия: перечень изделий или диапазон
// номеров (сравнение строк id: номера одной серии одной длины); без области —
// любое изделие.
func (c ConcessionView) InScope(itemID string) bool {
	g := c.Grant
	if len(g.ScopeItemIDs) == 0 && g.ScopeRangeFrom == "" && g.ScopeRangeTo == "" {
		return true
	}
	if slices.Contains(g.ScopeItemIDs, itemID) {
		return true
	}
	if g.ScopeRangeFrom == "" && g.ScopeRangeTo == "" {
		return false
	}
	return (g.ScopeRangeFrom == "" || itemID >= g.ScopeRangeFrom) && (g.ScopeRangeTo == "" || itemID <= g.ScopeRangeTo)
}

// ConcessionBook — разрешения на отклонение по записям журнала.
type ConcessionBook struct {
	Items map[string]*ConcessionView `json:"items"`
}

// NewConcessionBook — пустая книга.
func NewConcessionBook() *ConcessionBook { return &ConcessionBook{Items: map[string]*ConcessionView{}} }

// Apply добавляет запись к книге: выдача, отзыв, решение с разрешением.
// Порядок — по seq (порядок знания, AD-37).
func (b *ConcessionBook) Apply(r kernel.Record) {
	switch r.Type {
	case catalog.DecisionConcessionGranted:
		var d ConcessionGrantedData
		if !decode(r, &d) || d.ConcessionID == "" {
			return
		}
		if _, ok := b.Items[d.ConcessionID]; ok {
			return
		}
		v := &ConcessionView{Grant: d, GrantedEventID: r.EventID, GrantedAt: r.OccurredAt, StreamSeq: r.Seq}
		if t, err := time.Parse(time.RFC3339Nano, d.ValidUntil); err == nil {
			t = t.UTC()
			v.ValidUntil = &t
		}
		b.Items[d.ConcessionID] = v
	case catalog.DecisionConcessionRevoked:
		var d ConcessionRevokedData
		if decode(r, &d) {
			if v := b.Items[d.ConcessionID]; v != nil {
				v.Revoked, v.RevokedEventID = true, r.EventID
				v.StreamSeq = max(v.StreamSeq, r.Seq)
			}
		}
	case catalog.DecisionDispositionSet:
		var d DispositionSetData
		if decode(r, &d) && d.ConcessionID != "" {
			b.use(d.ConcessionID, ConcessionUse{EventID: r.EventID, ItemID: r.ItemID, NCID: d.NCID, At: r.OccurredAt})
		}
	case catalog.DecisionPresentationResolved:
		var d PresentationResolvedData
		if decode(r, &d) && d.ConcessionID != "" && d.Resolution == "accept_with_concession" {
			b.use(d.ConcessionID, ConcessionUse{EventID: r.EventID, ItemID: r.ItemID, At: r.OccurredAt})
		}
	}
}

func (b *ConcessionBook) use(id string, u ConcessionUse) {
	v := b.Items[id]
	if v == nil || slices.ContainsFunc(v.Uses, func(x ConcessionUse) bool { return x.EventID == u.EventID }) {
		return
	}
	v.Uses = append(v.Uses, u)
}

// Sorted — разрешения по id.
func (b *ConcessionBook) Sorted() []ConcessionView {
	ids := slices.Sorted(maps.Keys(b.Items))
	out := make([]ConcessionView, 0, len(ids))
	for _, id := range ids {
		out = append(out, *b.Items[id])
	}
	return out
}

// ConcessionGuard — гард применения разрешения к решению по изделию (FR-53,
// FR-54, AD-39): разрешение есть, для этого вида решения, не отозвано, срок
// не истёк на доменное «сейчас», изделие в области действия. Остаток лимита
// проверяет journal.Append атомарно с записью решения (двойной расход с двух
// копий api — 409 journal.concession_exhausted).
func ConcessionGuard(c *ConcessionView, id, itemID, kind string, at time.Time) error {
	if c == nil {
		return kernel.Refuse(errcodes.NonconformityConcessionNotApplicable, "concession_id", id, "reason", "разрешение не найдено")
	}
	if kind != "" && c.Grant.Kind != kind {
		return kernel.Refuse(errcodes.NonconformityConcessionNotApplicable, "concession_id", id, "reason", "разрешение выдано для решения «"+DispositionLabel(c.Grant.Kind)+"»")
	}
	switch c.Status(at) {
	case ConcessionRevoked:
		return kernel.Refuse(errcodes.NonconformityConcessionNotApplicable, "concession_id", id, "reason", "отозвано")
	case ConcessionExpired:
		return kernel.Refuse(errcodes.NonconformityConcessionNotApplicable, "concession_id", id, "reason", "истёк срок")
	}
	if !c.InScope(itemID) {
		return kernel.Refuse(errcodes.NonconformityConcessionNotApplicable, "concession_id", id, "reason", "изделие "+itemID+" вне области действия")
	}
	return nil
}

// GrantGuard — гард выдачи разрешения: id свободен, лимит ≥ 1, вид решения известен.
func GrantGuard(b *ConcessionBook, d ConcessionGrantedData) error {
	if _, ok := b.Items[d.ConcessionID]; ok {
		return kernel.Refuse(errcodes.NonconformityInvalidTransition, "action", "выдать разрешение", "nc_id", d.ConcessionID, "status", "уже выдано")
	}
	if d.Kind != "repair" && d.Kind != "use_as_is" {
		return kernel.Refuse(errcodes.ApiValidationFailed, "field", "kind", "reason", "repair | use_as_is")
	}
	if d.Limit < 1 {
		return kernel.Refuse(errcodes.ApiValidationFailed, "field", "limit", "reason", "не меньше 1")
	}
	return nil
}

// RevokeGuard — гард отзыва разрешения.
func RevokeGuard(b *ConcessionBook, id string) error {
	v := b.Items[id]
	if v == nil {
		return kernel.Refuse(errcodes.ApiNotFound, "object", "Разрешение на отклонение", "id", id)
	}
	if v.Revoked {
		return kernel.Refuse(errcodes.NonconformityInvalidTransition, "action", "отозвать разрешение", "nc_id", id, "status", "уже отозвано")
	}
	return nil
}
