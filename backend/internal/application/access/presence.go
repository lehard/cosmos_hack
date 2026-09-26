package access

import (
	"context"
	"slices"
	"sync"
	"time"

	"ant/internal/application/platform"
	"ant/internal/contracts/catalog"
	"ant/internal/contracts/errcodes"
	accessdom "ant/internal/domain/access"
)

// Присутствие на посту (эпик 37; FR-6, FR-82…FR-84, AD-15 барьер 2):
// проекция записей СКУД, ключа и сеансов рабочего места из журнала и порты
// записи фактов СКУД и отклонений присутствия. Права не меняются: это
// отдельная от политики свёртка (policy_seq от проходов не сдвигается).

// PresenceSource — снимок присутствия (проекция PresenceProjection).
type PresenceSource interface {
	// Presence — снимок; вызывающий не меняет его.
	Presence(ctx context.Context) (accessdom.Presence, error)
	// Passes — проходы по зонам (access.zone.passed) по возрастанию seq: история поста.
	Passes(ctx context.Context) ([]accessdom.Record, error)
}

// PresenceProjection — проекция присутствия над журналом (тот же порт чтения,
// что у политики, но типы — accessdom.PresenceTypes): сигнал «есть новое»
// (LISTEN/NOTIFY) — перечитать записи после своего seq.
type PresenceProjection struct {
	log PolicyLog
	// Every — наименьший интервал между перечитываниями журнала без сигнала.
	Every time.Duration
	// Now — инфраструктурные часы (AD-37); nil — системные.
	Now func() time.Time

	mu       sync.Mutex
	cur      accessdom.Presence
	passes   []accessdom.Record
	loaded   bool
	lastHead int64
	lastRead time.Time
}

// NewPresenceProjection — проекция присутствия над log (nil — пустая).
func NewPresenceProjection(log PolicyLog) *PresenceProjection {
	return &PresenceProjection{log: log, Every: 500 * time.Millisecond, cur: accessdom.NewPresence()}
}

var _ PresenceSource = (*PresenceProjection)(nil)

// Presence — действующее присутствие; ошибка чтения журнала — последнее известное.
func (p *PresenceProjection) Presence(ctx context.Context) (accessdom.Presence, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.catchUp(ctx, false); err != nil && !p.loaded {
		return accessdom.Presence{}, err
	}
	return p.cur, nil
}

// Passes — проходы по зонам, применённые проекцией.
func (p *PresenceProjection) Passes(ctx context.Context) ([]accessdom.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.catchUp(ctx, false); err != nil && !p.loaded {
		return nil, err
	}
	return slices.Clone(p.passes), nil
}

// Refresh — перечитать журнал сейчас (после своей записи).
func (p *PresenceProjection) Refresh(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.catchUp(ctx, true)
}

func (p *PresenceProjection) catchUp(ctx context.Context, force bool) error {
	if p.log == nil {
		p.loaded = true
		return nil
	}
	now := time.Now()
	if p.Now != nil {
		now = p.Now()
	}
	head := p.log.Head()
	due := force || !p.loaded || (head > p.lastHead) || (head == 0 && now.Sub(p.lastRead) >= p.Every)
	if !due {
		return nil
	}
	recs, err := p.log.Since(ctx, p.cur.Seq)
	if err != nil {
		return err
	}
	next := p.cur.Clone()
	for _, r := range recs {
		if err := next.Apply(r); err != nil {
			return platform.Fail(errcodes.ApiInternalError, "detail", err.Error())
		}
		if catalog.Type(r.Type) == catalog.AccessZonePassed {
			p.passes = append(p.passes, r)
		}
	}
	p.cur, p.loaded, p.lastHead, p.lastRead = next, true, head, now
	return nil
}

// ZonePassIn — проход через точку СКУД от адаптера СКУД (FR-82).
type ZonePassIn struct {
	// SourceEventID — id события в СКУД: из него однозначный event_id записи
	// (повтор опроса не пишет проход дважды, AD-7).
	SourceEventID string
	PersonID      string
	ZoneID        string
	ReaderID      string
	Enter         bool
	// At — время прохода по часам СКУД (occurred_at).
	At time.Time
}

// PresenceWriter — ведомый порт записей присутствия (адаптер —
// infrastructure/storage/access над кодеком движка, происхождение
// server_attested): факты СКУД, снятие допуска автоматически, отклонения
// присутствия и отказы в допуске (шина безопасности, AD-24).
type PresenceWriter interface {
	// ZonePassed — access.zone.passed в поток сотрудника; false — уже записан.
	ZonePassed(ctx context.Context, p ZonePassIn) (bool, error)
	// Revoked — access.workplace.revoked (реакция: причина — cause, основание — causeEventID).
	Revoked(ctx context.Context, s accessdom.WorkplaceSession, cause, causeEventID string, at time.Time) error
	// Deviation — security.presence.deviation в поток рабочего места (повтор по d.Key не пишется).
	Deviation(ctx context.Context, d accessdom.Deviation, at time.Time) error
	// AdmissionDenied — security.admission.denied: отказ в допуске фиксируется (FR-83).
	AdmissionDenied(ctx context.Context, workplaceID, personID string, failed []string, at time.Time) error
}

// ShiftWindow — смена графика (справочник смен, эпик 19): id и начало.
type ShiftWindow struct {
	ID    string
	Start time.Time
	End   time.Time
}

// ShiftSchedule — ведомый порт графика смен: смены, идущие в момент at.
type ShiftSchedule interface {
	ShiftsAt(ctx context.Context, at time.Time) ([]ShiftWindow, error)
}

// WithPresence подключает присутствие: проекцию, запись фактов СКУД и
// отклонений, график смен (nil — проверка «по графику должен быть» выключена).
func WithPresence(src PresenceSource, w PresenceWriter, shifts ShiftSchedule) Option {
	return func(s *Service) { s.presence, s.pw, s.shifts = src, w, shifts }
}

// presenceNow — снимок присутствия; нет проекции — ok=false (присутствие «неизвестно»).
func (s *Service) presenceNow(ctx context.Context) (accessdom.Presence, bool, error) {
	if s.presence == nil {
		return accessdom.Presence{}, false, nil
	}
	pr, err := s.presence.Presence(ctx)
	if err != nil {
		return accessdom.Presence{}, false, err
	}
	return pr, true, nil
}

// refreshPresence — проекция присутствия видит свою запись сразу.
func (s *Service) refreshPresence(ctx context.Context) {
	if r, ok := s.presence.(interface{ Refresh(context.Context) error }); ok {
		_ = r.Refresh(ctx)
	}
}

// zoneOf — зона СКУД рабочего места (зона цеха по справочнику мест).
func (s *Service) zoneOf(workplaceID string) string {
	if s.dir == nil {
		return ""
	}
	wp, _ := s.dir.Workplace(workplaceID)
	return wp.Zone
}
